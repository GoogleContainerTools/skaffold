package pass

import (
	"errors"

	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
)

// Verify pass validates an avo file.
var Verify = Concat(
	InstructionPass(VerifyMemOperands),
)

// VerifyMemOperands checks the instruction's memory operands.
func VerifyMemOperands(i *ir.Instruction) error {
	for _, op := range i.Operands {
		m, ok := op.(operand.Mem)
		if !ok {
			continue
		}

		if m.Base == nil {
			return errors.New("bad memory operand: missing base register")
		}

		if m.Index != nil && m.Scale == 0 {
			return errors.New("bad memory operand: index register with scale 0")
		}
	}
	return nil
}
