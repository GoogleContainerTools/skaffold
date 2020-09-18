/*
Copyright 2020 The Skaffold Authors

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

package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestWithLogFile(t *testing.T) {
	logDeploySucceeded := " - fake/deployment created"
	logDeployFailed := " - failed to deploy"
	logFilename := "- writing logs to " + filepath.Join(os.TempDir(), "skaffold", "deploy", "deploy.log")

	tests := []struct {
		description        string
		muted              Muted
		shouldErr          bool
		expectedNamespaces []string
		logsFound          []string
		logsNotFound       []string
	}{
		{
			description:        "all logs",
			muted:              muted(false),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logDeploySucceeded},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "mute build logs",
			muted:              muted(true),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logFilename},
			logsNotFound:       []string{logDeploySucceeded},
		},
		{
			description:        "failed deploy - all logs",
			muted:              muted(false),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logDeployFailed},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "failed deploy - muted logs",
			muted:              muted(true),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logFilename, logDeployFailed},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var mockOut bytes.Buffer

			var deployer = mockDeployer{
				muted:     test.muted,
				shouldErr: test.shouldErr,
			}

			deployOut, postDeployFn, _ := WithLogFile("deploy.log", &mockOut, test.muted)
			namespaces, err := deployer.Deploy(context.Background(), deployOut, nil)
			postDeployFn(err)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedNamespaces, namespaces)
			for _, found := range test.logsFound {
				t.CheckContains(found, mockOut.String())
			}
			for _, notFound := range test.logsNotFound {
				t.CheckFalse(strings.Contains(mockOut.String(), notFound))
			}
		})
	}
}

// Used just to show how output gets routed to different writers with the log file
type mockDeployer struct {
	muted     Muted
	shouldErr bool
}

func (fd *mockDeployer) Deploy(ctx context.Context, out io.Writer, _ []build.Artifact) ([]string, error) {
	if fd.shouldErr {
		fmt.Fprintln(out, " - failed to deploy")
		return nil, errors.New("failed to deploy")
	}

	fmt.Fprintln(out, " - fake/deployment created")
	return []string{"ns"}, nil
}

type muted bool

func (m muted) MuteDeploy() bool {
	return bool(m)
}
