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

package gcb

import (
	"testing"

	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerBuildSpec(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.Artifact
		expected    cloudbuild.Build
		shouldErr   bool
	}{
		{
			description: "normal docker build",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
						BuildArgs: map[string]*string{
							"arg1": util.StringPtr("value1"),
							"arg2": nil,
						},
					},
				},
			},
			expected: cloudbuild.Build{
				LogsBucket: "bucket",
				Source: &cloudbuild.Source{
					StorageSource: &cloudbuild.StorageSource{
						Bucket: "bucket",
						Object: "object",
					},
				},
				Steps: []*cloudbuild.BuildStep{{
					Name: "docker/docker",
					Args: []string{"build", "--tag", "nginx", "-f", "Dockerfile", "--build-arg", "arg1=value1", "--build-arg", "arg2", "."},
				}},
				Images: []string{"nginx"},
				Options: &cloudbuild.BuildOptions{
					DiskSizeGb:  100,
					MachineType: "n1-standard-1",
				},
				Timeout: "10m",
			},
		},
		{
			description: "docker build with artifact dependencies",
			artifact: &latest.Artifact{
				ImageName: "img1",
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
						BuildArgs: map[string]*string{
							"arg1": util.StringPtr("value1"),
							"arg2": nil,
						},
					},
				},
				Dependencies: []*latest.ArtifactDependency{{ImageName: "img2", Alias: "IMG2"}, {ImageName: "img3", Alias: "IMG3"}},
			},
			expected: cloudbuild.Build{
				LogsBucket: "bucket",
				Source: &cloudbuild.Source{
					StorageSource: &cloudbuild.StorageSource{
						Bucket: "bucket",
						Object: "object",
					},
				},
				Steps: []*cloudbuild.BuildStep{{
					Name: "docker/docker",
					Args: []string{"build", "--tag", "nginx", "-f", "Dockerfile", "--build-arg", "IMG2=img2:tag", "--build-arg", "IMG3=img3:tag", "--build-arg", "arg1=value1", "--build-arg", "arg2", "."},
				}},
				Images: []string{"nginx"},
				Options: &cloudbuild.BuildOptions{
					DiskSizeGb:  100,
					MachineType: "n1-standard-1",
				},
				Timeout: "10m",
			},
		},
		{
			description: "buildkit `secret` option not supported in GCB",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
						Secret: &latest.DockerSecret{
							ID: "secret",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "buildkit `ssh` option not supported in GCB",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: "Dockerfile",
						SSH:            "default",
					},
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, extra map[string]*string) (map[string]*string, error) {
				m := make(map[string]*string)
				for k, v := range args {
					m[k] = v
				}
				for k, v := range extra {
					m[k] = v
				}
				return m, nil
			})
			builder := NewBuilder(&mockConfig{}, &latest.GoogleCloudBuild{
				DockerImage: "docker/docker",
				DiskSizeGb:  100,
				MachineType: "n1-standard-1",
				Timeout:     "10m",
			})
			store := mockArtifactStore{
				"img2": "img2:tag",
				"img3": "img3:tag",
			}
			builder.ArtifactStore(store)
			desc, err := builder.buildSpec(test.artifact, "nginx", "bucket", "object")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, desc)
		})
	}
}

func TestPullCacheFrom(t *testing.T) {
	testutil.Run(t, "TestPullCacheFrom", func(t *testutil.T) {
		t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
			return args, nil
		})
		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: "Dockerfile",
					CacheFrom:      []string{"from/image1", "from/image2"},
				},
			},
		}
		builder := NewBuilder(&mockConfig{}, &latest.GoogleCloudBuild{
			DockerImage: "docker/docker",
		})
		desc, err := builder.dockerBuildSpec(artifact, "nginx2")

		expected := []*cloudbuild.BuildStep{{
			Name:       "docker/docker",
			Entrypoint: "sh",
			Args:       []string{"-c", "docker pull from/image1 || true"},
		}, {
			Name:       "docker/docker",
			Entrypoint: "sh",
			Args:       []string{"-c", "docker pull from/image2 || true"},
		}, {
			Name: "docker/docker",
			Args: []string{"build", "--tag", "nginx2", "-f", "Dockerfile", "--cache-from", "from/image1", "--cache-from", "from/image2", "."},
		}}

		t.CheckErrorAndDeepEqual(false, err, expected, desc.Steps)
	})
}
