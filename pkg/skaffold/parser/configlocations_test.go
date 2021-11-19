/*
Copyright 2021 The Skaffold Authors

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

package parser

import (
	"context"
	"reflect"
	"testing"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testSkaffoldYaml = `apiVersion: skaffold/v2beta26
kind: Config
build:
  artifacts:
    - image: leeroy-app
      context: leeroy-app
      requires:
        - image: base
          alias: BASE
deploy:
  kubectl:
    manifests:
      - leeroy-app/kubernetes/*
`

// TODO(aaron-prindle) currently an issue where the map value for
// &cfgs[0].Pipeline.Build is not what we would expect (expect: kyaml.Lookup("build") )

func TestUpdateYAMLNodes(t *testing.T) {
	tests := []struct {
		description      string
		skaffoldYamlText string
		shouldErr        bool
		err              error
		expected         []kyaml.Filter
	}{
		{
			description:      "valid map of yaml nodes for input skaffold.yaml file",
			skaffoldYamlText: testSkaffoldYaml,
			expected: []kyaml.Filter{
				kyaml.Lookup("build", "artifacts"),
				kyaml.Lookup("deploy", "manifests"),
			},
		},
		// TODO(aaron-prindle) add test for error case
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fp := t.TempFile("skaffoldyaml-", []byte(test.skaffoldYamlText))
			cfgs, err := GetConfigSet(context.TODO(), config.SkaffoldOptions{ConfigurationFile: fp})
			if err != nil {
				t.Fatalf(err.Error())
			}
			yamlNodes, err := Parse(cfgs[0])
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				root, err := kyaml.Parse(test.skaffoldYamlText)
				if err != nil {
					t.Fatalf(err.Error())
				}
				var seen bool
				for _, e := range test.expected {
					seen = false
					expectedNode, err := root.Pipe(e)
					if err != nil {
						t.Fatalf(err.Error())
					}
					for _, v := range yamlNodes.yamlNodes {
						if reflect.DeepEqual(expectedNode, v.RNode) {
							seen = true
						}
					}
					if seen != true {
						t.Errorf("unable to find expected yaml node: %v in the generated yaml node map: %v", expectedNode, yamlNodes.yamlNodes)
					}
				}
			}
		})
	}
}
