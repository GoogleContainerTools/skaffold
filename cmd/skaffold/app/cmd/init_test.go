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

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		args []string
	}{
		//TODO: mocked kompose test
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
			config := "skaffold.yaml.out"
			initArgs := append([]string{"init", "--force", "-f", config}, test.args...)
			os.Args = initArgs
			wd, _ := os.Getwd()
			os.Chdir(test.dir)
			defer os.Chdir(wd)
			init := NewCmdInit()
			if err := init.Execute(); err != nil {
				t.Fail()
			}
			checkGeneratedConfig(t, ".")
		})
	}
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml"), false)
	t.CheckNoError(err)

	output, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml.out"), false)
	t.CheckNoError(err)
	t.CheckDeepEqual(expectedOutput, output)
}
