/*
Copyright 2022 The Skaffold Authors

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
	"context"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestInferSync(t *testing.T) {
	tests := []struct {
		description string
		artifact    *latest.Artifact
		files       []string
		expected    map[string][]string
	}{
		{
			description: "matching and non-matching files",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
				Sync: &latest.Sync{
					Infer: []string{"kodata/**/*.js"},
				},
			},
			files: []string{
				"kodata/foo.js",
				"kodata/bar.css",
				"kodata/baz/frob.js",
				"main.go",
			},
			expected: map[string][]string{
				"kodata/foo.js":      {"/var/run/ko/foo.js"},
				"kodata/baz/frob.js": {"/var/run/ko/baz/frob.js"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := inferSync(context.TODO(), test.artifact, test.files)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestSyncDest(t *testing.T) {
	tests := []struct {
		description   string
		file          string
		workspace     string
		localBasePath string
		patterns      []string
		expectedDest  string
		wantErr       bool
	}{
		{
			description:   "matching pattern simple",
			file:          "kodata/page.html",
			localBasePath: "kodata",
			patterns:      []string{"kodata/page.html"},
			expectedDest:  "/var/run/ko/page.html",
		},
		{
			description:   "matching pattern doublestar",
			file:          "kodata/page.html",
			localBasePath: "kodata",
			patterns:      []string{"kodata/**/*"},
			expectedDest:  "/var/run/ko/page.html",
		},
		{
			description:   "both non-matching and matching pattern",
			file:          "kodata/page.html",
			localBasePath: "kodata",
			patterns:      []string{"kodata/*.css", "kodata/**/*"},
			expectedDest:  "/var/run/ko/page.html",
		},
		{
			description:   "no matching pattern",
			file:          "kodata/page.html",
			localBasePath: "kodata",
			expectedDest:  "",
		},
		{
			description:   "non-default directories",
			file:          "workspace/cmd/foo/kodata/page.html",
			workspace:     "workspace",
			localBasePath: "workspace/cmd/foo/kodata",
			patterns:      []string{"cmd/foo/kodata/**/*"},
			expectedDest:  "/var/run/ko/page.html",
		},
		{
			description:   "error on invalid pattern",
			file:          "kodata/page.html",
			localBasePath: "kodata",
			patterns:      []string{"["},
			wantErr:       true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actualDest, err := syncDest(test.file, test.workspace, test.localBasePath, test.patterns)
			t.CheckError(test.wantErr, err)
			t.CheckDeepEqual(test.expectedDest, actualDest)
		})
	}
}

func TestFindLocalKodataPath(t *testing.T) {
	tests := []struct {
		description  string
		artifact     *latest.Artifact
		expectedPath string
		wantErr      bool
	}{
		{
			description: "default",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
			},
			expectedPath: "kodata",
		},
		{
			description: "all values set",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						Dir:  "dir",
						Main: "main",
					},
				},
				Workspace: "workspace",
			},
			expectedPath: filepath.Join("workspace", "dir", "main", "kodata"),
		},
		{
			description: "error due to main wildcard",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						Main: "./...",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actualPath, err := findLocalKodataPath(test.artifact)
			t.CheckError(test.wantErr, err)
			t.CheckDeepEqual(test.expectedPath, actualPath)
		})
	}
}
