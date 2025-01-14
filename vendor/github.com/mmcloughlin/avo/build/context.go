package build

import (
	"errors"
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/mmcloughlin/avo/attr"
	"github.com/mmcloughlin/avo/buildtags"
	"github.com/mmcloughlin/avo/gotypes"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

//go:generate avogen -output zinstructions.go build
//go:generate avogen -output zinstructions_test.go buildtest

// Context maintains state for incrementally building an avo File.
type Context struct {
	pkg      *packages.Package
	file     *ir.File
	function *ir.Function
	global   *ir.Global
	errs     ErrorList
	reg.Collection
}

// NewContext initializes an empty build Context.
func NewContext() *Context {
	return &Context{
		file:       ir.NewFile(),
		Collection: *reg.NewCollection(),
	}
}

// Package sets the package the generated file will belong to. Required to be able to reference types in the package.
func (c *Context) Package(path string) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
	}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		c.adderror(err)
		return
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		for _, err := range pkg.Errors {
			c.adderror(err)
		}
		return
	}
	c.pkg = pkg
}

// Constraints sets build constraints for the file.
func (c *Context) Constraints(t buildtags.ConstraintsConvertable) {
	cs := t.ToConstraints()
	if err := cs.Validate(); err != nil {
		c.adderror(err)
		return
	}
	c.file.Constraints = cs
}

// Constraint appends a constraint to the file's build constraints.
func (c *Context) Constraint(t buildtags.ConstraintConvertable) {
	c.Constraints(append(c.file.Constraints, t.ToConstraint()))
}

// ConstraintExpr appends a constraint to the file's build constraints. The
// constraint to add is parsed from the given expression. The expression should
// look the same as the content following "// +build " in regular build
// constraint comments.
func (c *Context) ConstraintExpr(expr string) {
	constraint, err := buildtags.ParseConstraint(expr)
	if err != nil {
		c.adderror(err)
		return
	}
	c.Constraint(constraint)
}

// Function starts building a new function with the given name.
func (c *Context) Function(name string) {
	c.function = ir.NewFunction(name)
	c.file.AddSection(c.function)
}

// Doc sets documentation comment lines for the currently active function.
func (c *Context) Doc(lines ...string) {
	c.activefunc().Doc = lines
}

// Pragma adds a compiler directive to the currently active function.
func (c *Context) Pragma(directive string, args ...string) {
	c.activefunc().AddPragma(directive, args...)
}

// Attributes sets function attributes for the currently active function.
func (c *Context) Attributes(a attr.Attribute) {
	c.activefunc().Attributes = a
}

// Signature sets the signature for the currently active function.
func (c *Context) Signature(s *gotypes.Signature) {
	c.activefunc().SetSignature(s)
}

// SignatureExpr parses the signature expression and sets it as the active function's signature.
func (c *Context) SignatureExpr(expr string) {
	s, err := gotypes.ParseSignatureInPackage(c.types(), expr)
	if err != nil {
		c.adderror(err)
		return
	}
	c.Signature(s)
}

// Implement starts building a function of the given name, whose type is
// specified by a stub in the containing package.
func (c *Context) Implement(name string) {
	pkg := c.types()
	if pkg == nil {
		c.adderrormessage("no package specified")
		return
	}
	s, err := gotypes.LookupSignature(pkg, name)
	if err != nil {
		c.adderror(err)
		return
	}
	c.Function(name)
	c.Signature(s)
}

func (c *Context) types() *types.Package {
	if c.pkg == nil {
		return nil
	}
	return c.pkg.Types
}

// AllocLocal allocates size bytes in the stack of the currently active function.
// Returns a reference to the base pointer for the newly allocated region.
func (c *Context) AllocLocal(size int) operand.Mem {
	return c.activefunc().AllocLocal(size)
}

// Instruction adds an instruction to the active function.
func (c *Context) Instruction(i *ir.Instruction) {
	c.activefunc().AddInstruction(i)
}

// Label adds a label to the active function.
func (c *Context) Label(name string) {
	c.activefunc().AddLabel(ir.Label(name))
}

// Comment adds comment lines to the active function.
func (c *Context) Comment(lines ...string) {
	c.activefunc().AddComment(lines...)
}

// Commentf adds a formtted comment line.
func (c *Context) Commentf(format string, a ...any) {
	c.Comment(fmt.Sprintf(format, a...))
}

func (c *Context) activefunc() *ir.Function {
	if c.function == nil {
		c.adderrormessage("no active function")
		return ir.NewFunction("")
	}
	return c.function
}

// StaticGlobal adds a new static data section to the file and returns a pointer to it.
func (c *Context) StaticGlobal(name string) operand.Mem {
	c.global = ir.NewStaticGlobal(name)
	c.file.AddSection(c.global)
	return c.global.Base()
}

// DataAttributes sets the attributes on the current active global data section.
func (c *Context) DataAttributes(a attr.Attribute) {
	c.activeglobal().Attributes = a
}

// AddDatum adds constant v at offset to the current active global data section.
func (c *Context) AddDatum(offset int, v operand.Constant) {
	if err := c.activeglobal().AddDatum(ir.NewDatum(offset, v)); err != nil {
		c.adderror(err)
	}
}

// AppendDatum appends a constant to the current active global data section.
func (c *Context) AppendDatum(v operand.Constant) {
	c.activeglobal().Append(v)
}

func (c *Context) activeglobal() *ir.Global {
	if c.global == nil {
		c.adderrormessage("no active global")
		return ir.NewStaticGlobal("")
	}
	return c.global
}

func (c *Context) adderror(err error) {
	c.errs.addext(err)
}

func (c *Context) adderrormessage(msg string) {
	c.adderror(errors.New(msg))
}

// Result returns the built file and any accumulated errors.
func (c *Context) Result() (*ir.File, error) {
	return c.file, c.errs.Err()
}
