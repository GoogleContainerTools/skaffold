package recommender

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
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
			description: "makes err suggestion unhealhty status check (357)",
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
			r := ContainerError{}
			t.CheckDeepEqual(test.expected, r.Make(test.errCode), cmp.AllowUnexported(proto.Suggestion{}), protocmp.Transform())
		})
	}
}
