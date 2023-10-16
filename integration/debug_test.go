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
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDebug(t *testing.T) {
	tests := []struct {
		description   string
		dir           string
		config        string
		args          []string
		deployments   []string
		pods          []string
		ignoreWorkdir bool
	}{
		{
			description: "kubectl",
			dir:         "testdata/debug",
			deployments: []string{"java"},
			pods:        []string{"nodejs", "npm" /*, "python3"*/, "go" /*, "netcore"*/},
		},
		{
			description: "kustomize",
			dir:         "testdata/debug",
			args:        []string{"--profile", "kustomize"},
			deployments: []string{"java"},
			pods:        []string{"nodejs", "npm" /*, "python3"*/, "go" /*, "netcore"*/},
		},
		// TODO(#8811): Enable this test when issue is solve.
		// {
		// 	description: "buildpacks",
		// 	dir:         "testdata/debug",
		// 	args:        []string{"--profile", "buildpacks"},
		// 	deployments: []string{"java"},
		// 	pods:        []string{"nodejs", "npm" /*, "python3"*/, "go" /*, "netcore"*/},
		// },
		{
			description:   "helm",
			dir:           "examples/helm-deployment",
			deployments:   []string{"skaffold-helm"},
			ignoreWorkdir: true, // dockerfile doesn't have a workdir
		},
		{
			description:   "modules",
			dir:           "examples/multi-config-microservices",
			deployments:   []string{"leeroy-web", "leeroy-app"},
			ignoreWorkdir: true, // dockerfile doesn't have a workdir
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build(test.args...).InDir(test.dir).RunOrFail(t)

			ns, client := SetupNamespace(t)

			skaffold.Debug(test.args...).InDir(test.dir).InNs(ns.Name).RunBackground(t)

			verifyDebugAnnotations := func(annotations map[string]string) {
				var configs map[string]types.ContainerDebugConfiguration
				if anno, found := annotations["debug.cloud.google.com/config"]; !found {
					t.Errorf("deployment missing debug annotation: %v", annotations)
				} else if err := json.Unmarshal([]byte(anno), &configs); err != nil {
					t.Errorf("error unmarshalling debug annotation: %v: %v", anno, err)
				} else {
					for k, config := range configs {
						if !test.ignoreWorkdir && config.WorkingDir == "" {
							t.Errorf("debug config for %q missing WorkingDir: %v: %v", k, anno, config)
						}
						if config.Runtime == "" {
							t.Errorf("debug config for %q missing Runtime: %v: %v", k, anno, config)
						}
					}
				}
			}

			for _, podName := range test.pods {
				pod := client.GetPod(podName)

				annotations := pod.Annotations
				verifyDebugAnnotations(annotations)
			}

			for _, depName := range test.deployments {
				deploy := client.GetDeployment(depName)

				annotations := deploy.Spec.Template.GetAnnotations()
				verifyDebugAnnotations(annotations)
			}
		})
	}
}

func TestDockerDebug(t *testing.T) {
	t.Run("debug docker deployment", func(t *testing.T) {
		MarkIntegrationTest(t, CanRunWithoutGcp)
		skaffold.Build("-p", "docker").InDir("testdata/debug").RunOrFail(t)

		skaffold.Debug("-p", "docker").InDir("testdata/debug").RunBackground(t)
		defer skaffold.Delete("-p", "docker").InDir("testdata/debug").RunBackground(t)

		// use docker client to verify container has been created properly
		// check this container and verify entrypoint has been rewritten
		client := SetupDockerClient(t)
		var (
			verifyEntrypointRewrite bool
			verifySupportContainer  bool
			tries                   = 0
			sleepTime               = 2 * time.Second // retrieve containers every two seconds
			maxTries                = 15              // try for 30 seconds max
		)
		for {
			containers, err := client.ContainerList(context.Background(), dockertypes.ContainerListOptions{All: true})
			if err != nil {
				t.Fail()
			}
			time.Sleep(sleepTime)
			if len(containers) == 0 {
				continue
			}

			checkEntrypointRewrite(containers, &verifyEntrypointRewrite)
			checkSupportContainer(containers, &verifySupportContainer)

			if verifyEntrypointRewrite && verifySupportContainer {
				break
			}
			tries++
			if tries == maxTries {
				break
			}
		}

		if !verifyEntrypointRewrite {
			t.Error("couldn't verify rewritten container")
		}
		if !verifySupportContainer {
			t.Error("couldn't verify support container was created")
		}
	})
}

func checkEntrypointRewrite(containers []dockertypes.Container, found *bool) {
	if *found {
		return
	}
	for _, c := range containers {
		if strings.Contains(c.Command, "docker-entrypoint.sh") {
			*found = true
		}
	}
}

func checkSupportContainer(containers []dockertypes.Container, found *bool) {
	if *found {
		return
	}
	for _, c := range containers {
		if strings.Contains(c.Image, "gcr.io/k8s-skaffold/skaffold-debug-support") {
			*found = true
		}
	}
}

func TestFilterWithDebugging(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	// `filter` currently expects to receive a digested yaml
	renderedOutput := skaffold.Render("--digest-source=local").InDir("examples/getting-started").RunOrFailOutput(t)

	testutil.Run(t, "no --build-artifacts should transform all images", func(t *testutil.T) {
		transformedOutput := skaffold.Filter("--debugging").InDir("examples/getting-started").WithStdin(renderedOutput).RunOrFailOutput(t.T)
		transformedYaml := string(transformedOutput)
		if !strings.Contains(transformedYaml, "/dbg/go/bin/dlv") {
			t.Error("transformed yaml seems to be missing debugging details", transformedYaml)
		}
	})

	testutil.Run(t, "--build-artifacts=file should result in specific transforms", func(t *testutil.T) {
		buildFile := t.TempFile("build.txt", []byte(`{"builds":[{"imageName":"doesnotexist","tag":"doesnotexist:notag"}]}`))
		transformedOutput := skaffold.Filter("--debugging", "--build-artifacts="+buildFile).InDir("examples/getting-started").WithStdin(renderedOutput).RunOrFailOutput(t.T)
		transformedYaml := string(transformedOutput)
		if strings.Contains(transformedYaml, "/dbg/go/bin/dlv") {
			t.Error("transformed yaml should not include debugging details", transformedYaml)
		}
	})
}

func TestDebugEventsRPC_StatusCheck(t *testing.T) {
	t.Skipf("Disable the test dues to flakyness. https://github.com/GoogleContainerTools/skaffold/issues/7405")
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/jib").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Debug("--rpc-port", rpcAddr).InDir("testdata/jib").InNs(ns.Name).RunBackground(t)

	waitForDebugEvent(t, client, rpcAddr)
}

func TestDebugEventsRPC_NoStatusCheck(t *testing.T) {
	t.Skipf("Disable the test dues to flakyness. https://github.com/GoogleContainerTools/skaffold/issues/7405")
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/jib").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Debug("--rpc-port", rpcAddr, "--status-check=false").InDir("testdata/jib").InNs(ns.Name).RunBackground(t)

	waitForDebugEvent(t, client, rpcAddr)
}

// nolint add lint back after https://github.com/GoogleContainerTools/skaffold/issues/7405
func waitForDebugEvent(t *testing.T, client *NSKubernetesClient, rpcAddr string) {
	client.WaitForPodsReady()

	_, entries := apiEvents(t, rpcAddr)

	timeout := time.After(1 * time.Minute)
	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for debugging event")
		case entry := <-entries:
			switch entry.Event.GetEventType().(type) {
			case *proto.Event_DebuggingContainerEvent:
				// success!
				return
			default:
			}
		}
	}
}
