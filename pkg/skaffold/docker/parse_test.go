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
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const copyServerGo = `
FROM ubuntu:14.04
COPY server.go .
CMD server.go
`

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

const addNginx = `
FROM nginx
ADD nginx.conf /etc/nginx
CMD nginx
`

const multiCopy = `
FROM nginx
ADD test.conf /etc/test1
COPY test.conf /etc/test2
CMD nginx
`

const wildcards = `
FROM nginx
ADD *.go /tmp/
`

const wildcardsWorkdir = `
FROM nginx
WORKDIR /app/
ADD *.go ./
`

const wildcardsMatchesNone = `
FROM nginx
ADD *.none /tmp/
`

const wildcardsMatchesDirectory = `
FROM nginx
ADD * /tmp/
`

const oneWilcardMatchesNone = `
FROM nginx
ADD *.go *.none /tmp/
`

const multiStageDockerfile = `
FROM golang:1.9.2
WORKDIR /go/src/github.com/r2d4/leeroy/
COPY worker.go .
RUN go build -o worker .

FROM gcr.io/distroless/base
WORKDIR /root/
COPY --from=0 /go/src/github.com/r2d4/leeroy ./
`

const envTest = `
FROM busybox
ENV foo bar
COPY $foo /quux # COPY bar /quux
`

const envWorkdirTest = `
FROM busybox
ENV foo bar
WORKDIR ${foo}
COPY server.go .
`

const multiEnvTest = `
FROM busybox
ENV baz=bar \
    foo=docker
COPY $foo/nginx.conf . # COPY docker/nginx.conf .
`

const copyDirectory = `
FROM nginx
ADD . /etc/
COPY ./file /etc/file
`

const multiFileCopy = `
FROM ubuntu:14.04
COPY server.go file ./
`

const remoteFileAdd = `
FROM ubuntu:14.04
ADD https://example.com/test /test
`

const contextDockerfile = `
FROM nginx
ADD nginx.conf /etc/nginx
COPY . /files
`

// This has an ONBUILD instruction of "COPY . /onbuild"
const onbuild = `
FROM golang:onbuild
`

const onbuildError = `
FROM noimage:latest
ADD ./file /etc/file
`

const copyServerGoBuildArg = `
FROM ubuntu:14.04
ARG FOO
COPY $FOO .
`

const copyWorkerGoBuildArgSamePrefix = `
FROM ubuntu:14.04
ARG FOO=server.go
ARG FOO2
COPY $FOO2 .
`

const copyServerGoBuildArgCurlyBraces = `
FROM ubuntu:14.04
ARG FOO
COPY ${FOO} .
`

const copyServerGoBuildArgExtraWhitespace = `
FROM ubuntu:14.04
ARG  FOO
COPY $FOO .
`

const copyServerGoBuildArgDefaultValue = `
FROM ubuntu:14.04
ARG FOO=server.go
COPY $FOO .
`

const copyWorkerGoBuildArgRedefinedDefaultValue = `
FROM ubuntu:14.04
ARG FOO=server.go
ARG FOO=worker.go
COPY $FOO .
`

const copyServerGoBuildArgsAtTheTop = `
FROM ubuntu:14.04
ARG FOO=server.go
ARG FOO2=ignored
ARG FOO3=ignored
COPY $FOO .
`

const fromStage = `
FROM ubuntu:14.04 as base
FROM base as dist
FROM dist as prod
`

const fromStageIgnoreCase = `
FROM ubuntu:14.04 as BASE
FROM base as dist
FROM DIST as prod
`

const copyAll = `
FROM nginx
COPY . /
`

const fromScratch = `
FROM scratch
ADD ./file /etc/file
`

const fromScratchUppercase = `
FROM SCRATCH
ADD ./file /etc/file
`

const fromImageCaseSensitive = `
FROM jboss/wildfly:14.0.1.Final
ADD ./file /etc/file
`

type fakeImageFetcher struct {
	fetched []string
}

