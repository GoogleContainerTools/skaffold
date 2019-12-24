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
	"io"
	"io/ioutil"
	"testing"

	"github.com/buildpacks/pack"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type fakePack struct {
	Opts pack.BuildOptions
}

func (f *fakePack) runPack(_ context.Context, _ io.Writer, opts pack.BuildOptions) error {
	f.Opts = opts
	return nil
}

func TestBuild(t *testing.T) {
	tests := []struct {
		description     string
		artifact        *latest.BuildpackArtifact
		tag             string
		api             *testutil.FakeAPIClient
		pushImages      bool
		shouldErr       bool
		expectedOptions *pack.BuildOptions
	}{
		{
			description: "success",
			artifact: &latest.BuildpackArtifact{
				Builder:      "my/builder",
				RunImage:     "my/run",
				Dependencies: defaultBuildpackDependencies(),
			},
			tag: "img:tag",
			api: &testutil.FakeAPIClient{},
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "my/builder",
				RunImage: "my/run",
				Env:      map[string]string{},
				Image:    "img:latest",
				NoPull:   true,
			},
		},
		{
			description: "invalid ref",
			artifact: &latest.BuildpackArtifact{
				Builder:      "my/builder",
				RunImage:     "my/run",
				Dependencies: defaultBuildpackDependencies(),
			},
			tag:       "in valid ref",
			api:       &testutil.FakeAPIClient{},
			shouldErr: true,
		},
		{
			description: "force pull",
			artifact: &latest.BuildpackArtifact{
				Builder:      "my/builder",
				RunImage:     "my/run",
				ForcePull:    true,
				Dependencies: defaultBuildpackDependencies(),
			},
			tag: "img:tag",
			api: &testutil.FakeAPIClient{},
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "my/builder",
				RunImage: "my/run",
				Env:      map[string]string{},
				Image:    "img:latest",
				NoPull:   false,
			},
		},
		{
			description: "push error",
			artifact: &latest.BuildpackArtifact{
				Builder:      "my/builder",
				RunImage:     "my/run",
				Dependencies: defaultBuildpackDependencies(),
			},
			tag:        "img:tag",
			pushImages: true,
			api: &testutil.FakeAPIClient{
				ErrImagePush: true,
			},
			shouldErr: true,
		},
		{
			description: "invalid env",
			artifact: &latest.BuildpackArtifact{
				Builder:      "my/builder",
				RunImage:     "my/run",
				Env:          []string{"INVALID"},
				Dependencies: defaultBuildpackDependencies(),
			},
			tag:       "img:tag",
			api:       &testutil.FakeAPIClient{},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("file").Chdir()
			pack := &fakePack{}
			t.Override(&runPackFunc, pack.runPack)

			test.api.
				Add(test.artifact.Builder, "builderImageID").
				Add(test.artifact.RunImage, "runImageID").
				Add("img:latest", "builtImageID")
			localDocker := docker.NewLocalDaemon(test.api, nil, false, nil)

			builder := NewArtifactBuilder(localDocker, test.pushImages)
			_, err := builder.Build(context.Background(), ioutil.Discard, &latest.Artifact{
				Workspace: ".",
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: test.artifact,
				},
			}, test.tag)

			t.CheckError(test.shouldErr, err)
			if test.expectedOptions != nil {
				t.CheckDeepEqual(*test.expectedOptions, pack.Opts)
			}
		})
	}
}

func defaultBuildpackDependencies() *latest.BuildpackDependencies {
	return &latest.BuildpackDependencies{
		Paths: []string{"."},
	}
}
