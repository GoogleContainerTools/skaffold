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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Conn is the common interface to JSON-RPC clients and servers.
//
// Conn is bidirectional: it has no designated server or client end. It manages
// the JSON-RPC 2.0 protocol on top of a [Stream], correlating responses back to
// the calls that produced them and dispatching incoming requests to a
// [Handler].
//
// A Conn is created with [NewConn] and driven by a single read goroutine that
// is started by [Conn.Go]. The same Conn may be used concurrently for outgoing
// [Conn.Call] and [Conn.Notify] from many goroutines.
type Conn interface {
	// Call invokes method on the peer and waits for the response.
	//
	// params is marshaled with the connection's [Codec] before being sent; a nil
	// params sends no parameters. The response result is unmarshaled into result,
	// which may be nil to discard it. The returned [ID] is unique to this
	// connection and is the id under which the call was sent.
	//
	// Call returns the error response sent by the peer, the local marshaling or
	// write error, or ctx.Err() if ctx is canceled before the response arrives.
	Call(ctx context.Context, method string, params, result any) (ID, error)

	// Notify invokes method on the peer without waiting for a response.
	//
	// params is marshaled with the connection's [Codec]; a nil params sends no
	// parameters.
	Notify(ctx context.Context, method string, params any) error

	// Go starts the connection's read goroutine, dispatching incoming requests
	// to handler. It must be called exactly once per Conn and returns
	// immediately; block on [Conn.Done] to wait for the connection to
	// terminate.
	//
	// Dispatch is direct-return: handler's return values become the response,
	// no per-request reply machinery is allocated, and the request is a pooled
	// concrete value whose method and params are borrowed from the transport
	// frame, valid until the handler returns (see [Handler] for the lifetime
	// contract and [Request.Clone] for retention).
	//
	// By default a request handler runs inline on the read goroutine, so
	// handlers observe requests in wire order and the next message is not read
	// until the current handler returns. A handler that wants to overlap later
	// requests (for example a long-running call, or a server that issues calls
	// back to the peer) must release itself with [Async] or be wrapped with
	// [AsyncHandler]; otherwise a handler that issues a server-initiated
	// [Conn.Call] back into this same connection deadlocks the read goroutine,
	// because the response to that call cannot be read while the read goroutine
	// is blocked running the handler. A server-initiated [Conn.Notify] needs no
	// release, since it never waits for a response. See [Handler] for the full
	// reentrancy contract.
	//
	// [Conn.Close] is the authoritative teardown. Canceling ctx is observed only
	// between frames: the read loop checks ctx before it starts the next frame, so
	// a cancellation requests a graceful stop at the next frame boundary, but a
	// reader already blocked mid-frame (waiting on the peer) is not interrupted by
	// ctx-cancel. Close, by contrast, closes the underlying stream and so unblocks
	// a reader parked in the middle of a frame. To guarantee prompt termination,
	// call Close rather than relying on ctx cancellation. A termination caused by
	// canceling ctx is treated as a clean shutdown and is not reported by
	// [Conn.Err].
	Go(ctx context.Context, handler Handler)

	// Close stops accepting new work, waits for in-flight calls and handlers to
	// drain, closes the underlying stream, and blocks until the connection has
	// fully terminated (the read goroutine has exited). It reports the stream's
	// close error. After Close returns, [Conn.Done] is already closed.
	Close() error

	// Done returns a channel that is closed when the connection has fully
	// terminated: the read goroutine has exited and all in-flight work has
	// drained.
	Done() <-chan struct{}

	// Err reports the error that terminated the connection, or nil if it was
	// closed cleanly. A clean end of stream ([io.EOF]) is reported as nil.
	Err() error
}

// Option configures a [Conn] created by [NewConn].
type Option func(*conn)

// WithCodec sets the [Codec] used to marshal call params and unmarshal call
// results on the connection. When unset the connection uses [DefaultCodec].
func WithCodec(c Codec) Option {
	return func(cn *conn) {
		if c != nil {
			cn.codec = c
		}
	}
}

// WithPreempter sets the [Preempter] consulted for every incoming request before
// it is dispatched to the handler. The preempter runs inline on the read
// goroutine; a request it handles (by returning a result and an error other than
// [ErrNotHandled]) is answered immediately and never reaches the handler.
func WithPreempter(p Preempter) Option {
	return func(cn *conn) {
		cn.preempter = p
	}
}

