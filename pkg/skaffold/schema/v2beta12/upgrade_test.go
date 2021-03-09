/*
Copyright 2020 The Skaffold Authors

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

package v2beta12

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
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
			yaml: `apiVersion: skaffold/v2beta12
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
			expected: `apiVersion: skaffold/v2beta13
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
			description: "helm deploy with releases but no chart path set",
			yaml: `apiVersion: skaffold/v2beta12
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: skaffold`,
			expected: `apiVersion: skaffold/v2beta13
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: skaffold`,
		},
		{
			description: "helm deploy with multiple releases and mixed chart paths",
			yaml: `apiVersion: skaffold/v2beta12
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: foo1
      chartPath: foo1/bar
    - name: foo2
      chartPath: foo2/bar
      remote: true
    - name: foo3
      chartPath: foo3/bar
      remote: false`,
			expected: `apiVersion: skaffold/v2beta13
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
deploy:
  helm:
    releases:
    - name: foo1
      chartPath: foo1/bar
    - name: foo2
      remoteChart: foo2/bar
    - name: foo3
      chartPath: foo3/bar`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			verifyUpgrade(t.T, test.yaml, test.expected)
		})
	}
}

func verifyUpgrade(t *testing.T, input, output string) {
	config := NewSkaffoldConfig()
	err := yaml.UnmarshalStrict([]byte(input), config)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	testutil.CheckError(t, false, err)

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
