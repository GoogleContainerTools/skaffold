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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	emptydir = "testdata/empty-dir"
)

// Note: `custom-buildx` is not included as it depends on having a
// `skaffold-builder` builder configured and a registry to push to.
// TODO: remove nolint once we've reenabled integration tests
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
		description: "copying-empty-directory",
		dir:         emptydir,
		pods:        []string{"empty-dir"},
		targetLog:   "Hello world!",
	},
	{
		description: "getting-started",
		dir:         "examples/getting-started",
		pods:        []string{"getting-started"},
		targetLog:   "Hello world!",
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
		description: "custom-tests",
		dir:         "examples/custom-tests",
		pods:        []string{"custom-test"},
	},
	{
		description: "microservices",
		dir:         "examples/microservices",
		// See https://github.com/GoogleContainerTools/skaffold/issues/2372
		args:        []string{"--status-check=false"},
		deployments: []string{"leeroy-app", "leeroy-web"},
	},
	{
		description: "multi-config-microservices",
		dir:         "examples/multi-config-microservices",
		deployments: []string{"leeroy-app", "leeroy-web"},
	},
	{
		description: "remote-multi-config-microservices",
		dir:         "examples/remote-multi-config-microservices",
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
		pods:        []string{"getting-started-custom"},
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
		targetLog:   "Hello world!",
	},
	{
		description: "helm",
		dir:         "examples/helm-deployment",
		deployments: []string{"skaffold-helm"},
		targetLog:   "Hello world!",
	},
	{
		description: "multiple renderers mixed in",
		dir:         "examples/multiple-renderers",
		deployments: []string{"frontend", "backend", "go-guestbook-mongodb"},
	},
	{
		description: "multiple renderers mixed in",
		dir:         "examples/multiple-renderers",
		args:        []string{"-p", "mix-deploy"},
		deployments: []string{"frontend", "backend", "go-guestbook-mongodb"},
	},
}

func TestRun(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
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

func TestRunTail(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.targetLog == "" {
				t.SkipNow()
			}
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}
			ns, _ := SetupNamespace(t)

			args := append(test.args, "--tail")
			out := skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunLive(t)

			WaitForLogs(t, out, test.targetLog)

			skaffold.Delete().InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}

func TestRunTailDefaultNamespace(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.targetLog == "" {
				t.SkipNow()
			}
			if test.dir == emptydir {
				err := os.MkdirAll(filepath.Join(test.dir, "emptydir"), 0755)
				t.Log("Creating empty directory")
				if err != nil {
					t.Errorf("Error creating empty dir: %s", err)
				}
			}

			args := append(test.args, "--tail")
			out := skaffold.Run(args...).InDir(test.dir).WithEnv(test.env).RunLive(t)
			defer skaffold.Delete().InDir(test.dir).WithEnv(test.env).RunOrFail(t)
			WaitForLogs(t, out, test.targetLog)
		})
	}
}

func TestRunRenderOnly(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

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
			args: []string{"--digest-source=local", "--render-only", "--render-output", renderPath},
			dir:  "examples/getting-started",
			pods: []string{"getting-started"},
		}

		skaffold.Run(test.args...).InDir(test.dir).RunOrFail(t)

		dat, err := os.ReadFile(renderPath)
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
		// {
		//	description: "Google Cloud Build",
		//	dir:         "examples/google-cloud-build",
		//	pods:        []string{"getting-started"},
		// },
		{
			description: "Google Cloud Build with sub folder",
			dir:         "testdata/gcb-sub-folder",
			pods:        []string{"getting-started"},
		},
		{
			description: "Google Cloud Build with location",
			dir:         "testdata/gcb-with-location",
			pods:        []string{"getting-started"},
		},
		{
			description: "Google Cloud Build with source artifact dependencies",
			dir:         "examples/microservices",
			args:        []string{"-p", "gcb"},
			deployments: []string{"leeroy-app", "leeroy-web"},
		},
		// {
		//	description: "Google Cloud Build with Kaniko",
		//	dir:         "examples/gcb-kaniko",
		//	pods:        []string{"getting-started-kaniko"},
		// },
		// {
		//	description: "kaniko",
		//	dir:         "examples/kaniko",
		//	pods:        []string{"getting-started-kaniko"},
		// },
		// {
		//	description: "kaniko with target",
		//	dir:         "testdata/kaniko-target",
		//	pods:        []string{"getting-started-kaniko"},
		// },
		{
			description: "kaniko with sub folder",
			dir:         "testdata/kaniko-sub-folder",
			pods:        []string{"getting-started-kaniko"},
		},
		// {
		//	description: "kaniko microservices",
		//	dir:         "testdata/kaniko-microservices",
		//	deployments: []string{"leeroy-app", "leeroy-web"},
		// },
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
		if (os.Getenv("GKE_CLUSTER_NAME") == "integration-tests-arm" || os.Getenv("GKE_CLUSTER_NAME") == "integration-tests-hybrid") && strings.Contains(test.description, "buildpacks") {
			continue // buildpacks doesn't support arm64 builds, so skip run on these clusters
		}
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

	out := skaffold.Run("--tail", "-p", "pod").InDir("testdata/hello").InNs(ns.Name).RunLive(t)

	WaitForLogs(t, out,
		"Hello world! 0",
		"Hello world! 1",
		"Hello world! 2",
	)
}

func TestRunTailDeployment(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	ns, _ := SetupNamespace(t)

	out := skaffold.Run("--tail", "-p", "deployment").InDir("testdata/hello").InNs(ns.Name).RunLive(t)

	WaitForLogs(t, out,
		"Hello world! 0",
		"Hello world! 1",
		"Hello world! 2",
	)
}

