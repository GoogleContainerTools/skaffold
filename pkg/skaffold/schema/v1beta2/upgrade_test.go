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

package v1beta2

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v1beta3"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestUpgrade(t *testing.T) {
	yaml := `apiVersion: skaffold/v1beta2
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
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
      kaniko:
        buildContext:
          gcsBucket: skaffold-kaniko
        pullSecretName: e2esecret
        namespace: default
        cache: {}
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
    test:
     - image: gcr.io/k8s-skaffold/skaffold-example
       structureTests:
         - ./test/*
    deploy:
      kubectl:
        manifests:
        - k8s-*
`
	expected := `apiVersion: skaffold/v1beta3
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
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
      kaniko:
        buildContext:
          gcsBucket: skaffold-kaniko
        pullSecretName: e2esecret
        namespace: default
        cache: {}
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
    test:
     - image: gcr.io/k8s-skaffold/skaffold-example
       structureTests:
         - ./test/*
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

	expected := v1beta3.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
