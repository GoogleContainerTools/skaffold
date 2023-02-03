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

package kustomize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var NewTmpFS = newTmpFS

type TmpFSReal struct {
	root string
}

func (f TmpFSReal) WriteTo(path string, content []byte) error {
	dst, err := f.GetPath(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(dst, content, os.ModePerm)
}

func newTmpFS(rootPath string) TmpFS {
	return TmpFSReal{
		root: filepath.Clean(rootPath),
	}
}

func (f TmpFSReal) Cleanup() {
	os.RemoveAll(f.root)
}

func (f TmpFSReal) GetPath(path string) (string, error) {
	res := filepath.Join(f.root, path)
	if err := f.validate(res); err != nil {
		return "", err
	}
	return res, nil
}

func (f TmpFSReal) validate(path string) error {
	if path != f.root && !strings.HasPrefix(path, f.root+"/") {
		return fmt.Errorf("temporary file system operation is out of boundary, root: %s, trying to access %s", f.root, path)
	}
	return nil
}

type TmpFS interface {
	WriteTo(path string, content []byte) error
	Cleanup()
	GetPath(path string) (string, error)
}
