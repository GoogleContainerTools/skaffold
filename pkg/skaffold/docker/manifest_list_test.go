package docker

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func (fi fakeImage) Descriptor() (*v1.Descriptor, error) {
	return &v1.Descriptor{}, nil
}

func TestCreateManifestList(t *testing.T) {
	ctx := context.Background()
	targetTag := "gcr.io/skaffold/example:latest"
	images := []SinglePlatformImage{
		{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}, Image: "gcr.io/skaffold/example:b1234_linux_amd64"},
		{Platform: &v1.Platform{OS: "linux", Architecture: "arm64"}, Image: "gcr.io/skaffold/example:b1234_linux_arm64"},
	}

	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&remoteImage, func(ref name.Reference, options ...remote.Option) (v1.Image, error) {
			return &fakeImage{
				Reference: ref,
			}, nil
		})

		t.Override(&mutateAppendManifest, func(base v1.ImageIndex, adds ...mutate.IndexAddendum) v1.ImageIndex {
			for i, add := range adds {
				img := add.Add.(*fakeImage).Reference.Name()
				t.CheckDeepEqual(images[i].Image, img)
			}

			return &fakeImageIndex{}
		})

		t.Override(&remoteWriteIndex, func(ref name.Reference, ii v1.ImageIndex, options ...remote.Option) (rerr error) {
			return nil
		})

		manifestTag, err := CreateManifestList(ctx, images, targetTag)

		if err != nil {
			t.Fatalf("Error generating manifest list with target tag %s\n", targetTag)
		}

		if !strings.HasPrefix(manifestTag, targetTag) {
			t.Fatalf("Error in tag %s, %s not found\n", manifestTag, targetTag)
		}
	})
}
