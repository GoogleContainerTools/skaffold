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
	"reflect"
	"strings"
	"testing"

	testutil "github.com/GoogleCloudPlatform/skaffold/test"
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
			workspace:   "/go/src/github.com/skaffold",
			expected:    []string{"/go/src/github.com/skaffold/worker.go"},
		},
		{
			description: "copy twice",
			dockerfile:  multiCopy,
			workspace:   ".",
			expected:    []string{"test.conf"},
		},
		{
			description: "copy directory",
			dockerfile:  copyDirectory,
			workspace:   ".",
			expected:    []string{".", "file"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var r io.Reader
			r = strings.NewReader(test.dockerfile)
			if test.badReader {
				r = testutil.BadReader{}
			}
			deps, err := GetDockerfileDependencies(test.workspace, r)
			if err != nil && !test.shouldErr {
				t.Errorf("Test should have failed but didn't return error: %s, error: %s", test.description, err)
				return
			}
			if err == nil && test.shouldErr {
				t.Errorf("Test didn't return error but should have: %s", test.description)
				return
			}
			if !reflect.DeepEqual(deps, test.expected) {
				t.Errorf("Dependencies differ: actual: \n%+v\n expected \n%+v", deps, test.expected)
			}
		})
	}
}
