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

package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/karrick/godirwalk"
)

// TempFile creates a temporary file with a given content. Returns the file name
// and a teardown function that should be called to properly delete the file.
func TempFile(t *testing.T, prefix string, content []byte) (name string, tearDown func()) {
	file, err := ioutil.TempFile("", prefix)
	if err != nil {
		t.Error(err)
	}

	if err = ioutil.WriteFile(file.Name(), content, 0644); err != nil {
		t.Error(err)
	}

	return file.Name(), func() {
		syscall.Unlink(file.Name())
	}
}

// TempDir offers actions on a temp directory.
type TempDir struct {
	t               *testing.T
	root            string
	resetCurrentDir func()
}

// NewTempDir creates a temporary directory and a teardown function
// that should be called to properly delete the directory content.
func NewTempDir(t *testing.T) (tmp *TempDir, tearDown func()) {
	root, err := ioutil.TempDir("", "skaffold")
	if err != nil {
		t.Error(err)
	}

	tmpDir := &TempDir{
		t:               t,
		root:            root,
		resetCurrentDir: func() {},
	}

	return tmpDir, tmpDir.tearDown
}

func (h *TempDir) tearDown() {
	h.resetCurrentDir()
	os.RemoveAll(h.Root())
}

// Root returns the temp directory.
func (h *TempDir) Root() string {
	return h.root
}

// Remove deletes a file from the temp directory.
func (h *TempDir) Remove(file string) *TempDir {
	return h.failIfErr(os.Remove(h.Path(file)))
}

// Chtimes changes the times for a file in the temp directory.
func (h *TempDir) Chtimes(file string, t time.Time) *TempDir {
	return h.failIfErr(os.Chtimes(h.Path(file), t, t))
}

// Mkdir makes a sub-directory in the temp directory.
func (h *TempDir) Mkdir(dir string) *TempDir {
	return h.failIfErr(os.MkdirAll(h.Path(dir), os.ModePerm))
}

// Write write content to a file in the temp directory.
func (h *TempDir) Write(file, content string) *TempDir {
	h.failIfErr(os.MkdirAll(filepath.Dir(h.Path(file)), os.ModePerm))
	return h.failIfErr(ioutil.WriteFile(h.Path(file), []byte(content), os.ModePerm))
}

// WriteFiles write a list of files (path->content) in the temp directory.
func (h *TempDir) WriteFiles(files map[string]string) *TempDir {
	for path, content := range files {
		h.Write(path, content)
	}
	return h
}

// Touch creates a list of empty files in the temp directory.
func (h *TempDir) Touch(files ...string) *TempDir {
	for _, file := range files {
		h.Write(file, "")
	}
	return h
}

// Symlink creates a symlink.
func (h *TempDir) Symlink(dst, src string) *TempDir {
	h.failIfErr(os.MkdirAll(filepath.Dir(h.Path(src)), os.ModePerm))
	return h.failIfErr(os.Symlink(h.Path(dst), h.Path(src)))
}

// Rename renames a file from oldname to newname
func (h *TempDir) Rename(oldName, newName string) *TempDir {
	return h.failIfErr(os.Rename(h.Path(oldName), h.Path(newName)))
}

// List lists all the files in the temp directory.
func (h *TempDir) List() ([]string, error) {
	var files []string

	err := godirwalk.Walk(h.root, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, _ *godirwalk.Dirent) error {
			files = append(files, path)
			return nil
		},
	})

	return files, err
}

// Path returns the path to a file in the temp directory.
func (h *TempDir) Path(file string) string {
	elem := []string{h.root}
	elem = append(elem, strings.Split(file, "/")...)
	return filepath.Join(elem...)
}

func (h *TempDir) failIfErr(err error) *TempDir {
	if err != nil {
		h.t.Fatal(err)
	}
	return h
}

// Paths returns the paths to a list of files in the temp directory.
func (h *TempDir) Paths(files ...string) []string {
	var paths []string
	for _, file := range files {
		paths = append(paths, h.Path(file))
	}
	return paths
}

// Chdir changes current directory to this temp directory.
func (h *TempDir) Chdir() *TempDir {
	pwd, err := os.Getwd()
	if err != nil {
		h.t.Fatal("unable to get current directory")
	}

	err = os.Chdir(h.Root())
	if err != nil {
		h.t.Fatal("unable to change current directory")
	}

	h.resetCurrentDir = func() {
		if err := os.Chdir(pwd); err != nil {
			h.t.Fatal("unable to reset current directory")
		}
	}

	return h
}
