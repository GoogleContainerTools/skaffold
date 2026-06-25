// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

const initialOutgoingCallSlots = 16

// outgoingCallSlots stores outstanding generated calls by their numeric JSON-RPC
// id. Conn.Call only generates numeric ids, so the hot path can avoid hashing the
// full ID value and allocating a map bucket for each in-flight call.
//
// The table is open-addressed with linear probing. A removed entry leaves a
// tombstone so lookups behind it remain reachable; tombstones are compacted when
// the table grows or when all calls have drained.
type outgoingCallSlots struct {
	slots []outgoingCallSlot
	live  int // slots currently holding a waiter
	used  int // live slots plus tombstones
}

type outgoingCallSlot struct {
	waiter *waiter
	id     int64
	used   bool
}

// Len reports the number of outstanding calls.
func (s *outgoingCallSlots) Len() int { return s.live }

// Add registers w for id. Conn.Call only supplies generated numeric ids.
func (s *outgoingCallSlots) Add(id ID, w *waiter) {
	n, ok := id.Number()
	if !ok {
		panic("jsonrpc2: outgoing call id is not numeric")
	}
	if len(s.slots) == 0 {
		s.slots = make([]outgoingCallSlot, initialOutgoingCallSlots)
	} else if (s.used+1)*2 > len(s.slots) {
		s.rehash(len(s.slots) * 2)
	}

	mask := len(s.slots) - 1
	idx := int(uint64(n) & uint64(mask))
	tombstone := -1
	for {
		slot := &s.slots[idx]
		if !slot.used {
			if tombstone >= 0 {
				slot = &s.slots[tombstone]
			} else {
				s.used++
			}
			slot.id = n
			slot.waiter = w
			slot.used = true
			s.live++
			return
		}
		if slot.waiter == nil {
			if tombstone < 0 {
				tombstone = idx
			}
		} else if slot.id == n {
			panic("jsonrpc2: duplicate outgoing call id")
		}
		idx = (idx + 1) & mask
	}
}

// Take removes and returns the waiter for id, if it is still outstanding.
func (s *outgoingCallSlots) Take(id ID) (*waiter, bool) {
	n, ok := id.Number()
	if !ok || len(s.slots) == 0 {
		return nil, false
	}

	mask := len(s.slots) - 1
	idx := int(uint64(n) & uint64(mask))
	for {
		slot := &s.slots[idx]
		if !slot.used {
			return nil, false
		}
		if slot.waiter != nil && slot.id == n {
			w := slot.waiter
			slot.id = 0
			slot.waiter = nil
			s.live--
			if s.live == 0 {
				clear(s.slots)
				s.used = 0
			}
			return w, true
		}
		idx = (idx + 1) & mask
	}
}

// Drain removes every outstanding call and invokes f with its generated id and
// waiter. It is used when the read goroutine terminates and no response can
// arrive for any remaining call.
func (s *outgoingCallSlots) Drain(f func(ID, *waiter)) {
	if s.live == 0 {
		return
	}
	for i := range s.slots {
		if w := s.slots[i].waiter; w != nil {
			f(NewNumberID(s.slots[i].id), w)
		}
		s.slots[i] = outgoingCallSlot{}
	}
	s.live = 0
	s.used = 0
}

func (s *outgoingCallSlots) rehash(size int) {
	old := s.slots
	s.slots = make([]outgoingCallSlot, size)
	s.live = 0
	s.used = 0
	for i := range old {
		if old[i].waiter != nil {
			s.addNumber(old[i].id, old[i].waiter)
		}
	}
}

func (s *outgoingCallSlots) addNumber(id int64, w *waiter) {
	mask := len(s.slots) - 1
	idx := int(uint64(id) & uint64(mask))
	for {
		slot := &s.slots[idx]
		if !slot.used {
			slot.id = id
			slot.waiter = w
			slot.used = true
			s.live++
			s.used++
			return
		}
		idx = (idx + 1) & mask
	}
}
