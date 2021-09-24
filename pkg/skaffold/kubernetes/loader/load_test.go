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

package loader

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type ImageLoadingTest = struct {
	description   string
	cluster       string
	deployed      []graph.Artifact
	commands      util.Command
	shouldErr     bool
	expectedError string
}

func TestLoadImagesInKindNodes(t *testing.T) {
	tests := []ImageLoadingTest{
		{
			description: "load image",
			cluster:     "kind",
			deployed:    []graph.Artifact{{Tag: "tag1"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("kind load docker-image --name kind tag1", "output: image loaded"),
		},
		{
			description: "load missing image",
			cluster:     "other-kind",
			deployed:    []graph.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "docker.io/library/tag1").
				AndRunOut("kind load docker-image --name other-kind tag2", "output: image loaded"),
		},
		{
			description: "no new images",
			cluster:     "kind",
			deployed:    []graph.Artifact{{Tag: "tag0"}, {Tag: "docker.io/library/tag1"}, {Tag: "docker.io/tag2"}, {Tag: "gcr.io/test/tag3"}, {Tag: "someregistry.com/tag4"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "docker.io/library/tag0 docker.io/library/tag1 docker.io/library/tag2 gcr.io/test/tag3 someregistry.com/tag4"),
		},
		{
			description: "inspect error",
			deployed:    []graph.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOutErr("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to inspect",
		},
		{
			description: "load error",
			cluster:     "kind",
			deployed:    []graph.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOutErr("kind load docker-image --name kind tag", "output: error!", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "output: error!",
		},
		{
			description: "no artifact",
			deployed:    []graph.Artifact{},
		},
	}

	runImageLoadingTests(t, tests, func(i *ImageLoader, test ImageLoadingTest) error {
		return i.loadImagesInKindNodes(context.Background(), ioutil.Discard, test.cluster, test.deployed)
	})
}

func TestLoadImagesInK3dNodes(t *testing.T) {
	tests := []ImageLoadingTest{
		{
			description: "load image",
			cluster:     "k3d",
			deployed:    []graph.Artifact{{Tag: "tag1"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOut("k3d image import --cluster k3d tag1", "output: image loaded"),
		},
		{
			description: "load missing image",
			cluster:     "other-k3d",
			deployed:    []graph.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "docker.io/library/tag1").
				AndRunOut("k3d image import --cluster other-k3d tag2", "output: image loaded"),
		},
		{
			description: "no new images",
			cluster:     "k3d",
			deployed:    []graph.Artifact{{Tag: "tag0"}, {Tag: "docker.io/library/tag1"}, {Tag: "docker.io/tag2"}, {Tag: "gcr.io/test/tag3"}, {Tag: "someregistry.com/tag4"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "docker.io/library/tag0 docker.io/library/tag1 docker.io/library/tag2 gcr.io/test/tag3 someregistry.com/tag4"),
		},
		{
			description: "inspect error",
			deployed:    []graph.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOutErr("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to inspect",
		},
		{
			description: "load error",
			cluster:     "k3d",
			deployed:    []graph.Artifact{{Tag: "tag"}},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace namespace get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				AndRunOutErr("k3d image import --cluster k3d tag", "output: error!", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "output: error!",
		},
		{
			description: "no artifact",
			deployed:    []graph.Artifact{},
		},
	}

	runImageLoadingTests(t, tests, func(i *ImageLoader, test ImageLoadingTest) error {
		return i.loadImagesInK3dNodes(context.Background(), ioutil.Discard, test.cluster, test.deployed)
	})
}

func runImageLoadingTests(t *testing.T, tests []ImageLoadingTest, loadingFunc func(i *ImageLoader, test ImageLoadingTest) error) {
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			runCtx := &runcontext.RunContext{
				Opts: config.SkaffoldOptions{
					Namespace: "namespace",
				},
				KubeContext: "kubecontext",
			}

			i := NewImageLoader(runCtx.KubeContext, kubectl.NewCLI(runCtx, ""))
			err := loadingFunc(i, test)

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckNoError(err)
			}
		})
	}
}

func TestImagesToLoad(t *testing.T) {
	tests := []struct {
		name           string
		localImages    []graph.Artifact
		deployerImages []graph.Artifact
		builtImages    []graph.Artifact
		expectedImages []graph.Artifact
	}{
		{
			name:           "single image marked as local",
			localImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			deployerImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			expectedImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
		},
		{
			name:           "single image, but not marked as local",
			localImages:    []graph.Artifact{},
			deployerImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			expectedImages: nil,
		},
		{
			name:           "single image, marked as local but not found in deployer's manifests",
			localImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			deployerImages: []graph.Artifact{},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}},
			expectedImages: nil,
		},
		{
			name:           "two images marked as local",
			localImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			deployerImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			expectedImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
		},
		{
			name:           "two images, one marked as local and one not",
			localImages:    []graph.Artifact{{ImageName: "image2", Tag: "bar"}},
			deployerImages: []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			expectedImages: []graph.Artifact{{ImageName: "image2", Tag: "bar"}},
		},
		{
			name:           "two images, marked as local but only one found from the deployer's manifests",
			localImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			deployerImages: []graph.Artifact{{ImageName: "image2", Tag: "bar"}},
			builtImages:    []graph.Artifact{{ImageName: "image1", Tag: "foo"}, {ImageName: "image2", Tag: "bar"}},
			expectedImages: []graph.Artifact{{ImageName: "image2", Tag: "bar"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.CheckDeepEqual(test.expectedImages, imagesToLoad(test.localImages, test.deployerImages, test.builtImages))
		})
	}
}
