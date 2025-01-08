package build

import (
	"flag"
	"os"

	"github.com/mmcloughlin/avo/attr"
	"github.com/mmcloughlin/avo/buildtags"
	"github.com/mmcloughlin/avo/gotypes"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// ctx provides a global build context.
var ctx = NewContext()

// TEXT starts building a new function called name, with attributes a, and sets its signature (see SignatureExpr).
func TEXT(name string, a attr.Attribute, signature string) {
	ctx.Function(name)
	ctx.Attributes(a)
	ctx.SignatureExpr(signature)
}

// GLOBL declares a new static global data section with the given attributes.
func GLOBL(name string, a attr.Attribute) operand.Mem {
	// TODO(mbm): should this be static?
	g := ctx.StaticGlobal(name)
	ctx.DataAttributes(a)
	return g
}

// DATA adds a data value to the active data section.
func DATA(offset int, v operand.Constant) {
	ctx.AddDatum(offset, v)
}

var flags = NewFlags(flag.CommandLine)

// Generate builds and compiles the avo file built with the global context. This
// should be the final line of any avo program. Configuration is determined from command-line flags.
func Generate() {
	if !flag.Parsed() {
		flag.Parse()
	}
	cfg := flags.Config()

	status := Main(cfg, ctx)

	// To record coverage of integration tests we wrap main() functions in a test
	// functions. In this case we need the main function to terminate, therefore we
	// only exit for failure status codes.
	if status != 0 {
		os.Exit(status)
	}
}

// Package sets the package the generated file will belong to. Required to be able to reference types in the package.
func Package(path string) { ctx.Package(path) }

// Constraints sets build constraints for the file.
func Constraints(t buildtags.ConstraintsConvertable) { ctx.Constraints(t) }

// Constraint appends a constraint to the file's build constraints.
func Constraint(t buildtags.ConstraintConvertable) { ctx.Constraint(t) }

// ConstraintExpr appends a constraint to the file's build constraints. The
// constraint to add is parsed from the given expression. The expression should
// look the same as the content following "// +build " in regular build
// constraint comments.
func ConstraintExpr(expr string) { ctx.ConstraintExpr(expr) }

// GP8L allocates and returns a general-purpose 8-bit register (low byte).
func GP8L() reg.GPVirtual { return ctx.GP8L() }

// GP8H allocates and returns a general-purpose 8-bit register (high byte).
func GP8H() reg.GPVirtual { return ctx.GP8H() }

// GP8 allocates and returns a general-purpose 8-bit register (low byte).
func GP8() reg.GPVirtual { return ctx.GP8() }

// GP16 allocates and returns a general-purpose 16-bit register.
func GP16() reg.GPVirtual { return ctx.GP16() }

// GP32 allocates and returns a general-purpose 32-bit register.
func GP32() reg.GPVirtual { return ctx.GP32() }

// GP64 allocates and returns a general-purpose 64-bit register.
func GP64() reg.GPVirtual { return ctx.GP64() }

// XMM allocates and returns a 128-bit vector register.
func XMM() reg.VecVirtual { return ctx.XMM() }

// YMM allocates and returns a 256-bit vector register.
func YMM() reg.VecVirtual { return ctx.YMM() }

// ZMM allocates and returns a 512-bit vector register.
func ZMM() reg.VecVirtual { return ctx.ZMM() }

// K allocates and returns an opmask register.
func K() reg.OpmaskVirtual { return ctx.K() }

// Param returns a the named argument of the active function.
func Param(name string) gotypes.Component { return ctx.Param(name) }

// ParamIndex returns the ith argument of the active function.
func ParamIndex(i int) gotypes.Component { return ctx.ParamIndex(i) }

// Return returns a the named return value of the active function.
func Return(name string) gotypes.Component { return ctx.Return(name) }

// ReturnIndex returns the ith argument of the active function.
func ReturnIndex(i int) gotypes.Component { return ctx.ReturnIndex(i) }

// Load the function argument src into register dst. Returns the destination
// register. This is syntactic sugar: it will attempt to select the right MOV
// instruction based on the types involved.
func Load(src gotypes.Component, dst reg.Register) reg.Register { return ctx.Load(src, dst) }

// Store register src into return value dst. This is syntactic sugar: it will
// attempt to select the right MOV instruction based on the types involved.
func Store(src reg.Register, dst gotypes.Component) { ctx.Store(src, dst) }

// Dereference loads a pointer and returns its element type.
func Dereference(ptr gotypes.Component) gotypes.Component { return ctx.Dereference(ptr) }

// Function starts building a new function with the given name.
func Function(name string) { ctx.Function(name) }

// Doc sets documentation comment lines for the currently active function.
func Doc(lines ...string) { ctx.Doc(lines...) }

// Pragma adds a compiler directive to the currently active function.
func Pragma(directive string, args ...string) { ctx.Pragma(directive, args...) }

// Attributes sets function attributes for the currently active function.
func Attributes(a attr.Attribute) { ctx.Attributes(a) }

// SignatureExpr parses the signature expression and sets it as the active function's signature.
func SignatureExpr(expr string) { ctx.SignatureExpr(expr) }

// Implement starts building a function of the given name, whose type is
// specified by a stub in the containing package.
func Implement(name string) { ctx.Implement(name) }

// AllocLocal allocates size bytes in the stack of the currently active function.
// Returns a reference to the base pointer for the newly allocated region.
func AllocLocal(size int) operand.Mem { return ctx.AllocLocal(size) }

// Label adds a label to the active function.
func Label(name string) { ctx.Label(name) }

// Comment adds comment lines to the active function.
func Comment(lines ...string) { ctx.Comment(lines...) }

// Commentf adds a formtted comment line.
func Commentf(format string, a ...any) { ctx.Commentf(format, a...) }

// ConstData builds a static data section containing just the given constant.
func ConstData(name string, v operand.Constant) operand.Mem { return ctx.ConstData(name, v) }

// Instruction adds an instruction to the active function.
func Instruction(i *ir.Instruction) { ctx.Instruction(i) }
