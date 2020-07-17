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
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const copyServerGo = `
FROM ubuntu:14.04
COPY server.go .
CMD server.go
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

const wildcardsMatchesNone = `
FROM nginx
ADD *.none /tmp/
`

const oneWilcardMatchesNone = `
FROM nginx
ADD *.go *.none /tmp/
`

const multiStageDockerfile1 = `
FROM golang:1.9.2
WORKDIR /go/src/github.com/r2d4/leeroy/
COPY worker.go .
RUN go build -o worker .

FROM gcr.io/distroless/base
WORKDIR /root/
COPY --from=0 /go/src/github.com/r2d4/leeroy ./
`

const multiStageDockerfile2 = `
FROM golang:1.9.2
COPY worker.go .

FROM gcr.io/distroless/base
ADD server.go .
`

const envTest = `
FROM busybox
ENV foo bar
WORKDIR ${foo}   # WORKDIR /bar
COPY $foo /quux # COPY bar /quux
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

const baseImageNotFound = `
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

const fromScratchQuoted = `
FROM "scratch"
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

const fromScratchWithStageName = `
FROM scratch as stage
FROM stage
ADD ./file /etc/file
`

type fakeImageFetcher struct{}

func (f *fakeImageFetcher) fetch(image string, _ map[string]bool) (*v1.ConfigFile, error) {
	switch image {
	case "ubuntu:14.04", "busybox", "nginx", "golang:1.9.2", "jboss/wildfly:14.0.1.Final", "gcr.io/distroless/base":
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
	tests := []struct {
		description    string
		dockerfile     string
		workspace      string
		ignore         string
		ignoreFilename string
		buildArgs      map[string]*string
		env            []string

		expected  []string
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyServerGo,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "add dependency",
			dockerfile:  addNginx,
			workspace:   "docker",
			expected:    []string{"Dockerfile", "nginx.conf"},
		},
		{
			description: "wildcards",
			dockerfile:  wildcards,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go", "worker.go"},
		},
		{
			description: "wildcards matches none",
			dockerfile:  wildcardsMatchesNone,
			workspace:   ".",
			shouldErr:   true,
		},
		{
			description: "one wilcard matches none",
			dockerfile:  oneWilcardMatchesNone,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go", "worker.go"},
		},
		{
			description: "not found",
			expected:    []string{"Dockerfile"},
		},
		{
			// https://github.com/GoogleContainerTools/skaffold/issues/158
			description: "no dependencies on remote files",
			dockerfile:  remoteFileAdd,
			expected:    []string{"Dockerfile"},
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile1,
			workspace:   "",
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "multistage dockerfile with source dependencies in both stages",
			dockerfile:  multiStageDockerfile2,
			workspace:   "",
			expected:    []string{"Dockerfile", "server.go", "worker.go"},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    []string{"Dockerfile", "test.conf"},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    []string{"Dockerfile", "bar"},
		},
		{
			description: "multiple env test",
			dockerfile:  multiEnvTest,
			workspace:   ".",
			expected:    []string{"Dockerfile", filepath.Join("docker", "nginx.conf")},
		},
		{
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file", "server.go"},
		},
		{
			description: "dockerignore test",
			dockerfile:  copyDirectory,
			ignore:      "bar\ndocker/*",
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "dockerignore dockerfile",
			dockerfile:  copyServerGo,
			ignore:      "Dockerfile",
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "dockerignore with non canonical workspace",
			dockerfile:  contextDockerfile,
			workspace:   "docker/../docker",
			ignore:      "bar\ndocker/*",
			expected:    []string{"Dockerfile", "nginx.conf"},
		},
		{
			description: "ignore none",
			dockerfile:  copyAll,
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "bar\ndocker\n*.go",
			expected:    []string{".dot", "Dockerfile", "file", "test.conf"},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "onbuild with dockerignore",
			dockerfile:  onbuild,
			workspace:   ".",
			ignore:      "bar\ndocker/*",
			expected:    []string{".dot", "Dockerfile", "file", "server.go", "test.conf", "worker.go"},
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
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.StringPtr("worker.go")},
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with default value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with redefined default value",
			dockerfile:  copyWorkerGoBuildArgRedefinedDefaultValue,
			workspace:   ".",
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "build args all defined a the top",
			dockerfile:  copyServerGoBuildArgsAtTheTop,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "override default build arg",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("worker.go")},
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "ignore build arg and use default arg value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": nil},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "from base stage",
			dockerfile:  fromStage,
			workspace:   ".",
			expected:    []string{"Dockerfile"},
		},
		{
			description: "from base stage, ignoring case",
			dockerfile:  fromStageIgnoreCase,
			workspace:   ".",
			expected:    []string{"Dockerfile"},
		},
		{
			description: "from scratch",
			dockerfile:  fromScratch,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description: "from scratch quoted",
			dockerfile:  fromScratchQuoted,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description: "from scratch, ignoring case",
			dockerfile:  fromScratchUppercase,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description: "case sensitive",
			dockerfile:  fromImageCaseSensitive,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description: "build args with an environment variable",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("{{.FILE_NAME}}")},
			env:         []string{"FILE_NAME=server.go"},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "invalid go template as build arg",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("{{")},
			shouldErr:   true,
		},
		{
			description: "ignore with negative pattern",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "**\n!docker/**",
			expected:    []string{"Dockerfile", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf")},
		},
		{
			description: "ignore with negative filename",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "**\n!server.go",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "from scratch witch stage name",
			dockerfile:  fromScratchWithStageName,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description:    "find specific dockerignore",
			dockerfile:     copyDirectory,
			workspace:      ".",
			ignore:         "bar\ndocker/*",
			ignoreFilename: "Dockerfile.dockerignore",
			expected:       []string{".dot", "Dockerfile", "Dockerfile.dockerignore", "file", "server.go", "test.conf", "worker.go"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)
			t.Override(&util.OSEnviron, func() []string { return test.env })

			tmpDir := t.NewTempDir().
				Touch("docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file", ".dot")
			if test.dockerfile != "" {
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
			deps, err := GetDependencies(context.Background(), workspace, "Dockerfile", test.buildArgs, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, deps)
		})
	}
}

