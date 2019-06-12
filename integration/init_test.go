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

func TestInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	tests := []struct {
		name string
		dir  string
		args []string
	}{
		{
			name: "getting-started",
			dir:  "testdata/init/hello",
		},
		{
			name: "microservices",
			dir:  "testdata/init/microservices",
			args: []string{
				"-a", "leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app",
				"-a", "leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web",
			},
		},
		{
			name: "compose",
			dir:  "testdata/init/compose",
			args: []string{"--compose-file", "docker-compose.yaml"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, _, deleteNs := SetupNamespace(t)
			defer deleteNs()

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t)

			skaffold.Run().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t)
		})
	}
}
