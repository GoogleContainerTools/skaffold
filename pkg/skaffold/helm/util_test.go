/*
Copyright 2022 The Skaffold Authors

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

package helm

import (
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestWriteBuildArtifacts(t *testing.T) {
	tests := []struct {
		description string
		builds      []graph.Artifact
		result      string
	}{
		{
			description: "nil",
			builds:      nil,
			result:      `{"builds":null}`,
		},
		{
			description: "empty",
			builds:      []graph.Artifact{},
			result:      `{"builds":[]}`,
		},
		{
			description: "multiple images with tags",
			builds:      []graph.Artifact{{ImageName: "name", Tag: "name:tag"}, {ImageName: "name2", Tag: "name2:tag"}},
			result:      `{"builds":[{"imageName":"name","tag":"name:tag"},{"imageName":"name2","tag":"name2:tag"}]}`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file, cleanup, err := WriteBuildArtifacts(test.builds)
			t.CheckError(false, err)
			if content, err := ioutil.ReadFile(file); err != nil {
				t.Errorf("error reading file %q: %v", file, err)
			} else {
				t.CheckDeepEqual(test.result, string(content))
			}
			cleanup()
		})
	}
}
