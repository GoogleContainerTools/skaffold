package printer

import (
	"go/format"

	"github.com/mmcloughlin/avo/buildtags"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/ir"
)

type stubs struct {
	cfg Config
	prnt.Generator
}

// NewStubs constructs a printer for writing stub function declarations.
func NewStubs(cfg Config) Printer {
	return &stubs{cfg: cfg}
}

func (s *stubs) Print(f *ir.File) ([]byte, error) {
	s.Comment(s.cfg.GeneratedWarning())

	if len(f.Constraints) > 0 {
		constraints, err := buildtags.Format(f.Constraints)
		if err != nil {
			s.AddError(err)
		}
		s.NL()
		s.Printf(constraints)
	}

	s.NL()
	s.Printf("package %s\n", s.cfg.Pkg)
	for _, fn := range f.Functions() {
		s.NL()
		s.Comment(fn.Doc...)
		for _, pragma := range fn.Pragmas {
			s.pragma(pragma)
		}
		s.Printf("%s\n", fn.Stub())
	}

	// Apply formatting to the result. This is the simplest way to ensure
	// comment formatting rules introduced in Go 1.19 are applied.  See
	// https://go.dev/doc/comment.
	src, err := s.Result()
	if err != nil {
		return nil, err
	}

	return format.Source(src)
}

func (s *stubs) pragma(p ir.Pragma) {
	s.Printf("//go:%s", p.Directive)
	for _, arg := range p.Arguments {
		s.Printf(" %s", arg)
	}
	s.NL()
}
