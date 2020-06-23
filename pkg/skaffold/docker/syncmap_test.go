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
	"path/filepath"
	"sort"
	"testing"

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

func TestSyncMap(t *testing.T) {
	tests := []struct {
		description    string
		dockerfile     string
		workspace      string
		ignore         string
		ignoreFilename string
		buildArgs      map[string]*string

		expected  map[string][]string
		badReader bool
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyServerGo,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "add dependency",
			dockerfile:  addNginx,
			workspace:   "docker",
			expected:    map[string][]string{"nginx.conf": {"/etc/nginx"}},
		},
		{
			description: "copy subdirectory",
			dockerfile:  copySubdirectory,
			workspace:   ".",
			expected:    map[string][]string{filepath.Join("docker", "nginx.conf"): {"/nginx.conf"}, filepath.Join("docker", "bar"): {"/bar"}},
		},
		{
			description: "copy file after workdir",
			dockerfile:  copyWorkdir,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/app/server.go"}},
		},
		{
			description: "copy file with absolute dest after workdir",
			dockerfile:  copyWorkdirAbsDest,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/bar"}},
		},
		{
			description: "two copy commands with same destination",
			dockerfile:  copySameDest,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/server.go"}, "test.conf": {"/test.conf"}},
		},
		{
			description: "copy file with absolute dest dir after workdir",
			dockerfile:  copyWorkdirAbsDestDir,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/bar/server.go"}},
		},
		{
			description: "wildcards",
			dockerfile:  wildcards,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/tmp/server.go"}, "worker.go": {"/tmp/worker.go"}},
		},
		{
			description: "wildcards after workdir",
			dockerfile:  wildcardsWorkdir,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/app/server.go"}, "worker.go": {"/app/worker.go"}},
		},
		{
			description: "wildcards matches none",
			dockerfile:  wildcardsMatchesNone,
			workspace:   ".",
			shouldErr:   true,
		},
		{
			description: "one wildcard matches none",
			dockerfile:  oneWilcardMatchesNone,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/tmp/server.go"}, "worker.go": {"/tmp/worker.go"}},
		},
		{
			description: "wildcard matches directory, flattens contents",
			dockerfile:  wildcardsMatchesDirectory,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/tmp/.dot"}, "Dockerfile": {"/tmp/Dockerfile"}, filepath.Join("docker", "bar"): {"/tmp/bar"}, filepath.Join("docker", "nginx.conf"): {"/tmp/nginx.conf"}, "file": {"/tmp/file"}, "server.go": {"/tmp/server.go"}, "test.conf": {"/tmp/test.conf"}, "worker.go": {"/tmp/worker.go"}},
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
			expected:    map[string][]string{},
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile1,
			expected:    map[string][]string{},
		},
		{
			description: "multistage dockerfile, only dependencies in the latest image are syncable",
			dockerfile:  multiStageDockerfile2,
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    map[string][]string{"test.conf": {"/etc/test1", "/etc/test2"}},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    map[string][]string{"bar": {"/quux"}},
		},
		{
			description: "workdir depends on env",
			dockerfile:  envWorkdirTest,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/bar/server.go"}},
		},
		{
			description: "copy depends on env",
			dockerfile:  envCopyTest,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/bar"}},
		},
		{
			description: "multiple env test",
			dockerfile:  multiEnvTest,
			workspace:   ".",
			expected:    map[string][]string{filepath.Join("docker", "nginx.conf"): {"/nginx.conf"}},
		},
		{
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    map[string][]string{"file": {"/file"}, "server.go": {"/server.go"}},
		},
		{
			description: "dockerignore test",
			dockerfile:  copyDirectory,
			ignore:      "bar\ndocker/*",
			workspace:   ".",
			expected:    map[string][]string{".dockerignore": {"/etc/.dockerignore"}, ".dot": {"/etc/.dot"}, "Dockerfile": {"/etc/Dockerfile"}, "file": {"/etc/file"}, "server.go": {"/etc/server.go"}, "test.conf": {"/etc/test.conf"}, "worker.go": {"/etc/worker.go"}},
		},
		{
			description: "dockerignore dockerfile",
			dockerfile:  copyServerGo,
			ignore:      "Dockerfile",
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "dockerignore with non canonical workspace",
			dockerfile:  contextDockerfile,
			workspace:   "docker/../docker",
			ignore:      "bar\ndocker/*",
			expected:    map[string][]string{".dockerignore": {"/files/.dockerignore"}, "Dockerfile": {"/files/Dockerfile"}, "nginx.conf": {"/etc/nginx", "/files/nginx.conf"}},
		},
		{
			description: "ignore none",
			dockerfile:  copyAll,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/.dot"}, "Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    map[string][]string{"Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    map[string][]string{"Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "bar\ndocker/*\n*.go",
			expected:    map[string][]string{".dockerignore": {"/etc/.dockerignore"}, ".dot": {"/etc/.dot"}, "Dockerfile": {"/etc/Dockerfile"}, "file": {"/etc/file"}, "test.conf": {"/etc/test.conf"}},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/onbuild/.dot"}, "Dockerfile": {"/onbuild/Dockerfile"}, "bar": {"/onbuild/bar"}, filepath.Join("docker", "bar"): {"/onbuild/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/onbuild/docker/nginx.conf"}, "file": {"/onbuild/file"}, "server.go": {"/onbuild/server.go"}, "test.conf": {"/onbuild/test.conf"}, "worker.go": {"/onbuild/worker.go"}},
		},
		{
			description: "onbuild with dockerignore",
			dockerfile:  onbuild,
			workspace:   ".",
			ignore:      "bar\ndocker/*",
			expected:    map[string][]string{".dockerignore": {"/onbuild/.dockerignore"}, ".dot": {"/onbuild/.dot"}, "Dockerfile": {"/onbuild/Dockerfile"}, "file": {"/onbuild/file"}, "server.go": {"/onbuild/server.go"}, "test.conf": {"/onbuild/test.conf"}, "worker.go": {"/onbuild/worker.go"}},
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
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.StringPtr("worker.go")},
			expected:    map[string][]string{"worker.go": {"/worker.go"}},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "build args with default value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "build args with redefined default value",
			dockerfile:  copyWorkerGoBuildArgRedefinedDefaultValue,
			workspace:   ".",
			expected:    map[string][]string{"worker.go": {"/worker.go"}},
		},
		{
			description: "build args all defined a the top",
			dockerfile:  copyServerGoBuildArgsAtTheTop,
			workspace:   ".",
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "override default build arg",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("worker.go")},
			expected:    map[string][]string{"worker.go": {"/worker.go"}},
		},
		{
			description: "ignore build arg and use default arg value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": nil},
			expected:    map[string][]string{"server.go": {"/server.go"}},
		},
		{
			description: "from base stage",
			dockerfile:  fromStage,
			workspace:   ".",
			expected:    map[string][]string{},
		},
		{
			description: "from base stage, ignoring case",
			dockerfile:  fromStageIgnoreCase,
			workspace:   ".",
			expected:    map[string][]string{},
		},
		{
			description: "from scratch",
			dockerfile:  fromScratch,
			workspace:   ".",
			expected:    map[string][]string{"file": {"/etc/file"}},
		},
		{
			description: "from scratch, ignoring case",
			dockerfile:  fromScratchUppercase,
			workspace:   ".",
			expected:    map[string][]string{"file": {"/etc/file"}},
		},
		{
			description: "case sensitive",
			dockerfile:  fromImageCaseSensitive,
			workspace:   ".",
			expected:    map[string][]string{"file": {"/etc/file"}},
		},
		{
			description:    "find specific dockerignore",
			dockerfile:     copyDirectory,
			ignore:         "bar\ndocker/*",
			ignoreFilename: "Dockerfile.dockerignore",
			workspace:      ".",
			expected:       map[string][]string{"Dockerfile.dockerignore": {"/etc/Dockerfile.dockerignore"}, ".dot": {"/etc/.dot"}, "Dockerfile": {"/etc/Dockerfile"}, "file": {"/etc/file"}, "server.go": {"/etc/server.go"}, "test.conf": {"/etc/test.conf"}, "worker.go": {"/etc/worker.go"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)

			tmpDir := t.NewTempDir().
				Touch("docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file", ".dot")

			if !test.badReader {
				tmpDir.Write(test.workspace+"/Dockerfile", test.dockerfile)
			}

			if test.ignore != "" {
				ignoreFilename := ".dockerignore"
				if test.ignoreFilename != "" {
					ignoreFilename = test.ignoreFilename
				}
				tmpDir.Write(filepath.Join(test.workspace, ignoreFilename), test.ignore)
			}

			workspace := tmpDir.Path(test.workspace)
			deps, err := SyncMap(workspace, "Dockerfile", test.buildArgs, nil)

			// destinations are not sorted, but for the test assertion they must be
			for _, dsts := range deps {
				sort.Strings(dsts)
			}

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, deps)
		})
	}
}

