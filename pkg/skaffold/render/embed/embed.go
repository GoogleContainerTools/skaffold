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
package embed

import (
	_ "embed" //nolint https://github.com/golangci/golangci-lint/issues/1727
	"fmt"
	"os"
	"path/filepath"
)

//go:embed kpt
var kpt []byte

// UseBuiltinKpt guarantees the `kpt` always exists in $PATH/bin. `kpt` is needed in skaffold render.
func UseBuiltinKpt() error {
	err := os.WriteFile(getKptInstallPath(), kpt, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to add builtin kpt")
	}
	return nil
}

func getKptInstallPath() string {
	paths := filepath.SplitList(os.Getenv("PATH"))
	dir := paths[0]
	if dir == "" {
		// Unix shell semantics: path element "" means "."
		dir = "."
	}
	return filepath.Join(dir, "kpt")
}
