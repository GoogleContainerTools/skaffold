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

package local

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerCLIBuild(t *testing.T) {
	tests := []struct {
		description string
		localBuild  latest.LocalBuild
		extraEnv    []string
		expectedEnv []string
	}{
		{
			description: "docker build",
			localBuild:  latest.LocalBuild{},
			expectedEnv: []string{"KEY=VALUE"},
		},
		{
			description: "extra env",
			localBuild:  latest.LocalBuild{},
			extraEnv:    []string{"OTHER=VALUE"},
			expectedEnv: []string{"KEY=VALUE", "OTHER=VALUE"},
		},
		{
			description: "buildkit",
			localBuild:  latest.LocalBuild{UseBuildkit: true},
			expectedEnv: []string{"KEY=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description: "buildkit and extra env",
			localBuild:  latest.LocalBuild{UseBuildkit: true},
			extraEnv:    []string{"OTHER=VALUE"},
			expectedEnv: []string{"KEY=VALUE", "OTHER=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description: "env var collisions",
			localBuild:  latest.LocalBuild{UseBuildkit: true},
			extraEnv:    []string{"KEY=OTHER_VALUE", "DOCKER_BUILDKIT=0"},
			// env var collisions are handled by cmd.Run(). Last one wins.
			expectedEnv: []string{"KEY=VALUE", "KEY=OTHER_VALUE", "DOCKER_BUILDKIT=0", "DOCKER_BUILDKIT=1"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			t.Override(&docker.DefaultAuthHelper, testAuthHelper{})
			t.Override(&util.DefaultExecCommand, testutil.CmdRunEnv(
				"docker build . --file "+dockerfilePath+" -t tag --force-rm",
				test.expectedEnv,
			))
			t.Override(&util.OSEnviron, func() []string { return []string{"KEY=VALUE"} })
			t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, test.extraEnv, false, nil), nil
			})

			builder, err := NewBuilder(stubRunContext(test.localBuild))
			t.CheckNoError(err)

			artifact := &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}

			_, err = builder.buildDocker(context.Background(), ioutil.Discard, artifact, "tag")
			t.CheckNoError(err)
		})
	}
}
