// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

// PipelineClient is the pipelined client mode.
//
// It supports concurrent client-driven calls and notifications, but it does not
// dispatch server-initiated requests. The response reader is client-only: for
// frame-capable streams it scans borrowed response views and unmarshals the
// result directly into the waiting call's destination before the next frame can
// invalidate the borrowed bytes.
type PipelineClient struct {
	stream Stream
	frames frameStream
	fw     frameWriter
	codec  Codec

	readErr  error
	writeErr error
	closeErr error
	done     chan struct{}

	writeQueue []pipelineQueuedCall
	calls      densePipelineCallSlots

	seq atomic.Int64 // last allocated outgoing call id

	mu sync.Mutex

	writeQueueMu sync.Mutex
	reading      bool
	closing      bool
	doneClosed   bool
	writeRunning bool
}

// NewPipelineClient creates a pipelined client over stream.
func NewPipelineClient(stream Stream, opts ...Option) *PipelineClient {
	probe := &conn{codec: DefaultCodec}
	for _, opt := range opts {
		opt(probe)
	}
	frames, _ := stream.(frameStream)
	fw, _ := stream.(frameWriter)
	return &PipelineClient{
		stream: stream,
		frames: frames,
		fw:     fw,
		codec:  probe.codec,
		done:   make(chan struct{}),
	}
}

var _ Conn = (*PipelineClient)(nil)

// Call implements [Conn].
func (c *PipelineClient) Call(ctx context.Context, method string, params, result any) (ID, error) {
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return ID{}, fmt.Errorf("jsonrpc2: marshaling call parameters: %w", err)
	}

	idNum := c.seq.Add(1)
	id := NewNumberID(idNum)
	w := getPipelineWaiter(result)

	c.mu.Lock()
	queueWrite := false
	if shutErr := c.shuttingDownLocked(); shutErr != nil {
		c.mu.Unlock()
		putPipelineWaiter(w)
		return id, shutErr
	}
	c.calls.Add(id, w)
	queueWrite = c.frames != nil && c.calls.Len() > 1
	c.mu.Unlock()

	if queueWrite {
		c.enqueueCall(pipelineQueuedCall{ctx: ctx, id: idNum, method: method, params: raw})
	} else if err := c.writeCall(ctx, id, method, raw); err != nil {
		c.afterWrite(ctx, err)
		if c.retireCall(id) {
			putPipelineWaiter(w)
			return id, err
		}
		<-w.ready
		putPipelineWaiter(w)
		return id, err
	}

	select {
	case <-w.ready:
		err := w.err
		putPipelineWaiter(w)
		if err != nil {
			return id, err
		}
		return id, nil
	case <-ctx.Done():
		if c.retireCall(id) {
			putPipelineWaiter(w)
			return id, ctx.Err()
		}
		<-w.ready
		putPipelineWaiter(w)
		return id, ctx.Err()
	}
}

// Notify implements [Conn].
func (c *PipelineClient) Notify(ctx context.Context, method string, params any) error {
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return fmt.Errorf("jsonrpc2: marshaling notify parameters: %w", err)
	}

	c.mu.Lock()
	shutErr := c.shuttingDownLocked()
	c.mu.Unlock()
	if shutErr != nil {
		return shutErr
	}

	err = c.writeNotification(ctx, method, raw)
	c.afterWrite(ctx, err)
	return err
}

type pipelineQueuedCall struct {
	ctx    context.Context
	method string
	params RawMessage
	id     int64
}

func (c *PipelineClient) enqueueCall(call pipelineQueuedCall) {
	c.writeQueueMu.Lock()
	c.writeQueue = append(c.writeQueue, call)
	start := !c.writeRunning
	if start {
		c.writeRunning = true
	}
	c.writeQueueMu.Unlock()
	if start {
		go c.writeQueuedCalls()
	}
}

func (c *PipelineClient) writeQueuedCalls() {
	// A just-started writer yields once so a burst of concurrent Call goroutines
	// can enqueue behind the first direct writer and be drained by one writer
	// goroutine instead of all contending on the stream write mutex. In the
	// single-call path Call writes directly and never reaches this goroutine.
	runtime.Gosched()

	var spare []pipelineQueuedCall
	for {
		c.writeQueueMu.Lock()
		calls := c.writeQueue
		if len(calls) == 0 {
			if cap(spare) > cap(c.writeQueue) {
				c.writeQueue = spare
			}
			c.writeRunning = false
			c.writeQueueMu.Unlock()
			return
		}
		c.writeQueue = spare[:0]
		c.writeQueueMu.Unlock()

		if err := c.writeQueuedCallFrames(context.Background(), calls); err != nil {
			c.writeQueueMu.Lock()
			c.writeQueue = c.writeQueue[:0]
			c.writeRunning = false
			c.writeQueueMu.Unlock()
			c.afterWrite(context.Background(), err)
			return
		}
		spare = calls[:0]
	}
}

