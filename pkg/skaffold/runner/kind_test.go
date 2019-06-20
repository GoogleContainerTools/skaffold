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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestLoadImagesInKindNodes(t *testing.T) {
	var tests = []struct {
		description   string
		built         []build.Artifact
		deployed      []build.Artifact
		command       util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "load image",
			built:       []build.Artifact{{Tag: "tag1"}},
			deployed:    []build.Artifact{{Tag: "tag1"}},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				WithRun("kind load docker-image tag1"),
		},
		{
			description: "load missing image",
			built:       []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			deployed:    []build.Artifact{{Tag: "tag1"}, {Tag: "tag2"}},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "tag1").
				WithRun("kind load docker-image tag2"),
		},
		{
			description: "inspect error",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			command: testutil.NewFakeCmd(t).
				WithRunOutErr("kubectl get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to inspect",
		},
		{
			description: "load error",
			built:       []build.Artifact{{Tag: "tag"}},
			deployed:    []build.Artifact{{Tag: "tag"}},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				WithRunErr("kind load docker-image tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "unable to load",
		},
		{
			description: "ignore image that's not built",
			built:       []build.Artifact{{Tag: "built"}},
			deployed:    []build.Artifact{{Tag: "built"}, {Tag: "busybox"}},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl get nodes -ojsonpath='{@.items[*].status.images[*].names[*]}'", "").
				WithRun("kind load docker-image built"),
		},
		{
			description: "no artifact",
			deployed:    []build.Artifact{},
			command:     testutil.NewFakeCmd(t),
		},
		{
			description: "no built artifact",
			built:       []build.Artifact{},
			deployed:    []build.Artifact{{Tag: "busybox"}},
			command:     testutil.NewFakeCmd(t),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)

			r := &SkaffoldRunner{
				builds: test.built,
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