func TestSyncMap_deterministicOverwrite(t *testing.T) {
	const (
		simpleOverwrite = `
FROM ubuntu:14.04
ADD subfolder/bar bar
COPY baz /bar
ADD bar .
COPY foo bar
`
		implicitOverwrite1 = `
FROM ubuntu:14.04
COPY subfolder .
ADD baz bar
`
		implicitOverwrite2 = `
FROM ubuntu:14.04
ADD baz bar
COPY subfolder .
`
		implicitOverwrite3 = `
FROM ubuntu:14.04
COPY . .
COPY subfolder .
`
		implicitOverwrite4 = `
FROM ubuntu:14.04
COPY subfolder .
COPY . .
`
		implicitOverwrite5 = `
FROM ubuntu:14.04
ADD * .
`
		repeat = 3
	)

	tests := []struct {
		name       string
		dockerfile string
		expected   map[string][]string
	}{
		{
			name:       "simple overwrite",
			dockerfile: simpleOverwrite,
			expected:   map[string][]string{"foo": {"/bar"}},
		},
		{
			name:       "explicit overwrite by `baz`",
			dockerfile: implicitOverwrite1,
			expected:   map[string][]string{"baz": {"/bar"}},
		},
		{
			name:       "implicit overwrite by subfolder `bar`",
			dockerfile: implicitOverwrite2,
			expected:   map[string][]string{filepath.Join("subfolder", "bar"): {"/bar"}},
		},
		{
			name:       "implicit overwrite by subfolder of implicitly added `bar`",
			dockerfile: implicitOverwrite3,
			expected:   map[string][]string{filepath.Join("subfolder", "bar"): {"/bar", "/subfolder/bar"}, "baz": {"/baz"}, "foo": {"/foo"}},
		},
		{
			name:       "implicit overwrite by root folder of implicit subfolder file `bar`",
			dockerfile: implicitOverwrite4,
			expected:   map[string][]string{filepath.Join("subfolder", "bar"): {"/subfolder/bar"}, "bar": {"/bar"}, "baz": {"/baz"}, "foo": {"/foo"}},
		},
		{
			name:       "implicit overwrite by glob flattening according to alphabetical ordering (later wins)",
			dockerfile: implicitOverwrite5,
			expected:   map[string][]string{filepath.Join("subfolder", "bar"): {"/bar"}, "baz": {"/baz"}, "foo": {"/foo"}},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)

			tmpDir := t.NewTempDir().
				Touch("subfolder/bar", "baz", "foo", "bar", "ignored/bar").
				Write(".dockerignore", "Dockerfile\n.dockerignore\nignored/bar").
				Write("Dockerfile", test.dockerfile)

			for i := 0; i < repeat; i++ {
				deps, err := SyncMap(tmpDir.Root(), "Dockerfile", nil, nil)

				// destinations are not sorted, but for the test assertion they must be
				for _, dsts := range deps {
					sort.Strings(dsts)
				}

				t.CheckNoError(err)
				t.CheckDeepEqual(test.expected, deps)
			}
		})
	}
}
