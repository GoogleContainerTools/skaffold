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
package watch

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	testutil "github.com/GoogleCloudPlatform/skaffold/test"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"

	"github.com/spf13/afero"
)

var mockFS = map[string]string{
	"/tmp/skaffold/workspace_1/Dockerfile":             workspace1Dockerfile,
	"/tmp/skaffold/workspace_1/Dockerfile.MISSINGFILE": workspace1DockerfileMissingFile,
	"/tmp/skaffold/workspace_1/Dockerfile.SymlinkDep":  workspace1DockerfileSymlink,
	"/tmp/skaffold/workspace_1/file_a":                 "a",
	"/tmp/skaffold/workspace_1/file_b":                 "b",
	"/tmp/skaffold/workspace_1/file.symlink":           "/tmp/skaffold/workspace_1/file_a",

	"/tmp/skaffold/workspace_2/Dockerfile":              workspace2Dockerfile,
	"/tmp/skaffold/workspace_2/Dockerfile.build":        workspace2DockerfileBuild,
	"/tmp/skaffold/workspace_2/Dockerfile.ignored_path": dockerfileIgnoredFile,
	"/tmp/skaffold/workspace_2/file_a":                  "a",
	"/tmp/skaffold/workspace_2/vendor/vendor_file":      "vendor",
	"/tmp/skaffold/workspace_2/dir_a/file_a":            "b",
}

const (
	workspace1Dockerfile = `
FROM test
COPY file_a dst_a
COPY file_b dst_b
CMD ls
`

	workspace1DockerfileSymlink = `
FROM test
COPY file.symlink dst_a
COPY file_b dst_b
CMD ls`

	workspace1DockerfileMissingFile = `
FROM test
COPY file_MISSING dst_a
COPY file_b dst_b
CMD ls
`

	workspace2DockerfileBuild = `
FROM test
COPY * /
CMD ls`

	workspace2Dockerfile = `
FROM test
COPY file_a /
ADD dir_a/file_a /
CMD ls`

	dockerfileIgnoredFile = `
FROM test
COPY file_a /
ADD vendor/vendor_file /
CMD ls`
)

var artifactA = &config.Artifact{
	Workspace:      "/tmp/skaffold/workspace_1",
	DockerfilePath: "Dockerfile",
}

var artifactDefaultDockerfilePath = &config.Artifact{
	Workspace: "/tmp/skaffold/workspace_1",
}

var artifactMissingFile = &config.Artifact{
	Workspace:      "/tmp/skaffold/workspace_1",
	DockerfilePath: "Dockerfile.MISSINGFILE",
}

var artifactMissingDockerfile = &config.Artifact{
	Workspace:      "/tmp/skaffold/workspace_1",
	DockerfilePath: "NOT_FOUND",
}

var artifactDockerfileWithIgnoredDep = &config.Artifact{
	Workspace:      "/tmp/skaffold/workspace_2",
	DockerfilePath: "Dockerfile.ignored_path",
}

var artifactSymlinkDep = &config.Artifact{
	Workspace:      "/tmp/skaffold/workspace_1",
	DockerfilePath: "Dockerfile.SymlinkDep",
}

func initFS() {
	for p, contents := range mockFS {
		dir := filepath.Dir(p)

		if err := fs.MkdirAll(dir, 0750); err != nil {
			logrus.Fatalf("making mock fs dir %s", err)
		}
		if strings.HasSuffix(p, "symlink") {
			if err := os.Symlink(contents, p); err != nil {
				logrus.Fatalf("creating symlink file: %s", err)
			}
			continue
		}
		if err := afero.WriteFile(fs, p, []byte(contents), 0640); err != nil {
			logrus.Fatalf("writing mock fs file: %s", err)
		}
	}
}

func write(t *testing.T, path, contents string) {
	if err := afero.WriteFile(fs, path, []byte(contents), 0640); err != nil {
		t.Errorf("writing mock fs file: %s", err)
	}
}

