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

package cmd

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewCmdRun(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Chdir()
		t.Override(&opts, config.SkaffoldOptions{})

		cmd := NewCmdRun()
		cmd.SilenceUsage = true
		cmd.Execute()

		t.CheckDeepEqual(false, opts.Tail)
		t.CheckDeepEqual(false, opts.Force)
		t.CheckDeepEqual(false, opts.EnableRPC)
	})
}

type mockRunRunner struct {
	runner.Runner
	testRan            bool
	deployRan          bool
	artifactImageNames []string
}

func (r *mockRunRunner) Build(_ context.Context, _ io.Writer, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	var result []graph.Artifact
	for _, artifact := range artifacts {
		imageName := artifact.ImageName
		r.artifactImageNames = append(r.artifactImageNames, imageName)
		result = append(result, graph.Artifact{
			ImageName: imageName,
		})
	}

	return result, nil
}

func (r *mockRunRunner) Test(context.Context, io.Writer, []graph.Artifact) error {
	r.testRan = true
	return nil
}

func (r *mockRunRunner) DeployAndLog(context.Context, io.Writer, []graph.Artifact) error {
	r.deployRan = true
	return nil
}

func TestDoRun(t *testing.T) {
	tests := []struct {
		description string
		skipTests   bool
	}{
		{
			description: "Run with skip tests set to true",
			skipTests:   true,
		},
		{
			description: "Run with skip tests set to false",
			skipTests:   false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			mockRunner := &mockRunRunner{}
			t.Override(&createRunner, func(io.Writer, config.SkaffoldOptions) (runner.Runner, []*latest.SkaffoldConfig, *runcontext.RunContext, error) {
				return mockRunner, []*latest.SkaffoldConfig{{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								{ImageName: "first"},
								{ImageName: "second-test"},
								{ImageName: "test"},
								{ImageName: "aaabbbccc"},
							},
						},
					},
				}}, nil, nil
			})
			t.Override(&opts, config.SkaffoldOptions{
				TargetImages: []string{"test"},
				SkipTests:    test.skipTests,
			})

			err := doRun(context.Background(), ioutil.Discard)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.skipTests, !mockRunner.testRan)
			t.CheckDeepEqual([]string{"second-test", "test"}, mockRunner.artifactImageNames)
		})
	}
}
