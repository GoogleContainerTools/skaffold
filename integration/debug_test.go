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
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func TestDebug(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		dir         string
		args        []string
		deployments []string
		pods        []string
	}{
		{
			description: "kubectl",
			dir:         "testdata/debug",
			deployments: []string{"jib"},
			pods:        []string{"nodejs", "npm", "python3", "go"},
		},
		{
			description: "kustomize",
			args:        []string{"--profile", "kustomize"},
			dir:         "testdata/debug",
			deployments: []string{"jib"},
			pods:        []string{"nodejs", "npm", "python3", "go"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build(test.args...).InDir(test.dir).RunOrFail(t)

			ns, client := SetupNamespace(t)

			skaffold.Debug(test.args...).InDir(test.dir).InNs(ns.Name).RunBackground(t)

			client.WaitForPodsReady(test.pods...)
			for _, depName := range test.deployments {
				deploy := client.GetDeployment(depName)

				annotations := deploy.Spec.Template.GetAnnotations()
				if _, found := annotations["debug.cloud.google.com/config"]; !found {
					t.Errorf("deployment missing debug annotation: %v", annotations)
				}
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
			t.Fatalf("timed out waiting for port debugging event")
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
