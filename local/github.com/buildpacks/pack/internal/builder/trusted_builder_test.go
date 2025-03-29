package builder_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	bldr "github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/config"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestTrustedBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Trusted Builder", trustedBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func trustedBuilder(t *testing.T, when spec.G, it spec.S) {
	when("IsKnownTrustedBuilder", func() {
		it("matches exactly", func() {
			h.AssertTrue(t, bldr.IsKnownTrustedBuilder("paketobuildpacks/builder-jammy-base"))
			h.AssertFalse(t, bldr.IsKnownTrustedBuilder("paketobuildpacks/builder-jammy-base:latest"))
			h.AssertFalse(t, bldr.IsKnownTrustedBuilder("paketobuildpacks/builder-jammy-base:1.2.3"))
			h.AssertFalse(t, bldr.IsKnownTrustedBuilder("my/private/builder"))
		})
	})

	when("IsTrustedBuilder", func() {
		it("trust image without tag", func() {
			cfg := config.Config{
				TrustedBuilders: []config.TrustedBuilder{
					{
						Name: "my/trusted/builder-jammy",
					},
				},
			}

			trustedBuilders := []string{
				"my/trusted/builder-jammy",
				"my/trusted/builder-jammy:latest",
				"my/trusted/builder-jammy:1.2.3",
			}

			untrustedBuilders := []string{
				"my/private/builder",            // random builder
				"my/trusted/builder-jammy-base", // shared prefix
			}

			for _, builder := range trustedBuilders {
				isTrusted, err := bldr.IsTrustedBuilder(cfg, builder)
				h.AssertNil(t, err)
				h.AssertTrue(t, isTrusted)
			}

			for _, builder := range untrustedBuilders {
				isTrusted, err := bldr.IsTrustedBuilder(cfg, builder)
				h.AssertNil(t, err)
				h.AssertFalse(t, isTrusted)
			}
		})
		it("trust image with tag", func() {
			cfg := config.Config{
				TrustedBuilders: []config.TrustedBuilder{
					{
						Name: "my/trusted/builder-jammy:1.2.3",
					},
					{
						Name: "my/trusted/builder-jammy:latest",
					},
				},
			}

			trustedBuilders := []string{
				"my/trusted/builder-jammy:1.2.3",
				"my/trusted/builder-jammy:latest",
			}

			untrustedBuilders := []string{
				"my/private/builder",
				"my/trusted/builder-jammy",
				"my/trusted/builder-jammy:2.0.0",
				"my/trusted/builder-jammy-base",
			}

			for _, builder := range trustedBuilders {
				isTrusted, err := bldr.IsTrustedBuilder(cfg, builder)
				h.AssertNil(t, err)
				h.AssertTrue(t, isTrusted)
			}

			for _, builder := range untrustedBuilders {
				isTrusted, err := bldr.IsTrustedBuilder(cfg, builder)
				h.AssertNil(t, err)
				h.AssertFalse(t, isTrusted)
			}
		})
	})
}