func (c *PipelineClient) writeQueuedCallFrames(ctx context.Context, calls []pipelineQueuedCall) error {
	for _, call := range calls {
		writeCtx := ctx
		if call.ctx != nil {
			writeCtx = call.ctx
		}
		if err := c.writeCall(writeCtx, NewNumberID(call.id), call.method, call.params); err != nil {
			if writeCtx.Err() != nil {
				if w := c.retireNumberCall(call.id); w != nil {
					w.deliver(writeCtx.Err())
				}
				continue
			}
			return err
		}
	}
	return nil
}

// Go implements [Conn]. PipelineClient is client-only, so the handler is
// ignored and the read loop only correlates responses.
func (c *PipelineClient) Go(ctx context.Context, _ Handler) {
	c.mu.Lock()
	switch {
	case c.reading:
		c.mu.Unlock()
		panic("jsonrpc2: PipelineClient.Go called more than once")
	case c.doneClosed:
		c.mu.Unlock()
		return
	case c.closing:
		c.closeDoneLocked()
		c.mu.Unlock()
		return
	default:
		c.reading = true
		c.mu.Unlock()
	}
	go c.readResponses(ctx)
}

// Close implements [Conn].
func (c *PipelineClient) Close() error {
	c.mu.Lock()
	if c.doneClosed {
		err := c.closeErr
		c.mu.Unlock()
		return err
	}
	shouldClose := !c.closing
	c.closing = true
	c.drainCallsLocked(ErrClientClosing)
	if !c.reading {
		c.closeDoneLocked()
	}
	done := c.done
	c.mu.Unlock()

	if shouldClose {
		c.recordCloseError(c.stream.Close())
	}
	<-done
	return c.closeReturnError()
}

// Done implements [Conn].
func (c *PipelineClient) Done() <-chan struct{} { return c.done }

// Err implements [Conn].
func (c *PipelineClient) Err() error {
	err := c.terminationError()
	switch {
	case err == nil,
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrClosedPipe),
		errors.Is(err, os.ErrClosed),
		errors.Is(err, net.ErrClosed),
		errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, ErrServerClosing),
		errors.Is(err, ErrClientClosing):
		return nil
	default:
		return err
	}
}

func (c *PipelineClient) writeCall(ctx context.Context, id ID, method string, params RawMessage) error {
	if c.fw != nil {
		_, err := c.fw.writeCall(ctx, id, method, params)
		return err
	}
	if c.frames != nil {
		_, err := c.frames.WriteFrame(ctx, appendCallFields(nil, id, method, params))
		return err
	}
	_, err := c.stream.Write(ctx, callWire{id: id, method: method, params: params})
	return err
}

func (c *PipelineClient) writeNotification(ctx context.Context, method string, params RawMessage) error {
	if c.fw != nil {
		_, err := c.fw.writeNotification(ctx, method, params)
		return err
	}
	if c.frames != nil {
		_, err := c.frames.WriteFrame(ctx, appendNotificationFields(nil, method, params))
		return err
	}
	_, err := c.stream.Write(ctx, notificationWire{method: method, params: params})
	return err
}

func (c *PipelineClient) readResponses(ctx context.Context) {
	var err error
	for {
		if err = c.readResponse(ctx); err != nil {
			break
		}
	}
	c.finishRead(err)
}

func (c *PipelineClient) readResponse(ctx context.Context) error {
	if c.frames == nil {
		msg, _, err := c.stream.Read(ctx)
		if err != nil {
			return err
		}
		resp, ok := msg.(*Response)
		if !ok {
			return fmt.Errorf("jsonrpc2: pipeline client received non-response %T", msg)
		}
		c.deliverResponse(resp.id, resp.result, resp.err)
		return nil
	}

	frame, _, err := c.frames.ReadFrame(ctx)
	if err != nil {
		return err
	}
	if isJSONArray(frame) {
		return c.deliverBatchResponse(frame)
	}
	if id, result, ok, _ := scanPipelineResultResponseNumber(frame); ok {
		c.deliverNumberResponse(id, result, nil)
		return nil
	}
	view, err := ScanMessageView(frame)
	if err != nil {
		return fmt.Errorf("jsonrpc2: decoding response: %w", err)
	}
	switch view.Kind {
	case MessageViewResponseResult:
		id, ok := view.ID.ID()
		if !ok {
			return ErrInvalidRequest
		}
		c.deliverResponse(id, RawMessage(view.Result), nil)
		return nil
	case MessageViewResponseError:
		id, ok := view.ID.ID()
		if !ok {
			return ErrInvalidRequest
		}
		respErr, ok := view.Error.Owned()
		if !ok {
			return ErrInvalidRequest
		}
		c.deliverResponse(id, nil, respErr)
		return nil
	default:
		return fmt.Errorf("jsonrpc2: pipeline client received non-response %s", view.Kind)
	}
}

