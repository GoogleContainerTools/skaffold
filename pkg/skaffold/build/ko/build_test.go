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
	"context"
	"testing"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/publish"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

// TestBuild doesn't actually build (or publish) any container images, because
// it's a unit test. Instead, it only verifies that the Build() returns the
// image identifier.
func TestBuild(t *testing.T) {
	tests := []struct {
		description             string
		pushImages              bool
		imageRef                string
		expectedImageIdentifier string
	}{
		{
			description:             "pushed image with tag",
			pushImages:              true,
			imageRef:                "registry.example.com/repo/image1:tag1",
			expectedImageIdentifier: "tag1",
		},
		{
			description:             "sideloaded image",
			pushImages:              false,
			imageRef:                "registry.example.com/repo/image2:any",
			expectedImageIdentifier: "ab737430e80b",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			importPath := "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko" // this package
			b := stubKoArtifactBuilder(test.imageRef, test.expectedImageIdentifier, test.pushImages, importPath)

			artifact := &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				ImageName: importPath,
			}
			gotImageIdentifier, err := b.Build(context.Background(), nil, artifact, test.imageRef)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedImageIdentifier, gotImageIdentifier)
		})
	}
}

func Test_getImportPath(t *testing.T) {
	tests := []struct {
		description        string
		artifact           *latestV1.Artifact
		expectedImportPath string
	}{
		{
			description: "main is ignored when image name is ko-prefixed full Go import path",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Main: "./main-should-be-ignored",
					},
				},
				ImageName: "ko://git.example.com/org/foo",
			},
			expectedImportPath: "ko://git.example.com/org/foo",
		},
		{
			description: "plain image name",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				ImageName: "any-image-name-1",
			},
			expectedImportPath: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko", // this package
		},
		{
			description: "plain image name with workspace directory",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
				ImageName: "any-image-name-2",
				Workspace: "./testdata/package-main-in-root",
			},
			expectedImportPath: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/testdata/package-main-in-root",
		},
		{
			description: "plain image name with workspace directory and main",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Main: "./baz",
					},
				},
				ImageName: "any-image-name-3",
				Workspace: "./testdata/package-main-not-in-root",
			},
			expectedImportPath: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/testdata/package-main-not-in-root/baz",
		},
		{
			description: "plain image name with workspace directory and main and source directory",
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{
						Dir:  "package-main-not-in-root",
						Main: "./baz",
					},
				},
				ImageName: "any-image-name-4",
				Workspace: "./testdata",
			},
			expectedImportPath: "ko://github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/testdata/package-main-not-in-root/baz",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			gotImportPath, err := getImportPath(test.artifact)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedImportPath, gotImportPath)
		})
	}
}

func Test_getImageIdentifier(t *testing.T) {
	tests := []struct {
		description         string
		pushImages          bool
		imageRefFromPublish name.Reference
		ref                 string
		wantImageIdentifier string
	}{
		{
			description:         "returns tag for pushed image with tag",
			pushImages:          true,
			imageRefFromPublish: name.MustParseReference("registry.example.com/repo/image:tag"),
			ref:                 "anything", // not used
			wantImageIdentifier: "tag",
		},
		{
			description:         "returns digest for pushed image with digest",
			pushImages:          true,
			imageRefFromPublish: name.MustParseReference("gcr.io/google-containers/echoserver@sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229"),
			ref:                 "any value", // not used
			wantImageIdentifier: "sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229",
		},
		{
			description:         "returns docker image ID for sideloaded image",
			pushImages:          false,
			imageRefFromPublish: nil, // not used
			ref:                 "any value",
			wantImageIdentifier: "ab737430e80b",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			b := stubKoArtifactBuilder(test.ref, test.wantImageIdentifier, test.pushImages, "")

			gotImageIdentifier, err := b.getImageIdentifier(context.Background(), test.imageRefFromPublish, test.ref)
			t.CheckNoError(err)

			t.CheckDeepEqual(test.wantImageIdentifier, gotImageIdentifier)
		})
	}
}

// stubKoArtifactBuilder returns an instance of Builder.
// Both the localDocker and the publishImages fields of the Builder are fakes.
// This means that calling Build() on the returned Builder doesn't actually
// build or publish any images.
func stubKoArtifactBuilder(ref string, imageID string, pushImages bool, importpath string) *Builder {
	api := (&testutil.FakeAPIClient{}).Add(ref, imageID)
	localDocker := fakeLocalDockerDaemon(api)
	b := NewArtifactBuilder(localDocker, pushImages, config.RunModes.Build, nil)

	// Fake implementation of the `publishImages` function.
	// Returns a map with one entry: importpath -> ref
	// importpath and ref are arguments to the function creating the stub Builder.
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
