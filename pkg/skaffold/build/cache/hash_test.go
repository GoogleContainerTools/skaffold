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

package cache

import (
	"context"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func stubDependencyLister(dependencies []string) DependencyLister {
	return func(context.Context, *latest.Artifact) ([]string, error) {
		return dependencies, nil
	}
}

var mockCacheHasher = func(s string) (string, error) {
	if s == "not-found" {
		return "", os.ErrNotExist
	}
	return s, nil
}

var fakeArtifactConfig = func(a *latest.Artifact) (string, error) {
	if a.ArtifactType.DockerArtifact != nil {
		return "docker/target=" + a.ArtifactType.DockerArtifact.Target, nil
	}
	return "", nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		description  string
		dependencies []string
		artifact     *latest.Artifact
		mode         config.RunMode
		expected     string
	}{
		{
			description:  "hash for artifact",
			dependencies: []string{"a", "b"},
			artifact:     &latest.Artifact{},
			mode:         config.RunModes.Dev,
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description:  "ignore file not found",
			dependencies: []string{"a", "b", "not-found"},
			artifact:     &latest.Artifact{},
			mode:         config.RunModes.Dev,
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description:  "dependencies in different orders",
			dependencies: []string{"b", "a"},
			artifact:     &latest.Artifact{},
			mode:         config.RunModes.Dev,
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description: "no dependencies",
			artifact:    &latest.Artifact{},
			mode:        config.RunModes.Dev,
			expected:    "7c077ca2308714493d07163e1033c4282bd869ff6d477b3e77408587f95e2930",
		},
		{
			description: "docker target",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						Target: "target",
					},
				},
			},
			mode:     config.RunModes.Dev,
			expected: "e8ecd3e41bcdeb58b23b237c6c045e75e0b2e5c23a7f39cc5230afddaf49bcf9",
		},
		{
			description: "different docker target",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						Target: "other",
					},
				},
			},
			mode:     config.RunModes.Dev,
			expected: "9bc1e72592e8f51b33287f51e03d1bb063cb8eeed9e0542fd3e3da3f7f3a73d7",
		},
		{
			description:  "build args",
			dependencies: []string{"a", "b"},
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						BuildArgs: map[string]*string{
							"key": util.StringPtr("value"),
						},
					},
				},
			},
			mode:     config.RunModes.Dev,
			expected: "eb53afc0e8cbe348a0934b75260d154d83d3370e4414c25a9d3a67428211a0ea",
		},
		{
			description:  "buildpack in dev mode",
			dependencies: []string{"a", "b"},
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{},
				},
			},
			mode:     config.RunModes.Dev,
			expected: "17ca23f987471b9841213d256b1c163504f6d4ccc51613cd10a62c56424a89e6",
		},
		{
			description:  "buildpack in debug mode",
			dependencies: []string{"a", "b"},
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{},
				},
			},
			mode:     config.RunModes.Debug,
			expected: "a15f9e22a5c5a244c47a5205d577fdbf80e886a4b36915050113b082850a9c5c",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashFunction, mockCacheHasher)
			t.Override(&artifactConfigFunction, fakeArtifactConfig)

			depLister := stubDependencyLister(test.dependencies)
			actual, err := getHashForArtifact(context.Background(), depLister, test.artifact, test.mode)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestArtifactConfig(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		config1, err := artifactConfig(&latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					Target: "target",
				},
			},
		})
		t.CheckNoError(err)

		config2, err := artifactConfig(&latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					Target: "other",
				},
			},
		})
		t.CheckNoError(err)

		if config1 == config2 {
			t.Errorf("configs should be different: [%s] [%s]", config1, config2)
		}
	})
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		mode     config.RunMode
		expected string
	}{
		{
			mode:     config.RunModes.Debug,
			expected: "771e726436816ce229a2838b38aee8c85c7dda4411e7ba68cfd898473ae12ada",
		},
		{
			mode:     config.RunModes.Dev,
			expected: "31616940358b3c1535a1b4bcd0ffa8a1b851d0e5b10d7444c19825eb0f2ba69d",
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			artifact := &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						BuildArgs: map[string]*string{"one": stringPointer("1"), "two": stringPointer("2")},
					},
				},
			}

			t.Override(&hashFunction, mockCacheHasher)
			t.Override(&artifactConfigFunction, fakeArtifactConfig)

			actual, err := getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, test.mode)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)

			// Change order of buildargs
			artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"two": stringPointer("2"), "one": stringPointer("1")}
			actual, err = getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, test.mode)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)

			// Change build args, get different hash
			artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"one": stringPointer("1")}
			actual, err = getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, test.mode)

			t.CheckNoError(err)
			if actual == test.expected {
				t.Fatal("got same hash as different artifact; expected different hashes.")
			}
		})
	}
}

