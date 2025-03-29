package builder_test

import (
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDescriptor(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder", testDescriptor, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testDescriptor(t *testing.T, when spec.G, it spec.S) {
	when("CompatDescriptor", func() {
		when("missing apis", func() {
			it("makes a lifecycle from a blob", func() {
				descriptor := builder.CompatDescriptor(builder.LifecycleDescriptor{
					Info: builder.LifecycleInfo{},
					API: builder.LifecycleAPI{
						BuildpackVersion: api.MustParse("0.2"),
						PlatformVersion:  api.MustParse("0.3"),
					},
				})

				h.AssertEq(t, descriptor.API.BuildpackVersion.String(), "0.2")
				h.AssertEq(t, descriptor.API.PlatformVersion.String(), "0.3")

				// fill supported with deprecated field
				h.AssertEq(t, descriptor.APIs.Buildpack.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Buildpack.Supported.AsStrings(), []string{"0.2"})
				h.AssertEq(t, descriptor.APIs.Platform.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Platform.Supported.AsStrings(), []string{"0.3"})
			})
		})

		when("missing api", func() {
			it("sets lowest value supported", func() {
				descriptor := builder.CompatDescriptor(builder.LifecycleDescriptor{
					APIs: builder.LifecycleAPIs{
						Buildpack: builder.APIVersions{
							Supported: builder.APISet{api.MustParse("0.2"), api.MustParse("0.3")},
						},
						Platform: builder.APIVersions{
							Supported: builder.APISet{api.MustParse("1.2"), api.MustParse("2.3")},
						},
					},
				})

				h.AssertEq(t, descriptor.APIs.Buildpack.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Buildpack.Supported.AsStrings(), []string{"0.2", "0.3"})
				h.AssertEq(t, descriptor.APIs.Platform.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Platform.Supported.AsStrings(), []string{"1.2", "2.3"})

				// select lowest value for deprecated parameters
				h.AssertEq(t, descriptor.API.BuildpackVersion.String(), "0.2")
				h.AssertEq(t, descriptor.API.PlatformVersion.String(), "1.2")
			})

			it("sets lowest value supported + deprecated", func() {
				descriptor := builder.CompatDescriptor(builder.LifecycleDescriptor{
					APIs: builder.LifecycleAPIs{
						Buildpack: builder.APIVersions{
							Deprecated: builder.APISet{api.MustParse("0.1")},
							Supported:  builder.APISet{api.MustParse("0.2"), api.MustParse("0.3")},
						},
						Platform: builder.APIVersions{
							Deprecated: builder.APISet{api.MustParse("1.1")},
							Supported:  builder.APISet{api.MustParse("1.2"), api.MustParse("2.3")},
						},
					},
				})

				h.AssertEq(t, descriptor.APIs.Buildpack.Deprecated.AsStrings(), []string{"0.1"})
				h.AssertEq(t, descriptor.APIs.Buildpack.Supported.AsStrings(), []string{"0.2", "0.3"})
				h.AssertEq(t, descriptor.APIs.Platform.Deprecated.AsStrings(), []string{"1.1"})
				h.AssertEq(t, descriptor.APIs.Platform.Supported.AsStrings(), []string{"1.2", "2.3"})

				// select lowest value for deprecated parameters
				h.AssertEq(t, descriptor.API.BuildpackVersion.String(), "0.1")
				h.AssertEq(t, descriptor.API.PlatformVersion.String(), "1.1")
			})
		})

		when("missing api + apis", func() {
			it("makes a lifecycle from a blob", func() {
				descriptor := builder.CompatDescriptor(builder.LifecycleDescriptor{})

				h.AssertNil(t, descriptor.API.BuildpackVersion)
				h.AssertNil(t, descriptor.API.PlatformVersion)

				// fill supported with deprecated field
				h.AssertEq(t, descriptor.APIs.Buildpack.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Buildpack.Supported.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Platform.Deprecated.AsStrings(), []string{})
				h.AssertEq(t, descriptor.APIs.Platform.Supported.AsStrings(), []string{})
			})
		})
	})

	when("Earliest", func() {
		it("returns lowest value", func() {
			h.AssertEq(
				t,
				builder.APISet{api.MustParse("2.1"), api.MustParse("0.1"), api.MustParse("1.1")}.Earliest().String(),
				"0.1",
			)
		})
	})

	when("Latest", func() {
		it("returns highest value", func() {
			h.AssertEq(
				t,
				builder.APISet{api.MustParse("1.1"), api.MustParse("2.1"), api.MustParse("0.1")}.Latest().String(),
				"2.1",
			)
		})
	})
}
