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

package kubernetes

import (
	"log"
	"os/exec"

	"github.com/google/go-github/github"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
)

// CleanupDeployment cleans up all deployments related to the given pull request
func CleanupDeployment(pr *github.PullRequestEvent) error {
	log.Printf("Cleaning up deployments for PR %d", pr.GetNumber())
	selector := labels.Selector(pr.GetNumber())
	cmd := exec.Command("kubectl", "delete", "all", "--selector", selector)
	return cmd.Run()
}
