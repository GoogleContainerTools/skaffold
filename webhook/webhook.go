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
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/google/go-github/github"
)

const (
	port = ":8080"
)

func main() {
	//Setup the serve route to receive guthub events
	http.HandleFunc("/receive", handleGithubEvent)

	// Start the server
	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleGithubEvent(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get(constants.GithubEventHeader)
	if eventType != constants.PullRequestEvent {
		return
	}
	event := new(github.PullRequestEvent)
	if err := json.NewDecoder(r.Body).Decode(event); err != nil {
		log.Printf("error decoding pr event: %v", err)
	}
	if err := handlePullRequestEvent(event); err != nil {
		log.Printf("error handling pr event: %v", err)
	}
}

func handlePullRequestEvent(event *github.PullRequestEvent) error {
	// Cleanup any deployments if PR was merged or closed
	if event.GetAction() == constants.ClosedAction {
		return kubernetes.CleanupDeployment(event)
	}

	// Only continue if the docs-modifications label was added
	if event.GetAction() != constants.LabeledAction {
		return nil
	}

	prNumber := event.GetNumber()

	if event.PullRequest.GetMerged() || event.PullRequest.ClosedAt == nil {
		log.Printf("Pull request %d is either merged or closed, skipping docs deployment", prNumber)
		return nil
	}

	if !labels.DocsLabelExists(event.GetPullRequest().Labels) {
		log.Printf("Label %s not found on PR %d", constants.DocsLabel, prNumber)
		return nil
	}

	log.Printf("Label %s found on PR %d, creating service", constants.DocsLabel, prNumber)

	svc, err := kubernetes.CreateService(event)
	if err != nil {
		return errors.Wrap(err, "creating service")
	}

	ip, err := kubernetes.GetExternalIP(svc)
	if err != nil {
		return errors.Wrap(err, "getting external IP")
	}

	log.Printf("External IP %s ready for PR %d", ip, prNumber)

	// TODO: priyawadhwa@ to add logic for creating a deployment mapping to a service here
	return nil
}
