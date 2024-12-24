/*
Copyright 2024 The Skaffold Authors

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

package client

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/bmatcuk/doublestar"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

// Regex to transform a/b** to a/b*/**
var escapeDoubleStarWithoutSlashRegex = regexp.MustCompile(`([^/])\*\*`)

type repoHandlerMock struct {
	root            string
	downloadedFiles map[string]string
	uploadedFile    string
}

func (r *repoHandlerMock) filterOnlyFiles(paths []string) ([]string, error) {
	matches := []string{}
	for _, m := range paths {
		fileInfo, err := os.Stat(m)
		if err != nil {
			return nil, err
		}

		if !fileInfo.IsDir() {
			withoutPrefix := strings.TrimPrefix(m, r.root+string(filepath.Separator))
			matches = append(matches, filepath.ToSlash(withoutPrefix))
		}
	}
	return matches, nil
}

func (r *repoHandlerMock) matchGlob(matchGlob string) ([]string, error) {
	glob := escapeDoubleStarWithoutSlashRegex.ReplaceAllString(matchGlob, `${1}*/**`)
	glob = filepath.Join(r.root, glob)
	globMatches, err := doublestar.Glob(glob)
	if err != nil {
		return nil, err
	}

	return r.filterOnlyFiles(globMatches)
}

func (r *repoHandlerMock) withPrefix(prefix string) ([]string, error) {
	path := filepath.Join(r.root, prefix+"**")
	matches, err := doublestar.Glob(path)
	if err != nil {
		return nil, err
	}

	return r.filterOnlyFiles(matches)
}

func (r *repoHandlerMock) ListObjects(ctx context.Context, q *storage.Query) ([]string, error) {
	if q.MatchGlob != "" {
		return r.matchGlob(q.MatchGlob)
	}

	if q.Prefix != "" {
		return r.withPrefix(q.Prefix)
	}

	return nil, nil
}

func (r *repoHandlerMock) DownloadObject(ctx context.Context, localPath, uri string) error {
	r.downloadedFiles[uri] = localPath
	return nil
}

func (r *repoHandlerMock) UploadObject(ctx context.Context, objName string, content *os.File) error {
	r.uploadedFile = objName
	return nil
}

func (r *repoHandlerMock) Close() {}

func TestDownloadRecursive(t *testing.T) {
	tests := []struct {
		name                    string
		uri                     string
		dst                     string
		availableFiles          []string
		expectedDownloadedFiles map[string]string
	}{
		{
			name: "exact match with flat output",
			uri:  "gs://bucket/dir1/manifest1.yaml",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"manifest2.yaml",
				"main.go",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/manifest1.yaml",
			},
		},
		{
			name: "exact match with wildcard, flat output",
			uri:  "gs://bucket/*/manifest[12].yaml",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir1/manifest3.yaml",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/manifest1.yaml",
				"dir2/manifest2.yaml": "download/manifest2.yaml",
			},
		},
		{
			name: "exact match with ? wildcard with flat output",
			uri:  "gs://bucket/*/manifest?.yaml",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/manifest1.yaml",
				"dir2/manifest2.yaml": "download/manifest2.yaml",
			},
		},
		{
			name: "recursive match with folders creation",
			uri:  "gs://bucket/dir*",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/dir1/manifest1.yaml",
				"dir2/manifest2.yaml": "download/dir2/manifest2.yaml",
			},
		},
		{
			name: "recursive match with flat output",
			uri:  "gs://bucket/dir**",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/manifest1.yaml",
				"dir2/manifest2.yaml": "download/manifest2.yaml",
			},
		},
		{
			name: "recursive match from bucket with folders creation",
			uri:  "gs://bucket",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/bucket/dir1/manifest1.yaml",
				"dir2/manifest2.yaml": "download/bucket/dir2/manifest2.yaml",
			},
		},
		{
			name: "recursive match all bucket content with folders creation",
			uri:  "gs://bucket/*",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir1/sub1/main.go",
				"dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/dir1/manifest1.yaml",
				"dir1/sub1/main.go":   "download/dir1/sub1/main.go",
				"dir2/manifest2.yaml": "download/dir2/manifest2.yaml",
			},
		},
		{
			name: "recursive match all bucket content with flat structure",
			uri:  "gs://bucket/**",
			dst:  "download",
			availableFiles: []string{
				"dir1/manifest1.yaml",
				"dir1/sub1/main.go",
				"manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"dir1/manifest1.yaml": "download/manifest1.yaml",
				"dir1/sub1/main.go":   "download/main.go",
				"manifest2.yaml":      "download/manifest2.yaml",
			},
		},
		{
			name: "recursive match with folder creating and prefix removal",
			uri:  "gs://bucket/submodule/*/content/*",
			dst:  "download",
			availableFiles: []string{
				"submodule/a/content/dir1/manifest1.yaml",
				"submodule/b/content/dir2/manifest2.yaml",
				"submodule/c/content/dir3/Dockerfile",
				"submodule/dir4/main.go",
			},
			expectedDownloadedFiles: map[string]string{
				"submodule/a/content/dir1/manifest1.yaml": "download/dir1/manifest1.yaml",
				"submodule/b/content/dir2/manifest2.yaml": "download/dir2/manifest2.yaml",
				"submodule/c/content/dir3/Dockerfile":     "download/dir3/Dockerfile",
			},
		},
		{
			name: "recursive match with matching folder creating and prefix removal",
			uri:  "gs://bucket/submodule/*/content*",
			dst:  "download",
			availableFiles: []string{
				"submodule/a/content1/dir1/manifest1.yaml",
				"submodule/b/content2/dir2/manifest2.yaml",
			},
			expectedDownloadedFiles: map[string]string{
				"submodule/a/content1/dir1/manifest1.yaml": "download/content1/dir1/manifest1.yaml",
				"submodule/b/content2/dir2/manifest2.yaml": "download/content2/dir2/manifest2.yaml",
			},
		},
		{
			name: "no match",
			uri:  "gs://bucket/**/*.go",
			dst:  "download",
			availableFiles: []string{
				"submodule/a/content/dir1/manifest1.yaml",
				"submodule/b/content/dir2/manifest2.yaml",
				"submodule/c/content/dir3/Dockerfile",
			},
			expectedDownloadedFiles: map[string]string{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			td := t.NewTempDir()
			td.Touch(test.availableFiles...)
			root := td.Root()

			rh := &repoHandlerMock{
				root:            root,
				downloadedFiles: make(map[string]string),
			}
			t.Override(&GetBucketManager, func(ctx context.Context, bucketName string) (bucketHandler, error) {
				return rh, nil
			})

			n := Native{}
			err := n.DownloadRecursive(context.TODO(), test.uri, test.dst)
			t.CheckNoError(err)

			for uri, local := range test.expectedDownloadedFiles {
				test.expectedDownloadedFiles[uri] = filepath.FromSlash(local)
			}

			t.CheckMapsMatch(test.expectedDownloadedFiles, rh.downloadedFiles)
		})
	}
}

