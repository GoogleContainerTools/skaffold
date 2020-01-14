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

package docker

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const copySubdirectory = `
FROM ubuntu:14.04
COPY docker .
`

const copyWorkdir = `
FROM ubuntu:14.04
WORKDIR /app
COPY server.go .
`

const copyWorkdirAbsDest = `
FROM ubuntu:14.04
WORKDIR /app
COPY server.go /bar
`

const copySameDest = `
FROM ubuntu:14.04
COPY server.go .
COPY test.conf .
`

const copyWorkdirAbsDestDir = `
FROM ubuntu:14.04
WORKDIR /app
COPY server.go /bar/
`

const wildcardsWorkdir = `
FROM nginx
WORKDIR /app/
ADD *.go ./
`

const wildcardsMatchesDirectory = `
FROM nginx
ADD * /tmp/
`

const envWorkdirTest = `
FROM busybox
ENV foo bar
WORKDIR ${foo}
COPY server.go .
`

const envCopyTest = `
FROM busybox
ENV foo bar
COPY server.go ${foo}
`

const multipleSubFolders = `
FROM busybox
COPY foo/bar/qix/server.go .
`

func TestSyncRules(t *testing.T) {
	tests := []struct {
		description string
		dockerfile  string
		workspace   string
		ignore      string
		buildArgs   map[string]*string

		expected  []*latest.SyncRule
		badReader bool
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyServerGo,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "add dependency",
			dockerfile:  addNginx,
			workspace:   "docker",
			expected:    []*latest.SyncRule{{Src: "nginx.conf", Dest: "/etc/nginx"}},
		},
		{
			description: "copy subdirectory",
			dockerfile:  copySubdirectory,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "docker/**", Dest: "/", Strip: "docker"}},
		},
		{
			description: "copy file after workdir",
			dockerfile:  copyWorkdir,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/app"}},
		},
		{
			description: "copy file with absolute dest after workdir",
			dockerfile:  copyWorkdirAbsDest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/bar"}},
		},
		{
			description: "two copy commands with same destination",
			dockerfile:  copySameDest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}, {Src: "test.conf", Dest: "/"}},
		},
		{
			description: "copy file with absolute dest dir after workdir",
			dockerfile:  copyWorkdirAbsDestDir,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/bar"}},
		},
		{
			description: "wildcards",
			dockerfile:  wildcards,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "*.go", Dest: "/tmp"}},
		},
		{
			description: "wildcards after workdir",
			dockerfile:  wildcardsWorkdir,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "*.go", Dest: "/app"}},
		},
		{
			description: "wildcards matches none",
			dockerfile:  wildcardsMatchesNone,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "*.none", Dest: "/tmp"}},
		},
		{
			description: "one wildcard matches none",
			dockerfile:  oneWilcardMatchesNone,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "*.go", Dest: "/tmp"}, {Src: "*.none", Dest: "/tmp"}},
		},
		{
			description: "wildcard matches directory",
			dockerfile:  wildcardsMatchesDirectory,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "*", Dest: "/tmp"}},
		},
		{
			description: "bad read",
			badReader:   true,
			shouldErr:   true,
		},
		{
			// https://github.com/GoogleContainerTools/skaffold/issues/158
			description: "no dependencies on remote files",
			dockerfile:  remoteFileAdd,
			expected:    nil,
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile1,
			expected:    nil,
		},
		{
			description: "multistage dockerfile, only dependencies in the latest image are syncable",
			dockerfile:  multiStageDockerfile2,
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			expected:    []*latest.SyncRule{{Src: "test.conf", Dest: "/etc/test1"}, {Src: "test.conf", Dest: "/etc/test2"}},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "bar", Dest: "/quux"}},
		},
		{
			description: "workdir depends on env",
			dockerfile:  envWorkdirTest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/bar"}},
		},
		{
			description: "copy depends on env",
			dockerfile:  envCopyTest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/bar"}},
		},
		{
			description: "multiple env test",
			dockerfile:  multiEnvTest,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "docker/nginx.conf", Dest: "/", Strip: "docker"}},
		},
		{
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}, {Src: "file", Dest: "/"}},
		},
		{
			description: "dockerignore test",
			dockerfile:  copyDirectory,
			ignore:      "bar\ndocker/*",
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/etc"}, {Src: "./file", Dest: "/etc/file"}},
		},
		{
			description: "dockerignore dockerfile",
			dockerfile:  copyServerGo,
			ignore:      "Dockerfile",
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "dockerignore with non canonical workspace",
			dockerfile:  contextDockerfile,
			workspace:   "docker/../docker",
			ignore:      "bar\ndocker/*",
			expected:    []*latest.SyncRule{{Src: "nginx.conf", Dest: "/etc/nginx"}, {Src: "**", Dest: "/files"}},
		},
		{
			description: "ignore none",
			dockerfile:  copyAll,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/"}},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/"}},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/"}},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "bar\ndocker/*\n*.go",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/etc"}, {Src: "./file", Dest: "/etc/file"}},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/onbuild"}},
		},
		{
			description: "onbuild with dockerignore",
			dockerfile:  onbuild,
			workspace:   ".",
			ignore:      "bar\ndocker/*",
			expected:    []*latest.SyncRule{{Src: "**", Dest: "/onbuild"}},
		},
		{
			description: "base image not found",
			dockerfile:  baseImageNotFound,
			workspace:   ".",
			shouldErr:   true,
		},
		{
			description: "build args",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.StringPtr("worker.go")},
			expected:    []*latest.SyncRule{{Src: "worker.go", Dest: "/"}},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "build args with default value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "build args with redefined default value",
			dockerfile:  copyWorkerGoBuildArgRedefinedDefaultValue,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "worker.go", Dest: "/"}},
		},
		{
			description: "build args all defined a the top",
			dockerfile:  copyServerGoBuildArgsAtTheTop,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "override default build arg",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("worker.go")},
			expected:    []*latest.SyncRule{{Src: "worker.go", Dest: "/"}},
		},
		{
			description: "ignore build arg and use default arg value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": nil},
			expected:    []*latest.SyncRule{{Src: "server.go", Dest: "/"}},
		},
		{
			description: "from base stage",
			dockerfile:  fromStage,
			workspace:   ".",
			expected:    nil,
		},
		{
			description: "from base stage, ignoring case",
			dockerfile:  fromStageIgnoreCase,
			workspace:   ".",
			expected:    nil,
		},
		{
			description: "from scratch",
			dockerfile:  fromScratch,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "./file", Dest: "/etc/file"}},
		},
		{
			description: "from scratch, ignoring case",
			dockerfile:  fromScratchUppercase,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "./file", Dest: "/etc/file"}},
		},
		{
			description: "case sensitive",
			dockerfile:  fromImageCaseSensitive,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "./file", Dest: "/etc/file"}},
		},
		{
			description: "multiple sub-folders",
			dockerfile:  multipleSubFolders,
			workspace:   ".",
			expected:    []*latest.SyncRule{{Src: "foo/bar/qix/server.go", Dest: "/", Strip: "foo/bar/qix"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)

			tmpDir := t.NewTempDir().
				Touch("docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file", ".dot", "foo/bar/qix/server.go")

			if !test.badReader {
				tmpDir.Write(test.workspace+"/Dockerfile", test.dockerfile)
			}
			if test.ignore != "" {
				tmpDir.Write(test.workspace+"/.dockerignore", test.ignore)
			}

			workspace := tmpDir.Path(test.workspace)
			deps, err := SyncRules(workspace, "Dockerfile", test.buildArgs, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, deps)
		})
	}
}
