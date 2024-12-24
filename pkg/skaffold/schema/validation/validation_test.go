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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser/configlocations"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var (
	cfgWithErrors = &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ArtifactType: latest.ArtifactType{
							DockerArtifact: &latest.DockerArtifact{},
							BazelArtifact:  &latest.BazelArtifact{},
						},
					},
					{
						ArtifactType: latest.ArtifactType{
							BazelArtifact:  &latest.BazelArtifact{},
							KanikoArtifact: &latest.KanikoArtifact{},
						},
					},
				},
			},
			Deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					LegacyHelmDeploy: &latest.LegacyHelmDeploy{},
					KubectlDeploy:    &latest.KubectlDeploy{},
				},
			},
		},
	}
)

func TestValidateArtifactTypes(t *testing.T) {
	tests := []struct {
		description  string
		bc           latest.BuildConfig
		expectedErrs int
	}{
		{
			description: "gcb - builder not set",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{},
				},
				Artifacts: []*latest.Artifact{
					{
						ImageName: "leeroy-web",
						Workspace: "leeroy-web",
					},
				},
			},
		},
		{
			description: "gcb - custom builder  set",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{},
				},
				Artifacts: []*latest.Artifact{
					{
						ImageName:    "leeroy-web",
						Workspace:    "leeroy-web",
						ArtifactType: latest.ArtifactType{CustomArtifact: &latest.CustomArtifact{}},
					},
				},
			},
			expectedErrs: 1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			config := &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: test.bc,
				},
			}
			defaults.Set(config)
			cfg := &parser.SkaffoldConfigEntry{SkaffoldConfig: config,
				YAMLInfos: configlocations.NewYAMLInfos()}
			errs := validateArtifactTypes(cfg, test.bc)

			t.CheckDeepEqual(test.expectedErrs, len(errs))
		})
	}
}
func TestValidateSchema(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest.SkaffoldConfig
		shouldErr   bool
	}{
		{
			description: "config with errors",
			cfg:         cfgWithErrors,
			shouldErr:   true,
		},
		{
			description: "empty config",
			cfg:         &latest.SkaffoldConfig{},
			shouldErr:   true,
		},
		{
			description: "minimal config",
			cfg: &latest.SkaffoldConfig{
				APIVersion: "foo",
				Kind:       "bar",
			},
			shouldErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := Process(parser.SkaffoldConfigSet{&parser.SkaffoldConfigEntry{SkaffoldConfig: test.cfg, YAMLInfos: configlocations.NewYAMLInfos()}},
				Options{CheckDeploySource: false})

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
			actual := visitStructs(&parser.SkaffoldConfigEntry{YAMLInfos: configlocations.NewYAMLInfos()}, reflect.ValueOf(test.input), alwaysErr)
			t.CheckDeepEqual(test.expectedErrs, len(actual))
		})
	}
}

func TestValidateNetworkMode(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
		env         []string
	}{
		{
			description: "not a docker artifact",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/bazel",
					ArtifactType: latest.ArtifactType{
						BazelArtifact: &latest.BazelArtifact{},
					},
				},
			},
		},
		{
			description: "no networkmode",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/no-network",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{},
					},
				},
			},
		},
		{
			description: "bridge",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/bridge",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Bridge",
						},
					},
				},
			},
		},
		{
			description: "empty container's network stack",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "empty container's network stack in env var",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:-not-valid",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack '-not-valid' in env var",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:fußball",
						},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "wrong container's network stack 'fussball' in env var",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:unique",
						},
					},
				},
			},
		},
		{
			description: "container's network stack 'unique' in env var",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env: []string{"CONTAINER=unique"},
		},
		{
			description: "container's network stack 'unique-id.123'",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:unique-id.123",
						},
					},
				},
			},
		},
		{
			description: "container's network stack 'unique-id.123' in env var",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Container:{{.CONTAINER}}",
						},
					},
				},
			},
			env: []string{"CONTAINER=unique-id.123"},
		},
		{
			description: "none",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/none",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "None",
						},
					},
				},
			},
		},
		{
			description: "host",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/host",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Host",
						},
					},
				},
			},
		},
		{
			description: "custom networkmode",
			shouldErr:   false,
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/custom",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "my-network-mode",
						},
					},
				},
			},
		},
		{
			description: "case insensitive",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/case-insensitive",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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

			err := Process(parser.SkaffoldConfigSet{&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}}}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

