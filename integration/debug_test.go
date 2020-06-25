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
	"encoding/json"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func TestDebug(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		config      string
		args        []string
		deployments []string
		pods        []string
	}{
		{
			description: "kubectl",
			deployments: []string{"java"},
			pods:        []string{"nodejs", "npm", "python3", "go"},
		},
		{
			description: "kustomize",
			args:        []string{"--profile", "kustomize"},
			deployments: []string{"java"},
			pods:        []string{"nodejs", "npm", "python3", "go"},
		},
		{
			description: "buildpacks",
			args:        []string{"--profile", "buildpacks"},
			deployments: []string{"java"},
			pods:        []string{"nodejs", "npm", "python3", "go"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build(test.args...).InDir("testdata/debug").RunOrFail(t)

			ns, client := SetupNamespace(t)

			skaffold.Debug(test.args...).InDir("testdata/debug").InNs(ns.Name).RunBackground(t)

			verifyDebugAnnotations := func(annotations map[string]string) {
				var configs map[string]debug.ContainerDebugConfiguration
				if anno, found := annotations["debug.cloud.google.com/config"]; !found {
					t.Errorf("deployment missing debug annotation: %v", annotations)
				} else if err := json.Unmarshal([]byte(anno), &configs); err != nil {
					t.Errorf("error unmarshalling debug annotation: %v: %v", anno, err)
				} else {
					for k, config := range configs {
						if config.WorkingDir == "" {
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

func TestDebugEventsRPC_StatusCheck(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/jib").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Debug("--enable-rpc", "--rpc-port", rpcAddr).InDir("testdata/jib").InNs(ns.Name).RunBackground(t)

	waitForDebugEvent(t, client, rpcAddr)
}

func TestDebugEventsRPC_NoStatusCheck(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/jib").RunOrFail(t)

	ns, client := SetupNamespace(t)

	rpcAddr := randomPort()
	skaffold.Debug("--enable-rpc", "--rpc-port", rpcAddr, "--status-check=false").InDir("testdata/jib").InNs(ns.Name).RunBackground(t)

	waitForDebugEvent(t, client, rpcAddr)
}

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
