package stringset_test

import (
	"sort"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/stringset"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestStringSet(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testStringSet", testStringSet, spec.Parallel(), spec.Report(report.Terminal{}))
}

// NOTE: Do NOT use AssertSliceContains() or variants of it in theses tests, as they use `stringset`
func testStringSet(t *testing.T, when spec.G, it spec.S) {
	when("#FromSlice", func() {
		it("returns a map with elements as unique keys", func() {
			slice := []string{"a", "b", "a", "c"}

			set := stringset.FromSlice(slice)

			h.AssertEq(t, len(set), 3)

			_, ok := set["a"]
			h.AssertTrue(t, ok)

			_, ok = set["b"]
			h.AssertTrue(t, ok)

			_, ok = set["c"]
			h.AssertTrue(t, ok)
		})
	})

	when("#Compare", func() {
		it("returns elements in slice 1 but not in slice 2", func() {
			slice1 := []string{"a", "b", "c", "d"}
			slice2 := []string{"a", "c"}

			extra, _, _ := stringset.Compare(slice1, slice2)

			h.AssertEq(t, len(extra), 2)

			sort.Strings(extra)
			h.AssertEq(t, extra[0], "b")
			h.AssertEq(t, extra[1], "d")
		})

		it("returns elements in slice 2 missing from slice 1", func() {
			slice1 := []string{"a", "c"}
			slice2 := []string{"a", "b", "c", "d"}

			_, missing, _ := stringset.Compare(slice1, slice2)

			h.AssertEq(t, len(missing), 2)

			sort.Strings(missing)
			h.AssertEq(t, missing[0], "b")
			h.AssertEq(t, missing[1], "d")
		})

		it("returns elements present in both slices", func() {
			slice1 := []string{"a", "b", "c"}
			slice2 := []string{"b", "c", "d"}

			_, _, common := stringset.Compare(slice1, slice2)

			h.AssertEq(t, len(common), 2)

			sort.Strings(common)
			h.AssertEq(t, common[0], "b")
			h.AssertEq(t, common[1], "c")
		})
	})
}
