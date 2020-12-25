package tag

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"testing"
)

func TestInputDigest_GenerateTagWhenFileDoesntExist(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		depListner := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
			c := []string{artifact.ImageName}
			return c, nil
		}

		tagger, _ := NewInputDigestTagger(depListner)

		artifact := &latest.Artifact{
			ImageName: "image_name",
		}

		tag, _ := tagger.GenerateTag("", artifact)

		t.CheckDeepEqual("38e0b9de817f645c4bec37c0d4a3e58baecccb040f5718dc069a72c7385a0bed", tag)
	})
}

func TestInputDigest_GenerateTagWhenFileExist(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		depListner := func(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
			c := []string{"imput_digest.go"}
			return c, nil
		}

		tagger, _ := NewInputDigestTagger(depListner)

		artifact := &latest.Artifact{
			ImageName: "image_name",
		}

		tag, _ := tagger.GenerateTag("", artifact)

		t.CheckDeepEqual("087a9e0c49f92961c5a7eab28cc304d5fb4e148dd9d3eb607c8d23698922722d", tag)
	})
}
