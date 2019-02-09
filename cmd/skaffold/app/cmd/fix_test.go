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
		t.Run(test.description, func(t *testing.T) {
			cfgFile, teardown := testutil.TempFile(t, "config", []byte(test.inputYaml))
			defer teardown()

			var b bytes.Buffer
			err := runFix(&b, cfgFile, false, []string{"test"})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.output, b.String())
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

	cfgFile, teardown := testutil.TempFile(t, "config", []byte(inputYaml))
	defer teardown()

	var b bytes.Buffer
	err := runFix(&b, cfgFile, true, []string{"test"})

	output, _ := ioutil.ReadFile(cfgFile)

	testutil.CheckErrorAndDeepEqual(t, false, err, expectedOutput, string(output))
}
