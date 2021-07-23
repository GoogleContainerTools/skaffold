/*
Copyright 2021 The Skaffold Authors

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

package ko

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/publish"

	// latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestBuildKoImages(t *testing.T) {
	tests := []struct {
		description         string
		ref                 string
		imageID             string
		pushImages          bool
		importpath          string
		imageNameFromConfig string
		workspace           string
	}{
		{
			description:         "simple image name and sideload image",
			ref:                 "gcr.io/project-id/test-app1:testTag",
			imageID:             "imageID1",
			pushImages:          false,
			importpath:          "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko",
			imageNameFromConfig: "test-app1",
		},
		{
			description:         "ko import path image name and sideload image",
			ref:                 "gcr.io/project-id/example.com/myapp:myTag",
			imageID:             "imageID2",
			pushImages:          false,
			importpath:          "ko://example.com/myapp",
			imageNameFromConfig: "ko://example.com/myapp",
		},
		{
			description:         "simple image name and push image",
			ref:                 "gcr.io/project-id/test-app2:testTag",
			imageID:             "testTag",
			pushImages:          true,
			importpath:          "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko",
			imageNameFromConfig: "test-app2",
		},
		{
			description:         "ko import path image name and push image",
			ref:                 "gcr.io/project-id/example.com/myapp:myTag",
			imageID:             "myTag",
			pushImages:          true,
			importpath:          "ko://example.com/myapp",
			imageNameFromConfig: "ko://example.com/myapp",
		},
		{
			description:         "workspace is not cwd",
			ref:                 "gcr.io/project-id/example.com/test-app3:myTag",
			imageID:             "imageID3",
			pushImages:          false,
			importpath:          "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker",
			imageNameFromConfig: "test-app3",
			workspace:           "../docker",
		},
		{
			description:         "ko import path image name and workspace is not cwd",
			ref:                 "gcr.io/project-id/example.com/test-app4:myTag",
			imageID:             "imageID4",
			pushImages:          false,
			importpath:          "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker",
			imageNameFromConfig: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker",
			workspace:           "../docker",
		},
		{
			description:         "ko import path image name and workspace is not cwd and import path is subdirectory of cwd",
			ref:                 "gcr.io/project-id/example.com/test-app5:myTag",
			imageID:             "imageID5",
			pushImages:          false,
			importpath:          "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker",
			imageNameFromConfig: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/docker",
			workspace:           "..",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			b := stubKoArtifactBuilder(test.ref, test.imageID, test.pushImages, test.importpath)

			artifact := &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				Dependencies: []*latestV1.ArtifactDependency{},
				ImageName:    test.imageNameFromConfig,
				Workspace:    test.workspace,
			}

			var outBuffer bytes.Buffer
			gotImageID, err := b.Build(context.TODO(), &outBuffer, artifact, test.ref)
			t.CheckNoError(err)
			if gotImageID != test.imageID {
				t.Errorf("got image ID %s, wanted %s", gotImageID, test.imageID)
			}
			imageNameOut := strings.TrimSuffix(outBuffer.String(), "\n")
			if imageNameOut != test.ref {
				t.Errorf("image name output was %q, wanted %q", imageNameOut, test.ref)
			}
		})
	}
}

func stubKoArtifactBuilder(ref string, imageID string, pushImages bool, importpath string) *Builder {
	api := (&testutil.FakeAPIClient{}).Add(ref, imageID)
	localDocker := fakeLocalDockerDaemon(api)
	b := NewArtifactBuilder(localDocker, pushImages)

	// Fake implementation of the `publishImages` function.
	b.publishImages = func(_ context.Context, _ []string, _ publish.Interface, _ build.Interface) (map[string]name.Reference, error) {
		imageRef, err := name.ParseReference(ref)
		if err != nil {
			return nil, err
		}
		return map[string]name.Reference{
			importpath: imageRef,
		}, nil
	}
	return b
}

func fakeLocalDockerDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, false, nil)
}
