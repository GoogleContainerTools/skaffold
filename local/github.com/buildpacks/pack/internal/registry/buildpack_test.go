package registry_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/registry"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRegistryBuildpack(t *testing.T) {
	spec.Run(t, "Buildpack", testRegistryBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testRegistryBuildpack(t *testing.T, when spec.G, it spec.S) {
	when("#Validate", func() {
		it("errors when address is missing", func() {
			b := registry.Buildpack{
				Address: "",
			}

			h.AssertNotNil(t, registry.Validate(b))
		})

		it("errors when not a digest", func() {
			b := registry.Buildpack{
				Address: "example.com/some/package:18",
			}

			h.AssertNotNil(t, registry.Validate(b))
		})

		it("succeeds when address is a digest", func() {
			b := registry.Buildpack{
				Address: "example.com/some/package@sha256:8c27fe111c11b722081701dfed3bd55e039b9ce92865473cf4cdfa918071c566",
			}

			h.AssertNil(t, registry.Validate(b))
		})
	})

	when("#ParseNamespaceName", func() {
		it("should parse buildpack id into namespace and name", func() {
			const id = "heroku/rust@1.2.3"
			namespace, name, err := registry.ParseNamespaceName(id)

			h.AssertNil(t, err)
			h.AssertEq(t, namespace, "heroku")
			h.AssertEq(t, name, "rust@1.2.3")
		})

		it("should provide an error for invalid id", func() {
			const id = "bad id"
			_, _, err := registry.ParseNamespaceName(id)

			h.AssertNotNil(t, err)
		})
	})
}
