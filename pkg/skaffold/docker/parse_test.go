/*
Copyright 2018 Google LLC

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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/v1"
	"github.com/spf13/afero"
)

const copyDockerfile = `
FROM ubuntu:14.04
COPY server.go .
CMD server.go
`

const addDockerfile = `
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

const multiStageDockerfile = `
FROM golang:1.9.2
WORKDIR /go/src/github.com/r2d4/leeroy/
COPY worker.go .
RUN go build -o worker .

FROM gcr.io/distroless/base
WORKDIR /root/
COPY --from=0 /go/src/github.com/r2d4/leeroy .
CMD ["./worker"]
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
CMD nginx
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
CMD nginx
`

const dockerIgnore = `
bar
docker/*
`

// This has an ONBUILD instruction of "COPY . /go/src/app"
const onbuild = `
FROM golang:onbuild
`

const onbuildError = `
FROM noimage:latest
ADD ./file /etc/file
`

const onePortFromBaseImage = `
FROM oneport
`

const onePortFromBaseImageAndDockerfile = `
FROM oneport
EXPOSE 9000
`

const severalPortsFromBaseImage = `
FROM severalports
`

const severalPortsFromBaseImageAndDockerfile = `
FROM severalports
EXPOSE 9000 9001
EXPOSE 9002/tcp
`

func joinToTmpDir(base string, paths []string) []string {
	if paths == nil {
		return nil
	}
	ret := []string{}
	for _, p := range paths {
		ret = append(ret, filepath.Join(base, p))
	}
	return ret
}

var ImageConfigs = map[string]*v1.ConfigFile{
	"golang:onbuild": {
		Config: v1.Config{
			OnBuild: []string{
				"COPY . /go/src/app",
			},
		},
	},
	"ubuntu:14.04": {Config: v1.Config{}},
	"nginx":        {Config: v1.Config{}},
	"busybox":      {Config: v1.Config{}},
	"oneport": {
		Config: v1.Config{
			ExposedPorts: map[string]struct{}{
				"8000": {},
			},
		}},
	"severalports": {
		Config: v1.Config{
			ExposedPorts: map[string]struct{}{
				"8000":     {},
				"8001/tcp": {},
			},
		}},
}

func mockRetrieveImage(image string) (*v1.ConfigFile, error) {
	if cfg, ok := ImageConfigs[image]; ok {
		return cfg, nil
	}
	return nil, fmt.Errorf("No image found for %s", image)
}

func TestGetDockerfileDependencies(t *testing.T) {
	var tests = []struct {
		description  string
		dockerfile   string
		workspace    string
		dockerIgnore bool

		expected  []string
		badReader bool
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyDockerfile,
			workspace:   ".",
			expected:    []string{"Dockerfile", "server.go"},
		},
		{
			description: "add dependency",
			dockerfile:  addDockerfile,
			workspace:   "docker",
			expected:    []string{"docker/Dockerfile", "docker/nginx.conf"},
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
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile,
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
			description: "multi file copy",
			dockerfile:  multiFileCopy,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file", "server.go"},
		},
		{
			description:  "dockerignore test",
			dockerfile:   copyDirectory,
			dockerIgnore: true,
			workspace:    ".",
			expected:     []string{"Dockerfile", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description:  "dockerignore with context in parent directory test",
			dockerfile:   contextDockerfile,
			workspace:    "docker/../docker",
			dockerIgnore: true,
			expected:     []string{},
		},
		{
			description: "onbuild test",
			dockerfile:  onbuild,
			workspace:   ".",
			expected:    []string{"Dockerfile", "bar", "docker/bar", "docker/nginx.conf", "file", "server.go", "test.conf", "worker.go"},
		},
		{
			description: "onbuild error",
			dockerfile:  onbuildError,
			workspace:   ".",
			expected:    []string{"Dockerfile", "file"},
		},
	}

	RetrieveImage = mockRetrieveImage
	defer func() {
		RetrieveImage = retrieveImage
	}()
	defer util.ResetFs()

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			util.Fs = afero.NewMemMapFs()
			util.Fs.MkdirAll("docker", 0750)
			for _, file := range []string{"docker/nginx.conf", "docker/bar", "server.go", "test.conf", "worker.go", "bar", "file"} {
				afero.WriteFile(util.Fs, file, []byte(""), 0644)
			}

			if !test.badReader {
				afero.WriteFile(util.Fs, filepath.Join(test.workspace, "Dockerfile"), []byte(test.dockerfile), 0644)
			}

			if test.dockerIgnore {
				afero.WriteFile(util.Fs, ".dockerignore", []byte(dockerIgnore), 0644)
				afero.WriteFile(util.Fs, filepath.Join("docker", ".dockerignore"), []byte(dockerIgnore), 0644)
			}

			deps, err := GetDockerfileDependencies("Dockerfile", test.workspace)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, deps)
		})
	}
}

func TestPortsFromDockerfile(t *testing.T) {
	type args struct {
		dockerfile string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "one port from base image",
			args: args{dockerfile: onePortFromBaseImage},
			want: []string{"8000"},
		},
		{
			name: "two ports from base image",
			args: args{dockerfile: severalPortsFromBaseImage},
			want: []string{"8000", "8001/tcp"},
		},
		{
			name: "one port from dockerfile",
			args: args{dockerfile: onePortFromBaseImageAndDockerfile},
			want: []string{"8000", "9000"},
		},
		{
			name: "several port from dockerfile",
			args: args{dockerfile: severalPortsFromBaseImageAndDockerfile},
			want: []string{"8000", "8001/tcp", "9000", "9001", "9002/tcp"},
		},
	}

	RetrieveImage = mockRetrieveImage
	defer func() {
		RetrieveImage = retrieveImage
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.args.dockerfile)
			got, err := PortsFromDockerfile(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PortsFromDockerfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("PortsFromDockerfile() = %v, want %v", got, tt.want)
			}
		})
	}
}
