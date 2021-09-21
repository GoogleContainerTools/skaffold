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
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerCLIBuild(t *testing.T) {
	tests := []struct {
		description     string
		localBuild      latestV1.LocalBuild
		cfg             mockConfig
		extraEnv        []string
		expectedEnv     []string
		err             error
		expectedErr     error
		wantDockerCLI   bool
		expectedErrCode proto.StatusCode
	}{
		{
			description: "docker build",
			localBuild:  latestV1.LocalBuild{},
			cfg:         mockConfig{runMode: config.RunModes.Dev},
			expectedEnv: []string{"KEY=VALUE"},
		},
		{
			description: "extra env",
			localBuild:  latestV1.LocalBuild{},
			extraEnv:    []string{"OTHER=VALUE"},
			expectedEnv: []string{"KEY=VALUE", "OTHER=VALUE"},
		},
		{
			description:   "buildkit",
			localBuild:    latestV1.LocalBuild{UseBuildkit: util.BoolPtr(true)},
			wantDockerCLI: true,
			expectedEnv:   []string{"KEY=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description:   "buildkit and extra env",
			localBuild:    latestV1.LocalBuild{UseBuildkit: util.BoolPtr(true)},
			wantDockerCLI: true,
			extraEnv:      []string{"OTHER=VALUE"},
			expectedEnv:   []string{"KEY=VALUE", "OTHER=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description:   "env var collisions",
			localBuild:    latestV1.LocalBuild{UseBuildkit: util.BoolPtr(true)},
			wantDockerCLI: true,
			extraEnv:      []string{"KEY=OTHER_VALUE", "DOCKER_BUILDKIT=0"},
			// env var collisions are handled by cmd.Run(). Last one wins.
			expectedEnv: []string{"KEY=VALUE", "KEY=OTHER_VALUE", "DOCKER_BUILDKIT=0", "DOCKER_BUILDKIT=1"},
		},
		{
			description: "docker build internal error",
			localBuild:  latestV1.LocalBuild{UseDockerCLI: true},
			err:         errdefs.Cancelled(fmt.Errorf("cancelled")),
			expectedErr: newBuildError(errdefs.Cancelled(fmt.Errorf("cancelled")), mockConfig{runMode: config.RunModes.Dev}),
		},
		{
			description:     "docker build no space left error with prune for dev",
			localBuild:      latestV1.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{runMode: config.RunModes.Dev, prune: false},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("Docker ran out of memory. Please run 'docker system prune' to removed unused docker data or Run skaffold dev with --cleanup=true to clean up images built by skaffold"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build no space left error with prune for build",
			localBuild:      latestV1.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{runMode: config.RunModes.Build, prune: false},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("no space left. Docker ran out of memory. Please run 'docker system prune' to removed unused docker data"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build no space left error with prune true",
			localBuild:      latestV1.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{prune: true},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("no space left. Docker ran out of memory. Please run 'docker system prune' to removed unused docker data"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build system error",
			localBuild:      latestV1.LocalBuild{UseDockerCLI: true},
			err:             errdefs.System(fmt.Errorf("something else")),
			expectedErr:     fmt.Errorf("something else"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_SYSTEM_ERR,
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
				var pruneFlag string
				if test.cfg.Prune() {
					pruneFlag = " --force-rm"
				}
				mockCmd = testutil.CmdRunErr(
					"docker build . --file "+dockerfilePath+" -t tag"+pruneFlag,
					test.err,
				)
				t.Override(&util.DefaultExecCommand, mockCmd)
			}
			if test.wantDockerCLI {
				mockCmd = testutil.CmdRunEnv(
					"docker build . --file "+dockerfilePath+" -t tag",
					test.expectedEnv,
				)
				t.Override(&util.DefaultExecCommand, mockCmd)
			}
			t.Override(&util.OSEnviron, func() []string { return []string{"KEY=VALUE"} })

			builder := NewArtifactBuilder(fakeLocalDaemonWithExtraEnv(test.extraEnv), test.cfg, test.localBuild.UseDockerCLI, test.localBuild.UseBuildkit, false, mockArtifactResolver{make(map[string]string)}, nil)

			artifact := &latestV1.Artifact{
				Workspace: ".",
				ArtifactType: latestV1.ArtifactType{
					DockerArtifact: &latestV1.DockerArtifact{
						DockerfilePath: "Dockerfile",
					},
				},
			}

			_, err := builder.Build(context.Background(), ioutil.Discard, artifact, "tag")
			t.CheckError(test.err != nil, err)
			if mockCmd != nil {
				t.CheckDeepEqual(1, mockCmd.TimesCalled())
			}
			if test.err != nil && test.expectedErrCode != 0 {
				if ae, ok := err.(*sErrors.ErrDef); ok {
					t.CheckDeepEqual(test.expectedErrCode, ae.StatusCode(), protocmp.Transform())
					t.CheckErrorContains(test.expectedErr.Error(), ae)
				} else {
					t.Fatalf("expected to find an actionable error. not found")
				}
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

type mockConfig struct {
	runMode config.RunMode
	prune   bool
}

func (m mockConfig) GetKubeContext() string {
	return ""
}

func (m mockConfig) GlobalConfig() string {
	return ""
}

func (m mockConfig) MinikubeProfile() string {
	return ""
}

func (m mockConfig) GetInsecureRegistries() map[string]bool {
	return map[string]bool{}
}

func (m mockConfig) Mode() config.RunMode {
	return m.runMode
}

func (m mockConfig) Prune() bool {
	return m.prune
}

func (m mockConfig) ContainerDebugging() bool {
	return false
}
