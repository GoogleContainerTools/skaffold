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

package v1alpha5

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta1"
	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

func TestUpgrade_removeACR(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha5
kind: Config
build:
  artifacts:
  - image: myregistry.azurecr.io/skaffold-example
  acr: {}
deploy:
  kubectl:
    manifests:
      - k8s-*
`
	upgradeShouldFailt(t, yaml)
}

func TestUpgrade_removeACRInProfiles(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha5
kind: Config
build:
  artifacts:
  - image: myregistry.azurecr.io/skaffold-example
deploy:
  kubectl:
    manifests:
      - k8s-*
profiles:
 - name: test profile
   build: 
    acr: {}
`
	upgradeShouldFailt(t, yaml)
}

func TestUpgrade(t *testing.T) {
	yaml := `apiVersion: skaffold/v1alpha5
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
	expected := `apiVersion: skaffold/v1beta1
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
	verityUpgrade(t, yaml, expected)
}

func upgradeShouldFailt(t *testing.T, input string) {
	pipeline := NewSkaffoldPipeline()
	err := yaml.UnmarshalStrict([]byte(input), pipeline)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, pipeline.GetVersion())

	_, err = pipeline.Upgrade()
	testutil.CheckError(t, true, err)
}

func verityUpgrade(t *testing.T, input, output string) {
	pipeline := NewSkaffoldPipeline()
	err := yaml.UnmarshalStrict([]byte(input), pipeline)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, pipeline.GetVersion())

	upgraded, err := pipeline.Upgrade()
	testutil.CheckError(t, false, err)

	expected := v1beta1.NewSkaffoldPipeline()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
