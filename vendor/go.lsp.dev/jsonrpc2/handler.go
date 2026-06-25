// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"fmt"
	"sync"
)

// Handler is invoked to handle an incoming [Request]. It is the direct-return
// handler shape: the returned result (or error) becomes the call's response,
// so dispatch needs no per-request reply machinery. To answer a call with a
// JSON-RPC error, return that error (an [*Error] is sent verbatim; any other
// error is wrapped). For a notification the result is discarded and a non-nil
// error fails the connection.
//
// Lifetime contract: req, its Method, and its Params are valid only until the
// handler returns; the request struct is pooled and recycled afterward. The
// three legal retention patterns are:
//
//  1. Copy what you need before returning (unmarshal params, copy the method).
//  2. Take [Request.Clone] and retain the owned clone.
//  3. Release with [Async], which clones the request in place automatically;
//     the request then remains valid until the (now concurrent) handler
//     returns.
//
// The per-request ctx likewise dies with the handler; use [DetachContext] for
// work that outlives it. Builds tagged jsonrpc2poison scribble recycled
// requests so violations fail loudly in tests.
//
// Reentrancy and the deadlock-avoidance contract: in the synchronous case a
// Handler runs inline on the connection's read loop, which is blocked until
// the handler returns or releases itself. Because a [Conn] is symmetric, a
// handler may call back into the same connection while serving a request (the
// pattern LSP relies on), but it must observe one rule. To make a
// server-initiated *call* back to the peer with [Conn.Call] and await its
// response from within a handler, the handler MUST first release the read
// loop with [Async] (or be wrapped with [AsyncHandler]); otherwise it
// deadlocks, because the response to that callback must be read by the very
// read loop that is blocked running the handler. A server-initiated
// *notification* with [Conn.Notify] does not require [Async]: Notify only
// writes and never waits for a response.
//
// The connection enforces a deterministic outcome for a misbehaving handler:
// if a handler panics, the connection answers an unanswered call with an
// [InternalError] response, then fails the connection so the panic is
// surfaced through [Conn.Err] rather than silently swallowed.
type Handler func(ctx context.Context, req *Request) (result any, err error)

// DetachContext returns a context that is safe to retain after the handler
// returns. The per-request context handed to a handler is pooled and recycled
// with the request, so retaining it past the handler's return is illegal;
// DetachContext steps up to the connection-lifetime parent, keeping its values
// and deadline but dropping the request-scoped cancellation.
func DetachContext(ctx context.Context) context.Context {
	if ir, ok := ctx.(*incomingRequest); ok {
		return ir.parent
	}
	return ctx
}

// Preempter is an optional hook consulted for every incoming [Request] before
// it is dispatched to the [Handler]. It runs inline on the read loop, so it
// must not block; it is intended for fast, ordered handling such as
// "$/cancelRequest" notifications that must be observed ahead of the request
// they cancel. The request follows the same borrowed lifetime as a Handler's:
// valid only until Preempt returns.
type Preempter interface {
	// Preempt is called for each incoming request before it is dispatched.
	// Returning [ErrNotHandled] (or a nil handled value) defers the request to
	// the [Handler]; returning any other result handles the request inline and
	// it is not passed to the Handler.
	Preempt(ctx context.Context, req *Request) (handled any, err error)
}

// ErrNotHandled is returned by a [Preempter] (or a [Handler]) to indicate that
// the request was not handled and should fall through to the next stage.
const ErrNotHandled = constError("jsonrpc2: request not handled")

// MethodNotFoundHandler is a [Handler] that answers every call with the
// standard "method not found" error and drops notifications. It is intended to
// be the final handler in a chain.
func MethodNotFoundHandler(ctx context.Context, req *Request) (any, error) {
	if !req.IsCall() {
		return nil, nil
	}
	return nil, fmt.Errorf("%q: %w", req.Method(), ErrMethodNotFound)
}

// CancelHandler wraps handler to support cancellation by request id. It
// returns the wrapped handler and a canceller that, when called with the id of
// an in-flight call, cancels the context passed to that call's handler.
//
// The canceller is safe for concurrent use and is a no-op for an id that is
// not currently being handled.
func CancelHandler(handler Handler) (h Handler, canceller func(id ID)) {
	var mu sync.Mutex
	handling := make(map[ID]context.CancelFunc)

	h = func(ctx context.Context, req *Request) (any, error) {
		if !req.IsCall() {
			return handler(ctx, req)
		}

		id := req.ID()
		cancelCtx, cancel := context.WithCancel(ctx)

		mu.Lock()
		handling[id] = cancel
		mu.Unlock()
		// The deregistration runs inside the wrapped handler, strictly before
		// dispatch recycles the request, so the captured id stays valid.
		defer func() {
			mu.Lock()
			delete(handling, id)
			mu.Unlock()
			cancel()
		}()

		return handler(cancelCtx, req)
	}

	canceller = func(id ID) {
		mu.Lock()
		cancel, found := handling[id]
		mu.Unlock()
		if found {
			cancel()
		}
	}

	return h, canceller
}

