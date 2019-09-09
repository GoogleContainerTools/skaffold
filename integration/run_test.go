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
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		description string
		dir         string
		filename    string
		args        []string
		deployments []string
		pods        []string
		env         []string
		gcpOnly     bool
	}{
		{
			description: "getting-started",
			dir:         "examples/getting-started",
			pods:        []string{"getting-started"},
		}, {
			description: "nodejs",
			dir:         "examples/nodejs",
			deployments: []string{"node"},
		}, {
			description: "structure-tests",
			dir:         "examples/structure-tests",
			pods:        []string{"getting-started"},
		}, {
			description: "microservices",
			dir:         "examples/microservices",
			// See https://github.com/GoogleContainerTools/skaffold/issues/2372
			args:        []string{"--status-check=false"},
			deployments: []string{"leeroy-app", "leeroy-web"},
		}, {
			description: "envTagger",
			dir:         "examples/tagging-with-environment-variables",
			pods:        []string{"getting-started"},
			env:         []string{"FOO=foo"},
		}, {
			description: "bazel",
			dir:         "examples/bazel",
			pods:        []string{"bazel"},
		}, {
			description: "Google Cloud Build",
			dir:         "examples/google-cloud-build",
			pods:        []string{"getting-started"},
			gcpOnly:     true,
		}, {
			description: "Google Cloud Build with sub folder",
			dir:         "testdata/gcb-sub-folder",
			pods:        []string{"getting-started"},
			gcpOnly:     true,
		}, {
			description: "kaniko",
			dir:         "examples/kaniko",
			pods:        []string{"getting-started-kaniko"},
			gcpOnly:     true,
		}, {
			description: "kaniko local",
			dir:         "examples/kaniko-local",
			pods:        []string{"getting-started-kaniko"},
			gcpOnly:     true,
		}, {
			description: "kaniko local with target",
			dir:         "testdata/kaniko-target",
			pods:        []string{"getting-started-kaniko"},
			gcpOnly:     true,
		}, {
			description: "kaniko local with sub folder",
			dir:         "testdata/kaniko-sub-folder",
			pods:        []string{"getting-started-kaniko"},
			gcpOnly:     true,
		}, {
			description: "kaniko microservices",
			dir:         "testdata/kaniko-microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
			gcpOnly:     true,
		}, {
			description: "jib",
			dir:         "testdata/jib",
			deployments: []string{"web"},
		}, {
			description: "jib in googlecloudbuild",
			dir:         "testdata/jib",
			args:        []string{"-p", "gcb"},
			deployments: []string{"web"},
			gcpOnly:     true,
		}, {
			description: "jib gradle",
			dir:         "testdata/jib-gradle",
			deployments: []string{"web"},
		}, {
			description: "jib gradle in googlecloudbuild",
			dir:         "testdata/jib-gradle",
			args:        []string{"-p", "gcb"},
			deployments: []string{"web"},
			gcpOnly:     true,
		}, {
			description: "custom builder",
			dir:         "testdata/custom",
			pods:        []string{"bazel"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.gcpOnly && !ShouldRunGCPOnlyTests() {
				t.Skip("skipping gcp only test")
			}
			if !test.gcpOnly && ShouldRunGCPOnlyTests() {
				t.Skip("skipping test that is not gcp only")
			}

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			skaffold.Run(test.args...).WithConfig(test.filename).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)

			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)

			skaffold.Delete().WithConfig(test.filename).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}

func TestRunIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	// The first `skaffold run` creates resources (deployment.apps/leeroy-web, service/leeroy-app, deployment.apps/leeroy-app)
	out := skaffold.Run("-l", "skaffold.dev/run-id=notunique").InDir("examples/microservices").InNs(ns.Name).RunOrFailOutput(t)
	firstOut := string(out)
	if strings.Count(firstOut, "created") == 0 {
		t.Errorf("resources should have been created: %s", firstOut)
	}

	// Because we use the same custom `run-id`, the second `skaffold run` is idempotent:
	// + It has nothing to rebuild
	// + It leaves all resources unchanged
	out = skaffold.Run("-l", "skaffold.dev/run-id=notunique").InDir("examples/microservices").InNs(ns.Name).RunOrFailOutput(t)
	secondOut := string(out)
	if strings.Count(secondOut, "created") != 0 {
		t.Errorf("no resource should have been created: %s", secondOut)
	}
	if !strings.Contains(secondOut, "leeroy-web: Found") || !strings.Contains(secondOut, "leeroy-app: Found") {
		t.Errorf("both artifacts should be in cache: %s", secondOut)
	}
}

func TestRunUnstableChecked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	output, err := skaffold.Run("--status-check=true").InDir("testdata/unstable-deployment").InNs(ns.Name).RunWithCombinedOutput(t)
	if err == nil {
		t.Errorf("expected to see an error since the deployment is not stable: %s", output)
	} else if !strings.Contains(string(output), "deployment unstable-deployment failed") {
		t.Errorf("failed without saying the reason: %s", output)
	}
}

func TestRunUnstableNotChecked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	skaffold.Run().InDir("testdata/unstable-deployment").InNs(ns.Name).RunOrFail(t)
}
