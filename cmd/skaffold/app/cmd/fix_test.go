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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	v1 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestFix(t *testing.T) {
	tests := []struct {
		description   string
		inputYaml     string
		targetVersion string
		output        string
		shouldErr     bool
		cmpOptions    cmp.Options
	}{
		{
			description:   "v1alpha4 to latest",
			targetVersion: latest.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version),
		},
		{
			description:   "v1alpha1 to latest",
			targetVersion: latest.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version),
		},
		{
			description:   "v1alpha1 to v1",
			targetVersion: v1.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
`, v1.Version),
		},
		{
			description:   "already target version",
			targetVersion: latest.Version,
			inputYaml: fmt.Sprintf(`apiVersion: %s
kind: Config
`, latest.Version),
			output: "config is already version " + latest.Version + "\n",
		},
		{
			description: "invalid input",
			inputYaml:   "invalid",
			shouldErr:   true,
		},
		{
			description:   "validation fails",
			targetVersion: latest.Version,
			inputYaml: `apiVersion: skaffold/v3alpha1
kind: Config
build:
  artifacts:
  - imageName:
    dockerfilePath: dockerfile.test
`,
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfgFile := t.TempFile("config", []byte(test.inputYaml))

			var b bytes.Buffer
			err := fix(&b, cfgFile, "", test.targetVersion)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.output, b.String(), test.cmpOptions)
		})
	}
}

func TestFixToFileOverwrite(t *testing.T) {
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version)

	testutil.Run(t, "", func(t *testutil.T) {
		cfgFile := t.TempFile("config", []byte(inputYaml))

		var b bytes.Buffer
		err := fix(&b, cfgFile, cfgFile, latest.Version)

		output, _ := os.ReadFile(cfgFile)

		t.CheckNoError(err)
		t.CheckDeepEqual(expectedOutput, string(output), testutil.YamlObj(t.T))
	})
}
