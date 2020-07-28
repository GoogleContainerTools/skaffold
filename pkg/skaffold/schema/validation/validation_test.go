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
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
					HelmDeploy:    &latest.HelmDeploy{},
					KubectlDeploy: &latest.KubectlDeploy{},
				},
			},
		},
	}
)

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
			err := Process(test.cfg)

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
		artifacts   []*latest.Artifact
		shouldErr   bool
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
			description: "invalid networkmode",
			shouldErr:   true,
			artifacts: []*latest.Artifact{
				{
					ImageName: "image/bad",
					ArtifactType: latest.ArtifactType{
						DockerArtifact: &latest.DockerArtifact{
							NetworkMode: "Bad",
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

			err := Process(
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
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

			err := Process(
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				})

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

			errs := validateCustomDependencies([]*latest.Artifact{artifact})

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
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				})

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

			err := Process(
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateWorkspaces(t *testing.T) {
	tmpDir := testutil.NewTempDir(t).Touch("file")
	tmpFile := tmpDir.Path("file")

	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		shouldErr   bool
	}{
		{
			description: "no workspace",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
				},
			},
		},
		{
			description: "directory that exists",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: tmpDir.Root(),
				},
			},
		},
		{
			description: "error on non-existent location",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: "doesnotexist",
				},
			},
			shouldErr: true,
		},
		{
			description: "error on file",
			artifacts: []*latest.Artifact{
				{
					ImageName: "image",
					Workspace: tmpFile,
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// disable yamltags validation
			t.Override(&validateYamltags, func(interface{}) error { return nil })

			err := Process(
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Build: latest.BuildConfig{
							Artifacts: test.artifacts,
						},
					},
				})

			t.CheckError(test.shouldErr, err)
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

			err := Process(
				&latest.SkaffoldConfig{
					Pipeline: latest.Pipeline{
						Deploy: latest.DeployConfig{
							Logs: latest.LogsConfig{
								Prefix: test.prefix,
							},
						},
					},
				})

			t.CheckError(test.shouldErr, err)
		})
	}
}
