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

package error

import (
	"fmt"
	"testing"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUserError(t *testing.T) {
	tests := []struct {
		description string
		statusCode  proto.StatusCode
		expected    proto.StatusCode
		expectedErr string
		err         error
	}{
		{
			description: "internal system error",
			err:         fmt.Errorf("Error: (Internal Server Error: the server is currently unable to handle the request)"),
			statusCode:  proto.StatusCode_DEPLOY_KUSTOMIZE_USER_ERR,
			expected:    proto.StatusCode_DEPLOY_CLUSTER_INTERNAL_SYSTEM_ERR,
			expectedErr: "Deploy Failed. Error: (Internal Server Error: the server is currently unable to handle the request)." +
				" Something went wrong.",
		},
		{
			description: "not an internal system err",
			err:         fmt.Errorf("helm tiller not running"),
			statusCode:  proto.StatusCode_DEPLOY_HELM_USER_ERR,
			expected:    proto.StatusCode_DEPLOY_HELM_USER_ERR,
			expectedErr: "helm tiller not running",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&internalSystemErrSuggestion, func(_ interface{}) []*proto.Suggestion {
				return []*proto.Suggestion{{
					Action: "Something went wrong",
				}}
			})
			actual := UserError(test.err, test.statusCode)
			switch actualType := actual.(type) {
			case sErrors.ErrDef:
				t.CheckDeepEqual(test.expected, actualType.StatusCode())
			case sErrors.Problem:
				t.CheckDeepEqual(test.expected, actualType.ErrCode)
				actualErr := sErrors.ShowAIError(nil, actualType)
				t.CheckErrorContains(test.expectedErr, actualErr)
			default:
				t.CheckErrorContains(test.expectedErr, actualType)
			}
		})
	}
}