type fakeCommonAPIClient struct {
	client.CommonAPIClient
	expectedResponse []types.Container
}

func (f fakeCommonAPIClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.expectedResponse, nil
}

func TestValidateNetworkModeDockerContainerExists(t *testing.T) {
	tests := []struct {
		description    string
		artifacts      []*latest.Artifact
		clientResponse []types.Container
		shouldErr      bool
		env            []string
	}{
		{
			description: "no running containers",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/container",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
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
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				fakeClient := &fakeCommonAPIClient{
					CommonAPIClient: &testutil.FakeAPIClient{
						ErrVersion: true,
					},
					expectedResponse: test.clientResponse,
				}
				return docker.NewLocalDaemon(fakeClient, nil, false, nil), nil
			})

			err := ProcessWithRunContext(context.Background(), &runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"default": {
							Build: latest.BuildConfig{
								Artifacts: test.artifacts,
							},
						},
					},
					[]string{"default"}),
			})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateSyncRules(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
	}{
		{
			description: "no artifacts",
			artifacts:   nil,
		},
		{
			description: "no sync rules",
			artifacts: []*latest.Artifact{{
				ImageName: "img",
				Sync:      nil,
			}},
		},
		{
			description: "two good rules",
			artifacts: []*latest.Artifact{{
				ImageName: "img",
				Sync: &latest.Sync{Manual: []*latest.SyncRule{
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
			artifacts: []*latest.Artifact{{
				ImageName: "img",
				Sync: &latest.Sync{Manual: []*latest.SyncRule{
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
			artifacts: []*latest.Artifact{{
				ImageName: "img",
				Sync: &latest.Sync{Manual: []*latest.SyncRule{
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
			artifacts: []*latest.Artifact{{
				ImageName: "img",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{{
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

			err := Process(parser.SkaffoldConfigSet{&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				}}}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCustomDependencies(t *testing.T) {
	tests := []struct {
		description    string
		dependencies   *latest.CustomDependencies
		expectedErrors int
	}{
		{
			description: "no errors",
			dependencies: &latest.CustomDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
		}, {
			description: "ignore in conjunction with dockerfile",
			dependencies: &latest.CustomDependencies{
				Dockerfile: &latest.DockerfileDependency{
					Path: "some/path",
				},
				Ignore: []string{"ignoreme"},
			},
			expectedErrors: 1,
		}, {
			description: "ignore in conjunction with command",
			dependencies: &latest.CustomDependencies{
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
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					CustomArtifact: &latest.CustomArtifact{
						Dependencies: test.dependencies,
					},
				},
			}

			errs := validateCustomDependencies(&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: []*latest.Artifact{
								artifact,
							},
						},
					},
				},
			}, []*latest.Artifact{artifact})

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
			pfrs := []*latest.PortForwardResource{
				{
					Type: latest.ResourceType(test.resourceType),
				},
			}
			errs := validatePortForwardResources(&parser.SkaffoldConfigEntry{YAMLInfos: configlocations.NewYAMLInfos()}, pfrs)
			var err error
			if len(errs) > 0 {
				err = errs[0].Error
			}

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateImageNames(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
	}{
		{
			description: "no name",
			artifacts: []*latest.Artifact{{
				ImageName: "",
			}},
			shouldErr: true,
		},
		{
			description: "valid",
			artifacts: []*latest.Artifact{{
				ImageName: "img",
			}},
			shouldErr: false,
		},
		{
			description: "duplicates",
			artifacts: []*latest.Artifact{{
				ImageName: "img",
			}, {
				ImageName: "img",
			}},
			shouldErr: true,
		},
		{
			description: "shouldn't have a tag",
			artifacts: []*latest.Artifact{{
				ImageName: "img:tag",
			}},
			shouldErr: true,
		},
		{
			description: "shouldn't have a digest",
			artifacts: []*latest.Artifact{{
				ImageName: "img@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182",
			}},
			shouldErr: true,
		},
		{
			description: "no tag nor digest",
			artifacts: []*latest.Artifact{{
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
				parser.SkaffoldConfigSet{
					&parser.SkaffoldConfigEntry{
						YAMLInfos: configlocations.NewYAMLInfos(),
						SkaffoldConfig: &latest.SkaffoldConfig{
							Pipeline: latest.Pipeline{
								Build: latest.BuildConfig{
									Artifacts: test.artifacts,
								},
							},
						},
					},
				}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateJibPluginType(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
	}{
		{
			description: "no type",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{},
					},
				},
			},
		},
		{
			description: "maven",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{
							Type: "maven",
						},
					},
				},
			},
		},
		{
			description: "gradle",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{
							Type: "gradle",
						},
					},
				},
			},
		},
		{
			description: "empty",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{
							Type: "",
						},
					},
				},
			},
		},
		{
			description: "cAsE inSenSiTiVe",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{
							Type: "gRaDlE",
						},
					},
				},
			},
		},
		{
			description: "invalid type",
			shouldErr:   true,
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/jib",
					ArtifactType: latest.ArtifactType{
						JibArtifact: &latest.JibArtifact{
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

			err := Process(parser.SkaffoldConfigSet{&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				},
			}}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateKoSync(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		wantErr     bool
	}{
		{
			description: "basic infer sync with no errors",
			artifacts: []*latest.Artifact{{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"kodata/**/*"},
				},
			}},
		},
		{
			description: "no error for wildcard in Main when no infer sync set up",
			artifacts: []*latest.Artifact{{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						Main: "./...",
					},
				},
				ImageName: "test",
			}},
		},
		{
			description: "error for wildcard in Main when infer sync is set up",
			artifacts: []*latest.Artifact{{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						Main: "./...",
					},
				},
				Sync: &latest.Sync{
					Infer: []string{"kodata/**/*"},
				},
			}},
			wantErr: true,
		},
		{
			description: "error for patterns outside kodata",
			artifacts: []*latest.Artifact{{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
				Sync: &latest.Sync{
					Infer: []string{"**/*"},
				},
			}},
			wantErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfg := &parser.SkaffoldConfigEntry{
				SkaffoldConfig: &latest.SkaffoldConfig{
					APIVersion: latest.Version,
					Kind:       "Config",
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				},
				YAMLInfos: configlocations.NewYAMLInfos(),
			}
			err := Process(parser.SkaffoldConfigSet{cfg}, Options{CheckDeploySource: false})
			t.CheckError(test.wantErr, err)
		})
	}
}

