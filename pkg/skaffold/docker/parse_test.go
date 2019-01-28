/*
Copyright 2018 The Skaffold Authors

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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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
ADD *.go /tmp
`

const wildcardsMatchesNone = `
FROM nginx
ADD *.none /tmp
`

const oneWilcardMatchesNone = `
FROM nginx
ADD *.go *.none /tmp
`

const multiStageDockerfile = `
FROM golang:1.9.2
WORKDIR /go/src/github.com/r2d4/leeroy/
COPY worker.go .
RUN go build -o worker .

FROM gcr.io/distroless/base
WORKDIR /root/
COPY --from=0 /go/src/github.com/r2d4/leeroy .
`

const envTest = `
FROM busybox
ENV foo bar
WORKDIR ${foo}   # WORKDIR /bar
COPY $foo /quux # COPY bar /quux
`

const copyDirectory = `
FROM nginx
ADD . /etc/
COPY ./file /etc/file
`
const multiFileCopy = `
FROM ubuntu:14.04
COPY server.go file .
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

// This has an ONBUILD instruction of "COPY . /go/src/app"
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

const targets = `
FROM scratch as target1
ADD ./file /etc/file
FROM target1 as target2
ADD ./bar /etc/bar
FROM target2
ADD ./server.go /etc/server.go
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
					"COPY . /go/src/app",
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
		target      string

		expected  []string
		fetched   []string
		badReader bool
		shouldErr bool
	}{
		{
			description: "no target",
			dockerfile:  targets,
			workspace:   ".",
			expected:    []string{"Dockerfile", "bar", "file", "server.go"},
		},
		{
			description: "target",
			dockerfile:  targets,
			workspace:   ".",
			target:      "target1",
			expected:    []string{"Dockerfile", "file"},
		},
		{
			description: "transitive target",
			dockerfile:  targets,
			workspace:   ".",
			target:      "target2",
			expected:    []string{"Dockerfile", "bar", "file"},
		},
		{
			description: "unknown target",
			dockerfile:  targets,
			workspace:   ".",
			target:      "target3",
			shouldErr:   true,
		},
		{
			description: "copy dependency",
			dockerfile:  copyServerGo,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "add dependency",
			dockerfile:  addNginx,
			workspace:   "docker",
			expected:    []string{"Dockerfile", "nginx.conf"},
			fetched:     []string{"nginx"},
		},
		{
			description: "wildcards",
			dockerfile:  wildcards,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go", "worker.go"},
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
			expected:    []string{"Dockerfile", "server.go", "worker.go"},
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
			expected:    []string{"Dockerfile"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile,
			workspace:   "",
			expected:    []string{"Dockerfile", "worker.go"},
			fetched:     []string{"golang:1.9.2", "gcr.io/distroless/base"},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    []string{"Dockerfile", "test.conf"},
			fetched:     []string{"nginx"},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    []string{"Dockerfile", "bar"},
			fetched:     []string{"busybox"},
		},
		{
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "dockerignore test",
			dockerfile:  copyDirectory,
			ignore:      "bar\ndocker/*",
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "file", "server.go", "test.conf", "worker.go"},
			fetched:     []string{"nginx"},
		},
		{
			description: "dockerignore dockerfile",
			dockerfile:  copyServerGo,
			ignore:      "Dockerfile",
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "dockerignore with non canonical workspace",
			dockerfile:  contextDockerfile,
			workspace:   "docker/../docker",
			ignore:      "bar\ndocker/*",
			expected:    []string{"Dockerfile", "nginx.conf"},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore none",
			dockerfile:  copyAll,
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore dotfiles",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      ".*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
			fetched:     []string{"nginx"},
		},
		{
			description: "ignore dotfiles (root syntax)",
			dockerfile:  copyAll,
			workspace:   ".",
			ignore:      "/.*",
			expected:    []string{"Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
			fetched:     []string{"nginx"},
		},
		{
			description: "dockerignore with context in parent directory",
			dockerfile:  copyDirectory,
			workspace:   "docker/..",
			ignore:      "bar\ndocker/*\n*.go",
			expected:    []string{".dot", "Dockerfile", "file", "test.conf"},
			fetched:     []string{"nginx"},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    []string{".dot", "Dockerfile", "bar", filepath.Join("docker", "bar"), filepath.Join("docker", "nginx.conf"), "file", "server.go", "test.conf", "worker.go"},
			fetched:     []string{"golang:onbuild"},
		},
		{
			description: "onbuild error",
			dockerfile:  onbuildError,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
			fetched:     []string{"noimage:latest"},
		},
		{
			description: "build args",
			dockerfile:  copyServerGoBuildArg,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with same prefix",
			dockerfile:  copyWorkerGoBuildArgSamePrefix,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO2": util.StringPtr("worker.go")},
			expected:    []string{"Dockerfile", "worker.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with curly braces",
			dockerfile:  copyServerGoBuildArgCurlyBraces,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with extra whitespace",
			dockerfile:  copyServerGoBuildArgExtraWhitespace,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("server.go")},
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with default value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args with redefined default value",
			dockerfile:  copyWorkerGoBuildArgRedefinedDefaultValue,
			workspace:   ".",
			expected:    []string{"Dockerfile", "worker.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "build args all defined a the top",
			dockerfile:  copyServerGoBuildArgsAtTheTop,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "override default build arg",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": util.StringPtr("worker.go")},
			expected:    []string{"Dockerfile", "worker.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "ignore build arg and use default arg value",
			dockerfile:  copyServerGoBuildArgDefaultValue,
			workspace:   ".",
			buildArgs:   map[string]*string{"FOO": nil},
			expected:    []string{"Dockerfile", "server.go"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from base stage",
			dockerfile:  fromStage,
			workspace:   ".",
			expected:    []string{"Dockerfile"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from base stage, ignoring case",
			dockerfile:  fromStageIgnoreCase,
			workspace:   ".",
			expected:    []string{"Dockerfile"},
			fetched:     []string{"ubuntu:14.04"},
		},
		{
			description: "from scratch",
			dockerfile:  fromScratch,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
			fetched:     nil,
		},
		{
			description: "from scratch, ignoring case",
			dockerfile:  fromScratchUppercase,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
			fetched:     nil,
		},
		{
			description: "case sensitive",
			dockerfile:  fromImageCaseSensitive,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
			fetched:     []string{"jboss/wildfly:14.0.1.Final"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.NewTempDir(t)
			defer cleanup()

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
			deps, err := GetDependencies(context.Background(), workspace, &latest.DockerArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs:      test.buildArgs,
				Target:         test.target,
			})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, deps)
			testutil.CheckDeepEqual(t, test.fetched, imageFetcher.fetched)
		})
	}
}
