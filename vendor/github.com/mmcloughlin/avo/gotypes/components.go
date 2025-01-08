package gotypes

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"strconv"

	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// Sizes provides type sizes used by the standard Go compiler on amd64.
var Sizes = types.SizesFor("gc", "amd64")

// PointerSize is the size of a pointer on amd64.
var PointerSize = Sizes.Sizeof(types.Typ[types.UnsafePointer])

// Basic represents a primitive/basic type at a given memory address.
type Basic struct {
	Addr operand.Mem
	Type *types.Basic
}

// Component provides access to sub-components of a Go type.
type Component interface {
	// When the component has no further sub-components, Resolve will return a
	// reference to the components type and memory address. If there was an error
	// during any previous calls to Component methods, they will be returned at
	// resolution time.
	Resolve() (*Basic, error)
	Dereference(r reg.Register) Component // dereference a pointer
	Base() Component                      // base pointer of a string or slice
	Len() Component                       // length of a string or slice
	Cap() Component                       // capacity of a slice
	Real() Component                      // real part of a complex value
	Imag() Component                      // imaginary part of a complex value
	Index(int) Component                  // index into an array
	Field(string) Component               // access a struct field
}

// componenterr is an error that also provides a null implementation of the
// Component interface. This enables us to return an error from Component
// methods whilst also allowing method chaining to continue.
type componenterr string

func errorf(format string, args ...any) Component {
	return componenterr(fmt.Sprintf(format, args...))
}

func (c componenterr) Error() string                        { return string(c) }
func (c componenterr) Resolve() (*Basic, error)             { return nil, c }
func (c componenterr) Dereference(r reg.Register) Component { return c }
func (c componenterr) Base() Component                      { return c }
func (c componenterr) Len() Component                       { return c }
func (c componenterr) Cap() Component                       { return c }
func (c componenterr) Real() Component                      { return c }
func (c componenterr) Imag() Component                      { return c }
func (c componenterr) Index(int) Component                  { return c }
func (c componenterr) Field(string) Component               { return c }

type component struct {
	typ  types.Type
	addr operand.Mem
}

// NewComponent builds a component for the named type at the given address.
func NewComponent(t types.Type, addr operand.Mem) Component {
	return &component{
		typ:  t,
		addr: addr,
	}
}

func (c *component) Resolve() (*Basic, error) {
	b := toprimitive(c.typ)
	if b == nil {
		return nil, errors.New("component is not primitive")
	}
	return &Basic{
		Addr: c.addr,
		Type: b,
	}, nil
}

func (c *component) Dereference(r reg.Register) Component {
	p, ok := c.typ.Underlying().(*types.Pointer)
	if !ok {
		return errorf("not pointer type")
	}
	return NewComponent(p.Elem(), operand.Mem{Base: r})
}

// Reference: https://github.com/golang/go/blob/50bd1c4d4eb4fac8ddeb5f063c099daccfb71b26/src/reflect/value.go#L1800-L1804
//
//	type SliceHeader struct {
//		Data uintptr
//		Len  int
//		Cap  int
//	}
var slicehdroffsets = Sizes.Offsetsof([]*types.Var{
	types.NewField(token.NoPos, nil, "Data", types.Typ[types.Uintptr], false),
	types.NewField(token.NoPos, nil, "Len", types.Typ[types.Int], false),
	types.NewField(token.NoPos, nil, "Cap", types.Typ[types.Int], false),
})

func (c *component) Base() Component {
	if !isslice(c.typ) && !isstring(c.typ) {
		return errorf("only slices and strings have base pointers")
	}
	return c.sub("_base", int(slicehdroffsets[0]), types.Typ[types.Uintptr])
}

func (c *component) Len() Component {
	if !isslice(c.typ) && !isstring(c.typ) {
		return errorf("only slices and strings have length fields")
	}
	return c.sub("_len", int(slicehdroffsets[1]), types.Typ[types.Int])
}

func (c *component) Cap() Component {
	if !isslice(c.typ) {
		return errorf("only slices have capacity fields")
	}
	return c.sub("_cap", int(slicehdroffsets[2]), types.Typ[types.Int])
}

