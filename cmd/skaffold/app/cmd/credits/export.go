/*
Copyright 2019 The Skaffold Authors

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

package credits

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/rakyll/statik/fs"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/statik"
)

var Path string

// Export writes all the licenses and credit files to the `Path` folder.
func Export(ctx context.Context, out io.Writer) error {
	statikFS, err := statik.FS()
	if err != nil {
		return fmt.Errorf("opening embedded filesystem: %w", err)
	}

	if err := fs.Walk(statikFS, "/skaffold-credits", func(filePath string, fileInfo os.FileInfo, err error) error {
		newPath := path.Join(Path, "..", filePath)
		if fileInfo.IsDir() {
			err := os.Mkdir(newPath, 0755)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("creating directory %q: %w", newPath, err)
			}
		} else {
			file, err := statikFS.Open(filePath)
			if err != nil {
				return fmt.Errorf("opening %q in embedded filesystem: %w", filePath, err)
			}

			buf, err := ioutil.ReadAll(file)
			if err != nil {
				return fmt.Errorf("reading %q in embedded filesystem: %w", filePath, err)
			}

			if err := ioutil.WriteFile(newPath, buf, 0664); err != nil {
				return fmt.Errorf("writing %q to %q: %w", filePath, newPath, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	s, err := filepath.Abs(Path)
	if err != nil {
		return err
	}

	log.Printf("Successfully exported third party notices to %s", s)
	return nil
}
