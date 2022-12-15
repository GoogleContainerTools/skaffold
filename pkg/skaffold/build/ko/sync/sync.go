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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// kodataRoot is the directory in the container where ko places static assets.
// See https://github.com/google/ko/blob/2f230b88c4891ee3a71b01c1fa65e85e8d6b5f5b/README.md#static-assets
// and https://github.com/google/ko/blob/2f230b88c4891ee3a71b01c1fa65e85e8d6b5f5b/pkg/build/gobuild.go#L514
const kodataRoot = "/var/run/ko"

// Infer syncs static content in the kodata directory based on matching file name patterns.
// It returns maps of files to be copied and deleted.
func Infer(ctx context.Context, a *latest.Artifact, e filemon.Events) (toCopy map[string][]string, toDelete map[string][]string, err error) {
	toCopy, err = inferSync(ctx, a, append(e.Modified, e.Added...))
	if err != nil {
		return nil, nil, err
	}
	toDelete, err = inferSync(ctx, a, e.Deleted)
	if err != nil {
		return nil, nil, err
	}
	return
}

// inferSync determines if the files match any of the inferred file sync patterns configured for the artifact.
// For files that matches at least one pattern, the function determines the destination path.
// The return value is a map of source file location to destination path.
func inferSync(ctx context.Context, a *latest.Artifact, files []string) (map[string][]string, error) {
	localBasePath, err := findLocalKodataPath(a)
	if err != nil {
		return nil, err
	}
	toSync := map[string][]string{}
	for _, f := range files {
		dest, err := syncDest(f, a.Workspace, localBasePath, a.Sync.Infer)
		if err != nil {
			return nil, err
		}
		if dest != "" {
			log.Entry(ctx).Debugf("Syncing %q to %q", f, dest)
			toSync[f] = []string{dest}
		} else {
			log.Entry(ctx).Debugf("File %q does not match any sync pattern. Skipping sync", f)
		}
	}
	return toSync, nil
}

// syncDest returns the destination file paths if the input file path matches at least one of the patterns.
// If the file doesn't match any of the patterns, the function returns zero values.
func syncDest(f string, workspace string, localBasePath string, patterns []string) (string, error) {
	relPath, err := filepath.Rel(workspace, f)
	if err != nil {
		return "", err
	}
	for _, p := range patterns {
		matches, err := doublestar.PathMatch(filepath.FromSlash(p), relPath)
		if err != nil {
			return "", fmt.Errorf("pattern error for file %q and pattern %s: %w", relPath, p, err)
		}
		if matches {
			// find path to file relative to local static file directory
			localFile, err := filepath.Rel(localBasePath, f)
			if err != nil {
				return "", fmt.Errorf("relative path error for path %q and file %q: %w", localBasePath, f, err)
			}
			dest := strings.ReplaceAll(filepath.Join(kodataRoot, localFile), "\\", "/")
			return dest, nil
		}
	}
	return "", nil
}

// findLocalKodataPath returns the local path to static content for ko artifacts.
func findLocalKodataPath(a *latest.Artifact) (string, error) {
	if strings.Contains(a.KoArtifact.Main, "...") {
		// this error should be caught by validation earlier
		return "", fmt.Errorf("unable to infer file sync when ko.main contains the '...' wildcard")
	}
	path := filepath.Join(a.Workspace, a.KoArtifact.Dir, a.KoArtifact.Main, "kodata")
	return path, nil
}
