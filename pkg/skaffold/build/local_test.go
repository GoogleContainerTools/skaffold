/*
Copyright 2018 Google LLC

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

package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
)

type FakeTagger struct {
	Out string
	Err error
}

func (f *FakeTagger) GenerateFullyQualifiedImageName(workingDir string, tagOpts *tag.TagOptions) (string, error) {
	return f.Out, f.Err
}

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) { return nil, nil }

var testImage1 = &v1alpha2.Artifact{
	ImageName: "gcr.io/test/image",
	Workspace: "../../../testdata/docker",
	ArtifactType: v1alpha2.ArtifactType{
		DockerArtifact: &v1alpha2.DockerArtifact{},
	},
}

var testImage2 = &v1alpha2.Artifact{
	ImageName: "gcr.io/test/image2",
	Workspace: "../../../testdata/docker",
	ArtifactType: v1alpha2.ArtifactType{
		DockerArtifact: &v1alpha2.DockerArtifact{},
	},
}

func TestLocalRun(t *testing.T) {
	defer func(h docker.AuthConfigHelper) { docker.DefaultAuthHelper = h }(docker.DefaultAuthHelper)
	docker.DefaultAuthHelper = testAuthHelper{}

	// Set a bad KUBECONFIG path so we don't parse a real one that happens to be
	// present on the host
	unsetEnvs := testutil.SetEnvs(t, map[string]string{"KUBECONFIG": "badpath"})
	defer unsetEnvs(t)
	var tests = []struct {
		description  string
		config       *v1alpha2.BuildConfig
		out          io.Writer
		api          docker.DockerAPIClient
		tagger       tag.Tagger
		localCluster bool
		artifacts    []*v1alpha2.Artifact
		expected     []Build
		shouldErr    bool
	}{
		{
			description: "single build",
			out:         &bytes.Buffer{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					testImage1,
				},
				BuildType: v1alpha2.BuildType{
					LocalBuild: &v1alpha2.LocalBuild{
						SkipPush: util.BoolPtr(false),
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api:    testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expected: []Build{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:imageid",
					Artifact:  testImage1,
				},
			},
		},
		{
			description: "subset build",
			out:         &bytes.Buffer{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "gcr.io/test/image",
						Workspace: "../../../testdata/docker",
						ArtifactType: v1alpha2.ArtifactType{
							DockerArtifact: &v1alpha2.DockerArtifact{},
						},
					},
					{
						ImageName: "gcr.io/test/image2",
						Workspace: "../../../testdata/docker",
						ArtifactType: v1alpha2.ArtifactType{
							DockerArtifact: &v1alpha2.DockerArtifact{},
						},
					},
				},
				BuildType: v1alpha2.BuildType{
					LocalBuild: &v1alpha2.LocalBuild{
						SkipPush: util.BoolPtr(true),
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			artifacts: []*v1alpha2.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Workspace: "../../../testdata/docker",
					ArtifactType: v1alpha2.ArtifactType{
						DockerArtifact: &v1alpha2.DockerArtifact{},
					},
				},
			},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expected: []Build{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:imageid",
					Artifact:  testImage1,
				},
			},
		},
		{
			description:  "local cluster bad writer",
			out:          &testutil.BadWriter{},
			config:       &v1alpha2.BuildConfig{},
			shouldErr:    true,
			localCluster: true,
		},
		{
			description: "error image build",
			out:         &bytes.Buffer{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageBuild: true,
			}),
			shouldErr: true,
		},
		{
			description: "error image tag",
			out:         &bytes.Buffer{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageTag: true,
			}),
			shouldErr: true,
		},
		{
			description: "bad writer",
			out:         &testutil.BadWriter{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:    &tag.ChecksumTagger{},
			api:       testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			shouldErr: true,
		},
		{
			description: "error image list",
			out:         &testutil.BadWriter{},
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageList: true,
			}),
			shouldErr: true,
		},
		{
			description: "error tagger",
			config: &v1alpha2.BuildConfig{
				Artifacts: []*v1alpha2.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:    &FakeTagger{Err: fmt.Errorf("")},
			api:       testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := LocalBuilder{
				BuildConfig:  test.config,
				api:          test.api,
				localCluster: test.localCluster,
			}
			if test.artifacts == nil {
				test.artifacts = test.config.Artifacts
			}

			res, err := l.Build(context.Background(), test.out, test.tagger, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
		})
	}
}
