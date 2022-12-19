/*
Copyright 2022 The Skaffold Authors

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

package integration

import (
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

func TestDelete(t *testing.T) {
	var tests = []struct {
		description string
		dir         string
		args        []string
		pods        []string
		deployments []string
		env         []string
	}{
		{
			description: "getting-started",
			dir:         "testdata/getting-started",
			pods:        []string{"getting-started"},
		},
		{
			description: "microservices",
			dir:         "examples/microservices",
			args:        []string{"--status-check=false"},
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		{
			description: "multi-config-microservices",
			dir:         "examples/multi-config-microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		{
			description: "multiple deployers",
			dir:         "testdata/deploy-multiple",
			pods:        []string{"deploy-kubectl", "deploy-kustomize"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			ns, client := SetupNamespace(t)

			args := append(test.args, "--cache-artifacts=false")
			skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)

			client.WaitForPodsReady(test.pods...)
			client.waitForDeploymentsToStabilizeWithTimeout(time.Minute*2, test.deployments...)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}

func TestDeleteNonExistedHelmResource(t *testing.T) {
	var tests = []struct {
		description string
		dir         string
		env         []string
	}{
		{
			description: "helm deployment doesn't exist.",
			dir:         "testdata/helm",
			env:         []string{"TEST_NS=test-ns"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			ns, _ := SetupNamespace(t)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}
