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
	"errors"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type ImageLoadingTest = struct {
	description   string
	cluster       string
	built         []build.Artifact
	deployed      []build.Artifact
	commands      util.Command
	shouldErr     bool
	expectedError string
}

func TestLoadImagesInKindNodes(t *testing.T) {
	tests := []ImageLoadingTest{
		{
			description: "load image",
			cluster:     "kind",
			built:       []build.Artifact{{Tag: "tag1"}},
			deployed:    []build.Artifact{{Tag: "tag1"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("kind load docker-image --name kind tag1", "output: image loaded"),
		},
		{
			description: "load missing image",
			cluster:     "other-kind",
			built:       []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			deployed:    []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "tag1").
				AndRunOut("kind load docker-image --name other-kind tag2", "output: image loaded"),
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
			cluster:     "kind",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOutErr("kind load docker-image --name kind tag", "output: error!", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "output: error!",
		},
		{
			description: "ignore image that's not built",
			cluster:     "kind",
			built:       []build.Artifact{{Tag: "built"}},
			deployed:    []build.Artifact{{Tag: "built"}, {Tag: "busybox"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("kind load docker-image --name kind built", ""),
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

	runImageLoadingTests(t, tests, func(r *SkaffoldRunner, test ImageLoadingTest) error {
		return r.loadImagesInKindNodes(context.Background(), ioutil.Discard, test.cluster, test.deployed)
	})
}

func TestLoadImagesInK3dNodes(t *testing.T) {
	tests := []ImageLoadingTest{
		{
			description: "load image",
			cluster:     "k3d",
			built:       []build.Artifact{{Tag: "tag1"}},
			deployed:    []build.Artifact{{Tag: "tag1"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("k3d image import --cluster k3d tag1", "output: image loaded"),
		},
		{
			description: "load missing image",
			cluster:     "other-k3d",
			built:       []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			deployed:    []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "tag1").
				AndRunOut("k3d image import --cluster other-k3d tag2", "output: image loaded"),
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
			cluster:     "k3d",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOutErr("k3d image import --cluster k3d tag", "output: error!", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "output: error!",
		},
		{
			description: "ignore image that's not built",
			cluster:     "k3d",
			built:       []build.Artifact{{Tag: "built"}},
			deployed:    []build.Artifact{{Tag: "built"}, {Tag: "busybox"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("k3d image import --cluster k3d built", ""),
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

	runImageLoadingTests(t, tests, func(r *SkaffoldRunner, test ImageLoadingTest) error {
		return r.loadImagesInK3dNodes(context.Background(), ioutil.Discard, test.cluster, test.deployed)
	})
}

func runImageLoadingTests(t *testing.T, tests []ImageLoadingTest, loadingFunc func(r *SkaffoldRunner, test ImageLoadingTest) error) {
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			runCtx := &runcontext.RunContext{
				Opts: config.SkaffoldOptions{
					Namespace: "namespace",
				},
				KubeContext: "kubecontext",
			}

			r := &SkaffoldRunner{
				runCtx:     runCtx,
				kubectlCLI: kubectl.NewFromRunContext(runCtx),
				builds:     test.built,
			}
			err := loadingFunc(r, test)

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckNoError(err)
			}
		})
	}
}
