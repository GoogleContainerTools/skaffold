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
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name             string
		dir              string
		args             []string
		skipSkaffoldYaml bool
	}{
		{
			name: "getting-started",
			dir:  "../examples/getting-started",
		},
		{
			name: "microservices",
			dir:  "../examples/microservices",
			args: []string{
				"-a", "leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app",
				"-a", "leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web",
			},
		},
		{
			name:             "compose",
			dir:              "../examples/compose",
			args:             []string{"--compose-file", "docker-compose.yaml"},
			skipSkaffoldYaml: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, _, deleteNs := SetupNamespace(t)
			defer deleteNs()

			if !test.skipSkaffoldYaml {
				oldYamlPath := filepath.Join(test.dir, "skaffold.yaml")
				oldYaml, err := removeOldSkaffoldYaml(oldYamlPath)
				if err != nil {
					t.Fatalf("removing original skaffold.yaml: %s", err)
				}
				defer restoreOldSkaffoldYaml(oldYaml, oldYamlPath)
			}

			generatedYaml := "skaffold.yaml.out"

			initArgs := append([]string{"--force"}, test.args...)
			skaffold.Init(initArgs...).InDir(test.dir).WithConfig(generatedYaml).RunOrFail(t)

			skaffold.Run().InDir(test.dir).WithConfig(generatedYaml).InNs(ns.Name).RunOrFail(t)

			err := os.Remove(filepath.Join(test.dir, generatedYaml))
			if err != nil {
				t.Errorf("error removing generated skaffold yaml: %v", err)
			}
		})
	}
}

func removeOldSkaffoldYaml(path string) ([]byte, error) {
	skaffoldYaml, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = os.Remove(path); err != nil {
		return nil, err
	}
	return skaffoldYaml, nil
}

func restoreOldSkaffoldYaml(contents []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(contents); err != nil {
		return err
	}
	return nil
}
