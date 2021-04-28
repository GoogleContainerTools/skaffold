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

package validation

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	cfgWithErrors = &latest_v1.SkaffoldConfig{
		Pipeline: latest_v1.Pipeline{
			Build: latest_v1.BuildConfig{
				Artifacts: []*latest_v1.Artifact{
					{
						ArtifactType: latest_v1.ArtifactType{
							DockerArtifact: &latest_v1.DockerArtifact{},
							BazelArtifact:  &latest_v1.BazelArtifact{},
						},
					},
					{
						ArtifactType: latest_v1.ArtifactType{
							BazelArtifact:  &latest_v1.BazelArtifact{},
							KanikoArtifact: &latest_v1.KanikoArtifact{},
						},
					},
				},
			},
			Deploy: latest_v1.DeployConfig{
				DeployType: latest_v1.DeployType{
					HelmDeploy:    &latest_v1.HelmDeploy{},
					KubectlDeploy: &latest_v1.KubectlDeploy{},
				},
			},
		},
	}
)

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest_v1.SkaffoldConfig
		shouldErr   bool
	}{
		{
			description: "config with errors",
			cfg:         cfgWithErrors,
			shouldErr:   true,
		},
		{
			description: "empty config",
			cfg:         &latest_v1.SkaffoldConfig{},
			shouldErr:   true,
		},
		{
			description: "minimal config",
			cfg: &latest_v1.SkaffoldConfig{
				APIVersion: "foo",
				Kind:       "bar",
			},
			shouldErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := Process([]*latest_v1.SkaffoldConfig{test.cfg})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func alwaysErr(_ interface{}) error {
	return fmt.Errorf("always fail")
}

type emptyStruct struct{}
type nestedEmptyStruct struct {
	N emptyStruct
}

func TestVisitStructs(t *testing.T) {
	tests := []struct {
		description  string
		input        interface{}
		expectedErrs int
	}{
		{
			description:  "single struct to validate",
			input:        emptyStruct{},
			expectedErrs: 1,
		},
		{
			description:  "recurse into nested struct",
			input:        nestedEmptyStruct{},
			expectedErrs: 2,
		},
		{
			description: "check all slice items",
			input: struct {
				A []emptyStruct
			}{
				A: []emptyStruct{{}, {}},
			},
			expectedErrs: 3,
		},
		{
			description: "recurse into slices",
			input: struct {
				A []nestedEmptyStruct
			}{
				A: []nestedEmptyStruct{
					{
						N: emptyStruct{},
					},
				},
			},
			expectedErrs: 3,
		},
		{
			description: "recurse into ptr slices",
			input: struct {
				A []*nestedEmptyStruct
			}{
				A: []*nestedEmptyStruct{
					{
						N: emptyStruct{},
					},
				},
			},
			expectedErrs: 3,
		},
		{
			description: "ignore empty slices",
			input: struct {
				A []emptyStruct
			}{},
			expectedErrs: 1,
		},
		{
			description: "ignore nil pointers",
			input: struct {
				A *struct{}
			}{},
			expectedErrs: 1,
		},
		{
			description: "recurse into members",
			input: struct {
				A, B emptyStruct
			}{
				A: emptyStruct{},
				B: emptyStruct{},
			},
			expectedErrs: 3,
		},
		{
			description: "recurse into ptr members",
			input: struct {
				A, B *emptyStruct
			}{
				A: &emptyStruct{},
				B: &emptyStruct{},
			},
			expectedErrs: 3,
		},
		{
			description: "ignore other fields",
			input: struct {
				A emptyStruct
				C int
			}{
				A: emptyStruct{},
				C: 2,
			},
			expectedErrs: 2,
		},
		{
			description: "unexported fields",
			input: struct {
				a emptyStruct
			}{
				a: emptyStruct{},
			},
			expectedErrs: 1,
		},
		{
			description: "exported and unexported fields",
			input: struct {
				a, A, b emptyStruct
			}{
				a: emptyStruct{},
				A: emptyStruct{},
				b: emptyStruct{},
			},
			expectedErrs: 2,
		},
		{
			description: "unexported nil ptr fields",
			input: struct {
				a *emptyStruct
			}{
				a: nil,
			},
			expectedErrs: 1,
		},
		{
			description: "unexported ptr fields",
			input: struct {
				a *emptyStruct
			}{
				a: &emptyStruct{},
			},
			expectedErrs: 1,
		},
		{
			description: "unexported and exported ptr fields",
			input: struct {
				a, A, b *emptyStruct
			}{
				a: &emptyStruct{},
				A: &emptyStruct{},
				b: &emptyStruct{},
			},
			expectedErrs: 2,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := visitStructs(test.input, alwaysErr)

			t.CheckDeepEqual(test.expectedErrs, len(actual))
		})
	}
}

func TestValidateNetworkMode(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest_v1.Artifact
		shouldErr   bool
		env         []string
	}{
		{
			description: "not a docker artifact",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/bazel",
					ArtifactType: latest_v1.ArtifactType{
						BazelArtifact: &latest_v1.BazelArtifact{},
					},
				},
			},
		},
		{
			description: "no networkmode",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/no-network",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{},
					},
				},
			},
		},
		{
			description: "bridge",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/bridge",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Bridge",
						},
					},
				},
			},
		},
		{
			description: "empty container's network stack",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "empty container's network stack in env var",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env:       []string{"CONTAINER="},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack '-not-valid'",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:-not-valid",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack '-not-valid' in env var",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env:       []string{"CONTAINER=-not-valid"},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack 'fussball'",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:fußball",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack 'fussball' in env var",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env:       []string{"CONTAINER=fußball"},
			shouldErr: true,
		},
		{
			description: "container's network stack 'unique'",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:unique",
						},
					},
				},
			},
		},
		{
			description: "container's network stack 'unique' in env var",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env: []string{"CONTAINER=unique"},
		},
		{
			description: "container's network stack 'unique-id.123'",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:unique-id.123",
						},
					},
				},
			},
		},
		{
			description: "container's network stack 'unique-id.123' in env var",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env: []string{"CONTAINER=unique-id.123"},
		},
		{
			description: "none",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/none",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "None",
						},
					},
				},
			},
		},
		{
			description: "host",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/host",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Host",
						},
					},
				},
			},
		},
		{
			description: "invalid networkmode",
			shouldErr:   true,
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/bad",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Bad",
						},
					},
				},
			},
		},
		{
			description: "case insensitive",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/case-insensitive",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "bRiDgE",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })
			t.Override(&util.OSEnviron, func() []string { return test.env })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Build: latest_v1.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

type fakeCommonAPIClient struct {
	client.CommonAPIClient
	expectedResponse []types.Container
}

func (f fakeCommonAPIClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	return f.expectedResponse, nil
}

func TestValidateNetworkModeDockerContainerExists(t *testing.T) {
	tests := []struct {
		description    string
		artifacts      []*latest_v1.Artifact
		clientResponse []types.Container
		shouldErr      bool
		env            []string
	}{
		{
			description: "no running containers",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:foo",
						},
					},
				},
			},
			clientResponse: []types.Container{},
			shouldErr:      true,
		},
		{
			description: "not matching running containers",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:foo",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID:    "not-foo",
					Names: []string{"/bar"},
				},
			},
			shouldErr: true,
		},
		{
			description: "existing running container referenced by id",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:foo",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID: "foo",
				},
			},
		},
		{
			description: "existing running container referenced by first id chars",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:123",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID: "1234567890",
				},
			},
		},
		{
			description: "existing running container referenced by name",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:foo",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID:    "no-foo",
					Names: []string{"/foo"},
				},
			},
		},
		{
			description: "non existing running container referenced by id in envvar",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{ .CONTAINER }}",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID: "non-foo",
				},
			},
			env:       []string{"CONTAINER=foo"},
			shouldErr: true,
		},
		{
			description: "existing running container referenced by id in envvar",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{ .CONTAINER }}",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID: "foo",
				},
			},
			env: []string{"CONTAINER=foo"},
		},
		{
			description: "existing running container referenced by name in envvar",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest_v1.ArtifactType{
						DockerArtifact: &latest_v1.DockerArtifact{
							NetworkMode: "Container:{{ .CONTAINER }}",
						},
					},
				},
			},
			clientResponse: []types.Container{
				{
					ID:    "non-foo",
					Names: []string{"/foo"},
				},
			},
			env: []string{"CONTAINER=foo"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })
			t.Override(&util.OSEnviron, func() []string { return test.env })
			t.Override(&docker.NewAPIClient, func(docker.Config) (docker.LocalDaemon, error) {
				fakeClient := &fakeCommonAPIClient{
					CommonAPIClient: &testutil.FakeAPIClient{
						ErrVersion: true,
					},
					expectedResponse: test.clientResponse,
				}
				return docker.NewLocalDaemon(fakeClient, nil, false, nil), nil
			})

			err := ProcessWithRunContext(&runcontext.RunContext{
				Pipelines: runcontext.NewPipelines([]latest_v1.Pipeline{
					{
						Build: latest_v1.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}),
			})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateSyncRules(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest_v1.Artifact
		shouldErr   bool
	}{
		{
			description: "no artifacts",
			artifacts:   nil,
		},
		{
			description: "no sync rules",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
				Sync:      nil,
			}},
		},
		{
			description: "two good rules",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
				Sync: &latest_v1.Sync{Manual: []*latest_v1.SyncRule{
					{
						Src:  "src/**/*.js",
						Dest: ".",
					},
					{
						Src:   "src/**/*.js",
						Dest:  ".",
						Strip: "src/",
					},
				}},
			}},
		},
		{
			description: "one good one bad rule",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
				Sync: &latest_v1.Sync{Manual: []*latest_v1.SyncRule{
					{
						Src:   "src/**/*.js",
						Dest:  ".",
						Strip: "/src",
					},
					{
						Src:   "src/**/*.py",
						Dest:  ".",
						Strip: "src/",
					},
				}},
			}},
			shouldErr: true,
		},
		{
			description: "two bad rules",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
				Sync: &latest_v1.Sync{Manual: []*latest_v1.SyncRule{
					{
						Dest:  ".",
						Strip: "src",
					},
					{
						Src:   "**/*.js",
						Dest:  ".",
						Strip: "src/",
					},
				}},
			}},
			shouldErr: true,
		},
		{
			description: "stripping part of folder name is valid",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
				Sync: &latest_v1.Sync{
					Manual: []*latest_v1.SyncRule{{
						Src:   "srcsomeother/**/*.js",
						Dest:  ".",
						Strip: "src",
					}},
				},
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Build: latest_v1.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCustomDependencies(t *testing.T) {
	tests := []struct {
		description    string
		dependencies   *latest_v1.CustomDependencies
		expectedErrors int
	}{
		{
			description: "no errors",
			dependencies: &latest_v1.CustomDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
		}, {
			description: "ignore in conjunction with dockerfile",
			dependencies: &latest_v1.CustomDependencies{
				Dockerfile: &latest_v1.DockerfileDependency{
					Path: "some/path",
				},
				Ignore: []string{"ignoreme"},
			},
			expectedErrors: 1,
		}, {
			description: "ignore in conjunction with command",
			dependencies: &latest_v1.CustomDependencies{
				Command: "bazel query deps",
				Ignore:  []string{"ignoreme"},
			},
			expectedErrors: 1,
		}, {
			description:  "nil dependencies",
			dependencies: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest_v1.Artifact{
				ArtifactType: latest_v1.ArtifactType{
					CustomArtifact: &latest_v1.CustomArtifact{
						Dependencies: test.dependencies,
					},
				},
			}

			errs := validateCustomDependencies([]*latest_v1.Artifact{artifact})

			t.CheckDeepEqual(test.expectedErrors, len(errs))
		})
	}
}

