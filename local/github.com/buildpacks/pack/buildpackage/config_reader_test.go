package buildpackage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildpackageConfigReader(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Buildpackage Config Reader", testBuildpackageConfigReader, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildpackageConfigReader(t *testing.T, when spec.G, it spec.S) {
	when("#Read", func() {
		var tmpDir string

		it.Before(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "buildpackage-config-test")
			h.AssertNil(t, err)
		})

		it.After(func() {
			os.RemoveAll(tmpDir)
		})

		it("returns default buildpack config", func() {
			expected := buildpackage.Config{
				Buildpack: dist.BuildpackURI{
					URI: ".",
				},
				Platform: dist.Platform{
					OS: "linux",
				},
			}
			actual := buildpackage.DefaultConfig()

			h.AssertEq(t, actual, expected)
		})

		it("returns default extension config", func() {
			expected := buildpackage.Config{
				Extension: dist.BuildpackURI{
					URI: ".",
				},
				Platform: dist.Platform{
					OS: "linux",
				},
			}
			actual := buildpackage.DefaultExtensionConfig()

			h.AssertEq(t, actual, expected)
		})

		it("returns correct config when provided toml file is valid", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(validPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			config, err := packageConfigReader.Read(configFile)
			h.AssertNil(t, err)

			h.AssertEq(t, config.Platform.OS, "windows")
			h.AssertEq(t, config.Buildpack.URI, "https://example.com/bp/a.tgz")
			h.AssertEq(t, len(config.Dependencies), 1)
			h.AssertEq(t, config.Dependencies[0].URI, "https://example.com/bp/b.tgz")
		})

		it("returns a config with 'linux' as default when platform is missing", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(validPackageWithoutPlatformToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			config, err := packageConfigReader.Read(configFile)
			h.AssertNil(t, err)

			h.AssertEq(t, config.Platform.OS, "linux")
		})

		it("returns an error when toml decode fails", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(brokenPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)

			h.AssertError(t, err, "decoding toml")
		})

		it("returns an error when buildpack uri is invalid", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(invalidBPURIPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "invalid locator")
			h.AssertError(t, err, "invalid/uri@version-is-invalid")
		})

		it("returns an error when platform os is invalid", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(invalidPlatformOSPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "invalid 'platform.os' configuration")
			h.AssertError(t, err, "only ['linux', 'windows'] is permitted")
		})

		it("returns an error when dependency uri is invalid", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(invalidDepURIPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "invalid locator")
			h.AssertError(t, err, "invalid/uri@version-is-invalid")
		})

		it("returns an error when unknown array table is present", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(invalidDepTablePackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "unknown configuration element")
			h.AssertError(t, err, "dependenceis")
			h.AssertNotContains(t, err.Error(), ".image")
			h.AssertError(t, err, configFile)
		})

		it("returns an error when unknown buildpack key is present", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(unknownBPKeyPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "unknown configuration element ")
			h.AssertError(t, err, "buildpack.url")
			h.AssertError(t, err, configFile)
		})

		it("returns an error when multiple unknown keys are present", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(multipleUnknownKeysPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "unknown configuration elements")
			h.AssertError(t, err, "'buildpack.url'")
			h.AssertError(t, err, "', '")
			h.AssertError(t, err, "'dependenceis'")
		})

		it("returns an error when both dependency options are configured", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(conflictingDependencyKeysPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "dependency configured with both 'uri' and 'image'")
		})

		it("returns an error no buildpack is configured", func() {
			configFile := filepath.Join(tmpDir, "package.toml")

			err := os.WriteFile(configFile, []byte(missingBuildpackPackageToml), os.ModePerm)
			h.AssertNil(t, err)

			packageConfigReader := buildpackage.NewConfigReader()

			_, err = packageConfigReader.Read(configFile)
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "missing 'buildpack.uri' configuration")
		})
	})
}

const validPackageToml = `
[buildpack]
uri = "https://example.com/bp/a.tgz"

[[dependencies]]
uri = "https://example.com/bp/b.tgz"

[platform]
os = "windows"
`

const validPackageWithoutPlatformToml = `
[buildpack]
uri = "https://example.com/bp/a.tgz"

[[dependencies]]
uri = "https://example.com/bp/b.tgz"
`

const brokenPackageToml = `
[buildpack # missing closing bracket
uri = "https://example.com/bp/a.tgz"

[dependencies]] # missing opening bracket
uri = "https://example.com/bp/b.tgz"
`

const invalidBPURIPackageToml = `
[buildpack]
uri = "invalid/uri@version-is-invalid"
`

const invalidDepURIPackageToml = `
[buildpack]
uri = "noop-buildpack.tgz"

[[dependencies]]
uri = "invalid/uri@version-is-invalid"
`

const invalidDepTablePackageToml = `
[buildpack]
uri = "noop-buildpack.tgz"

[[dependenceis]] # Notice: this is misspelled
image = "some/package-dep"
`

const invalidPlatformOSPackageToml = `
[buildpack]
uri = "https://example.com/bp/a.tgz"

[platform]
os = "some-incorrect-platform"
`

const unknownBPKeyPackageToml = `
[buildpack]
url = "noop-buildpack.tgz"
`

const multipleUnknownKeysPackageToml = `
[buildpack]
url = "noop-buildpack.tgz"

[[dependenceis]] # Notice: this is misspelled
image = "some/package-dep"
`

const conflictingDependencyKeysPackageToml = `
[buildpack]
uri = "noop-buildpack.tgz"

[[dependencies]]
uri = "bp/b"
image = "some/package-dep"
`

const missingBuildpackPackageToml = `
[[dependencies]]
uri = "bp/b"
`
