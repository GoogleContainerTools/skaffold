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
	"fmt"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
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

var testImage1 = &config.Artifact{
	ImageName: "gcr.io/test/image",
	Workspace: ".",
}

var testImage2 = &config.Artifact{
	ImageName: "gcr.io/test/image2",
	Workspace: ".",
}

func TestLocalRun(t *testing.T) {
	auth := docker.DefaultAuthHelper
	defer func() { docker.DefaultAuthHelper = auth }()
	docker.DefaultAuthHelper = testAuthHelper{}

	// Set a bad KUBECONFIG path so we don't parse a real one that happens to be
	// present on the host
	unsetEnvs := testutil.SetEnvs(t, map[string]string{"KUBECONFIG": "badpath"})
	defer unsetEnvs(t)
	var tests = []struct {
		description  string
		config       *config.BuildConfig
		out          io.Writer
		api          docker.DockerAPIClient
		tagger       tag.Tagger
		localCluster bool
		artifacts    []*config.Artifact

		expectedBuild *BuildResult
		shouldErr     bool
	}{
		{
			description: "single build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					testImage1,
				},
				BuildType: config.BuildType{
					LocalBuild: &config.LocalBuild{
						SkipPush: util.BoolPtr(false),
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api:    testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expectedBuild: &BuildResult{
				[]Build{
					{
						ImageName: "gcr.io/test/image",
						Tag:       "gcr.io/test/image:imageid",
						Artifact:  testImage1,
					},
				},
			},
		},
		{
			description: "subset build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "gcr.io/test/image",
						Workspace: ".",
					},
					{
						ImageName: "gcr.io/test/image2",
						Workspace: ".",
					},
				},
				BuildType: config.BuildType{
					LocalBuild: &config.LocalBuild{
						SkipPush: util.BoolPtr(true),
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			artifacts: []*config.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Workspace: ".",
				},
			},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expectedBuild: &BuildResult{
				[]Build{
					{
						ImageName: "gcr.io/test/image",
						Tag:       "gcr.io/test/image:imageid",
						Artifact:  testImage1,
					},
				},
			},
		},
		{
			description:  "local cluster bad writer",
			out:          &testutil.BadWriter{},
			config:       &config.BuildConfig{},
			shouldErr:    true,
			localCluster: true,
		},
		{
			description: "error image build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
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
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
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
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
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
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
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
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
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
			res, err := l.Build(test.out, test.tagger, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedBuild, res)
		})
	}
}
