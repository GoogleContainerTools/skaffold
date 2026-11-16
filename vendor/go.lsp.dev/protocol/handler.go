// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"
	"fmt"

	"go.lsp.dev/jsonrpc2"
)

// CancelHandler returns a [jsonrpc2.Handler] that observes "$/cancelRequest"
// notifications and cancels the in-flight request they name.
func CancelHandler(handler jsonrpc2.Handler) jsonrpc2.Handler {
	handler, canceller := jsonrpc2.CancelHandler(handler)

	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if req.Method() != MethodCancelRequest {
			// TODO(iancottrell): See if we can generate a reply for the request to be
			// cancelled at the point of cancellation rather than waiting for gopls to
			// naturally reply. To do that, we need to keep track of whether a reply has
			// been sent already and be careful about racing between the two paths.
			return handler(ctx, req)
		}

		var params CancelParams
		if err := Unmarshal(req.Params(), &params); err != nil {
			return nil, replyParseError(err)
		}

		switch id := params.ID.(type) {
		case Integer:
			canceller(jsonrpc2.NewNumberID(int64(id)))
		case String:
			canceller(jsonrpc2.NewStringID(string(id)))
		default:
			return nil, replyParseError(fmt.Errorf("malformed cancel id %v", params.ID))
		}

		return nil, nil
	}
}

// Handlers wraps handler with the standard LSP middleware chain: cancellation
// and asynchronous dispatch.
func Handlers(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return CancelHandler(jsonrpc2.AsyncHandler(handler))
}

// Call invokes method on conn with params, decoding the response into result. If
// ctx is canceled while the call is outstanding, a "$/cancelRequest" notification
// is sent for the call's id.
func Call(ctx context.Context, conn jsonrpc2.Conn, method string, params, result any) error {
	id, err := conn.Call(ctx, method, params, result)
	if ctx.Err() != nil {
		notifyCancel(ctx, conn, id)
	}

	return err
}

// notifyCancel sends a "$/cancelRequest" notification for id over a detached
// context so the cancellation is delivered even though the caller's context is
// already done.
func notifyCancel(ctx context.Context, conn jsonrpc2.Conn, id jsonrpc2.ID) {
	ctx = context.WithoutCancel(ctx)
	// The notification is best-effort: the request may already have completed.
	_ = conn.Notify(ctx, MethodCancelRequest, &CancelParams{ID: idToProgressToken(id)})
}

// idToProgressToken converts a jsonrpc2 request id into the [ProgressToken] union
// carried by [CancelParams].
func idToProgressToken(id jsonrpc2.ID) ProgressToken {
	if n, ok := id.Number(); ok {
		return Integer(n) //nolint:gosec // LSP request IDs are within the int32 range
	}
	s, _ := id.StringValue()

	return String(s)
}

// replyParseError returns a parse error wrapping err.
func replyParseError(err error) error {
	return fmt.Errorf("%w: %w", jsonrpc2.ErrParse, err)
}
