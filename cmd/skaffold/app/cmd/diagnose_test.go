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

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDoDiagnose(t *testing.T) {
	tests := []struct {
		description string
		yamlOnly    bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "yaml only set to true",
			yamlOnly:    true,
			expected: `apiVersion: testVersion
kind: Config
metadata:
  name: config1
---
apiVersion: testVersion
kind: Config
metadata:
  name: config2
`,
		},
		{
			description: "yaml only set to false",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getRunContext, func(context.Context, config.SkaffoldOptions, []util.VersionedConfig) (*runcontext.RunContext, error) {
				return nil, fmt.Errorf("cannot get the runtime context")
			})
			t.Override(&yamlOnly, test.yamlOnly)
			t.Override(&getCfgs, func(context.Context, config.SkaffoldOptions) ([]util.VersionedConfig, error) {
				return []util.VersionedConfig{
					&latest.SkaffoldConfig{
						APIVersion: "testVersion",
						Kind:       "Config",
						Metadata: latest.Metadata{
							Name: "config1",
						},
					},
					&latest.SkaffoldConfig{
						APIVersion: "testVersion",
						Kind:       "Config",
						Metadata: latest.Metadata{
							Name: "config2",
						},
					},
				}, nil
			})
			var b bytes.Buffer
			err := doDiagnose(context.Background(), &b)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, b.String())
		})
	}
}
