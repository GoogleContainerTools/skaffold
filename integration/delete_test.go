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
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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
			description: "templated fields must exist",
			dir:         "testdata/helm-render-delete",
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

func TestDeleteDockerDeployer(t *testing.T) {
	tests := []struct {
		description        string
		dir                string
		args               []string
		deployedContainers []string
	}{
		{
			description:        "run with one container",
			dir:                "testdata/docker-deploy",
			args:               []string{"-p", "one-container"},
			deployedContainers: []string{"docker-bert-img-1"},
		},
		{
			description:        "run with more than one container",
			dir:                "testdata/docker-deploy",
			args:               []string{"-p", "more-than-one-container"},
			deployedContainers: []string{"docker-bert-img-2", "docker-ernie-img-2"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			ctx := context.Background()
			skaffold.Run(test.args...).InDir(test.dir).RunOrFail(t.T)
			skaffold.Delete(test.args...).InDir(test.dir).RunOrFail(t.T)

			client := SetupDockerClient(t.T)
			cs := getContainers(ctx, t, test.deployedContainers, client)
			t.CheckDeepEqual(0, len(cs))
		})
	}
}

func getContainers(ctx context.Context, t *testutil.T, deployedContainers []string, client docker.LocalDaemon) []types.Container {
	t.Helper()

	containersFilters := []filters.KeyValuePair{}
	for _, c := range deployedContainers {
		containersFilters = append(containersFilters, filters.Arg("name", c))
	}

	cl, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(containersFilters...),
	})
	t.CheckNoError(err)

	return cl
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
