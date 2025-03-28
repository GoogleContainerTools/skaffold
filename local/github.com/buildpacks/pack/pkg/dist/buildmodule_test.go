package dist_test

import (
	"testing"

	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"

	"github.com/heroku/color"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestBuildModule(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testBuildModule", testBuildModule, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildModule(t *testing.T, when spec.G, it spec.S) {
	var info dist.ModuleInfo

	it.Before(func() {
		info = dist.ModuleInfo{
			ID:      "some-id",
			Name:    "some-name",
			Version: "some-version",
		}
	})

	when("#FullName", func() {
		when("version", func() {
			when("blank", func() {
				it.Before(func() {
					info.Version = ""
				})

				it("prints ID", func() {
					h.AssertEq(t, info.FullName(), "some-id")
				})
			})

			when("not blank", func() {
				it("prints ID and version", func() {
					h.AssertEq(t, info.FullName(), "some-id@some-version")
				})
			})
		})
	})

	when("#FullNameWithVersion", func() {
		when("version", func() {
			when("blank", func() {
				it.Before(func() {
					info.Version = ""
				})

				it("errors", func() {
					_, err := info.FullNameWithVersion()
					h.AssertNotNil(t, err)
				})
			})

			when("not blank", func() {
				it("prints ID and version", func() {
					actual, err := info.FullNameWithVersion()
					h.AssertNil(t, err)
					h.AssertEq(t, actual, "some-id@some-version")
				})
			})
		})
	})

	when("#String", func() {
		it("returns #FullName", func() {
			info.Version = ""
			h.AssertEq(t, info.String(), info.FullName())
		})
	})

	when("#Match", func() {
		when("IDs and versions match", func() {
			it("returns true", func() {
				other := dist.ModuleInfo{
					ID:      "some-id",
					Version: "some-version",
				}
				h.AssertEq(t, info.Match(other), true)
			})
		})

		when("only IDs match", func() {
			it("returns false", func() {
				other := dist.ModuleInfo{
					ID:      "some-id",
					Version: "some-other-version",
				}
				h.AssertEq(t, info.Match(other), false)
			})
		})
	})
}
