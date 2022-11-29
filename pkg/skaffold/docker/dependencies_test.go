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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const copyEmptydirectory = `
FROM ubuntu:14.04
COPY emptydir .
`

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

const simpleArtifactDependency = `
ARG BASE
FROM $BASE
COPY worker.go .
`

const multiStageArtifactDependency = `
ARG BASE
FROM golang:1.9.2
COPY worker.go .

FROM $BASE AS foo
FROM foo as bar
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

const buildKitDockerfile = `
# syntax = docker/dockerfile:1-experimental
FROM golang:1.9.2
COPY server.go .
RUN --mount=type=cache,target=/go/pkg/mod go build .
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

const invalidFrom = `
FROM
COPY . /
`

const fromV1Manifest = `
FROM library/ruby:2.3.0
ADD ./file /etc/file
`

type fakeImageFetcher struct{}

func (f *fakeImageFetcher) fetch(_ context.Context, image string, _ Config) (*v1.ConfigFile, error) {
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
	case "library/ruby:2.3.0":
		return nil, fmt.Errorf("retrieving image \"library/ruby:2.3.0\": unsupported MediaType: \"application/vnd.docker.distribution.manifest.v1+prettyjws\", see https://github.com/google/go-containerregistry/issues/377")
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
			description: "copy empty directory",
			dockerfile:  copyEmptydirectory,
			workspace:   ".",
			expected:    []string{"Dockerfile", "emptydir"},
		},
		{
			description: "buildkit dockerfile",
			dockerfile:  buildKitDockerfile,
			workspace:   "",
			expected:    []string{"Dockerfile", "server.go"},
		},
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
			description: "simple dockerfile with artifact dependency",
			dockerfile:  simpleArtifactDependency,
			workspace:   "",
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "multistage dockerfile with artifact dependency",
			dockerfile:  multiStageArtifactDependency,
			workspace:   "",
			expected:    []string{"Dockerfile", "worker.go"},
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
			ignore:      "emptydir\nbar\ndocker/*",
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
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "emptydir", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "emptydir", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "emptydir", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "emptydir\nbar\ndocker\n*.go",
			expected:    []string{".dot", "Dockerfile", "file", "test.conf"},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "emptydir", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "onbuild with dockerignore",
			dockerfile:  onbuild,
			workspace:   ".",
			ignore:      "emptydir\nbar\ndocker/*",
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
			buildArgs:   map[string]*string{"FOO": util.Ptr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.Ptr("worker.go")},
			expected:    []string{"Dockerfile", "worker.go"},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.Ptr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.Ptr("server.go")},
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
			buildArgs:   map[string]*string{"FOO": util.Ptr("worker.go")},
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
			buildArgs:   map[string]*string{"FOO": util.Ptr("{{.FILE_NAME}}")},
			env:         []string{"FILE_NAME=server.go"},
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "invalid go template as build arg",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.Ptr("{{")},
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
			ignore:         "emptydir\nbar\ndocker/*",
			ignoreFilename: "Dockerfile.dockerignore",
			expected:       []string{".dot", "Dockerfile", "Dockerfile.dockerignore", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "invalid dockerfile",
			dockerfile:  invalidFrom,
			workspace:   ".",
			shouldErr:   true,
		},
		{
			description: "old manifest version - watch local file dependency.",
			dockerfile:  fromV1Manifest,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)
			t.Override(&util.OSEnviron, func() []string { return test.env })

			tmpDir := t.NewTempDir().
				Touch("docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file", ".dot")
			tmpDir.Mkdir("emptydir")
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
			m := mockConfig{
				mode: config.RunModes.Dev,
			}
			deps, err := GetDependencies(context.Background(), NewBuildConfig(workspace, "test", "Dockerfile", test.buildArgs), m)
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

func TestGetDependenciesCached(t *testing.T) {
	imageFetcher := fakeImageFetcher{}
	tests := []struct {
		description        string
		retrieveImgMock    func(context.Context, string, Config) (*v1.ConfigFile, error)
		dependencyCache    map[string][]string
		dependencyCacheErr map[string]error
		expected           []string
		shouldErr          bool
	}{
		{
			description:     "with no cached results getDependencies will retrieve image",
			retrieveImgMock: imageFetcher.fetch,
			dependencyCache: map[string][]string{},
			expected:        []string{"Dockerfile", "server.go"},
		},
		{
			description: "with cached results getDependencies should read from cache",
			retrieveImgMock: func(context.Context, string, Config) (*v1.ConfigFile, error) {
				return nil, fmt.Errorf("unexpected call")
			},
			dependencyCache: map[string][]string{"dummy": {"random.go"}},
			expected:        []string{"random.go"},
		},
		{
			description: "with cached results is error getDependencies should read from cache",
			retrieveImgMock: func(context.Context, string, Config) (*v1.ConfigFile, error) {
				return &v1.ConfigFile{}, nil
			},
			dependencyCacheErr: map[string]error{"dummy": fmt.Errorf("remote manifest fetch")},
			shouldErr:          true,
		},
		{
			description:     "with cached results for dockerfile in another app",
			retrieveImgMock: imageFetcher.fetch,
			dependencyCache: map[string][]string{"another": {"random.go"}},
			expected:        []string{"Dockerfile", "server.go"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&RetrieveImage, test.retrieveImgMock)
			t.Override(&util.OSEnviron, func() []string { return []string{} })
			t.Override(&dependencyCache, util.NewSyncStore[[]string]())

			tmpDir := t.NewTempDir().Touch("server.go", "random.go")
			tmpDir.Write("Dockerfile", copyServerGo)

			for k, v := range test.dependencyCache {
				dependencyCache.Exec(k, func() ([]string, error) {
					return v, nil
				})
			}
			for k, v := range test.dependencyCacheErr {
				dependencyCache.Exec(k, func() ([]string, error) {
					return nil, v
				})
			}
			deps, err := GetDependenciesCached(context.Background(), NewBuildConfig(tmpDir.Root(), "dummy", "Dockerfile", map[string]*string{}), nil)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deps)
		})
	}
}
