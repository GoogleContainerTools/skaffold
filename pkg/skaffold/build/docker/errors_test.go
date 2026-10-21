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
	"io"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestDockerBuildError(t *testing.T) {
	tests := []struct {
		description    string
		dockerfilepath string
		expected       string
		shouldErr      bool
	}{
		{
			description:    "docker file present",
			dockerfilepath: "Dockerfile",
		},
		{
			description:    "docker file not present",
			dockerfilepath: "DockerfileNotExist",
			shouldErr:      true,
			expected: `Dockerfile not found. Please check config \'dockerfile\' for artifact test-image.
Refer https://skaffold.dev/docs/references/yaml/#build-artifacts-docker for details.`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("Dockerfile").Chdir()
			dockerfilePath, _ := filepath.Abs("Dockerfile")
			t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
				return args, nil
			})
			t.Override(&util.DefaultExecCommand, testutil.CmdRun(
				"docker build . --file "+dockerfilePath+" -t tag",
			))
			t.Override(&docker.DefaultAuthHelper, stubAuth{})
			builder := NewArtifactBuilder(fakeLocalDaemonWithExtraEnv([]string{}, false), mockConfig{}, true, nil, false, false, mockArtifactResolver{make(map[string]string)}, nil)

			artifact := &latest.Artifact{
				ImageName: "test-image",
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: test.dockerfilepath,
					},
				},
			}

			_, err := builder.Build(context.Background(), io.Discard, artifact, "tag", platform.Matcher{})
			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains("", err)
			}
		})
	}
}
