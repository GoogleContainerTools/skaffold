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

package bazel

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildBazel(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Mkdir("bin").Chdir()
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("bazel build //:app.tar --color=no").AndRunOut(
			"bazel cquery //:app.tar --output starlark --starlark:expr target.files.to_list()[0].path",
			"bin/app.tar").AndRunOut("bazel info execution_root", ""))
		testutil.CreateFakeImageTar("bazel:app", "bin/app.tar")

		artifact := &latest.Artifact{
			Workspace: ".",
			ArtifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: "//:app.tar",
				},
			},
		}

		builder := NewArtifactBuilder(fakeLocalDaemon(), &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", platform.Matcher{})

		t.CheckNoError(err)
	})
}

func TestBazelTarPathPrependExecutionRoot(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("bazel build //:app.tar --color=no").AndRunOut(
			"bazel cquery //:app.tar --output starlark --starlark:expr target.files.to_list()[0].path",
			"app.tar").AndRunOut("bazel info execution_root", ".."))
		testutil.CreateFakeImageTar("bazel:app", "../app.tar")

		artifact := &latest.Artifact{
			Workspace: "..",
			ArtifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: "//:app.tar",
				},
			},
		}

		builder := NewArtifactBuilder(fakeLocalDaemon(), &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", platform.Matcher{})

		t.CheckNoError(err)
	})
}

func TestBazelAddPlatforms(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("bazel build //:app.tar --platforms=//platforms:linux-x86_64 --color=no").AndRunOut(
			"bazel cquery //:app.tar --output starlark --starlark:expr target.files.to_list()[0].path",
			"app.tar").AndRunOut("bazel info execution_root", ".."))
		testutil.CreateFakeImageTar("bazel:app", "../app.tar")

		artifact := &latest.Artifact{
			Workspace: "..",
			ArtifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: "//:app.tar",
					PlatformMappings: []latest.BazelPlatformMapping{
						{
							Platform:            "linux/amd64",
							BazelPlatformTarget: "//platforms:linux-x86_64",
						},
					},
				},
			},
		}

		testPlatform := platform.Matcher{Platforms: []specs.Platform{{Architecture: "amd64", OS: "linux"}}}

		builder := NewArtifactBuilder(fakeLocalDaemon(), &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", testPlatform)

		t.CheckNoError(err)
	})
}

func TestBuildBazelFailInvalidTarget(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: "//:invalid-target",
				},
			},
		}

		builder := NewArtifactBuilder(nil, &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", platform.Matcher{})

		t.CheckErrorContains("the bazel build target should end with .tar", err)
	})
}

func TestBazelTarPath(t *testing.T) {
	testutil.Run(t, "EmptyExecutionRoot", func(t *testutil.T) {
		osSpecificPath := filepath.Join("absolute", "path", "bin")
		t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
			"bazel cquery //:skaffold_example.tar --output starlark --starlark:expr target.files.to_list()[0].path --arg1 --arg2",
			fmt.Sprintf("%s\n", osSpecificPath),
		).AndRunOut("bazel info execution_root", ""))

		bazelBin, err := bazelTarPath(context.Background(), ".", &latest.BazelArtifact{
			BuildArgs:   []string{"--arg1", "--arg2"},
			BuildTarget: "//:skaffold_example.tar",
		})

		t.CheckNoError(err)
		t.CheckDeepEqual(osSpecificPath, bazelBin)
	})
	testutil.Run(t, "AbsoluteExecutionRoot", func(t *testutil.T) {
		osSpecificPath := filepath.Join("var", "tmp", "bazel-execution-roots", "abcdefg", "execroot", "workspace_name")
		t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
			"bazel cquery //:skaffold_example.tar --output starlark --starlark:expr target.files.to_list()[0].path --arg1 --arg2",
			"bazel-bin/darwin-fastbuild-ST-confighash/path/to/bin\n",
		).AndRunOut("bazel info execution_root", osSpecificPath))

		bazelBin, err := bazelTarPath(context.Background(), ".", &latest.BazelArtifact{
			BuildArgs:   []string{"--arg1", "--arg2"},
			BuildTarget: "//:skaffold_example.tar",
		})

		t.CheckNoError(err)
		expected := filepath.Join(osSpecificPath, "bazel-bin", "darwin-fastbuild-ST-confighash", "path", "to", "bin")
		t.CheckDeepEqual(expected, bazelBin)
	})
}

func fakeLocalDaemon() docker.LocalDaemon {
	return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil)
}

type mockConfig struct {
	docker.Config
}

func (c *mockConfig) GetInsecureRegistries() map[string]bool { return nil }