const pipelineResultResponsePrefix = `{"jsonrpc":"2.0","id":`

func isJSONArray(frame []byte) bool {
	i := skipSpace(frame, 0)
	return i < len(frame) && frame[i] == '['
}

func scanPipelineResultResponseNumber(frame []byte) (id int64, result RawMessage, ok bool, err error) {
	i := skipSpace(frame, 0)
	id, result, next, ok, err := scanPipelineResultResponseNumberAt(frame, i)
	if !ok || err != nil {
		return 0, result, ok, err
	}
	if skipSpace(frame, next) != len(frame) {
		return 0, nil, false, ErrInvalidRequest
	}
	return id, result, true, nil
}

func scanPipelineResultResponseNumberAt(frame []byte, i int) (id int64, result RawMessage, next int, ok bool, err error) {
	if !hasLiteralAt(frame, i, pipelineResultResponsePrefix) {
		return 0, nil, i, false, nil
	}
	i += len(pipelineResultResponsePrefix)

	n, next, ok := parsePositiveInt64(frame, i)
	if !ok {
		return 0, nil, i, false, nil
	}
	i = next
	if !hasLiteralAt(frame, i, `,"result":`) {
		return 0, nil, i, false, nil
	}
	i += len(`,"result":`)

	valStart := i
	if hasLiteralAt(frame, i, "null") {
		i += len("null")
		i = skipSpace(frame, i)
		if i >= len(frame) || frame[i] != '}' {
			return 0, nil, i, false, ErrInvalidRequest
		}
		return n, RawMessage(frame[valStart : valStart+len("null")]), i + 1, true, nil
	}

	valEnd, ok := scanValue(frame, i)
	if !ok {
		return 0, nil, i, false, ErrParse
	}
	i = skipSpace(frame, valEnd)
	if i >= len(frame) || frame[i] != '}' {
		return 0, nil, i, false, ErrInvalidRequest
	}
	return n, RawMessage(frame[valStart:valEnd]), i + 1, true, nil
}

func (c *PipelineClient) deliverBatchResponse(frame []byte) error {
	if ok, err := c.deliverFastBatchResponse(frame); ok || err != nil {
		return err
	}

	i := skipSpace(frame, 0)
	spans, ok := scanArrayElements(frame, i)
	if !ok {
		return ErrInvalidRequest
	}
	for _, span := range spans {
		msg, err := DecodeMessage(span)
		if err != nil {
			return fmt.Errorf("jsonrpc2: decoding batch response: %w", err)
		}
		resp, ok := msg.(*Response)
		if !ok {
			return fmt.Errorf("jsonrpc2: pipeline client received non-response %T in batch", msg)
		}
		c.deliverResponse(resp.id, resp.result, resp.err)
	}
	return nil
}

func (c *PipelineClient) deliverFastBatchResponse(frame []byte) (ok bool, err error) {
	i := skipSpace(frame, 0)
	if i >= len(frame) || frame[i] != '[' {
		return false, nil
	}
	i = skipSpace(frame, i+1)
	if i < len(frame) && frame[i] == ']' {
		if skipSpace(frame, i+1) != len(frame) {
			return false, ErrInvalidRequest
		}
		return true, ErrInvalidRequest
	}
	var results []pipelineNumberResult
	for {
		id, result, next, ok, err := scanPipelineResultResponseNumberAt(frame, i)
		if !ok || err != nil {
			return false, nil
		}
		results = append(results, pipelineNumberResult{id: id, result: result})
		i = skipSpace(frame, next)
		if i >= len(frame) {
			return false, nil
		}
		switch frame[i] {
		case ',':
			i = skipSpace(frame, i+1)
			continue
		case ']':
			if skipSpace(frame, i+1) != len(frame) {
				return false, ErrInvalidRequest
			}
			for _, r := range results {
				c.deliverNumberResponse(r.id, r.result, nil)
			}
			return true, nil
		default:
			return false, nil
		}
	}
}

type pipelineNumberResult struct {
	result RawMessage
	id     int64
}

