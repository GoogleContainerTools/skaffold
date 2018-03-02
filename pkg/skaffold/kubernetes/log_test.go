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

package kubernetes

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
)

func errorGetStream(r *restclient.Request) (io.ReadCloser, error) {
	return nil, fmt.Errorf("error stream")
}

func fakeGetStream(r *restclient.Request) (io.ReadCloser, error) {
	b := bytes.NewBufferString("logs\nlogs\nlogs\n")
	return closerBuffer{b}, nil
}

func getStreamWithError(r *restclient.Request) (io.ReadCloser, error) {
	return closerBuffer{testutil.BadReader{}}, nil
}

type closerBuffer struct {
	io.Reader
}

func (closerBuffer) Close() error { return nil }

func TestStreamLogs(t *testing.T) {
	var tests = []struct {
		description string
		initialObj  *v1.Pod
		getStream   func(r *restclient.Request) (io.ReadCloser, error)
		out         io.Writer

		shouldErr bool
	}{
		{
			description: "get logs no error",
			initialObj:  podReadyState,
			getStream:   fakeGetStream,
			out:         &bytes.Buffer{},
		},
		{
			description: "pod bad state",
			initialObj:  podBadPhase,
			getStream:   fakeGetStream,
			out:         &bytes.Buffer{},
			shouldErr:   true,
		},
		{
			description: "error getting stream",
			initialObj:  podReadyState,
			getStream:   errorGetStream,
			out:         &bytes.Buffer{},
			shouldErr:   true,
		},
		{
			description: "error reading stream",
			initialObj:  podReadyState,
			getStream:   getStreamWithError,
			out:         &bytes.Buffer{},
			shouldErr:   true,
		},
		{
			description: "error bad writer",
			initialObj:  podReadyState,
			getStream:   fakeGetStream,
			out:         testutil.BadWriter{},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			client := fake.NewSimpleClientset(test.initialObj)
			getStream = test.getStream
			err := StreamLogs(test.out, client.CoreV1(), "image_name")
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
