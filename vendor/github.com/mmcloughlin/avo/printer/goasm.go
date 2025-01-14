package printer

import (
	"strconv"
	"strings"

	"github.com/mmcloughlin/avo/buildtags"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
)

// dot is the pesky unicode dot used in Go assembly.
const dot = "\u00b7"

type goasm struct {
	cfg Config
	prnt.Generator

	instructions []*ir.Instruction
	clear        bool
}

// NewGoAsm constructs a printer for writing Go assembly files.
func NewGoAsm(cfg Config) Printer {
	return &goasm{cfg: cfg}
}

func (p *goasm) Print(f *ir.File) ([]byte, error) {
	p.header(f)
	for _, s := range f.Sections {
		switch s := s.(type) {
		case *ir.Function:
			p.function(s)
		case *ir.Global:
			p.global(s)
		default:
			panic("unknown section type")
		}
	}
	return p.Result()
}

func (p *goasm) header(f *ir.File) {
	p.Comment(p.cfg.GeneratedWarning())

	if len(f.Constraints) > 0 {
		constraints, err := buildtags.Format(f.Constraints)
		if err != nil {
			p.AddError(err)
		}
		p.NL()
		p.Printf(constraints)
	}

	if len(f.Includes) > 0 {
		p.NL()
		p.includes(f.Includes)
	}
}

func (p *goasm) includes(paths []string) {
	for _, path := range paths {
		p.Printf("#include \"%s\"\n", path)
	}
}

func (p *goasm) function(f *ir.Function) {
	p.NL()
	p.Comment(f.Stub())

	if len(f.ISA) > 0 {
		p.Comment("Requires: " + strings.Join(f.ISA, ", "))
	}

	// Reference: https://github.com/golang/go/blob/b115207baf6c2decc3820ada4574ef4e5ad940ec/src/cmd/internal/obj/util.go#L166-L176
	//
	//		if p.As == ATEXT {
	//			// If there are attributes, print them. Otherwise, skip the comma.
	//			// In short, print one of these two:
	//			// TEXT	foo(SB), DUPOK|NOSPLIT, $0
	//			// TEXT	foo(SB), $0
	//			s := p.From.Sym.Attribute.TextAttrString()
	//			if s != "" {
	//				fmt.Fprintf(&buf, "%s%s", sep, s)
	//				sep = ", "
	//			}
	//		}
	//
	p.Printf("TEXT %s%s(SB)", dot, f.Name)
	if f.Attributes != 0 {
		p.Printf(", %s", f.Attributes.Asm())
	}
	p.Printf(", %s\n", textsize(f))

	p.clear = true
	for _, node := range f.Nodes {
		switch n := node.(type) {
		case *ir.Instruction:
			p.instruction(n)
			if n.IsTerminal || n.IsUnconditionalBranch() {
				p.flush()
			}
		case ir.Label:
			p.flush()
			p.ensureclear()
			p.Printf("%s:\n", n)
		case *ir.Comment:
			p.flush()
			p.ensureclear()
			for _, line := range n.Lines {
				p.Printf("\t// %s\n", line)
			}
		default:
			panic("unexpected node type")
		}
	}
	p.flush()
}

func (p *goasm) instruction(i *ir.Instruction) {
	p.instructions = append(p.instructions, i)
	p.clear = false
}

func (p *goasm) flush() {
	if len(p.instructions) == 0 {
		return
	}

	// Determine instruction width. Instructions with no operands are not
	// considered in this calculation.
	width := 0
	for _, i := range p.instructions {
		opcode := i.OpcodeWithSuffixes()
		if len(i.Operands) > 0 && len(opcode) > width {
			width = len(opcode)
		}
	}

	// Output instruction block.
	for _, i := range p.instructions {
		if len(i.Operands) > 0 {
			p.Printf("\t%-*s%s\n", width+1, i.OpcodeWithSuffixes(), joinOperands(i.Operands))
		} else {
			p.Printf("\t%s\n", i.OpcodeWithSuffixes())
		}
	}

	p.instructions = nil
}

func (p *goasm) ensureclear() {
	if !p.clear {
		p.NL()
		p.clear = true
	}
}

func (p *goasm) global(g *ir.Global) {
	p.NL()
	for _, d := range g.Data {
		a := operand.NewDataAddr(g.Symbol, d.Offset)
		p.Printf("DATA %s/%d, %s\n", a.Asm(), d.Value.Bytes(), d.Value.Asm())
	}
	p.Printf("GLOBL %s(SB), %s, $%d\n", g.Symbol, g.Attributes.Asm(), g.Size)
}

func textsize(f *ir.Function) string {
	// Reference: https://github.com/golang/go/blob/b115207baf6c2decc3820ada4574ef4e5ad940ec/src/cmd/internal/obj/util.go#L260-L265
	//
	//		case TYPE_TEXTSIZE:
	//			if a.Val.(int32) == objabi.ArgsSizeUnknown {
	//				str = fmt.Sprintf("$%d", a.Offset)
	//			} else {
	//				str = fmt.Sprintf("$%d-%d", a.Offset, a.Val.(int32))
	//			}
	//
	s := "$" + strconv.Itoa(f.FrameBytes())
	if argsize := f.ArgumentBytes(); argsize > 0 {
		return s + "-" + strconv.Itoa(argsize)
	}
	return s
}

func joinOperands(operands []operand.Op) string {
	asm := make([]string, len(operands))
	for i, op := range operands {
		asm[i] = op.Asm()
	}
	return strings.Join(asm, ", ")
}
