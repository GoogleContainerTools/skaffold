// Package pass implements processing passes on avo Files.
package pass

import (
	"io"

	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/printer"
)

// Compile pass compiles an avo file. Upon successful completion the avo file
// may be printed to Go assembly.
var Compile = Concat(
	Verify,
	FunctionPass(PruneJumpToFollowingLabel),
	FunctionPass(PruneDanglingLabels),
	FunctionPass(LabelTarget),
	FunctionPass(CFG),
	InstructionPass(ZeroExtend32BitOutputs),
	FunctionPass(Liveness),
	FunctionPass(AllocateRegisters),
	FunctionPass(BindRegisters),
	FunctionPass(VerifyAllocation),
	FunctionPass(EnsureBasePointerCalleeSaved),
	Func(IncludeTextFlagHeader),
	FunctionPass(PruneSelfMoves),
	FunctionPass(RequiredISAExtensions),
)

// Interface for a processing pass.
type Interface interface {
	Execute(*ir.File) error
}

// Func adapts a function to the pass Interface.
type Func func(*ir.File) error

// Execute calls p.
func (p Func) Execute(f *ir.File) error {
	return p(f)
}

// FunctionPass is a convenience for implementing a full file pass with a
// function that operates on each avo Function independently.
type FunctionPass func(*ir.Function) error

// Execute calls p on every function in the file. Exits on the first error.
func (p FunctionPass) Execute(f *ir.File) error {
	for _, fn := range f.Functions() {
		if err := p(fn); err != nil {
			return err
		}
	}
	return nil
}

// InstructionPass is a convenience for implementing a full file pass with a
// function that operates on each Instruction independently.
type InstructionPass func(*ir.Instruction) error

// Execute calls p on every instruction in the file. Exits on the first error.
func (p InstructionPass) Execute(f *ir.File) error {
	for _, fn := range f.Functions() {
		for _, i := range fn.Instructions() {
			if err := p(i); err != nil {
				return err
			}
		}
	}
	return nil
}

// Concat returns a pass that executes the given passes in order, stopping on the first error.
func Concat(passes ...Interface) Interface {
	return Func(func(f *ir.File) error {
		for _, p := range passes {
			if err := p.Execute(f); err != nil {
				return err
			}
		}
		return nil
	})
}

// Output pass prints a file.
type Output struct {
	Writer  io.WriteCloser
	Printer printer.Printer
}

// Execute prints f with the configured Printer and writes output to Writer.
func (o *Output) Execute(f *ir.File) error {
	b, err := o.Printer.Print(f)
	if err != nil {
		return err
	}
	if _, err = o.Writer.Write(b); err != nil {
		return err
	}
	return o.Writer.Close()
}
