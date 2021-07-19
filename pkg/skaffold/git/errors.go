/*
Copyright 2021 The Skaffold Authors

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

package git

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// SyncOffErr returns error when git repository sync is turned off by the user but the repository clone doesn't exist inside the cache directory.
func SyncOffErr(g latestV1.GitInfo, repoCacheDir string) error {
	msg := fmt.Sprintf("cache directory %q for repository %q at ref %q does not exist, and repo sync is explicitly turned off via flag `--sync-remote-cache`", repoCacheDir, g.Repo, g.Ref)
	return sErrors.NewError(fmt.Errorf(msg),
		proto.ActionableErr{
			Message: msg,
			ErrCode: proto.StatusCode_CONFIG_REMOTE_REPO_CACHE_NOT_FOUND_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_ENABLE_REMOTE_REPO_SYNC,
					Action:         fmt.Sprintf("Either clone the repository manually inside %q, or set flag `--sync-remote-cache` to `always` or `missing`", repoCacheDir),
				},
			},
		})
}