func TestNormalizeDockerfilePath(t *testing.T) {
	tests := []struct {
		description string
		files       []string
		dockerfile  string

		expected string // relative path
	}{
		{
			description: "dockerfile found in context",
			files:       []string{"Dockerfile", "context/Dockerfile"},
			dockerfile:  "Dockerfile",
			expected:    "context/Dockerfile",
		},
		{
			description: "path to dockerfile resolved in context first",
			files:       []string{"context/context/Dockerfile", "context/Dockerfile"},
			dockerfile:  "context/Dockerfile",
			expected:    "context/context/Dockerfile",
		},
		{
			description: "path to dockerfile in working directory",
			files:       []string{"context/Dockerfile"},
			dockerfile:  "context/Dockerfile",
			expected:    "context/Dockerfile",
		},
		{
			description: "workspace dockerfile when missing in context",
			files:       []string{"Dockerfile", "context/randomfile.txt"},
			dockerfile:  "Dockerfile",
			expected:    "Dockerfile",
		},
		{
			description: "explicit dockerfile path",
			files:       []string{"context/Dockerfile", "elsewhere/Dockerfile"},
			dockerfile:  "elsewhere/Dockerfile",
			expected:    "elsewhere/Dockerfile",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			d := t.NewTempDir()
			t.Chdir(d.Root())

			d.Mkdir("context")
			d.Touch(test.files...)

			f, err := NormalizeDockerfilePath(d.Path("context"), test.dockerfile)

			t.CheckNoError(err)
			checkSameFile(t, d.Path(test.expected), f)
		})
	}
}

func checkSameFile(t *testutil.T, expected, result string) {
	t.Helper()

	i1, err := os.Stat(expected)
	t.CheckNoError(err)

	i2, err := os.Stat(result)
	t.CheckNoError(err)

	if !os.SameFile(i1, i2) {
		t.Errorf("returned wrong file\n   got: %s\nwanted: %s", result, expected)
	}
}
