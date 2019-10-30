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

package buildpacks

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		description string
		tag         string
		latest      string
		builder     string
		runImage    string
		api         *testutil.FakeAPIClient
		forcePull   bool
		pushImages  bool
		shouldErr   bool
	}{
		{
			description: "success",
			tag:         "img:tag",
			latest:      "img:latest",
			builder:     "my/builder",
			runImage:    "my/run",
			api:         &testutil.FakeAPIClient{},
		},
		{
			description: "invalid ref",
			tag:         "in valid ref",
			api:         &testutil.FakeAPIClient{},
			shouldErr:   true,
		},
		{
			description: "force pull",
			tag:         "img:tag",
			latest:      "img:latest",
			builder:     "my/builder",
			runImage:    "my/run",
			forcePull:   true,
			api:         &testutil.FakeAPIClient{},
		},
		{
			description: "force pull error",
			tag:         "img:tag",
			latest:      "img:latest",
			builder:     "my/builder",
			runImage:    "my/run",
			forcePull:   true,
			api: &testutil.FakeAPIClient{
				ErrImagePull: true,
			},
			shouldErr: true,
		},
		{
			description: "push error",
			tag:         "img:tag",
			latest:      "img:latest",
			builder:     "my/builder",
			runImage:    "my/run",
			pushImages:  true,
			api: &testutil.FakeAPIClient{
				ErrImagePush: true,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("file").Chdir()
			test.api.
				Add(test.builder, "builderImageID").
				Add(test.runImage, "runImageID").
				Add(test.latest, "builtImageID")
			localDocker := docker.NewLocalDaemon(test.api, nil, false, nil)

			builder := NewArtifactBuilder(localDocker, test.pushImages)
			_, err := builder.Build(context.Background(), ioutil.Discard, &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Builder:   test.builder,
						RunImage:  test.runImage,
						ForcePull: test.forcePull,
						Dependencies: &latest.BuildpackDependencies{
							Paths: []string{"."},
						},
					},
				},
			}, test.tag)

			t.CheckError(test.shouldErr, err)
		})
	}
}
