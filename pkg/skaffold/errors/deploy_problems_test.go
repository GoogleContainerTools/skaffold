/*
Copyright 2020 The Skaffold Authors

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

package errors

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSuggestDeployFailedAction(t *testing.T) {
	tests := []struct {
		description string
		context     string
		isMinikube  bool
		expected    []*proto.Suggestion
	}{
		{
			description: "minikube status",
			context:     "minikube",
			isMinikube:  true,
			expected: []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
				Action: "Check if minikube is running using `minikube status` command and try again.",
			}},
		},
		{
			description: "minikube status named ctx",
			context:     "test_cluster",
			isMinikube:  true,
			expected: []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
				Action:         "Check if minikube is running using `minikube status -p test_cluster` command and try again.",
			}},
		},
		{
			description: "gke cluster",
			context:     "gke_test",
			isMinikube:  false,
			expected: []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
				Action:         "Check your connection for the cluster",
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runCtx := runcontext.RunContext{
				KubeContext: test.context,
			}
			t.Override(&isMinikube, func(string) bool {
				return test.isMinikube
			})
			actual := suggestDeployFailedAction(runCtx)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
