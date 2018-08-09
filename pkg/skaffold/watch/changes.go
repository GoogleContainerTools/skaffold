/*
Copyright 2018 The Skaffold Authors

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

package watch

import (
	"os"

	"github.com/pkg/errors"
)

type fileMap map[string]os.FileInfo

func stat(deps func() ([]string, error)) (fileMap, error) {
	paths, err := deps()
	if err != nil {
		return nil, errors.Wrap(err, "listing files")
	}

	fm := make(fileMap)

	for _, path := range paths {
		fm[path], err = os.Stat(path)
		if err != nil {
			return nil, errors.Wrapf(err, "stating file [%s]", path)
		}
	}

	return fm, nil
}

func hasChanged(prev, curr fileMap) bool {
	if len(prev) != len(curr) {
		return true
	}

	for k, prevV := range prev {
		currV, ok := curr[k]
		if !ok {
			// Deleted
			return true
		}
		if prevV.ModTime() != currV.ModTime() {
			// Ignore directory time changes
			if !currV.IsDir() && !prevV.IsDir() {
				return true
			}
		}
	}
	for k := range curr {
		if _, ok := prev[k]; !ok {
			// Created
			return true
		}
	}

	return false
}
