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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		args      []string
		config    string
		shouldErr bool
	}{
		//TODO: mocked kompose test
		{
			name:   "getting-started",
			dir:    "testdata/init/hello",
			config: "skaffold.yaml.out",
		},
		{
			name:   "ignore existing tags",
			dir:    "testdata/init/ignore-tags",
			config: "skaffold.yaml.out",
		},
		{
			name:   "microservices (backwards compatibility)",
			dir:    "testdata/init/microservices",
			config: "skaffold.yaml.out",
			args: []string{
				"-a", `leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app`,
				"-a", `leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web`,
			},
		},
		{
			name:   "microservices",
			dir:    "testdata/init/microservices",
			config: "skaffold.yaml.out",
			args: []string{
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
			},
		},
		{
			name: "error writing config file",
			dir:  "testdata/init/microservices",
			// erroneous config file as . is a directory
			config: ".",
			args: []string{
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
				"-a", `{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			initArgs := append([]string{"init", "--force", "-f", test.config}, test.args...)
			os.Args = initArgs
			t.Chdir(test.dir)
			init := NewCmdInit()
			err := init.Execute()
			t.CheckError(test.shouldErr, err)
			checkGeneratedConfig(t, ".")
		})
	}
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		args        []string
		expectedOut string
		shouldErr   bool
	}{
		{
			name: "analyze microservices",
			dir:  "testdata/init/microservices",
			args: []string{"--analyze"},
			expectedOut: strip(`{
							"dockerfiles":["leeroy-app/Dockerfile","leeroy-web/Dockerfile"],
							"images":["gcr.io/k8s-skaffold/leeroy-app","gcr.io/k8s-skaffold/leeroy-web"]
							}`),
		},
		{
			name: "analyze microservices new format",
			dir:  "testdata/init/microservices",
			args: []string{"--analyze", "--XXenableJibInit"},
			expectedOut: strip(`{
									"builders":[
										{"name":"Docker","payload":{"path":"leeroy-app/Dockerfile"}},
										{"name":"Docker","payload":{"path":"leeroy-web/Dockerfile"}}
									],
									"images":[
										{"name":"gcr.io/k8s-skaffold/leeroy-app","foundMatch":false},
										{"name":"gcr.io/k8s-skaffold/leeroy-web","foundMatch":false}]}`),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			var out bytes.Buffer
			initArgs := append([]string{"init", "--force"}, test.args...)
			os.Args = initArgs
			t.Chdir(test.dir)
			init := NewCmdInit()
			init.SetOut(&out)
			err := init.Execute()
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedOut, out.String())
		})
	}
}

func strip(s string) string {
	cutString := "\n\t\r"
	stripped := ""
	for _, r := range s {
		if strings.ContainsRune(cutString, r) {
			continue
		}
		stripped = fmt.Sprintf("%s%c", stripped, r)
	}
	return stripped
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml"), false)
	t.CheckNoError(err)

	output, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml.out"), false)
	t.CheckNoError(err)
	t.CheckDeepEqual(expectedOutput, output)
}
