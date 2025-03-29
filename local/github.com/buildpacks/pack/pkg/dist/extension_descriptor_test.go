package dist_test

import (
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestExtensionDescriptor(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testExtensionDescriptor", testExtensionDescriptor, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testExtensionDescriptor(t *testing.T, when spec.G, it spec.S) {
	when("#EscapedID", func() {
		it("returns escaped ID", func() {
			extDesc := dist.ExtensionDescriptor{
				WithInfo: dist.ModuleInfo{ID: "some/id"},
			}
			h.AssertEq(t, extDesc.EscapedID(), "some_id")
		})
	})

	when("#Kind", func() {
		it("returns 'extension'", func() {
			extDesc := dist.ExtensionDescriptor{}
			h.AssertEq(t, extDesc.Kind(), buildpack.KindExtension)
		})
	})

	when("#API", func() {
		it("returns the api", func() {
			extDesc := dist.ExtensionDescriptor{
				WithAPI: api.MustParse("0.99"),
			}
			h.AssertEq(t, extDesc.API().String(), "0.99")
		})
	})

	when("#Info", func() {
		it("returns the module info", func() {
			info := dist.ModuleInfo{
				ID:      "some-id",
				Name:    "some-name",
				Version: "some-version",
			}
			extDesc := dist.ExtensionDescriptor{
				WithInfo: info,
			}
			h.AssertEq(t, extDesc.Info(), info)
		})
	})

	when("#Order", func() {
		it("returns empty", func() {
			var empty dist.Order
			extDesc := dist.ExtensionDescriptor{}
			h.AssertEq(t, extDesc.Order(), empty)
		})
	})

	when("#Stacks", func() {
		it("returns empty", func() {
			var empty []dist.Stack
			extDesc := dist.ExtensionDescriptor{}
			h.AssertEq(t, extDesc.Stacks(), empty)
		})
	})

	when("#Targets", func() {
		it("returns the api", func() {
			targets := []dist.Target{{
				OS:   "fake-os",
				Arch: "fake-arch",
			}}
			extDesc := dist.ExtensionDescriptor{
				WithTargets: targets,
			}
			h.AssertEq(t, extDesc.Targets(), targets)
		})
	})
}
