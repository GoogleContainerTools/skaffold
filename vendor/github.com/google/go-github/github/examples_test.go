// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v18/github"
)

func ExampleClient_Markdown() {
	client := github.NewClient(nil)

	input := "# heading #\n\nLink to issue #1"
	opt := &github.MarkdownOptions{Mode: "gfm", Context: "google/go-github"}

	output, _, err := client.Markdown(context.Background(), input, opt)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(output)
}

func ExampleRepositoriesService_GetReadme() {
	client := github.NewClient(nil)

	readme, _, err := client.Repositories.GetReadme(context.Background(), "google", "go-github", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	content, err := readme.GetContent()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("google/go-github README:\n%v\n", content)
}

func ExampleRepositoriesService_List() {
	client := github.NewClient(nil)

	user := "willnorris"
	opt := &github.RepositoryListOptions{Type: "owner", Sort: "updated", Direction: "desc"}

	repos, _, err := client.Repositories.List(context.Background(), user, opt)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Recently updated repositories by %q: %v", user, github.Stringify(repos))
}

func ExampleRepositoriesService_CreateFile() {
	// In this example we're creating a new file in a repository using the
	// Contents API. Only 1 file per commit can be managed through that API.

	// Note that authentication is needed here as you are performing a modification
	// so you will need to modify the example to provide an oauth client to
	// github.NewClient() instead of nil. See the following documentation for more
	// information on how to authenticate with the client:
	// https://godoc.org/github.com/google/go-github/github#hdr-Authentication
	client := github.NewClient(nil)

	ctx := context.Background()
	fileContent := []byte("This is the content of my file\nand the 2nd line of it")

	// Note: the file needs to be absent from the repository as you are not
	// specifying a SHA reference here.
	opts := &github.RepositoryContentFileOptions{
		Message:   github.String("This is my commit message"),
		Content:   fileContent,
		Branch:    github.String("master"),
		Committer: &github.CommitAuthor{Name: github.String("FirstName LastName"), Email: github.String("user@example.com")},
	}
	_, _, err := client.Repositories.CreateFile(ctx, "myOrganization", "myRepository", "myNewFile.md", opts)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func ExampleUsersService_ListAll() {
	client := github.NewClient(nil)
	opts := &github.UserListOptions{}
	for {
		users, _, err := client.Users.ListAll(context.Background(), opts)
		if err != nil {
			log.Fatalf("error listing users: %v", err)
		}
		if len(users) == 0 {
			break
		}
		opts.Since = *users[len(users)-1].ID
		// Process users...
	}
}

func ExamplePullRequestsService_Create() {
	// In this example we're creating a PR and displaying the HTML url at the end.

	// Note that authentication is needed here as you are performing a modification
	// so you will need to modify the example to provide an oauth client to
	// github.NewClient() instead of nil. See the following documentation for more
	// information on how to authenticate with the client:
	// https://godoc.org/github.com/google/go-github/github#hdr-Authentication
	client := github.NewClient(nil)

	newPR := &github.NewPullRequest{
		Title:               github.String("My awesome pull request"),
		Head:                github.String("branch_to_merge"),
		Base:                github.String("master"),
		Body:                github.String("This is the description of the PR created with the package `github.com/google/go-github/github`"),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(context.Background(), "myOrganization", "myRepository", newPR)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
}

func ExampleTeamsService_ListTeams() {
	// This example shows how to get a team ID corresponding to a given team name.

	// Note that authentication is needed here as you are performing a lookup on
	// an organization's administrative configuration, so you will need to modify
	// the example to provide an oauth client to github.NewClient() instead of nil.
	// See the following documentation for more information on how to authenticate
	// with the client:
	// https://godoc.org/github.com/google/go-github/github#hdr-Authentication
	client := github.NewClient(nil)

	teamName := "Developers team"
	ctx := context.Background()
	opts := &github.ListOptions{}

	for {
		teams, resp, err := client.Teams.ListTeams(ctx, "myOrganization", opts)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, t := range teams {
			if t.GetName() == teamName {
				fmt.Printf("Team %q has ID %d\n", teamName, t.GetID())
				return
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Printf("Team %q was not found\n", teamName)
}
