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

package local

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
)

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) { return nil, nil }

func TestLocalRun(t *testing.T) {
	defer func(h docker.AuthConfigHelper) { docker.DefaultAuthHelper = h }(docker.DefaultAuthHelper)
	docker.DefaultAuthHelper = testAuthHelper{}

	var tests = []struct {
		description      string
		api              testutil.FakeAPIClient
		tags             tag.ImageTags
		artifacts        []*latest.Artifact
		expected         []build.Artifact
		expectedWarnings []string
		expectedPushed   []string
		pushImages       bool
		shouldErr        bool
	}{
		{
			description: "single build (local)",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				}},
			},
			tags:       tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			api:        testutil.FakeAPIClient{},
			pushImages: false,
			expected: []build.Artifact{{
				ImageName: "gcr.io/test/image",
				Tag:       "gcr.io/test/image:1",
			}},
		},
		{
			description: "error getting image digest",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				}},
			},
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			api: testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			shouldErr: true,
		},
		{
			description: "single build (remote)",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				}},
			},
			tags:       tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			api:        testutil.FakeAPIClient{},
			pushImages: true,
			expected: []build.Artifact{{
				ImageName: "gcr.io/test/image",
				Tag:       "gcr.io/test/image:tag@sha256:7368613235363a31e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			}},
			expectedPushed: []string{"sha256:7368613235363a31e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		},
		{
			description: "error build",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				}},
			},
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			api: testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "dont push on build error",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{},
				}},
			},
			tags:       tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			pushImages: true,
			api: testutil.FakeAPIClient{
				ErrImageBuild: true,
			},
			shouldErr: true,
		},
		{
			description: "unkown artifact type",
			artifacts:   []*latest.Artifact{{}},
			shouldErr:   true,
		},
		{
			description: "cache-from images already pulled",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1", "pull2"},
					},
				}},
			},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{
					"pull1": "imageID1",
					"pull2": "imageID2",
				},
			},
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			expected: []build.Artifact{{
				ImageName: "gcr.io/test/image",
				Tag:       "gcr.io/test/image:1",
			}},
		},
		{
			description: "pull cache-from images",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1", "pull2"},
					},
				}},
			},
			api: testutil.FakeAPIClient{
				TagToImageID: map[string]string{"pull1": "imageid", "pull2": "anotherimageid"},
			},
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			expected: []build.Artifact{{
				ImageName: "gcr.io/test/image",
				Tag:       "gcr.io/test/image:1",
			}},
		},
		{
			description: "ignore cache-from pull error",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull1"},
					},
				}},
			},
			api: testutil.FakeAPIClient{
				ErrImagePull: true,
				TagToImageID: map[string]string{"pull1": ""},
			},
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			expected: []build.Artifact{{
				ImageName: "gcr.io/test/image",
				Tag:       "gcr.io/test/image:1",
			}},
			expectedWarnings: []string{"Cache-From image couldn't be pulled: pull1\n"},
		},
		{
			description: "error checking cache-from image",
			artifacts: []*latest.Artifact{{
				ImageName: "gcr.io/test/image",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						CacheFrom: []string{"pull"},
					},
				}},
			},
			api: testutil.FakeAPIClient{
				ErrImageInspect: true,
			},
			tags:      tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(w warnings.Warner) { warnings.Printf = w }(warnings.Printf)
			fakeWarner := &warnings.Collect{}
			warnings.Printf = fakeWarner.Warnf
			cfg := latest.BuildConfig{
				BuildType: latest.BuildType{
					LocalBuild: &latest.LocalBuild{},
				},
			}
			runContext := &runcontext.RunContext{
				Cfg: &latest.Pipeline{
					Build: cfg,
				},
				Opts: &config.SkaffoldOptions{},
			}
			event.InitializeState(runContext)
			l := Builder{
				cfg:         &latest.LocalBuild{},
				localDocker: docker.NewLocalDaemon(&test.api, nil, false, map[string]bool{}),
				pushImages:  test.pushImages,
				runCtx:      *runContext,
			}

			res, err := l.Build(context.Background(), ioutil.Discard, test.tags, test.artifacts)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, res)
			testutil.CheckDeepEqual(t, test.expectedWarnings, fakeWarner.Warnings)
			testutil.CheckDeepEqual(t, test.expectedPushed, test.api.Pushed)
		})
	}
}