func (c *component) Real() Component {
	if !iscomplex(c.typ) {
		return errorf("only complex types have real values")
	}
	f := complextofloat(c.typ)
	return c.sub("_real", 0, f)
}

func (c *component) Imag() Component {
	if !iscomplex(c.typ) {
		return errorf("only complex types have imaginary values")
	}
	f := complextofloat(c.typ)
	return c.sub("_imag", int(Sizes.Sizeof(f)), f)
}

func (c *component) Index(i int) Component {
	a, ok := c.typ.Underlying().(*types.Array)
	if !ok {
		return errorf("not array type")
	}
	if int64(i) >= a.Len() {
		return errorf("array index out of bounds")
	}
	// Reference: https://github.com/golang/tools/blob/bcd4e47d02889ebbc25c9f4bf3d27e4124b0bf9d/go/analysis/passes/asmdecl/asmdecl.go#L482-L494
	//
	//		case asmArray:
	//			tu := t.Underlying().(*types.Array)
	//			elem := tu.Elem()
	//			// Calculate offset of each element array.
	//			fields := []*types.Var{
	//				types.NewVar(token.NoPos, nil, "fake0", elem),
	//				types.NewVar(token.NoPos, nil, "fake1", elem),
	//			}
	//			offsets := arch.sizes.Offsetsof(fields)
	//			elemoff := int(offsets[1])
	//			for i := 0; i < int(tu.Len()); i++ {
	//				cc = appendComponentsRecursive(arch, elem, cc, suffix+"_"+strconv.Itoa(i), i*elemoff)
	//			}
	//
	elem := a.Elem()
	elemsize := int(Sizes.Sizeof(types.NewArray(elem, 2)) - Sizes.Sizeof(types.NewArray(elem, 1)))
	return c.sub("_"+strconv.Itoa(i), i*elemsize, elem)
}

func (c *component) Field(n string) Component {
	s, ok := c.typ.Underlying().(*types.Struct)
	if !ok {
		return errorf("not struct type")
	}
	// Reference: https://github.com/golang/tools/blob/13ba8ad772dfbf0f451b5dd0679e9c5605afc05d/go/analysis/passes/asmdecl/asmdecl.go#L471-L480
	//
	//		case asmStruct:
	//			tu := t.Underlying().(*types.Struct)
	//			fields := make([]*types.Var, tu.NumFields())
	//			for i := 0; i < tu.NumFields(); i++ {
	//				fields[i] = tu.Field(i)
	//			}
	//			offsets := arch.sizes.Offsetsof(fields)
	//			for i, f := range fields {
	//				cc = appendComponentsRecursive(arch, f.Type(), cc, suffix+"_"+f.Name(), off+int(offsets[i]))
	//			}
	//
	fields := make([]*types.Var, s.NumFields())
	for i := 0; i < s.NumFields(); i++ {
		fields[i] = s.Field(i)
	}
	offsets := Sizes.Offsetsof(fields)
	for i, f := range fields {
		if f.Name() == n {
			return c.sub("_"+n, int(offsets[i]), f.Type())
		}
	}
	return errorf("struct does not have field '%s'", n)
}

func (c *component) sub(suffix string, offset int, t types.Type) *component {
	s := *c
	if s.addr.Symbol.Name != "" {
		s.addr.Symbol.Name += suffix
	}
	s.addr = s.addr.Offset(offset)
	s.typ = t
	return &s
}

func isslice(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}

func isstring(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

func iscomplex(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && (b.Info()&types.IsComplex) != 0
}

func complextofloat(t types.Type) types.Type {
	switch Sizes.Sizeof(t) {
	case 16:
		return types.Typ[types.Float64]
	case 8:
		return types.Typ[types.Float32]
	}
	panic("bad")
}

// toprimitive determines whether t is primitive (cannot be reduced into
// components). If it is, it returns the basic type for t, otherwise returns
// nil.
func toprimitive(t types.Type) *types.Basic {
	switch b := t.(type) {
	case *types.Basic:
		if (b.Info() & (types.IsString | types.IsComplex)) == 0 {
			return b
		}
	case *types.Pointer:
		return types.Typ[types.Uintptr]
	}
	return nil
}
