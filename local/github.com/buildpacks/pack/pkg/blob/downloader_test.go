package blob_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/blob"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDownloader(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Downloader", testDownloader, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testDownloader(t *testing.T, when spec.G, it spec.S) {
	when("#Download", func() {
		var (
			cacheDir string
			err      error
			subject  blob.Downloader
		)

		it.Before(func() {
			cacheDir, err = os.MkdirTemp("", "cache")
			h.AssertNil(t, err)
			subject = blob.NewDownloader(&logger{io.Discard}, cacheDir)
		})

		it.After(func() {
			h.AssertNil(t, os.RemoveAll(cacheDir))
		})

		when("is path", func() {
			var (
				relPath string
			)

			it.Before(func() {
				relPath = filepath.Join("testdata", "blob")
			})

			when("is absolute", func() {
				it("return the absolute path", func() {
					absPath, err := filepath.Abs(relPath)
					h.AssertNil(t, err)

					b, err := subject.Download(context.TODO(), absPath)
					h.AssertNil(t, err)
					assertBlob(t, b)
				})
			})

			when("is relative", func() {
				it("resolves the absolute path", func() {
					b, err := subject.Download(context.TODO(), relPath)
					h.AssertNil(t, err)
					assertBlob(t, b)
				})
			})

			when("path is a file:// uri", func() {
				it("resolves the absolute path", func() {
					absPath, err := filepath.Abs(relPath)
					h.AssertNil(t, err)

					uri, err := paths.FilePathToURI(absPath, "")
					h.AssertNil(t, err)

					b, err := subject.Download(context.TODO(), uri)
					h.AssertNil(t, err)
					assertBlob(t, b)
				})
			})
		})

		when("is uri", func() {
			var (
				server *ghttp.Server
				uri    string
				tgz    string
			)

			it.Before(func() {
				server = ghttp.NewServer()
				uri = server.URL() + "/downloader/somefile.tgz"

				tgz = h.CreateTGZ(t, filepath.Join("testdata", "blob"), "./", 0777)
			})

			it.After(func() {
				os.Remove(tgz)
				server.Close()
			})

			when("uri is valid", func() {
				it.Before(func() {
					server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("ETag", "A")
						http.ServeFile(w, r, tgz)
					})

					server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(304)
					})
				})

				it("downloads from a 'http(s)://' URI", func() {
					b, err := subject.Download(context.TODO(), uri)
					h.AssertNil(t, err)
					assertBlob(t, b)
				})

				it("uses cache from a 'http(s)://' URI tgz", func() {
					b, err := subject.Download(context.TODO(), uri)
					h.AssertNil(t, err)
					assertBlob(t, b)

					b, err = subject.Download(context.TODO(), uri)
					h.AssertNil(t, err)
					assertBlob(t, b)
				})
			})

			when("uri is invalid", func() {
				when("uri file is not found", func() {
					it.Before(func() {
						server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(404)
						})

						server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
							w.WriteHeader(404)
						})
					})

					it("should return error", func() {
						_, err := subject.Download(context.TODO(), uri)
						h.AssertError(t, err, "could not download")
						h.AssertError(t, err, "http status '404'")
					})
				})

				when("uri is unsupported", func() {
					it("should return error", func() {
						_, err := subject.Download(context.TODO(), "not-supported://file.tgz")
						h.AssertError(t, err, "unsupported protocol 'not-supported'")
					})
				})
			})
		})
	})
}

func assertBlob(t *testing.T, b blob.Blob) {
	t.Helper()
	r, err := b.Open()
	h.AssertNil(t, err)
	defer r.Close()

	_, bytes, err := archive.ReadTarEntry(r, "file.txt")
	h.AssertNil(t, err)

	h.AssertEq(t, string(bytes), "contents")
}

type logger struct {
	writer io.Writer
}

func (l *logger) Debugf(format string, v ...interface{}) {
	fmt.Fprintln(l.writer, format, v)
}

func (l *logger) Infof(format string, v ...interface{}) {
	fmt.Fprintln(l.writer, format, v)
}

func (l *logger) Writer() io.Writer {
	return l.writer
}
