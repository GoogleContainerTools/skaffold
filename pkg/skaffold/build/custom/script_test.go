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

package custom

import (
	"context"
	"io/ioutil"
	"os/exec"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRetrieveEnv(t *testing.T) {
	tests := []struct {
		description   string
		tag           string
		pushImages    bool
		buildContext  string
		additionalEnv []string
		environ       []string
		expected      []string
	}{

		{
			description:  "make sure tags are correct",
			tag:          "gcr.io/image/tag:mytag",
			environ:      nil,
			buildContext: "/some/path",
			expected:     []string{"IMAGE=gcr.io/image/tag:mytag", "IMAGES=gcr.io/image/tag:mytag", "PUSH_IMAGE=false", "BUILD_CONTEXT=/some/path", "IMAGE_REPO=gcr.io/image/tag", "IMAGE_TAG=mytag"},
		}, {
			description:  "make sure environ is correctly applied",
			tag:          "gcr.io/image/tag:anothertag",
			environ:      []string{"PATH=/path", "HOME=/root"},
			buildContext: "/some/path",
			expected:     []string{"IMAGE=gcr.io/image/tag:anothertag", "IMAGES=gcr.io/image/tag:anothertag", "PUSH_IMAGE=false", "BUILD_CONTEXT=/some/path", "IMAGE_REPO=gcr.io/image/tag", "IMAGE_TAG=anothertag", "PATH=/path", "HOME=/root"},
		}, {
			description: "push image is true",
			tag:         "gcr.io/image/push:tag",
			pushImages:  true,
			expected:    []string{"IMAGE=gcr.io/image/push:tag", "IMAGES=gcr.io/image/push:tag", "PUSH_IMAGE=true", "BUILD_CONTEXT=", "IMAGE_REPO=gcr.io/image/push", "IMAGE_TAG=tag"},
		}, {
			description:   "add additional env",
			tag:           "gcr.io/image/push:tag",
			pushImages:    true,
			additionalEnv: []string{"KUBECONTEXT=mycluster"},
			expected:      []string{"IMAGE=gcr.io/image/push:tag", "IMAGES=gcr.io/image/push:tag", "PUSH_IMAGE=true", "BUILD_CONTEXT=", "IMAGE_REPO=gcr.io/image/push", "IMAGE_TAG=tag", "KUBECONTEXT=mycluster"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.environ })
			t.Override(&buildContext, func(string) (string, error) { return test.buildContext, nil })

			builder := NewArtifactBuilder(nil, nil, test.pushImages, test.additionalEnv)
			actual, err := builder.retrieveEnv(&latest.Artifact{}, test.tag)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestRetrieveCmd(t *testing.T) {
	tests := []struct {
		description       string
		artifact          *latest.Artifact
		tag               string
		env               []string
		expected          *exec.Cmd
		expectedOnWindows *exec.Cmd
	}{
		{
			description: "artifact with workspace set",
			artifact: &latest.Artifact{
				Workspace: "workspace",
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{
						BuildCommand: "./build.sh",
					},
				},
			},
			tag:               "image:tag",
			expected:          expectedCmd("workspace", "sh", []string{"-c", "./build.sh"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=workspace", "IMAGE_REPO=image", "IMAGE_TAG=tag"}),
			expectedOnWindows: expectedCmd("workspace", "cmd.exe", []string{"/C", "./build.sh"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=workspace", "IMAGE_REPO=image", "IMAGE_TAG=tag"}),
		},
		{
			description: "buildcommand with multiple args",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{
						BuildCommand: "./build.sh --flag=$IMAGES --anotherflag",
					},
				},
			},
			tag:               "image:tag",
			expected:          expectedCmd("", "sh", []string{"-c", "./build.sh --flag=$IMAGES --anotherflag"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=", "IMAGE_REPO=image", "IMAGE_TAG=tag"}),
			expectedOnWindows: expectedCmd("", "cmd.exe", []string{"/C", "./build.sh --flag=$IMAGES --anotherflag"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=", "IMAGE_REPO=image", "IMAGE_TAG=tag"}),
		},
		{
			description: "buildcommand with go template",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{
						BuildCommand: "./build.sh --flag={{ .FLAG }}",
					},
				},
			},
			tag:               "image:tag",
			env:               []string{"FLAG=some-flag"},
			expected:          expectedCmd("", "sh", []string{"-c", "./build.sh --flag=some-flag"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=", "IMAGE_REPO=image", "IMAGE_TAG=tag", "FLAG=some-flag"}),
			expectedOnWindows: expectedCmd("", "cmd.exe", []string{"/C", "./build.sh --flag=some-flag"}, []string{"IMAGE=image:tag", "IMAGES=image:tag", "PUSH_IMAGE=false", "BUILD_CONTEXT=", "IMAGE_REPO=image", "IMAGE_TAG=tag", "FLAG=some-flag"}),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.env })
			t.Override(&buildContext, func(string) (string, error) { return test.artifact.Workspace, nil })

			builder := NewArtifactBuilder(nil, nil, false, nil)
			cmd, err := builder.retrieveCmd(context.Background(), ioutil.Discard, test.artifact, test.tag)

			t.CheckNoError(err)
			if runtime.GOOS == "windows" {
				t.CheckDeepEqual(test.expectedOnWindows.Args, cmd.Args)
				t.CheckDeepEqual(test.expectedOnWindows.Dir, cmd.Dir)
				t.CheckDeepEqual(test.expectedOnWindows.Env, cmd.Env)
			} else {
				t.CheckDeepEqual(test.expected.Args, cmd.Args)
				t.CheckDeepEqual(test.expected.Dir, cmd.Dir)
				t.CheckDeepEqual(test.expected.Env, cmd.Env)
			}
		})
	}
}

func expectedCmd(dir, buildCommand string, args, env []string) *exec.Cmd {
	cmd := exec.Command(buildCommand, args...)
	cmd.Dir = dir
	cmd.Env = env
	return cmd
}
