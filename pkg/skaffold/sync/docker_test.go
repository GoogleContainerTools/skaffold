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

package sync

import (
	"context"
	"strings"
	"testing"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerSync(t *testing.T) {
	tests := []struct {
		description string
		item        *Item
		expected    []string
	}{
		{
			description: "additions are added via tar",
			item: &Item{
				Image: "image:123",
				Artifact: &latestV2.Artifact{
					ImageName: "image",
				},
				Copy: syncMap{"test.go": {"/test.go"}},
			},
			expected: []string{"docker exec -i image tar xmf - -C / --no-same-owner"},
		},
		{
			description: "one deletion",
			item: &Item{
				Image: "image:123",
				Artifact: &latestV2.Artifact{
					ImageName: "image",
				},
				Delete: syncMap{"test.go": {"/test.go"}},
			},
			expected: []string{"docker exec -i image rm -rf -- /test.go"},
		},
		{
			description: "two deletions",
			item: &Item{
				Image: "image:123",
				Artifact: &latestV2.Artifact{
					ImageName: "image",
				},
				Delete: syncMap{"test.go": {"/test.go"}, "foobar.js": {"/dev/js/foobar.js"}},
			},
			expected: []string{"docker exec -i image rm -rf -- /dev/js/foobar.js /test.go"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cmdRecord := &TestCmdRecorder{}

			t.Override(&util.DefaultExecCommand, cmdRecord)
			NewContainerSyncer().Sync(context.Background(), nil, test.item)

			// sync maps are unordered, but we can split the resulting command strings and compare elements
			t.CheckElementsMatch(strings.Split(test.expected[0], " "), strings.Split(cmdRecord.cmds[0], " "))
		})
	}
}
