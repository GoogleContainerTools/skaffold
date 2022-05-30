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
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DownloadFromURL downloads the provided manifests from their URLs,
// and returns a relative path pointing to the URL temp dir.
func DownloadFromURL(manifests []string) (string, error) {
	if err := os.MkdirAll(ManifestTmpDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create the tmp directory: %w", err)
	}
	for _, manifest := range manifests {
		if manifest == "" || !util.IsURL(manifest) {
			return "", fmt.Errorf("%v is not a valid URL", manifest)
		}
		resp, err := http.Get(manifest)
		if err != nil {
			return "", fmt.Errorf("failed to download manifest fom URL: %w", err)
		}
		defer resp.Body.Close()
		// TODO discuss about file names and deletion
		// fileName := path.Base(manifest)

		// destinationPath := path.Join(ManifestTmpDir, fileName)
		// f, err := os.Create(destinationPath)
		f, err := os.CreateTemp(ManifestTmpDir, "url-manifest-")
		if err != nil {
			return "", fmt.Errorf("failed to create manifest file: %w", err)
		}
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return "", fmt.Errorf("copying manifest file failed: %w", err)
		}
	}
	return ManifestTmpDir, nil
}
