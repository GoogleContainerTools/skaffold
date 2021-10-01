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

package inspect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintTestsList(t *testing.T) {
	tests := []struct {
		description string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all tests",
			expected:    `{"tests":[{"testType":"structure-test","structureTest":"structure-test-i0","structureTestArgs":null},{"testType":"custom-test","Command":"custom-test-i0","TimeoutSeconds":0,"Dependencies":null}]}` + "\n",
		},
		{
			description: "print all tests for one module",
			expected:    `{"tests":[{"testType":"structure-test","structureTest":"structure-test-i0","structureTestArgs":null}]}` + "\n",
			module:      []string{"cfg1"},
		},
		{
			description: "print all tests for two activated profiles",

			expected: `{"tests":[{"testType":"structure-test","structureTest":"structure-test-i0","structureTestArgs":null},{"testType":"custom-test","Command":"custom-test-i0","TimeoutSeconds":0,"Dependencies":null}]}` + "\n",
			profiles: []string{"custom-test", "structure-test"},
		},
		{
			description: "print all tests for one module and an activated profile",

			expected: `{"tests":[{"testType":"structure-test","structureTest":"structure-test-i0","structureTestArgs":null}]}` + "\n",
			module:   []string{"cfg1"},
			profiles: []string{"custom-test"},
		},
		{
			description: "actionable error",
			err:         sErrors.MainConfigFileNotFoundErr("path/to/skaffold.yaml", fmt.Errorf("failed to read file : %q", "skaffold.yaml")),
			expected:    `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/skaffold.yaml\": failed to read file : \"skaffold.yaml\". Check that the specified configuration file exists at \"path/to/skaffold.yaml\"."}` + "\n",
		},
		{
			description: "generic error",
			err:         errors.New("some error occurred"),
			expected:    `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{
					Metadata: v2.Metadata{Name: "cfg1"},
					Pipeline: v2.Pipeline{Test: []*v2.TestCase{{StructureTests: []string{"structure-test-i0"}}}},
					Profiles: []v2.Profile{
						{Name: "custom-test", Pipeline: v2.Pipeline{Test: []*v2.TestCase{{CustomTests: []v2.CustomTest{{Command: "custom-test-i0"}}}}}}},
				}, SourceFile: "path/to/cfg1"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{
					Metadata: v2.Metadata{Name: "cfg2"},
					Pipeline: v2.Pipeline{Test: []*v2.TestCase{{CustomTests: []v2.CustomTest{{Command: "custom-test-i0"}}}}},
					Profiles: []v2.Profile{
						{Name: "structure-test", Pipeline: v2.Pipeline{Test: []*v2.TestCase{{StructureTests: []string{"structure-test-i0"}}}}}},
				}, SourceFile: "path/to/cfg1"},
			}
			t.Override(&inspect.GetConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
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
			var buf bytes.Buffer
			err := PrintTestsList(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.module})
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
