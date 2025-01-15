package ir

import (
	"errors"

	"github.com/mmcloughlin/avo/attr"
	"github.com/mmcloughlin/avo/buildtags"
	"github.com/mmcloughlin/avo/gotypes"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// Node is a part of a Function.
type Node interface {
	node()
}

// Label within a function.
type Label string

func (l Label) node() {}

// Comment represents a multi-line comment.
type Comment struct {
	Lines []string
}

func (c *Comment) node() {}

// NewComment builds a Comment consisting of the provided lines.
func NewComment(lines ...string) *Comment {
	return &Comment{
		Lines: lines,
	}
}

// Instruction is a single instruction in a function.
type Instruction struct {
	Opcode   string
	Suffixes []string
	Operands []operand.Op

	Inputs  []operand.Op
	Outputs []operand.Op

	IsTerminal       bool
	IsBranch         bool
	IsConditional    bool
	CancellingInputs bool

	// ISA is the list of required instruction set extensions.
	ISA []string

	// CFG.
	Pred []*Instruction
	Succ []*Instruction

	// LiveIn/LiveOut are sets of live register IDs pre/post execution.
	LiveIn  reg.MaskSet
	LiveOut reg.MaskSet
}

func (i *Instruction) node() {}

// OpcodeWithSuffixes returns the full opcode, including dot-separated suffixes.
func (i *Instruction) OpcodeWithSuffixes() string {
	opcode := i.Opcode
	for _, s := range i.Suffixes {
		opcode += "." + s
	}
	return opcode
}

// IsUnconditionalBranch reports whether i is an unconditional branch.
func (i Instruction) IsUnconditionalBranch() bool {
	return i.IsBranch && !i.IsConditional
}

// TargetLabel returns the label referenced by this instruction. Returns nil if
// no label is referenced.
func (i Instruction) TargetLabel() *Label {
	if !i.IsBranch {
		return nil
	}
	if len(i.Operands) == 0 {
		return nil
	}
	if ref, ok := i.Operands[0].(operand.LabelRef); ok {
		lbl := Label(ref)
		return &lbl
	}
	return nil
}

// Registers returns all registers involved in the instruction.
func (i Instruction) Registers() []reg.Register {
	var rs []reg.Register
	for _, op := range i.Operands {
		rs = append(rs, operand.Registers(op)...)
	}
	return rs
}

// InputRegisters returns all registers read by this instruction.
func (i Instruction) InputRegisters() []reg.Register {
	var rs []reg.Register
	for _, op := range i.Inputs {
		rs = append(rs, operand.Registers(op)...)
	}
	if i.CancellingInputs && rs[0] == rs[1] {
		rs = []reg.Register{}
	}
	for _, op := range i.Outputs {
		if operand.IsMem(op) {
			rs = append(rs, operand.Registers(op)...)
		}
	}
	return rs
}

// OutputRegisters returns all registers written by this instruction.
func (i Instruction) OutputRegisters() []reg.Register {
	var rs []reg.Register
	for _, op := range i.Outputs {
		if r, ok := op.(reg.Register); ok {
			rs = append(rs, r)
		}
	}
	return rs
}

// Section is a part of a file.
type Section interface {
	section()
}

// File represents an assembly file.
type File struct {
	Constraints buildtags.Constraints
	Includes    []string
	Sections    []Section
}

// NewFile initializes an empty file.
func NewFile() *File {
	return &File{}
}

// AddSection appends a Section to the file.
func (f *File) AddSection(s Section) {
	f.Sections = append(f.Sections, s)
}

// Functions returns all functions in the file.
func (f *File) Functions() []*Function {
	var fns []*Function
	for _, s := range f.Sections {
		if fn, ok := s.(*Function); ok {
			fns = append(fns, fn)
		}
	}
	return fns
}

// Pragma represents a function compiler directive.
type Pragma struct {
	Directive string
	Arguments []string
}

// Function represents an assembly function.
type Function struct {
	Name       string
	Attributes attr.Attribute
	Pragmas    []Pragma
	Doc        []string
	Signature  *gotypes.Signature
	LocalSize  int

	Nodes []Node

	// LabelTarget maps from label name to the following instruction.
	LabelTarget map[Label]*Instruction

	// Register allocation.
	Allocation reg.Allocation

	// ISA is the list of required instruction set extensions.
	ISA []string
}

func (f *Function) section() {}

// NewFunction builds an empty function of the given name.
func NewFunction(name string) *Function {
	return &Function{
		Name:      name,
		Signature: gotypes.NewSignatureVoid(),
	}
}

// AddPragma adds a pragma to this function.
func (f *Function) AddPragma(directive string, args ...string) {
	f.Pragmas = append(f.Pragmas, Pragma{
		Directive: directive,
		Arguments: args,
	})
}

// SetSignature sets the function signature.
func (f *Function) SetSignature(s *gotypes.Signature) {
	f.Signature = s
}

// AllocLocal allocates size bytes in this function's stack.
// Returns a reference to the base pointer for the newly allocated region.
func (f *Function) AllocLocal(size int) operand.Mem {
	ptr := operand.NewStackAddr(f.LocalSize)
	f.LocalSize += size
	return ptr
}

// AddInstruction appends an instruction to f.
func (f *Function) AddInstruction(i *Instruction) {
	f.AddNode(i)
}

// AddLabel appends a label to f.
func (f *Function) AddLabel(l Label) {
	f.AddNode(l)
}

// AddComment adds comment lines to f.
func (f *Function) AddComment(lines ...string) {
	f.AddNode(NewComment(lines...))
}

// AddNode appends a Node to f.
func (f *Function) AddNode(n Node) {
	f.Nodes = append(f.Nodes, n)
}

// Instructions returns just the list of instruction nodes.
func (f *Function) Instructions() []*Instruction {
	var is []*Instruction
	for _, n := range f.Nodes {
		i, ok := n.(*Instruction)
		if ok {
			is = append(is, i)
		}
	}
	return is
}

// Labels returns just the list of label nodes.
func (f *Function) Labels() []Label {
	var lbls []Label
	for _, n := range f.Nodes {
		lbl, ok := n.(Label)
		if ok {
			lbls = append(lbls, lbl)
		}
	}
	return lbls
}

// Stub returns the Go function declaration.
func (f *Function) Stub() string {
	return "func " + f.Name + f.Signature.String()
}

// FrameBytes returns the size of the stack frame in bytes.
func (f *Function) FrameBytes() int {
	return f.LocalSize
}

// ArgumentBytes returns the size of the arguments in bytes.
func (f *Function) ArgumentBytes() int {
	return f.Signature.Bytes()
}

// Datum represents a data element at a particular offset of a data section.
type Datum struct {
	Offset int
	Value  operand.Constant
}

// NewDatum builds a Datum from the given constant.
func NewDatum(offset int, v operand.Constant) Datum {
	return Datum{
		Offset: offset,
		Value:  v,
	}
}

// Interval returns the range of bytes this datum will occupy within its section.
func (d Datum) Interval() (int, int) {
	return d.Offset, d.Offset + d.Value.Bytes()
}

// Overlaps returns whether d overlaps with other.
func (d Datum) Overlaps(other Datum) bool {
	s, e := d.Interval()
	so, eo := other.Interval()
	return !(eo <= s || e <= so)
}

// Global represents a DATA section.
type Global struct {
	Symbol     operand.Symbol
	Attributes attr.Attribute
	Data       []Datum
	Size       int
}

// NewGlobal constructs an empty DATA section.
func NewGlobal(sym operand.Symbol) *Global {
	return &Global{
		Symbol: sym,
	}
}

// NewStaticGlobal is a convenience for building a static DATA section.
func NewStaticGlobal(name string) *Global {
	return NewGlobal(operand.NewStaticSymbol(name))
}

func (g *Global) section() {}

// Base returns a pointer to the start of the data section.
func (g *Global) Base() operand.Mem {
	return operand.NewDataAddr(g.Symbol, 0)
}

// Grow ensures that the data section has at least the given size.
func (g *Global) Grow(size int) {
	if g.Size < size {
		g.Size = size
	}
}

// AddDatum adds d to this data section, growing it if necessary. Errors if the datum overlaps with existing data.
func (g *Global) AddDatum(d Datum) error {
	for _, other := range g.Data {
		if d.Overlaps(other) {
			return errors.New("overlaps existing datum")
		}
	}
	g.add(d)
	return nil
}

// Append the constant to the end of the data section.
func (g *Global) Append(v operand.Constant) {
	g.add(Datum{
		Offset: g.Size,
		Value:  v,
	})
}

func (g *Global) add(d Datum) {
	_, end := d.Interval()
	g.Grow(end)
	g.Data = append(g.Data, d)
}
