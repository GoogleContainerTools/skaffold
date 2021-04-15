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

package deploy

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
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
				Action:         "Check if minikube is running using \"minikube status\" command and try again.",
			}},
		},
		{
			description: "minikube status named ctx",
			context:     "test_cluster",
			isMinikube:  true,
			expected: []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_MINIKUBE_STATUS,
				Action:         "Check if minikube is running using \"minikube status -p test_cluster\" command and try again.",
			}},
		},
		{
			description: "gke cluster",
			context:     "gke_test",
			expected: []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_CLUSTER_CONNECTION,
				Action:         "Check your connection for the cluster",
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := mockConfig{kubeContext: test.context}
			if test.isMinikube {
				cfg.minikube = test.context
			}
			actual := suggestDeployFailedAction(cfg)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type mockConfig struct {
	minikube    string
	kubeContext string
}

func (m mockConfig) MinikubeProfile() string                { return m.minikube }
func (m mockConfig) GetPipelines() []latest.Pipeline        { return []latest.Pipeline{} }
func (m mockConfig) GetWorkingDir() string                  { return "" }
func (m mockConfig) GlobalConfig() string                   { return "" }
func (m mockConfig) ConfigurationFile() string              { return "" }
func (m mockConfig) DefaultRepo() *string                   { return &m.minikube }
func (m mockConfig) SkipRender() bool                       { return true }
func (m mockConfig) Prune() bool                            { return true }
func (m mockConfig) GetKubeContext() string                 { return m.kubeContext }
func (m mockConfig) GetInsecureRegistries() map[string]bool { return map[string]bool{} }
func (m mockConfig) Mode() config.RunMode                   { return config.RunModes.Dev }