func hasLiteralAt(data []byte, i int, lit string) bool {
	if i < 0 || len(data)-i < len(lit) {
		return false
	}
	for j := range lit {
		if data[i+j] != lit[j] {
			return false
		}
	}
	return true
}

func parsePositiveInt64(data []byte, i int) (n int64, next int, ok bool) {
	if i >= len(data) || data[i] < '1' || data[i] > '9' {
		return 0, i, false
	}
	const maxInt64 = int64(1<<63 - 1)
	for i < len(data) {
		c := data[i]
		if c < '0' || c > '9' {
			break
		}
		digit := int64(c - '0')
		if n > (maxInt64-digit)/10 {
			return 0, i, false
		}
		n = n*10 + digit
		i++
	}
	return n, i, true
}

func (c *PipelineClient) deliverResponse(id ID, result RawMessage, respErr error) {
	w := c.takeCall(id)
	if w == nil {
		return
	}
	c.deliverWaiter(w, result, respErr)
}

func (c *PipelineClient) deliverNumberResponse(id int64, result RawMessage, respErr error) {
	w := c.takeNumberCall(id)
	if w == nil {
		return
	}
	c.deliverWaiter(w, result, respErr)
}

func (c *PipelineClient) deliverWaiter(w *pipelineWaiter, result RawMessage, respErr error) {
	if respErr != nil {
		w.deliver(respErr)
		return
	}
	if w.result == nil {
		w.deliver(nil)
		return
	}
	if err := unmarshalResult(c.codec, result, w.result); err != nil {
		w.deliver(fmt.Errorf("jsonrpc2: unmarshaling result: %w", err))
		return
	}
	w.deliver(nil)
}

func (c *PipelineClient) retireCall(id ID) bool {
	c.mu.Lock()
	_, ok := c.calls.Take(id)
	c.mu.Unlock()
	return ok
}

func (c *PipelineClient) retireNumberCall(id int64) *pipelineWaiter {
	c.mu.Lock()
	w, _ := c.calls.TakeNumber(id)
	c.mu.Unlock()
	return w
}

func (c *PipelineClient) takeCall(id ID) *pipelineWaiter {
	c.mu.Lock()
	w, _ := c.calls.Take(id)
	c.mu.Unlock()
	return w
}

func (c *PipelineClient) takeNumberCall(id int64) *pipelineWaiter {
	c.mu.Lock()
	w, _ := c.calls.TakeNumber(id)
	c.mu.Unlock()
	return w
}

func (c *PipelineClient) afterWrite(ctx context.Context, err error) {
	if err == nil || ctx.Err() != nil {
		return
	}
	c.fail(err)
}

func (c *PipelineClient) fail(err error) {
	c.mu.Lock()
	if c.writeErr == nil {
		c.writeErr = err
	}
	shouldClose := !c.closing
	c.closing = true
	c.drainCallsLocked(err)
	closeDone := !c.reading
	c.mu.Unlock()

	if shouldClose {
		c.recordCloseError(c.stream.Close())
	}
	if closeDone {
		c.mu.Lock()
		c.closeDoneLocked()
		c.mu.Unlock()
	}
}

func (c *PipelineClient) finishRead(err error) {
	c.mu.Lock()
	c.reading = false
	if c.readErr == nil {
		c.readErr = err
	}
	shouldClose := !c.closing
	c.closing = true
	c.drainCallsLocked(err)
	c.mu.Unlock()

	if shouldClose {
		c.recordCloseError(c.stream.Close())
	}

	c.mu.Lock()
	c.closeDoneLocked()
	c.mu.Unlock()
}

func (c *PipelineClient) shuttingDownLocked() error {
	switch {
	case c.closing:
		return ErrClientClosing
	case c.readErr != nil:
		return fmt.Errorf("%w: %w", ErrClientClosing, c.readErr)
	case c.writeErr != nil:
		return fmt.Errorf("%w: %w", ErrClientClosing, c.writeErr)
	case c.doneClosed:
		return ErrClientClosing
	default:
		return nil
	}
}

func (c *PipelineClient) drainCallsLocked(err error) {
	c.calls.Drain(func(_ ID, w *pipelineWaiter) {
		w.deliver(err)
	})
}

func (c *PipelineClient) closeDoneLocked() {
	if !c.doneClosed {
		close(c.done)
		c.doneClosed = true
	}
}

func (c *PipelineClient) recordCloseError(err error) {
	c.mu.Lock()
	if c.closeErr == nil {
		c.closeErr = err
	}
	c.mu.Unlock()
}

