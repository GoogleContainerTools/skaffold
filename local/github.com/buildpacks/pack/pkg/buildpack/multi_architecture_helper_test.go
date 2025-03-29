package buildpack_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestMultiArchConfig(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testMultiArchConfig", testMultiArchConfig, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testMultiArchConfig(t *testing.T, when spec.G, it spec.S) {
	var (
		err                  error
		outBuf               bytes.Buffer
		logger               *logging.LogWithWriters
		multiArchConfig      *buildpack.MultiArchConfig
		targetsFromBuildpack []dist.Target
		targetsFromExtension []dist.Target
		targetsFromFlags     []dist.Target
		tmpDir               string
	)

	it.Before(func() {
		targetsFromBuildpack = []dist.Target{{OS: "linux", Arch: "amd64"}}
		targetsFromFlags = []dist.Target{{OS: "linux", Arch: "arm64", ArchVariant: "v6"}}
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		tmpDir, err = os.MkdirTemp("", "test-multi-arch")
		h.AssertNil(t, err)
	})

	it.After(func() {
		os.RemoveAll(tmpDir)
	})

	when("#Targets", func() {
		when("buildpack targets are defined", func() {
			it.Before(func() {
				multiArchConfig, err = buildpack.NewMultiArchConfig(targetsFromBuildpack, []dist.Target{}, logger)
				h.AssertNil(t, err)
			})

			it("returns buildpack targets", func() {
				h.AssertEq(t, len(multiArchConfig.Targets()), 1)
				h.AssertEq(t, multiArchConfig.Targets()[0].OS, "linux")
				h.AssertEq(t, multiArchConfig.Targets()[0].Arch, "amd64")
			})
		})

		when("buildpack targets are not defined, but flags are provided", func() {
			it.Before(func() {
				multiArchConfig, err = buildpack.NewMultiArchConfig([]dist.Target{}, targetsFromFlags, logger)
				h.AssertNil(t, err)
			})

			it("returns targets from flags", func() {
				h.AssertEq(t, len(multiArchConfig.Targets()), 1)
				h.AssertEq(t, multiArchConfig.Targets()[0].OS, "linux")
				h.AssertEq(t, multiArchConfig.Targets()[0].Arch, "arm64")
				h.AssertEq(t, multiArchConfig.Targets()[0].ArchVariant, "v6")
			})
		})

		when("buildpack targets are defined and flags are provided", func() {
			it.Before(func() {
				multiArchConfig, err = buildpack.NewMultiArchConfig(targetsFromBuildpack, targetsFromFlags, logger)
				h.AssertNil(t, err)
			})

			it("returns targets from flags", func() {
				// flags overrides the targets in the configuration files
				h.AssertEq(t, len(multiArchConfig.Targets()), 1)
				h.AssertEq(t, multiArchConfig.Targets()[0].OS, "linux")
				h.AssertEq(t, multiArchConfig.Targets()[0].Arch, "arm64")
				h.AssertEq(t, multiArchConfig.Targets()[0].ArchVariant, "v6")
			})
		})
	})

	when("#CopyConfigFiles", func() {
		when("buildpack root folder exists", func() {
			var rootFolder string

			it.Before(func() {
				rootFolder = filepath.Join(tmpDir, "some-buildpack")
				targetsFromBuildpack = []dist.Target{{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64", ArchVariant: "v8"}}
				multiArchConfig, err = buildpack.NewMultiArchConfig(targetsFromBuildpack, []dist.Target{}, logger)
				h.AssertNil(t, err)

				// dummy multi-platform buildpack structure
				os.MkdirAll(filepath.Join(rootFolder, "linux", "amd64"), 0755)
				os.MkdirAll(filepath.Join(rootFolder, "linux", "arm64", "v8"), 0755)
				_, err = os.Create(filepath.Join(rootFolder, "buildpack.toml"))
				h.AssertNil(t, err)
			})

			it("copies the buildpack.toml to each target platform folder", func() {
				paths, err := multiArchConfig.CopyConfigFiles(rootFolder, "buildpack")
				h.AssertNil(t, err)
				h.AssertEq(t, len(paths), 2)
				h.AssertPathExists(t, filepath.Join(rootFolder, "linux", "amd64", "buildpack.toml"))
				h.AssertPathExists(t, filepath.Join(rootFolder, "linux", "arm64", "v8", "buildpack.toml"))
			})
		})

		when("extension root folder exists", func() {
			var rootFolder string

			it.Before(func() {
				rootFolder = filepath.Join(tmpDir, "some-extension")
				targetsFromExtension = []dist.Target{{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64", ArchVariant: "v8"}}
				multiArchConfig, err = buildpack.NewMultiArchConfig(targetsFromExtension, []dist.Target{}, logger)
				h.AssertNil(t, err)

				// dummy multi-platform extension structure
				os.MkdirAll(filepath.Join(rootFolder, "linux", "amd64"), 0755)
				os.MkdirAll(filepath.Join(rootFolder, "linux", "arm64", "v8"), 0755)
				_, err = os.Create(filepath.Join(rootFolder, "extension.toml"))
				h.AssertNil(t, err)
			})

			it("copies the extension.toml to each target platform folder", func() {
				paths, err := multiArchConfig.CopyConfigFiles(rootFolder, "extension")
				h.AssertNil(t, err)
				h.AssertEq(t, len(paths), 2)
				h.AssertPathExists(t, filepath.Join(rootFolder, "linux", "amd64", "extension.toml"))
				h.AssertPathExists(t, filepath.Join(rootFolder, "linux", "arm64", "v8", "extension.toml"))
			})
		})
	})

	when("#PlatformRootFolder", func() {
		var target dist.Target

		when("root folder exists", func() {
			var bpURI string

			it.Before(func() {
				os.MkdirAll(filepath.Join(tmpDir, "linux", "arm64", "v8"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "windows", "amd64", "v2", "windows@10.0.20348.1970"), 0755)
				bpURI, err = paths.FilePathToURI(tmpDir, "")
				h.AssertNil(t, err)
			})

			when("target has 'os'", func() {
				when("'os' directory exists", func() {
					it.Before(func() {
						target = dist.Target{OS: "linux"}
					})

					it("returns <root>/<os directory>", func() {
						found, path := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertTrue(t, found)
						h.AssertEq(t, path, filepath.Join(tmpDir, "linux"))
					})
				})

				when("'os' directory doesn't exist", func() {
					it.Before(func() {
						target = dist.Target{OS: "darwin"}
					})

					it("returns not found", func() {
						found, _ := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertFalse(t, found)
					})
				})
			})

			when("target has 'os' and 'arch'", func() {
				when("'arch' directory exists", func() {
					it.Before(func() {
						target = dist.Target{OS: "linux", Arch: "arm64"}
					})

					it("returns <root>/<os directory>/<arch directory>", func() {
						found, path := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertTrue(t, found)
						h.AssertEq(t, path, filepath.Join(tmpDir, "linux", "arm64"))
					})
				})

				when("'arch' directory doesn't exist", func() {
					it.Before(func() {
						target = dist.Target{OS: "linux", Arch: "amd64"}
					})

					it("returns <root>/<os directory>", func() {
						found, path := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertTrue(t, found)
						h.AssertEq(t, path, filepath.Join(tmpDir, "linux"))
					})
				})
			})

			when("target has 'os', 'arch' and 'variant'", func() {
				it.Before(func() {
					target = dist.Target{OS: "linux", Arch: "arm64", ArchVariant: "v8"}
				})

				it("returns <root>/<os directory>/<arch directory>/<variant directory>", func() {
					found, path := buildpack.PlatformRootFolder(bpURI, target)
					h.AssertTrue(t, found)
					h.AssertEq(t, path, filepath.Join(tmpDir, "linux", "arm64", "v8"))
				})
			})

			when("target has 'os', 'arch', 'variant' and name@version", func() {
				when("all directories exist", func() {
					it.Before(func() {
						target = dist.Target{OS: "windows", Arch: "amd64", ArchVariant: "v2", Distributions: []dist.Distribution{{Name: "windows", Version: "10.0.20348.1970"}}}
					})

					it("returns <root>/<os directory>/<arch directory>/<variant directory>/<distro name directory>@<distro version directory>", func() {
						found, path := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertTrue(t, found)
						h.AssertEq(t, path, filepath.Join(tmpDir, "windows", "amd64", "v2", "windows@10.0.20348.1970"))
					})
				})

				when("version doesn't exist", func() {
					it.Before(func() {
						target = dist.Target{OS: "windows", Arch: "amd64", ArchVariant: "v2", Distributions: []dist.Distribution{{Name: "windows", Version: "foo"}}}
					})

					it("returns the most specific matching directory (<root>/<os directory>/<arch directory>/<variant directory>)", func() {
						found, path := buildpack.PlatformRootFolder(bpURI, target)
						h.AssertTrue(t, found)
						h.AssertEq(t, path, filepath.Join(tmpDir, "windows", "amd64", "v2"))
					})
				})
			})
		})
	})
}
