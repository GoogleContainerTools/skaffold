/*
Copyright 2019 Cornelius Weig

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

/*****************************************************
 * NOTE The original version of this script is due to
 *    Balint Pato and was published as part of Skaffold
 *    (https://github.com/GoogleContainerTools/skaffold)
 *    under the following license:

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

*****************************************************/

// listpullreqs.go lists pull requests since the last release.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	token string
	org   string
	repo  string
)

const longDescription = `The script uses the GitHub API to retrieve a list of all merged pull
requests since the last release. The found pull requests are then
printed as markdown changelog with their commit summary and a link
to the pull request on GitHub.`

var rootCmd = &cobra.Command{
	Use:     "release-notes {org} {repo}",
	Example: "release-notes GoogleContainerTools skaffold",
	Short:   "Generate a markdown changelog of merged pull requests since last release",
	Long:    longDescription,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		org, repo = args[0], args[1]
		printPullRequests()
	},
}

func main() {
	rootCmd.Flags().StringVar(&token, "token", "", "Specify personal Github Token if you are hitting a rate limit anonymously. https://github.com/settings/tokens")
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func printPullRequests() {
	ctx := contextWithCtrlCHandler()
	client := getClient(ctx)

	releases, _, err := client.Repositories.ListReleases(ctx, org, repo, &github.ListOptions{})
	if err != nil {
		logrus.Fatalf("Failed to list releases: %v", err)
	}
	if len(releases) == 0 {
		logrus.Warningf("Could not find any releases for %s/%s", org, repo)
		return
	}
	lastReleaseTime := releases[0].GetPublishedAt()
	fmt.Printf("Collecting pull request that were merged since the last release: %s (%s)\n", releases[0].GetTagName(), lastReleaseTime)

	for page := 1; page != 0; {
		pullRequests, resp, err := client.PullRequests.List(ctx, org, repo, &github.PullRequestListOptions{
			State:     "closed",
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: 20,
				Page:    page,
			},
		})
		if err != nil {
			logrus.Fatalf("Failed to list pull requests: %v", err)
		}
		page = resp.NextPage

		for idx := range pullRequests {
			pr := pullRequests[idx]
			if pr.GetUpdatedAt().Before(lastReleaseTime.Time) {
				page = 0 // we are done now
				break
			}
			if pr.MergedAt != nil && pr.MergedAt.After(lastReleaseTime.Time) {
				fmt.Printf("* %s [#%d](https://github.com/%s/%s/pull/%d)\n", pr.GetTitle(), pr.GetNumber(), org, repo, pr.GetNumber())
			}
		}
	}
}

func getClient(ctx context.Context) *github.Client {
	if len(token) == 0 {
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func contextWithCtrlCHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)

	go func() {
		<-sigs
		signal.Stop(sigs)
		cancel()
		logrus.Infof("Aborted.")
	}()

	return ctx
}
