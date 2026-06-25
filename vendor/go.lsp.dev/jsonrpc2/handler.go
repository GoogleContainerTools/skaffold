// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"fmt"
	"sync"
)

// Handler is invoked to handle incoming requests.
//
// The Replier sends a reply to the request and must be called exactly once.
type Handler func(ctx context.Context, reply Replier, req Request) error

// Replier is passed to handlers to allow them to reply to the request.
//
// If err is set then result will be ignored.
type Replier func(ctx context.Context, result interface{}, err error) error

// MethodNotFoundHandler is a Handler that replies to all call requests with the
// standard method not found response.
//
// This should normally be the final handler in a chain.
func MethodNotFoundHandler(ctx context.Context, reply Replier, req Request) error {
	return reply(ctx, nil, fmt.Errorf("%q: %w", req.Method(), ErrMethodNotFound))
}

// ReplyHandler creates a Handler that panics if the wrapped handler does
// not call Reply for every request that it is passed.
func ReplyHandler(handler Handler) (h Handler) {
	h = Handler(func(ctx context.Context, reply Replier, req Request) error {
		called := false
		err := handler(ctx, func(ctx context.Context, result interface{}, err error) error {
			if called {
				panic(fmt.Errorf("request %q replied to more than once", req.Method()))
			}
			called = true

			return reply(ctx, result, err)
		}, req)
		if !called {
			panic(fmt.Errorf("request %q was never replied to", req.Method()))
		}
		return err
	})

	return h
}

// CancelHandler returns a handler that supports cancellation, and a function
// that can be used to trigger canceling in progress requests.
func CancelHandler(handler Handler) (h Handler, canceller func(id ID)) {
	var mu sync.Mutex
	handling := make(map[ID]context.CancelFunc)

	h = Handler(func(ctx context.Context, reply Replier, req Request) error {
		if call, ok := req.(*Call); ok {
			cancelCtx, cancel := context.WithCancel(ctx)
			ctx = cancelCtx

			mu.Lock()
			handling[call.ID()] = cancel
			mu.Unlock()

			innerReply := reply
			reply = func(ctx context.Context, result interface{}, err error) error {
				mu.Lock()
				delete(handling, call.ID())
				mu.Unlock()
				return innerReply(ctx, result, err)
			}
		}
		return handler(ctx, reply, req)
	})

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

// AsyncHandler returns a handler that processes each request goes in its own
// goroutine.
//
// The handler returns immediately, without the request being processed.
// Each request then waits for the previous request to finish before it starts.
//
// This allows the stream to unblock at the cost of unbounded goroutines
// all stalled on the previous one.
func AsyncHandler(handler Handler) (h Handler) {
	nextRequest := make(chan struct{})
	close(nextRequest)

	h = Handler(func(ctx context.Context, reply Replier, req Request) error {
		waitForPrevious := nextRequest
		nextRequest = make(chan struct{})
		unlockNext := nextRequest
		innerReply := reply
		reply = func(ctx context.Context, result interface{}, err error) error {
			close(unlockNext)
			return innerReply(ctx, result, err)
		}

		go func() {
			<-waitForPrevious
			_ = handler(ctx, reply, req)
		}()
		return nil
	})

	return h
}
