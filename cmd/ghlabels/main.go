package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var configPaths = []string{
	".",
	".github",
}

type Label struct {
	Name        string   `json:"name"`
	Color       string   `json:"color"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

func main() {
	config := flag.String("config", "labels.json", "Label config")

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		panic("not enough arguments")
	}

	repo := args[0]

	githubToken := os.Getenv("GITHUB_TOKEN")

	if len(githubToken) < 1 {
		panic("no github token")
	}

	var filePath string

	for _, p := range configPaths {
		filePath = path.Join(p, *config)

		if _, err := os.Stat(filePath); err == nil {
			break
		}
	}

	file, err := os.Open(filePath) // nolint: gosec
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var labels []Label

	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&labels); err != nil {
		panic(err)
	}

	if len(labels) < 1 {
		panic("no labels provided")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	repoSegments := strings.Split(repo, "/")
	owner := repoSegments[0]
	repo = repoSegments[1]

	_, _, err = client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		panic(err)
	}

	for _, label := range labels {
		currentName := label.Name
		var exists bool
		var currentLabel *github.Label

		githubLabel, _, err := client.Issues.GetLabel(context.Background(), owner, repo, label.Name)
		if err != nil {
			// label does not exist with this name
			// check aliases

			for _, alias := range label.Aliases {
				githubLabel, _, err := client.Issues.GetLabel(context.Background(), owner, repo, alias)
				if err != nil {
					continue
				}

				currentName = alias
				exists = true
				currentLabel = githubLabel

				break
			}
		} else {
			exists = true
			currentLabel = githubLabel
		}

		// create label
		if !exists {
			l := &github.Label{
				Name:        github.String(label.Name),
				Color:       github.String(label.Color),
				Description: github.String(label.Description),
			}

			_, _, err := client.Issues.CreateLabel(context.Background(), owner, repo, l)
			if err != nil {
				panic(err)
			}

			continue
		}

		// update label
		currentLabel.Name = github.String(label.Name)
		currentLabel.Color = github.String(label.Color)
		currentLabel.Description = github.String(label.Description)

		_, _, err = client.Issues.EditLabel(context.Background(), owner, repo, currentName, currentLabel)
		if err != nil {
			panic(err)
		}
	}
}
