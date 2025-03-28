package target_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/target"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPlatforms(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "TestPlatforms", testPlatforms, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPlatforms(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		var err error
		h.AssertNil(t, err)
	})
	when("target#SupportsPlatform", func() {
		it("should return false when target not supported", func() {
			b := target.SupportsPlatform("os", "arm", "v6")
			h.AssertFalse(t, b)
		})
		it("should parse targets as expected", func() {
			b := target.SupportsPlatform("linux", "arm", "v6")
			h.AssertTrue(t, b)
		})
	})
}
