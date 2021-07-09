// +build windows

/*
Copyright 2021 The Skaffold Authors

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

package hooks

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildHooks(t *testing.T) {
	workDir, _ := filepath.Abs("./foo")
	tests := []struct {
		description string
		artifact    v1.Artifact
		image       string
		pushImage   bool
		preHookOut  string
		postHookOut string
	}{
		{
			description: "linux, darwin build hook",
			artifact: v1.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: v1.BuildHooks{
					PreHooks: []v1.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo pre-hook running with IMAGE=$IMAGE,PUSH_IMAGE=$PUSH_IMAGE,IMAGE_REPO=$IMAGE_REPO,IMAGE_TAG=$IMAGE_TAG,BUILD_CONTEXT=$BUILD_CONTEXT"},
						},
					},
					PostHooks: []v1.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with IMAGE=$IMAGE,PUSH_IMAGE=$PUSH_IMAGE,IMAGE_REPO=$IMAGE_REPO,IMAGE_TAG=$IMAGE_TAG,BUILD_CONTEXT=$BUILD_CONTEXT"},
						},
					},
				},
			},
			image:     "gcr.io/foo/img1:latest",
			pushImage: true,
		},
		{
			description: "windows build hook",
			artifact: v1.Artifact{
				ImageName: "img1",
				Workspace: "./foo",
				LifecycleHooks: v1.BuildHooks{
					PreHooks: []v1.HostHook{
						{
							OS:      []string{"windows"},
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with %IMAGE%,%PUSH_IMAGE%,%IMAGE_REPO%,%IMAGE_TAG%,%BUILD_CONTEXT%"},
						},
					},
					PostHooks: []v1.HostHook{
						{
							OS:      []string{"windows"},
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with %IMAGE%,%PUSH_IMAGE%,%IMAGE_REPO%,%IMAGE_TAG%,%BUILD_CONTEXT%"},
						},
					},
				},
			},
			image:       "gcr.io/foo/img1:latest",
			pushImage:   true,
			preHookOut:  fmt.Sprintf("pre-hook running with IMAGE=gcr.io/foo/img1:latest,PUSH_IMAGE=true,IMAGE_REPO=gcr.io/foo,IMAGE_TAG=latest,BUILD_CONTEXT=%s\r\n", workDir),
			postHookOut: fmt.Sprintf("post-hook running with IMAGE=gcr.io/foo/img1:latest,PUSH_IMAGE=true,IMAGE_REPO=gcr.io/foo,IMAGE_TAG=latest,BUILD_CONTEXT=%s\r\n", workDir),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			opts, err := NewBuildEnvOpts(&test.artifact, test.image, test.pushImage)
			t.CheckNoError(err)
			runner := BuildRunner(test.artifact.LifecycleHooks, opts)
			var preOut, postOut bytes.Buffer
			err = runner.RunPreHooks(context.Background(), &preOut)
			t.CheckNoError(err)
			t.CheckContains(test.preHookOut, preOut.String())
			err = runner.RunPostHooks(context.Background(), &postOut)
			t.CheckNoError(err)
			t.CheckContains(test.postHookOut, postOut.String())
		})
	}
}
