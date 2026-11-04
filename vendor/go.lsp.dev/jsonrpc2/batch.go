// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"sync"
)

// frameStream is the optional, package-internal extension of [Stream] that
// exposes raw frame access. The built-in framers implement it so that the
// connection can read a frame, classify it as a single message or a batch
// array, and write a batch response array as one frame, without the message
// boundary that [Stream.Read] and [Stream.Write] impose. A Stream that does not
// implement frameStream falls back to single-message reads and cannot receive
// batches.
type frameStream interface {
	// ReadFrame returns the raw JSON body of the next frame without decoding it.
	// The returned slice is valid only until the next read.
	ReadFrame(ctx context.Context) ([]byte, int64, error)

	// WriteFrame frames the already-encoded JSON body data and writes it as a
	// single frame.
	WriteFrame(ctx context.Context, data []byte) (int64, error)
}

// maxConnDirectUnmarshalResult bounds the Conn response fast path that
// unmarshals a borrowed result on the read goroutine. Larger results fall back
// to DecodeMessage so expensive payload decoding remains on the caller's
// goroutine and cannot head-of-line block unrelated incoming frames.
const maxConnDirectUnmarshalResult = 64 << 10

type nextRead struct {
	req   RequestMessage
	msgs  []RequestMessage
	resp  *Response
	batch bool
	hasRV bool
}

// readNext reads the next frame and classifies it. On success exactly one of
// req, msgs, resp, or hasRV is set: resp for a single response, req for a
// single request on the v1 interface path, hasRV (filling *rv) for a single
// request scanned into the caller's concrete value on the direct-return path,
// and msgs for a batch of one or more requests with batch=true. A batch frame
// containing only malformed entries still yields the slice so the dispatcher
// can answer with error responses.
//
// When the stream does not support raw frames, readNext falls back to the
// single-message [Stream.Read] and never reports a batch.
func (c *conn) readNext(ctx context.Context, rv *Request) (nextRead, error) {
	fs, ok := c.stream.(frameStream)
	if !ok {
		return c.readNextMessage(ctx)
	}

	frame, _, ferr := fs.ReadFrame(ctx)
	if ferr != nil {
		return nextRead{}, ferr
	}
	return c.classifyFrame(frame, rv)
}

func (c *conn) readNextMessage(ctx context.Context) (nextRead, error) {
	msg, _, err := c.stream.Read(ctx)
	if err != nil {
		return nextRead{}, err
	}
	if r, isResp := msg.(*Response); isResp {
		return nextRead{resp: r}, nil
	}
	return nextRead{req: msg.(RequestMessage)}, nil
}

func (c *conn) classifyFrame(frame []byte, rv *Request) (nextRead, error) {
	i := skipSpace(frame, 0)
	if i < len(frame) && frame[i] == '[' {
		return c.classifyBatchFrame(frame)
	}

	hasScannedRequest, deliveredResponse := c.classifyScannedFrame(frame, rv)
	if hasScannedRequest {
		return nextRead{hasRV: true}, nil
	}
	if deliveredResponse {
		return nextRead{}, nil
	}
	return decodeFrameMessage(frame)
}

func (c *conn) classifyBatchFrame(frame []byte) (nextRead, error) {
	reqs, isBatch, err := parseBatch(frame)
	if err != nil {
		return nextRead{batch: true}, err
	}
	// An empty array is answered with a single (unbracketed) error response, so
	// it is routed through the non-batch path with a single synthetic member.
	if !isBatch && len(reqs) == 1 {
		return nextRead{req: reqs[0]}, nil
	}
	return nextRead{msgs: reqs, batch: isBatch}, nil
}

func (c *conn) classifyScannedFrame(frame []byte, rv *Request) (hasRequest, deliveredResponse bool) {
	// A single classification scan records the frame's top-level member spans
	// once. Requests fill the caller's concrete value directly (no message
	// box); numeric-id success responses deliver their borrowed result inline,
	// before the next read can invalidate it. Everything the classifier does
	// not recognize -- malformed objects, error responses, string ids,
	// oversized results -- falls through to the general decode path below, so
	// those semantics are unchanged. This is an optimization gate, not the
	// correctness boundary.
	var f fields
	if !scanValidObject(frame, &f) {
		return false, false
	}
	// Client-only connections have no handler; their peers do not send
	// requests, and a stray one takes the boxed fallback.
	if c.handler != nil && f.hasMethod && !f.hasResult && !f.hasError {
		return f.fillRequest(rv) == nil, false
	}
	return false, c.deliverScannedNumberFields(&f)
}

func (c *conn) deliverScannedNumberFields(f *fields) bool {
	if f.hasMethod || !f.hasResult || f.hasError || !f.hasID || isNullLiteral(f.id) {
		return false
	}
	id, ok := decodeID(f.id)
	if !ok || !id.IsNumber() || len(f.result) > maxConnDirectUnmarshalResult {
		return false
	}
	c.deliverNumberResponse(id.num, f.result, nil)
	return true
}

func decodeFrameMessage(frame []byte) (nextRead, error) {
	msg, err := DecodeMessage(frame)
	if err != nil {
		return nextRead{}, err
	}
	if r, isResp := msg.(*Response); isResp {
		return nextRead{resp: r}, nil
	}
	return nextRead{req: msg.(RequestMessage)}, nil
}

func scanValidObject(frame []byte, f *fields) bool {
	end, ok := scanObject(frame, f)
	return ok && skipSpace(frame, end) == len(frame) && f.validVersion()
}

