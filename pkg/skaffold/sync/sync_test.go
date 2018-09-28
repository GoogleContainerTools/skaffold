package sync

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShouldSync(t *testing.T) {
	var tests = []struct {
		description string
		artifact    *v1alpha3.Artifact
		evt         watch.Events
		shouldErr   bool
		expected    *SyncItem
	}{
		{
			description: "match copy",
			artifact: &v1alpha3.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added: []string{"index.html"},
			},
			expected: &SyncItem{
				Image: "test",
				Copy: map[string]string{
					"index.html": "index.html",
				},
				Delete: map[string]string{},
			},
		},
		{
			description: "sync all",
			artifact: &v1alpha3.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*": ".",
				},
				Workspace: "node",
			},
			evt: watch.Events{
				Added:    []string{"node/index.html"},
				Modified: []string{"node/server.js"},
				Deleted:  []string{"node/package.json"},
			},
			expected: &SyncItem{
				Image: "test",
				Copy: map[string]string{
					"node/server.js":  "server.js",
					"node/index.html": "index.html",
				},
				Delete: map[string]string{
					"node/package.json": "package.json",
				},
			},
		},
		{
			description: "not copy syncable",
			artifact: &v1alpha3.Artifact{
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
			artifact: &v1alpha3.Artifact{
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
			artifact: &v1alpha3.Artifact{
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
			artifact: &v1alpha3.Artifact{
				Sync: map[string]string{
					"[*.html": "*",
				},
				Workspace: ".",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := NewSyncItem(test.artifact, test.evt)
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
			files:       []string{"static/index.html", "static/test.html"},
			syncPatterns: map[string]string{
				"static/*.html": "/html",
			},
			expected: map[string]string{
				"static/index.html": "/html/index.html",
				"static/test.html":  "/html/test.html",
			},
		},
		{
			description: "file not in . copies to correct destination",
			files:       []string{"node/server.js"},
			context:     "node",
			syncPatterns: map[string]string{
				"*.js": "/",
			},
			expected: map[string]string{
				"node/server.js": "/server.js",
			},
		},
		{
			description: "file change not relative to context throws error",
			files:       []string{"node/server.js", "/something/test.js"},
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
