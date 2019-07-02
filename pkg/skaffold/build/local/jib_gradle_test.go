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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestBuildJibGradleToDocker(t *testing.T) {
	var tests = []struct {
		description   string
		artifact      *latest.JibGradleArtifact
		cmd           util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibGradleArtifact{},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :jibDockerBuild --image=img:tag"),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibGradleArtifact{Flags: []string{"--flag1", "--flag2"}},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :jibDockerBuild --image=img:tag --flag1 --flag2"),
		},
		{
			description: "build with project",
			artifact:    &latest.JibGradleArtifact{Project: "project"},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :project:jibDockerBuild --image=img:tag"),
		},
		{
			description:   "fail build",
			artifact:      &latest.JibGradleArtifact{},
			cmd:           testutil.FakeRunErr(t, "gradle -Djib.console=plain :jibDockerBuild --image=img:tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)

			api := &testutil.FakeAPIClient{
				TagToImageID: map[string]string{"img:tag": "imageID"},
			}

			builder := &Builder{
				pushImages:  false,
				localDocker: docker.NewLocalDaemon(api, nil, false, map[string]bool{}),
			}
			result, err := builder.buildJibGradle(context.Background(), ioutil.Discard, ".", test.artifact, "img:tag")

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("imageID", result)
			}
		})
	}
}

func TestBuildJibGradleToRegistry(t *testing.T) {
	var tests = []struct {
		description   string
		artifact      *latest.JibGradleArtifact
		cmd           util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "remote build",
			artifact:    &latest.JibGradleArtifact{},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :jib --image=img:tag"),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibGradleArtifact{Flags: []string{"--flag1", "--flag2"}},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :jib --image=img:tag --flag1 --flag2"),
		},
		{
			description: "build with project",
			artifact:    &latest.JibGradleArtifact{Project: "project"},
			cmd:         testutil.FakeRun(t, "gradle -Djib.console=plain :project:jib --image=img:tag"),
		},
		{
			description:   "fail build",
			artifact:      &latest.JibGradleArtifact{},
			cmd:           testutil.FakeRunErr(t, "gradle -Djib.console=plain :jib --image=img:tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "gradle build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)
			t.Override(&getRemoteDigest, func(identifier string, _ map[string]bool) (string, error) {
				if identifier == "img:tag" {
					return "digest", nil
				}
				return "", errors.New("unknown remote tag")
			})

			builder := &Builder{
				pushImages:  true,
				localDocker: docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, map[string]bool{}),
			}
			result, err := builder.buildJibGradle(context.Background(), ioutil.Discard, ".", test.artifact, "img:tag")

			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("digest", result)
			}
		})
	}
}
