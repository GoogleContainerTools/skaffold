package reg

// MaskSet maps register IDs to masks.
type MaskSet map[ID]uint16

// NewEmptyMaskSet builds an empty register mask set.
func NewEmptyMaskSet() MaskSet {
	return MaskSet{}
}

// NewMaskSetFromRegisters forms a mask set from the given register list.
func NewMaskSetFromRegisters(rs []Register) MaskSet {
	s := NewEmptyMaskSet()
	for _, r := range rs {
		s.AddRegister(r)
	}
	return s
}

// Clone returns a copy of s.
func (s MaskSet) Clone() MaskSet {
	c := NewEmptyMaskSet()
	for id, mask := range s {
		c.Add(id, mask)
	}
	return c
}

// Add mask to the given register ID.
// Reports whether this made any change to the set.
func (s MaskSet) Add(id ID, mask uint16) bool {
	if (s[id] & mask) == mask {
		return false
	}
	s[id] |= mask
	return true
}

// AddRegister is a convenience for adding the register's (ID, mask) to the set.
// Reports whether this made any change to the set.
func (s MaskSet) AddRegister(r Register) bool {
	return s.Add(r.ID(), r.Mask())
}

// Discard clears masked bits from register ID.
// Reports whether this made any change to the set.
func (s MaskSet) Discard(id ID, mask uint16) bool {
	if curr, found := s[id]; !found || (curr&mask) == 0 {
		return false
	}
	s[id] &^= mask
	if s[id] == 0 {
		delete(s, id)
	}
	return true
}

// DiscardRegister is a convenience for discarding the register's (ID, mask) from the set.
// Reports whether this made any change to the set.
func (s MaskSet) DiscardRegister(r Register) bool {
	return s.Discard(r.ID(), r.Mask())
}

// Update adds masks in t to s.
// Reports whether this made any change to the set.
func (s MaskSet) Update(t MaskSet) bool {
	change := false
	for id, mask := range t {
		change = s.Add(id, mask) || change
	}
	return change
}

// Difference returns the set of registers in s but not t.
func (s MaskSet) Difference(t MaskSet) MaskSet {
	d := s.Clone()
	d.DifferenceUpdate(t)
	return d
}

// DifferenceUpdate removes every element of t from s.
func (s MaskSet) DifferenceUpdate(t MaskSet) bool {
	change := false
	for id, mask := range t {
		change = s.Discard(id, mask) || change
	}
	return change
}

// Equals returns true if s and t contain the same masks.
func (s MaskSet) Equals(t MaskSet) bool {
	if len(s) != len(t) {
		return false
	}
	for id, mask := range s {
		if _, found := t[id]; !found || mask != t[id] {
			return false
		}
	}
	return true
}

// OfKind returns the set of elements of s with kind k.
func (s MaskSet) OfKind(k Kind) MaskSet {
	t := NewEmptyMaskSet()
	for id, mask := range s {
		if id.Kind() == k {
			t.Add(id, mask)
		}
	}
	return t
}
