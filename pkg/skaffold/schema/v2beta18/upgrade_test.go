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

package v2beta18

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta19"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade(t *testing.T) {
	yaml := `apiVersion: skaffold/v2beta18
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
      secret:
        id: id
        src: /file.txt
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
  - image: gcr.io/k8s-skaffold/buildpacks
    buildpacks:
      builder: gcr.io/buildpacks/builder:v1
    sync:
      auto: true
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
  kustomize:
    paths:
    - kustomization-main
portForward:
  - resourceType: deployment
    resourceName: leeroy-app
    port: 8080
    localPort: 9001
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          cache: {}
      cluster:
        pullSecretName: e2esecret
        pullSecretPath: secret.json
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
	expected := `apiVersion: skaffold/v2beta19
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
      secret:
        id: id
        src: /file.txt
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
  - image: gcr.io/k8s-skaffold/buildpacks
    buildpacks:
      builder: gcr.io/buildpacks/builder:v1
    sync:
      auto: true
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
  kustomize:
    paths:
    - kustomization-main
portForward:
  - resourceType: deployment
    resourceName: leeroy-app
    port: 8080
    localPort: 9001
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          cache: {}
      cluster:
        pullSecretName: e2esecret
        pullSecretPath: secret.json
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

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