func TestBuildArgsEnvSubstitution(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		original := util.OSEnviron
		defer func() { util.OSEnviron = original }()
		util.OSEnviron = func() []string {
			return []string{"FOO=bar"}
		}

		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{"env": stringPointer("${{.FOO}}")},
				},
			},
		}

		t.Override(&hashFunction, mockCacheHasher)
		t.Override(&artifactConfigFunction, fakeArtifactConfig)

		depLister := stubDependencyLister([]string{"dep"})
		hash1, err := getHashForArtifact(context.Background(), depLister, artifact, config.RunModes.Build)

		t.CheckNoError(err)

		// Make sure hash is different with a new env

		util.OSEnviron = func() []string {
			return []string{"FOO=baz"}
		}

		hash2, err := getHashForArtifact(context.Background(), depLister, artifact, config.RunModes.Build)

		t.CheckNoError(err)
		if hash1 == hash2 {
			t.Fatal("hashes are the same even though build arg changed")
		}
	})
}

func TestCacheHasher(t *testing.T) {
	tests := []struct {
		description   string
		differentHash bool
		newFilename   string
		update        func(oldFile string, folder *testutil.TempDir)
	}{
		{
			description:   "change filename",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
			},
		},
		{
			description:   "change file contents",
			differentHash: true,
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			description:   "change both",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			description:   "change nothing",
			differentHash: false,
			update:        func(oldFile string, folder *testutil.TempDir) {},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			originalFile := "foo"
			originalContents := "contents"

			tmpDir := t.NewTempDir().
				Write(originalFile, originalContents)

			path := originalFile
			depLister := stubDependencyLister([]string{tmpDir.Path(originalFile)})

			oldHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{}, config.RunModes.Build)
			t.CheckNoError(err)

			test.update(originalFile, tmpDir)
			if test.newFilename != "" {
				path = test.newFilename
			}

			depLister = stubDependencyLister([]string{tmpDir.Path(path)})
			newHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{}, config.RunModes.Build)

			t.CheckNoError(err)
			t.CheckFalse(test.differentHash && oldHash == newHash)
			t.CheckFalse(!test.differentHash && oldHash != newHash)
		})
	}
}

func TestHashBuildArgs(t *testing.T) {
	tests := []struct {
		description  string
		artifactType latest.ArtifactType
		expected     []string
		mode         config.RunMode
	}{
		{
			description: "docker artifact with build args for dev",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{
						"foo": stringPointer("bar"),
					},
				},
			},
			mode:     config.RunModes.Dev,
			expected: []string{"SKAFFOLD_GO_LDFLAGS=-w", "foo=bar"},
		}, {
			description: "docker artifact with build args for debug",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{
						"foo": stringPointer("bar"),
					},
				},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"SKAFFOLD_GO_GCFLAGS='all=-N -l'", "foo=bar"},
		}, {
			description: "docker artifact without build args for debug",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"SKAFFOLD_GO_GCFLAGS='all=-N -l'"},
		}, {
			description: "docker artifact without build args for dev",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{},
			},
			mode:     config.RunModes.Dev,
			expected: []string{"SKAFFOLD_GO_LDFLAGS=-w"},
		}, {
			description: "kaniko artifact with build args",
			artifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{
					BuildArgs: map[string]*string{},
				},
			},
			expected: nil,
		}, {
			description: "kaniko artifact without build args",
			artifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{},
			},
		}, {
			description: "buildpacks artifact with env for dev",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{
					Env: []string{"foo=bar"},
				},
			},
			mode:     config.RunModes.Dev,
			expected: []string{"GOOGLE_GOLDFLAGS=-w", "foo=bar"},
		}, {
			description: "buildpacks artifact without env for dev",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{},
			},
			mode:     config.RunModes.Dev,
			expected: []string{"GOOGLE_GOLDFLAGS=-w"},
		}, {
			description: "buildpacks artifact with env for debug",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{
					Env: []string{"foo=bar"},
				},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"GOOGLE_GOGCFLAGS='all=-N -l'", "foo=bar"},
		}, {
			description: "buildpacks artifact without env for debug",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"GOOGLE_GOGCFLAGS='all=-N -l'"},
		}, {
			description: "custom artifact, dockerfile dependency, with build args",
			artifactType: latest.ArtifactType{
				CustomArtifact: &latest.CustomArtifact{
					Dependencies: &latest.CustomDependencies{
						Dockerfile: &latest.DockerfileDependency{
							BuildArgs: map[string]*string{},
						},
					},
				},
			},
			expected: nil,
		}, {
			description: "custom artifact, no dockerfile dependency",
			artifactType: latest.ArtifactType{
				CustomArtifact: &latest.CustomArtifact{
					Dependencies: &latest.CustomDependencies{},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := hashBuildArgs(&latest.Artifact{
				ArtifactType: test.artifactType,
			}, test.mode)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func stringPointer(s string) *string {
	return &s
}
