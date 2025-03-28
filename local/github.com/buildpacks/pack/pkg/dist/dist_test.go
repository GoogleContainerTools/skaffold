package dist_test

import (
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDist(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testDist", testDist, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testDist(t *testing.T, when spec.G, it spec.S) {
	when("ModuleLayers", func() {
		when("Get", func() {
			var (
				buildpackLayers dist.ModuleLayers
				apiVersion      *api.Version
			)
			it.Before(func() {
				var err error
				apiVersion, err = api.NewVersion("0.0")
				h.AssertNil(t, err)

				buildpackLayers = dist.ModuleLayers{
					"buildpack": {
						"version1": {
							API:         apiVersion,
							LayerDiffID: "buildpack-v1-diff",
						},
					},
					"other-buildpack": {
						"version1": {
							API:         apiVersion,
							LayerDiffID: "other-buildpack-v2-diff",
						},
						"version2": {
							API:         apiVersion,
							LayerDiffID: "other-buildpack-v2-diff",
						},
					},
				}
			})

			when("ID and Version are provided and present", func() {
				it("succeeds", func() {
					out, ok := buildpackLayers.Get("buildpack", "version1")
					h.AssertEq(t, ok, true)
					h.AssertEq(t, out, dist.ModuleLayerInfo{
						API:         apiVersion,
						LayerDiffID: "buildpack-v1-diff",
					})
				})
			})

			when("ID is present, Version is left empty, but can be inferred", func() {
				it("succeeds", func() {
					out, ok := buildpackLayers.Get("buildpack", "")
					h.AssertEq(t, ok, true)
					h.AssertEq(t, out, dist.ModuleLayerInfo{
						API:         apiVersion,
						LayerDiffID: "buildpack-v1-diff",
					})
				})
			})

			when("ID is present, Version is left empty and cannot be inferred", func() {
				it("fails", func() {
					_, ok := buildpackLayers.Get("other-buildpack", "")
					h.AssertEq(t, ok, false)
				})
			})

			when("ID is NOT provided", func() {
				it("fails", func() {
					_, ok := buildpackLayers.Get("missing-buildpack", "")
					h.AssertEq(t, ok, false)
				})
			})
		})

		when("Add", func() {
			when("a new buildpack is added", func() {
				it("succeeds", func() {
					layers := dist.ModuleLayers{}
					apiVersion, _ := api.NewVersion("0.0")
					descriptor := dist.BuildpackDescriptor{WithAPI: apiVersion, WithInfo: dist.ModuleInfo{ID: "test", Name: "test", Version: "1.0"}}
					dist.AddToLayersMD(layers, &descriptor, "")
					layerInfo, ok := layers.Get(descriptor.Info().ID, descriptor.Info().Version)
					h.AssertEq(t, ok, true)
					h.AssertEq(t, layerInfo.Name, descriptor.Info().Name)
				})
			})
		})
	})

	when("ImageOrURI", func() {
		when("DisplayString", func() {
			when("uri", func() {
				when("blank", func() {
					it("returns image", func() {
						toTest := dist.ImageOrURI{
							ImageRef: dist.ImageRef{
								ImageName: "some-image-name",
							},
						}
						h.AssertEq(t, toTest.DisplayString(), "some-image-name")
					})
				})

				when("not blank", func() {
					it("returns uri", func() {
						toTest := dist.ImageOrURI{
							BuildpackURI: dist.BuildpackURI{
								URI: "some-uri",
							},
							ImageRef: dist.ImageRef{
								ImageName: "some-image-name",
							},
						}
						h.AssertEq(t, toTest.DisplayString(), "some-uri")
					})
				})
			})
		})
	})
}
