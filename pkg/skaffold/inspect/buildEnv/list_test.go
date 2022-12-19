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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestPrintBuildEnvsList(t *testing.T) {
	tests := []struct {
		description string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all build environments",
			expected: `{"build_envs":[` +
				`{"type":"local","path":"path/to/cfg1","module":"cfg1"},` +
				`{"type":"googleCloudBuild","path":"path/to/cfg2","module":"cfg2"}` +
				"]}\n",
		},
		{
			description: "print all build environments for one module",
			expected: `{"build_envs":[` +
				`{"type":"local","path":"path/to/cfg1","module":"cfg1"}` +
				"]}\n",
			module: []string{"cfg1"},
		},
		{
			description: "print all build environments for two activated profiles",

			expected: `{"build_envs":[` +
				`{"type":"cluster","path":"path/to/cfg1","module":"cfg1"},` +
				`{"type":"local","path":"path/to/cfg2","module":"cfg2"}` +
				"]}\n",
			profiles: []string{"local", "cluster"},
		},
		{
			description: "print all build environments for one module and an activated profile",

			expected: `{"build_envs":[` +
				`{"type":"cluster","path":"path/to/cfg1","module":"cfg1"}` +
				"]}\n",
			module:   []string{"cfg1"},
			profiles: []string{"cluster"},
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
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg1"},
					Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}},
					Profiles: []latest.Profile{
						{Name: "cluster", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
					}}, SourceFile: "path/to/cfg1"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg2"},
					Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}}}},
					Profiles: []latest.Profile{
						{Name: "local", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					}}, SourceFile: "path/to/cfg2"},
			}
			t.Override(&inspect.GetConfigSet, func(ctx context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Build.BuildType = profile.Build.BuildType
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintBuildEnvsList(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.module, Profiles: test.profiles})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
