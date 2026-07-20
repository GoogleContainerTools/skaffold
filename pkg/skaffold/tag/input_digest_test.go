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

package tag

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestInputDigest(t *testing.T) {
	fileContents1, fileContents2 := []byte("hello\ngo\n"), []byte("bye\ngo\n")

	testutil.Run(t, "SameDigestForRelAndAbsPath", func(t *testutil.T) {
		dir := t.TempDir()
		cwdBackup, err := os.Getwd()
		t.RequireNoError(err)
		t.RequireNoError(os.Chdir(dir))
		defer func() { t.RequireNoError(os.Chdir(cwdBackup)) }()

		file := "temp.file"
		t.RequireNoError(os.WriteFile(file, fileContents1, 0644))

		relPathHash, err := fileHasher(file, ".")
		t.CheckErrorAndDeepEqual(false, err, "f68f0e22ae9dc02857924d257668431c", relPathHash)
		absPathHash, err := fileHasher(filepath.Join(dir, file), dir)
		t.CheckErrorAndDeepEqual(false, err, relPathHash, absPathHash)
	})

	testutil.Run(t, "SameDigestForTwoDifferentAbsPaths", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp.file"), filepath.Join(dir2, "temp.file")
		t.RequireNoError(os.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(os.WriteFile(file2, fileContents1, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckErrorAndDeepEqual(false, err, "f68f0e22ae9dc02857924d257668431c", hash1)
		hash2, err := fileHasher(file2, dir2)
		t.CheckErrorAndDeepEqual(false, err, hash1, hash2)
	})

	testutil.Run(t, "DifferentDigestForDifferentFilenames", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp1.file"), filepath.Join(dir2, "temp2.file")
		t.RequireNoError(os.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(os.WriteFile(file2, fileContents1, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckNoError(err)
		hash2, err := fileHasher(file2, dir2)
		t.CheckNoError(err)
		t.CheckFalse(hash1 == hash2)
	})

	testutil.Run(t, "DifferentDigestForDifferentContent", func(t *testutil.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		file1, file2 := filepath.Join(dir1, "temp.file"), filepath.Join(dir2, "temp.file")
		t.RequireNoError(os.WriteFile(file1, fileContents1, 0644))
		t.RequireNoError(os.WriteFile(file2, fileContents2, 0644))

		hash1, err := fileHasher(file1, dir1)
		t.CheckNoError(err)
		hash2, err := fileHasher(file2, dir2)
		t.CheckNoError(err)
		t.CheckFalse(hash1 == hash2)
	})
}
func TestGenerateTag(t *testing.T) {
	testutil.Run(t, "CompareTagWithAndWithoutDockerfile", func(t *testutil.T) {
		runCtx := &runcontext.RunContext{}
		dockerfile1Path := filepath.Join(t.TempDir(), "Dockerfile1")
		dockerfile2Path := filepath.Join(t.TempDir(), "Dockerfile2")

		f, err := os.Create(dockerfile1Path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		_, err = io.WriteString(f, `
FROM busybox

CMD [ "ps", "faux" ]
`)
		if err != nil {
			panic(err)
		}

		f, err = os.Create(dockerfile2Path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		_, err = io.WriteString(f, `
FROM busybox

CMD [ "true" ]
`)
		if err != nil {
			panic(err)
		}

		digestExample, _ := NewInputDigestTagger(runCtx, graph.ToArtifactGraph(runCtx.Artifacts()))
		tag1, err := digestExample.GenerateTag(context.Background(), latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: dockerfile1Path,
				},
			},
		})
		if err != nil {
			t.Fatalf("Generate first tag failed: %v", err)
		}

		digestExample, _ = NewInputDigestTagger(runCtx, graph.ToArtifactGraph(runCtx.Artifacts()))
		tag2, err := digestExample.GenerateTag(context.Background(), latest.Artifact{
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: dockerfile2Path,
				},
			},
		})
		if err != nil {
			t.Fatalf("Generate second tag failed: %v", err)
		}

		if diff := cmp.Diff(tag1, tag2); diff == "" {
			t.Error("Tag does not differ between first and second Dockerfile")
		}
	})
}
