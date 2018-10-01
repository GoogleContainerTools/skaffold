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
package sync

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewSyncItem(t *testing.T) {
	var tests = []struct {
		description string
		artifact    *latest.Artifact
		evt         watch.Events
		builds      []build.Artifact
		shouldErr   bool
		expected    *Item
	}{
		{
			description: "match copy",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added: []string{"index.html"},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					"index.html": "index.html",
				},
				Delete: map[string]string{},
			},
		},
		{
			description: "no tag for image",
			artifact: &latest.Artifact{
				ImageName: "notbuildyet",
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added: []string{"index.html"},
			},
			shouldErr: true,
		},
		{
			description: "sync all",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*": ".",
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
				Deleted:  []string{filepath.Join("node", "package.json")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					filepath.Join("node", "server.js"):  "server.js",
					filepath.Join("node", "index.html"): "index.html",
				},
				Delete: map[string]string{
					filepath.Join("node", "package.json"): "package.json",
				},
			},
		},
		{
			description: "not copy syncable",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"main.go"},
				Deleted: []string{"index.html"},
			},
		},
		{
			description: "not delete syncable",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": "/static",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
		},
		{
			description: "err bad pattern",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"[*.html": "*",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
			shouldErr: true,
		},
		{
			description: "no change no sync",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": "*",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := NewItem(test.artifact, test.evt, test.builds)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}

}

func TestIntersect(t *testing.T) {
	var tests = []struct {
		description  string
		syncPatterns map[string]string
		files        []string
		context      string
		expected     map[string]string
		shouldErr    bool
	}{
		{
			description: "nil sync patterns doesn't sync",
			expected:    map[string]string{},
		},
		{
			description: "copy nested file to correct destination",
			files:       []string{filepath.Join("static", "index.html"), filepath.Join("static", "test.html")},
			syncPatterns: map[string]string{
				filepath.Join("static", "*.html"): "/html",
			},
			expected: map[string]string{
				filepath.Join("static", "index.html"): "/html/index.html",
				filepath.Join("static", "test.html"):  "/html/test.html",
			},
		},
		{
			description: "file not in . copies to correct destination",
			files:       []string{filepath.Join("node", "server.js")},
			context:     "node",
			syncPatterns: map[string]string{
				"*.js": "/",
			},
			expected: map[string]string{
				filepath.Join("node", "server.js"): "/server.js",
			},
		},
		{
			description: "file change not relative to context throws error",
			files:       []string{filepath.Join("node", "server.js"), filepath.Join("/", "something", "test.js")},
			context:     "node",
			syncPatterns: map[string]string{
				"*.js": "/",
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := intersect(test.context, test.syncPatterns, test.files)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}
