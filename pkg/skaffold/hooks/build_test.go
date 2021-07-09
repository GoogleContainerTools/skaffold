// +build !windows

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
							Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_IMAGE=$SKAFFOLD_IMAGE,SKAFFOLD_PUSH_IMAGE=$SKAFFOLD_PUSH_IMAGE,SKAFFOLD_IMAGE_REPO=$SKAFFOLD_IMAGE_REPO,SKAFFOLD_IMAGE_TAG=$SKAFFOLD_IMAGE_TAG,SKAFFOLD_BUILD_CONTEXT=$SKAFFOLD_BUILD_CONTEXT"},
						},
					},
					PostHooks: []v1.HostHook{
						{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_IMAGE=$SKAFFOLD_IMAGE,SKAFFOLD_PUSH_IMAGE=$SKAFFOLD_PUSH_IMAGE,SKAFFOLD_IMAGE_REPO=$SKAFFOLD_IMAGE_REPO,SKAFFOLD_IMAGE_TAG=$SKAFFOLD_IMAGE_TAG,SKAFFOLD_BUILD_CONTEXT=$SKAFFOLD_BUILD_CONTEXT"},
						},
					},
				},
			},
			image:       "gcr.io/foo/img1:latest",
			pushImage:   true,
			preHookOut:  fmt.Sprintf("pre-hook running with SKAFFOLD_IMAGE=gcr.io/foo/img1:latest,SKAFFOLD_PUSH_IMAGE=true,SKAFFOLD_IMAGE_REPO=gcr.io/foo,SKAFFOLD_IMAGE_TAG=latest,SKAFFOLD_BUILD_CONTEXT=%s\n", workDir),
			postHookOut: fmt.Sprintf("post-hook running with SKAFFOLD_IMAGE=gcr.io/foo/img1:latest,SKAFFOLD_PUSH_IMAGE=true,SKAFFOLD_IMAGE_REPO=gcr.io/foo,SKAFFOLD_IMAGE_TAG=latest,SKAFFOLD_BUILD_CONTEXT=%s\n", workDir),
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
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with %SKAFFOLD_IMAGE%,%SKAFFOLD_PUSH_IMAGE%,%SKAFFOLD_IMAGE_REPO%,%SKAFFOLD_IMAGE_TAG%,%SKAFFOLD_BUILD_CONTEXT%"},
						},
					},
					PostHooks: []v1.HostHook{
						{
							OS:      []string{"windows"},
							Command: []string{"cmd.exe", "/C", "echo pre-hook running with %SKAFFOLD_IMAGE%,%SKAFFOLD_PUSH_IMAGE%,%SKAFFOLD_IMAGE_REPO%,%SKAFFOLD_IMAGE_TAG%,%SKAFFOLD_BUILD_CONTEXT%"},
						},
					},
				},
			},
			image:     "gcr.io/foo/img1:latest",
			pushImage: true,
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
