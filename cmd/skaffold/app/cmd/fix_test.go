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
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFix(t *testing.T) {
	tests := []struct {
		description string
		inputYaml   string
		output      string
		shouldErr   bool
	}{
		{
			description: "v1alpha4 to latest",
			inputYaml: `apiVersion: skaffold/v1alpha4
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`, latest.Version),
		},
		{
			description: "v1alpha1 to latest",
			inputYaml: `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: docker/image
    dockerfilePath: dockerfile.test
deploy:
  kubectl:
    manifests:
    - paths:
      - k8s/deployment.yaml
`,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`, latest.Version),
		},
		{
			description: "already latest version",
			inputYaml: fmt.Sprintf(`apiVersion: %s
kind: Config
`, latest.Version),
			output: "config is already latest version\n",
		},
		{
			description: "invalid input",
			inputYaml:   "invalid",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfgFile := t.TempFile("config", []byte(test.inputYaml))

			var b bytes.Buffer
			err := fix(&b, cfgFile, false)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.output, b.String())
		})
	}
}

func TestFixOverwrite(t *testing.T) {
	inputYaml := `apiVersion: skaffold/v1alpha4
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`
	expectedOutput := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`, latest.Version)

	testutil.Run(t, "", func(t *testutil.T) {
		cfgFile := t.TempFile("config", []byte(inputYaml))

		var b bytes.Buffer
		err := fix(&b, cfgFile, true)

		output, _ := ioutil.ReadFile(cfgFile)

		t.CheckNoError(err)
		t.CheckDeepEqual(expectedOutput, string(output))
	})
}
