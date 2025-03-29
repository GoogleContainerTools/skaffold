package index_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/index"
	h "github.com/buildpacks/pack/testhelpers"
)

var dockerRegistry *h.TestRegistryConfig

func TestIndexFactory(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	h.RequireDocker(t)

	dockerRegistry = h.RunRegistry(t)
	defer dockerRegistry.StopRegistry(t)

	os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerConfigDir)
	spec.Run(t, "Fetcher", testIndexFactory, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testIndexFactory(t *testing.T, when spec.G, it spec.S) {
	var (
		indexFactory  *index.IndexFactory
		imageIndex    imgutil.ImageIndex
		indexRepoName string
		err           error
		tmpDir        string
	)

	it.Before(func() {
		tmpDir, err = os.MkdirTemp("", "index-factory-test")
		h.AssertNil(t, err)
		indexFactory = index.NewIndexFactory(authn.DefaultKeychain, tmpDir)
	})

	it.After(func() {
		os.RemoveAll(tmpDir)
	})

	when("#CreateIndex", func() {
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
		})

		when("no options are provided", func() {
			it("creates an image index", func() {
				imageIndex, err = indexFactory.CreateIndex(indexRepoName)
				h.AssertNil(t, err)
				h.AssertNotNil(t, imageIndex)
			})
		})
	})

	when("#Exists", func() {
		when("index exists on disk", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				setUpLocalIndex(t, indexFactory, indexRepoName)
			})

			it("returns true", func() {
				h.AssertTrue(t, indexFactory.Exists(indexRepoName))
			})
		})

		when("index does not exist on disk", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
			})

			it("returns false", func() {
				h.AssertFalse(t, indexFactory.Exists(indexRepoName))
			})
		})
	})

	when("#LoadIndex", func() {
		when("index exists on disk", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				setUpLocalIndex(t, indexFactory, indexRepoName)
			})

			it("loads the index from disk", func() {
				imageIndex, err = indexFactory.LoadIndex(indexRepoName)
				h.AssertNil(t, err)
				h.AssertNotNil(t, imageIndex)
			})
		})

		when("index does not exist on disk", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
			})

			it("errors with a message", func() {
				_, err = indexFactory.LoadIndex(indexRepoName)
				h.AssertError(t, err, fmt.Sprintf("Image: '%s' not found", indexRepoName))
			})
		})
	})

	when("#FetchIndex", func() {
		when("index exists in a remote registry", func() {
			var remoteIndexRepoName string

			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				remoteIndexRepoName = newTestImageIndexName("fetch-remote")
				setUpRandomRemoteIndex(t, remoteIndexRepoName, 1, 1)
			})

			it("creates an index with the underlying remote index", func() {
				_, err = indexFactory.FetchIndex(indexRepoName, imgutil.FromBaseIndex(remoteIndexRepoName))
				h.AssertNil(t, err)
			})
		})

		when("index does not exist in a remote registry", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
			})

			it("errors with a message", func() {
				_, err = indexFactory.FetchIndex(indexRepoName, imgutil.FromBaseIndex(indexRepoName))
				h.AssertNotNil(t, err)
			})
		})
	})

	when("#FindIndex", func() {
		when("index exists on disk", func() {
			it.Before(func() {
				indexRepoName = h.NewRandomIndexRepoName()
				setUpLocalIndex(t, indexFactory, indexRepoName)
			})

			it("finds the index on disk", func() {
				imageIndex, err = indexFactory.FindIndex(indexRepoName)
				h.AssertNil(t, err)
				h.AssertNotNil(t, imageIndex)
			})
		})

		when("index exists in a remote registry", func() {
			it.Before(func() {
				indexRepoName = newTestImageIndexName("find-remote")
				setUpRandomRemoteIndex(t, indexRepoName, 1, 1)
			})

			it("finds the index in the remote registry", func() {
				imageIndex, err = indexFactory.FindIndex(indexRepoName)
				h.AssertNil(t, err)
				h.AssertNotNil(t, imageIndex)
			})
		})
	})
}

func setUpLocalIndex(t *testing.T, indexFactory *index.IndexFactory, indexRepoName string) {
	imageIndex, err := indexFactory.CreateIndex(indexRepoName)
	h.AssertNil(t, err)
	h.AssertNil(t, imageIndex.SaveDir())
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

	err = remote.WriteIndex(ref, randomIndex, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	h.AssertNil(t, err)

	return randomIndex
}
