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

const Dockerfile = "Dockerfile"

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
			mode:     config.RunModes.Dev,
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
			mode:     config.RunModes.Dev,
			expected: "f3f710a4ec1d1bfb2a9b8ef2b4b7cc5f254102d17095a71872821b396953a4ce",
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
			expected: "d99ab295a682897269b4db0fe7c136ea1ecd542150fa224ee03155b4e3e995d9",
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
			expected: "c3a878f799b2a6532db71683a09771af4f9d20ef5884c57642a272934e5c93ea",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&fileHasherFunc, mockCacheHasher)
			t.Override(&artifactConfigFunc, fakeArtifactConfig)
			if test.artifact.DockerArtifact != nil {
				tmpDir := t.NewTempDir()
				tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
				test.artifact.Workspace = tmpDir.Path(".")
				test.artifact.DockerArtifact.DockerfilePath = Dockerfile
			}

			depLister := stubDependencyLister(test.dependencies)
			actual, err := newArtifactHasher(nil, depLister, test.mode).hash(context.Background(), test.artifact)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestGetHashForArtifactWithDependencies(t *testing.T) {
	tests := []struct {
		description string
		artifacts   []*latest.Artifact
		fileDeps    map[string][]string // keyed on artifact ImageName, returns a list of mock file dependencies.
		mode        config.RunMode
		expected    string
	}{
		{
			description: "hash for artifact with two dependencies",
			artifacts: []*latest.Artifact{
				{ImageName: "img1", Dependencies: []*latest.ArtifactDependency{{ImageName: "img2"}, {ImageName: "img3"}}},
				{ImageName: "img2", Dependencies: []*latest.ArtifactDependency{{ImageName: "img4"}}, ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target2"}}},
				{ImageName: "img3", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target3"}}},
				{ImageName: "img4", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target4"}}},
			},
			fileDeps: map[string][]string{"img1": {"a"}, "img2": {"b"}, "img3": {"c"}, "img4": {"d"}},
			mode:     config.RunModes.Dev,
			expected: "ccd159a9a50853f89ab6784530b58d658a0b349c92828eba335f1074f9a63bb3",
		},
		{
			description: "hash for artifact with two dependencies in different order",
			artifacts: []*latest.Artifact{
				{ImageName: "img1", Dependencies: []*latest.ArtifactDependency{{ImageName: "img3"}, {ImageName: "img2"}}},
				{ImageName: "img2", Dependencies: []*latest.ArtifactDependency{{ImageName: "img4"}}, ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target2"}}},
				{ImageName: "img3", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target3"}}},
				{ImageName: "img4", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target4"}}},
			},
			fileDeps: map[string][]string{"img1": {"a"}, "img2": {"b"}, "img3": {"c"}, "img4": {"d"}},
			mode:     config.RunModes.Dev,
			expected: "ccd159a9a50853f89ab6784530b58d658a0b349c92828eba335f1074f9a63bb3",
		},
		{
			description: "hash for artifact with different dependencies (img4 builder changed)",
			artifacts: []*latest.Artifact{
				{ImageName: "img1", Dependencies: []*latest.ArtifactDependency{{ImageName: "img2"}, {ImageName: "img3"}}},
				{ImageName: "img2", Dependencies: []*latest.ArtifactDependency{{ImageName: "img4"}}, ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target2"}}},
				{ImageName: "img3", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target3"}}},
				{ImageName: "img4", ArtifactType: latest.ArtifactType{BuildpackArtifact: &latest.BuildpackArtifact{Builder: "builder"}}},
			},
			fileDeps: map[string][]string{"img1": {"a"}, "img2": {"b"}, "img3": {"c"}, "img4": {"d"}},
			mode:     config.RunModes.Dev,
			expected: "26defaa1291289f40b756b83824f0549a3a9c03cca5471bd268f0ac6e499aba6",
		},
		{
			description: "hash for artifact with different dependencies (img4 files changed)",
			artifacts: []*latest.Artifact{
				{ImageName: "img1", Dependencies: []*latest.ArtifactDependency{{ImageName: "img2"}, {ImageName: "img3"}}},
				{ImageName: "img2", Dependencies: []*latest.ArtifactDependency{{ImageName: "img4"}}, ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target2"}}},
				{ImageName: "img3", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{Target: "target3"}}},
				{ImageName: "img4", ArtifactType: latest.ArtifactType{BuildpackArtifact: &latest.BuildpackArtifact{}}},
			},
			fileDeps: map[string][]string{"img1": {"a"}, "img2": {"b"}, "img3": {"c"}, "img4": {"e"}},
			mode:     config.RunModes.Dev,
			expected: "bab56a88d483fa97ae072b027a46681177628156839b7e390842e6243b1ac6aa",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&fileHasherFunc, mockCacheHasher)
			t.Override(&artifactConfigFunc, fakeArtifactConfig)
			g := build.ToArtifactGraph(test.artifacts)

			for _, a := range test.artifacts {
				if a.DockerArtifact != nil {
					tmpDir := t.NewTempDir()
					tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
					a.Workspace = tmpDir.Path(".")
					a.DockerArtifact.DockerfilePath = Dockerfile
				}
			}

			depLister := func(_ context.Context, a *latest.Artifact) ([]string, error) {
				return test.fileDeps[a.ImageName], nil
			}

			actual, err := newArtifactHasher(g, depLister, test.mode).hash(context.Background(), test.artifacts[0])

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
			expected: "a8544410acafce64325abfffcb21e75efdcd575bd9f8d3be2a516125ec547651",
		},
		{
			mode:     config.RunModes.Dev,
			expected: "f5b610f4fea07461411b2ea0e2cddfd2ffc28d1baed49180f5d3acee5a18f9e7",
		},
	}
	for _, test := range tests {
		testutil.Run(t, "", func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
			artifact := &latest.Artifact{
				Workspace: tmpDir.Path("."),
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: Dockerfile,
						BuildArgs:      map[string]*string{"one": util.StringPtr("1"), "two": util.StringPtr("2")},
					},
				},
			}
			t.Override(&fileHasherFunc, mockCacheHasher)
			t.Override(&artifactConfigFunc, fakeArtifactConfig)
			actual, err := newArtifactHasher(nil, stubDependencyLister(nil), test.mode).hash(context.Background(), artifact)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)

			// Change order of buildargs
			artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"two": util.StringPtr("2"), "one": util.StringPtr("1")}
			actual, err = newArtifactHasher(nil, stubDependencyLister(nil), test.mode).hash(context.Background(), artifact)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)

			// Change build args, get different hash
			artifact.ArtifactType.DockerArtifact.BuildArgs = map[string]*string{"one": util.StringPtr("1")}
			actual, err = newArtifactHasher(nil, stubDependencyLister(nil), test.mode).hash(context.Background(), artifact)

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
		tmpDir := t.NewTempDir()
		tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
		artifact := &latest.Artifact{
			Workspace: tmpDir.Path("."),
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs:      map[string]*string{"env": util.StringPtr("${{.FOO}}")},
					DockerfilePath: Dockerfile,
				},
			},
		}

		t.Override(&fileHasherFunc, mockCacheHasher)
		t.Override(&artifactConfigFunc, fakeArtifactConfig)

		depLister := stubDependencyLister([]string{"dep"})
		hash1, err := newArtifactHasher(nil, depLister, config.RunModes.Build).hash(context.Background(), artifact)

		t.CheckNoError(err)

		// Make sure hash is different with a new env

		util.OSEnviron = func() []string {
			return []string{"FOO=baz"}
		}

		hash2, err := newArtifactHasher(nil, depLister, config.RunModes.Build).hash(context.Background(), artifact)

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

			oldHash, err := newArtifactHasher(nil, depLister, config.RunModes.Build).hash(context.Background(), &latest.Artifact{})
			t.CheckNoError(err)

			test.update(originalFile, tmpDir)
			if test.newFilename != "" {
				path = test.newFilename
			}

			depLister = stubDependencyLister([]string{tmpDir.Path(path)})
			newHash, err := newArtifactHasher(nil, depLister, config.RunModes.Build).hash(context.Background(), &latest.Artifact{})

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
						"foo": util.StringPtr("bar"),
					},
				},
			},
			mode:     config.RunModes.Dev,
			expected: []string{"foo=bar"},
		}, {
			description: "docker artifact with build args for debug",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					BuildArgs: map[string]*string{
						"foo": util.StringPtr("bar"),
					},
				},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"SKAFFOLD_GO_GCFLAGS=all=-N -l", "foo=bar"},
		}, {
			description: "docker artifact without build args for debug",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"SKAFFOLD_GO_GCFLAGS=all=-N -l"},
		}, {
			description: "docker artifact without build args for dev",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{},
			},
			mode: config.RunModes.Dev,
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
			expected: []string{"foo=bar"},
		}, {
			description: "buildpacks artifact without env for dev",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{},
			},
			mode: config.RunModes.Dev,
		}, {
			description: "buildpacks artifact with env for debug",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{
					Env: []string{"foo=bar"},
				},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"GOOGLE_GOGCFLAGS=all=-N -l", "foo=bar"},
		}, {
			description: "buildpacks artifact without env for debug",
			artifactType: latest.ArtifactType{
				BuildpackArtifact: &latest.BuildpackArtifact{},
			},
			mode:     config.RunModes.Debug,
			expected: []string{"GOOGLE_GOGCFLAGS=all=-N -l"},
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
			a := &latest.Artifact{
				ArtifactType: test.artifactType,
			}
			if test.artifactType.DockerArtifact != nil {
				tmpDir := t.NewTempDir()
				tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
				a.Workspace = tmpDir.Path(".")
				a.ArtifactType.DockerArtifact.DockerfilePath = Dockerfile
			}
			if test.artifactType.KanikoArtifact != nil {
				tmpDir := t.NewTempDir()
				tmpDir.Write("./Dockerfile", "ARG SKAFFOLD_GO_GCFLAGS\nFROM foo")
				a.Workspace = tmpDir.Path(".")
				a.ArtifactType.KanikoArtifact.DockerfilePath = Dockerfile
			}
			actual, err := hashBuildArgs(a, test.mode)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
