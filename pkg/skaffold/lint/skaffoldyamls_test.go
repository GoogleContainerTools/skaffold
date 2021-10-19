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

package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var testSkaffoldYaml = `apiVersion: skaffold/v2beta21
kind: Config
build:
  artifacts:
    - image: test-app
`

var invalidSkaffoldYaml = `invalid{\{\Yaml}`

func TestGetSkaffoldYamlsLintResults(t *testing.T) {
	tests := []struct {
		description            string
		moduleAndSkaffoldYamls map[string]string
		profiles               []string
		module                 []string
		shouldErr              bool
		err                    error
		expected               map[string]*[]Result
	}{
		{
			description:            "get all skaffold yaml lint rules for 2 skaffold yaml files",
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml, "cfg1": testSkaffoldYaml},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule: &Rule{
							Filter: YamlFieldFilter{
								Filter: yaml.FieldMatcher{Name: "apiVersion", StringRegexValue: fmt.Sprintf("[^%s]", version.Get().ConfigVersion)},
							},
							RuleID:   SkaffoldYamlAPIVersionOutOfDate,
							RuleType: YamlFieldLintRule,
							Explanation: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest version: `apiVersion: %s` "+
								"or run the 'skaffold fix' command to have skaffold upgrade this for you.", version.Get().ConfigVersion),
						},
						Line:   1,
						Column: 12,
					},
				},
				"cfg1": {
					{
						Rule: &Rule{
							Filter: YamlFieldFilter{
								Filter: yaml.FieldMatcher{Name: "apiVersion", StringRegexValue: fmt.Sprintf("[^%s]", version.Get().ConfigVersion)},
							},
							RuleID:   SkaffoldYamlAPIVersionOutOfDate,
							RuleType: YamlFieldLintRule,
							Explanation: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest version: `apiVersion: %s` "+
								"or run the 'skaffold fix' command to have skaffold upgrade this for you.", version.Get().ConfigVersion),
						},
						Line:   1,
						Column: 12,
					},
				},
			},
		},
		{
			description:            "get all skaffold yaml lint rules for one module",
			moduleAndSkaffoldYamls: map[string]string{"cfg0": testSkaffoldYaml, "cfg1": testSkaffoldYaml},
			module:                 []string{"cfg0"},
			expected: map[string]*[]Result{
				"cfg0": {
					{
						Rule: &Rule{
							Filter: YamlFieldFilter{
								Filter: yaml.FieldMatcher{Name: "apiVersion", StringRegexValue: fmt.Sprintf("[^%s]", version.Get().ConfigVersion)},
							},
							RuleID:   SkaffoldYamlAPIVersionOutOfDate,
							RuleType: YamlFieldLintRule,
							Explanation: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest version: `apiVersion: %s` "+
								"or run the 'skaffold fix' command to have skaffold upgrade this for you.", version.Get().ConfigVersion),
						},
						Line:   1,
						Column: 12,
					},
				},
			},
		},
		{
			description:            "invalid skaffold yaml file",
			moduleAndSkaffoldYamls: map[string]string{"cfg0": invalidSkaffoldYaml},
			shouldErr:              true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpdir := t.TempDir()
			configSet := parser.SkaffoldConfigSet{}
			for module, skaffoldyamlText := range test.moduleAndSkaffoldYamls {
				fp := filepath.Join(tmpdir, fmt.Sprintf("%s.yaml", module))
				err := ioutil.WriteFile(fp, []byte(skaffoldyamlText), 0644)
				if err != nil {
					t.Fatalf("error creating skaffold.yaml file with name %s: %v", fp, err)
				}
				configSet = append(configSet, &parser.SkaffoldConfigEntry{SkaffoldConfig: &v1.SkaffoldConfig{
					Metadata: v1.Metadata{Name: module},
				},
					SourceFile: fp,
				})
				// test overwrites file paths for expected SkaffoldYamlRules as they are made dynamically
				results := test.expected[module]
				if results == nil {
					continue
				}
				for i := range *results {
					(*results)[i].AbsFilePath = configSet[len(configSet)-1].SourceFile
					(*results)[i].RelFilePath = configSet[len(configSet)-1].SourceFile
				}
			}
			t.Override(&realWorkDir, func() (string, error) {
				return "", nil
			})
			t.Override(&getConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !util.StrSliceContains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Test = profile.Test
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			results, err := GetSkaffoldYamlsLintResults(context.Background(), Options{
				OutFormat: "json", Modules: test.module, Profiles: test.profiles})
			t.CheckError(test.shouldErr, err)
			expectedResults := &[]Result{}

			// this is done to enforce result order
			for i := 0; i < len(test.expected); i++ {
				*expectedResults = append(*expectedResults, *test.expected[fmt.Sprintf("cfg%d", i)]...)
			}
			if !test.shouldErr {
				t.CheckDeepEqual(expectedResults, results)
			}
		})
	}
}
