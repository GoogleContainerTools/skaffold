// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !gojay
// +build !gojay

package protocol

import (
	"context"
	"encoding/json"
	"fmt"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/pkg/xcontext"
)

// CancelHandler handler of cancelling.
func CancelHandler(handler jsonrpc2.Handler) jsonrpc2.Handler {
	handler, canceller := jsonrpc2.CancelHandler(handler)

	h := func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if req.Method() != MethodCancelRequest {
			// TODO(iancottrell): See if we can generate a reply for the request to be cancelled
			// at the point of cancellation rather than waiting for gopls to naturally reply.
			// To do that, we need to keep track of whether a reply has been sent already and
			// be careful about racing between the two paths.
			// TODO(iancottrell): Add a test that watches the stream and verifies the response
			// for the cancelled request flows.
			reply := func(ctx context.Context, resp interface{}, err error) error {
				// https://microsoft.github.io/language-server-protocol/specifications/specification-current/#cancelRequest
				if ctx.Err() != nil && err == nil {
					err = ErrRequestCancelled
				}
				ctx = xcontext.Detach(ctx)

				return reply(ctx, resp, err)
			}

			return handler(ctx, reply, req)
		}

		var params CancelParams
		if err := json.Unmarshal(req.Params(), &params); err != nil {
			return replyParseError(ctx, reply, err)
		}

		switch id := params.ID.(type) {
		case int32:
			canceller(jsonrpc2.NewNumberID(id))
		case string:
			canceller(jsonrpc2.NewStringID(id))
		default:
			return replyParseError(ctx, reply, fmt.Errorf("request ID %v malformed", id))
		}

		return reply(ctx, nil, nil)
	}

	return h
}
