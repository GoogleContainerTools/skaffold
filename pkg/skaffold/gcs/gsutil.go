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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
)

const GsutilExec = "gsutil"

type Gsutil struct{}

// Copy calls `gsutil cp [-r] <source_url> <destination_url>
func (g *Gsutil) Copy(ctx context.Context, src, dst string, recursive bool) error {
	args := []string{"cp"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, src, dst)
	cmd := exec.CommandContext(ctx, GsutilExec, args...)
	out, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return fmt.Errorf("copy file(s) with %s failed: %w", GsutilExec, err)
	}
	log.Entry(ctx).Info(out)
	return nil
}

// SyncObject syncs the target Google Cloud Storage object with skaffold's local cache and returns the local path to the object.
func SyncObject(ctx context.Context, g latest.GoogleCloudStorageInfo, opts config.SkaffoldOptions) (string, error) {
	remoteCacheDir, err := config.GetRemoteCacheDir(opts)
	if err != nil {
		return "", fmt.Errorf("failed determining Google Cloud Storage object cache directory: %w", err)
	}
	if err := os.MkdirAll(remoteCacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed creating Google Cloud Storage object cache directory: %w", err)
	}

	subDir, err := getPerObjectCacheDir(g)
	if err != nil {
		return "", fmt.Errorf("failed determining Google Cloud Storage object cache directory for %s: %w", g.Path, err)
	}
	// Determine the name of the file to preserve it in the cache.
	parts := strings.Split(g.Path, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("failed parsing Google Cloud Storage object %s", g.Path)
	}
	fileName := parts[len(parts)-1]
	if len(fileName) == 0 {
		return "", fmt.Errorf("failed parsing Google Cloud Storage object %s", g.Path)
	}
	cacheDir := filepath.Join(remoteCacheDir, subDir, fileName)
	// If cache doesn't exist and cloning is disabled then we can't move forward.
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) && opts.SyncRemoteCache.CloneDisabled() {
		return "", syncDisabledErr(g, cacheDir)
	}
	// If sync property is false then skip fetching latest object from remote storage.
	if g.Sync != nil && !*g.Sync {
		return cacheDir, nil
	}
	// If sync is turned off via flag `--sync-remote-cache` then skip fetching latest object from remote storage.
	if opts.SyncRemoteCache.FetchDisabled() {
		return cacheDir, nil
	}

	gcs := Gsutil{}
	// Non-recursive since this should only download a single file - the remote Skaffold Config.
	if err := gcs.Copy(ctx, g.Path, cacheDir, false); err != nil {
		return "", fmt.Errorf("failed to cache Google Cloud Storage object %s: %w", g.Path, err)
	}
	return cacheDir, nil
}

// getPerObjectCacheDir returns the directory used per Google Cloud Storage Object. Directory is a hash of the path provided.
func getPerObjectCacheDir(g latest.GoogleCloudStorageInfo) (string, error) {
	inputs := []string{g.Path}
	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))[:32], nil
}

// syncDisabledErr returns error to use when remote sync is turned off by the user and the Google Cloud Storage object doesn't exist inside the cache directory.
func syncDisabledErr(g latest.GoogleCloudStorageInfo, cacheDir string) error {
	msg := fmt.Sprintf("cache directory %q for Google Cloud Storage object %q does not exist and remote cache sync is explicitly disabled via flag `--sync-remote-cache`", cacheDir, g.Path)
	return sErrors.NewError(fmt.Errorf(msg),
		&proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode(proto.StatusCode_CONFIG_REMOTE_REPO_CACHE_NOT_FOUND_ERR),
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_ENABLE_REMOTE_REPO_SYNC,
					Action:         fmt.Sprintf("Either download the Google Cloud Storage object manually to %q or set flag `--sync-remote-cache` to `always` or `missing`", cacheDir),
				},
			},
		})
}
