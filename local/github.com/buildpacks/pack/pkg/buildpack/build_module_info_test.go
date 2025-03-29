package buildpack_test

import (
	"fmt"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildModuleInfo(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "BuildModuleInfo", testBuildModuleInfo, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildModuleInfo(t *testing.T, when spec.G, it spec.S) {
	when("#ParseFlattenBuildModules", func() {
		when("buildpacksID have format <buildpack>@<version>", func() {
			var buildModules []string
			when("one buildpackID is provided", func() {
				it.Before(func() {
					buildModules = []string{"some-buildpack@version-1"}
				})

				it("parses successfully", func() {
					flattenModuleInfos, err := buildpack.ParseFlattenBuildModules(buildModules)
					h.AssertNil(t, err)
					h.AssertNotNil(t, flattenModuleInfos)
					h.AssertTrue(t, len(flattenModuleInfos.FlattenModules()) == 1)
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[0].ID, "some-buildpack")
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[0].Version, "version-1")
				})
			})

			when("more than one buildpackID is provided", func() {
				it.Before(func() {
					buildModules = []string{"some-buildpack@version-1, another-buildpack@version-2"}
				})

				it("parses multiple buildpackIDs", func() {
					flattenModuleInfos, err := buildpack.ParseFlattenBuildModules(buildModules)
					h.AssertNil(t, err)
					h.AssertNotNil(t, flattenModuleInfos)
					h.AssertTrue(t, len(flattenModuleInfos.FlattenModules()) == 1)
					h.AssertTrue(t, len(flattenModuleInfos.FlattenModules()[0].BuildModule()) == 2)
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[0].ID, "some-buildpack")
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[0].Version, "version-1")
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[1].ID, "another-buildpack")
					h.AssertEq(t, flattenModuleInfos.FlattenModules()[0].BuildModule()[1].Version, "version-2")
				})
			})
		})

		when("buildpacksID don't have format <buildpack>@<version>", func() {
			when("@<version> is missing", func() {
				it("errors with a descriptive message", func() {
					_, err := buildpack.ParseFlattenBuildModules([]string{"some-buildpack"})
					h.AssertNotNil(t, err)
					h.AssertError(t, err, fmt.Sprintf("invalid format %s; please use '<buildpack-id>@<buildpack-version>' to add buildpacks to be flattened", "some-buildpack"))
				})
			})

			when("<version> is missing", func() {
				it("errors with a descriptive message", func() {
					_, err := buildpack.ParseFlattenBuildModules([]string{"some-buildpack@"})
					h.AssertNotNil(t, err)
					h.AssertError(t, err, fmt.Sprintf("invalid format %s; '<buildpack-id>' and '<buildpack-version>' must be specified", "some-buildpack@"))
				})
			})

			when("<buildpack> is missing", func() {
				it("errors with a descriptive message", func() {
					_, err := buildpack.ParseFlattenBuildModules([]string{"@version-1"})
					h.AssertNotNil(t, err)
					h.AssertError(t, err, fmt.Sprintf("invalid format %s; '<buildpack-id>' and '<buildpack-version>' must be specified", "@version-1"))
				})
			})

			when("multiple @ are used", func() {
				it("errors with a descriptive message", func() {
					_, err := buildpack.ParseFlattenBuildModules([]string{"some-buildpack@@version-1"})
					h.AssertNotNil(t, err)
					h.AssertError(t, err, fmt.Sprintf("invalid format %s; please use '<buildpack-id>@<buildpack-version>' to add buildpacks to be flattened", "some-buildpack@@version-1"))
				})
			})
		})
	})
}
