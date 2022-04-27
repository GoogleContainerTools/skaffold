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

package schema

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/GoogleContainerTools/skaffold/fs"
)

// Print prints the json schema for a given version.
func Print(out io.Writer, version string) error {
	filename := path.Join("assets/schemas_generated", strings.TrimPrefix(version, "skaffold/")+".json")

	content, err := fs.AssetsFS.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("schema %q not found: %w", version, err)
	}

	_, err = out.Write(content)
	return err
}
