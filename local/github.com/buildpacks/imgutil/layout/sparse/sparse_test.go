package sparse_test

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
	"github.com/buildpacks/imgutil/layout/sparse"
	h "github.com/buildpacks/imgutil/testhelpers"
)

func TestImage(t *testing.T) {
	spec.Run(t, "LayoutSparseImage", testImage, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testImage(t *testing.T, when spec.G, it spec.S) {
	var (
		testImage v1.Image
		tmpDir    string
		imagePath string
		err       error
	)

	it.Before(func() {
		testImage = h.RemoteRunnableBaseImage(t)

		// creates the directory to save all the OCI images on disk
		tmpDir, err = os.MkdirTemp("", "layout-sparse")
		h.AssertNil(t, err)
	})

	it.After(func() {
		// removes all images created
		os.RemoveAll(tmpDir)
	})

	when("#Save", func() {
		it.Before(func() {
			imagePath = filepath.Join(tmpDir, "sparse-layout-image")
		})

		it.After(func() {
			// removes all images created
			os.RemoveAll(imagePath)
		})

		when("name(s) provided", func() {
			it("creates an image and save it also in the additional path provided", func() {
				image, err := sparse.NewImage(imagePath, testImage)
				h.AssertNil(t, err)
				anotherPath := filepath.Join(tmpDir, "another-sparse-layout-image")

				// save on disk in OCI
				err = image.Save(anotherPath)
				h.AssertNil(t, err)

				//  expected blobs: manifest, config, layer
				h.AssertBlobsLen(t, imagePath, 2)
				index := h.ReadIndexManifest(t, imagePath)
				h.AssertEq(t, len(index.Manifests), 1)

				// assert image saved on additional path
				h.AssertBlobsLen(t, anotherPath, 2)
				index = h.ReadIndexManifest(t, anotherPath)
				h.AssertEq(t, len(index.Manifests), 1)
			})
		})

		when("no additional names are provided", func() {
			it("creates an image without layers", func() {
				image, err := sparse.NewImage(imagePath, testImage)
				h.AssertNil(t, err)

				// save
				err = image.Save()
				h.AssertNil(t, err)

				// expected blobs: manifest, config
				h.AssertBlobsLen(t, imagePath, 2)
			})
		})

		when("#AnnotateRefName", func() {
			it("creates an image and save it with `org.opencontainers.image.ref.name` annotation", func() {
				image, err := sparse.NewImage(imagePath, testImage)
				h.AssertNil(t, err)

				// adds org.opencontainers.image.ref.name annotation
				image.AnnotateRefName("my-tag")

				// save
				err = image.Save()
				h.AssertNil(t, err)

				// expected blobs: manifest, config
				h.AssertBlobsLen(t, imagePath, 2)

				// assert org.opencontainers.image.ref.name annotation
				index := h.ReadIndexManifest(t, imagePath)
				h.AssertEq(t, len(index.Manifests), 1)
				h.AssertEq(t, 1, len(index.Manifests[0].Annotations))
				h.AssertEqAnnotation(t, index.Manifests[0], layout.ImageRefNameKey, "my-tag")
			})
		})

		when("#MediaType", func() {
			it("returns the base image media type when there are no requested media type changes", func() {
				image, err := sparse.NewImage(imagePath, testImage)
				h.AssertNil(t, err)

				err = image.Save()
				h.AssertNil(t, err)

				expectedMediaType, err := testImage.MediaType()
				h.AssertNil(t, err)

				actualMediaType, err := image.MediaType()
				h.AssertNil(t, err)
				h.AssertEq(t, actualMediaType, expectedMediaType)
			})

			it("mutates the media type to the specified media type", func() {
				image, err := sparse.NewImage(imagePath, testImage, layout.WithMediaTypes(imgutil.OCITypes))
				h.AssertNil(t, err)

				err = image.Save()
				h.AssertNil(t, err)

				h.AssertOCIMediaTypes(t, image)
			})
		})

		when("#Digest", func() {
			it("returns the original image digest when there are no modifications", func() {
				image, err := sparse.NewImage(imagePath, testImage)
				h.AssertNil(t, err)
				h.AssertNil(t, image.Save())

				expectedDigest, err := testImage.Digest()
				h.AssertNil(t, err)

				actualDigest, err := image.Digest()
				h.AssertNil(t, err)
				h.AssertEq(t, actualDigest, expectedDigest)
			})
		})
	})
}
