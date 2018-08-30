/*
Copyright 2018 The Skaffold Authors

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

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	token   string
	fromTag string
	toTag   string
)

var rootCmd = &cobra.Command{
	Use:        "listpullreqs fromTag toTag",
	Short:      "Lists pull requests between two versions in our changelog markdown format",
	ArgAliases: []string{"fromTag", "toTag"},
	Run: func(cmd *cobra.Command, args []string) {
		printPullRequests()
	},
}

const org = "GoogleContainerTools"
const repo = "skaffold"

func main() {
	rootCmd.Flags().StringVar(&token, "token", "", "Specify personal Github Token if you are hitting a rate limit anonymously. https://github.com/settings/tokens")
	rootCmd.Flags().StringVar(&fromTag, "fromTag", "", "comparison of commits is based on this tag (defaults to the latest tag in the repo)")
	rootCmd.Flags().StringVar(&toTag, "toTag", "master", "this is the commit that is compared with fromTag")
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func printPullRequests() {
	client := getClient()

	if len(fromTag) == 0 {
		tags, _, _ := client.Repositories.ListTags(context.Background(), org, repo, &github.ListOptions{})
		fromTag = tags[0].GetName()
	}

	fmt.Println(fmt.Sprintf("Collecting PR commits between %s and %s...", fromTag, toTag))
	fmt.Println("---------")

	comparison := commits(client)

	repositoryCommits := comparison.Commits

	mergeRe := regexp.MustCompile("Merge pull request #(.*) from.*")
	pullRequestCommitRe := regexp.MustCompile(".* \\(#(.*)\\)")
	for idx := range repositoryCommits {
		commit := repositoryCommits[idx]
		msg := *commit.Commit.Message
		match := mergeRe.FindStringSubmatch(msg)
		if match == nil {
			match = pullRequestCommitRe.FindStringSubmatch(msg)
			if match == nil {
				continue
			}
		}
		prID, _ := strconv.Atoi(match[1])

		pullRequest, _, _ := client.PullRequests.Get(context.Background(), org, repo, prID)
		fmt.Printf("* %s [#%d](https://github.com/%s/%s/pull/%d)\n", pullRequest.GetTitle(), prID, org, repo, prID)
	}
}

func getClient() *github.Client {
	if len(token) <= 0 {
		return github.NewClient(nil)
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func commits(client *github.Client) *github.CommitsComparison {
	commits, resp, e := client.Repositories.CompareCommits(context.Background(), org, repo, fromTag, toTag)
	if e != nil {
		fmt.Println(fmt.Errorf("error %s, %s", e, resp))
		os.Exit(1)
	}
	return commits
}
