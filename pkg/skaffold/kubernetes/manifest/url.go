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

package manifest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

var ManifestsFromURL = "manifests_from_url"

// DownloadFromURL downloads the provided manifests from their URLs,
// and returns a slice containing downloaded file destinations.
func DownloadFromURL(manifests []string) ([]string, error) {
	dir := filepath.Join(ManifestTmpDir, ManifestsFromURL)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create the tmp directory: %w", err)
	}
	var paths []string
	for _, manifest := range manifests {
		out, err := downloadFromURL(dir, manifest)

		if err != nil {
			return nil, err
		}
		paths = append(paths, out)
	}
	return paths, nil
}

func downloadFromURL(destDir string, manifest string) (string, error) {
	if manifest == "" || !util.IsURL(manifest) {
		return "", fmt.Errorf("%s is not a valid URL", manifest)
	}

	resp, err := http.Get(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to download manifest from %s, err : %w", manifest, err)
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp(destDir, "*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write manifest to file, err: %w", err)
	}

	return f.Name(), nil
}
