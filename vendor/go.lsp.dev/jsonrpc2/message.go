// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import "strings"

// Request is the concrete request shape used by direct-return dispatch. It
// is a value, not an interface: the scanner fills it in place and dispatch
// embeds it in the per-request bookkeeping, so a request needs no message box.
// Method and params spans are borrowed from the transport frame and are valid
// until the handler returns.
type Request struct {
	id     ID
	method string
	params RawMessage
	isCall bool
}

// ID returns the request id. It is the zero ID for a notification.
func (r *Request) ID() ID { return r.id }

// Method returns the request method.
func (r *Request) Method() string { return r.method }

// Params returns the request parameters, nil when absent or null.
func (r *Request) Params() RawMessage { return r.params }

// IsCall reports whether the request expects a response.
func (r *Request) IsCall() bool { return r.isCall }

// Clone returns a copy of the request whose method and params own their
// bytes, safe to retain after the handler returns. It is the escape hatch for
// the borrowed-lifetime contract: the original request's spans alias the
// transport frame and die with the handler, and the original struct itself is
// recycled, so any retention must go through Clone.
func (r *Request) Clone() *Request {
	// The id already owns its bytes (decodeID copies string ids), so a plain
	// copy suffices; only the method and params spans borrow from the frame.
	return &Request{
		id:     r.id,
		method: strings.Clone(r.method),
		params: RawMessage(cloneBytes(r.params)),
		isCall: r.isCall,
	}
}

// Call is a request that expects a [Response]. The response carries a matching
// [ID].
type Call struct {
	method string
	params RawMessage
	id     ID
}

// compile-time checks that *Call satisfies the request interfaces.
var (
	_ Message        = (*Call)(nil)
	_ RequestMessage = (*Call)(nil)
)

// NewCall constructs a [*Call] for the supplied id, method, and pre-encoded
// parameters. The params bytes are retained verbatim; pass nil for no
// parameters.
func NewCall(id ID, method string, params RawMessage) *Call {
	return &Call{
		id:     id,
		method: method,
		params: params,
	}
}

// ID reports the identifier of the call.
func (c *Call) ID() ID { return c.id }

// Method implements [RequestMessage].
func (c *Call) Method() string { return c.method }

// Params implements [RequestMessage].
func (c *Call) Params() RawMessage { return c.params }

func (*Call) jsonrpc2Message() {}

func (*Call) jsonrpc2Request() {}

// Notification is a request that does not expect a response and therefore
// carries no [ID].
type Notification struct {
	method string
	params RawMessage
}

// compile-time checks that *Notification satisfies the request interfaces.
var (
	_ Message        = (*Notification)(nil)
	_ RequestMessage = (*Notification)(nil)
)

// NewNotification constructs a [*Notification] for the supplied method and
// pre-encoded parameters. The params bytes are retained verbatim; pass nil for
// no parameters.
func NewNotification(method string, params RawMessage) *Notification {
	return &Notification{
		method: method,
		params: params,
	}
}

// Method implements [RequestMessage].
func (n *Notification) Method() string { return n.method }

// Params implements [RequestMessage].
func (n *Notification) Params() RawMessage { return n.params }

func (*Notification) jsonrpc2Message() {}

func (*Notification) jsonrpc2Request() {}

// Response is a reply to a [Call]. It carries the same [ID] as the call it
// answers, and exactly one of a result or an error.
type Response struct {
	result RawMessage
	err    error
	id     ID
}

// compile-time check that *Response implements Message.
var _ Message = (*Response)(nil)

// NewResponse constructs a [*Response] for the supplied id, pre-encoded result,
// and error. When err is non-nil the result is ignored on the wire.
func NewResponse(id ID, result RawMessage, err error) *Response {
	return &Response{
		id:     id,
		result: result,
		err:    err,
	}
}

// ID reports the identifier of the response.
func (r *Response) ID() ID { return r.id }

// Result reports the raw, already-encoded result of the response, or nil when
// the response carries an error.
func (r *Response) Result() RawMessage { return r.result }

// Err reports the error of the response, or nil on success.
func (r *Response) Err() error { return r.err }

func (*Response) jsonrpc2Message() {}