func TestValidateLogsConfig(t *testing.T) {
	tests := []struct {
		prefix    string
		cfg       latest.LogsConfig
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

			err := Process(parser.SkaffoldConfigSet{&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Deploy: latest.DeployConfig{
							Logs: latest.LogsConfig{
								Prefix: test.prefix,
							},
						},
					},
				}}}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateGCBConfig(t *testing.T) {
	tests := []struct {
		desc      string
		bc        latest.BuildConfig
		shouldErr bool
	}{
		{
			desc: "valid worker pool",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{
						WorkerPool: "projects/prj/locations/loc/workerPools/pool",
					}}},
		},
		{
			desc: "invalid worker pool",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{
						WorkerPool: "projects/prj/locations//loc/workerPools/pool",
					}}},
			shouldErr: true,
		},
		{
			desc: "invalid worker pool",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{
						WorkerPool: "projects/prj",
					}}},
			shouldErr: true,
		},
		{
			desc: "empty worker pool",
			bc: latest.BuildConfig{
				BuildType: latest.BuildType{
					GoogleCloudBuild: &latest.GoogleCloudBuild{}}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.desc, func(t *testutil.T) {
			err := validateGCBConfig(&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: test.bc,
					},
				},
			}, test.bc)

			t.CheckDeepEqual(test.shouldErr, len(err) > 0)
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
			artifacts := make([]*latest.Artifact, test.artifactLen)
			for i := 0; i < test.artifactLen; i++ {
				a := fmt.Sprintf("artifact%d", i+1)
				artifacts[i] = &latest.Artifact{ImageName: a}
			}

			setDependencies(artifacts, test.dependency)
			errs := validateAcyclicDependencies(&parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{
					YAMLInfos: configlocations.NewYAMLInfos(),
				},
			}, artifacts)
			expected := []ErrorWithLocation{
				{
					Error:    fmt.Errorf(`cycle detected in build dependencies involving "artifact1"`),
					Location: configlocations.MissingLocation(),
				},
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
//
//	m = {
//	   0 : {1, 2},
//	   2 : {3},
//	}
//
// implies that a[0] artifact depends on a[1] and a[2]; and a[2] depends on a[3].
func setDependencies(a []*latest.Artifact, d map[int][]int) {
	for k, dep := range d {
		for i := range dep {
			a[k].Dependencies = append(a[k].Dependencies, &latest.ArtifactDependency{
				ImageName: a[dep[i]].ImageName,
			})
		}
	}
}

func TestValidateUniqueDependencyAliases(t *testing.T) {
	cfgs := parser.SkaffoldConfigSet{
		&parser.SkaffoldConfigEntry{
			SkaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "artifact1",
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "alias2", ImageName: "artifact2a"},
									{Alias: "alias2", ImageName: "artifact2b"},
								},
							},
							{
								ImageName: "artifact2",
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "alias1", ImageName: "artifact1"},
									{Alias: "alias2", ImageName: "artifact1"},
								},
							},
						},
					},
				},
			},
		},
	}
	expected := []ErrorWithLocation{
		{
			Error:    fmt.Errorf(`invalid build dependency for artifact "artifact1": alias "alias2" repeated`),
			Location: configlocations.MissingLocation(),
		},
		{
			Error:    fmt.Errorf(`unknown build dependency "artifact2a" for artifact "artifact1"`),
			Location: configlocations.MissingLocation(),
		},
	}
	for i := range cfgs {
		cfgs[i].YAMLInfos = configlocations.NewYAMLInfos()
	}
	errs := validateArtifactDependencies(cfgs)
	testutil.CheckDeepEqual(t, expected, errs, cmp.Comparer(errorsComparer))
}