func TestUploadFile(t *testing.T) {
	tests := []struct {
		name                string
		uri                 string
		localFile           string
		availableFiles      []string
		expectedCreatedFile string
		shouldError         bool
	}{
		{
			name:      "upload file to existing folder using local name",
			uri:       "gs://bucket/folder",
			localFile: "manifest.yaml",
			availableFiles: []string{
				"folder/main.go",
			},
			expectedCreatedFile: "folder/manifest.yaml",
		},
		{
			name:      "upload file to existing folder using new name",
			uri:       "gs://bucket/folder/newmanifest.yaml",
			localFile: "manifest.yaml",
			availableFiles: []string{
				"folder/main.go",
			},
			expectedCreatedFile: "folder/newmanifest.yaml",
		},
		{
			name:      "upload file to not existing subfolder using local name",
			uri:       "gs://bucket/folder/newfolder/",
			localFile: "manifest.yaml",
			availableFiles: []string{
				"folder/main.go",
			},
			expectedCreatedFile: "folder/newfolder/manifest.yaml",
		},
		{
			name:      "upload file to not existing subfolder using new name",
			uri:       "gs://bucket/folder/newfolder/newmanifest.yaml",
			localFile: "manifest.yaml",
			availableFiles: []string{
				"folder/main.go",
			},
			expectedCreatedFile: "folder/newfolder/newmanifest.yaml",
		},
		{
			name:                "upload file to root of bucket",
			uri:                 "gs://bucket",
			localFile:           "manifest.yaml",
			expectedCreatedFile: "manifest.yaml",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			bucketTd := t.NewTempDir()
			bucketTd.Touch(test.availableFiles...)
			bucketRoot := bucketTd.Root()
			rh := &repoHandlerMock{
				root:            bucketRoot,
				downloadedFiles: make(map[string]string),
			}
			t.Override(&GetBucketManager, func(ctx context.Context, bucketName string) (bucketHandler, error) {
				return rh, nil
			})

			localFileTd := t.NewTempDir()
			localFileTd.Touch(test.localFile)
			locaFullPath := filepath.Join(localFileTd.Root(), test.localFile)

			n := Native{}
			err := n.UploadFile(context.TODO(), locaFullPath, test.uri)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedCreatedFile, rh.uploadedFile)
		})
	}
}
