package strings_test

import (
	"testing"

	"github.com/buildpacks/pack/internal/strings"

	"github.com/sclevine/spec"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestValueOrDefault(t *testing.T) {
	spec.Run(t, "Strings", func(t *testing.T, when spec.G, it spec.S) {
		var (
			assert = h.NewAssertionManager(t)
		)

		when("#ValueOrDefault", func() {
			it("returns value when value is non-empty", func() {
				output := strings.ValueOrDefault("some-value", "-")
				assert.Equal(output, "some-value")
			})

			it("returns default when value is empty", func() {
				output := strings.ValueOrDefault("", "-")
				assert.Equal(output, "-")
			})
		})

		when("#Title", func() {
			it("returns the provided string with title casing", func() {
				output := strings.Title("to title case")
				assert.Equal(output, "To Title Case")
			})
		})
	})
}
