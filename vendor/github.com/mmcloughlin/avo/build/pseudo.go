package build

import (
	"github.com/mmcloughlin/avo/attr"
	"github.com/mmcloughlin/avo/gotypes"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

//go:generate avogen -output zmov.go mov

// Param returns a the named argument of the active function.
func (c *Context) Param(name string) gotypes.Component {
	return c.activefunc().Signature.Params().Lookup(name)
}

// ParamIndex returns the ith argument of the active function.
func (c *Context) ParamIndex(i int) gotypes.Component {
	return c.activefunc().Signature.Params().At(i)
}

// Return returns a the named return value of the active function.
func (c *Context) Return(name string) gotypes.Component {
	return c.activefunc().Signature.Results().Lookup(name)
}

// ReturnIndex returns the ith argument of the active function.
func (c *Context) ReturnIndex(i int) gotypes.Component {
	return c.activefunc().Signature.Results().At(i)
}

// Load the function argument src into register dst. Returns the destination
// register. This is syntactic sugar: it will attempt to select the right MOV
// instruction based on the types involved.
func (c *Context) Load(src gotypes.Component, dst reg.Register) reg.Register {
	b, err := src.Resolve()
	if err != nil {
		c.adderror(err)
		return dst
	}
	c.mov(b.Addr, dst, int(gotypes.Sizes.Sizeof(b.Type)), int(dst.Size()), b.Type)
	return dst
}

// Store register src into return value dst. This is syntactic sugar: it will
// attempt to select the right MOV instruction based on the types involved.
func (c *Context) Store(src reg.Register, dst gotypes.Component) {
	b, err := dst.Resolve()
	if err != nil {
		c.adderror(err)
		return
	}
	c.mov(src, b.Addr, int(src.Size()), int(gotypes.Sizes.Sizeof(b.Type)), b.Type)
}

// Dereference loads a pointer and returns its element type.
func (c *Context) Dereference(ptr gotypes.Component) gotypes.Component {
	r := c.GP64()
	c.Load(ptr, r)
	return ptr.Dereference(r)
}

// ConstData builds a static data section containing just the given constant.
func (c *Context) ConstData(name string, v operand.Constant) operand.Mem {
	g := c.StaticGlobal(name)
	c.DataAttributes(attr.RODATA | attr.NOPTR)
	c.AppendDatum(v)
	return g
}
