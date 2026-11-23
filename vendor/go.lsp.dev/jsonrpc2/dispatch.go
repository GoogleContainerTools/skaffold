// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// requestValue converts a boxed wire request (a batch member or a
// single-message fallback) into the concrete dispatch shape. The spans keep
// borrowing whatever the box borrowed; ownership questions are settled by the
// release path exactly as for scanned requests.
func requestValue(req RequestMessage) Request {
	switch m := req.(type) {
	case *Call:
		return Request{id: m.id, method: m.method, params: m.params, isCall: true}
	case *Notification:
		return Request{method: m.method, params: m.params}
	default:
		// invalidRequest never reaches here; the dispatch paths answer it
		// before conversion.
		return Request{}
	}
}

// setupRequest takes a pooled incomingRequest, fills it from rv, consults the
// optional Preempter, and registers the request as in-flight. It returns
// done=true when the request has already been fully answered (preempted, or a
// call rejected because the connection is shutting down), in which case the
// caller must not invoke the handler.
func (c *conn) setupRequest(ctx context.Context, rv *Request, bc *batchCollector) (ir *incomingRequest, done bool) {
	ir = getIncomingRequest()
	ir.parent = ctx
	ir.id = rv.id
	ir.isCall = rv.isCall
	ir.request = *rv

	// A Preempter, if configured, runs inline on the read goroutine before the
	// request is dispatched. ErrNotHandled or a nil handled value defers to the
	// handler; every other result or error is answered here.
	if c.preempter != nil {
		result, perr := c.preempter.Preempt(ir, &ir.request)
		if !errors.Is(perr, ErrNotHandled) && (result != nil || perr != nil) {
			c.updateInFlight(func(s *inFlightState) { s.incoming++ })
			c.completeRequest(ir, bc, result, perr)
			return nil, true
		}
	}

	var shutErr error
	c.updateInFlight(func(s *inFlightState) {
		s.incoming++
		if rv.isCall {
			if s.incomingByID == nil {
				s.incomingByID = make(map[ID]*incomingRequest)
			}
			s.incomingByID[rv.id] = ir
			shutErr = s.shuttingDown(ErrServerClosing)
		}
	})

	if shutErr != nil {
		// Reject a call that arrived while shutting down with an immediate error
		// response, and account for it so the connection can reach idle.
		c.completeRequest(ir, bc, nil, shutErr)
		return nil, true
	}

	return ir, false
}

// completeRequest answers a request without invoking the handler (a preempted
// request or one rejected during shutdown), releases its bookkeeping, and
// recycles it.
func (c *conn) completeRequest(ir *incomingRequest, bc *batchCollector, result any, err error) {
	if ir.isCall && ir.replied.done.CompareAndSwap(false, true) {
		_ = c.sendResponse(ir, ir.id, bc, result, err)
	}
	c.afterHandle(ir)
	putIncomingRequest(ir)
}

// runHandler invokes the handler and answers the request from its return
// values, performing the request-scoped cleanup through defers so that a
// panic cannot leak the in-flight counter or the incomingByID entry and
// deadlock a later Close.
//
// Defers run last-in-first-out: the recover defer (registered last) runs
// first and answers an unanswered call, then the soft release frees the read
// loop (or the batch dispatch loop), then afterHandle decrements the
// in-flight counter. The caller owns the pool put, after its final read of
// ir's release flags.
func (c *conn) runHandler(ir *incomingRequest, bc *batchCollector) {
	defer c.afterHandle(ir)
	// release(true) is the soft release that frees the read loop in the
	// synchronous case (when the handler did not call Async) and after a panic.
	// It is idempotent, so a request that already released early via Async is
	// unaffected.
	defer ir.rel.release(true)
	defer func() {
		if r := recover(); r != nil {
			// A handler panic is a request-scoped failure, not a process-wide one:
			// answer an unanswered call with an internal error so the caller (and
			// any batch flush awaiting this member) is not left hanging, then fail
			// the connection so the panic is surfaced rather than silently
			// swallowed.
			if ir.isCall && ir.replied.done.CompareAndSwap(false, true) {
				_ = c.sendResponse(ir, ir.id, bc, nil, Errorf(InternalError, "jsonrpc2: handler panicked: %v", r))
			}
			c.fail(fmt.Errorf("jsonrpc2: handler for %q panicked: %v", ir.request.method, r))
		}
	}()

	result, err := c.handler(ir, &ir.request)
	if !ir.isCall {
		// A notification has no response; a handler error is a connection-level
		// failure.
		if err != nil {
			c.fail(err)
		}
		return
	}
	if ir.replied.done.CompareAndSwap(false, true) {
		_ = c.sendResponse(ir, ir.id, bc, result, err)
	}
}

