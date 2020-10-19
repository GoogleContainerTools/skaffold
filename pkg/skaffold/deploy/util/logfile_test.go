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
			muted:              mutedDeploy(false),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logDeploySucceeded},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "mute deploy logs",
			muted:              mutedDeploy(true),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logFilename},
			logsNotFound:       []string{logDeploySucceeded},
		},
		{
			description:        "failed deploy - all logs",
			muted:              mutedDeploy(false),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logDeployFailed},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "failed deploy - mutedDeploy logs",
			muted:              mutedDeploy(true),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logFilename},
			logsNotFound:       []string{logDeployFailed},
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
			postDeployFn()

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

func TestWithStatusCheckLogFile(t *testing.T) {
	logDeploySucceeded := " - deployment/leeroy-app is ready. [1/2 deployment(s) still pending]"
	logDeployFailed := " - deployment/leeroy-app failed. could not pull image"
	logFilename := "- writing logs to " + filepath.Join(os.TempDir(), "skaffold", "status-check", "status-check.log")

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
			muted:              mutedStatusCheck(false),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logDeploySucceeded},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "mute status check logs",
			muted:              mutedStatusCheck(true),
			shouldErr:          false,
			expectedNamespaces: []string{"ns"},
			logsFound:          []string{logFilename},
			logsNotFound:       []string{logDeploySucceeded},
		},
		{
			description:        "failed status-check - all logs",
			muted:              mutedStatusCheck(false),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logDeployFailed},
			logsNotFound:       []string{logFilename},
		},
		{
			description:        "failed status-check - mutedDeploy logs",
			muted:              mutedStatusCheck(true),
			shouldErr:          true,
			expectedNamespaces: nil,
			logsFound:          []string{logFilename},
			logsNotFound:       []string{logDeployFailed},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var mockOut bytes.Buffer

			var deployer = mockStatusChecker{
				muted:     test.muted,
				shouldErr: test.shouldErr,
			}

			deployOut, postDeployFn, _ := WithStatusCheckLogFile("status-check.log", &mockOut, test.muted)
			namespaces, err := deployer.Deploy(context.Background(), deployOut, nil)
			postDeployFn()

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

// Used just to show how output gets routed to different writers with the log file
type mockStatusChecker struct {
	muted     Muted
	shouldErr bool
}

func (fd *mockStatusChecker) Deploy(ctx context.Context, out io.Writer, _ []build.Artifact) ([]string, error) {
	if fd.shouldErr {
		fmt.Fprintln(out, " - deployment/leeroy-app failed. could not pull image")
		return nil, errors.New("- deployment/leeroy-app failed. could not pull image")
	}

	fmt.Fprintln(out, "  - deployment/leeroy-app is ready. [1/2 deployment(s) still pending]")
	return []string{"ns"}, nil
}

type mutedDeploy bool

func (m mutedDeploy) MuteDeploy() bool {
	return bool(m)
}

func (m mutedDeploy) MuteStatusCheck() bool {
	return false
}

type mutedStatusCheck bool

func (m mutedStatusCheck) MuteDeploy() bool {
	return false
}

func (m mutedStatusCheck) MuteStatusCheck() bool {
	return bool(m)
}
