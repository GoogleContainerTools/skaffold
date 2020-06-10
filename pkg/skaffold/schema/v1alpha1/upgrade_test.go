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

package v1alpha1

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade_gitTagger(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  tagPolicy: gitCommit
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  tagPolicy:
    gitCommit: {}
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_sha256Tagger(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  tagPolicy: sha256
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  tagPolicy:
    sha256: {}
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_deploy(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
deploy:
  kubectl:
    manifests:
    - paths:
      - k8s-*
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
deploy:
  kubectl:
    manifests:
    - k8s-*
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_helm(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
deploy:
  helm:
    releases:
    - name: release
      chartPath: path
      valuesFilePath: valuesFile
      values: {key:value}
      namespace: ns
      version: 1.0
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
deploy:
  helm:
    releases:
    - name: release
      chartPath: path
      valuesFilePath: valuesFile
      values: {key:value}
      namespace: ns
      version: 1.0
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_dockerfile(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
    dockerfilePath: Dockerfile
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfilePath: Dockerfile
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_buildargs(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
    buildArgs: {key:value}
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
    docker:
      buildArgs: {key:value}
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_gcb(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  googleCloudBuild:
    projectId: PROJECT
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  googleCloudBuild:
    projectId: PROJECT
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_local(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha1
kind: Config
build:
  local:
    skipPush: true
`
	expected := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  local:
    skipPush: true
`
	verifyUpgrade(t, yaml, expected)
}

func verifyUpgrade(t *testing.T, input, output string) {
	config := NewSkaffoldConfig()
	err := yaml.UnmarshalStrict([]byte(input), config)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	testutil.CheckError(t, false, err)

	expected := v1alpha2.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
