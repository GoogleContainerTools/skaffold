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

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
)

const (
	emptydir = "testdata/empty-dir"
)

// Note: `custom-buildx` is not included as it depends on having a
// `skaffold-builder` builder configured and a registry to push to.
// TODO: remove nolint once we've reenabled integration tests
//
//nolint:golint,unused
var tests = []struct {
	description string
	dir         string
	args        []string
	deployments []string
	pods        []string
	env         []string
	targetLog   string
}{
	{
		description: "buildpacks NodeJS",
		dir:         "examples/buildpacks-node",
		deployments: []string{"web"},
	},
}

func TestRun(t *testing.T) {
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)

			ns, client := SetupNamespace(t)
			args := append(test.args, "--cache-artifacts=false")
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}
			skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)

			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}
