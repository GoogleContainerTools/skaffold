/*
Copyright 2018 The Skaffold Authors

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

package v1alpha3

import (
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade_imageName(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha3
kind: Config
build:
  artifacts:
  - imageName: gcr.io/k8s-skaffold/skaffold-example
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
	expected := `apiVersion: skaffold/v1alpha4
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
    deploy:
      kubectl:
        manifests:
        - k8s-*
`
	verityUpgrade(t, yaml, expected)
}

func TestUpgrade_skipPush(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha3
kind: Config
build:
  local:	
    skipPush: false
profiles:
  - name: testEnv1
    build:
      local:
        skipPush: true
  - name: testEnv2
    build:
      local:
        skipPush: false
`
	expected := `apiVersion: skaffold/v1alpha4
kind: Config
build:
  local:	
    push: true
profiles:
  - name: testEnv1
    build:
      local:
        push: false
  - name: testEnv2
    build:
      local:
        push: true
`
	verityUpgrade(t, yaml, expected)
}

func verityUpgrade(t *testing.T, input, output string) {
	pipeline := NewSkaffoldPipeline()
	err := yaml.UnmarshalStrict([]byte(input), pipeline)
	testutil.CheckError(t, false, err)

	upgraded, err := pipeline.Upgrade()
	testutil.CheckError(t, false, err)

	expected := v1alpha4.NewSkaffoldPipeline()
	err = yaml.UnmarshalStrict([]byte(output), expected)
	testutil.CheckError(t, false, err)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