func TestRunTest(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		testDir      string
		testFile     string
		args         []string
		skipTests    bool
		expectedText string
	}{
		{
			description:  "Run test",
			testDir:      "testdata/custom-test",
			testFile:     "testdata/custom-test/runtest",
			args:         []string{"--profile", "custom"},
			skipTests:    false,
			expectedText: "foo\n",
		},
		{
			description:  "Run test with skip test false",
			testDir:      "testdata/custom-test",
			testFile:     "testdata/custom-test/runtest",
			args:         []string{"--profile", "custom", "--skip-tests=false"},
			skipTests:    false,
			expectedText: "foo\n",
		},
		{
			description: "Run test with skip test true",
			testDir:     "testdata/custom-test",
			testFile:    "testdata/custom-test/runtest",
			args:        []string{"--profile", "custom", "--skip-tests=True"},
			skipTests:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer os.Remove(test.testFile)

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir(test.testDir).RunOrFail(t)

			ns, client := SetupNamespace(t)
			skaffold.Run(test.args...).InDir(test.testDir).InNs(ns.Name).RunLive(t)

			client.WaitForPodsReady("custom-test-example")

			err := wait.PollImmediate(time.Millisecond*500, 1*time.Minute, func() (bool, error) {
				_, e := os.Stat(test.testFile)
				if test.skipTests {
					if !os.IsNotExist(e) {
						t.Fatalf("Tests are not skipped.")
					}
					return true, nil
				}
				out, e := os.ReadFile(test.testFile)
				failNowIfError(t, e)
				return string(out) == test.expectedText, nil
			})
			failNowIfError(t, err)
		})
	}
}

// TestRunNoOptFlags tests to ensure that flags that don't require a value to be passed work when no value is passed
func TestRunNoOptFlags(t *testing.T) {
	test := struct {
		description string
		dir         string
		targetLog   string
		pods        []string
		args        []string
	}{
		description: "getting-started",
		dir:         "testdata/getting-started",
		pods:        []string{"getting-started"},
		targetLog:   "Hello world!",
		args: []string{
			"--port-forward",
			"--status-check",
		},
	}

	MarkIntegrationTest(t, CanRunWithoutGcp)
	t.Run(test.description, func(t *testing.T) {
		ns, _ := SetupNamespace(t)

		args := append(test.args, "--tail")
		out := skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).RunLive(t)
		defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)

		WaitForLogs(t, out, test.targetLog)
	})
}

func TestRunWithMultiPlatform(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	type image struct {
		name string
		pod  string
	}

	tests := []struct {
		description       string
		dir               string
		images            []image
		tag               string
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "Run with multiplatform linux/arm64 and linux/amd64",
			dir:               "examples/cross-platform-builds",
			images:            []image{{name: "skaffold-example", pod: "getting-started"}},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description: "Run with multiplatform linux/arm64 and linux/amd64 in a multi config project",
			dir:         "testdata/multi-config-pods",
			images: []image{
				{name: "multi-config-module1", pod: "module1"},
				{name: "multi-config-module2", pod: "module2"},
			},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defaultRepo := "gcr.io/k8s-skaffold"

			platforms := platformsCliValue(t, test.expectedPlatforms)
			ns, client := SetupNamespace(t)
			args := []string{"--platform", platforms, "--default-repo", defaultRepo, "--tag", test.tag}

			skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).RunOrFail(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)

			for _, image := range test.images {
				pod := client.GetPod(image.pod)

				checkRemoteImagePlatforms(t, fmt.Sprintf("%s/%s:%s", defaultRepo, image.name, test.tag), test.expectedPlatforms)
				checkNodeAffinity(t, test.expectedPlatforms, pod)
			}
		})
	}
}

func checkNodeAffinity(t *testing.T, expectedPlatforms []v1.Platform, pod *k8sv1.Pod) {
	t.Helper()
	failIfNodeAffinityNotSet(t, pod)
	nodeAffinityPlatforms := getPlatformsFromNodeAffinity(pod)
	checkPlatformsEqual(t, nodeAffinityPlatforms, expectedPlatforms)
}

func failIfNodeAffinityNotSet(t *testing.T, pod *k8sv1.Pod) {
	if pod.Spec.Affinity == nil {
		t.Fatalf("Affinity not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity == nil {
		t.Fatalf("NodeAffinity not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatalf("RequiredDuringSchedulingIgnoredDuringExecution not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms == nil {
		t.Fatalf("NodeSelectorTerms not defined in spec")
	}
}

func getPlatformsFromNodeAffinity(pod *k8sv1.Pod) []v1.Platform {
	var platforms []v1.Platform
	nodeAffinityPlatforms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	for _, np := range nodeAffinityPlatforms {
		os, arch := "", ""
		for _, me := range np.MatchExpressions {
			if me.Key == "kubernetes.io/os" {
				os = strings.Join(me.Values, "")
			}

			if me.Key == "kubernetes.io/arch" {
				arch = strings.Join(me.Values, "")
			}
		}

		platforms = append(platforms, v1.Platform{OS: os, Architecture: arch})
	}

	return platforms
}

func platformsCliValue(t *testing.T, platforms []v1.Platform) string {
	t.Helper()

	var platformsCliValue []string
	for _, platform := range platforms {
		platformsCliValue = append(platformsCliValue, fmt.Sprintf("%s/%s", platform.OS, platform.Architecture))
	}

	return strings.Join(platformsCliValue, ",")
}
