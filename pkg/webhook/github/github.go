/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
)

// Client provides the context and client with necessary auth
// for interacting with the Github API
type Client struct {
	ctx context.Context
	*github.Client
}

// NewClient returns a github client with the necessary auth
func NewClient() *Client {
	githubToken := os.Getenv(constants.GithubAccessToken)
	// Setup the token for github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Return a client instance from github
	client := github.NewClient(tc)
	return &Client{
		Client: client,
		ctx:    context.Background(),
	}
}

// CommentOnPR comments message on the PR
func (g *Client) CommentOnPR(pr *github.PullRequestEvent, message string) error {
	comment := &github.IssueComment{
		Body: &message,
	}

	log.Printf("Creating comment on PR %d: %s", pr.PullRequest.GetNumber(), message)
	_, _, err := g.Client.Issues.CreateComment(g.ctx, constants.GithubOwner, constants.GithubRepo, pr.PullRequest.GetNumber(), comment)
	if err != nil {
		return fmt.Errorf("creating github comment: %w", err)
	}
	log.Printf("Successfully commented on PR %d.", pr.GetNumber())
	return nil
}

// RemoveLabelFromPR removes label from pr
func (g *Client) RemoveLabelFromPR(pr *github.PullRequestEvent, label string) error {
	_, err := g.Client.Issues.RemoveLabelForIssue(g.ctx, constants.GithubOwner, constants.GithubRepo, pr.GetNumber(), label)
	if err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}
	log.Printf("Successfully deleted label from PR %d", pr.GetNumber())
	return nil
}
