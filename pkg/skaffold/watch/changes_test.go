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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/karrick/godirwalk"
)

func TestHasChanged(t *testing.T) {
	var tests = []struct {
		description     string
		setup           func(folder string) error
		update          func(folder string) error
		expectedChanged bool
	}{
		{
			description: "no file",
			setup:       func(string) error { return nil },
			update:      func(string) error { return nil },
		},
		{
			description:     "added",
			expectedChanged: true,
			setup:           func(string) error { return nil },
			update: func(folder string) error {
				return ioutil.WriteFile(filepath.Join(folder, "added.txt"), []byte{}, os.ModePerm)
			},
		},
		{
			description:     "removed",
			expectedChanged: true,
			setup: func(folder string) error {
				return ioutil.WriteFile(filepath.Join(folder, "removed.txt"), []byte{}, os.ModePerm)
			},
			update: func(folder string) error {
				return os.Remove(filepath.Join(folder, "removed.txt"))
			},
		},
		{
			description:     "modified",
			expectedChanged: true,
			setup: func(folder string) error {
				return ioutil.WriteFile(filepath.Join(folder, "file.txt"), []byte("initial"), os.ModePerm)
			},
			update: func(folder string) error {
				return os.Chtimes(filepath.Join(folder, "file.txt"), time.Now(), time.Now().Add(2*time.Second))
			},
		},
		{
			description:     "removed and added",
			expectedChanged: true,
			setup: func(folder string) error {
				return ioutil.WriteFile(filepath.Join(folder, "removed.txt"), []byte{}, os.ModePerm)
			},
			update: func(folder string) error {
				err := os.Remove(filepath.Join(folder, "removed.txt"))
				if err != nil {
					return err
				}
				err = ioutil.WriteFile(filepath.Join(folder, "added.txt"), []byte{}, os.ModePerm)
				if err != nil {
					return err
				}
				return os.Chtimes(filepath.Join(folder, "added.txt"), time.Now(), time.Now().Add(2*time.Second))
			},
		},
		{
			description:     "ignore modified directory",
			expectedChanged: false,
			setup: func(folder string) error {
				return os.Mkdir(filepath.Join(folder, "dir"), os.ModePerm)
			},
			update: func(folder string) error {
				return os.Chtimes(filepath.Join(folder, "dir"), time.Now(), time.Now().Add(2*time.Second))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tmpDir, cleanup := testutil.TempDir(t)
			defer cleanup()

			if err := test.setup(tmpDir); err != nil {
				t.Fatal("Unable to setup test directory", err)
			}
			prev, err := stat(listFiles(tmpDir))
			if err != nil {
				t.Fatal("Unable to setup test directory", err)
			}

			if err := test.update(tmpDir); err != nil {
				t.Fatal("Unable to update test directory", err)
			}
			curr, err := stat(listFiles(tmpDir))
			if err != nil {
				t.Fatal("Unable to update test directory", err)
			}

			changed := hasChanged(prev, curr)

			testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedChanged, changed)
		})
	}
}

func listFiles(dir string) func() ([]string, error) {
	return func() ([]string, error) {
		var files []string

		err := godirwalk.Walk(dir, &godirwalk.Options{
			Unsorted: true,
			Callback: func(path string, _ *godirwalk.Dirent) error {
				files = append(files, path)
				return nil
			},
		})

		return files, err
	}
}
