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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRun(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		dir         string
		args        []string
		deployments []string
		pods        []string
		env         []string
	}{
		{
			description: "getting-started",
			dir:         "examples/getting-started",
			pods:        []string{"getting-started"},
		},
		{
			description: "nodejs",
			dir:         "examples/nodejs",
			deployments: []string{"node"},
		},
		{
			description: "structure-tests",
			dir:         "examples/structure-tests",
			pods:        []string{"getting-started"},
		},
		{
			description: "microservices",
			dir:         "examples/microservices",
			// See https://github.com/GoogleContainerTools/skaffold/issues/2372
			args:        []string{"--status-check=false"},
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		{
			description: "envTagger",
			dir:         "examples/tagging-with-environment-variables",
			pods:        []string{"getting-started"},
			env:         []string{"FOO=foo"},
		},
		{
			description: "bazel",
			dir:         "examples/bazel",
			pods:        []string{"bazel"},
		},
		{
			description: "jib",
			dir:         "testdata/jib",
			deployments: []string{"web"},
		},
		{
			description: "jib gradle",
			dir:         "examples/jib-gradle",
			deployments: []string{"web"},
		},
		{
			description: "profiles",
			dir:         "examples/profiles",
			args:        []string{"-p", "minikube-profile"},
			pods:        []string{"hello-service"},
		},
		{
			description: "multiple deployers",
			dir:         "testdata/deploy-multiple",
			pods:        []string{"deploy-kubectl", "deploy-kustomize"},
		},
		{
			description: "custom builder",
			dir:         "examples/custom",
			pods:        []string{"getting-started"},
		},
		{
			description: "buildpacks Go",
			dir:         "examples/buildpacks",
			deployments: []string{"web"},
		},
		{
			description: "buildpacks NodeJS",
			dir:         "examples/buildpacks-node",
			deployments: []string{"web"},
		},
		{
			description: "buildpacks Python",
			dir:         "examples/buildpacks-python",
			deployments: []string{"web"},
		},
		{
			description: "buildpacks Java",
			dir:         "examples/buildpacks-java",
			deployments: []string{"web"},
		},
		{
			description: "kustomize",
			dir:         "examples/getting-started-kustomize",
			deployments: []string{"skaffold-kustomize-dev"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ns, client := SetupNamespace(t)

			skaffold.Run(test.args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)

			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}

func TestRunRenderOnly(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	testutil.Run(t, "write rendered manifest to provided filepath", func(tu *testutil.T) {
		tmpDir := tu.NewTempDir()
		renderPath := tmpDir.Path("output.yaml")

		test := struct {
			description string
			renderPath  string
			args        []string
			dir         string
			pods        []string
		}{
			args: []string{"--render-only", "--render-output", renderPath},
			dir:  "examples/getting-started",
			pods: []string{"getting-started"},
		}

		skaffold.Run(test.args...).InDir(test.dir).RunOrFail(t)

		dat, err := ioutil.ReadFile(renderPath)
		tu.CheckNoError(err)

		tu.CheckMatches("name: getting-started", string(dat))
	})
}

func TestRunGCPOnly(t *testing.T) {
	MarkIntegrationTest(t, NeedsGcp)

	tests := []struct {
		description string
		dir         string
		args        []string
		deployments []string
		pods        []string
	}{
		{
			description: "Google Cloud Build",
			dir:         "examples/google-cloud-build",
			pods:        []string{"getting-started"},
		},
		{
			description: "Google Cloud Build with sub folder",
			dir:         "testdata/gcb-sub-folder",
			pods:        []string{"getting-started"},
		},
		{
			description: "Google Cloud Build with Kaniko",
			dir:         "examples/gcb-kaniko",
			pods:        []string{"getting-started-kaniko"},
		},
		{
			description: "kaniko",
			dir:         "examples/kaniko",
			pods:        []string{"getting-started-kaniko"},
		},
		{
			description: "kaniko with target",
			dir:         "testdata/kaniko-target",
			pods:        []string{"getting-started-kaniko"},
		},
		{
			description: "kaniko with sub folder",
			dir:         "testdata/kaniko-sub-folder",
			pods:        []string{"getting-started-kaniko"},
		},
		{
			description: "kaniko microservices",
			dir:         "testdata/kaniko-microservices",
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		{
			description: "jib in googlecloudbuild",
			dir:         "testdata/jib",
			args:        []string{"-p", "gcb"},
			deployments: []string{"web"},
		},
		{
			description: "jib gradle in googlecloudbuild",
			dir:         "examples/jib-gradle",
			args:        []string{"-p", "gcb"},
			deployments: []string{"web"},
		},
		{
			description: "buildpacks on Cloud Build",
			dir:         "examples/buildpacks",
			args:        []string{"-p", "gcb"},
			deployments: []string{"web"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ns, client := SetupNamespace(t)

			skaffold.Run(test.args...).InDir(test.dir).InNs(ns.Name).RunOrFail(t)

			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)
		})
	}
}

func TestRunIdempotent(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

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
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	output, err := skaffold.Run().InDir("testdata/unstable-deployment").InNs(ns.Name).RunWithCombinedOutput(t)
	if err == nil {
		t.Errorf("expected to see an error since the deployment is not stable: %s", output)
	} else if !strings.Contains(string(output), "unstable-deployment failed") {
		t.Errorf("failed without saying the reason: %s", output)
	}
}

func TestRunUnstableNotChecked(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	skaffold.Run("--status-check=false").InDir("testdata/unstable-deployment").InNs(ns.Name).RunOrFail(t)
}

func TestRunTailPod(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	out := skaffold.Run("--tail", "-p", "pod").InDir("testdata/hello").InNs(ns.Name).RunBackground(t)

	WaitForLogs(t, out,
		"Hello world! 0",
		"Hello world! 1",
		"Hello world! 2",
	)
}

func TestRunTailDeployment(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	out := skaffold.Run("--tail", "-p", "deployment").InDir("testdata/hello").InNs(ns.Name).RunBackground(t)

	WaitForLogs(t, out,
		"Hello world! 0",
		"Hello world! 1",
		"Hello world! 2",
	)
}
