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

var fakeArtifactConfig = func(a *latest.Artifact, mode config.RunMode) (string, error) {
	if a.ArtifactType.DockerArtifact != nil {
		return "docker/target=" + a.ArtifactType.DockerArtifact.Target, nil
	}
	return string(mode), nil
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
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description:  "ignore file not found",
			dependencies: []string{"a", "b", "not-found"},
			artifact:     &latest.Artifact{},
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description:  "dependencies in different orders",
			dependencies: []string{"b", "a"},
			artifact:     &latest.Artifact{},
			expected:     "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
		},
		{
			description: "no dependencies",
			artifact:    &latest.Artifact{},
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
			expected: "f947b5aad32734914aa2dea0ec95bceff257037e6c2a529007183c3f21547eae",
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
			expected: "09b366c764d0e39f942283cc081d5522b9dde52e725376661808054e3ed0177f",
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
			expected: "eb53afc0e8cbe348a0934b75260d154d83d3370e4414c25a9d3a67428211a0ea",
		},
		{
			description:  "env variables",
			dependencies: []string{"a", "b"},
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{
						Env: []string{"key=value"},
					},
				},
			},
			expected: "948abd8933667600679259dbb38cf2cc55c3098def78baec8dcd4d6851b9e0cd",
		},
		{
			description:  "devmode",
			dependencies: []string{"a", "b"},
			artifact:     &latest.Artifact{},
			mode:         config.RunModes.Dev,
			expected:     "c17f949a0b1e4296dba726284454488ad8d7ef51a1199eafc7cc0b7e43dec6ca",
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
		}, config.RunModes.Build)
		t.CheckNoError(err)

		config2, err := artifactConfig(&latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					Target: "other",
				},
			},
		}, config.RunModes.Build)
		t.CheckNoError(err)

		if config1 == config2 {
			t.Errorf("configs should be different: [%s] [%s]", config1, config2)
		}
	})
}

func TestArtifactConfigDevMode(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		artifact := latest.ArtifactType{
			BuildpackArtifact: &latest.BuildpackArtifact{
				Builder: "any/builder",
			},
		}
		sync := &latest.Sync{
			Auto: &latest.Auto{},
		}

		conf, err := artifactConfig(&latest.Artifact{
			ArtifactType: artifact,
			Sync:         sync,
		}, config.RunModes.Build)
		t.CheckNoError(err)

		configDevMode, err := artifactConfig(&latest.Artifact{
			ArtifactType: artifact,
			Sync:         sync,
		}, config.RunModes.Dev)
		t.CheckNoError(err)

		if conf == configDevMode {
			t.Errorf("configs should be different: [%s] [%s]", conf, configDevMode)
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

func TestRetrieveBuildArgs(t *testing.T) {
	tests := []struct {
		description  string
		artifactType latest.ArtifactType
		expected     map[string]*string
	}{
		{
			description: "docker artifact with build args",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{},
				},
			},
			expected: map[string]*string{},
		}, {
			description: "docker artifact without build args",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{},
			},
		}, {
			description: "kaniko artifact with build args",
			artifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{
					BuildArgs: map[string]*string{},
				},
			},
			expected: map[string]*string{},
		}, {
			description: "kaniko artifact without build args",
			artifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{},
			},
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
			expected: map[string]*string{},
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
			actual := retrieveBuildArgs(&latest.Artifact{
				ArtifactType: test.artifactType,
			})

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestConvertBuildArgsToStringArray(t *testing.T) {
	tests := []struct {
		description string
		buildArgs   map[string]*string
		expected    []string
	}{
		{
			description: "regular key:value build args",
			buildArgs: map[string]*string{
				"one": stringPointer("1"),
				"two": stringPointer("2"),
			},
			expected: []string{"one=1", "two=2"},
		}, {
			description: "empty key:value build args",
			buildArgs: map[string]*string{
				"one": stringPointer(""),
				"two": stringPointer(""),
			},
			expected: []string{"one=", "two="},
		}, {
			description: "build args with nil value",
			buildArgs: map[string]*string{
				"one": nil,
				"two": nil,
			},
			expected: []string{"one", "two"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := convertBuildArgsToStringArray(test.buildArgs)

			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func stringPointer(s string) *string {
	return &s
}
