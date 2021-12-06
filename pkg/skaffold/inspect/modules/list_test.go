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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPrintModulesList(t *testing.T) {
	tests := []struct {
		description string
		configSet   parser.SkaffoldConfigSet
		includeAll  bool
		err         error
		expected    string
	}{
		{
			description: "print modules",
			configSet: parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: ""}}, SourceFile: "path/to/cfg1", SourceIndex: 0, IsRootConfig: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: "cfg1"}}, SourceFile: "path/to/cfg1", SourceIndex: 1, IsRootConfig: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: "cfg2"}}, SourceFile: "path/to/cfg2", SourceIndex: 0, IsRemote: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: ""}}, SourceFile: "path/to/cfg3", SourceIndex: 0},
			},
			expected: `{"modules":[{"name":"cfg1","path":"path/to/cfg1","isRoot":true},{"name":"cfg2","path":"path/to/cfg2","isRemote":true}]}` + "\n",
		},
		{
			description: "print modules; include all",
			configSet: parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: ""}}, SourceFile: "path/to/cfg1", SourceIndex: 0, IsRootConfig: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: "cfg1"}}, SourceFile: "path/to/cfg1", SourceIndex: 1, IsRootConfig: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: "cfg2"}}, SourceFile: "path/to/cfg2", SourceIndex: 0, IsRemote: true},
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &v2.SkaffoldConfig{Metadata: v2.Metadata{Name: ""}}, SourceFile: "path/to/cfg3", SourceIndex: 0},
			},
			includeAll: true,
			expected: `{"modules":[` +
				`{"name":"__config_0","path":"path/to/cfg1","isRoot":true},` +
				`{"name":"cfg1","path":"path/to/cfg1","isRoot":true},` +
				`{"name":"cfg2","path":"path/to/cfg2","isRemote":true},` +
				`{"name":"__config_0","path":"path/to/cfg3"}]}` + "\n",
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
			err := PrintModulesList(context.Background(), &buf, inspect.Options{OutFormat: "json", ModulesOptions: inspect.ModulesOptions{IncludeAll: test.includeAll}})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