func TestValidateValidDependencyAliases(t *testing.T) {
	cfgs := parser.SkaffoldConfigSet{
		&parser.SkaffoldConfigEntry{
			SkaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "artifact1",
							},
							{
								ImageName: "artifact2",
								ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{},
								},
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "ARTIFACT_1", ImageName: "artifact1"},
									{Alias: "1_ARTIFACT", ImageName: "artifact1"},
								},
							},
							{
								ImageName: "artifact3",
								ArtifactType: latest.ArtifactType{
									DockerArtifact: &latest.DockerArtifact{},
								},
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "artifact!", ImageName: "artifact1"},
									{Alias: "artifact#1", ImageName: "artifact1"},
								},
							},
							{
								ImageName: "artifact4",
								ArtifactType: latest.ArtifactType{
									CustomArtifact: &latest.CustomArtifact{},
								},
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "alias1", ImageName: "artifact1"},
									{Alias: "alias2", ImageName: "artifact2"},
								},
							},
							{
								ImageName: "artifact5",
								ArtifactType: latest.ArtifactType{
									BuildpackArtifact: &latest.BuildpackArtifact{},
								},
								Dependencies: []*latest.ArtifactDependency{
									{Alias: "artifact!", ImageName: "artifact1"},
									{Alias: "artifact#1", ImageName: "artifact1"},
								},
							},
						},
					},
				},
			},
		}}
	expected := []ErrorWithLocation{
		{
			Error:    fmt.Errorf(`invalid build dependency for artifact "artifact2": alias "1_ARTIFACT" doesn't match required pattern %q`, dependencyAliasPattern),
			Location: configlocations.MissingLocation(),
		},
		{
			Error:    fmt.Errorf(`invalid build dependency for artifact "artifact3": alias "artifact!" doesn't match required pattern %q`, dependencyAliasPattern),
			Location: configlocations.MissingLocation(),
		},
		{
			Error:    fmt.Errorf(`invalid build dependency for artifact "artifact3": alias "artifact#1" doesn't match required pattern %q`, dependencyAliasPattern),
			Location: configlocations.MissingLocation(),
		},
	}

	for i := range cfgs {
		cfgs[i].YAMLInfos = configlocations.NewYAMLInfos()
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
		cfg         latest.BuildConfig
		shouldErr   bool
	}{
		{
			description: "ShaTagger can be used when tryImportMissing is disabled",
			shouldErr:   false,
			cfg: latest.BuildConfig{
				BuildType: latest.BuildType{
					LocalBuild: &latest.LocalBuild{
						TryImportMissing: false,
					},
				},
				TagPolicy: latest.TagPolicy{
					ShaTagger: &latest.ShaTagger{},
				},
			},
		},
		{
			description: "ShaTagger can not be used when tryImportMissing is enabled",
			shouldErr:   true,
			cfg: latest.BuildConfig{
				BuildType: latest.BuildType{
					LocalBuild: &latest.LocalBuild{
						TryImportMissing: true,
					},
				},
				TagPolicy: latest.TagPolicy{
					ShaTagger: &latest.ShaTagger{},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{
					YAMLInfos: configlocations.NewYAMLInfos(),
					SkaffoldConfig: &latest.SkaffoldConfig{
						Pipeline: latest.Pipeline{
							Build: test.cfg,
						},
					},
				},
			}, Options{CheckDeploySource: false})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCustomTest(t *testing.T) {
	tests := []struct {
		description    string
		command        string
		dependencies   *latest.CustomTestDependencies
		expectedErrors int
	}{
		{
			description: "no errors",
			command:     "echo Hello!",
			dependencies: &latest.CustomTestDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
		}, {
			description: "empty command",
			command:     "",
			dependencies: &latest.CustomTestDependencies{
				Paths:  []string{"somepath"},
				Ignore: []string{"anotherpath"},
			},
			expectedErrors: 1,
		}, {
			description: "use both path and command",
			command:     "echo Hello!",
			dependencies: &latest.CustomTestDependencies{
				Command: "bazel query deps",
				Paths:   []string{"somepath"},
			},
			expectedErrors: 1,
		}, {
			description: "ignore in conjunction with command",
			command:     "echo Hello!",
			dependencies: &latest.CustomTestDependencies{
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
			testCase := &latest.TestCase{
				ImageName: "image",
				CustomTests: []latest.CustomTest{{
					Command:      test.command,
					Dependencies: test.dependencies,
				}},
			}

			errs := validateCustomTest(&parser.SkaffoldConfigEntry{
				YAMLInfos: configlocations.NewYAMLInfos(),
				SkaffoldConfig: &latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Test: []*latest.TestCase{
							testCase,
						},
					},
				},
			}, []*latest.TestCase{testCase})
			t.CheckDeepEqual(test.expectedErrors, len(errs))
		})
	}
}