// NewConn creates a connection over stream. The connection does not start
// reading until [Conn.Go] is called.
func NewConn(stream Stream, opts ...Option) Conn {
	c := &conn{
		stream: stream,
		codec:  DefaultCodec,
		done:   make(chan struct{}),
	}
	// The stream is fixed for the connection's lifetime, so resolve the optional
	// concrete-write extension once here instead of type-asserting on every send.
	c.fw, _ = stream.(frameWriter)
	c.state.closer = stream
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// conn is the concrete [Conn]. Its in-flight state is funneled through
// updateInFlight under a single mutex, so that idle detection, shutdown, and
// write-error propagation observe a consistent view.
type conn struct {
	state inFlightState // mutated only inside updateInFlight

	stream    Stream
	fw        frameWriter // stream's concrete-write extension, resolved once; nil if unsupported
	codec     Codec
	preempter Preempter // optional, consulted before the handler

	// handler dispatches every incoming request. It is set once in Go, before
	// the read goroutine starts, and is nil only on client-only connections
	// that never receive requests.
	handler Handler

	done chan struct{} // closed when the connection is fully terminated
	seq  atomic.Int64  // last allocated outgoing call id

	stateMu sync.Mutex
}

// inFlightState records the in-flight calls and requests of a [conn]. It is the
// single source of truth for idle detection and shutdown and is mutated only
// inside conn.updateInFlight while holding conn.stateMu.
type inFlightState struct {
	readErr  error // set when the read goroutine exits
	writeErr error // set when a write fails for a non-canceled reason

	// closer shuts down the stream. It is invoked once, when the connection is
	// idle and shutting down, then set to nil and its result recorded in closeErr.
	closer   io.Closer
	closeErr error

	incomingByID map[ID]*incomingRequest // outstanding incoming calls, keyed by id

	outgoingCalls denseCallSlots // outstanding generated numeric calls

	incoming    int  // incoming requests not yet fully processed
	connClosing bool // Close has been called
	reading     bool // the read goroutine is running
}

// idle reports whether no work is in flight. The read goroutine may still be
// running, but nothing else is operating on behalf of the connection.
func (s *inFlightState) idle() bool {
	return s.outgoingCalls.Len() == 0 && s.incoming == 0
}

// shuttingDown reports the error that should reject new work, or nil. It wraps
// errClosing with the read or write error when the stream itself has broken.
func (s *inFlightState) shuttingDown(errClosing error) error {
	if s.connClosing {
		return errClosing
	}
	if s.readErr != nil {
		return fmt.Errorf("%w: %w", errClosing, s.readErr)
	}
	if s.writeErr != nil {
		return fmt.Errorf("%w: %w", errClosing, s.writeErr)
	}
	return nil
}

// waiter is the rendezvous for one outgoing call. It is registered in the
// numeric outgoing call slots before the call is written and retired (exactly
// once) by whichever of the read goroutine or the canceling Call wins the
// lock-guarded removal. waiters are pooled; a waiter is only returned to the
// pool after it has been removed from the slots, so the peer can never deliver
// to a recycled waiter.
type waiter struct {
	result any
	err    error
	ready  chan struct{} // buffered, capacity 1

	// response is set for the general DecodeMessage path. resultReady is set for
	// the borrowed-frame fast path after the read goroutine has unmarshaled the
	// response result directly into result.
	response    *Response
	resultReady bool
}

// waiterPool recycles waiters to cut a per-call allocation.
var waiterPool = sync.Pool{
	New: func() any {
		return &waiter{ready: make(chan struct{}, 1)}
	},
}

func getWaiter(result any) *waiter {
	w := waiterPool.Get().(*waiter)
	w.result = result
	return w
}

func putWaiter(w *waiter) {
	// Drain a delivered-but-unread response so the channel is empty for reuse.
	select {
	case <-w.ready:
	default:
	}
	w.response = nil
	w.result = nil
	w.resultReady = false
	w.err = nil
	waiterPool.Put(w)
}

// incomingRequest tracks an incoming request while it is handled. It is itself
// the per-request [context.Context] passed to the handler: cancellation (by the
// read loop on completion, or by write-error propagation) reaches the handler
// through this value, and it carries the parent's deadline and values.
//
// Folding the request context into incomingRequest removes the per-request
// context.WithCancel allocation (the cancelCtx struct plus its CancelFunc
// closure) from the dispatch hot path. The real cancellation machinery is
// created lazily, only the first time Done is observed, by delegating to a
// stdlib context.WithCancel(parent); a handler that never inspects Done (the
// common void path) pays nothing. Deadline and Value delegate to the parent
// without locking so they stay allocation-free.
type incomingRequest struct {
	parent     context.Context
	realCtx    context.Context    // lazily created on first Done
	realCancel context.CancelFunc // cancels realCtx; nil until realCtx is created

	// request is the concrete request being dispatched. It is a value field so
	// the request body shares this struct's single allocation; the handler
	// receives &ir.request (an interior pointer, no extra alloc).
	request Request

	// rel and replied are value fields, not separate heap objects: folding them
	// into incomingRequest means one dispatch-path allocation (this struct) backs
	// the request context, the release token, and the replied flag together. The
	// releaser is reached via &ir.rel (an interior pointer, no extra alloc) for
	// the asyncKey{} value and the dispatch handoff; replied is reached via
	// &ir.replied. The merged struct stays GC-managed exactly as before, so a
	// handler may still retain the request context with no lifetime change.
	rel releaser
	id  ID

	mu      sync.Mutex
	replied repliedFlag

	isCall bool

	canceled bool // cancel was called before realCtx existed
}

// compile-time check that *incomingRequest satisfies context.Context.
var _ context.Context = (*incomingRequest)(nil)

// Deadline implements [context.Context] by delegating to the parent.
func (ir *incomingRequest) Deadline() (deadline time.Time, ok bool) {
	return ir.parent.Deadline()
}

// Value implements [context.Context]. It returns the request's [releaser] for
// the internal asyncKey so that [Async] can release the request without a
// separate context.WithValue wrapper allocation on the dispatch hot path; every
// other key delegates to the parent. It is kept lock-free because the
// synchronous dispatch path may read context values.
func (ir *incomingRequest) Value(key any) any {
	if _, ok := key.(asyncKey); ok && ir.rel.active {
		return &ir.rel
	}
	return ir.parent.Value(key)
}

// Done implements [context.Context]. The cancellation channel is created lazily
// the first time it is requested by delegating to a stdlib
// context.WithCancel(parent), so a handler that never selects on Done (the void
// hot path) never forces the allocation. A cancel that arrived before the first
// Done is replayed onto the freshly created context.
func (ir *incomingRequest) Done() <-chan struct{} {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	if ir.realCtx == nil {
		ir.realCtx, ir.realCancel = context.WithCancel(ir.parent)
		if ir.canceled {
			ir.realCancel()
		}
	}
	return ir.realCtx.Done()
}

// Err implements [context.Context]. When the cancellation channel has been
// materialized it reports the delegate's error; otherwise it reports
// context.Canceled if cancel was already called, and finally falls back to the
// parent's error so a parent deadline or cancellation is visible even when Done
// was never observed.
func (ir *incomingRequest) Err() error {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	switch {
	case ir.realCtx != nil:
		return ir.realCtx.Err()
	case ir.canceled:
		return context.Canceled
	default:
		return ir.parent.Err()
	}
}

// cloneRequestOwned replaces the request's borrowed method and params spans
// with owned copies, in place, so the request remains valid after the read
// loop reuses the transport frame. It runs at the hard-release (Async) escape
// point, on the releasing goroutine, strictly before the read loop can read
// the next frame.
func (ir *incomingRequest) cloneRequestOwned() {
	ir.request.method = strings.Clone(ir.request.method)
	ir.request.params = cloneBytes(ir.request.params)
}

// cancel cancels the request context. If the cancellation channel has not yet
// been materialized, the cancellation is recorded and replayed when Done first
// creates it. cancel is idempotent and safe for concurrent use; it is invoked by
// the read loop when the request completes and by write-error propagation.
func (ir *incomingRequest) cancel() {
	ir.mu.Lock()
	canceler := ir.realCancel
	ir.canceled = true
	ir.mu.Unlock()
	if canceler != nil {
		canceler()
	}
}

// updateInFlight runs f under stateMu to mutate the in-flight state, then closes
// the stream and the done channel if the connection has become idle while
// shutting down. All state mutation goes through here so that idle detection and
// shutdown observe a consistent view; f must not block (it may only do
// non-blocking channel operations) and must not re-enter updateInFlight.
func (c *conn) updateInFlight(f func(s *inFlightState)) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	s := &c.state
	f(s)

	select {
	case <-c.done:
		// Already fully done; f must not have re-introduced work.
		return
	default:
	}

	if s.idle() && s.shuttingDown(ErrServerClosing) != nil {
		if s.closer != nil {
			s.closeErr = s.closer.Close()
			s.closer = nil
		}
		if !s.reading {
			// The read goroutine has stopped (or never started) and nothing is in
			// flight: the connection is fully terminated.
			close(c.done)
		}
		// Otherwise the read goroutine is still blocked in Read; closing the stream
		// above unblocks it, and its exit will run updateInFlight again with
		// reading=false to close done.
	}
}

