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

package debug

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFindArtifact(t *testing.T) {
	buildArtifacts := []graph.Artifact{
		{ImageName: "image1", Tag: "tag1"},
	}
	tests := []struct {
		description string
		source      string
		returnNil   bool
	}{
		{
			description: "found",
			source:      "image1",
			returnNil:   false,
		},
		{
			description: "not found",
			source:      "image2",
			returnNil:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := findArtifact(test.source, buildArtifacts)

			t.CheckDeepEqual(test.returnNil, result == nil)
		})
	}
}

func TestEnvAsMap(t *testing.T) {
	tests := []struct {
		description string
		source      []string
		result      map[string]string
	}{
		{"nil", nil, map[string]string{}},
		{"empty", []string{}, map[string]string{}},
		{"single", []string{"a=b"}, map[string]string{"a": "b"}},
		{"multiple", []string{"a=b", "c=d"}, map[string]string{"c": "d", "a": "b"}},
		{"embedded equals", []string{"a=b=c", "c=d"}, map[string]string{"c": "d", "a": "b=c"}},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := envAsMap(test.source)

			t.CheckDeepEqual(test.result, result)
		})
	}
}

type testAdapter struct {
	executable *types.ExecutableContainer
}

func (t *testAdapter) GetContainer() *types.ExecutableContainer {
	return t.executable
}

func (t *testAdapter) Apply() {}
