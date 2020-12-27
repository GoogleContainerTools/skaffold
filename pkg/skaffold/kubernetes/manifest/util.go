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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	manifestsStagingFolder       = "manifest_tmp"
	renderedManifestsStagingFile = "rendered_manifest.yaml"
	gcsPrefix                    = "gs://"
)

var ManifestTmpDir = filepath.Join(os.TempDir(), manifestsStagingFolder)

// Write writes manifests to a file, a writer or a GCS bucket.
func Write(manifests string, output string, manifestOut io.Writer) error {
	switch {
	case output == "":
		_, err := fmt.Fprintln(manifestOut, manifests)
		return err
	case strings.HasPrefix(output, gcsPrefix):
		tempDir, err := ioutil.TempDir("", manifestsStagingFolder)
		if err != nil {
			return writeErr(fmt.Errorf("failed to create the tmp directory: %w", err))
		}
		defer os.RemoveAll(tempDir)
		tempFile := filepath.Join(tempDir, renderedManifestsStagingFile)
		if err := dumpToFile(manifests, tempFile); err != nil {
			return err
		}
		gcs := util.Gsutil{}
		if err := gcs.Copy(context.Background(), tempFile, output, false); err != nil {
			return writeErr(fmt.Errorf("failed to copy rendered manifests to GCS: %w", err))
		}
		return nil
	default:
		return dumpToFile(manifests, output)
	}
}

func dumpToFile(manifests string, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("opening file for writing manifests: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(manifests + "\n")
	return err
}