// Call implements [Conn].
func (c *conn) Call(ctx context.Context, method string, params, result any) (ID, error) {
	// Marshal before allocating the call id so a local marshal failure returns a
	// zero ID (no call was sent) and does not burn a sequence number — matching
	// SyncClient.Call.
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return ID{}, fmt.Errorf("jsonrpc2: marshaling call parameters: %w", err)
	}

	id := NewNumberID(c.seq.Add(1))

	w := getWaiter(result)

	var shutErr error
	c.updateInFlight(func(s *inFlightState) {
		if shutErr = s.shuttingDown(ErrClientClosing); shutErr != nil {
			return
		}
		s.outgoingCalls.Add(id, w)
	})
	if shutErr != nil {
		putWaiter(w)
		return id, shutErr
	}

	if err := c.writeCall(ctx, id, method, raw); err != nil {
		// The write failed, so the peer will never answer. If we win the removal we
		// own the waiter; otherwise the read goroutine retired it on a read-side
		// failure and is committed to delivering once, so wait before pooling.
		c.retireCallWaiter(id, w)
		return id, err
	}

	select {
	case <-w.ready:
		// The read goroutine delivered the response and removed the waiter from the
		// slot table, so we own it again.
		return id, c.finishCallWaiter(w, result)

	case <-ctx.Done():
		// The caller gave up. If we win the lock-guarded removal we own the waiter
		// and the read goroutine will never deliver to it. Otherwise the read
		// goroutine already removed it from the slots and is committed to delivering
		// exactly once, so wait for that delivery before pooling the waiter; pooling
		// it early would let a reused waiter receive this call's late response.
		c.retireCallWaiter(id, w)
		return id, ctx.Err()
	}
}

