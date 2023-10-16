/*
Copyright 2023 The Skaffold Authors

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

package runner

import (
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type MockActionsRunner struct {
	ExecEnvByAction map[string]actions.ExecEnv
	OrderedExecEnvs []actions.ExecEnv
	AcsByExecEnv    map[actions.ExecEnv][]string
}

func (mar MockActionsRunner) Exec(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, action string) error {
	return nil
}

func (mar MockActionsRunner) ExecAll(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	return nil
}

func TestActionsRunner_Creation(t *testing.T) {
	tests := []struct {
		description       string
		pipelinesCfg      map[string]latest.Pipeline
		orderedPipelines  []string
		envFile           string
		expectedDockerAcs []string
		inputRunCtx       runcontext.RunContext
		expectedAr        actions.Runner
	}{
		{
			description: "default to docker when no execution mode is specified",
			inputRunCtx: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{
					"config1": {CustomActions: []latest.Action{{Name: "action1"}, {Name: "action2"}}},
					"config2": {
						CustomActions: []latest.Action{
							{
								Name: "action3",
							},
							{
								Name: "action4",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
									},
								},
							},
						},
					},
				}, []string{"config1", "config2"}),
			},
			expectedDockerAcs: []string{"action1", "action2", "action3"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var createdDockerAcs []string
			t.Override(&docker.NewExecEnv, func(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, resources []*latest.PortForwardResource, network string, envMap map[string]string, acs []latest.Action) (*docker.ExecEnv, error) {
				for _, a := range acs {
					createdDockerAcs = append(createdDockerAcs, a.Name)
				}
				return &docker.ExecEnv{}, nil
			})

			_, err := GetActionsRunner(context.TODO(), &test.inputRunCtx, &label.DefaultLabeller{}, "", "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedDockerAcs, createdDockerAcs)
		})
	}
}
