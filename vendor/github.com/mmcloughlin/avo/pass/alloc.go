package pass

import (
	"errors"
	"math"
	"sort"

	"github.com/mmcloughlin/avo/reg"
)

// edge is an edge of the interference graph, indicating that registers X and Y
// must be in non-conflicting registers.
type edge struct {
	X, Y reg.ID
}

// Allocator is a graph-coloring register allocator.
type Allocator struct {
	registers  []reg.ID
	priority   map[reg.ID]int
	allocation reg.Allocation
	edges      []*edge
	possible   map[reg.ID][]reg.ID
}

// NewAllocator builds an allocator for the given physical registers.
func NewAllocator(rs []reg.Physical) (*Allocator, error) {
	// Set of IDs, excluding restricted registers.
	idset := map[reg.ID]bool{}
	for _, r := range rs {
		if (r.Info() & reg.Restricted) != 0 {
			continue
		}
		idset[r.ID()] = true
	}

	if len(idset) == 0 {
		return nil, errors.New("no allocatable registers")
	}

	// Produce slice of unique register IDs.
	var ids []reg.ID
	for id := range idset {
		ids = append(ids, id)
	}

	a := &Allocator{
		registers:  ids,
		priority:   map[reg.ID]int{},
		allocation: reg.NewEmptyAllocation(),
		possible:   map[reg.ID][]reg.ID{},
	}
	a.sortregisters()

	return a, nil
}

// NewAllocatorForKind builds an allocator for the given kind of registers.
func NewAllocatorForKind(k reg.Kind) (*Allocator, error) {
	f := reg.FamilyOfKind(k)
	if f == nil {
		return nil, errors.New("unknown register family")
	}
	return NewAllocator(f.Registers())
}

// SetPriority sets the priority of the given regiser to p. Higher priority
// registers are preferred in allocations. By default all registers have 0
// priority. Priority will only apply to subsequent Add() calls, therefore
// typically all SetPriority calls should happen at allocator initialization.
func (a *Allocator) SetPriority(id reg.ID, p int) {
	a.priority[id] = p
	a.sortregisters()
}

// sortregisters sorts the list of available registers: higher priority first,
// falling back to sorting by ID.
func (a *Allocator) sortregisters() {
	sort.Slice(a.registers, func(i, j int) bool {
		ri, rj := a.registers[i], a.registers[j]
		pi, pj := a.priority[ri], a.priority[rj]
		return (pi > pj) || (pi == pj && ri < rj)
	})
}

// AddInterferenceSet records that r interferes with every register in s. Convenience wrapper around AddInterference.
func (a *Allocator) AddInterferenceSet(r reg.Register, s reg.MaskSet) {
	for id, mask := range s {
		if (r.Mask() & mask) != 0 {
			a.AddInterference(r.ID(), id)
		}
	}
}

// AddInterference records that x and y must be assigned to non-conflicting physical registers.
func (a *Allocator) AddInterference(x, y reg.ID) {
	a.Add(x)
	a.Add(y)
	a.edges = append(a.edges, &edge{X: x, Y: y})
}

// Add adds a register to be allocated. Does nothing if the register has already been added.
func (a *Allocator) Add(v reg.ID) {
	if !v.IsVirtual() {
		return
	}
	if _, found := a.possible[v]; found {
		return
	}
	a.possible[v] = a.possibleregisters(v)
}

// Allocate allocates physical registers.
func (a *Allocator) Allocate() (reg.Allocation, error) {
	for {
		if err := a.update(); err != nil {
			return nil, err
		}

		if a.remaining() == 0 {
			break
		}

		v := a.mostrestricted()
		if err := a.alloc(v); err != nil {
			return nil, err
		}
	}
	return a.allocation, nil
}

// update possible allocations based on edges.
func (a *Allocator) update() error {
	var rem []*edge
	for _, e := range a.edges {
		x := a.allocation.LookupDefault(e.X)
		y := a.allocation.LookupDefault(e.Y)
		switch {
		case x.IsVirtual() && y.IsVirtual():
			rem = append(rem, e)
			continue
		case x.IsPhysical() && y.IsPhysical():
			if x == y {
				return errors.New("impossible register allocation")
			}
		case x.IsPhysical() && y.IsVirtual():
			a.discardconflicting(y, x)
		case x.IsVirtual() && y.IsPhysical():
			a.discardconflicting(x, y)
		default:
			panic("unreachable")
		}
	}
	a.edges = rem

	return nil
}

// mostrestricted returns the virtual register with the least possibilities.
func (a *Allocator) mostrestricted() reg.ID {
	n := int(math.MaxInt32)
	var v reg.ID
	for w, p := range a.possible {
		// On a tie, choose the smallest ID in numeric order. This avoids
		// non-deterministic allocations due to map iteration order.
		if len(p) < n || (len(p) == n && w < v) {
			n = len(p)
			v = w
		}
	}
	return v
}

// discardconflicting removes registers from vs possible list that conflict with p.
func (a *Allocator) discardconflicting(v, p reg.ID) {
	a.possible[v] = filterregisters(a.possible[v], func(r reg.ID) bool {
		return r != p
	})
}

// alloc attempts to allocate a register to v.
func (a *Allocator) alloc(v reg.ID) error {
	ps := a.possible[v]
	if len(ps) == 0 {
		return errors.New("failed to allocate registers")
	}
	p := ps[0]
	a.allocation[v] = p
	delete(a.possible, v)
	return nil
}

// remaining returns the number of unallocated registers.
func (a *Allocator) remaining() int {
	return len(a.possible)
}

// possibleregisters returns all allocate-able registers for the given virtual.
func (a *Allocator) possibleregisters(v reg.ID) []reg.ID {
	return filterregisters(a.registers, func(r reg.ID) bool {
		return v.Kind() == r.Kind()
	})
}

func filterregisters(in []reg.ID, predicate func(reg.ID) bool) []reg.ID {
	var rs []reg.ID
	for _, r := range in {
		if predicate(r) {
			rs = append(rs, r)
		}
	}
	return rs
}
