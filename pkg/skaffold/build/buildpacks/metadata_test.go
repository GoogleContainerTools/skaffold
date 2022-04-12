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

package buildpacks

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSyncRules(t *testing.T) {
	tests := []struct {
		description   string
		labels        map[string]string
		expectedRules []*latest.SyncRule
		shouldErr     bool
	}{
		{
			description: "missing labels",
			labels:      map[string]string{},
		},
		{
			description: "invalid labels",
			labels: map[string]string{
				"io.buildpacks.build.metadata": "invalid",
			},
			shouldErr: true,
		},
		{
			description: "valid labels",
			labels: map[string]string{
				"io.buildpacks.build.metadata": `{
					"bom":[{
						"metadata":{
							"devmode.sync": [
								{"src":"src-value1","dest":"dest-value1"},
								{"src":"src-value2","dest":"dest-value2"}
							]
						}
					}]
				}`,
			},
			expectedRules: []*latest.SyncRule{
				{Src: "src-value1", Dest: "dest-value1"},
				{Src: "src-value2", Dest: "dest-value2"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			rules, err := SyncRules(test.labels)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedRules, rules)
		})
	}
}
