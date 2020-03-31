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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/gcs"
	pkggithub "github.com/GoogleContainerTools/skaffold/pkg/webhook/github"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
)

const (
	port = ":8080"
)

func main() {
	// Setup the serve route to receive github events
	http.HandleFunc("/receive", handleGithubEvent)
	flag.Parse()
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
		commentOnGithub(event, "Error creating deployment, please see controller logs for details.")
		log.Printf("error handling pr event: %v", err)
	}
}

func handlePullRequestEvent(event *github.PullRequestEvent) error {
	log.Printf("handling pull request event: %+v", event)
	// Cleanup any deployments if PR was merged or closed
	if event.GetAction() == constants.ClosedAction {
		return kubernetes.CleanupDeployment(event)
	}

	// Only continue if the docs-modifications label was added
	if event.GetAction() != constants.LabeledAction {
		return nil
	}

	prNumber := event.GetNumber()

	if event.PullRequest.GetState() != constants.OpenState {
		log.Printf("Pull request %d is either merged or closed, skipping docs deployment", prNumber)
		return nil
	}

	if !labels.DocsLabelExists(event.GetPullRequest().Labels) {
		log.Printf("Label %s not found on PR %d", constants.DocsLabel, prNumber)
		return nil
	}

	// If a PR was relabeled, we need to first cleanup preexisting deployments
	if err := kubernetes.CleanupDeployment(event); err != nil {
		return fmt.Errorf("cleaning up deployment: %w", err)
	}

	// Create service for the PR and get the associated external IP
	log.Printf("Label %s found on PR %d, creating service", constants.DocsLabel, prNumber)
	svc, err := kubernetes.CreateService(event)
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}

	ip, err := kubernetes.GetExternalIP(svc)
	if err != nil {
		return fmt.Errorf("getting external IP: %w", err)
	}

	// Create a deployment which maps to the service
	log.Printf("Creating deployment for pull request %d", prNumber)
	deployment, err := kubernetes.CreateDeployment(event, svc, ip)
	if err != nil {
		return fmt.Errorf("creating deployment for PR %d: %w", prNumber, err)
	}
	response := succeeded
	if err := kubernetes.WaitForDeploymentToStabilize(deployment, ip); err != nil {
		log.Printf("Deployment didn't stabilize, commenting with failure message...")
		response = failed
	}

	msg, err := response(deployment, event, ip)
	if err != nil {
		return fmt.Errorf("getting github message: %w", err)
	}

	if err := commentOnGithub(event, msg); err != nil {
		return fmt.Errorf("commenting on github: %w", err)
	}

	return nil
}

func succeeded(d *appsv1.Deployment, event *github.PullRequestEvent, ip string) (string, error) {
	baseURL := kubernetes.BaseURL(ip)
	return fmt.Sprintf("Please visit [%s](%s) to view changes to the docs.", baseURL, baseURL), nil
}

func failed(d *appsv1.Deployment, event *github.PullRequestEvent, ip string) (string, error) {
	name, err := gcs.UploadDeploymentLogsToBucket(d, event.GetNumber())
	if err != nil {
		return "", fmt.Errorf("uploading logs to bucket: %w", err)
	}
	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", constants.LogsGCSBucket, name)
	return fmt.Sprintf("Error creating deployment %s, please visit %s to view logs.", d.Name, url), nil
}

func commentOnGithub(event *github.PullRequestEvent, msg string) error {
	githubClient := pkggithub.NewClient()
	if err := githubClient.CommentOnPR(event, msg); err != nil {
		return fmt.Errorf("commenting on PR %d: %w", event.GetNumber(), err)
	}
	if err := githubClient.RemoveLabelFromPR(event, constants.DocsLabel); err != nil {
		return fmt.Errorf("removing %s label from PR %d: %w", constants.DocsLabel, event.GetNumber(), err)
	}
	return nil
}
