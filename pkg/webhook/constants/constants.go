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

package constants

const (
	// GithubAccessToken is the env variable auth for container-tools-bot is stored in
	GithubAccessToken = "GITHUB_ACCESS_TOKEN"
	// GithubOwner is the owner of the repository
	GithubOwner = "GoogleContainerTools"
	// GithubRepo is the name of the repo
	GithubRepo = "skaffold"
	// GithubEventHeader is the header key used to describe a github event
	GithubEventHeader = "X-GitHub-Event"
	// PullRequestEvent is the Github header value for pull requests
	PullRequestEvent = "pull_request"
	// OpenState is the state of an open PR
	OpenState = "open"

	// when a PR is closed
	ClosedAction = "closed"
	// when a PR is labeled
	LabeledAction = "labeled"

	// DocsLabel kicks off the controller when added to a PR
	DocsLabel = "docs-modifications"

	// Namespace is the namespace deployments and services will be created in
	Namespace = "default"

	// HugoPort is the port that hugo defaults to
	HugoPort = 1313

	// DeploymentImage is the image the controller deploys, must contain hugo and git
	DeploymentImage = "gcr.io/k8s-skaffold/docs-controller:latest"

	// LogsGCSBucket is the GCS bucket logs are uploaded to
	LogsGCSBucket = "webhook-logs"
)
