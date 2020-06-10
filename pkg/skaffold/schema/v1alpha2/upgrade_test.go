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

package v1alpha2

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade_helmReleaseValuesFile(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha2
kind: Config
deploy:
  helm:
    releases:
    - name: test release
      valuesFilePath: values.yaml
`
	expected := `apiVersion: skaffold/v1alpha3
kind: Config
deploy:
  helm:
    releases:
    - name: test release
      valuesFiles:
      - values.yaml
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_helmReleaseValuesFileWithProfile(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha2
kind: Config
profiles:
- name: test
  deploy:
    helm:
      releases:
      - name: test
        valuesFilePath: values.yaml
`
	expected := `apiVersion: skaffold/v1alpha3
kind: Config
profiles:
- name: test
  deploy:
    helm:
      releases:
      - name: test
        valuesFiles:
        - values.yaml
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgrade_kanikoWithProfile(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
  kaniko:
    gcsBucket: k8s-skaffold
    pullSecret: /a/secret/path/kaniko.json
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - imageName: gcr.io/k8s-skaffold/skaffold-example
    deploy:
      kubectl:
        manifests:
        - k8s-*
`
	expected := `apiVersion: skaffold/v1alpha3
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
  kaniko:
    buildContext:
      gcsBucket: k8s-skaffold
    pullSecret: /a/secret/path/kaniko.json
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
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

func TestUpgrade_helmReleaseOverrides(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha2
kind: Config
deploy:
  helm:
    releases:
    - name: test release
      overrides:
        global:
          localstack:
            enabled: true
`
	expected := `apiVersion: skaffold/v1alpha3
kind: Config
deploy:
  helm:
    releases:
    - name: test release
      overrides:
        global:
          localstack:
            enabled: true
`
	verifyUpgrade(t, yaml, expected)
}

func verifyUpgrade(t *testing.T, input, output string) {
	config := NewSkaffoldConfig()
	err := yaml.UnmarshalStrict([]byte(input), config)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	testutil.CheckError(t, false, err)

	expected := v1alpha3.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