func TestValidatePortForwardResources(t *testing.T) {
	tests := []struct {
		resourceType string
		shouldErr    bool
	}{
		{resourceType: "pod"},
		{resourceType: "Deployment"},
		{resourceType: "service"},
		{resourceType: "replicaset"},
		{resourceType: "replicationcontroller"},
		{resourceType: "statefulset"},
		{resourceType: "daemonset"},
		{resourceType: "cronjob"},
		{resourceType: "job"},
		{resourceType: "dne", shouldErr: true},
	}
	for _, test := range tests {
		testutil.Run(t, test.resourceType, func(t *testutil.T) {
			pfrs := []*latest_v1.PortForwardResource{
				{
					Type: latest_v1.ResourceType(test.resourceType),
				},
			}
			errs := validatePortForwardResources(pfrs)
			var err error
			if len(errs) > 0 {
				err = errs[0]
			}

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateImageNames(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest_v1.Artifact
		shouldErr   bool
	}{
		{
			description: "no name",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "",
			}},
			shouldErr: true,
		},
		{
			description: "valid",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
			}},
			shouldErr: false,
		},
		{
			description: "duplicates",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img",
			}, {
				ImageName: "img",
			}},
			shouldErr: true,
		},
		{
			description: "shouldn't have a tag",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img:tag",
			}},
			shouldErr: true,
		},
		{
			description: "shouldn't have a digest",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182",
			}},
			shouldErr: true,
		},
		{
			description: "no tag nor digest",
			artifacts: []*latest_v1.Artifact{{
				ImageName: "img:tag@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182",
			}},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Build: latest_v1.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateJibPluginType(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest_v1.Artifact
		shouldErr   bool
	}{
		{
			description: "no type",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{},
					},
				},
			},
		},
		{
			description: "maven",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{
							Type: "maven",
						},
					},
				},
			},
		},
		{
			description: "gradle",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{
							Type: "gradle",
						},
					},
				},
			},
		},
		{
			description: "empty",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{
							Type: "",
						},
					},
				},
			},
		},
		{
			description: "cAsE inSenSiTiVe",
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{
							Type: "gRaDlE",
						},
					},
				},
			},
		},
		{
			description: "invalid type",
			shouldErr:   true,
			artifacts: []*latest_v1.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest_v1.ArtifactType{
						JibArtifact: &latest_v1.JibArtifact{
							Type: "invalid",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Build: latest_v1.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateLogsConfig(t *testing.T) {
	tests := []struct {
		prefix    string
		cfg       latest_v1.LogsConfig
		shouldErr bool
	}{
		{prefix: "auto", shouldErr: false},
		{prefix: "container", shouldErr: false},
		{prefix: "podAndContainer", shouldErr: false},
		{prefix: "none", shouldErr: false},
		{prefix: "", shouldErr: false},
		{prefix: "unknown", shouldErr: true},
	}
	for _, test := range tests {
		testutil.Run(t, test.prefix, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{
							Logs: latest_v1.LogsConfig{
								Prefix: test.prefix,
							},
						},
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateAcyclicDependencies(t *testing.T) {
	tests := []struct {
		description string
		artifactLen int
		dependency  map[int][]int
		shouldErr   bool
	}{
		{
			description: "artifacts with no dependency",
			artifactLen: 5,
		},
		{
			description: "artifacts with no circular dependencies 1",
			dependency: map[int][]int{
				0: {2, 3},
				1: {3},
				2: {1},
				3: {4},
			},
			artifactLen: 5,
		},
		{
			description: "artifacts with no circular dependencies 2",
			dependency: map[int][]int{
				0: {4, 5},
				1: {4, 5},
				2: {4, 5},
				3: {4, 5},
			},
			artifactLen: 6,
		},
		{
			description: "artifacts with circular dependencies",
			dependency: map[int][]int{
				0: {2, 3},
				1: {0},
				2: {1},
				3: {4},
			},
			artifactLen: 5,
			shouldErr:   true,
		},
		{
			description: "artifacts with circular dependencies (self)",
			dependency: map[int][]int{
				0: {0},
				1: {},
			},
			artifactLen: 2,
			shouldErr:   true,
		},
		{
			description: "0 artifacts",
			artifactLen: 0,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifacts := make([]*latest_v1.Artifact, test.artifactLen)
			for i := 0; i < test.artifactLen; i++ {
				a := fmt.Sprintf("artifact%d", i+1)
				artifacts[i] = &latest_v1.Artifact{ImageName: a}
			}

			setDependencies(artifacts, test.dependency)
			errs := validateAcyclicDependencies(artifacts)
			expected := []error{
				fmt.Errorf(`cycle detected in build dependencies involving "artifact1"`),
			}
			if test.shouldErr {
				t.CheckDeepEqual(expected, errs, cmp.Comparer(errorsComparer))
			} else {
				t.CheckDeepEqual(0, len(errs))
			}
		})
	}
}

// setDependencies constructs a graph of artifact dependencies using the map as an adjacency list representation of indices in the artifacts array.
// For example:
// m = {
//    0 : {1, 2},
//    2 : {3},
//}
// implies that a[0] artifact depends on a[1] and a[2]; and a[2] depends on a[3].
func setDependencies(a []*latest_v1.Artifact, d map[int][]int) {
	for k, dep := range d {
		for i := range dep {
			a[k].Dependencies = append(a[k].Dependencies, &latest_v1.ArtifactDependency{
				ImageName: a[dep[i]].ImageName,
			})
		}
	}
}

func TestValidateUniqueDependencyAliases(t *testing.T) {
	cfgs := []*latest_v1.SkaffoldConfig{
		{
			Pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					Artifacts: []*latest_v1.Artifact{
						{
							ImageName: "artifact1",
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "alias2", ImageName: "artifact2a"},
								{Alias: "alias2", ImageName: "artifact2b"},
							},
						},
						{
							ImageName: "artifact2",
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "alias1", ImageName: "artifact1"},
								{Alias: "alias2", ImageName: "artifact1"},
							},
						},
					},
				},
			},
		},
	}
	expected := []error{
		fmt.Errorf(`invalid build dependency for artifact "artifact1": alias "alias2" repeated`),
		fmt.Errorf(`unknown build dependency "artifact2a" for artifact "artifact1"`),
	}
	errs := validateArtifactDependencies(cfgs)
	testutil.CheckDeepEqual(t, expected, errs, cmp.Comparer(errorsComparer))
}

