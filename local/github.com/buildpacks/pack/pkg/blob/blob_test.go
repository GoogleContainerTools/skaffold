package blob_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/blob"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBlob(t *testing.T) {
	spec.Run(t, "Buildpack", testBlob, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBlob(t *testing.T, when spec.G, it spec.S) {
	when("#Blob", func() {
		when("#Open", func() {
			var (
				blobDir  = filepath.Join("testdata", "blob")
				blobPath string
			)

			when("dir", func() {
				it.Before(func() {
					blobPath = blobDir
				})
				it("returns a tar reader", func() {
					assertBlob(t, blob.NewBlob(blobPath))
				})
			})

			when("tgz", func() {
				it.Before(func() {
					blobPath = h.CreateTGZ(t, blobDir, ".", -1)
				})

				it.After(func() {
					h.AssertNil(t, os.Remove(blobPath))
				})
				it("returns a tar reader", func() {
					assertBlob(t, blob.NewBlob(blobPath))
				})
			})

			when("tar", func() {
				it.Before(func() {
					blobPath = h.CreateTAR(t, blobDir, ".", -1)
				})

				it.After(func() {
					h.AssertNil(t, os.Remove(blobPath))
				})
				it("returns a tar reader", func() {
					assertBlob(t, blob.NewBlob(blobPath))
				})
			})
		})
	})
}
