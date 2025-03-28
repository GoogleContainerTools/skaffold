package cache_test

import (
	"context"
	"testing"

	"github.com/buildpacks/pack/pkg/cache"

	"github.com/buildpacks/imgutil/local"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestImageCache(t *testing.T) {
	h.RequireDocker(t)
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "ImageCache", testImageCache, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testImageCache(t *testing.T, when spec.G, it spec.S) {
	when("#NewImageCache", func() {
		var dockerClient client.CommonAPIClient

		it.Before(func() {
			var err error
			dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
			h.AssertNil(t, err)
		})

		when("#Name", func() {
			it("should return the image reference used in intialization", func() {
				refName := "gcr.io/my/repo:tag"
				ref, err := name.ParseReference(refName, name.WeakValidation)
				h.AssertNil(t, err)
				subject := cache.NewImageCache(ref, dockerClient)
				actual := subject.Name()
				if actual != refName {
					t.Fatalf("Incorrect cache name expected %s, got %s", refName, actual)
				}
			})
		})

		it("resolves implied tag", func() {
			ref, err := name.ParseReference("my/repo:latest", name.WeakValidation)
			h.AssertNil(t, err)
			subject := cache.NewImageCache(ref, dockerClient)

			ref, err = name.ParseReference("my/repo", name.WeakValidation)
			h.AssertNil(t, err)
			expected := cache.NewImageCache(ref, dockerClient)

			h.AssertEq(t, subject.Name(), expected.Name())
		})

		it("resolves implied registry", func() {
			ref, err := name.ParseReference("index.docker.io/my/repo", name.WeakValidation)
			h.AssertNil(t, err)
			subject := cache.NewImageCache(ref, dockerClient)
			ref, err = name.ParseReference("my/repo", name.WeakValidation)
			h.AssertNil(t, err)
			expected := cache.NewImageCache(ref, dockerClient)
			if subject.Name() != expected.Name() {
				t.Fatalf("The same repo name should result in the same image")
			}
		})
	})

	when("#Type", func() {
		var (
			dockerClient client.CommonAPIClient
		)

		it.Before(func() {
			var err error
			dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
			h.AssertNil(t, err)
		})

		it("returns the cache type", func() {
			ref, err := name.ParseReference("my/repo", name.WeakValidation)
			h.AssertNil(t, err)
			subject := cache.NewImageCache(ref, dockerClient)
			expected := cache.Image
			h.AssertEq(t, subject.Type(), expected)
		})
	})

	when("#Clear", func() {
		var (
			imageName    string
			dockerClient client.CommonAPIClient
			subject      *cache.ImageCache
			ctx          context.Context
		)

		it.Before(func() {
			var err error
			dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
			h.AssertNil(t, err)
			ctx = context.TODO()

			ref, err := name.ParseReference(h.RandString(10), name.WeakValidation)
			h.AssertNil(t, err)
			subject = cache.NewImageCache(ref, dockerClient)
			h.AssertNil(t, err)
			imageName = subject.Name()
		})

		when("there is a cache image", func() {
			it.Before(func() {
				img, err := local.NewImage(imageName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())
			})

			it("removes the image", func() {
				err := subject.Clear(ctx)
				h.AssertNil(t, err)
				images, err := dockerClient.ImageList(context.TODO(), image.ListOptions{
					Filters: filters.NewArgs(filters.KeyValuePair{
						Key:   "reference",
						Value: imageName,
					}),
				})
				h.AssertNil(t, err)
				h.AssertEq(t, len(images), 0)
			})
		})

		when("there is no cache image", func() {
			it("does not fail", func() {
				err := subject.Clear(ctx)
				h.AssertNil(t, err)
			})
		})
	})
}
