package reg

// Collection represents a collection of virtual registers. This is primarily
// useful for allocating virtual registers with distinct IDs.
type Collection struct {
	idx map[Kind]Index
}

// NewCollection builds an empty register collection.
func NewCollection() *Collection {
	return &Collection{
		idx: map[Kind]Index{},
	}
}

// VirtualRegister allocates and returns a new virtual register of the given kind and width.
func (c *Collection) VirtualRegister(k Kind, s Spec) Virtual {
	idx := c.idx[k]
	c.idx[k]++
	return NewVirtual(idx, k, s)
}

// GP8L allocates and returns a general-purpose 8-bit register (low byte).
func (c *Collection) GP8L() GPVirtual { return c.GP(S8L) }

// GP8H allocates and returns a general-purpose 8-bit register (high byte).
func (c *Collection) GP8H() GPVirtual { return c.GP(S8H) }

// GP8 allocates and returns a general-purpose 8-bit register (low byte).
func (c *Collection) GP8() GPVirtual { return c.GP8L() }

// GP16 allocates and returns a general-purpose 16-bit register.
func (c *Collection) GP16() GPVirtual { return c.GP(S16) }

// GP32 allocates and returns a general-purpose 32-bit register.
func (c *Collection) GP32() GPVirtual { return c.GP(S32) }

// GP64 allocates and returns a general-purpose 64-bit register.
func (c *Collection) GP64() GPVirtual { return c.GP(S64) }

// GP allocates and returns a general-purpose register of the given width.
func (c *Collection) GP(s Spec) GPVirtual { return newgpv(c.VirtualRegister(KindGP, s)) }

// XMM allocates and returns a 128-bit vector register.
func (c *Collection) XMM() VecVirtual { return c.Vec(S128) }

// YMM allocates and returns a 256-bit vector register.
func (c *Collection) YMM() VecVirtual { return c.Vec(S256) }

// ZMM allocates and returns a 512-bit vector register.
func (c *Collection) ZMM() VecVirtual { return c.Vec(S512) }

// Vec allocates and returns a vector register of the given width.
func (c *Collection) Vec(s Spec) VecVirtual { return newvecv(c.VirtualRegister(KindVector, s)) }

// K allocates and returns an opmask register.
func (c *Collection) K() OpmaskVirtual { return newopmaskv(c.VirtualRegister(KindOpmask, S64)) }
