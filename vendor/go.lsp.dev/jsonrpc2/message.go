// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/segmentio/encoding/json"
)

// Message is the interface to all JSON-RPC message types.
//
// They share no common functionality, but are a closed set of concrete types
// that are allowed to implement this interface.
//
// The message types are *Call, *Response and *Notification.
type Message interface {
	// jsonrpc2Message is used to make the set of message implementations a
	// closed set.
	jsonrpc2Message()
}

// Request is the shared interface to jsonrpc2 messages that request
// a method be invoked.
//
// The request types are a closed set of *Call and *Notification.
type Request interface {
	Message

	// Method is a string containing the method name to invoke.
	Method() string
	// Params is either a struct or an array with the parameters of the method.
	Params() json.RawMessage

	// jsonrpc2Request is used to make the set of request implementations closed.
	jsonrpc2Request()
}

// Call is a request that expects a response.
//
// The response will have a matching ID.
type Call struct {
	// Method is a string containing the method name to invoke.
	method string
	// Params is either a struct or an array with the parameters of the method.
	params json.RawMessage
	// id of this request, used to tie the Response back to the request.
	id ID
}

// make sure a Call implements the Request, json.Marshaler and json.Unmarshaler and interfaces.
var (
	_ Request          = (*Call)(nil)
	_ json.Marshaler   = (*Call)(nil)
	_ json.Unmarshaler = (*Call)(nil)
)

// NewCall constructs a new Call message for the supplied ID, method and
// parameters.
func NewCall(id ID, method string, params interface{}) (*Call, error) {
	p, merr := marshalInterface(params)
	req := &Call{
		id:     id,
		method: method,
		params: p,
	}
	return req, merr
}

// ID returns the current call id.
func (c *Call) ID() ID { return c.id }

// Method implements Request.
func (c *Call) Method() string { return c.method }

// Params implements Request.
func (c *Call) Params() json.RawMessage { return c.params }

// jsonrpc2Message implements Request.
func (Call) jsonrpc2Message() {}

// jsonrpc2Request implements Request.
func (Call) jsonrpc2Request() {}

// MarshalJSON implements json.Marshaler.
func (c Call) MarshalJSON() ([]byte, error) {
	req := wireRequest{
		Method: c.method,
		Params: &c.params,
		ID:     &c.id,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return data, fmt.Errorf("marshaling call: %w", err)
	}

	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *Call) UnmarshalJSON(data []byte) error {
	var req wireRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.ZeroCopy()
	if err := dec.Decode(&req); err != nil {
		return fmt.Errorf("unmarshaling call: %w", err)
	}

	c.method = req.Method
	if req.Params != nil {
		c.params = *req.Params
	}
	if req.ID != nil {
		c.id = *req.ID
	}

	return nil
}

// Response is a reply to a Request.
//
// It will have the same ID as the call it is a response to.
type Response struct {
	// result is the content of the response.
	result json.RawMessage
	// err is set only if the call failed.
	err error
	// ID of the request this is a response to.
	id ID
}

// make sure a Response implements the Message, json.Marshaler and json.Unmarshaler and interfaces.
var (
	_ Message          = (*Response)(nil)
	_ json.Marshaler   = (*Response)(nil)
	_ json.Unmarshaler = (*Response)(nil)
)

// NewResponse constructs a new Response message that is a reply to the
// supplied. If err is set result may be ignored.
func NewResponse(id ID, result interface{}, err error) (*Response, error) {
	r, merr := marshalInterface(result)
	resp := &Response{
		id:     id,
		result: r,
		err:    err,
	}
	return resp, merr
}

// ID returns the current response id.
func (r *Response) ID() ID { return r.id }

// Result returns the Response result.
func (r *Response) Result() json.RawMessage { return r.result }

// Err returns the Response error.
func (r *Response) Err() error { return r.err }

// jsonrpc2Message implements Message.
func (r *Response) jsonrpc2Message() {}