// handleRequest dispatches a single (non-batch) incoming request inline on
// the read goroutine. The handler runs to completion on this goroutine in the
// common synchronous case, so it spawns no goroutine and handlers observe
// requests in wire order.
//
// When the handler releases itself with [Async], the release effect spawns a
// fresh read goroutine to take over the reader role and records the handoff;
// handleRequest then reports it so the current read loop returns while this
// goroutine finishes the handler concurrently. The successor reader owns loop
// termination, so the reader role is always held by exactly one goroutine.
func (c *conn) handleRequest(ctx context.Context, rv *Request) (handedOff bool) {
	ir, done := c.setupRequest(ctx, rv, nil)
	if done {
		return false
	}

	// The inline releaser carries the state for the Async handoff (ch left
	// nil). It is a value field of ir, so initializing it in place allocates
	// nothing.
	ir.rel = releaser{active: true, conn: c, ctx: ctx, ir: ir}

	c.runHandler(ir, nil)

	// The pool put is the LAST touch on ir: it must come after the release-flag
	// reads, because the instant ir is pooled another reader can recycle it. A
	// detached (Async) request is never pooled -- its lifetime escaped the
	// dispatch path and a detached handler may legally hold it until it returns
	// on its own schedule.
	handedOff = ir.rel.handedOff
	if !ir.rel.detached {
		putIncomingRequest(ir)
	}
	return handedOff
}

// handleBoxedRequest serves the rare boxed single-message paths: the
// non-frame Stream.Read fallback and the synthetic empty-batch error member.
func (c *conn) handleBoxedRequest(ctx context.Context, req RequestMessage) (handedOff bool) {
	if inv, ok := req.(*invalidRequest); ok {
		// A malformed request never reaches the handler; answer it directly
		// with a null-id error response.
		c.writeInvalid(ctx, nil, inv.err)
		return false
	}
	rv := requestValue(req)
	return c.handleRequest(ctx, &rv)
}

// handleBatchMember dispatches one batch member. Unlike the inline
// single-message path, a batch member runs in its own goroutine gated by a
// [releaser] channel so that an async member can overlap the remaining
// members of the same batch: the dispatch loop blocks until the member either
// calls [Async] (releasing it to run concurrently) or returns.
func (c *conn) handleBatchMember(ctx context.Context, req RequestMessage, bc *batchCollector) {
	if inv, ok := req.(*invalidRequest); ok {
		// A malformed batch member never reaches the handler; answer it directly
		// with a null-id error response so the valid members are still served.
		c.writeInvalid(ctx, bc, inv.err)
		return
	}

	rv := requestValue(req)
	ir, done := c.setupRequest(ctx, &rv, bc)
	if done {
		return
	}

	ir.rel = releaser{active: true, ch: make(chan struct{}), ir: ir}

	// The gate channel must be captured before the member goroutine starts: a
	// fast synchronous member can close it AND recycle ir (zeroing ir.rel)
	// before this goroutine would otherwise load the field.
	ch := ir.rel.ch
	go c.runBatchMember(ir, bc)

	// Block until the member releases the dispatch loop: immediately for an
	// async member (via Async), or when the handler returns for a synchronous
	// one.
	<-ch
}

// runBatchMember runs one batch member's handler on its own goroutine and
// recycles the request afterward. The release flags are written by the hard
// release on this same goroutine (Async is called by the handler running
// here), so the post-run read carries no race.
func (c *conn) runBatchMember(ir *incomingRequest, bc *batchCollector) {
	c.runHandler(ir, bc)
	if !ir.rel.detached {
		putIncomingRequest(ir)
	}
}

// repliedFlag reports whether a request has been answered. Direct-return
// dispatch answers in exactly one place per path, but an [Async]-detached
// handler returns (and answers) on its own goroutine while connection
// shutdown can answer the same call from another, so the first-answer claim
// stays an atomic CAS rather than a plain bool.
type repliedFlag struct {
	done atomic.Bool
}

// sendResponse marshals result (or err) into a response wire envelope and
// either writes it immediately or, for a batch member, hands it to the
// collector.
func (c *conn) sendResponse(ctx context.Context, id ID, bc *batchCollector, result any, err error) error {
	resp := responseWire{id: id, err: err}
	if err == nil {
		raw, merr := marshalParams(c.codec, result)
		if merr != nil {
			resp.err = Errorf(InternalError, "jsonrpc2: marshaling response result: %v", merr)
		} else {
			resp.result = raw
		}
	}
	// The id may be reused by the peer as soon as the response is sent, so drop it
	// from the incoming map before writing.
	c.updateInFlight(func(s *inFlightState) {
		delete(s.incomingByID, id)
	})

	if bc != nil {
		bc.add(ctx, resp)
		return nil
	}
	return c.writeResponse(ctx, resp.id, resp.result, resp.err)
}

// afterHandle releases the per-request resources and decrements the in-flight
// counter so the connection can progress toward idle.
func (c *conn) afterHandle(ir *incomingRequest) {
	ir.cancel()
	c.updateInFlight(func(s *inFlightState) {
		if ir.isCall {
			delete(s.incomingByID, ir.id)
		}
		s.incoming--
	})
}

// writeInvalid answers a malformed batch member with a null-id error response,
// collecting it into the batch array. It does not touch the in-flight counters
// because a malformed member is never registered as in-flight work.
func (c *conn) writeInvalid(ctx context.Context, bc *batchCollector, err *Error) {
	resp := responseWire{id: ID{}, err: err}
	if bc != nil {
		bc.add(ctx, resp)
		return
	}
	_ = c.writeResponse(ctx, resp.id, resp.result, resp.err)
}

// fail records err as the connection's terminating error and closes the stream
// so the read goroutine unwinds. It is used for handler-returned connection
// errors.
func (c *conn) fail(err error) {
	c.updateInFlight(func(s *inFlightState) {
		if s.writeErr == nil {
			s.writeErr = err
		}
		if s.closer != nil {
			s.closeErr = s.closer.Close()
			s.closer = nil
		}
	})
}
