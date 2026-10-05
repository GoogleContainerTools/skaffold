// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import "context"

// SingleClient is the single-flight, caller-owned-read-loop client mode.
//
// It is an alias of [SyncClient] so the current low-latency synchronous client
// becomes the named baseline for the mode split without adding another runtime
// layer.
type SingleClient = SyncClient

// NewSingleClient creates a [SingleClient] over stream.
func NewSingleClient(stream Stream, opts ...Option) (*SingleClient, error) {
	return NewSyncClient(stream, opts...)
}

// Peer is the bidirectional JSON-RPC endpoint mode.
//
// It names the existing [Conn] capability set explicitly: both sides may send
// calls, notifications, and responses, with [Async] required for handlers that
// wait on server-initiated calls.
type Peer = Conn

// NewPeer creates a bidirectional peer endpoint over stream.
func NewPeer(stream Stream, opts ...Option) Peer {
	return NewConn(stream, opts...)
}

// Server is the server-oriented runtime mode.
//
// It currently uses [Conn] because the existing connection already provides the
// server read loop, preemption, async handler release, and batch dispatch. The
// separate name keeps server-only call sites from depending on client-mode
// constructors.
type Server = Conn

// NewServer creates a server endpoint over stream.
func NewServer(stream Stream, opts ...Option) Server {
	return NewConn(stream, opts...)
}

// BatchClient is the raw-frame batch client mode.
//
// It exposes only frame primitives so batch-mode callers can write a complete
// JSON-RPC batch array and read the response frame without passing through the
// single-message [Conn.Call] API. The supplied Stream must be frame-capable; the
// built-in framers satisfy this requirement.
type BatchClient struct {
	stream Stream
	frames frameStream
}

// NewBatchClient creates a raw-frame batch client over stream.
func NewBatchClient(stream Stream) (*BatchClient, error) {
	frames, ok := stream.(frameStream)
	if !ok {
		return nil, errFrameStreamRequired
	}
	return &BatchClient{stream: stream, frames: frames}, nil
}

// WriteFrame writes one already-encoded JSON frame.
func (c *BatchClient) WriteFrame(ctx context.Context, data []byte) (int64, error) {
	return c.frames.WriteFrame(ctx, data)
}

// ReadFrame reads one raw JSON response frame.
func (c *BatchClient) ReadFrame(ctx context.Context) (data []byte, n int64, err error) {
	return c.frames.ReadFrame(ctx)
}

// Close closes the underlying stream.
func (c *BatchClient) Close() error {
	return c.stream.Close()
}

// BatchServer is the batch-capable server endpoint mode.
//
// It names the existing Conn batch dispatch path explicitly. New batch-only
// server APIs can grow behind this type without changing single-client or peer
// constructors.
type BatchServer struct {
	Conn
}

// NewBatchServer creates a batch-capable server endpoint over stream.
func NewBatchServer(stream Stream, opts ...Option) *BatchServer {
	return &BatchServer{Conn: NewConn(stream, opts...)}
}

const errFrameStreamRequired = constError("jsonrpc2: batch mode requires a frame-capable stream")

// denseCallSlots stores waiters for generated, monotonically increasing numeric
// call IDs.
//
// Unlike outgoingCallSlots, it does not need tombstones or probing for generated
// dense IDs. It keeps a power-of-two ring window from base through base+len-1
// and advances base as low IDs are retired.
type denseCallSlots struct {
	slots []denseCallSlot
	base  int64
	live  int
}

type denseCallSlot struct {
	waiter *waiter
	id     int64
}

func (s *denseCallSlots) Len() int { return s.live }

func (s *denseCallSlots) Add(id ID, w *waiter) {
	n, ok := id.Number()
	if !ok {
		panic("jsonrpc2: dense call id is not numeric")
	}
	if n <= 0 {
		panic("jsonrpc2: dense call id must be positive")
	}
	if len(s.slots) == 0 {
		s.base = n
		s.slots = make([]denseCallSlot, initialOutgoingCallSlots)
	} else if s.live == 0 {
		s.base = n
	}
	if n < s.base {
		s.rebase(n)
	}
	if need := int(n - s.base + 1); need > len(s.slots) {
		s.grow(need)
	}
	idx := s.index(n)
	if s.slots[idx].waiter != nil {
		panic("jsonrpc2: duplicate dense call id")
	}
	s.slots[idx] = denseCallSlot{id: n, waiter: w}
	s.live++
}

func (s *denseCallSlots) Take(id ID) (*waiter, bool) {
	n, ok := id.Number()
	if !ok || s.live == 0 || len(s.slots) == 0 || n < s.base || int(n-s.base) >= len(s.slots) {
		return nil, false
	}
	idx := s.index(n)
	slot := &s.slots[idx]
	if slot.waiter == nil || slot.id != n {
		return nil, false
	}
	w := slot.waiter
	*slot = denseCallSlot{}
	s.live--
	if s.live == 0 {
		clear(s.slots)
		s.base = 0
		return w, true
	}
	if n == s.base {
		s.advanceBase()
	}
	return w, true
}

func (s *denseCallSlots) Drain(f func(ID, *waiter)) {
	if s.live == 0 {
		return
	}
	for i := range s.slots {
		if w := s.slots[i].waiter; w != nil {
			f(NewNumberID(s.slots[i].id), w)
			s.slots[i] = denseCallSlot{}
		}
	}
	s.live = 0
	s.base = 0
}

func (s *denseCallSlots) index(id int64) int {
	return int(uint64(id) & uint64(len(s.slots)-1))
}

func (s *denseCallSlots) advanceBase() {
	for s.live > 0 {
		idx := s.index(s.base)
		if s.slots[idx].waiter != nil && s.slots[idx].id == s.base {
			return
		}
		s.base++
	}
}

func (s *denseCallSlots) rebase(base int64) {
	if s.live == 0 {
		s.base = base
		return
	}
	maxID := base
	for i := range s.slots {
		if s.slots[i].waiter != nil && s.slots[i].id > maxID {
			maxID = s.slots[i].id
		}
	}
	if need := int(maxID - base + 1); need > len(s.slots) {
		s.grow(need)
	}
	s.base = base
}

func (s *denseCallSlots) grow(need int) {
	size := len(s.slots)
	for size < need {
		size *= 2
	}
	old := s.slots
	s.slots = make([]denseCallSlot, size)
	for i := range old {
		if old[i].waiter != nil {
			s.slots[s.index(old[i].id)] = old[i]
		}
	}
}