// MarshalJSON implements json.Marshaler.
func (r Response) MarshalJSON() ([]byte, error) {
	resp := &wireResponse{
		Error: toError(r.err),
		ID:    &r.id,
	}
	if resp.Error == nil {
		resp.Result = &r.result
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return data, fmt.Errorf("marshaling notification: %w", err)
	}

	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Response) UnmarshalJSON(data []byte) error {
	var resp wireResponse
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.ZeroCopy()
	if err := dec.Decode(&resp); err != nil {
		return fmt.Errorf("unmarshaling jsonrpc response: %w", err)
	}

	if resp.Result != nil {
		r.result = *resp.Result
	}
	if resp.Error != nil {
		r.err = resp.Error
	}
	if resp.ID != nil {
		r.id = *resp.ID
	}

	return nil
}

func toError(err error) *Error {
	if err == nil {
		// no error, the response is complete
		return nil
	}

	var wrapped *Error
	if errors.As(err, &wrapped) {
		// already a wire error, just use it
		return wrapped
	}

	result := &Error{Message: err.Error()}
	if errors.As(err, &wrapped) {
		// if we wrapped a wire error, keep the code from the wrapped error
		// but the message from the outer error
		result.Code = wrapped.Code
	}

	return result
}

// Notification is a request for which a response cannot occur, and as such
// it has not ID.
type Notification struct {
	// Method is a string containing the method name to invoke.
	method string

	params json.RawMessage
}

// make sure a Notification implements the Request, json.Marshaler and json.Unmarshaler and interfaces.
var (
	_ Request          = (*Notification)(nil)
	_ json.Marshaler   = (*Notification)(nil)
	_ json.Unmarshaler = (*Notification)(nil)
)

// NewNotification constructs a new Notification message for the supplied
// method and parameters.
func NewNotification(method string, params interface{}) (*Notification, error) {
	p, merr := marshalInterface(params)
	notify := &Notification{
		method: method,
		params: p,
	}
	return notify, merr
}

// Method implements Request.
func (n *Notification) Method() string { return n.method }

// Params implements Request.
func (n *Notification) Params() json.RawMessage { return n.params }

// jsonrpc2Message implements Request.
func (Notification) jsonrpc2Message() {}

// jsonrpc2Request implements Request.
func (Notification) jsonrpc2Request() {}

// MarshalJSON implements json.Marshaler.
func (n Notification) MarshalJSON() ([]byte, error) {
	req := wireRequest{
		Method: n.method,
		Params: &n.params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return data, fmt.Errorf("marshaling notification: %w", err)
	}

	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *Notification) UnmarshalJSON(data []byte) error {
	var req wireRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.ZeroCopy()
	if err := dec.Decode(&req); err != nil {
		return fmt.Errorf("unmarshaling notification: %w", err)
	}

	n.method = req.Method
	if req.Params != nil {
		n.params = *req.Params
	}

	return nil
}

// DecodeMessage decodes data to Message.
func DecodeMessage(data []byte) (Message, error) {
	var msg combined
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.ZeroCopy()
	if err := dec.Decode(&msg); err != nil {
		return nil, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}

	if msg.Method == "" {
		// no method, should be a response
		if msg.ID == nil {
			return nil, ErrInvalidRequest
		}

		resp := &Response{
			id: *msg.ID,
		}
		if msg.Error != nil {
			resp.err = msg.Error
		}
		if msg.Result != nil {
			resp.result = *msg.Result
		}

		return resp, nil
	}

	// has a method, must be a request
	if msg.ID == nil {
		// request with no ID is a notify
		notify := &Notification{
			method: msg.Method,
		}
		if msg.Params != nil {
			notify.params = *msg.Params
		}

		return notify, nil
	}

	// request with an ID, must be a call
	call := &Call{
		method: msg.Method,
		id:     *msg.ID,
	}
	if msg.Params != nil {
		call.params = *msg.Params
	}

	return call, nil
}

// marshalInterface marshal obj to json.RawMessage.
func marshalInterface(obj interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return json.RawMessage{}, fmt.Errorf("failed to marshal json: %w", err)
	}
	return json.RawMessage(data), nil
}
