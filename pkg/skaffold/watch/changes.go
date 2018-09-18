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
	"time"

	"github.com/pkg/errors"
)

type fileMap struct {
	count        int
	lastModified time.Time
}

func stat(deps func() ([]string, error)) (fileMap, error) {
	paths, err := deps()
	if err != nil {
		return fileMap{}, errors.Wrap(err, "listing files")
	}

	last, err := lastModified(paths)
	if err != nil {
		return fileMap{}, err
	}

	return fileMap{
		count:        len(paths),
		lastModified: last,
	}, nil
}

func lastModified(paths []string) (time.Time, error) {
	var last time.Time

	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Ignore files that don't exist
			}

			return last, errors.Wrapf(err, "unable to stat file %s", path)
		}

		if stat.IsDir() {
			continue // Ignore time changes on directories
		}

		modTime := stat.ModTime()
		if modTime.After(last) {
			last = modTime
		}
	}

	return last, nil
}

func hasChanged(prev, curr fileMap) bool {
	return prev.count != curr.count || !prev.lastModified.Equal(curr.lastModified)
}
