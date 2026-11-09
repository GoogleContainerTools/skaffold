// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"fmt"
	"sync"
)

// SyncClient is a synchronous JSON-RPC 2.0 client that owns its read loop: it
// never starts a background read goroutine. Each [SyncClient.Call] writes the
// request and then reads frames on the caller's own goroutine until the matching
// response arrives, so a round trip collapses the dedicated-reader-to-caller
// hand-off that a [Conn] pays (the third goroutine hop). On an in-process
// transport this is the lowest-latency request path the package offers.
//
// The tradeoff is the reason it is a distinct type rather than the default Call
// path: a SyncClient cannot asynchronously receive server-initiated requests
// (there is no background reader to dispatch them), and its calls are
// serialized — exactly one call may be outstanding at a time, guarded by an
// internal mutex. A peer that issues server-to-client calls, or a caller that
// needs concurrent in-flight calls on one connection, must use [Conn] with
// [Conn.Go]. SyncClient is for the common client-drives-server request/reply
// shape over an in-process or point-to-point transport.
//
// A SyncClient requires a [Stream] that also exposes raw frame access (the
// built-in [NewNDJSONStream] and [NewHeaderStream] framers do); a plain Stream
// without frame access is rejected by [NewSyncClient].
type SyncClient struct {
	stream frameStream
	codec  Codec

	mu     sync.Mutex // serializes Call: one outstanding request at a time
	seq    int64      // last allocated call id (guarded by mu)
	closed bool       // Close has been called (guarded by mu)
}

// NewSyncClient creates a [SyncClient] over stream. It returns an error if stream
// does not expose raw frame access (the built-in framers do). The codec defaults
// to [DefaultCodec]; pass [WithCodec] via opts to override it.
func NewSyncClient(stream Stream, opts ...Option) (*SyncClient, error) {
	fs, ok := stream.(frameStream)
	if !ok {
		return nil, fmt.Errorf("jsonrpc2: SyncClient requires a frame-capable Stream, got %T", stream)
	}
	// Reuse the Conn option set to resolve the codec without duplicating WithCodec.
	probe := &conn{codec: DefaultCodec}
	for _, opt := range opts {
		opt(probe)
	}
	return &SyncClient{stream: fs, codec: probe.codec}, nil
}

// Call invokes method on the peer and blocks on the caller's goroutine until the
// matching response arrives, decoding its result into result (which may be nil
// to discard it). params is marshaled with the client's codec; a nil params
// sends no parameters.
//
// Calls are serialized: a second Call blocks until the first returns. Frames
// that arrive before the matching response (a notification or a response to an
// abandoned call) are skipped.
func (c *SyncClient) Call(ctx context.Context, method string, params, result any) (ID, error) {
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return ID{}, fmt.Errorf("jsonrpc2: marshaling call parameters: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ID{}, ErrClientClosing
	}

	c.seq++
	id := NewNumberID(c.seq)

	if err := c.writeCall(ctx, id, method, raw); err != nil {
		return id, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return id, err
		}
		frame, _, ferr := c.stream.ReadFrame(ctx)
		if ferr != nil {
			return id, ferr
		}
		msg, derr := DecodeMessage(frame)
		if derr != nil {
			return id, fmt.Errorf("jsonrpc2: decoding response: %w", derr)
		}
		resp, ok := msg.(*Response)
		if !ok || resp.id != id {
			// A frame that is not our response (a server notification, or a late
			// response to an abandoned call) is skipped; the next frame is read.
			continue
		}
		if resp.err != nil {
			return id, resp.err
		}
		if err := unmarshalResult(c.codec, resp.result, result); err != nil {
			return id, fmt.Errorf("jsonrpc2: unmarshaling result: %w", err)
		}
		return id, nil
	}
}

// Notify sends a notification to the peer without waiting for a response. params
// is marshaled with the client's codec; a nil params sends no parameters.
func (c *SyncClient) Notify(ctx context.Context, method string, params any) error {
	raw, err := marshalParams(c.codec, params)
	if err != nil {
		return fmt.Errorf("jsonrpc2: marshaling notify parameters: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClientClosing
	}
	return c.writeNotification(ctx, method, raw)
}

// writeCall frames and writes a call envelope directly from its concrete fields.
func (c *SyncClient) writeCall(ctx context.Context, id ID, method string, params RawMessage) error {
	if fw, ok := c.stream.(frameWriter); ok {
		_, err := fw.writeCall(ctx, id, method, params)
		return err
	}
	return c.writeFramed(ctx, func(buf []byte) []byte {
		return appendCallFields(buf, id, method, params)
	})
}

// writeNotification frames and writes a notification envelope.
func (c *SyncClient) writeNotification(ctx context.Context, method string, params RawMessage) error {
	if fw, ok := c.stream.(frameWriter); ok {
		_, err := fw.writeNotification(ctx, method, params)
		return err
	}
	return c.writeFramed(ctx, func(buf []byte) []byte {
		return appendNotificationFields(buf, method, params)
	})
}

// writeFramed encodes one envelope into a fresh buffer and writes it via
// WriteFrame, the fallback for a frameStream that does not also implement
// frameWriter.
func (c *SyncClient) writeFramed(ctx context.Context, appendBody func([]byte) []byte) error {
	buf := appendBody(nil)
	_, err := c.stream.WriteFrame(ctx, buf)
	return err
}

// Close closes the underlying stream. After Close, further calls return
// [ErrClientClosing].
func (c *SyncClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if cl, ok := c.stream.(interface{ Close() error }); ok {
		return cl.Close()
	}
	return nil
}