// dispatch handles one frame's worth of boxed requests: the rare
// single-message fallbacks (a non-frame stream's Read, or the synthetic
// empty-batch error member) and batch arrays. A batch is handled member by
// member, collecting the responses for the call members into a single array
// that is written with one frame, while a batch made up entirely of
// notifications produces no response per the JSON-RPC 2.0 specification.
//
// It reports handedOff=true when an inline single-message handler released the
// read loop with [Async] and a successor reader has taken over. The batch path
// keeps its goroutine-per-member dispatch and never hands off the reader, so it
// always reports false.
func (c *conn) dispatch(ctx context.Context, req RequestMessage, msgs []RequestMessage, batch bool) (handedOff bool) {
	if !batch {
		if req != nil {
			return c.handleBoxedRequest(ctx, req)
		}
		return false
	}
	c.dispatchBatch(ctx, msgs)
	return false
}

// batchCollector accumulates the response bodies of a batch's call members and
// writes them as one array frame when the batch is fully handled.
type batchCollector struct {
	c        *conn
	resps    []responseWire // response envelopes, one per call member
	pending  int            // call members not yet replied
	mu       sync.Mutex
	released bool // the dispatch loop has finished enqueuing members
}

// dispatchBatch parses, validates, and handles each member of a batch, then
// writes the collected responses as a single array frame.
func (c *conn) dispatchBatch(ctx context.Context, msgs []RequestMessage) {
	// An empty array, or a non-array element where an object was required, has
	// already been turned into a single InvalidRequest member by parseBatch, so
	// msgs is never empty here.
	calls := 0
	for _, req := range msgs {
		if respondsInBatch(req) {
			calls++
		}
	}

	bc := &batchCollector{c: c, pending: calls}
	if calls > 0 {
		bc.resps = make([]responseWire, 0, calls)
	}

	for _, req := range msgs {
		c.handleBatchMember(ctx, req, bc)
	}

	bc.finish(ctx)
}

// finish marks the enqueuing phase complete and flushes the collected responses
// if every call member has already replied. When some members are still
// in flight (async handlers), the last reply flushes instead.
func (bc *batchCollector) finish(ctx context.Context) {
	bc.mu.Lock()
	bc.released = true
	flush := bc.pending == 0 && len(bc.resps) > 0
	resps := bc.resps
	bc.mu.Unlock()
	if flush {
		bc.write(ctx, resps)
	}
}

// add records one call member's encoded response and flushes the batch once the
// final member has replied and enqueuing is complete.
func (bc *batchCollector) add(ctx context.Context, resp responseWire) {
	bc.mu.Lock()
	bc.resps = append(bc.resps, resp)
	bc.pending--
	flush := bc.released && bc.pending == 0
	resps := bc.resps
	bc.mu.Unlock()
	if flush {
		bc.write(ctx, resps)
	}
}

// write joins the response envelopes into one JSON array and emits it as a
// single frame.
func (bc *batchCollector) write(ctx context.Context, resps []responseWire) {
	fs, ok := bc.c.stream.(frameStream)
	if !ok {
		return
	}

	var shutErr error
	bc.c.updateInFlight(func(s *inFlightState) {
		shutErr = s.shuttingDown(ErrServerClosing)
	})
	if shutErr != nil {
		return
	}

	buf := appendResponseBatch(make([]byte, 0, 2), resps)

	_, err := fs.WriteFrame(ctx, buf)
	bc.c.afterWrite(ctx, err)
}

// parseBatch parses a batch array into its request members and reports whether
// the response must be a batch array. Per the JSON-RPC 2.0 specification an
// empty array "[]" is itself an invalid request answered with a single
// (unbracketed) error response, so it returns isBatch=false with one synthetic
// invalid member; any non-empty array returns isBatch=true. A member that is not
// a valid request object yields an InvalidRequest member in its place so the
// valid members are still handled.
func parseBatch(frame []byte) (reqs []RequestMessage, isBatch bool, err error) {
	parsed, perr := ParseRequests(frame)
	if perr != nil {
		return nil, false, perr
	}
	if len(parsed) == 0 {
		// An empty batch "[]" is answered with a single error response carrying a
		// null id, not a one-element array.
		return []RequestMessage{invalidBatchMember(ErrInvalidRequest)}, false, nil
	}

	out := make([]RequestMessage, 0, len(parsed))
	for _, pm := range parsed {
		if pm.Err != nil {
			out = append(out, invalidBatchMember(pm.Err))
			continue
		}
		out = append(out, pm.Msg)
	}
	return out, true, nil
}

// invalidBatchMember builds a placeholder call that carries no real id and is
// answered with err. It lets a malformed batch member flow through the normal
// dispatch path and produce a spec-compliant error response with a null id.
func invalidBatchMember(err *Error) RequestMessage {
	return &invalidRequest{err: err}
}

// invalidRequest is a synthetic [RequestMessage] standing in for a malformed batch
// member. It is always answered with a null-id error response and never reaches
// the user handler.
type invalidRequest struct {
	err *Error
}

func (*invalidRequest) Method() string { return "" }

func (*invalidRequest) Params() RawMessage { return nil }

func (*invalidRequest) jsonrpc2Message() {}

func (*invalidRequest) jsonrpc2Request() {}

// respondsInBatch reports whether req contributes a response body to a batch's
// response array: calls and malformed members do, notifications do not.
func respondsInBatch(req RequestMessage) bool {
	switch req.(type) {
	case *Call, *invalidRequest:
		return true
	default:
		return false
	}
}
