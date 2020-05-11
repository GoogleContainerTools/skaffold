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

package docker

import (
	"archive/tar"
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDockerContext(t *testing.T) {
	for _, dir := range []string{".", "sub"} {
		testutil.Run(t, dir, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&RetrieveImage, imageFetcher.fetch)
			t.NewTempDir().
				Write(dir+"/.dockerignore", "**/ignored.txt\nalsoignored.txt").
				Write(dir+"/Dockerfile", "FROM busybox\nCOPY ./files /files").
				Touch(dir + "/files/ignored.txt").
				Touch(dir + "/files/included.txt").
				Touch(dir + "/ignored.txt").
				Touch(dir + "/alsoignored.txt").
				Chdir()

			artifact := &latest.DockerArtifact{
				DockerfilePath: "Dockerfile",
			}

			reader, writer := io.Pipe()
			go func() {
				err := CreateDockerTarContext(context.Background(), writer, dir, artifact, nil)
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
				t.CheckNoError(err)

				files[header.Name] = true
			}

			t.CheckFalse(files["ignored.txt"])
			t.CheckFalse(files["alsoignored.txt"])
			t.CheckFalse(files["files/ignored.txt"])
			t.CheckTrue(files["files/included.txt"])
			t.CheckTrue(files["Dockerfile"])
		})
	}
}