func TestWatch(t *testing.T) {
	var tests = []struct {
		description string
		artifacts   []*config.Artifact

		writes     []string
		expected   *WatchEvent
		sendCancel bool
		shouldErr  bool
	}{
		{
			description: "write file and ignored file",
			artifacts: []*config.Artifact{
				artifactA,
				artifactDockerfileWithIgnoredDep,
			},
			writes: []string{
				"/tmp/skaffold/workspace_2/vendor/vendor_file",
				"/tmp/skaffold/workspace_1/file_a",
			},
			expected: &WatchEvent{
				EventType:       "notify.Write",
				ChangedArtifact: artifactA,
			},
		},
		{
			description: "missing dockerfile",
			artifacts:   []*config.Artifact{artifactMissingDockerfile},
			shouldErr:   true,
		},
		{
			description: "send cancel",
			artifacts:   []*config.Artifact{artifactA},
			sendCancel:  true,
			expected: &WatchEvent{
				EventType: "WatchStop",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			watcher := &FSWatcher{}
			eventCh := make(chan *WatchEvent, 1)
			readyCh := make(chan *WatchEvent, 1)
			errCh := make(chan error, 1)
			cancel := make(chan struct{}, 1)
			go func() {
				evt, err := watcher.Watch(test.artifacts, readyCh, cancel)
				if err != nil {
					errCh <- err
					return
				}

				eventCh <- evt
			}()

			select {
			case err := <-errCh:
				testutil.CheckError(t, test.shouldErr, err)
				return
			case readyEvt := <-readyCh:
				if readyEvt.EventType != WatchReady {
					t.Errorf("Got unknown watch event %s, expected %s", readyEvt.EventType, WatchReady)
				}
			}

			if test.sendCancel {
				cancel <- struct{}{}
			}

			for _, p := range test.writes {
				write(t, p, "")
			}
			// Now check to see if the watch registered a change event
			select {
			case e := <-eventCh:
				if !reflect.DeepEqual(e, test.expected) {
					t.Errorf("Expected %+v, Actual %+v", test.expected, e)
				}
			case err := <-errCh:
				testutil.CheckError(t, test.shouldErr, err)
				return
			}

		})
	}
}

func TestAddDepsForArtifact(t *testing.T) {
	var tests = []struct {
		description string
		artifact    *config.Artifact
		expected    map[string]*config.Artifact

		shouldErr bool
	}{
		{
			description: "add deps",
			artifact:    artifactA,
			expected: map[string]*config.Artifact{
				"/tmp/skaffold/workspace_1/file_a": artifactA,
				"/tmp/skaffold/workspace_1/file_b": artifactA,
			},
		},
		{
			description: "missing dockerfile",
			artifact:    artifactMissingDockerfile,
			expected:    map[string]*config.Artifact{},
			shouldErr:   true,
		},
		{
			description: "missing file",
			artifact:    artifactMissingFile,
			expected:    map[string]*config.Artifact{},
			shouldErr:   true,
		},
		{
			description: "symlink file",
			artifact:    artifactSymlinkDep,
			expected: map[string]*config.Artifact{
				"/tmp/skaffold/workspace_1/file_b": artifactSymlinkDep,
			},
		},
		{
			description: "default dockerfile path",
			artifact:    artifactDefaultDockerfilePath,
			expected: map[string]*config.Artifact{
				"/tmp/skaffold/workspace_1/file_a": artifactDefaultDockerfilePath,
				"/tmp/skaffold/workspace_1/file_b": artifactDefaultDockerfilePath,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m := map[string]*config.Artifact{}
			err := addDepsForArtifact(test.artifact, m)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, m)
		})
	}
}

func TestAddWatchForDeps(t *testing.T) {
	var tests = []struct {
		description     string
		depsToArtifacts map[string]*config.Artifact
		c               chan notify.EventInfo
		shouldErr       bool
	}{
		{
			description: "error bad path",
			depsToArtifacts: map[string]*config.Artifact{
				"bad.path": artifactA,
			},
			c:         make(chan notify.EventInfo, 1),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := addWatchForDeps(test.depsToArtifacts, test.c)
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestMain(m *testing.M) {
	cleanup := func() {
		if err := fs.RemoveAll("/tmp/skaffold"); err != nil {
			logrus.Fatalf("Removing testing temp dir: %s", err)
		}
	}

	cleanup()
	initFS()
	exit := m.Run()
	cleanup()
	os.Exit(exit)
}
