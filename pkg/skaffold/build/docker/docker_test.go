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

package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerCLIBuild(t *testing.T) {
	tests := []struct {
		description string
		localBuild  latest.LocalBuild
		mode        config.RunMode
		extraEnv    []string
		expectedEnv []string
		err         error
		expectedErr error
	}{
		{
			description: "docker build",
			localBuild:  latest.LocalBuild{},
			mode:        config.RunModes.Dev,
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
		{
			description: "docker build internal error",
			localBuild:  latest.LocalBuild{UseDockerCLI: true},
			err:         errdefs.Cancelled(fmt.Errorf("cancelled")),
			expectedErr: newBuildError(errdefs.Cancelled(fmt.Errorf("cancelled"))),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
				return args, nil
			})
			t.Override(&docker.DefaultAuthHelper, stubAuth{})
			var mockCmd *testutil.FakeCmd

			if test.err != nil {
				mockCmd = testutil.CmdRunErr(
					"docker build . --file "+dockerfilePath+" -t tag",
					test.err,
				)
				t.Override(&util.DefaultExecCommand, mockCmd)
			} else if test.localBuild.UseBuildkit || test.localBuild.UseDockerCLI {
				mockCmd = testutil.CmdRunEnv(
					"docker build . --file "+dockerfilePath+" -t tag",
					test.expectedEnv,
				)
				t.Override(&util.DefaultExecCommand, mockCmd)
			}
			t.Override(&util.OSEnviron, func() []string { return []string{"KEY=VALUE"} })

			builder := NewArtifactBuilder(fakeLocalDaemonWithExtraEnv(test.extraEnv), test.localBuild.UseDockerCLI, test.localBuild.UseBuildkit, false, false, test.mode, nil, mockArtifactResolver{make(map[string]string)})

			artifact := &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}

			_, err := builder.Build(context.Background(), ioutil.Discard, artifact, "tag")
			t.CheckError(test.err != nil, err)
			if mockCmd != nil {
				t.CheckDeepEqual(1, mockCmd.TimesCalled())
			}
		})
	}
}

func fakeLocalDaemonWithExtraEnv(extraEnv []string) docker.LocalDaemon {
	return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, extraEnv, false, nil)
}

type mockArtifactResolver struct {
	m map[string]string
}

func (r mockArtifactResolver) GetImageTag(imageName string) (string, bool) {
	if r.m == nil {
		return "", false
	}
	val, found := r.m[imageName]
	return val, found
}

type stubAuth struct{}

func (t stubAuth) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t stubAuth) GetAllAuthConfigs(context.Context) (map[string]types.AuthConfig, error) {
	return nil, nil
}
