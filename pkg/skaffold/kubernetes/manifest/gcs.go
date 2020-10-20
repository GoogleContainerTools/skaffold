/*
Copyright 2020 The Skaffold Authors

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

package manifest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DownloadFromGCS downloads all provided manifests from a remote GCS bucket,
// and returns a relative path pointing to the GCS temp dir.
func DownloadFromGCS(manifests []string) (string, error) {
	if err := os.MkdirAll(ManifestTmpDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create the tmp directory: %w", err)
	}
	for _, manifest := range manifests {
		if manifest == "" || !strings.HasPrefix(manifest, gcsPrefix) {
			return "", fmt.Errorf("%v is not a valid GCS path", manifest)
		}
		gcs := util.Gsutil{}
		if err := gcs.Copy(context.Background(), manifest, ManifestTmpDir, true); err != nil {
			return "", fmt.Errorf("failed to download manifests fom GCS: %w", err)
		}
	}
	return ManifestTmpDir, nil
}
