/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestEvalBuildArgs(t *testing.T) {
	tests := []struct {
		description string
		dockerfile  string
		mode        config.RunMode
		buildArgs   map[string]*string
		extra       map[string]*string
		expected    map[string]*string
	}{
		{
			description: "debug with exact build args",
			dockerfile: `ARG foo1
ARG foo2
ARG foo3
ARG SKAFFOLD_GO_GCFLAGS
FROM bar1`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Debug,
			expected: map[string]*string{
				"SKAFFOLD_GO_GCFLAGS": util.Ptr("all=-N -l"),
				"foo1":                util.Ptr("one"),
				"foo2":                util.Ptr("two"),
				"foo3":                util.Ptr("three"),
			},
		},
		{
			description: "debug with extra build args",
			dockerfile: `ARG foo1
ARG foo3
ARG SKAFFOLD_GO_GCFLAGS
FROM bar1`,
			mode: config.RunModes.Debug,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			expected: map[string]*string{
				"SKAFFOLD_GO_GCFLAGS": util.Ptr("all=-N -l"),
				"foo1":                util.Ptr("one"),
				"foo2":                util.Ptr("two"),
				"foo3":                util.Ptr("three"),
			},
		},
		{
			description: "debug with additional build args",
			dockerfile: `ARG foo1
ARG foo3
ARG foo4
ARG SKAFFOLD_GO_GCFLAGS
FROM bar1`,
			mode: config.RunModes.Debug,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			extra: map[string]*string{
				"foo4": util.Ptr("four"),
				"foo5": util.Ptr("five"),
			},
			expected: map[string]*string{
				"SKAFFOLD_GO_GCFLAGS": util.Ptr("all=-N -l"),
				"foo1":                util.Ptr("one"),
				"foo2":                util.Ptr("two"),
				"foo3":                util.Ptr("three"),
				"foo4":                util.Ptr("four"),
			},
		},
		{
			description: "debug with extra default args",
			dockerfile: `ARG foo1
ARG foo3
FROM bar1`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Debug,
			expected: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
		},
		{
			description: "debug with exact default args for multistage",
			dockerfile: `ARG SKAFFOLD_GO_GCFLAGS
ARG foo1
FROM bar1
ARG SKAFFOLD_GO_GCFLAGS
ARG foo2
FROM bar2
ARG foo3`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Debug,
			expected: map[string]*string{
				"SKAFFOLD_GO_GCFLAGS": util.Ptr("all=-N -l"),
				"foo1":                util.Ptr("one"),
				"foo2":                util.Ptr("two"),
				"foo3":                util.Ptr("three"),
			},
		},
		{
			description: "debug with extra default args for multistage",
			dockerfile: `ARG foo1
ARG SKAFFOLD_RUN_MODE
FROM bar1
ARG SKAFFOLD_GO_GCFLAGS
ARG foo2
FROM bar2
ARG foo3`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Debug,
			expected: map[string]*string{
				"SKAFFOLD_RUN_MODE":   util.Ptr("debug"),
				"SKAFFOLD_GO_GCFLAGS": util.Ptr("all=-N -l"),
				"foo1":                util.Ptr("one"),
				"foo2":                util.Ptr("two"),
				"foo3":                util.Ptr("three"),
			},
		},
		{
			description: "dev with exact build args",
			dockerfile: `ARG foo1
ARG foo2
ARG foo3
ARG SKAFFOLD_RUN_MODE
FROM bar1`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Dev,
			expected: map[string]*string{
				"SKAFFOLD_RUN_MODE": util.Ptr("dev"),
				"foo1":              util.Ptr("one"),
				"foo2":              util.Ptr("two"),
				"foo3":              util.Ptr("three"),
			},
		},
		{
			description: "dev with extra build args",
			dockerfile: `ARG foo1
ARG foo3
FROM bar1`,
			buildArgs: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
			mode: config.RunModes.Dev,
			expected: map[string]*string{
				"foo1": util.Ptr("one"),
				"foo2": util.Ptr("two"),
				"foo3": util.Ptr("three"),
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			artifact := &latest.DockerArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs:      test.buildArgs,
			}

			tmpDir := t.NewTempDir()
			tmpDir.Write("./Dockerfile", test.dockerfile)
			workspace := tmpDir.Path(".")

			actual, err := EvalBuildArgs(test.mode, workspace, artifact.DockerfilePath, artifact.BuildArgs, test.extra)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestCreateBuildArgsFromArtifacts(t *testing.T) {
	tests := []struct {
		description string
		r           ArtifactResolver
		deps        []*latest.ArtifactDependency
		args        map[string]*string
	}{
		{
			description: "can resolve artifacts",
			r:           mockArtifactResolver{m: map[string]string{"img1": "tag1", "img2": "tag2", "img3": "tag3", "img4": "tag4"}},
			deps:        []*latest.ArtifactDependency{{ImageName: "img3", Alias: "alias3"}, {ImageName: "img4", Alias: "alias4"}},
			args:        map[string]*string{"alias3": util.Ptr("tag3"), "alias4": util.Ptr("tag4")},
		},
		{
			description: "cannot resolve artifacts",
			r:           mockArtifactResolver{m: make(map[string]string)},
			args:        map[string]*string{"alias3": nil, "alias4": nil},
			deps:        []*latest.ArtifactDependency{{ImageName: "img3", Alias: "alias3"}, {ImageName: "img4", Alias: "alias4"}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := ResolveDependencyImages(test.deps, test.r, false)
			t.CheckDeepEqual(test.args, args)
		})
	}
}

func TestBuildArgTemplating(t *testing.T) {
	envs := map[string]string{
		"MY_KEY":    "abc123",
		"SO_SECRET": "do not say it",
	}

	args := map[string]*string{
		"MY_KEY":    util.Ptr("{{ .MY_KEY}}"),
		"SO_SECRET": util.Ptr("{{ .SO_SECRET}}"),
	}

	dockerFile := `
	ARG MY_KEY
	ARG SO_SECRET
	ARG foo3
	ARG SKAFFOLD_GO_GCFLAGS
	FROM bar1`

	testutil.Run(t, "test_templating_from_env_variables", func(t *testutil.T) {
		t.SetEnvs(envs)
		artifact := &latest.DockerArtifact{
			DockerfilePath: "Dockerfile",
			BuildArgs:      args,
		}

		tmpDir := t.NewTempDir()
		tmpDir.Write("./Dockerfile", dockerFile)
		workspace := tmpDir.Path(".")

		filledMap, err := EvalBuildArgs(config.RunModes.Dev, workspace, artifact.DockerfilePath, artifact.BuildArgs, nil)

		t.CheckNil(err)
		t.CheckMatches(*filledMap["MY_KEY"], "abc123")
		t.CheckMatches(*filledMap["SO_SECRET"], "do not say it")
	})
}

type mockArtifactResolver struct {
	m map[string]string
}

func (r mockArtifactResolver) GetImageTag(imageName string) (string, bool) {
	val, found := r.m[imageName]
	return val, found
}
