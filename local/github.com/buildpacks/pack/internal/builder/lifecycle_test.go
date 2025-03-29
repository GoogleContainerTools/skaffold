package builder_test

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/blob"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestLifecycle(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testLifecycle", testLifecycle, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testLifecycle(t *testing.T, when spec.G, it spec.S) {
	when("#NewLifecycle", func() {
		when("platform api 0.3", func() {
			it("makes a lifecycle from a blob", func() {
				_, err := builder.NewLifecycle(blob.NewBlob(filepath.Join("testdata", "lifecycle", "platform-0.3")))
				h.AssertNil(t, err)
			})
		})

		when("platform api 0.4", func() {
			it("makes a lifecycle from a blob", func() {
				_, err := builder.NewLifecycle(blob.NewBlob(filepath.Join("testdata", "lifecycle", "platform-0.4")))
				h.AssertNil(t, err)
			})
		})

		when("the blob can't open", func() {
			it("throws an error", func() {
				_, err := builder.NewLifecycle(blob.NewBlob(filepath.Join("testdata", "doesn't exist")))
				h.AssertError(t, err, "open lifecycle blob")
			})
		})

		when("there is no descriptor file", func() {
			it("throws an error", func() {
				_, err := builder.NewLifecycle(&fakeEmptyBlob{})
				h.AssertError(t, err, "could not find entry path 'lifecycle.toml': not exist")
			})
		})

		when("the descriptor file isn't valid", func() {
			var tmpDir string

			it.Before(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "lifecycle")
				h.AssertNil(t, err)

				h.AssertNil(t, os.WriteFile(filepath.Join(tmpDir, "lifecycle.toml"), []byte(`
[api]
  platform "0.1"
`), 0711))
			})

			it.After(func() {
				h.AssertNil(t, os.RemoveAll(tmpDir))
			})

			it("returns an error", func() {
				_, err := builder.NewLifecycle(blob.NewBlob(tmpDir))
				h.AssertError(t, err, "decoding descriptor")
			})
		})

		when("the lifecycle has incomplete list of binaries", func() {
			var tmpDir string

			it.Before(func() {
				var err error
				tmpDir, err = os.MkdirTemp("", "")
				h.AssertNil(t, err)

				h.AssertNil(t, os.WriteFile(filepath.Join(tmpDir, "lifecycle.toml"), []byte(`
[api]
  platform = "0.2"
  buildpack = "0.3"

[lifecycle]
  version = "1.2.3"
`), os.ModePerm))

				h.AssertNil(t, os.Mkdir(filepath.Join(tmpDir, "lifecycle"), os.ModePerm))
				h.AssertNil(t, os.WriteFile(filepath.Join(tmpDir, "lifecycle", "analyzer"), []byte("content"), os.ModePerm))
				h.AssertNil(t, os.WriteFile(filepath.Join(tmpDir, "lifecycle", "detector"), []byte("content"), os.ModePerm))
				h.AssertNil(t, os.WriteFile(filepath.Join(tmpDir, "lifecycle", "builder"), []byte("content"), os.ModePerm))
			})

			it.After(func() {
				h.AssertNil(t, os.RemoveAll(tmpDir))
			})

			it("returns an error", func() {
				_, err := builder.NewLifecycle(blob.NewBlob(tmpDir))
				h.AssertError(t, err, "validating binaries")
			})
		})
	})
}

type fakeEmptyBlob struct {
}

func (f *fakeEmptyBlob) Open() (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		tw := tar.NewWriter(pw)
		defer tw.Close()
	}()
	return pr, nil
}
