// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"io"
	"sync"
)

// frameBuf is a pooled channel-stream frame. The pointer-to-struct shape keeps
// sync.Pool round trips allocation-free: the same *frameBuf box travels from
// the composing writer through the data channel to the reader and back to the
// pool, so steady-state traffic recycles both the box and its byte array.
type frameBuf struct {
	b []byte
}

// frameBufPool recycles channel-stream frames across all stream pairs.
// Ownership is exclusive at every instant: a frameBuf is held by exactly one
// of the pool, a composing writer, a data channel, or the delivered reader.
var frameBufPool = sync.Pool{
	New: func() any {
		return &frameBuf{b: make([]byte, 0, encodeBufInitCap)}
	},
}

// getFrameBuf returns a frame with a reset, possibly recycled byte array.
func getFrameBuf() *frameBuf {
	fb := frameBufPool.Get().(*frameBuf)
	fb.b = fb.b[:0]
	return fb
}

// putFrameBuf returns fb to the pool unless its array has grown beyond
// encodeBufMaxCap, in which case the frame is dropped so the oversized array
// can be collected (mirroring putEncodeBuf's policy).
func putFrameBuf(fb *frameBuf) {
	if cap(fb.b) > encodeBufMaxCap {
		return
	}
	frameBufPool.Put(fb)
}

// NewChannelStreamPair returns two connected in-memory streams backed by
// bounded channels of encoded JSON-RPC frames.
//
// The capacity controls the number of frames each direction can buffer before a
// writer blocks. A capacity of zero gives rendezvous semantics. Negative
// capacities panic, matching make(chan T, capacity).
//
// This stream pair is for same-process peers that need an in-memory JSON-RPC
// transport; it is not a replacement for network or stdio transports.
//
// Frames sent with WriteFrame are copied before they are queued, so the caller
// may reuse or mutate its buffer immediately after WriteFrame returns. A frame
// returned by ReadFrame is valid only until the next read on the same stream:
// delivered frames are recycled through a frame pool, which is what lets
// steady-state traffic queue frames without copying or allocating. Each stream
// supports one reader at a time, matching the package's connection read loop.
// Closing either stream closes the pair, makes later reads and writes fail
// with io.EOF, and unblocks pending reads and writes with io.EOF.
func NewChannelStreamPair(capacity int) (left, right Stream) {
	if capacity < 0 {
		panic("jsonrpc2: negative channel stream capacity")
	}
	p := &channelStreamPair{
		done: make(chan struct{}),
		aToB: make(chan *frameBuf, capacity),
		bToA: make(chan *frameBuf, capacity),
	}
	return &channelStream{in: p.bToA, out: p.aToB, pair: p}, &channelStream{in: p.aToB, out: p.bToA, pair: p}
}

type channelStreamPair struct {
	done chan struct{}
	aToB chan *frameBuf
	bToA chan *frameBuf
	once sync.Once
}

type channelStream struct {
	in   <-chan *frameBuf
	out  chan<- *frameBuf
	pair *channelStreamPair

	// lastBuf is the frame most recently delivered by ReadFrame. The next
	// ReadFrame recycles it, which is what bounds a delivered frame's validity
	// to "until the next read". It is only touched by the stream's single
	// reader goroutine.
	lastBuf *frameBuf

	// writeMu serializes concurrent senders in front of the data channel.
	// Funneling senders through a mutex keeps them off the multi-channel
	// select: without it, every blocked sender parks on (and is woken by) all
	// three select cases, and the select-lock contention measurably regresses
	// the parallel round-trip rows.
	writeMu sync.Mutex
}

// Read implements [Stream]. A decoded request's method and params borrow the
// delivered frame, which is recycled at the next read; copy what you retain.
func (s *channelStream) Read(ctx context.Context) (Message, int64, error) {
	frame, n, err := s.ReadFrame(ctx)
	if err != nil {
		return nil, 0, err
	}
	msg, derr := DecodeMessage(frame)
	if derr != nil {
		return nil, 0, derr
	}
	return msg, n, nil
}

func (s *channelStream) Write(ctx context.Context, msg Message) (int64, error) {
	frame, err := EncodeMessage(msg)
	if err != nil {
		return 0, err
	}
	// Adopt the encoded frame's array rather than copying it; the recycled
	// array the frame carried is dropped for the collector.
	fb := getFrameBuf()
	fb.b = frame
	n, serr := s.sendFrame(ctx, fb)
	if serr != nil {
		putFrameBuf(fb)
	}
	return n, serr
}

func (s *channelStream) ReadFrame(ctx context.Context) (frame []byte, n int64, err error) {
	if last := s.lastBuf; last != nil {
		s.lastBuf = nil
		putFrameBuf(last)
	}
	if err := s.closedOrCanceled(ctx); err != nil {
		return nil, 0, err
	}
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	case <-s.pair.done:
		return nil, 0, io.EOF
	case fb := <-s.in:
		s.lastBuf = fb
		return fb.b, int64(len(fb.b)), nil
	}
}

func (s *channelStream) WriteFrame(ctx context.Context, data []byte) (int64, error) {
	// The WriteFrame contract lets the caller reuse its buffer immediately, so
	// the data is copied -- but into a recycled frame, not a fresh allocation.
	fb := getFrameBuf()
	fb.b = append(fb.b, data...)
	n, err := s.sendFrame(ctx, fb)
	if err != nil {
		putFrameBuf(fb)
	}
	return n, err
}

func (s *channelStream) writeCall(ctx context.Context, id ID, method string, params RawMessage) (int64, error) {
	return s.composeAndSend(ctx, func(buf []byte) []byte {
		return appendCallFields(buf, id, method, params)
	})
}

func (s *channelStream) writeNotification(ctx context.Context, method string, params RawMessage) (int64, error) {
	return s.composeAndSend(ctx, func(buf []byte) []byte {
		return appendNotificationFields(buf, method, params)
	})
}

func (s *channelStream) writeResponse(ctx context.Context, id ID, result RawMessage, err error) (int64, error) {
	return s.composeAndSend(ctx, func(buf []byte) []byte {
		return appendResponseFields(buf, id, result, err)
	})
}

// composeAndSend appends one frame into a pooled buffer and hands the buffer's
// ownership to the receiving end through the data channel. No copy is queued:
// the receiver recycles the frame when it is retired by the next read.
func (s *channelStream) composeAndSend(ctx context.Context, appendFrame func([]byte) []byte) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	fb := getFrameBuf()
	fb.b = appendFrame(fb.b)
	n, err := s.sendFrame(ctx, fb)
	if err != nil {
		putFrameBuf(fb)
	}
	return n, err
}

func (s *channelStream) sendFrame(ctx context.Context, fb *frameBuf) (int64, error) {
	if err := s.closedOrCanceled(ctx); err != nil {
		return 0, err
	}
	// The frame's ownership transfers the instant the channel send succeeds; a
	// fast receiver can retire and recycle it before this function returns, so
	// the reported length must be read before the send.
	n := int64(len(fb.b))
	// Holding writeMu across a blocking send is safe: the receiving end never
	// takes this mutex, and Close (or cancellation) still unblocks the send
	// through the selected done channels, after which the mutex is released.
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-s.pair.done:
		return 0, io.EOF
	case s.out <- fb:
		return n, nil
	}
}

func (s *channelStream) closedOrCanceled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.pair.done:
		return io.EOF
	default:
		return nil
	}
}

func (s *channelStream) Close() error {
	s.pair.once.Do(func() { close(s.pair.done) })
	return nil
}
