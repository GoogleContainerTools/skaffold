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
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
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
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			t.Override(&docker.DefaultAuthHelper, testAuthHelper{})
			t.Override(&docker.EvalBuildArgs, func(mode config.RunMode, workspace string, a *latest.DockerArtifact) (map[string]*string, error) {
				return a.BuildArgs, nil
			})
			t.Override(&util.DefaultExecCommand, testutil.CmdRunEnv(
				"docker build . --file "+dockerfilePath+" -t tag --force-rm",
				test.expectedEnv,
			))
			t.Override(&cluster.GetClient, func() cluster.Client { return fakeMinikubeClient{} })
			t.Override(&util.OSEnviron, func() []string { return []string{"KEY=VALUE"} })
			t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
				return fakeLocalDaemonWithExtraEnv(test.extraEnv), nil
			})

			builder, err := NewBuilder(&mockConfig{
				local: test.localBuild,
			})
			t.CheckNoError(err)

			artifact := &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}

			_, err = builder.buildDocker(context.Background(), ioutil.Discard, artifact, "tag", test.mode)
			t.CheckNoError(err)
		})
	}
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}

func fakeLocalDaemonWithExtraEnv(extraEnv []string) docker.LocalDaemon {
	return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, extraEnv, false, nil)
}

type fakeMinikubeClient struct{}

func (fakeMinikubeClient) IsMinikube(kubeContext string) bool { return false }
func (fakeMinikubeClient) MinikubeExec(arg ...string) (*exec.Cmd, error) {
	return exec.Command("minikube", arg...), nil
}
