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
	"fmt"
	"io/ioutil"
	"strings"
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
	"github.com/pkg/errors"
)

type testAuthHelper struct{}

type testResult struct {
	buildResult build.Result
	shouldErr   bool
}

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
		expectedResults  []testResult
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						Result: build.Artifact{
							ImageName: "gcr.io/test/image",
							Tag:       "gcr.io/test/image:1",
						},
					},
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						Error: errors.New("building [gcr.io/test/image]"),
					},
					shouldErr: true,
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						Result: build.Artifact{
							ImageName: "gcr.io/test/image",
							Tag:       "gcr.io/test/image:tag@sha256:7368613235363a31e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						},
					},
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						Error: errors.New("building [gcr.io/test/image]"),
					},
					shouldErr: true,
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{},
							},
						},
						Error: errors.New("building [gcr.io/test/image]"),
					},
					shouldErr: true,
				},
			},
		},
		{
			description: "unknown artifact type",
			artifacts:   []*latest.Artifact{{}},
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{},
						Error:  errors.New("unable to find tag for image "),
					},
					shouldErr: true,
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{
									CacheFrom: []string{"pull1", "pull2"},
								},
							},
						},
						Result: build.Artifact{
							ImageName: "gcr.io/test/image",
							Tag:       "gcr.io/test/image:1",
						},
					},
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{
									CacheFrom: []string{"pull1", "pull2"},
								},
							},
						},
						Result: build.Artifact{
							ImageName: "gcr.io/test/image",
							Tag:       "gcr.io/test/image:1",
						},
					},
				},
			},
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
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{
									CacheFrom: []string{"pull1"},
								},
							},
						},
						Result: build.Artifact{
							ImageName: "gcr.io/test/image",
							Tag:       "gcr.io/test/image:1",
						},
					},
				},
			},
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
			tags: tag.ImageTags(map[string]string{"gcr.io/test/image": "gcr.io/test/image:tag"}),
			expectedResults: []testResult{
				{
					buildResult: build.Result{
						Target: latest.Artifact{
							ImageName: "gcr.io/test/image",
							ArtifactType: latest.ArtifactType{
								DockerArtifact: &latest.DockerArtifact{
									CacheFrom: []string{"pull"},
								},
							},
						},
						Error: fmt.Errorf("building [gcr.io/test/image]"),
					},
					shouldErr: true,
				},
			},
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
			event.InitializeState(&runcontext.RunContext{
				Cfg: &latest.Pipeline{
					Build: cfg,
				},
				Opts: &config.SkaffoldOptions{},
			})
			l := Builder{
				cfg:         &latest.LocalBuild{},
				localDocker: docker.NewLocalDaemon(&test.api, nil, false, map[string]bool{}),
				pushImages:  test.pushImages,
			}

			res, err := l.Build(context.Background(), ioutil.Discard, test.tags, test.artifacts)

			// none of these tests should fail before the builds start. assert that err is nil here first.
			// testutil.CheckError(t, false, err)
			testutil.CheckError(t, test.shouldErr, err)

			// build results are returned in a list, of which we can't guarantee order.
			// loop through the expected results, and find the matching build result by target artifact.
			found := false
			for _, testRes := range test.expectedResults {
				for _, buildRes := range res {
					if buildRes.Target.ImageName == testRes.buildResult.Target.ImageName {
						found = true
						// the embedded error in the build result contains a stack trace which we can't reproduce.
						// directly compare the fields of the build result and optional error.
						testutil.CheckError(t, testRes.shouldErr, buildRes.Error)
						if testRes.shouldErr {
							if !strings.Contains(buildRes.Error.Error(), testRes.buildResult.Error.Error()) {
								t.Errorf("build error %s does not match expected error: %s", buildRes.Error.Error(), testRes.buildResult.Error.Error())
							}
							// testutil.CheckDeepEqual(t, testRes.buildResult.Error.Error(), buildRes.Error.Error())
						}
						testutil.CheckDeepEqual(t, testRes.buildResult.Target, buildRes.Target)
						testutil.CheckDeepEqual(t, testRes.buildResult.Result, buildRes.Result)
					}
				}
				if !found {
					t.Errorf("expected result %+v not found in build results", testRes)
				}
				found = false
			}

			testutil.CheckDeepEqual(t, test.expectedWarnings, fakeWarner.Warnings)
			testutil.CheckDeepEqual(t, test.expectedPushed, test.api.Pushed)
		})
	}
}
