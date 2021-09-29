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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
			t.Override(&getRunContext, func(context.Context, config.SkaffoldOptions, []util.VersionedConfig) (*v2.RunContext, error) {
				return nil, fmt.Errorf("cannot get the runtime context")
			})
			t.Override(&yamlOnly, test.yamlOnly)
			t.Override(&getCfgs, func(context.Context, config.SkaffoldOptions) ([]util.VersionedConfig, error) {
				return []util.VersionedConfig{
					&latestV2.SkaffoldConfig{
						APIVersion: "testVersion",
						Kind:       "Config",
						Metadata: latestV2.Metadata{
							Name: "config1",
						},
					},
					&latestV2.SkaffoldConfig{
						APIVersion: "testVersion",
						Kind:       "Config",
						Metadata: latestV2.Metadata{
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
