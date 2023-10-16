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

func TestPrintProfilesList(t *testing.T) {
	tests := []struct {
		description string
		configSet   parser.SkaffoldConfigSet
		buildEnv    inspect.BuildEnv
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all profiles",
			configSet: parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg1"}, Profiles: []latest.Profile{
					{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p2", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
				}}, SourceFile: "path/to/cfg1"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg2"}, Profiles: []latest.Profile{
					{Name: "p3", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p4", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}}}}},
				}}, SourceFile: "path/to/cfg2"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Profiles: []latest.Profile{
					{Name: "p5", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
				}}, SourceFile: "path/to/cfg2"},
			},
			expected: `{"profiles":[` +
				`{"name":"p1","path":"path/to/cfg1","module":"cfg1"},` +
				`{"name":"p2","path":"path/to/cfg1","module":"cfg1"},` +
				`{"name":"p3","path":"path/to/cfg2","module":"cfg2"},` +
				`{"name":"p4","path":"path/to/cfg2","module":"cfg2"},` +
				`{"name":"p5","path":"path/to/cfg2"}` +
				"]}\n",
		},
		{
			description: "print all profiles for one module",
			configSet: parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg1"}, Profiles: []latest.Profile{
					{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p2", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
				}}, SourceFile: "path/to/cfg1"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg2"}, Profiles: []latest.Profile{
					{Name: "p3", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p4", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}}}}},
				}}, SourceFile: "path/to/cfg2"},
			},
			expected: `{"profiles":[` +
				`{"name":"p3","path":"path/to/cfg2","module":"cfg2"},` +
				`{"name":"p4","path":"path/to/cfg2","module":"cfg2"}` +
				"]}\n",
			module: []string{"cfg2"},
		},
		{
			description: "print all profiles for one module and gcb build-env",
			configSet: parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg1"}, Profiles: []latest.Profile{
					{Name: "p1", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p2", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{}}}}},
				}}, SourceFile: "path/to/cfg1"},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{Metadata: latest.Metadata{Name: "cfg2"}, Profiles: []latest.Profile{
					{Name: "p3", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}}},
					{Name: "p4", Pipeline: latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{}}}}},
				}}, SourceFile: "path/to/cfg2"},
			},
			expected: `{"profiles":[` +
				`{"name":"p4","path":"path/to/cfg2","module":"cfg2"}` +
				"]}\n",
			module:   []string{"cfg2"},
			buildEnv: inspect.BuildEnvs.GoogleCloudBuild,
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
			t.Override(&inspect.GetConfigSet, func(ctx context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				if len(opts.ConfigurationFilter) == 0 {
					return test.configSet, test.err
				}
				var set parser.SkaffoldConfigSet
				if stringslice.Contains(opts.ConfigurationFilter, "cfg1") {
					set = append(set, test.configSet[0])
				}
				if stringslice.Contains(opts.ConfigurationFilter, "cfg2") {
					set = append(set, test.configSet[1])
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintProfilesList(context.Background(), &buf, inspect.Options{OutFormat: "json", Modules: test.module, ProfilesOptions: inspect.ProfilesOptions{BuildEnv: test.buildEnv}})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
