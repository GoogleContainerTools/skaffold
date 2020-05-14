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

	yaml "gopkg.in/yaml.v2"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgradeSetKubectlDeploy(t *testing.T) {
	tests := []struct {
		description string
		yaml        string
		expected    string
	}{
		{
			description: "no deployer defined set default",
			yaml: `apiVersion: skaffold/v2beta3
kind: Config
deploy:
  kustomize:	
    paths:	
    - kustomization-test`,
			expected: `apiVersion: skaffold/v2beta4
kind: Config
deploy:
  kustomize:
    paths:
    - kustomization-test`,
		},
		{
			description: "deployer defined",
			yaml: `apiVersion: skaffold/v2beta3
kind: Config`,
			expected: `apiVersion: skaffold/v2beta4
kind: Config
deploy:
   kubectl: {}`,
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

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	t.CheckErrorAndDeepEqual(false, err, expected, upgraded)
}
