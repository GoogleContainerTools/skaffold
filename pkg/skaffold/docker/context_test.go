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

package docker

import (
	"archive/tar"
	"io"
	"testing"

	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerContext(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	imageFetcher := fakeImageFetcher{}
	RetrieveImage = imageFetcher.fetch
	defer func() { RetrieveImage = retrieveImage }()

	artifact := &latest.DockerArtifact{
		DockerfilePath: "Dockerfile",
		BuildArgs:      map[string]*string{},
	}

	tmpDir.Write("files/ignored.txt", "")
	tmpDir.Write("files/included.txt", "")
	tmpDir.Write(".dockerignore", "**/ignored.txt\nalsoignored.txt")
	tmpDir.Write("Dockerfile", "FROM alpine\nCOPY ./files /files")
	tmpDir.Write("ignored.txt", "")
	tmpDir.Write("alsoignored.txt", "")

	reader, writer := io.Pipe()
	go func() {
		err := CreateDockerTarContext(writer, tmpDir.Root(), artifact)
		if err != nil {
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	files := make(map[string]bool)
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		files[header.Name] = true
	}

	if files["ignored.txt"] {
		t.Error("File ignored.txt should have been excluded, but was not")
	}
	if files["alsoignored.txt"] {
		t.Error("File alsoignored.txt should have been excluded, but was not")
	}
	if files["files/ignored.txt"] {
		t.Error("File files/ignored.txt should have been excluded, but was not")
	}
	if !files["files/included.txt"] {
		t.Error("File files/included.txt should have been included, but was not")
	}
	if !files["Dockerfile"] {
		t.Error("File Dockerfile should have been included, but was not")
	}
}
