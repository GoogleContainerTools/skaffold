package pass

import (
	"errors"

	"github.com/mmcloughlin/avo/gotypes"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// ZeroExtend32BitOutputs applies the rule that "32-bit operands generate a
// 32-bit result, zero-extended to a 64-bit result in the destination
// general-purpose register" (Intel Software Developer’s Manual, Volume 1,
// 3.4.1.1).
func ZeroExtend32BitOutputs(i *ir.Instruction) error {
	for j, op := range i.Outputs {
		if !operand.IsR32(op) {
			continue
		}
		r, ok := op.(reg.GP)
		if !ok {
			panic("r32 operand should satisfy reg.GP")
		}
		i.Outputs[j] = r.As64()
	}
	return nil
}

// Liveness computes register liveness.
func Liveness(fn *ir.Function) error {
	// Note this implementation is initially naive so as to be "obviously correct".
	// There are a well-known optimizations we can apply if necessary.

	is := fn.Instructions()

	// Process instructions in reverse: poor approximation to topological sort.
	// TODO(mbm): process instructions in topological sort order
	for l, r := 0, len(is)-1; l < r; l, r = l+1, r-1 {
		is[l], is[r] = is[r], is[l]
	}

	// Initialize.
	for _, i := range is {
		i.LiveIn = reg.NewMaskSetFromRegisters(i.InputRegisters())
		i.LiveOut = reg.NewEmptyMaskSet()
	}

	// Iterative dataflow analysis.
	for {
		changes := false

		for _, i := range is {
			// out[n] = UNION[s IN succ[n]] in[s]
			for _, s := range i.Succ {
				if s == nil {
					continue
				}
				changes = i.LiveOut.Update(s.LiveIn) || changes
			}

			// in[n] = use[n] UNION (out[n] - def[n])
			def := reg.NewMaskSetFromRegisters(i.OutputRegisters())
			changes = i.LiveIn.Update(i.LiveOut.Difference(def)) || changes
		}

		if !changes {
			break
		}
	}

	return nil
}

// AllocateRegisters performs register allocation.
func AllocateRegisters(fn *ir.Function) error {
	// Initialize one allocator per kind.
	as := map[reg.Kind]*Allocator{}
	for _, i := range fn.Instructions() {
		for _, r := range i.Registers() {
			k := r.Kind()
			if _, found := as[k]; !found {
				a, err := NewAllocatorForKind(k)
				if err != nil {
					return err
				}
				as[k] = a
			}
		}
	}

	// De-prioritize the base pointer register. This can be used as a general
	// purpose register, but it's callee-save so needs to be saved/restored if
	// it is clobbered. For this reason we prefer to avoid using it unless
	// forced to by register pressure.
	for k, a := range as {
		f := reg.FamilyOfKind(k)
		for _, r := range f.Registers() {
			if (r.Info() & reg.BasePointer) != 0 {
				// Negative priority penalizes this register relative to all
				// others (having default zero priority).
				a.SetPriority(r.ID(), -1)
			}
		}
	}

	// Populate registers to be allocated.
	for _, i := range fn.Instructions() {
		for _, r := range i.Registers() {
			as[r.Kind()].Add(r.ID())
		}
	}

	// Record register interferences.
	for _, i := range fn.Instructions() {
		for _, d := range i.OutputRegisters() {
			k := d.Kind()
			out := i.LiveOut.OfKind(k)
			out.DiscardRegister(d)
			as[k].AddInterferenceSet(d, out)
		}
	}

	// Execute register allocation.
	fn.Allocation = reg.NewEmptyAllocation()
	for _, a := range as {
		al, err := a.Allocate()
		if err != nil {
			return err
		}
		if err := fn.Allocation.Merge(al); err != nil {
			return err
		}
	}

	return nil
}

// BindRegisters applies the result of register allocation, replacing all virtual registers with their assigned physical registers.
func BindRegisters(fn *ir.Function) error {
	for _, i := range fn.Instructions() {
		for idx := range i.Operands {
			i.Operands[idx] = operand.ApplyAllocation(i.Operands[idx], fn.Allocation)
		}
		for idx := range i.Inputs {
			i.Inputs[idx] = operand.ApplyAllocation(i.Inputs[idx], fn.Allocation)
		}
		for idx := range i.Outputs {
			i.Outputs[idx] = operand.ApplyAllocation(i.Outputs[idx], fn.Allocation)
		}
	}
	return nil
}

// VerifyAllocation performs sanity checks following register allocation.
func VerifyAllocation(fn *ir.Function) error {
	// All registers should be physical.
	for _, i := range fn.Instructions() {
		for _, r := range i.Registers() {
			if reg.ToPhysical(r) == nil {
				return errors.New("non physical register found")
			}
		}
	}

	return nil
}

// EnsureBasePointerCalleeSaved ensures that the base pointer register will be
// saved and restored if it has been clobbered by the function.
func EnsureBasePointerCalleeSaved(fn *ir.Function) error {
	// Check to see if the base pointer is written to.
	clobbered := false
	for _, i := range fn.Instructions() {
		for _, r := range i.OutputRegisters() {
			if p := reg.ToPhysical(r); p != nil && (p.Info()&reg.BasePointer) != 0 {
				clobbered = true
			}
		}
	}

	if !clobbered {
		return nil
	}

	// This function clobbers the base pointer register so we need to ensure it
	// will be saved and restored. The Go assembler will do this automatically,
	// with a few exceptions detailed below. In summary, we can usually ensure
	// this happens by ensuring the function is not frameless (apart from
	// NOFRAME functions).
	//
	// Reference: https://github.com/golang/go/blob/3f4977bd5800beca059defb5de4dc64cd758cbb9/src/cmd/internal/obj/x86/obj6.go#L591-L609
	//
	//		var bpsize int
	//		if ctxt.Arch.Family == sys.AMD64 &&
	//			!p.From.Sym.NoFrame() && // (1) below
	//			!(autoffset == 0 && p.From.Sym.NoSplit()) && // (2) below
	//			!(autoffset == 0 && !hasCall) { // (3) below
	//			// Make room to save a base pointer.
	//			// There are 2 cases we must avoid:
	//			// 1) If noframe is set (which we do for functions which tail call).
	//			// 2) Scary runtime internals which would be all messed up by frame pointers.
	//			//    We detect these using a heuristic: frameless nosplit functions.
	//			//    TODO: Maybe someday we label them all with NOFRAME and get rid of this heuristic.
	//			// For performance, we also want to avoid:
	//			// 3) Frameless leaf functions
	//			bpsize = ctxt.Arch.PtrSize
	//			autoffset += int32(bpsize)
	//			p.To.Offset += int64(bpsize)
	//		} else {
	//			bpsize = 0
	//		}
	//
	if fn.Attributes.NOFRAME() {
		return errors.New("NOFRAME function clobbers base pointer register")
	}

	if fn.LocalSize == 0 {
		fn.AllocLocal(int(gotypes.PointerSize))
	}

	return nil
}