// finishCallWaiter returns the delivered call outcome and returns w to the pool.
func (c *conn) finishCallWaiter(w *waiter, result any) error {
	resp, waitErr, resultReady := w.response, w.err, w.resultReady
	putWaiter(w)
	if waitErr != nil {
		return waitErr
	}
	if resp == nil {
		return nil
	}
	if resp.err != nil {
		return resp.err
	}
	if resultReady {
		return nil
	}
	if err := unmarshalResult(c.codec, resp.result, result); err != nil {
		return fmt.Errorf("jsonrpc2: unmarshaling result: %w", err)
	}
	return nil
}

// retireCallWaiter retires id from outgoing calls before returning w to the pool.
func (c *conn) retireCallWaiter(id ID, w *waiter) {
	if c.retireCall(id) {
		putWaiter(w)
		return
	}
	<-w.ready
	putWaiter(w)
}

// retireCall removes the outgoing call id from the numeric slots if it is still
// present, reporting whether this call performed the removal. It is the
// lock-guarded hand-off that makes waiter pooling safe: only the goroutine that
// removes the entry owns the waiter afterwards.
func (c *conn) retireCall(id ID) (removed bool) {
	c.updateInFlight(func(s *inFlightState) {
		_, removed = s.outgoingCalls.Take(id)
	})
	return removed
}

// Notify implements [Conn].
func (c *conn) Notify(ctx context.Context, method string, params any) error {
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return fmt.Errorf("jsonrpc2: marshaling notify parameters: %w", err)
	}

	var shutErr error
	c.updateInFlight(func(s *inFlightState) {
		shutErr = s.shuttingDown(ErrClientClosing)
	})
	if shutErr != nil {
		return shutErr
	}

	return c.writeNotification(ctx, method, raw)
}

// frameWriter is the optional, package-internal extension of [Stream] that
// frames a message from its concrete fields, so the connection hot path can emit
// a call, notification, or response without boxing a wire value into the
// [Message] interface (which forces a heap allocation per send). The built-in
// framers implement it; a [Stream] that does not is written through the boxed
// [Stream.Write] fallback.
type frameWriter interface {
	writeCall(ctx context.Context, id ID, method string, params RawMessage) (int64, error)
	writeNotification(ctx context.Context, method string, params RawMessage) (int64, error)
	writeResponse(ctx context.Context, id ID, result RawMessage, err error) (int64, error)
}

