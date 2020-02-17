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
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/statik"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type fakeFileSystem struct {
	Files map[string][]byte
}

type fakeFile struct {
	http.File
	content io.Reader
}

func (f *fakeFileSystem) Open(name string) (http.File, error) {
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

func (f *fakeFile) Close() error {
	return nil
}

func TestPrint(t *testing.T) {
	fs := &fakeFileSystem{
		Files: map[string][]byte{
			"/schemas/v1.json": []byte("{SCHEMA}"),
		},
	}

	testutil.Run(t, "found", func(t *testutil.T) {
		t.Override(&statik.FS, func() (http.FileSystem, error) { return fs, nil })

		var out bytes.Buffer
		err := Print(&out, "skaffold/v1")

		t.CheckNoError(err)
		t.CheckDeepEqual("{SCHEMA}", out.String())
	})

	testutil.Run(t, "not found", func(t *testutil.T) {
		t.Override(&statik.FS, func() (http.FileSystem, error) { return fs, nil })

		var out bytes.Buffer
		err := Print(&out, "skaffold/v0")

		t.CheckErrorContains("schema \"skaffold/v0\" not found", err)
		t.CheckEmpty(out.String())
	})
}
