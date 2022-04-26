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

package testutil

import (
	"bytes"
	"io"
	"io/fs"
	"os"
)

type FakeFileSystem struct {
	Files map[string][]byte
}

type fakeFile struct {
	fs.File
	content io.Reader
}

func (f FakeFileSystem) Open(name string) (fs.File, error) {
	content, found := f.Files[name]
	if !found {
		return nil, os.ErrNotExist
	}

	return &fakeFile{
		content: bytes.NewBuffer(content),
	}, nil
}

func (f *fakeFile) Read(p []byte) (n int, err error) {
	return f.content.Read(p)
}

func (f FakeFileSystem) ReadFile(name string) ([]byte, error) {
	content, found := f.Files[name]
	if !found {
		return nil, os.ErrNotExist
	}

	return content, nil
}

func (f *fakeFile) Close() error {
	return nil
}