func TestValidateSingleKubeContext(t *testing.T) {
	tests := []struct {
		description string
		configs     []*latest_v1.SkaffoldConfig
		err         []error
	}{
		{
			description: "different kubeContext specified",
			configs: []*latest_v1.SkaffoldConfig{
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{
							KubeContext: "",
						},
					},
				},
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{
							KubeContext: "foo",
						},
					},
				},
			},
			err: []error{errors.New("all configs should have the same value for `deploy.kubeContext`")},
		},
		{
			description: "same kubeContext specified",
			configs: []*latest_v1.SkaffoldConfig{
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{
							KubeContext: "foo",
						},
					},
				},
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{
							KubeContext: "foo",
						},
					},
				},
			},
		},
		{
			description: "no kubeContext specified",
			configs: []*latest_v1.SkaffoldConfig{
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{},
					},
				},
				{
					Pipeline: latest_v1.Pipeline{
						Deploy: latest_v1.DeployConfig{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			errs := validateSingleKubeContext(test.configs)
			t.CheckDeepEqual(test.err, errs, cmp.Comparer(errorsComparer))
		})
	}
}

func TestValidateValidDependencyAliases(t *testing.T) {
	cfgs := []*latest_v1.SkaffoldConfig{
		{
			Pipeline: latest_v1.Pipeline{
				Build: latest_v1.BuildConfig{
					Artifacts: []*latest_v1.Artifact{
						{
							ImageName: "artifact1",
						},
						{
							ImageName: "artifact2",
							ArtifactType: latest_v1.ArtifactType{
								DockerArtifact: &latest_v1.DockerArtifact{},
							},
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "ARTIFACT_1", ImageName: "artifact1"},
								{Alias: "1_ARTIFACT", ImageName: "artifact1"},
							},
						},
						{
							ImageName: "artifact3",
							ArtifactType: latest_v1.ArtifactType{
								DockerArtifact: &latest_v1.DockerArtifact{},
							},
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "artifact!", ImageName: "artifact1"},
								{Alias: "artifact#1", ImageName: "artifact1"},
							},
						},
						{
							ImageName: "artifact4",
							ArtifactType: latest_v1.ArtifactType{
								CustomArtifact: &latest_v1.CustomArtifact{},
							},
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "alias1", ImageName: "artifact1"},
								{Alias: "alias2", ImageName: "artifact2"},
							},
						},
						{
							ImageName: "artifact5",
							ArtifactType: latest_v1.ArtifactType{
								BuildpackArtifact: &latest_v1.BuildpackArtifact{},
							},
							Dependencies: []*latest_v1.ArtifactDependency{
								{Alias: "artifact!", ImageName: "artifact1"},
								{Alias: "artifact#1", ImageName: "artifact1"},
							},
						},
					},
				},
			},
		},
	}
	expected := []error{
		fmt.Errorf(`invalid build dependency for artifact "artifact2": alias "1_ARTIFACT" doesn't match required pattern %q`, dependencyAliasPattern),
		fmt.Errorf(`invalid build dependency for artifact "artifact3": alias "artifact!" doesn't match required pattern %q`, dependencyAliasPattern),
		fmt.Errorf(`invalid build dependency for artifact "artifact3": alias "artifact#1" doesn't match required pattern %q`, dependencyAliasPattern),
	}
	errs := validateArtifactDependencies(cfgs)
	testutil.CheckDeepEqual(t, expected, errs, cmp.Comparer(errorsComparer))
}

func errorsComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Error() == b.Error()
}

func TestValidateTaggingPolicy(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest_v1.BuildConfig
		shouldErr   bool
	}{
		{
			description: "ShaTagger can be used when tryImportMissing is disabled",
			shouldErr:   false,
			cfg: latest_v1.BuildConfig{
				BuildType: latest_v1.BuildType{
					LocalBuild: &latest_v1.LocalBuild{
						TryImportMissing: false,
					},
				},
				TagPolicy: latest_v1.TagPolicy{
					ShaTagger: &latest_v1.ShaTagger{},
				},
			},
		},
		{
			description: "ShaTagger can not be used when tryImportMissing is enabled",
			shouldErr:   true,
			cfg: latest_v1.BuildConfig{
				BuildType: latest_v1.BuildType{
					LocalBuild: &latest_v1.LocalBuild{
						TryImportMissing: true,
					},
				},
				TagPolicy: latest_v1.TagPolicy{
					ShaTagger: &latest_v1.ShaTagger{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				[]*latest_v1.SkaffoldConfig{{
					Pipeline: latest_v1.Pipeline{
						Build: test.cfg,
					},
				}})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCustomTest(t *testing.T) {
	tests := []struct {
		description    string
		command        string
		dependencies   *latest_v1.CustomTestDependencies
		expectedErrors int
	}{
		{
			description: "no errors",
			command:     "echo Hello!",
			dependencies: &latest_v1.CustomTestDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
		}, {
			description: "empty command",
			command:     "",
			dependencies: &latest_v1.CustomTestDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
			expectedErrors: 1,
		}, {
			description: "use both path and command",
			command:     "echo Hello!",
			dependencies: &latest_v1.CustomTestDependencies{
				Command: "bazel query deps",
				Paths:   []string{"somepath"},
			},
			expectedErrors: 1,
		}, {
			description: "ignore in conjunction with command",
			command:     "echo Hello!",
			dependencies: &latest_v1.CustomTestDependencies{
				Command: "bazel query deps",
				Ignore:  []string{"ignoreme"},
			},
			expectedErrors: 1,
		}, {
			command:      "echo Hello!",
			description:  "nil dependencies",
			dependencies: nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testCase := &latest_v1.TestCase{
				ImageName: "image",
				CustomTests: []latest_v1.CustomTest{{
					Command:      test.command,
					Dependencies: test.dependencies,
				}},
			}

			errs := validateCustomTest([]*latest_v1.TestCase{testCase})
			t.CheckDeepEqual(test.expectedErrors, len(errs))
		})
	}
}
