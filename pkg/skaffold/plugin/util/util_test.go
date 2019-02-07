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

package util

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var firstExecutionEnvironment = &latest.ExecutionEnvironment{Name: "first"}
var secondExecutionEnvironment = &latest.ExecutionEnvironment{Name: "second"}

var artifactOne = &latest.Artifact{ImageName: "first"}
var artifactTwo = &latest.Artifact{ImageName: "second", ExecutionEnvironment: secondExecutionEnvironment}

func TestGroupArtifactsByEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		artifacts []*latest.Artifact
		env       *latest.ExecutionEnvironment
		expected  map[*latest.ExecutionEnvironment][]*latest.Artifact
	}{
		{
			name: "group by environment",
			artifacts: []*latest.Artifact{
				artifactOne,
				artifactTwo,
			},
			env: firstExecutionEnvironment,
			expected: map[*latest.ExecutionEnvironment][]*latest.Artifact{
				firstExecutionEnvironment:  {artifactOne},
				secondExecutionEnvironment: {artifactTwo},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GroupArtifactsByEnvironment(test.artifacts, test.env)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expected, actual)
		})
	}
}