func (c *PipelineClient) closeReturnError() error {
	c.mu.Lock()
	err := c.closeErr
	c.mu.Unlock()
	return err
}

func (c *PipelineClient) terminationError() error {
	c.mu.Lock()
	err := c.terminationErrorLocked()
	c.mu.Unlock()
	return err
}

func (c *PipelineClient) terminationErrorLocked() error {
	switch {
	case c.writeErr != nil:
		return c.writeErr
	case c.readErr != nil && !errors.Is(c.readErr, io.EOF):
		return c.readErr
	default:
		return c.closeErr
	}
}

type pipelineWaiter struct {
	ready  chan struct{}
	result any
	err    error
}

func (w *pipelineWaiter) deliver(err error) {
	w.err = err
	w.ready <- struct{}{}
}

var pipelineWaiterPool = sync.Pool{
	New: func() any {
		return &pipelineWaiter{ready: make(chan struct{}, 1)}
	},
}

func getPipelineWaiter(result any) *pipelineWaiter {
	w := pipelineWaiterPool.Get().(*pipelineWaiter)
	w.result = result
	return w
}

func putPipelineWaiter(w *pipelineWaiter) {
	select {
	case <-w.ready:
	default:
	}
	w.result = nil
	w.err = nil
	pipelineWaiterPool.Put(w)
}

type densePipelineCallSlots struct {
	slots []densePipelineCallSlot
	base  int64
	live  int
}

type densePipelineCallSlot struct {
	waiter *pipelineWaiter
	id     int64
}

func (s *densePipelineCallSlots) Len() int { return s.live }

func (s *densePipelineCallSlots) Add(id ID, w *pipelineWaiter) {
	n, ok := id.Number()
	if !ok {
		panic("jsonrpc2: dense pipeline id is not numeric")
	}
	if n <= 0 {
		panic("jsonrpc2: dense pipeline id must be positive")
	}
	if len(s.slots) == 0 {
		s.base = n
		s.slots = make([]densePipelineCallSlot, initialOutgoingCallSlots)
	} else if s.live == 0 {
		s.base = n
	}
	if n < s.base {
		s.rebase(n)
	}
	if need := int(n - s.base + 1); need > len(s.slots) {
		s.compactBase()
		if need = int(n - s.base + 1); need > len(s.slots) {
			s.grow(need)
		}
	}
	idx := s.index(n)
	if s.slots[idx].waiter != nil {
		panic("jsonrpc2: duplicate dense pipeline id")
	}
	s.slots[idx] = densePipelineCallSlot{id: n, waiter: w}
	s.live++
}

func (s *densePipelineCallSlots) Take(id ID) (*pipelineWaiter, bool) {
	n, ok := id.Number()
	if !ok {
		return nil, false
	}
	return s.TakeNumber(n)
}

func (s *densePipelineCallSlots) TakeNumber(n int64) (*pipelineWaiter, bool) {
	if s.live == 0 || len(s.slots) == 0 || n < s.base || int(n-s.base) >= len(s.slots) {
		return nil, false
	}
	idx := s.index(n)
	slot := &s.slots[idx]
	if slot.waiter == nil || slot.id != n {
		return nil, false
	}
	w := slot.waiter
	*slot = densePipelineCallSlot{}
	s.live--
	if s.live == 0 {
		clear(s.slots)
		s.base = 0
		return w, true
	}
	return w, true
}

func (s *densePipelineCallSlots) Drain(f func(ID, *pipelineWaiter)) {
	if s.live == 0 {
		return
	}
	for i := range s.slots {
		if w := s.slots[i].waiter; w != nil {
			f(NewNumberID(s.slots[i].id), w)
			s.slots[i] = densePipelineCallSlot{}
		}
	}
	s.live = 0
	s.base = 0
}

func (s *densePipelineCallSlots) index(id int64) int {
	return int(uint64(id) & uint64(len(s.slots)-1))
}

func (s *densePipelineCallSlots) rebase(base int64) {
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

func (s *densePipelineCallSlots) compactBase() {
	if s.live == 0 {
		s.base = 0
		return
	}
	base := int64(1<<63 - 1)
	for i := range s.slots {
		if slot := &s.slots[i]; slot.waiter != nil && slot.id < base {
			base = slot.id
		}
	}
	s.base = base
}

func (s *densePipelineCallSlots) grow(need int) {
	size := len(s.slots)
	for size < need {
		size *= 2
	}
	old := s.slots
	s.slots = make([]densePipelineCallSlot, size)
	for i := range old {
		if old[i].waiter != nil {
			s.slots[s.index(old[i].id)] = old[i]
		}
	}
}
