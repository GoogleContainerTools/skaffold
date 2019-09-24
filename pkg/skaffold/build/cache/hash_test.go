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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDependencyLister struct {
	dependencies []string
}

func (m *stubDependencyLister) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return m.dependencies, nil
}

var mockCacheHasher = func(s string) (string, error) {
	return s, nil
}

var fakeArtifactConfig = func(a *latest.Artifact) (string, error) {
	if a.ArtifactType.DockerArtifact != nil {
		return "docker/target=" + a.ArtifactType.DockerArtifact.Target, nil
	}
	return "other", nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		description  string
		dependencies []string
		artifact     *latest.Artifact
		expected     string
	}{
		{
			description:  "hash for artifact",
			dependencies: []string{"a", "b"},
			artifact:     &latest.Artifact{},
			expected:     "1caa15f7ce87536bddbac30a39768e8e3b212bf591f9b64926fa50c40b614c66",
		},
		{
			description:  "dependencies in different orders",
			dependencies: []string{"b", "a"},
			artifact:     &latest.Artifact{},
			expected:     "1caa15f7ce87536bddbac30a39768e8e3b212bf591f9b64926fa50c40b614c66",
		},
		{
			description: "no dependencies",
			artifact:    &latest.Artifact{},
			expected:    "53ebd85adc9b03923a7dacfe6002879af526ef6067d441419d6e62fb9bf608ab",
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
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashFunction, mockCacheHasher)
			t.Override(&artifactConfigFunction, fakeArtifactConfig)

			depLister := &stubDependencyLister{dependencies: test.dependencies}
			actual, err := getHashForArtifact(context.Background(), depLister, test.artifact)

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
	testutil.Run(t, "", func(t *testutil.T) {
		expected := "f5b610f4fea07461411b2ea0e2cddfd2ffc28d1baed49180f5d3acee5a18f9e7"

		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{"one": stringPointer("1"), "two": stringPointer("2")},
				},
			},
		}

		t.Override(&hashFunction, mockCacheHasher)
		t.Override(&artifactConfigFunction, fakeArtifactConfig)

		actual, err := getHashForArtifact(context.Background(), &stubDependencyLister{}, artifact)

		t.CheckNoError(err)
		t.CheckDeepEqual(expected, actual)

		// Change order of buildargs
		artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"two": stringPointer("2"), "one": stringPointer("1")}
		actual, err = getHashForArtifact(context.Background(), &stubDependencyLister{}, artifact)

		t.CheckNoError(err)
		t.CheckDeepEqual(expected, actual)

		// Change build args, get different hash
		artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"one": stringPointer("1")}
		actual, err = getHashForArtifact(context.Background(), &stubDependencyLister{}, artifact)

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

		depLister := &stubDependencyLister{dependencies: []string{"dep"}}
		hash1, err := getHashForArtifact(context.Background(), depLister, artifact)

		t.CheckNoError(err)

		// Make sure hash is different with a new env

		util.OSEnviron = func() []string {
			return []string{"FOO=baz"}
		}

		hash2, err := getHashForArtifact(context.Background(), depLister, artifact)

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
			depLister := &stubDependencyLister{dependencies: []string{tmpDir.Path(originalFile)}}

			oldHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{})
			t.CheckNoError(err)

			test.update(originalFile, tmpDir)
			if test.newFilename != "" {
				path = test.newFilename
			}

			depLister = &stubDependencyLister{dependencies: []string{tmpDir.Path(path)}}
			newHash, err := getHashForArtifact(context.Background(), depLister, &latest.Artifact{})

			t.CheckNoError(err)
			t.CheckDeepEqual(false, test.differentHash && oldHash == newHash)
			t.CheckDeepEqual(false, !test.differentHash && oldHash != newHash)
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
		t.Run(test.description, func(t *testing.T) {
			actual := retrieveBuildArgs(&latest.Artifact{
				ArtifactType: test.artifactType,
			})
			testutil.CheckDeepEqual(t, test.expected, actual)
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
			description: "build args with nil vlaue",
			buildArgs: map[string]*string{
				"one": nil,
				"two": nil,
			},
			expected: []string{"one", "two"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := convertBuildArgsToStringArray(test.buildArgs)
			testutil.CheckDeepEqual(t, test.expected, actual)
		})
	}
}

func stringPointer(s string) *string {
	return &s
}
