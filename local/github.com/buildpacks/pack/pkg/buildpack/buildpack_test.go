package buildpack_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "buildpack", testBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildpack(t *testing.T, when spec.G, it spec.S) {
	var writeBlobToFile = func(bp buildpack.BuildModule) string {
		t.Helper()

		bpReader, err := bp.Open()
		h.AssertNil(t, err)

		tmpDir, err := os.MkdirTemp("", "")
		h.AssertNil(t, err)

		p := filepath.Join(tmpDir, "bp.tar")
		bpWriter, err := os.Create(p)
		h.AssertNil(t, err)

		_, err = io.Copy(bpWriter, bpReader)
		h.AssertNil(t, err)

		err = bpReader.Close()
		h.AssertNil(t, err)

		return p
	}

	when("#BuildpackFromRootBlob", func() {
		it("parses the descriptor file", func() {
			bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
				openFn: func() io.ReadCloser {
					tarBuilder := archive.TarBuilder{}
					tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"
homepage = "http://geocities.com/cool-bp"

[[stacks]]
id = "some.stack.id"
`))
					return tarBuilder.Reader(archive.DefaultTarWriterFactory())
				},
			}, archive.DefaultTarWriterFactory(), nil)
			h.AssertNil(t, err)

			h.AssertEq(t, bp.Descriptor().API().String(), "0.3")
			h.AssertEq(t, bp.Descriptor().Info().ID, "bp.one")
			h.AssertEq(t, bp.Descriptor().Info().Version, "1.2.3")
			h.AssertEq(t, bp.Descriptor().Info().Homepage, "http://geocities.com/cool-bp")
			h.AssertEq(t, bp.Descriptor().Stacks()[0].ID, "some.stack.id")
		})

		it("translates blob to distribution format", func() {
			bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
				openFn: func() io.ReadCloser {
					tarBuilder := archive.TarBuilder{}
					tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"

[[stacks]]
id = "some.stack.id"
`))

					tarBuilder.AddDir("bin", 0700, time.Now())
					tarBuilder.AddFile("bin/detect", 0700, time.Now(), []byte("detect-contents"))
					tarBuilder.AddFile("bin/build", 0700, time.Now(), []byte("build-contents"))
					return tarBuilder.Reader(archive.DefaultTarWriterFactory())
				},
			}, archive.DefaultTarWriterFactory(), nil)
			h.AssertNil(t, err)

			h.AssertNil(t, bp.Descriptor().EnsureTargetSupport(dist.DefaultTargetOSLinux, dist.DefaultTargetArch, "", ""))

			tarPath := writeBlobToFile(bp)
			defer os.Remove(tarPath)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one",
				h.IsDirectory(),
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
			)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3",
				h.IsDirectory(),
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
			)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3/bin",
				h.IsDirectory(),
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
			)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3/bin/detect",
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
				h.ContentEquals("detect-contents"),
			)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3/bin/build",
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
				h.ContentEquals("build-contents"),
			)
		})

		it("translates blob to windows bat distribution format", func() {
			bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
				openFn: func() io.ReadCloser {
					tarBuilder := archive.TarBuilder{}
					tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.9"

[buildpack]
id = "bp.one"
version = "1.2.3"
`))

					tarBuilder.AddDir("bin", 0700, time.Now())
					tarBuilder.AddFile("bin/detect", 0700, time.Now(), []byte("detect-contents"))
					tarBuilder.AddFile("bin/build.bat", 0700, time.Now(), []byte("build-contents"))
					return tarBuilder.Reader(archive.DefaultTarWriterFactory())
				},
			}, archive.DefaultTarWriterFactory(), nil)
			h.AssertNil(t, err)

			bpDescriptor := bp.Descriptor().(*dist.BuildpackDescriptor)
			h.AssertTrue(t, bpDescriptor.WithWindowsBuild)
			h.AssertFalse(t, bpDescriptor.WithLinuxBuild)

			tarPath := writeBlobToFile(bp)
			defer os.Remove(tarPath)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3/bin/build.bat",
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
				h.ContentEquals("build-contents"),
			)
		})

		it("translates blob to windows exe distribution format", func() {
			bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
				openFn: func() io.ReadCloser {
					tarBuilder := archive.TarBuilder{}
					tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"
`))

					tarBuilder.AddDir("bin", 0700, time.Now())
					tarBuilder.AddFile("bin/detect", 0700, time.Now(), []byte("detect-contents"))
					tarBuilder.AddFile("bin/build.exe", 0700, time.Now(), []byte("build-contents"))
					return tarBuilder.Reader(archive.DefaultTarWriterFactory())
				},
			}, archive.DefaultTarWriterFactory(), nil)
			h.AssertNil(t, err)

			bpDescriptor := bp.Descriptor().(*dist.BuildpackDescriptor)
			h.AssertTrue(t, bpDescriptor.WithWindowsBuild)
			h.AssertFalse(t, bpDescriptor.WithLinuxBuild)

			tarPath := writeBlobToFile(bp)
			defer os.Remove(tarPath)

			h.AssertOnTarEntry(t, tarPath,
				"/cnb/buildpacks/bp.one/1.2.3/bin/build.exe",
				h.HasFileMode(0755),
				h.HasModTime(archive.NormalizedDateTime),
				h.ContentEquals("build-contents"),
			)
		})

		it("surfaces errors encountered while reading blob", func() {
			realBlob := &readerBlob{
				openFn: func() io.ReadCloser {
					tarBuilder := archive.TarBuilder{}
					tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"

[[stacks]]
id = "some.stack.id"
`))
					return tarBuilder.Reader(archive.DefaultTarWriterFactory())
				},
			}

			bp, err := buildpack.FromBuildpackRootBlob(&errorBlob{
				realBlob: realBlob,
				limit:    4,
			}, archive.DefaultTarWriterFactory(), nil)
			h.AssertNil(t, err)

			bpReader, err := bp.Open()
			h.AssertNil(t, err)

			_, err = io.Copy(io.Discard, bpReader)
			h.AssertError(t, err, "error from errBlob (reached limit of 4)")
		})

		when("calculating permissions", func() {
			bpTOMLData := `
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"

[[stacks]]
id = "some.stack.id"
`

			when("no exec bits set", func() {
				it("sets to 0755 if directory", func() {
					bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}
							tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(bpTOMLData))
							tarBuilder.AddDir("some-dir", 0600, time.Now())
							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					}, archive.DefaultTarWriterFactory(), nil)
					h.AssertNil(t, err)

					tarPath := writeBlobToFile(bp)
					defer os.Remove(tarPath)

					h.AssertOnTarEntry(t, tarPath,
						"/cnb/buildpacks/bp.one/1.2.3/some-dir",
						h.HasFileMode(0755),
					)
				})
			})

			when("no exec bits set", func() {
				it("sets to 0755 if 'bin/detect' or 'bin/build'", func() {
					bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}
							tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(bpTOMLData))
							tarBuilder.AddFile("bin/detect", 0600, time.Now(), []byte("detect-contents"))
							tarBuilder.AddFile("bin/build", 0600, time.Now(), []byte("build-contents"))
							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					}, archive.DefaultTarWriterFactory(), nil)
					h.AssertNil(t, err)

					bpDescriptor := bp.Descriptor().(*dist.BuildpackDescriptor)
					h.AssertFalse(t, bpDescriptor.WithWindowsBuild)
					h.AssertTrue(t, bpDescriptor.WithLinuxBuild)

					tarPath := writeBlobToFile(bp)
					defer os.Remove(tarPath)

					h.AssertOnTarEntry(t, tarPath,
						"/cnb/buildpacks/bp.one/1.2.3/bin/detect",
						h.HasFileMode(0755),
					)

					h.AssertOnTarEntry(t, tarPath,
						"/cnb/buildpacks/bp.one/1.2.3/bin/build",
						h.HasFileMode(0755),
					)
				})
			})

			when("not directory, 'bin/detect', or 'bin/build'", func() {
				it("sets to 0755 if ANY exec bit is set", func() {
					bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}
							tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(bpTOMLData))
							tarBuilder.AddFile("some-file", 0700, time.Now(), []byte("some-data"))
							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					}, archive.DefaultTarWriterFactory(), nil)
					h.AssertNil(t, err)

					tarPath := writeBlobToFile(bp)
					defer os.Remove(tarPath)

					h.AssertOnTarEntry(t, tarPath,
						"/cnb/buildpacks/bp.one/1.2.3/some-file",
						h.HasFileMode(0755),
					)
				})
			})

			when("not directory, 'bin/detect', or 'bin/build'", func() {
				it("sets to 0644 if NO exec bits set", func() {
					bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}
							tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(bpTOMLData))
							tarBuilder.AddFile("some-file", 0600, time.Now(), []byte("some-data"))
							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					}, archive.DefaultTarWriterFactory(), nil)
					h.AssertNil(t, err)

					tarPath := writeBlobToFile(bp)
					defer os.Remove(tarPath)

					h.AssertOnTarEntry(t, tarPath,
						"/cnb/buildpacks/bp.one/1.2.3/some-file",
						h.HasFileMode(0644),
					)
				})
			})
		})

		when("there is no descriptor file", func() {
			it("returns error", func() {
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertError(t, err, "could not find entry path 'buildpack.toml'")
			})
		})

		when("there is no api field", func() {
			it("assumes an api version", func() {
				bp, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
[buildpack]
id = "bp.one"
version = "1.2.3"

[[stacks]]
id = "some.stack.id"`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertNil(t, err)
				h.AssertEq(t, bp.Descriptor().API().String(), "0.1")
			})
		})

		when("there is no id", func() {
			it("returns error", func() {
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
[buildpack]
id = ""
version = "1.2.3"

[[stacks]]
id = "some.stack.id"`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertError(t, err, "'buildpack.id' is required")
			})
		})

		when("there is no version", func() {
			it("returns error", func() {
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
[buildpack]
id = "bp.one"
version = ""

[[stacks]]
id = "some.stack.id"`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertError(t, err, "'buildpack.version' is required")
			})
		})

		when("both stacks and order are present", func() {
			it("returns error", func() {
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
[buildpack]
id = "bp.one"
version = "1.2.3"

[[stacks]]
id = "some.stack.id"

[[order]]
[[order.group]]
  id = "bp.nested"
  version = "bp.nested.version"
`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertError(t, err, "cannot have both 'targets'/'stacks' and an 'order' defined")
			})
		})

		when("missing stacks and order", func() {
			it("does not return an error", func() {
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
[buildpack]
id = "bp.one"
version = "1.2.3"
`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), nil)
				h.AssertNil(t, err)
			})
		})

		when("hardlink is present", func() {
			var bpRootFolder string

			it.Before(func() {
				bpRootFolder = filepath.Join("testdata", "buildpack-with-hardlink")
				// create a hard link
				err := os.Link(filepath.Join(bpRootFolder, "original-file"), filepath.Join(bpRootFolder, "original-file-2"))
				h.AssertNil(t, err)
			})

			it.After(func() {
				os.RemoveAll(filepath.Join(bpRootFolder, "original-file-2"))
			})

			it("hardlink is preserved in the output tar file", func() {
				bp, err := buildpack.FromBuildpackRootBlob(blob.NewBlob(bpRootFolder), archive.DefaultTarWriterFactory(), nil)
				h.AssertNil(t, err)

				tarPath := writeBlobToFile(bp)
				defer os.Remove(tarPath)

				h.AssertOnTarEntries(t, tarPath,
					"/cnb/buildpacks/bp.one/1.2.3/original-file",
					"/cnb/buildpacks/bp.one/1.2.3/original-file-2",
					h.AreEquivalentHardLinks(),
				)
			})
		})

		when("there are wrong things in the file", func() {
			it("warns", func() {
				outBuf := bytes.Buffer{}
				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				_, err := buildpack.FromBuildpackRootBlob(&readerBlob{
					openFn: func() io.ReadCloser {
						tarBuilder := archive.TarBuilder{}
						tarBuilder.AddFile("buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "bp.one"
version = "1.2.3"
homepage = "http://geocities.com/cool-bp"
sbom-formats = ["this should not warn"]
clear-env = true

[[targets]]
os = "some-os"
arch = "some-arch"
variant = "some-arch-variant"
[[targets.distributions]]
name = "some-distro-name"
version = "some-distro-version"
[[targets.distros]]
name = "some-distro-name"
versions = ["some-distro-version"]

[metadata]
this-key = "is totally allowed and should not warn"
`))
						return tarBuilder.Reader(archive.DefaultTarWriterFactory())
					},
				}, archive.DefaultTarWriterFactory(), logger)
				h.AssertNil(t, err)
				h.AssertContains(t, outBuf.String(), "Warning: Ignoring unexpected key(s) in descriptor for buildpack bp.one: targets.distributions, targets.distributions.name, targets.distributions.version, targets.distros.versions")
			})
		})
	})

	when("#Match", func() {
		it("compares, using only the id and version", func() {
			other := dist.ModuleInfo{
				ID:          "same",
				Version:     "1.2.3",
				Description: "something else",
				Homepage:    "something else",
				Keywords:    []string{"something", "else"},
				Licenses: []dist.License{
					{
						Type: "MIT",
						URI:  "https://example.com",
					},
				},
			}

			self := dist.ModuleInfo{
				ID:      "same",
				Version: "1.2.3",
			}

			match := self.Match(other)

			h.AssertEq(t, match, true)

			self.ID = "different"
			match = self.Match(other)

			h.AssertEq(t, match, false)
		})
	})

	when("#Set", func() {
		it("creates a set", func() {
			values := []string{"a", "b", "c", "a"}
			set := buildpack.Set(values)
			h.AssertEq(t, len(set), 3)
		})
	})

	when("#ToNLayerTar", func() {
		var (
			tmpDir     string
			expectedBP []expectedBuildpack
			err        error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "")
			h.AssertNil(t, err)
		})

		it.After(func() {
			err := os.RemoveAll(tmpDir)
			if runtime.GOOS != "windows" {
				// avoid "The process cannot access the file because it is being used by another process"
				// error on Windows
				h.AssertNil(t, err)
			}
		})

		when("BuildModule contains only an individual buildpack (default)", func() {
			it.Before(func() {
				expectedBP = []expectedBuildpack{
					{
						id:      "buildpack-1-id",
						version: "buildpack-1-version-1",
					},
				}
			})

			it("returns 1 tar files", func() {
				bp := buildpack.FromBlob(
					&dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack-1-id",
							Version: "buildpack-1-version-1",
							Name:    "buildpack-1",
						},
					},
					&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}

							// Buildpack 1
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id", 0700, time.Now())
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-1-id"
version = "buildpack-1-version-1"

`))
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/detect", 0700, time.Now(), []byte("detect-contents"))
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/build", 0700, time.Now(), []byte("build-contents"))

							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					},
				)

				tarPaths, err := buildpack.ToNLayerTar(tmpDir, bp)
				h.AssertNil(t, err)
				h.AssertEq(t, len(tarPaths), 1)
				assertBuildpacksToTar(t, tarPaths, expectedBP)
			})
		})

		when("BuildModule contains N flattened buildpacks", func() {
			it.Before(func() {
				expectedBP = []expectedBuildpack{
					{
						id:      "buildpack-1-id",
						version: "buildpack-1-version-1",
					},
					{
						id:      "buildpack-2-id",
						version: "buildpack-2-version-1",
					},
				}
			})
			when("not running on windows", func() {
				it("returns N tar files", func() {
					h.SkipIf(t, runtime.GOOS == "windows", "")
					bp := buildpack.FromBlob(
						&dist.BuildpackDescriptor{
							WithAPI: api.MustParse("0.3"),
							WithInfo: dist.ModuleInfo{
								ID:      "buildpack-1-id",
								Version: "buildpack-1-version-1",
								Name:    "buildpack-1",
							},
						},
						&readerBlob{
							openFn: func() io.ReadCloser {
								tarBuilder := archive.TarBuilder{}

								// Buildpack 1
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id", 0700, time.Now())
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1", 0700, time.Now())
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-1-id"
version = "buildpack-1-version-1"

`))
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin", 0700, time.Now())
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/detect", 0700, time.Now(), []byte("detect-contents"))
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/build", 0700, time.Now(), []byte("build-contents"))

								// Buildpack 2
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-2-id", 0700, time.Now())
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1", 0700, time.Now())
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-2-id"
version = "buildpack-2-version-1"

`))
								tarBuilder.AddDir("/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin", 0700, time.Now())
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin/detect", 0700, time.Now(), []byte("detect-contents"))
								tarBuilder.AddFile("/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin/build", 0700, time.Now(), []byte("build-contents"))

								return tarBuilder.Reader(archive.DefaultTarWriterFactory())
							},
						},
					)

					tarPaths, err := buildpack.ToNLayerTar(tmpDir, bp)
					h.AssertNil(t, err)
					h.AssertEq(t, len(tarPaths), 2)
					assertBuildpacksToTar(t, tarPaths, expectedBP)
				})
			})

			when("running on windows", func() {
				it("returns N tar files", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "")
					bp := buildpack.FromBlob(
						&dist.BuildpackDescriptor{
							WithAPI: api.MustParse("0.3"),
							WithInfo: dist.ModuleInfo{
								ID:      "buildpack-1-id",
								Version: "buildpack-1-version-1",
								Name:    "buildpack-1",
							},
						},
						&readerBlob{
							openFn: func() io.ReadCloser {
								tarBuilder := archive.TarBuilder{}
								// Windows tar format
								tarBuilder.AddDir("Files", 0700, time.Now())
								tarBuilder.AddDir("Hives", 0700, time.Now())
								tarBuilder.AddDir("Files/cnb", 0700, time.Now())
								tarBuilder.AddDir("Files/cnb/builpacks", 0700, time.Now())

								// Buildpack 1
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-1-id", 0700, time.Now())
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1", 0700, time.Now())
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-1-id"
version = "buildpack-1-version-1"

`))
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin", 0700, time.Now())
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/detect.bat", 0700, time.Now(), []byte("detect-contents"))
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/build.bat", 0700, time.Now(), []byte("build-contents"))

								// Buildpack 2
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-2-id", 0700, time.Now())
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1", 0700, time.Now())
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-2-id"
version = "buildpack-2-version-1"

`))
								tarBuilder.AddDir("Files/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin", 0700, time.Now())
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin/detect.bat", 0700, time.Now(), []byte("detect-contents"))
								tarBuilder.AddFile("Files/cnb/buildpacks/buildpack-2-id/buildpack-2-version-1/bin/build.bat", 0700, time.Now(), []byte("build-contents"))

								return tarBuilder.Reader(archive.DefaultTarWriterFactory())
							},
						},
					)

					tarPaths, err := buildpack.ToNLayerTar(tmpDir, bp)
					h.AssertNil(t, err)
					h.AssertEq(t, len(tarPaths), 2)
					assertWindowsBuildpacksToTar(t, tarPaths, expectedBP)
				})
			})
		})

		when("BuildModule contains buildpacks with same ID but different versions", func() {
			it.Before(func() {
				expectedBP = []expectedBuildpack{
					{
						id:      "buildpack-1-id",
						version: "buildpack-1-version-1",
					},
					{
						id:      "buildpack-1-id",
						version: "buildpack-1-version-2",
					},
				}
			})

			it("returns N tar files one per each version", func() {
				bp := buildpack.FromBlob(
					&dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack-1-id",
							Version: "buildpack-1-version-1",
							Name:    "buildpack-1",
						},
					},
					&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}

							// Buildpack 1
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id", 0700, time.Now())
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-1-id"
version = "buildpack-1-version-1"

`))
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/detect", 0700, time.Now(), []byte("detect-contents"))
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/bin/build", 0700, time.Now(), []byte("build-contents"))

							// Buildpack 2 same as before but with different version
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-2", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-2/buildpack.toml", 0700, time.Now(), []byte(`
api = "0.3"

[buildpack]
id = "buildpack-2-id"
version = "buildpack-2-version-1"

`))
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-2/bin", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-2/bin/detect", 0700, time.Now(), []byte("detect-contents"))
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-2/bin/build", 0700, time.Now(), []byte("build-contents"))

							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					},
				)

				tarPaths, err := buildpack.ToNLayerTar(tmpDir, bp)
				h.AssertNil(t, err)
				h.AssertEq(t, len(tarPaths), 2)
				assertBuildpacksToTar(t, tarPaths, expectedBP)
			})
		})

		when("BuildModule could not be read", func() {
			it("surfaces errors encountered while reading blob", func() {
				_, err = buildpack.ToNLayerTar(tmpDir, &errorBuildModule{})
				h.AssertError(t, err, "opening blob")
			})
		})

		when("BuildModule is empty", func() {
			it("returns a path to an empty tarball", func() {
				bp := buildpack.FromBlob(
					&dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack-1-id",
							Version: "buildpack-1-version-1",
							Name:    "buildpack-1",
						},
					},
					&readerBlob{
						openFn: func() io.ReadCloser {
							return io.NopCloser(strings.NewReader(""))
						},
					},
				)

				tarPaths, err := buildpack.ToNLayerTar(tmpDir, bp)
				h.AssertNil(t, err)
				h.AssertEq(t, len(tarPaths), 1)
				h.AssertNotNil(t, tarPaths[0].Path())
			})
		})

		when("BuildModule contains unexpected elements in the tarball file", func() {
			it.Before(func() {
				expectedBP = []expectedBuildpack{
					{
						id:      "buildpack-1-id",
						version: "buildpack-1-version-1",
					},
				}
			})

			it("throws an error", func() {
				bp := buildpack.FromBlob(
					&dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack-1-id",
							Version: "buildpack-1-version-1",
							Name:    "buildpack-1",
						},
					},
					&readerBlob{
						openFn: func() io.ReadCloser {
							tarBuilder := archive.TarBuilder{}

							// Buildpack 1
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id", 0700, time.Now())
							tarBuilder.AddDir("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1", 0700, time.Now())
							tarBuilder.AddFile("/cnb/buildpacks/buildpack-1-id/buildpack-1-version-1/../hack", 0700, time.Now(), []byte("harmful content"))
							return tarBuilder.Reader(archive.DefaultTarWriterFactory())
						},
					},
				)

				_, err = buildpack.ToNLayerTar(tmpDir, bp)
				h.AssertError(t, err, "contains unexpected special elements")
			})
		})
	})
}

type errorBlob struct {
	count    int
	limit    int
	realBlob buildpack.Blob
}

func (e *errorBlob) Open() (io.ReadCloser, error) {
	if e.count < e.limit {
		e.count += 1
		return e.realBlob.Open()
	}
	return nil, fmt.Errorf("error from errBlob (reached limit of %d)", e.limit)
}

type readerBlob struct {
	openFn func() io.ReadCloser
}

func (r *readerBlob) Open() (io.ReadCloser, error) {
	return r.openFn(), nil
}

type errorBuildModule struct {
}

func (eb *errorBuildModule) Open() (io.ReadCloser, error) {
	return nil, errors.New("something happened opening the build module")
}

func (eb *errorBuildModule) Descriptor() buildpack.Descriptor {
	return nil
}

type expectedBuildpack struct {
	id      string
	version string
}

func assertBuildpacksToTar(t *testing.T, actual []buildpack.ModuleTar, expected []expectedBuildpack) {
	t.Helper()
	for _, expectedBP := range expected {
		found := false
		for _, moduleTar := range actual {
			if expectedBP.id == moduleTar.Info().ID && expectedBP.version == moduleTar.Info().Version {
				found = true
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s", expectedBP.id),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s/%s", expectedBP.id, expectedBP.version),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s/%s/bin", expectedBP.id, expectedBP.version),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s/%s/bin/build", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s/%s/bin/detect", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("/cnb/buildpacks/%s/%s/buildpack.toml", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				break
			}
		}
		h.AssertTrue(t, found)
	}
}

func assertWindowsBuildpacksToTar(t *testing.T, actual []buildpack.ModuleTar, expected []expectedBuildpack) {
	t.Helper()
	for _, expectedBP := range expected {
		found := false
		for _, moduleTar := range actual {
			if expectedBP.id == moduleTar.Info().ID && expectedBP.version == moduleTar.Info().Version {
				found = true
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s", expectedBP.id),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s/%s", expectedBP.id, expectedBP.version),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s/%s/bin", expectedBP.id, expectedBP.version),
					h.IsDirectory(),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s/%s/bin/build.bat", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s/%s/bin/detect.bat", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				h.AssertOnTarEntry(t, moduleTar.Path(), fmt.Sprintf("Files/cnb/buildpacks/%s/%s/buildpack.toml", expectedBP.id, expectedBP.version),
					h.HasFileMode(0700),
				)
				break
			}
		}
		h.AssertTrue(t, found)
	}
}
