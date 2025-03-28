package slices_test

import (
	"testing"

	"github.com/sclevine/spec"

	"github.com/buildpacks/pack/internal/slices"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestMapString(t *testing.T) {
	spec.Run(t, "Slices", func(t *testing.T, when spec.G, it spec.S) {
		var (
			assert = h.NewAssertionManager(t)
		)

		when("#MapString", func() {
			it("maps each value", func() {
				input := []string{"hello", "1", "2", "world"}
				expected := []string{"hello.", "1.", "2.", "world."}
				fn := func(v string) string {
					return v + "."
				}

				output := slices.MapString(input, fn)
				assert.Equal(output, expected)
			})
		})
	})
}
