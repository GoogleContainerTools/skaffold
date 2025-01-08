package x86

import (
	"errors"

	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
)

// build constructs an instruction object from a list of acceptable forms, and
// given input operands and suffixes.
func build(forms []form, suffixes sffxs, ops []operand.Op) (*ir.Instruction, error) {
	for i := range forms {
		f := &forms[i]
		if f.match(suffixes, ops) {
			return f.build(suffixes, ops), nil
		}
	}
	return nil, errors.New("bad operands")
}

// form represents an instruction form.
type form struct {
	Opcode        opc
	SuffixesClass sffxscls
	Features      feature
	ISAs          isas
	Arity         uint8
	Operands      oprnds
}

// feature is a flags enumeration type representing instruction properties.
type feature uint8

const (
	featureTerminal feature = 1 << iota
	featureBranch
	featureConditionalBranch
	featureCancellingInputs
)

// oprnds is a list of explicit and implicit operands of an instruction form.
// The size of the array is output by optab generator.
type oprnds [maxoperands]oprnd

// oprnd represents an explicit or implicit operand to an instruction form.
type oprnd struct {
	Type     uint8
	Implicit bool
	Action   action
}

// action an instruction form applies to an operand.
type action uint8

const (
	actionN action = iota
	actionR
	actionW
	actionRW action = actionR | actionW
)

// Read reports if the action includes read.
func (a action) Read() bool { return (a & actionR) != 0 }

// Read reports if the action includes write.
func (a action) Write() bool { return (a & actionW) != 0 }

// match reports whether this form matches the given suffixes and operand
// list.
func (f *form) match(suffixes sffxs, ops []operand.Op) bool {
	// Match suffix.
	accept := f.SuffixesClass.SuffixesSet()
	if !accept[suffixes] {
		return false
	}

	// Match operands.
	if len(ops) != int(f.Arity) {
		return false
	}

	for i, op := range ops {
		t := oprndtype(f.Operands[i].Type)
		if !t.Match(op) {
			return false
		}
	}

	return true
}

// build the full instruction object for this form and the given suffixes and
// operands. Assumes the form already matches the inputs.
func (f *form) build(suffixes sffxs, ops []operand.Op) *ir.Instruction {
	// Base instruction properties.
	i := &ir.Instruction{
		Opcode:           f.Opcode.String(),
		Suffixes:         suffixes.Strings(),
		Operands:         ops,
		IsTerminal:       (f.Features & featureTerminal) != 0,
		IsBranch:         (f.Features & featureBranch) != 0,
		IsConditional:    (f.Features & featureConditionalBranch) != 0,
		CancellingInputs: (f.Features & featureCancellingInputs) != 0,
		ISA:              f.ISAs.List(),
	}

	// Input/output operands.
	for _, spec := range f.Operands {
		if spec.Type == 0 {
			break
		}

		var op operand.Op
		if spec.Implicit {
			op = implreg(spec.Type).Register()
		} else {
			op, ops = ops[0], ops[1:]
		}

		if spec.Action.Read() {
			i.Inputs = append(i.Inputs, op)
		}
		if spec.Action.Write() {
			i.Outputs = append(i.Outputs, op)
		}
	}

	return i
}
