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

package recommender

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestContainerErrorMake(t *testing.T) {
	tests := []struct {
		description string
		errCode     proto.StatusCode
		expected    *proto.Suggestion
	}{
		{
			description: "makes err suggestion for terminated containers (303)",
			errCode:     proto.StatusCode_STATUSCHECK_CONTAINER_TERMINATED,
			expected: &proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_CHECK_CONTAINER_LOGS,
				Action:         "Try checking container logs",
			},
		},
		{
			description: "makes err suggestion unhealthy status check (357)",
			errCode:     proto.StatusCode_STATUSCHECK_UNHEALTHY,
			expected: &proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_CHECK_READINESS_PROBE,
				Action:         "Try checking container config `readinessProbe`",
			},
		},
		{
			description: "makes err suggestion for failed image pulls (300)",
			errCode:     proto.StatusCode_STATUSCHECK_IMAGE_PULL_ERR,
			expected: &proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_CHECK_CONTAINER_IMAGE,
				Action:         "Try checking container config `image`",
			},
		},
		{
			description: "returns nil suggestion if no case matches",
			errCode:     proto.StatusCode_BUILD_CANCELLED,
			expected:    &NilSuggestion,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(
				test.expected,
				ContainerError{}.Make(test.errCode),
				cmp.AllowUnexported(proto.Suggestion{}),
				protocmp.Transform(),
			)
		})
	}
}
