package layer_test

import (
	"archive/tar" //nolint
	"testing"

	ilayer "github.com/buildpacks/imgutil/layer"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/layer"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestTarWriterFactory(t *testing.T) {
	spec.Run(t, "WriterFactory", testWriterFactory, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testWriterFactory(t *testing.T, when spec.G, it spec.S) {
	when("#NewWriterFactory", func() {
		it("returns an error for invalid image OS", func() {
			_, err := layer.NewWriterFactory("not-an-os")
			h.AssertError(t, err, "provided image OS 'not-an-os' must be either 'linux' or 'windows'")
		})
	})

	when("#NewWriter", func() {
		it("returns a regular tar writer for Linux", func() {
			factory, err := layer.NewWriterFactory("linux")
			h.AssertNil(t, err)

			_, ok := factory.NewWriter(nil).(*tar.Writer)
			if !ok {
				t.Fatal("returned writer was not a regular tar writer")
			}
		})

		it("returns a Windows layer writer for Windows", func() {
			factory, err := layer.NewWriterFactory("windows")
			h.AssertNil(t, err)

			_, ok := factory.NewWriter(nil).(*ilayer.WindowsWriter)
			if !ok {
				t.Fatal("returned writer was not a Windows layer writer")
			}
		})
	})
}
