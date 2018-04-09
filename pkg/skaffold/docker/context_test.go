/*
Copyright 2018 Google LLC

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

package docker

import (
	"archive/tar"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func TestDockerContext(t *testing.T) {
	reader, writer := io.Pipe()
	go func() {
		err := CreateDockerTarContext(writer, "Dockerfile", "../../../testdata/docker")
		if err != nil {
			writer.CloseWithError(errors.Wrap(err, "creating docker context"))
			panic(err)
		}
		writer.Close()
	}()

	var files []string
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			panic(errors.Wrap(err, "reading tar headers"))
		}

		files = append(files, header.Name)
	}

	if util.StrSliceContains(files, "ignored.txt") {
		t.Error("File ignored.txt should have been excluded, but was not")
	}

	if util.StrSliceContains(files, "files/ignored.txt") {
		t.Error("File files/ignored.txt should have been excluded, but was not")
	}

	if !util.StrSliceContains(files, "files/included.txt") {
		t.Error("File files/included.txt should have been included, but was not")
	}

	if !util.StrSliceContains(files, "Dockerfile") {
		t.Error("File Dockerfile should have been included, but was not")
	}
}