// writeCall frames and sends a call envelope from its concrete fields, avoiding
// the per-send Message box on framers that implement frameWriter.
func (c *conn) writeCall(ctx context.Context, id ID, method string, params RawMessage) error {
	if c.fw != nil {
		return c.guardedWrite(ctx, func() (int64, error) {
			return c.fw.writeCall(ctx, id, method, params)
		})
	}
	return c.write(ctx, callWire{id: id, method: method, params: params})
}

// writeNotification frames and sends a notification envelope from its concrete
// fields, avoiding the per-send Message box on framers that implement
// frameWriter.
func (c *conn) writeNotification(ctx context.Context, method string, params RawMessage) error {
	if c.fw != nil {
		return c.guardedWrite(ctx, func() (int64, error) {
			return c.fw.writeNotification(ctx, method, params)
		})
	}
	return c.write(ctx, notificationWire{method: method, params: params})
}

// writeResponse frames and sends a response envelope from its concrete fields,
// avoiding the per-send Message box on framers that implement frameWriter.
func (c *conn) writeResponse(ctx context.Context, id ID, result RawMessage, respErr error) error {
	if c.fw != nil {
		return c.guardedWrite(ctx, func() (int64, error) {
			return c.fw.writeResponse(ctx, id, result, respErr)
		})
	}
	return c.write(ctx, responseWire{id: id, result: result, err: respErr})
}

// guardedWrite performs the shutdown check, runs the stream write effect, and
// propagates an unattributable write failure into the in-flight state, sharing
// the body of write across the concrete frameWriter paths.
func (c *conn) guardedWrite(ctx context.Context, do func() (int64, error)) error {
	var shutErr error
	c.updateInFlight(func(s *inFlightState) {
		shutErr = s.shuttingDown(ErrServerClosing)
	})
	if shutErr != nil {
		return shutErr
	}

	_, err := do()
	c.afterWrite(ctx, err)
	return err
}

// write sends msg on the stream after checking that the connection is not
// shutting down, and propagates an unattributable write failure into the
// in-flight state so that in-flight incoming calls are canceled. It is the
// boxed-Message path used for foreign [Stream] implementations and by the batch
// collector; the connection's own call/notification/response sends use the
// concrete writeCall/writeNotification/writeResponse helpers above.
//
// The [Stream] contract already serializes concurrent writers and emits each
// frame contiguously, so write holds no additional lock across the I/O.
func (c *conn) write(ctx context.Context, msg Message) error {
	return c.guardedWrite(ctx, func() (int64, error) {
		return c.stream.Write(ctx, msg)
	})
}

// afterWrite records a broken-writer error in the in-flight state. A failure
// that cannot be attributed to context cancellation means the writer is broken;
// since responses for incoming calls can no longer be delivered, those calls are
// canceled so the connection can drain toward idle.
func (c *conn) afterWrite(ctx context.Context, err error) {
	if err == nil || ctx.Err() != nil {
		return
	}
	c.updateInFlight(func(s *inFlightState) {
		if s.writeErr == nil {
			s.writeErr = err
			for _, r := range s.incomingByID {
				r.cancel()
			}
		}
	})
}

// Go implements [Conn].
func (c *conn) Go(ctx context.Context, handler Handler) {
	c.handler = handler
	c.updateInFlight(func(s *inFlightState) {
		s.reading = true
	})
	go c.readIncoming(ctx)
}

// readIncoming is the read goroutine. It reads frames from the stream,
// correlates responses to waiters, and dispatches requests to the handler until
// the stream returns an error.
//
// A single synchronous request is handled inline on this goroutine, so the
// common path spawns no goroutine. When such an inline handler releases itself
// with [Async], dispatch starts a fresh readIncoming goroutine to take over the
// reader role and reports handedOff=true; this goroutine then returns
// immediately, leaving the handler to finish concurrently and the successor
// reader to own loop termination. The reader role is therefore held by exactly
// one goroutine at a time, and the terminal cleanup below runs exactly once: on
// the goroutine that breaks the loop on a read error, never on a handed-off one.
func (c *conn) readIncoming(ctx context.Context) {
	var err error
	for {
		var (
			next nextRead
			rv   Request
		)
		next, err = c.readNext(ctx, &rv)
		if err != nil {
			break
		}
		if next.resp != nil {
			c.deliverResponse(next.resp)
			continue
		}
		if next.hasRV {
			if c.handleRequest(ctx, &rv) {
				// The handler released itself with Async; a successor reader owns
				// loop termination from here.
				return
			}
			continue
		}
		if c.dispatch(ctx, next.req, next.msgs, next.batch) {
			// A handler released itself with Async; a successor reader has taken
			// over and owns loop termination. Do not run the terminal cleanup.
			return
		}
	}

	c.updateInFlight(func(s *inFlightState) {
		s.reading = false
		s.readErr = err
		// With the reader stopped, outstanding outgoing calls can never be
		// answered; retire them with the terminating error.
		s.outgoingCalls.Drain(func(id ID, w *waiter) {
			w.deliver(&Response{id: id, err: err})
		})
	})
}

