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

package v2beta3

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade(t *testing.T) {
	tests := []struct {
		description string
		yaml        string
		expected    string
	}{
		{
			description: "no helm deploy",
			yaml: `apiVersion: skaffold/v2beta3
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
  kustomize:
    paths:
    - kustomization-main`,
			expected: `apiVersion: skaffold/v2beta4
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
  kustomize:
    paths:
    - kustomization-main`,
		},
		{
			description: "helm deploy with releases but no values set",
			yaml: `apiVersion: skaffold/v2beta3
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: skaffold
      chartPath: dummy`,
			expected: `apiVersion: skaffold/v2beta4
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: skaffold
      chartPath: dummy`,
		},
		{
			description: "helm deploy with multiple releases values set",
			yaml: `apiVersion: skaffold/v2beta3
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: foo
      values:
        image1: foo
        image2: bar
    - name: bat
      values:
        image1: bat`,
			expected: `apiVersion: skaffold/v2beta4
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: foo
      artifactOverrides:
        image1: foo
        image2: bar
    - name: bat
      artifactOverrides:
        image1: bat`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			verifyUpgrade(t, test.yaml, test.expected)
		})
	}
}

func verifyUpgrade(t *testutil.T, input, output string) {
	config := NewSkaffoldConfig()

	err := yaml.UnmarshalStrict([]byte(input), config)
	t.CheckNoError(err)
	t.CheckDeepEqual(Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	t.CheckNoError(err)

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	t.CheckNoError(err)
	t.CheckDeepEqual(expected, upgraded)
}