// AsyncHandler wraps handler so that every request is released for concurrent
// handling as soon as it is received: the read loop hands its role to a
// successor immediately and the wrapped handler continues concurrently.
//
// Requests are still started in wire order, but they run concurrently and
// carry no mutual ordering guarantee once released. The release clones the
// request's borrowed spans, so the wrapped handler keeps a valid request for
// as long as it runs.
func AsyncHandler(handler Handler) Handler {
	return func(ctx context.Context, req *Request) (any, error) {
		Async(ctx)
		return handler(ctx, req)
	}
}

// Async signals that the current request may be handled concurrently with
// requests that arrive after it. When ctx is a request context served by a
// connection, the read loop is released to process the next message
// immediately; the remainder of the handler then runs concurrently. The
// request's borrowed method and params are cloned in place by the release, so
// they remain valid until the handler returns.
//
// Async must be called at most once per request context. Calling it on a
// context that does not carry a release token (for example a non-request
// context) is a no-op.
func Async(ctx context.Context) {
	if r, ok := ctx.Value(asyncKey{}).(*releaser); ok {
		r.release(false)
	}
}

// asyncKey is the context key under which a request's [releaser] is stored.
type asyncKey struct{}

// releaser implements the one-shot release of a request for concurrent
// handling. The first release runs the release effect exactly once; a request
// that never calls [Async] is released by the dispatch path's own deferred
// "soft" release after the handler returns, so it is handled synchronously,
// while one that does call Async is released early and may overlap later
// requests.
//
// The effect is selected by the populated fields rather than a per-request
// closure, so dispatch allocates nothing for it:
//
//   - Batch member (ch != nil): either kind of release closes ch, unblocking
//     the dispatch loop that spawned the member's goroutine.
//   - Inline single message (ch == nil): a hard release (Async) starts the
//     successor read goroutine that takes over the reader role and records the
//     handoff; a soft release is a no-op, since the read loop simply continues
//     once the inline handler returns.
//
// A release effect must be safe to run on whichever goroutine first releases
// the request; the inline handoff only ever runs on the read goroutine, since
// the inline handler runs there.
type releaser struct {
	ctx context.Context

	// ch, when non-nil, marks the batch-member mode: release closes it.
	ch chan struct{}

	// ir points back to the request this releaser belongs to. A hard release
	// (Async) is the single escape point at which a request outlives the frame
	// it borrows from, so release clones the request's borrowed spans in place
	// before the read loop can advance to the next frame.
	ir *incomingRequest

	// conn drives the inline single-message handoff (ch == nil); the successor
	// reader dispatches through conn.handler.
	conn     *conn
	mu       sync.Mutex
	released bool

	// active marks the releaser as set up for an in-flight request. It is needed
	// because the releaser is a value field of incomingRequest, so the zero
	// value must be distinguishable from a releaser that dispatch has
	// initialized; only an active releaser is handed out for the asyncKey{}
	// context value.
	active bool

	// detached is set by a hard release on the releasing goroutine. The
	// dispatch path reads it after the handler returns (same goroutine) to
	// decide whether the request may be pooled: a detached request's lifetime
	// escaped the dispatch path and is left to the garbage collector.
	detached bool

	handedOff bool // set by an inline hard release so the reader loop returns
}

// release runs the release effect exactly once. A hard release (soft=false)
// panics if the request was already released, catching a double call to
// [Async]; a soft release (the dispatch path's own fallback after the handler
// returns) is idempotent.
func (r *releaser) release(soft bool) {
	r.mu.Lock()
	if r.released {
		r.mu.Unlock()
		if !soft {
			panic("jsonrpc2: Async called multiple times")
		}
		return
	}
	r.released = true
	r.mu.Unlock()

	if !soft {
		r.detached = true
		if r.ir != nil {
			// The handler is detaching from the read loop, so the borrowed method
			// and params spans must stop aliasing the transport frame before the
			// loop can read (and thereby invalidate) the next frame. Cloning here --
			// before the successor reader starts or the batch dispatch loop
			// resumes -- is what makes the hard release the single legal escape
			// point for a request.
			r.ir.cloneRequestOwned()
		}
	}

	switch {
	case r.ch != nil:
		// Batch member: free the dispatch loop whether the release was soft or hard.
		close(r.ch)
	case !soft:
		// Inline single message, hard release (Async): hand the reader role to a
		// fresh goroutine and signal the current read loop to return. A soft release
		// (the handler returned) needs no action; the read loop continues inline.
		r.handedOff = true
		go r.conn.readIncoming(r.ctx)
	}
}
