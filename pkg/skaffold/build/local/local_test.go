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
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	"k8s.io/client-go/tools/clientcmd/api"
)

type FakeTagger struct {
	Out string
	Err error
}

func (f *FakeTagger) GenerateFullyQualifiedImageName(workingDir string, tagOpts *tag.Options) (string, error) {
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

	restore := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: "cluster1"})
	defer restore()

	tmp, cleanup := testutil.TempDir(t)
	defer cleanup()

	ioutil.WriteFile(filepath.Join(tmp, "Dockerfile"), []byte(""), 0640)

	var tests = []struct {
		description  string
		config       *v1alpha2.LocalBuild
		out          io.Writer
		api          docker.APIClient
		tagger       tag.Tagger
		artifacts    []*v1alpha2.Artifact
		expected     []build.Artifact
		localCluster bool
		shouldErr    bool
	}{
		{
			description: "single build",
			out:         ioutil.Discard,
			config: &v1alpha2.LocalBuild{
				SkipPush: util.BoolPtr(false),
			},
			artifacts: []*v1alpha2.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Workspace: tmp,
					ArtifactType: v1alpha2.ArtifactType{
						DockerArtifact: &v1alpha2.DockerArtifact{},
					},
				},
			},
			tagger: &tag.ChecksumTagger{},
			api:    testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expected: []build.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:imageid",
				},
			},
		},
		{
			description: "subset build",
			out:         ioutil.Discard,
			config: &v1alpha2.LocalBuild{
				SkipPush: util.BoolPtr(true),
			},
			tagger: &tag.ChecksumTagger{},
			artifacts: []*v1alpha2.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Workspace: tmp,
					ArtifactType: v1alpha2.ArtifactType{
						DockerArtifact: &v1alpha2.DockerArtifact{},
					},
				},
			},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			expected: []build.Artifact{
				{
					ImageName: "gcr.io/test/image",
					Tag:       "gcr.io/test/image:imageid",
				},
			},
		},
		{
			description:  "local cluster bad writer",
			out:          &testutil.BadWriter{},
			config:       &v1alpha2.LocalBuild{},
			shouldErr:    true,
			localCluster: true,
		},
		{
			description: "error image build",
			out:         ioutil.Discard,
			artifacts:   []*v1alpha2.Artifact{{}},
			tagger:      &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageBuild: true,
			}),
			shouldErr: true,
		},
		{
			description: "error image tag",
			out:         ioutil.Discard,
			artifacts:   []*v1alpha2.Artifact{{}},
			tagger:      &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageTag: true,
			}),
			shouldErr: true,
		},
		{
			description: "bad writer",
			out:         &testutil.BadWriter{},
			artifacts:   []*v1alpha2.Artifact{{}},
			tagger:      &tag.ChecksumTagger{},
			api:         testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			shouldErr:   true,
		},
		{
			description: "error image inspect",
			out:         &testutil.BadWriter{},
			artifacts:   []*v1alpha2.Artifact{{}},
			tagger:      &tag.ChecksumTagger{},
			api: testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{
				ErrImageInspect: true,
			}),
			shouldErr: true,
		},
		{
			description: "error tagger",
			out:         ioutil.Discard,
			artifacts:   []*v1alpha2.Artifact{{}},
			tagger:      &FakeTagger{Err: fmt.Errorf("")},
			api:         testutil.NewFakeImageAPIClient(map[string]string{}, &testutil.FakeImageAPIOptions{}),
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := Builder{
				api:          test.api,
				localCluster: test.localCluster,
			}

			res, err := l.Build(context.Background(), color.NewWriter(test.out, color.None), test.tagger, test.artifacts)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
		})
	}
}
