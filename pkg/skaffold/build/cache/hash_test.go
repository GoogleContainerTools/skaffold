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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
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

var fakeArtifactConfig = func(a *latest.Artifact, devMode bool) (string, error) {
	if a.ArtifactType.DockerArtifact != nil {
		return "docker/target=" + a.ArtifactType.DockerArtifact.Target, nil
	}
	if devMode {
		return "devmode", nil
	}
	return "other", nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		description  string
		dependencies []string
		artifact     *latest.Artifact
		devMode      bool
		expected     string
	}{
		{
			description:  "hash for artifact",
			dependencies: []string{"a", "b"},
			artifact:     &latest.Artifact{},
			expected:     "448cef891dcc63fe03414342751e30b9f591c3a9dbf488d785e55269591cc26a",
		},
		{
			description:  "ignore file not found",
			dependencies: []string{"a", "b", "not-found"},
			artifact:     &latest.Artifact{},
			expected:     "448cef891dcc63fe03414342751e30b9f591c3a9dbf488d785e55269591cc26a",
		},
		{
			description:  "dependencies in different orders",
			dependencies: []string{"b", "a"},
			artifact:     &latest.Artifact{},
			expected:     "448cef891dcc63fe03414342751e30b9f591c3a9dbf488d785e55269591cc26a",
		},
		{
			description: "no dependencies",
			artifact:    &latest.Artifact{},
			expected:    "5c1047030c6cccf3b28e879b588526e146c8d671d4d9b7fbb5ecb922af0bf9a5",
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
			expected: "2bf7c8b1399d0c6a5c1b39e8fdb60dbfdcf90f5322de6f98d3378ea532c3248a",
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
			expected: "7eb5019be5af98ce5d62a302b23408a278b307a89cdc32ee49ab8e66dfcaac78",
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
			expected: "4b04663930accc2e32680ae57b4f9981d4e673d393f6dd1da4bcb03cc59639a9",
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
			expected: "3df767d39ea17545165f5cc9fd4ad20b02dc40e7cb24bb3188a42d34161101ee",
		},
		{
			description:  "devmode",
			dependencies: []string{"a", "b"},
			artifact:     &latest.Artifact{},
			devMode:      true,
			expected:     "d18a2202185518cd139dbdff9be2e4fb972a5de7f2f2448c41b05cb310d1e875",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashFunction, mockCacheHasher)
			t.Override(&artifactConfigFunction, fakeArtifactConfig)

			depLister := stubDependencyLister(test.dependencies)
			actual, err := getHashForArtifact(context.Background(), depLister, test.artifact, build.BuilderOptions{}, test.devMode)

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
		}, false)
		t.CheckNoError(err)

		config2, err := artifactConfig(&latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					Target: "other",
				},
			},
		}, false)
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

		config, err := artifactConfig(&latest.Artifact{
			ArtifactType: artifact,
			Sync:         sync,
		}, false)
		t.CheckNoError(err)

		configDevMode, err := artifactConfig(&latest.Artifact{
			ArtifactType: artifact,
			Sync:         sync,
		}, true)
		t.CheckNoError(err)

		if config == configDevMode {
			t.Errorf("configs should be different: [%s] [%s]", config, configDevMode)
		}
	})
}

func TestBuildArgs(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		expected := "8f39eede184d46ce6122895c7c9c2bf93cb8f69c9322fe710d1d8384c351e657"

		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{"one": stringPointer("1"), "two": stringPointer("2")},
				},
			},
		}

		t.Override(&hashFunction, mockCacheHasher)
		t.Override(&artifactConfigFunction, fakeArtifactConfig)

		actual, err := getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, build.BuilderOptions{}, false)

		t.CheckNoError(err)
		t.CheckDeepEqual(expected, actual)

		// Change order of buildargs
		artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"two": stringPointer("2"), "one": stringPointer("1")}
		actual, err = getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, build.BuilderOptions{}, false)

		t.CheckNoError(err)
		t.CheckDeepEqual(expected, actual)

		// Change build args, get different hash
		artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"one": stringPointer("1")}
		actual, err = getHashForArtifact(context.Background(), stubDependencyLister(nil), artifact, build.BuilderOptions{}, false)

		t.CheckNoError(err)
		if actual == expected {
			t.Fatal("got same hash as different artifact; expected different hashes.")
		}
	})
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
		hash1, err := getHashForArtifact(context.Background(), depLister, artifact, build.BuilderOptions{}, false)

		t.CheckNoError(err)

		// Make sure hash is different with a new env

		util.OSEnviron = func() []string {
			return []string{"FOO=baz"}
		}

		hash2, err := getHashForArtifact(context.Background(), depLister, artifact, build.BuilderOptions{}, false)

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

			oldHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{}, build.BuilderOptions{}, false)
			t.CheckNoError(err)

			test.update(originalFile, tmpDir)
			if test.newFilename != "" {
				path = test.newFilename
			}

			depLister = stubDependencyLister([]string{tmpDir.Path(path)})
			newHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{}, build.BuilderOptions{}, false)

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
