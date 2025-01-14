package operand

import (
	"fmt"

	"github.com/mmcloughlin/avo/reg"
)

// Op is an operand.
type Op interface {
	Asm() string
}

// Symbol represents a symbol name.
type Symbol struct {
	Name   string
	Static bool // only visible in current source file
}

// NewStaticSymbol builds a static Symbol. Static symbols are only visible in the current source file.
func NewStaticSymbol(name string) Symbol {
	return Symbol{Name: name, Static: true}
}

func (s Symbol) String() string {
	n := s.Name
	if s.Static {
		n += "<>"
	}
	return n
}

// Mem represents a memory reference.
type Mem struct {
	Symbol Symbol
	Disp   int
	Base   reg.Register
	Index  reg.Register
	Scale  uint8
}

// NewParamAddr is a convenience to build a Mem operand pointing to a function
// parameter, which is a named offset from the frame pointer pseudo register.
func NewParamAddr(name string, offset int) Mem {
	return Mem{
		Symbol: Symbol{
			Name:   name,
			Static: false,
		},
		Disp: offset,
		Base: reg.FramePointer,
	}
}

// NewStackAddr returns a memory reference relative to the stack pointer.
func NewStackAddr(offset int) Mem {
	return Mem{
		Disp: offset,
		Base: reg.StackPointer,
	}
}

// NewDataAddr returns a memory reference relative to the named data symbol.
func NewDataAddr(sym Symbol, offset int) Mem {
	return Mem{
		Symbol: sym,
		Disp:   offset,
		Base:   reg.StaticBase,
	}
}

// Offset returns a reference to m plus idx bytes.
func (m Mem) Offset(idx int) Mem {
	a := m
	a.Disp += idx
	return a
}

// Idx returns a new memory reference with (Index, Scale) set to (r, s).
func (m Mem) Idx(r reg.Register, s uint8) Mem {
	a := m
	a.Index = r
	a.Scale = s
	return a
}

// Asm returns an assembly syntax representation of m.
func (m Mem) Asm() string {
	a := m.Symbol.String()
	if a != "" {
		a += fmt.Sprintf("%+d", m.Disp)
	} else if m.Disp != 0 {
		a += fmt.Sprintf("%d", m.Disp)
	}
	if m.Base != nil {
		a += fmt.Sprintf("(%s)", m.Base.Asm())
	}
	if m.Index != nil && m.Scale != 0 {
		a += fmt.Sprintf("(%s*%d)", m.Index.Asm(), m.Scale)
	}
	return a
}

// Rel is an offset relative to the instruction pointer.
type Rel int32

// Asm returns an assembly syntax representation of r.
func (r Rel) Asm() string {
	return fmt.Sprintf(".%+d", r)
}

// LabelRef is a reference to a label.
type LabelRef string

// Asm returns an assembly syntax representation of l.
func (l LabelRef) Asm() string {
	return string(l)
}

// Registers returns the list of all operands involved in the given operand.
func Registers(op Op) []reg.Register {
	switch op := op.(type) {
	case reg.Register:
		return []reg.Register{op}
	case Mem:
		var r []reg.Register
		if op.Base != nil {
			r = append(r, op.Base)
		}
		if op.Index != nil {
			r = append(r, op.Index)
		}
		return r
	case Constant, Rel, LabelRef:
		return nil
	}
	panic("unknown operand type")
}

// ApplyAllocation returns an operand with allocated registers replaced. Registers missing from the allocation are left alone.
func ApplyAllocation(op Op, a reg.Allocation) Op {
	switch op := op.(type) {
	case reg.Register:
		return a.LookupRegisterDefault(op)
	case Mem:
		op.Base = a.LookupRegisterDefault(op.Base)
		op.Index = a.LookupRegisterDefault(op.Index)
		return op
	}
	return op
}
