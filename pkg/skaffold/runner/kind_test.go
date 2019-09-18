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

package runner

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLoadImagesInKindNodes(t *testing.T) {
	tests := []struct {
		description   string
		built         []build.Artifact
		deployed      []build.Artifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "load image",
			built:       []build.Artifact{{Tag: "tag1"}},
			deployed:    []build.Artifact{{Tag: "tag1"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRun("kind load docker-image tag1"),
		},
		{
			description: "load missing image",
			built:       []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			deployed:    []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "tag1").
				AndRun("kind load docker-image tag2"),
		},
		{
			description: "inspect error",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOutErr("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to inspect",
		},
		{
			description: "load error",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunErr("kind load docker-image tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to load",
		},
		{
			description: "ignore image that's not built",
			built:       []build.Artifact{{Tag: "built"}},
			deployed:    []build.Artifact{{Tag: "built"}, {Tag: "busybox"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRun("kind load docker-image built"),
		},
		{
			description: "no artifact",
			deployed:    []build.Artifact{},
		},
		{
			description: "no built artifact",
			built:       []build.Artifact{},
			deployed:    []build.Artifact{{Tag: "busybox"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			r := &SkaffoldRunner{
				builds: test.built,
				runCtx: &runcontext.RunContext{
					Opts: config.SkaffoldOptions{
						Namespace: "namespace",
					},
					KubeContext: "kubecontext",
				},
			}
			err := r.loadImagesInKindNodes(context.Background(), ioutil.Discard, test.deployed)

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckNoError(err)
			}
		})
	}
}
