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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	testutil "github.com/GoogleCloudPlatform/skaffold/test"
	"github.com/sirupsen/logrus"

	"github.com/spf13/afero"
)

var tmpDir string

var mockFS = map[string][]string{
	"Dockerfile":              {"COPY 1 /", "ADD dir/2 /"},
	"Dockerfile.MISSINGFILE":  {"COPY file_MISSING /"},
	"Dockerfile.symlinkdep":   {"COPY 5 /", "COPY 5 /"},
	"Dockerfile.star":         {"COPY * /"},
	"Dockerfile.ignored_file": {"COPY vendor/3 /", "COPY 1 /"},
	// regular files
	"1":         nil,
	"dir/2":     nil,
	"vendor/3":  nil,
	"4.symlink": {"1"},
	"5":         nil,
}

func initFS() {
	for p, contentSlice := range mockFS {
		fullPath := filepath.Join(tmpDir, p)
		contents := strings.Join(contentSlice, "\n")
		dir := filepath.Dir(fullPath)
		if err := fs.MkdirAll(dir, 0750); err != nil {
			logrus.Fatalf("making mock fs dir %s", err)
		}
		if strings.HasSuffix(fullPath, "symlink") {
			if err := os.Symlink(filepath.Join(tmpDir, contents), fullPath); err != nil {
				logrus.Fatalf("creating symlink file: %s", err)
			}
			continue
		}
		if err := afero.WriteFile(fs, fullPath, []byte(contents), 0640); err != nil {
			logrus.Fatalf("writing mock fs file: %s", err)
		}
	}
}

func write(t *testing.T, path, contents string) {
	if err := afero.WriteFile(fs, filepath.Join(tmpDir, path), []byte(contents), 0640); err != nil {
		t.Errorf("writing mock fs file: %s", err)
	}
}

func TestWatch(t *testing.T) {
	var tests = []struct {
		description    string
		artifacts      []*config.Artifact
		dockerfiles    []string
		writes         []string
		expected       *WatchEvent
		expectedChange []string
		sendCancel     bool
		shouldErr      bool
	}{
		{
			description: "write file and ignored file",
			dockerfiles: []string{"Dockerfile.ignored_file", "Dockerfile"},
			writes: []string{
				"vendor/3",
				"dir/2",
			},
			expectedChange: []string{"Dockerfile"},
		},
		{
			description: "missing dockerfile",
			dockerfiles: []string{"Dockerfile.MISSINGFILE"},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			artifacts := []*config.Artifact{}
			for _, d := range test.dockerfiles {
				artifacts = append(artifacts, &config.Artifact{
					DockerfilePath: d,
					Workspace:      tmpDir,
				})
			}
			watcher := &FSWatcher{}
			eventCh := make(chan *WatchEvent, 1)
			readyCh := make(chan *WatchEvent, 1)
			errCh := make(chan error, 1)
			cancel := make(chan struct{}, 1)
			go func() {
				evt, err := watcher.Watch(artifacts, readyCh, cancel)
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
				if e.ChangedArtifacts == nil {
					t.Errorf("No changed artifacts but expected %s", test.expectedChange)
				}
				actual := []string{}
				for _, d := range e.ChangedArtifacts {
					actual = append(actual, d.DockerfilePath)
				}
				if !reflect.DeepEqual(actual, test.expectedChange) {
					t.Errorf("Expected %+v, Actual %+v", test.expectedChange, actual)
				}
				return
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
		dockerfile  string
		expected    []string

		shouldErr bool
	}{
		{
			description: "add deps",
			dockerfile:  "Dockerfile",
			expected:    []string{"1", "dir/2"},
		},
		{
			description: "missing dockerfile",
			dockerfile:  "not a real file",
			shouldErr:   true,
		},
		{
			description: "missing file",
			dockerfile:  "Dockerfile.MISSINGFILE",
			shouldErr:   true,
		},
		{
			description: "symlink deps ignored",
			dockerfile:  "Dockerfile.symlinkdep",
			expected:    []string{"5"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m := map[string][]*config.Artifact{}
			a := &config.Artifact{
				Workspace:      tmpDir,
				DockerfilePath: test.dockerfile,
			}
			expectedMap := map[string][]*config.Artifact{}
			for _, d := range test.expected {
				p := filepath.Join(tmpDir, d)
				arts, ok := expectedMap[p]
				if !ok {
					expectedMap[p] = []*config.Artifact{a}
					continue
				}
				expectedMap[p] = append(arts, a)
			}
			err := addDepsForArtifact(a, m)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, expectedMap, m)
		})
	}
}

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "skaffold")
	if err != nil {
		logrus.Fatalf("Getting temp dir: %s", err)
	}
	// On macOS the /tmp is symlinked to /private/tmp
	// the dockerfile parser won't accept symlinks, so we evaluate
	// the symlink for our tmp dir
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		logrus.Fatalf("Evaluating possible temp dir symlink: %s", err)
	}
	cleanup := func() {
		if err := fs.RemoveAll(tmpDir); err != nil {
			logrus.Fatalf("Removing testing temp dir: %s", err)
		}
	}
	initFS()
	exit := m.Run()
	cleanup()
	os.Exit(exit)
}
