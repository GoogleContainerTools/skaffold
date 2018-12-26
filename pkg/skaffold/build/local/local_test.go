/*
Copyright 2018 The Skaffold Authors

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

package local

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
)

type FakeTagger struct {
	Out string
	Err error
}

func (f *FakeTagger) GenerateFullyQualifiedImageName(workingDir string, tagOpts tag.Options) (string, error) {
	return f.Out, f.Err
}

func (f *FakeTagger) Labels() map[string]string {
	return map[string]string{}
}

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) { return nil, nil }

func TestLocalRun(t *testing.T) {
	defer func(h docker.AuthConfigHelper) { docker.DefaultAuthHelper = h }(docker.DefaultAuthHelper)
	docker.DefaultAuthHelper = testAuthHelper{}

	var tests = []struct {
		description  string
		out          io.Writer
		api          docker.APIClient
		tagger       tag.Tagger
		artifacts    []*latest.Artifact
		expected     []build.Artifact
		localCluster bool
		shouldErr    bool
	}{
		{
			description: "single build",
			out:         ioutil.Discard,
			artifacts: []*latest.Artifact{
				{
					ImageName: "gcr.io/test/image",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{},
					},
				},
			},
			tagger: &FakeTagger{Out: "gcr.io/test/image:tag"},
			api:    &testutil.FakeAPIClient{},
			expected: []build.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:tag",
				},
			},
		},
		{
			description: "subset build",
			out:         ioutil.Discard,
			tagger:      &FakeTagger{Out: "gcr.io/test/image:tag"},
			artifacts: []*latest.Artifact{
				{
					ImageName: "gcr.io/test/image",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{},
					},
				},
			},
			api: &testutil.FakeAPIClient{},
			expected: []build.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:tag",
				},
			},
		},
		{
			description:  "local cluster bad writer",
			out:          &testutil.BadWriter{},
			shouldErr:    true,
			localCluster: true,
		},
		{
			description: "error image build",
			out:         ioutil.Discard,
			artifacts:   []*latest.Artifact{{}},
			api: &testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "error image tag",
			out:         ioutil.Discard,
			artifacts:   []*latest.Artifact{{}},
			api: &testutil.FakeAPIClient{
				ErrImageTag: true,
			},
			shouldErr: true,
		},
		{
			description: "bad writer",
			out:         &testutil.BadWriter{},
			artifacts:   []*latest.Artifact{{}},
			api:         &testutil.FakeAPIClient{},
			shouldErr:   true,
		},
		{
			description: "error image inspect",
			out:         &testutil.BadWriter{},
			artifacts:   []*latest.Artifact{{}},
			api: &testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			shouldErr: true,
		},
		{
			description: "error tagger",
			out:         ioutil.Discard,
			artifacts:   []*latest.Artifact{{}},
			tagger:      &FakeTagger{Err: fmt.Errorf("")},
			api:         &testutil.FakeAPIClient{},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := Builder{
				cfg:          &latest.LocalBuild{},
				api:          test.api,
				localCluster: test.localCluster,
			}

			res, err := l.Build(context.Background(), test.out, test.tagger, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
		})
	}
}
