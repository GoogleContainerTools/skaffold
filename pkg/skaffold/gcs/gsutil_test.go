/*
Copyright 2023 The Skaffold Authors

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

package gcs

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	file      = "source/file"
	gcsFile   = "gs://bucket/file"
	folder    = "source/"
	gcsFolder = "gs://bucket/folder/"
)

func TestCopy(t *testing.T) {
	tests := []struct {
		description string
		src         string
		dst         string
		commands    util.Command
		recursive   bool
		shouldErr   bool
	}{
		{
			description: "copy single file",
			src:         file,
			dst:         gcsFile,
			commands:    testutil.CmdRunOut(fmt.Sprintf("gsutil cp %s %s", file, gcsFile), "logs"),
		},
		{
			description: "copy recursively",
			src:         folder,
			dst:         gcsFolder,
			commands:    testutil.CmdRunOut(fmt.Sprintf("gsutil cp -r %s %s", folder, gcsFolder), "logs"),
			recursive:   true,
		},
		{
			description: "copy failed",
			src:         file,
			dst:         gcsFile,
			commands:    testutil.CmdRunOutErr(fmt.Sprintf("gsutil cp %s %s", file, gcsFile), "logs", fmt.Errorf("file not found")),
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			gcs := Gsutil{}
			err := gcs.Copy(context.Background(), test.src, test.dst, test.recursive)

			t.CheckError(test.shouldErr, err)
		})
	}
}

type gcsClientMock struct {
	err error
}

func (g gcsClientMock) DownloadRecursive(ctx context.Context, src, dst string) error {
	return g.err
}

func TestSyncObject(t *testing.T) {
	source := "gs://my-bucket/dir1/*"
	path := "configs/skaffold.yaml"
	sourceHash := "B7fSU6BUuHFJuLenty9ErOwyxPVqO5cB"

	tests := []struct {
		description string
		g           latest.GoogleCloudStorageInfo
		gsutilErr   error
		syncFlag    string
		existing    bool
		shouldErr   bool
		expected    string
	}{
		{
			description: "first time copy succeeds",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path},
			syncFlag:    "always",
			expected:    sourceHash,
		},
		{
			description: "first time copy fails",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path},
			gsutilErr:   fmt.Errorf("not found"),
			syncFlag:    "always",
			shouldErr:   true,
		},
		{
			description: "first time copy with sync off via flag fails",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path},
			syncFlag:    "never",
			shouldErr:   true,
		},
		{
			description: "existing copy update succeeds",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path},
			syncFlag:    "always",
			existing:    true,
			expected:    sourceHash,
		},
		{
			description: "existing copy with sync off via flag succeeds",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path},
			syncFlag:    "never",
			existing:    true,
			expected:    sourceHash,
		},
		{
			description: "existing copy with sync off succeeds",
			g:           latest.GoogleCloudStorageInfo{Source: source, Path: path, Sync: util.Ptr(false)},
			syncFlag:    "always",
			existing:    true,
			expected:    sourceHash,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			td := t.NewTempDir()
			if test.existing {
				td.Touch(sourceHash)
			}

			syncRemote := &config.SyncRemoteCacheOption{}
			_ = syncRemote.Set(test.syncFlag)
			opts := config.SkaffoldOptions{RemoteCacheDir: td.Root(), SyncRemoteCache: *syncRemote}

			gcsClient := gcsClientMock{}
			if test.gsutilErr != nil {
				gcsClient.err = test.gsutilErr
			}
			t.Override(&GetGCSClient, func() gscClient {
				return gcsClient
			})

			path, err := SyncObjects(context.Background(), test.g, opts)
			var expected string
			if !test.shouldErr {
				expected = filepath.Join(td.Root(), test.expected)
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expected, path)
		})
	}
}
