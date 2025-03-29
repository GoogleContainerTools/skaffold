package remote_test

import (
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	remote2 "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/remote"
	h "github.com/buildpacks/imgutil/testhelpers"
)

func TestRemoteNewIndex(t *testing.T) {
	dockerConfigDir, err := os.MkdirTemp("", "test.docker.config.remote.index.dir")
	h.AssertNil(t, err)
	defer os.RemoveAll(dockerConfigDir)

	dockerRegistry = h.NewDockerRegistry(h.WithAuth(dockerConfigDir))
	dockerRegistry.Start(t)
	defer dockerRegistry.Stop(t)
	os.Setenv("DOCKER_CONFIG", dockerConfigDir)
	defer os.Unsetenv("DOCKER_CONFIG")

	spec.Run(t, "RemoteNewIndex", testNewIndex, spec.Parallel(), spec.Report(report.Terminal{}))
}

const numberOfManifests = 2

func testNewIndex(t *testing.T, when spec.G, it spec.S) {
	var (
		idx                 imgutil.ImageIndex
		manifests           []v1.Hash
		remoteIndexRepoName string
		xdgPath             string
		err                 error
	)

	it.Before(func() {
		// creates the directory to save all the OCI images on disk
		remoteIndexRepoName = newTestImageIndexName("random")
		randomIndex := setUpRandomRemoteIndex(t, remoteIndexRepoName, 1, numberOfManifests)
		manifests = h.DigestsFromImageIndex(t, randomIndex)
	})

	it.After(func() {
		err = os.RemoveAll(xdgPath)
		h.AssertNil(t, err)
	})

	when("#NewIndex", func() {
		it("should have expected indexOptions", func() {
			idx, err = remote.NewIndex(
				"some-index",
				imgutil.WithInsecure(),
				imgutil.WithKeychain(authn.DefaultKeychain),
				imgutil.WithXDGRuntimePath(xdgPath),
			)
			h.AssertNil(t, err)

			imgIx, ok := idx.(*imgutil.CNBIndex)
			h.AssertEq(t, ok, true)
			h.AssertEq(t, imgIx.XdgPath, xdgPath)
			h.AssertEq(t, imgIx.RepoName, "some-index")
		})

		it("should return an error when index with the given repoName doesn't exists", func() {
			_, err = remote.NewIndex(
				"my-index",
				imgutil.WithKeychain(authn.DefaultKeychain),
				imgutil.FromBaseIndex("some-none-existing-index"),
			)
			h.AssertNotEq(t, err, nil)
		})

		it("should return ImageIndex with expected output", func() {
			idx, err = remote.NewIndex(
				"my-index",
				imgutil.WithKeychain(authn.DefaultKeychain),
				imgutil.FromBaseIndex(remoteIndexRepoName),
			)
			h.AssertNil(t, err)

			imgIx, ok := idx.(*imgutil.CNBIndex)
			h.AssertEq(t, ok, true)

			mfest, err := imgIx.IndexManifest()
			h.AssertNil(t, err)
			h.AssertNotNil(t, mfest)
			h.AssertEq(t, len(mfest.Manifests), numberOfManifests)
		})

		it("should able to call #ImageIndex", func() {
			idx, err = remote.NewIndex(
				"my-index",
				imgutil.WithKeychain(authn.DefaultKeychain),
				imgutil.FromBaseIndex(remoteIndexRepoName),
			)
			h.AssertNil(t, err)

			imgIx, ok := idx.(*imgutil.CNBIndex)
			h.AssertEq(t, ok, true)

			// some none existing hash
			hash1, err := v1.NewHash(
				"sha256:b9d056b83bb6446fee29e89a7fcf10203c562c1f59586a6e2f39c903597bda34",
			)
			h.AssertNil(t, err)

			_, err = imgIx.ImageIndex.ImageIndex(hash1)
			// err is "no child with digest"
			h.AssertNotEq(t, err.Error(), "empty index")
		})

		it("should able to call #Image", func() {
			idx, err = remote.NewIndex(
				"my-index",
				imgutil.WithKeychain(authn.DefaultKeychain),
				imgutil.FromBaseIndex(remoteIndexRepoName),
			)
			h.AssertNil(t, err)

			imgIdx, ok := idx.(*imgutil.CNBIndex)
			h.AssertEq(t, ok, true)

			// select one valid digest from the index
			_, err = imgIdx.Image(manifests[0])
			h.AssertNil(t, err)
		})
	})
}

func newTestImageIndexName(name string) string {
	return dockerRegistry.RepoName(name + "-" + h.RandString(10))
}

// setUpRandomRemoteIndex creates a random image index with the provided (count) number of manifest
// each manifest will have the provided number of layers
func setUpRandomRemoteIndex(t *testing.T, repoName string, layers, count int64) v1.ImageIndex {
	ref, err := name.ParseReference(repoName, name.WeakValidation)
	h.AssertNil(t, err)

	randomIndex, err := random.Index(1024, layers, count)
	h.AssertNil(t, err)

	err = remote2.WriteIndex(ref, randomIndex, remote2.WithAuthFromKeychain(authn.DefaultKeychain))
	h.AssertNil(t, err)

	return randomIndex
}
