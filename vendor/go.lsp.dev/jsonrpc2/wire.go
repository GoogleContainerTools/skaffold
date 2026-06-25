// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

// envelopePrefix is the shared, constant head of every JSON-RPC envelope.
const envelopePrefix = `{"jsonrpc":"2.0"`

// callWire, notificationWire, and responseWire are the internal, allocation-
// free wire representations used by the connection hot path. They are value
// types so callers can pass them through interfaces without forcing a heap
// allocation, while the public *Call, *Notification, and *Response types remain
// available for the external API and decode results.
type callWire struct {
	id     ID
	method string
	params RawMessage
}

type notificationWire struct {
	method string
	params RawMessage
}

type responseWire struct {
	err    error
	id     ID
	result RawMessage
}

func (callWire) jsonrpc2Message() {}

func (notificationWire) jsonrpc2Message() {}

func (responseWire) jsonrpc2Message() {}

// EncodeMessage encodes a [Message] into a freshly allocated JSON-RPC envelope.
//
// The envelope is built by appending directly into a pooled buffer with no
// reflection: method names and string identifiers are written through a
// fast-escape routine, integer identifiers through strconv, and params/result
// raw values verbatim. The returned slice is a right-sized copy that the caller
// owns; the pooled buffer is recycled before returning.
func EncodeMessage(msg Message) ([]byte, error) {
	bp := getEncodeBuf()
	buf := appendMessage(*bp, msg)
	out := make([]byte, len(buf))
	copy(out, buf)
	*bp = buf
	putEncodeBuf(bp)
	return out, nil
}

// AppendMessage appends msg's JSON-RPC envelope to dst and returns the extended
// slice.
//
// Unlike [EncodeMessage], AppendMessage does not allocate an owned right-sized
// result. The returned bytes alias dst's backing array, making this the preferred
// API for callers that own an output buffer or write batch envelopes directly.
func AppendMessage(dst []byte, msg Message) []byte {
	return appendMessage(dst, msg)
}

// AppendCall appends a JSON-RPC call envelope to dst.
func AppendCall(dst []byte, id ID, method string, params RawMessage) []byte {
	return appendCallFields(dst, id, method, params)
}

// AppendNotification appends a JSON-RPC notification envelope to dst.
func AppendNotification(dst []byte, method string, params RawMessage) []byte {
	return appendNotificationFields(dst, method, params)
}

// AppendResponse appends a JSON-RPC response envelope to dst.
func AppendResponse(dst []byte, id ID, result RawMessage, err error) []byte {
	return appendResponseFields(dst, id, result, err)
}

// AppendBatch appends a JSON-RPC batch array containing msgs to dst.
//
// The function appends exactly the messages it is given. A valid JSON-RPC batch
// request contains at least one request/notification member, and a valid batch
// response contains at least one response member; callers that need to enforce
// those protocol roles should do so before calling AppendBatch.
func AppendBatch(dst []byte, msgs []Message) []byte {
	dst = append(dst, '[')
	first := true
	for _, msg := range msgs {
		temp := dst
		if !first {
			temp = append(temp, ',')
		}
		next := appendMessage(temp, msg)
		if len(next) > len(temp) {
			dst = next
			first = false
		}
	}
	return append(dst, ']')
}

// appendMessage appends the wire envelope of msg to dst and returns the extended
// slice. It dispatches on the concrete message type; the [Message] set is closed
// so the default case is unreachable for well-formed values.
func appendMessage(dst []byte, msg Message) []byte {
	switch m := msg.(type) {
	case *Call:
		return appendCallFields(dst, m.id, m.method, m.params)
	case *Notification:
		return appendNotificationFields(dst, m.method, m.params)
	case *Response:
		return appendResponseFields(dst, m.id, m.result, m.err)
	case callWire:
		return appendCallFields(dst, m.id, m.method, m.params)
	case notificationWire:
		return appendNotificationFields(dst, m.method, m.params)
	case responseWire:
		return appendResponseFields(dst, m.id, m.result, m.err)
	default:
		return dst
	}
}

// appendCallFields appends a call envelope from its concrete fields:
//
//	{"jsonrpc":"2.0","method":<esc>,"params":<raw?>,"id":<id>}
func appendCallFields(dst []byte, id ID, method string, params RawMessage) []byte {
	dst = append(dst, envelopePrefix...)
	dst = append(dst, `,"method":`...)
	dst = appendQuotedString(dst, method)
	if len(params) > 0 {
		dst = append(dst, `,"params":`...)
		dst = append(dst, params...)
	}
	dst = append(dst, `,"id":`...)
	dst = id.appendID(dst)
	return append(dst, '}')
}

// appendNotificationFields appends a notification envelope (no id member) from
// its concrete fields:
//
//	{"jsonrpc":"2.0","method":<esc>,"params":<raw?>}
func appendNotificationFields(dst []byte, method string, params RawMessage) []byte {
	dst = append(dst, envelopePrefix...)
	dst = append(dst, `,"method":`...)
	dst = appendQuotedString(dst, method)
	if len(params) > 0 {
		dst = append(dst, `,"params":`...)
		dst = append(dst, params...)
	}
	return append(dst, '}')
}

func appendResponseBatch(dst []byte, resps []responseWire) []byte {
	dst = append(dst, '[')
	for i, resp := range resps {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendResponseFields(dst, resp.id, resp.result, resp.err)
	}
	return append(dst, ']')
}

// appendResponseFields appends a response envelope from its concrete fields. Per
// the specification a response always carries an id (null when unknown) and
// exactly one of result or error; a successful response always emits a result
// member, defaulting to null when no result bytes are present.
//
//	{"jsonrpc":"2.0","id":<id>,"result":<raw|null>}
//	{"jsonrpc":"2.0","id":<id>,"error":{...}}
func appendResponseFields(dst []byte, id ID, result RawMessage, err error) []byte {
	dst = append(dst, envelopePrefix...)
	dst = append(dst, `,"id":`...)
	dst = id.appendID(dst)
	if err != nil {
		dst = append(dst, `,"error":`...)
		dst = appendError(dst, toWireError(err))
		return append(dst, '}')
	}
	dst = append(dst, `,"result":`...)
	if len(result) > 0 {
		dst = append(dst, result...)
	} else {
		dst = append(dst, 'n', 'u', 'l', 'l')
	}
	return append(dst, '}')
}
