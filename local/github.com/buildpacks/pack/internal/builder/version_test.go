package builder_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestVersion(t *testing.T) {
	spec.Run(t, "testVersion", testVersion, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testVersion(t *testing.T, when spec.G, it spec.S) {
	when("#VersionMustParse", func() {
		it("parses string", func() {
			testVersion := "1.2.3"
			version := builder.VersionMustParse(testVersion)
			h.AssertEq(t, testVersion, version.String())
		})
	})

	when("#Equal", func() {
		var version = builder.VersionMustParse("1.2.3")

		it("matches", func() {
			otherVersion := builder.VersionMustParse("1.2.3")
			h.AssertTrue(t, version.Equal(otherVersion))
		})

		it("returns false if doesn't match exactly", func() {
			otherVersion := builder.VersionMustParse("1.2")
			h.AssertFalse(t, version.Equal(otherVersion))
		})

		it("handles nil case", func() {
			h.AssertFalse(t, version.Equal(nil))
		})
	})

	when("MarshalText", func() {
		it("marshals text", func() {
			testVersion := "1.2.3"
			version := builder.VersionMustParse(testVersion)
			bytesVersion, err := version.MarshalText()
			h.AssertNil(t, err)
			h.AssertEq(t, bytesVersion, []byte(testVersion))
		})
	})

	when("UnmarshalText", func() {
		it("overwrites existing version", func() {
			testVersion := "1.2.3"
			version := builder.VersionMustParse(testVersion)

			newVersion := "1.4.5"
			err := version.UnmarshalText([]byte(newVersion))
			h.AssertNil(t, err)
			h.AssertEq(t, version.String(), newVersion)
		})

		it("fails if provided invalid semver", func() {
			testVersion := "1.2.3"
			version := builder.VersionMustParse(testVersion)

			newVersion := "1.x"
			err := version.UnmarshalText([]byte(newVersion))
			h.AssertError(t, err, "invalid semantic version")
		})
	})
}