func TestValidateKubectlManifests(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		description string
		configs     []*latest.SkaffoldConfig
		files       []string
		shouldErr   bool
	}{
		{
			description: "specified manifest file exists",
			configs: []*latest.SkaffoldConfig{
				{
					Pipeline: latest.Pipeline{
						Render: latest.RenderConfig{
							Generate: latest.Generate{
								RawK8s: []string{filepath.Join(tempDir, "validation-test-exists.yaml")},
							},
						},
					},
				},
			},
			files: []string{"validation-test-exists.yaml"},
		},
		{
			description: "specified manifest file does not exist",
			configs: []*latest.SkaffoldConfig{
				{
					Pipeline: latest.Pipeline{
						Render: latest.RenderConfig{
							Generate: latest.Generate{
								RawK8s: []string{filepath.Join(tempDir, "validation-test-missing.yaml")},
							},
						},
					},
				},
			},
			files:     []string{},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			for _, file := range test.files {
				out, err := os.Create(filepath.Join(tempDir, file))
				if err != nil {
					t.Errorf("error creating manifest file %s: %v", file, err)
				}
				err = out.Close()
				if err != nil {
					t.Errorf("error closing manifest file %s: %v", file, err)
				}
			}

			set := parser.SkaffoldConfigSet{}
			for _, c := range test.configs {
				set = append(set, &parser.SkaffoldConfigEntry{SkaffoldConfig: c, YAMLInfos: configlocations.NewYAMLInfos()})
			}
			errs := validateKubectlManifests(set)
			var err error
			if len(errs) > 0 {
				err = errs[0].Error
			}
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCloudRunLocation(t *testing.T) {
	tests := []struct {
		description      string
		deploy           latest.DeployConfig
		cloudRunLocation string
		cloudRunProject  string
		command          string
		shouldErr        bool
	}{
		{
			description: "location specified in config",
			deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					CloudRunDeploy: &latest.CloudRunDeploy{
						ProjectID: "test-project",
						Region:    "test-region",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "location not specified, config present",
			deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					CloudRunDeploy: &latest.CloudRunDeploy{},
				},
			},
			shouldErr: true,
		},
		{
			description: "location not specified, command doesn't deploy",
			deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					CloudRunDeploy: &latest.CloudRunDeploy{},
				},
			},
			command:   "diagnose",
			shouldErr: false,
		},
		{
			description: "location specified via flag",
			deploy: latest.DeployConfig{
				DeployType: latest.DeployType{
					CloudRunDeploy: &latest.CloudRunDeploy{},
				},
			},
			cloudRunLocation: "test-region",
			shouldErr:        false,
		},
		{
			description:     "project specified via flag, no location",
			cloudRunProject: "test-project",
			shouldErr:       true,
		},
		{
			description:      "project and location specified via flag",
			cloudRunProject:  "test-project",
			cloudRunLocation: "test-location",
			shouldErr:        false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			command := test.command
			if command == "" {
				command = "run"
			}
			err := ProcessWithRunContext(context.Background(), &runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"default": {
							Deploy: test.deploy,
						},
					},
					[]string{"default"}),
				Opts: config.SkaffoldOptions{
					CloudRunProject:  test.cloudRunProject,
					CloudRunLocation: test.cloudRunLocation,
					Command:          command,
				},
			})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateCustomActions(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		errMsg      string
		cfg         runcontext.RunContext
	}{
		{
			description: "repeated action names in same config",
			shouldErr:   true,
			errMsg:      "found duplicate custom action action1. Custom action names must be unique",
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"default": {
							CustomActions: []latest.Action{
								{Name: "action1", Containers: []latest.VerifyContainer{{Name: "container1"}}},
								{Name: "action1", Containers: []latest.VerifyContainer{{Name: "container2"}}},
							},
						},
					},
					[]string{"default"}),
			},
		},
		{
			description: "repeated container names in different actions, same config",
			shouldErr:   true,
			errMsg:      "found duplicate container name repeated-container-name in custom action. Container names must be unique",
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"default": {
							CustomActions: []latest.Action{
								{Name: "action1", Containers: []latest.VerifyContainer{{Name: "repeated-container-name"}}},
								{Name: "action2", Containers: []latest.VerifyContainer{
									{Name: "repeated-container-name"},
									{Name: "container1"},
								}},
							},
						},
					},
					[]string{"default"}),
			},
		},
		{
			description: "repeated action names in different configs",
			shouldErr:   true,
			errMsg:      "found duplicate custom action cross-module-action. Custom action names must be unique",
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"config1": {
							CustomActions: []latest.Action{{Name: "cross-module-action", Containers: []latest.VerifyContainer{{Name: "container1"}}}},
						},
						"config2": {
							CustomActions: []latest.Action{
								{Name: "cross-module-action", Containers: []latest.VerifyContainer{{Name: "container2"}}},
								{Name: "action1", Containers: []latest.VerifyContainer{{Name: "container3"}}},
							},
						},
					},
					[]string{"config1", "config2"}),
			},
		},
		{
			description: "repeated container names in different configs",
			shouldErr:   true,
			errMsg:      "found duplicate container name repeated-container-name in custom action. Container names must be unique",
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"config1": {
							CustomActions: []latest.Action{{Name: "action1", Containers: []latest.VerifyContainer{{Name: "repeated-container-name"}}}},
						},
						"config2": {
							CustomActions: []latest.Action{
								{Name: "action2", Containers: []latest.VerifyContainer{{Name: "container2"}}},
								{Name: "action3", Containers: []latest.VerifyContainer{{Name: "repeated-container-name"}}},
							},
						},
					},
					[]string{"config1", "config2"}),
			},
		},
		{
			description: "unique custom action names and containers across configs",
			shouldErr:   false,
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"config1": {
							CustomActions: []latest.Action{{Name: "action1", Containers: []latest.VerifyContainer{{Name: "container1"}}}},
						},
						"config2": {
							CustomActions: []latest.Action{
								{Name: "action2", Containers: []latest.VerifyContainer{{Name: "container2"}}},
								{Name: "action3", Containers: []latest.VerifyContainer{{Name: "container3"}}},
							},
						},
					},
					[]string{"config1", "config2"}),
			},
		},
		{
			description: "custom action with two execution modes defined",
			shouldErr:   true,
			errMsg:      "custom action action1 have more than one execution mode defined. custom actions must have only one execution mode",
			cfg: runcontext.RunContext{
				Pipelines: runcontext.NewPipelines(
					map[string]latest.Pipeline{
						"config1": {
							CustomActions: []latest.Action{{
								Name: "action1",
								ExecutionModeConfig: latest.ActionExecutionModeConfig{
									VerifyExecutionModeType: latest.VerifyExecutionModeType{
										KubernetesClusterExecutionMode: &latest.KubernetesClusterVerifier{},
										LocalExecutionMode:             &latest.LocalVerifier{},
									},
								},
								Containers: []latest.VerifyContainer{{Name: "container1"}},
							}},
						},
					},
					[]string{"config1"}),
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := ProcessWithRunContext(context.Background(), &test.cfg)
			t.CheckError(test.shouldErr, err)

			if test.shouldErr {
				t.CheckErrorContains(test.errMsg, err)
			}
		})
	}
}
