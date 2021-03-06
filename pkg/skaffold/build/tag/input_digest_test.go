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
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestInputDigest_GenerateTagWhenFileDoesntExist(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		mockDependenciesForArtifact := func(ctx context.Context, a *latest.Artifact, cfg docker.Config, r docker.ArtifactResolver) ([]string, error) {
			c := []string{"imput_digest.go"}
			return c, nil
		}
		getDependenciesForArtifact = mockDependenciesForArtifact

		tagger, _ := NewInputDigestTagger(nil, nil)

		artifact := &latest.Artifact{
			ImageName: "image_name",
		}

		tag, _ := tagger.GenerateTag("", artifact)

		t.CheckDeepEqual("38e0b9de817f645c4bec37c0d4a3e58baecccb040f5718dc069a72c7385a0bed", tag)
	})
}

func TestInputDigest_GenerateCorrectChecksumForSingleFile(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		dir := t.TempDir()
		d1 := []byte("hello\ngo\n")
		filePath := filepath.Join(dir, "temp.file")
		ioutil.WriteFile(filePath, d1, 0644)

		hash, _ := fileHasher(filePath)

		// because we are hashing content of file and it's path
		// we can't get a stable hash in testing because call t.TempDir()
		// will return a folder to a random name
		re := regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
		t.CheckTrue(re.MatchString(hash))
	})
}
