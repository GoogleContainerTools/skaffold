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
	"os"
	"os/exec"
	"reflect"
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
			expected:     []string{"BUILD_CONTEXT=/some/path", "IMAGES=gcr.io/image/tag:mytag", "PUSH_IMAGE=false"},
		}, {
			description:  "make sure environ is correctly applied",
			tag:          "gcr.io/image/tag:anothertag",
			environ:      []string{"PATH=/path", "HOME=/root"},
			buildContext: "/some/path",
			expected:     []string{"BUILD_CONTEXT=/some/path", "HOME=/root", "IMAGES=gcr.io/image/tag:anothertag", "PATH=/path", "PUSH_IMAGE=false"},
		}, {
			description: "push image is true",
			tag:         "gcr.io/image/push:tag",
			pushImages:  true,
			expected:    []string{"BUILD_CONTEXT=", "IMAGES=gcr.io/image/push:tag", "PUSH_IMAGE=true"},
		}, {
			description:   "add additional env",
			tag:           "gcr.io/image/push:tag",
			pushImages:    true,
			additionalEnv: []string{"KUBECONTEXT=mycluster"},
			expected:      []string{"BUILD_CONTEXT=", "IMAGES=gcr.io/image/push:tag", "KUBECONTEXT=mycluster", "PUSH_IMAGE=true"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return test.environ })
			t.Override(&buildContext, func(string) (string, error) { return test.buildContext, nil })

			artifactBuilder := NewArtifactBuilder(test.pushImages, test.additionalEnv)
			actual, err := artifactBuilder.retrieveEnv(&latest.Artifact{}, test.tag)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestRetrieveCmd(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.Artifact
		tag         string
		expected    *exec.Cmd
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
			tag:      "image:tag",
			expected: expectedCmd("./build.sh", "workspace", nil, []string{"BUILD_CONTEXT=workspace", "IMAGES=image:tag", "PUSH_IMAGE=false"}),
		}, {
			description: "buildcommand with multiple args",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{
						BuildCommand: "./build.sh --flag --anotherflag",
					},
				},
			},
			tag:      "image:tag",
			expected: expectedCmd("./build.sh", "", []string{"--flag", "--anotherflag"}, []string{"BUILD_CONTEXT=", "IMAGES=image:tag", "PUSH_IMAGE=false"}),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return nil })
			t.Override(&buildContext, func(string) (string, error) { return test.artifact.Workspace, nil })

			builder := NewArtifactBuilder(false, nil)
			cmd, err := builder.retrieveCmd(test.artifact, test.tag)

			t.CheckNoError(err)
			// cmp.Diff cannot access unexported fields in *exec.Cmd, so use reflect.DeepEqual here directly
			if !reflect.DeepEqual(test.expected, cmd) {
				t.Errorf("Expected result different from actual result. Expected: \n%v, \nActual: \n%v", test.expected, cmd)
			}
		})
	}
}

func expectedCmd(buildCommand, dir string, args, env []string) *exec.Cmd {
	cmd := exec.Command(buildCommand, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
