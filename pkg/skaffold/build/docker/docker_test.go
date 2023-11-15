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
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDockerCLIBuild(t *testing.T) {
	tests := []struct {
		description     string
		localBuild      latest.LocalBuild
		cliFlags        []string
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
			localBuild:  latest.LocalBuild{},
			cfg:         mockConfig{runMode: config.RunModes.Dev},
			expectedEnv: []string{"KEY=VALUE"},
		},
		{
			description: "extra env",
			localBuild:  latest.LocalBuild{},
			extraEnv:    []string{"OTHER=VALUE"},
			expectedEnv: []string{"KEY=VALUE", "OTHER=VALUE"},
		},
		{
			description:   "buildkit",
			localBuild:    latest.LocalBuild{UseBuildkit: util.Ptr(true)},
			wantDockerCLI: true,
			expectedEnv:   []string{"KEY=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description:   "cliFlags",
			cliFlags:      []string{"--platform", "linux/amd64"},
			localBuild:    latest.LocalBuild{},
			wantDockerCLI: true,
			expectedEnv:   []string{"KEY=VALUE"},
		},
		{
			description:   "buildkit and extra env",
			localBuild:    latest.LocalBuild{UseBuildkit: util.Ptr(true)},
			wantDockerCLI: true,
			extraEnv:      []string{"OTHER=VALUE"},
			expectedEnv:   []string{"KEY=VALUE", "OTHER=VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description:   "env var collisions",
			localBuild:    latest.LocalBuild{UseBuildkit: util.Ptr(true)},
			wantDockerCLI: true,
			extraEnv:      []string{"KEY=OTHER_VALUE", "DOCKER_BUILDKIT=0"},
			// DOCKER_BUILDKIT should be overridden
			expectedEnv: []string{"KEY=OTHER_VALUE", "DOCKER_BUILDKIT=1"},
		},
		{
			description: "docker build internal error",
			localBuild:  latest.LocalBuild{UseDockerCLI: true},
			err:         errdefs.Cancelled(fmt.Errorf("cancelled")),
			expectedErr: newBuildError(errdefs.Cancelled(fmt.Errorf("cancelled")), mockConfig{runMode: config.RunModes.Dev}),
		},
		{
			description:     "docker build no space left error with prune for dev",
			localBuild:      latest.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{runMode: config.RunModes.Dev, prune: false},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("Docker ran out of memory. Please run 'docker system prune' to removed unused docker data or Run skaffold dev with --cleanup=true to clean up images built by skaffold"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build no space left error with prune for build",
			localBuild:      latest.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{runMode: config.RunModes.Build, prune: false},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("no space left. Docker ran out of memory. Please run 'docker system prune' to removed unused docker data"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build no space left error with prune true",
			localBuild:      latest.LocalBuild{UseDockerCLI: true},
			cfg:             mockConfig{prune: true},
			err:             errdefs.System(fmt.Errorf("no space left")),
			expectedErr:     fmt.Errorf("no space left. Docker ran out of memory. Please run 'docker system prune' to removed unused docker data"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR,
		},
		{
			description:     "docker build system error",
			localBuild:      latest.LocalBuild{UseDockerCLI: true},
			err:             errdefs.System(fmt.Errorf("something else")),
			expectedErr:     fmt.Errorf("something else"),
			expectedErrCode: proto.StatusCode_BUILD_DOCKER_SYSTEM_ERR,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
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
				cmdLine := "docker build . --file " + dockerfilePath + " -t tag"
				if len(test.cliFlags) > 0 {
					cmdLine += " " + strings.Join(test.cliFlags, " ")
				}
				mockCmd = testutil.CmdRunEnv(cmdLine, test.expectedEnv)
				t.Override(&util.DefaultExecCommand, mockCmd)
			}
			t.Override(&util.OSEnviron, func() []string { return []string{"KEY=VALUE"} })

			builder := NewArtifactBuilder(fakeLocalDaemonWithExtraEnv(test.extraEnv), test.cfg, test.localBuild.UseDockerCLI, false, test.localBuild.UseBuildkit, false, mockArtifactResolver{make(map[string]string)}, nil)

			artifact := &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
						CliFlags:       test.cliFlags,
					},
				},
			}

			_, err := builder.Build(context.Background(), io.Discard, artifact, "tag", platform.Matcher{})
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

func TestDockerCLICheckCacheFromArgs(t *testing.T) {
	tests := []struct {
		description       string
		artifact          *latest.Artifact
		tag               string
		expectedCacheFrom []string
	}{
		{
			description: "multiple cache-from images",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/k8s-skaffold/test",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"from/image1", "from/image2"},
					},
				},
			},
			tag:               "tag",
			expectedCacheFrom: []string{"from/image1", "from/image2"},
		},
		{
			description: "cache-from self uses tagged image",
			artifact: &latest.Artifact{
				ImageName: "gcr.io/k8s-skaffold/test",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"gcr.io/k8s-skaffold/test"},
					},
				},
			},
			tag:               "gcr.io/k8s-skaffold/test:tagged",
			expectedCacheFrom: []string{"gcr.io/k8s-skaffold/test:tagged"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			a := *test.artifact
			a.Workspace = "."
			a.DockerArtifact.DockerfilePath = dockerfilePath
			t.Override(&docker.DefaultAuthHelper, stubAuth{})
			t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
				return args, nil
			})

			mockCmd := testutil.CmdRun(
				"docker build . --file " + dockerfilePath + " -t " + test.tag + " --cache-from " + strings.Join(test.expectedCacheFrom, " --cache-from "),
			)
			t.Override(&util.DefaultExecCommand, mockCmd)

			builder := NewArtifactBuilder(fakeLocalDaemonWithExtraEnv([]string{}), mockConfig{}, true, false, util.Ptr(false), false, mockArtifactResolver{make(map[string]string)}, nil)
			_, err := builder.Build(context.Background(), io.Discard, &a, test.tag, platform.Matcher{})
			t.CheckNoError(err)
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
