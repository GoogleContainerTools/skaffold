/*
<<<<<<< HEAD
Copyright 2020 The Skaffold Authors
=======
Copyright 2019 The Skaffold Authors
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941

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

	yaml "gopkg.in/yaml.v2"

<<<<<<< HEAD
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta4"
=======
	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade(t *testing.T) {
<<<<<<< HEAD
	tests := []struct {
		description string
		yaml        string
		expected    string
	}{
		{
			description: "no helm deploy",
			yaml: `apiVersion: skaffold/v2beta3
=======
	yaml := `apiVersion: skaffold/v2beta3
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
<<<<<<< HEAD
=======
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  - image: gcr.io/k8s-skaffold/jib-maven
    jib:
      args: ['-v', '--activate-profiles', 'prof']
      project: dir
  - image: gcr.io/k8s-skaffold/jib-gradle
    jib:
      args: ['-v']
  googleCloudBuild:
    projectId: test-project
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
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
<<<<<<< HEAD
    - kustomization-main`,
			expected: `apiVersion: skaffold/v2beta4
=======
    - kustomization-main
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          cache: {}
      cluster:
        pullSecretName: e2esecret
        namespace: default
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
        - kustomization-test
  - name: test local
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        docker:
          dockerfile: path/to/Dockerfile
      local:
        push: false
    deploy:
      kubectl:
        manifests:
        - k8s-*
      kustomize: {}
`
	expected := `apiVersion: skaffold/v2beta4
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
<<<<<<< HEAD
=======
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  - image: gcr.io/k8s-skaffold/jib-maven
    jib:
      args: ['-v', '--activate-profiles', 'prof']
      project: dir
  - image: gcr.io/k8s-skaffold/jib-gradle
    jib:
      args: ['-v']
  googleCloudBuild:
    projectId: test-project
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
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
<<<<<<< HEAD
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
	t.CheckErrorAndDeepEqual(false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	t.CheckError(false, err)
=======
    - kustomization-main
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          cache: {}
      cluster:
        pullSecretName: e2esecret
        namespace: default
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
        - kustomization-test
  - name: test local
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        docker:
          dockerfile: path/to/Dockerfile
      local:
        push: false
    deploy:
      kubectl:
        manifests:
        - k8s-*
      kustomize: {}
`
	verifyUpgrade(t, yaml, expected)
}

func verifyUpgrade(t *testing.T, input, output string) {
	config := NewSkaffoldConfig()
	err := yaml.UnmarshalStrict([]byte(input), config)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	testutil.CheckError(t, false, err)
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

<<<<<<< HEAD
	t.CheckErrorAndDeepEqual(false, err, expected, upgraded)
=======
	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
>>>>>>> d43417a8588f9c52cf717199deb05ae72757d941
}