func (f *fakeImageFetcher) fetch(image string) (*v1.ConfigFile, error) {
	f.fetched = append(f.fetched, image)

	switch image {
	case "ubuntu:14.04", "busybox", "nginx", "golang:1.9.2", "jboss/wildfly:14.0.1.Final":
		return &v1.ConfigFile{}, nil
	case "golang:onbuild":
		return &v1.ConfigFile{
			Config: v1.Config{
				OnBuild: []string{
					"COPY . /onbuild",
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("no image found for %s", image)
}

func TestGetDependencies(t *testing.T) {
	var tests = []struct {
		description string
		dockerfile  string
		workspace   string
		ignore      string
		buildArgs   map[string]*string

		expected  map[string][]string
		fetched   []string
		badReader bool
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyServerGo,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "add dependency",
			dockerfile:  addNginx,
			workspace:   "docker",
			expected:    map[string][]string{"Dockerfile": nil, "nginx.conf": {"/etc/nginx"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "copy subdirectory",
			dockerfile:  copySubdirectory,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, filepath.Join("docker", "nginx.conf"): {"/nginx.conf"}, filepath.Join("docker", "bar"): {"/bar"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "copy file after workdir",
			dockerfile:  copyWorkdir,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/app/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "copy file with absolute dest after workdir",
			dockerfile:  copyWorkdirAbsDest,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/bar"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "two copy commands with same destination",
			dockerfile:  copySameDest,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "copy file with absolute dest dir after workdir",
			dockerfile:  copyWorkdirAbsDestDir,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/bar/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "wildcards",
			dockerfile:  wildcards,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/tmp/server.go"}, "worker.go": {"/tmp/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "wildcards after workdir",
			dockerfile:  wildcardsWorkdir,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/app/server.go"}, "worker.go": {"/app/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "wildcards matches none",
			dockerfile:  wildcardsMatchesNone,
			workspace:   ".",
			fetched:     []string{"nginx"},
			shouldErr:   true,
		},
		{
			description: "one wilcard matches none",
			dockerfile:  oneWilcardMatchesNone,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/tmp/server.go"}, "worker.go": {"/tmp/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "wilcard matches directory, flattens contents",
			dockerfile:  wildcardsMatchesDirectory,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/tmp/.dot"}, "Dockerfile": {"/tmp/Dockerfile"}, "bar": {"/tmp/bar"}, filepath.Join("docker", "bar"): {"/tmp/bar"}, filepath.Join("docker", "nginx.conf"): {"/tmp/nginx.conf"}, "file": {"/tmp/file"}, "server.go": {"/tmp/server.go"}, "test.conf": {"/tmp/test.conf"}, "worker.go": {"/tmp/worker.go"}},
			fetched:     []string{"nginx"},
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
			expected:    map[string][]string{"Dockerfile": nil},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile,
			workspace:   "",
			// TODO(corneliusweig) destinations are not resolved across stages in multistage dockerfiles. Is there a use-case for that?
			expected: map[string][]string{"Dockerfile": nil, "worker.go": {"/go/src/github.com/r2d4/leeroy/worker.go"}},
			fetched:  []string{"golang:1.9.2", "gcr.io/distroless/base"},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "test.conf": {"/etc/test1", "/etc/test2"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "bar": {"/quux"}},
			fetched:     []string{"busybox"},
		},
		{
			description: "workdir depends on env",
			dockerfile:  envWorkdirTest,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/bar/server.go"}},
			fetched:     []string{"busybox"},
		},
		{
			description: "multiple env test",
			dockerfile:  multiEnvTest,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, filepath.Join("docker", "nginx.conf"): {"/nginx.conf"}},
			fetched:     []string{"busybox"},
		},
		{
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "file": {"/file"}, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "dockerignore test",
			dockerfile:  copyDirectory,
			ignore:      "bar\ndocker/*",
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/etc/.dot"}, "Dockerfile": {"/etc/Dockerfile"}, "file": {"/etc/file", "/etc/file"}, "server.go": {"/etc/server.go"}, "test.conf": {"/etc/test.conf"}, "worker.go": {"/etc/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "dockerignore dockerfile",
			dockerfile:  copyServerGo,
			ignore:      "Dockerfile",
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "dockerignore with non canonical workspace",
			dockerfile:  contextDockerfile,
			workspace:   "docker/../docker",
			ignore:      "bar\ndocker/*",
			expected:    map[string][]string{"Dockerfile": {"/files/Dockerfile"}, "nginx.conf": {"/etc/nginx", "/files/nginx.conf"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore none",
			dockerfile:  copyAll,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/.dot"}, "Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    map[string][]string{"Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    map[string][]string{"Dockerfile": {"/Dockerfile"}, "bar": {"/bar"}, filepath.Join("docker", "bar"): {"/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/docker/nginx.conf"}, "file": {"/file"}, "server.go": {"/server.go"}, "test.conf": {"/test.conf"}, "worker.go": {"/worker.go"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "bar\ndocker/*\n*.go",
			expected:    map[string][]string{".dot": {"/etc/.dot"}, "Dockerfile": {"/etc/Dockerfile"}, "file": {"/etc/file", "/etc/file"}, "test.conf": {"/etc/test.conf"}},
			fetched:     []string{"nginx"},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    map[string][]string{".dot": {"/onbuild/.dot"}, "Dockerfile": {"/onbuild/Dockerfile"}, "bar": {"/onbuild/bar"}, filepath.Join("docker", "bar"): {"/onbuild/docker/bar"}, filepath.Join("docker", "nginx.conf"): {"/onbuild/docker/nginx.conf"}, "file": {"/onbuild/file"}, "server.go": {"/onbuild/server.go"}, "test.conf": {"/onbuild/test.conf"}, "worker.go": {"/onbuild/worker.go"}},
			fetched:     []string{"golang:onbuild"},
		},
		{
			description: "onbuild with dockerignore",
			dockerfile:  onbuild,
			workspace:   ".",
			ignore:      "bar\ndocker/*",
			expected:    map[string][]string{".dot": {"/onbuild/.dot"}, "Dockerfile": {"/onbuild/Dockerfile"}, "file": {"/onbuild/file"}, "server.go": {"/onbuild/server.go"}, "test.conf": {"/onbuild/test.conf"}, "worker.go": {"/onbuild/worker.go"}},
			fetched:     []string{"golang:onbuild"},
		},
		{
			description: "onbuild error",
			dockerfile:  onbuildError,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "file": {"/etc/file"}},
			fetched:     []string{"noimage:latest"},
		},
		{
			description: "build args",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.StringPtr("worker.go")},
			expected:    map[string][]string{"Dockerfile": nil, "worker.go": {"/worker.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with default value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with redefined default value",
			dockerfile:  copyWorkerGoBuildArgRedefinedDefaultValue,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "worker.go": {"/worker.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args all defined a the top",
			dockerfile:  copyServerGoBuildArgsAtTheTop,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "override default build arg",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("worker.go")},
			expected:    map[string][]string{"Dockerfile": nil, "worker.go": {"/worker.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "ignore build arg and use default arg value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": nil},
			expected:    map[string][]string{"Dockerfile": nil, "server.go": {"/server.go"}},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from base stage",
			dockerfile:  fromStage,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from base stage, ignoring case",
			dockerfile:  fromStageIgnoreCase,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from scratch",
			dockerfile:  fromScratch,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "file": {"/etc/file"}},
			fetched:     nil,
		},
		{
			description: "from scratch, ignoring case",
			dockerfile:  fromScratchUppercase,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "file": {"/etc/file"}},
			fetched:     nil,
		},
		{
			description: "case sensitive",
			dockerfile:  fromImageCaseSensitive,
			workspace:   ".",
			expected:    map[string][]string{"Dockerfile": nil, "file": {"/etc/file"}},
			fetched:     []string{"jboss/wildfly:14.0.1.Final"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()
			originalWorkingDir := WorkingDir
			WorkingDir = func(tagged string) (string, error) {
				return "/", nil
			}
			defer func() {
				WorkingDir = originalWorkingDir
			}()

			imageFetcher := fakeImageFetcher{}
			RetrieveImage = imageFetcher.fetch
			defer func() { RetrieveImage = retrieveImage }()

			for _, file := range []string{"docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file", ".dot"} {
				tmpDir.Write(file, "")
			}

			if !test.badReader {
				tmpDir.Write(test.workspace+"/Dockerfile", test.dockerfile)
			}

			if test.ignore != "" {
				tmpDir.Write(test.workspace+"/.dockerignore", test.ignore)
			}

			workspace := tmpDir.Path(test.workspace)
			deps, err := GetDependencies(context.Background(), workspace, "Dockerfile", test.buildArgs)

			// destinations are not sorted, but for the test assertion they must be
			for _, dsts := range deps {
				sort.Strings(dsts)
			}

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, deps)
			testutil.CheckDeepEqual(t, test.fetched, imageFetcher.fetched)
		})
	}
}
