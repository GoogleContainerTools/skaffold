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
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInit(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
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
			name: "ignore existing tags",
			dir:  "testdata/init/ignore-tags",
		},
		{
			name: "microservices (backwards compatibility)",
			dir:  "testdata/init/microservices",
			args: []string{
				"-a", `leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app`,
				"-a", `leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web`,
			},
		},
		{
			name: "microservices",
			dir:  "testdata/init/microservices",
			args: []string{
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			initArgs := append([]string{"--force"}, test.args...)

			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

			checkGeneratedConfig(t, test.dir)

			// Make sure the skaffold yaml can be parsed
			skaffold.Diagnose().InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)
		})
	}
}

func TestInitCompose(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	tests := []struct {
		name string
		dir  string
		args []string
	}{
		{
			name: "compose",
			dir:  "testdata/init/compose",
			args: []string{"--compose-file", "docker-compose.yaml"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			ns, _, deleteNs := SetupNamespace(t.T)
			defer deleteNs()

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig("skaffold.yaml.out").RunOrFail(t.T)

			checkGeneratedConfig(t, test.dir)

			// Make sure the skaffold yaml and the kubernetes manifests created by kompose are ok
			skaffold.Run().InDir(test.dir).WithConfig("skaffold.yaml.out").InNs(ns.Name).RunOrFail(t.T)
		})
	}
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := ioutil.ReadFile(filepath.Join(dir, "skaffold.yaml"))
	t.CheckNoError(err)

	output, err := ioutil.ReadFile(filepath.Join(dir, "skaffold.yaml.out"))
	t.CheckNoError(err)
	t.CheckDeepEqual(string(expectedOutput), string(output))
}
