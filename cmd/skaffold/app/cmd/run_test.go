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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
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
	artifactImageNames []string
}

func (r *mockRunRunner) Build(_ context.Context, _ io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var result []build.Artifact
	for _, artifact := range artifacts {
		imageName := artifact.ImageName
		r.artifactImageNames = append(r.artifactImageNames, imageName)
		result = append(result, build.Artifact{
			ImageName: imageName,
		})
	}

	return result, nil
}

func (r *mockRunRunner) DeployAndLog(context.Context, io.Writer, []build.Artifact) error {
	return nil
}

func TestBuildImageFlag(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		mockRunner := &mockRunRunner{}
		t.Override(&createRunner, func(config.SkaffoldOptions) (runner.Runner, []*latest.SkaffoldConfig, error) {
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
			}}, nil
		})
		t.Override(&opts, config.SkaffoldOptions{
			TargetImages: []string{"test"},
		})

		err := doRun(context.Background(), ioutil.Discard)
		t.CheckNoError(err)
		t.CheckDeepEqual([]string{"second-test", "test"}, mockRunner.artifactImageNames)
	})
}
