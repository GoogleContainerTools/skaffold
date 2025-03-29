package archive_test

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/pack/pkg/archive"

	h "github.com/buildpacks/pack/testhelpers"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestTarBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "TarBuilder", testTarBuilder, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testTarBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		tmpDir     string
		tarBuilder archive.TarBuilder
	)

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "tar-builder-test")
		h.AssertNil(t, err)
		tarBuilder = archive.TarBuilder{}
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tmpDir))
	})

	when("#AddFile", func() {
		it("adds file", func() {
			tarBuilder.AddFile("file1", 0777, archive.NormalizedDateTime, []byte("file-1 content"))
			reader := tarBuilder.Reader(archive.DefaultTarWriterFactory())
			tr := tar.NewReader(reader)

			verify := h.NewTarVerifier(t, tr, 0, 0)
			verify.NextFile("file1", "file-1 content", int64(os.ModePerm))
			verify.NoMoreFilesExist()
		})
	})

	when("#AddDir", func() {
		it("adds dir", func() {
			tarBuilder.AddDir("path/of/dir", 0777, archive.NormalizedDateTime)
			reader := tarBuilder.Reader(archive.DefaultTarWriterFactory())
			tr := tar.NewReader(reader)

			verify := h.NewTarVerifier(t, tr, 0, 0)
			verify.NextDirectory("path/of/dir", int64(os.ModePerm))
			verify.NoMoreFilesExist()
		})
	})

	when("#WriteToPath", func() {
		it("writes to path", func() {
			path := filepath.Join(tmpDir, "some.txt")
			h.AssertNil(t, tarBuilder.WriteToPath(path, archive.DefaultTarWriterFactory()))
		})

		it("fails if dir doesn't exist", func() {
			path := "dir/some.txt"
			h.AssertError(t, tarBuilder.WriteToPath(path, archive.DefaultTarWriterFactory()), "create file for tar")
		})
	})
}
