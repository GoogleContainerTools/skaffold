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
)

func TestDebug(t *testing.T) {
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
		remoteOnly  bool
	}{
		{
			description: "jib+kubectl",
			dir:         "testdata/jib",
			deployments: []string{"web"},
		},
		{
			description: "jib+kustomize",
			args:        []string{"--profile", "kustomize"},
			dir:         "testdata/jib",
			deployments: []string{"web"},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Debug(test.args...).WithConfig(test.filename).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunBackground(t)
			defer stop()

			client.WaitForPodsReady(test.pods...)
			client.WaitForDeploymentsToStabilize(test.deployments...)
			for _, depName := range test.deployments {
				deploy := client.GetDeployment(depName)
				annotations := deploy.Spec.Template.GetAnnotations()
				if _, found := annotations["debug.cloud.google.com/config"]; !found {
					t.Errorf("deployment missing debug annotation: %v", annotations)
				}
			}

			skaffold.Delete().WithConfig(test.filename).InDir(test.dir).InNs(ns.Name).WithEnv(test.env).RunOrFail(t)
		})
	}
}
