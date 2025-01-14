package pass

import (
	"sort"

	"github.com/mmcloughlin/avo/ir"
)

// RequiredISAExtensions determines ISA extensions required for the given
// function. Populates the ISA field.
func RequiredISAExtensions(fn *ir.Function) error {
	// Collect ISA set.
	set := map[string]bool{}
	for _, i := range fn.Instructions() {
		for _, isa := range i.ISA {
			set[isa] = true
		}
	}

	if len(set) == 0 {
		return nil
	}

	// Populate the function's ISA field with the unique sorted list.
	for isa := range set {
		fn.ISA = append(fn.ISA, isa)
	}
	sort.Strings(fn.ISA)

	return nil
}
