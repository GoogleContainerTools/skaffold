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
	"io"
	"strings"
	"testing"

	"sort"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
	"github.com/spf13/afero"
)

const copyDockerfile = `
FROM ubuntu:14.04
COPY server.go .
CMD server.go
`

const addDockerfile = `
FROM gcr.io/nginx
ADD nginx.conf /etc/nginx
CMD nginx
`

const multiCopy = `
FROM gcr.io/nginx
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
FROM gcr.io/nginx
ADD . /etc/
COPY ./file /etc/file
CMD nginx
`

func TestGetDockerfileDependencies(t *testing.T) {
	var tests = []struct {
		description string
		dockerfile  string
		workspace   string

		expected  []string
		badReader bool
		shouldErr bool
	}{
		{
			description: "copy dependency",
			dockerfile:  copyDockerfile,
			workspace:   ".",
			expected:    []string{"server.go"},
		},
		{
			description: "add dependency",
			dockerfile:  addDockerfile,
			workspace:   "docker",
			expected:    []string{"docker/nginx.conf"},
		},
		{
			description: "bad read",
			badReader:   true,
			shouldErr:   true,
		},
		{
			description: "multistage dockerfile",
			dockerfile:  multiStageDockerfile,
			workspace:   "",
			expected:    []string{"worker.go"},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    []string{"test.conf"},
		},
		{
			description: "env test",
			dockerfile:  envTest,
			workspace:   ".",
			expected:    []string{"bar"},
		},
	}

	pkgFS := fs
	defer func() {
		fs = pkgFS
	}()
	fs = afero.NewMemMapFs()
	fs.MkdirAll("docker", 0750)
	files := []string{"docker/nginx.conf", "server.go", "test.conf", "worker.go", "bar", "file"}
	for _, name := range files {
		afero.WriteFile(fs, name, []byte(""), 0644)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var r io.Reader
			r = strings.NewReader(test.dockerfile)
			if test.badReader {
				r = testutil.BadReader{}
			}
			deps, err := GetDockerfileDependencies(test.workspace, r)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, deps)
		})
	}
}

func TestExpandDeps(t *testing.T) {
	var tests = []struct {
		description string
		in          []string
		out         []string
		shouldErr   bool
	}{
		{
			description: "add single files",
			in:          []string{"test/a", "test/b", "test/c"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add directory",
			in:          []string{"test"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add directory trailing slash",
			in:          []string{"test/"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "file not exist",
			in:          []string{"test/d"},
			shouldErr:   true,
		},
		{
			description: "add wildcard star",
			in:          []string{"*"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
		{
			description: "add wildcard any character",
			in:          []string{"test/?"},
			out:         []string{"test/a", "test/b", "test/c"},
		},
	}

	pkgFS := fs
	defer func() {
		fs = pkgFS
	}()
	fs = afero.NewMemMapFs()
	fs.MkdirAll("test", 0755)
	files := []string{"test/a", "test/b", "test/c"}
	for _, name := range files {
		afero.WriteFile(fs, name, []byte(""), 0644)
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := expandDeps(".", test.in)
			// Sort both slices for reproducibility
			sort.Strings(actual)
			sort.Strings(test.out)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.out, actual)
		})
	}
}