// deliver signals the waiter with resp. The ready channel has capacity one and
// the waiter is delivered to exactly once (guarded by the outgoing call slots),
// so the send never blocks.
func (w *waiter) deliver(resp *Response) {
	w.response = resp
	w.ready <- struct{}{}
}

// deliverResult unmarshals a borrowed-frame result directly into the caller's
// destination before the next read can invalidate result. It is used only after
// the response has been removed from outgoingCalls, so the waiter is delivered
// exactly once and can safely carry the outcome back to Call.
func (w *waiter) deliverResult(codec Codec, result RawMessage, respErr error) {
	if respErr != nil {
		w.err = respErr
		w.ready <- struct{}{}
		return
	}
	if w.result != nil {
		if err := unmarshalResult(codec, result, w.result); err != nil {
			w.err = fmt.Errorf("jsonrpc2: unmarshaling result: %w", err)
			w.ready <- struct{}{}
			return
		}
	}
	w.resultReady = true
	w.ready <- struct{}{}
}

// deliverResponse routes an incoming response to its waiting call. The lookup
// and removal happen together under the state lock so that the canceling Call
// and the read goroutine cannot both retire the same waiter.
func (c *conn) deliverResponse(resp *Response) {
	var w *waiter
	c.updateInFlight(func(s *inFlightState) {
		w, _ = s.outgoingCalls.Take(resp.id)
	})
	if w != nil {
		w.deliver(resp)
	}
	// An unmatched response (no pending call) is dropped: it answers a call that
	// was already canceled or never made.
}

// deliverNumberResponse routes an already-scanned numeric response to its waiter
// and unmarshals the borrowed result bytes before the next frame read.
func (c *conn) deliverNumberResponse(idNum int64, result RawMessage, respErr error) {
	id := NewNumberID(idNum)
	var w *waiter
	c.updateInFlight(func(s *inFlightState) {
		w, _ = s.outgoingCalls.Take(id)
	})
	if w != nil {
		w.deliverResult(c.codec, result, respErr)
	}
	// An unmatched response (no pending call) is dropped: it answers a call that
	// was already canceled or never made.
}

// Close implements [Conn].
func (c *conn) Close() error {
	c.updateInFlight(func(s *inFlightState) {
		s.connClosing = true
	})
	<-c.done
	return c.closeError()
}

// closeError reports the stream's close error under the state lock.
func (c *conn) closeError() error {
	var err error
	c.updateInFlight(func(s *inFlightState) {
		err = s.closeErr
	})
	return err
}

// Done implements [Conn].
func (c *conn) Done() <-chan struct{} { return c.done }

// Err implements [Conn]. A clean end of stream ([io.EOF]) and the shutdown
// sentinels are reported as nil so that a graceful Close does not surface a
// spurious error.
//
// Cancellation contract: the connection treats cancellation of the context
// passed to [Conn.Go] as a request for a graceful local shutdown, not as a
// connection failure. When the read loop stops because that context is canceled
// or its deadline expires, [context.Canceled] and [context.DeadlineExceeded] are
// folded into the clean-close set and Err reports nil. Genuine transport failures
// (a broken pipe reported as anything other than these shutdown causes) are still
// surfaced. [Conn.Close] remains the authoritative teardown and always reports
// nil on a clean shutdown.
func (c *conn) Err() error {
	var err error
	c.updateInFlight(func(s *inFlightState) {
		switch {
		case s.writeErr != nil:
			err = s.writeErr
		case s.readErr != nil && !errors.Is(s.readErr, io.EOF):
			err = s.readErr
		default:
			err = s.closeErr
		}
	})
	// A clean end of stream, an error that is the consequence of closing our own
	// stream during a graceful shutdown, the shutdown sentinels themselves (a
	// reply rejected because we are already closing), or a context cancellation
	// that asked the read loop to stop are not connection failures.
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
	}
	return err
}
