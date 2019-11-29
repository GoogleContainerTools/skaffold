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

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDebug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

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

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Debug(test.args...).InDir(test.dir).InNs(ns.Name).RunBackground(t)
			defer stop()

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

func TestDebugEventsRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	rpcAddr := randomPort()

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/jib").RunOrFail(t)

	ns, client, deleteNs := SetupNamespace(t)
	defer deleteNs()

	stop := skaffold.Debug("--enable-rpc", "--rpc-port", rpcAddr).InDir("testdata/jib").InNs(ns.Name).RunBackground(t)
	defer stop()

	client.WaitForPodsReady()

	// read a preset number of entries from the event log
	logEntries := retrieveGrpcLogEntries(t, rpcAddr, 6)

	metaEntries, buildEntries, deployEntries, debuggingEntries := 0, 0, 0, 0
	for _, entry := range logEntries {
		switch entry.Event.GetEventType().(type) {
		case *proto.Event_MetaEvent:
			metaEntries++
		case *proto.Event_BuildEvent:
			buildEntries++
		case *proto.Event_DeployEvent:
			deployEntries++
		case *proto.Event_DebuggingContainerEvent:
			debuggingEntries++
		default:
		}
	}
	// make sure we have exactly 1 meta entry, 2 deploy entries and 2 build entries
	testutil.CheckDeepEqual(t, 1, metaEntries)
	testutil.CheckDeepEqual(t, 2, deployEntries)
	testutil.CheckDeepEqual(t, 2, buildEntries)
	testutil.CheckDeepEqual(t, 1, debuggingEntries)
}
