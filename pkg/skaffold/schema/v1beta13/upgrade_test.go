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

package v1beta13

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta14"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade(t *testing.T) {
	yaml := `apiVersion: skaffold/v1beta13
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  - image: gcr.io/k8s-skaffold/jib-maven
    jibMaven:
      args: ['-v']
      profile: prof
      module: dir
  - image: gcr.io/k8s-skaffold/jib-gradle
    jibGradle:
      args: ['-v']
  googleCloudBuild:
    projectId: test-project
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          buildContext:
            gcsBucket: skaffold-kaniko
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
`
	expected := `apiVersion: skaffold/v1beta14
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  - image: gcr.io/k8s-skaffold/jib-maven
    jib:
      args: ['-v', '--activate-profiles', 'prof']
      project: dir
      type: maven
  - image: gcr.io/k8s-skaffold/jib-gradle
    jib:
      args: ['-v']
      type: gradle
  googleCloudBuild:
    projectId: test-project
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          buildContext:
            gcsBucket: skaffold-kaniko
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
`
	verifyUpgrade(t, yaml, expected)
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
