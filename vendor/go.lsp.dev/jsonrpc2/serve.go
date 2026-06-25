// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// NOTE: This file provides the network-serving surface that downstream
// consumers (such as go.lsp.dev/protocol) build on. It is intentionally similar
// to net/http: a [StreamServer] serves one accepted connection at a time, and
// [Serve] accepts connections from a [net.Listener] and hands each to the
// server on its own goroutine.

// StreamServer is used to serve incoming jsonrpc2 clients communicating over a
// newly created connection.
//
// ServeStream is called once per accepted connection, on its own goroutine, with
// a [Conn] wrapping that connection. It should drive the connection (typically by
// calling [Conn.Go] and waiting on [Conn.Done]) and return when the connection is
// finished. The connection's stream is closed by [Serve] after ServeStream
// returns.
type StreamServer interface {
	ServeStream(context.Context, Conn) error
}

// ServerFunc is an adapter that implements the [StreamServer] interface using an
// ordinary function.
type ServerFunc func(context.Context, Conn) error

// ServeStream implements [StreamServer] by calling f(ctx, c).
func (f ServerFunc) ServeStream(ctx context.Context, c Conn) error {
	return f(ctx, c)
}

// HandlerServer returns a [StreamServer] that serves each incoming connection by
// dispatching its requests to h.
//
// For each connection it starts the read goroutine with [Conn.Go], waits for the
// connection to terminate with [Conn.Done], and reports [Conn.Err].
func HandlerServer(h Handler) StreamServer {
	return ServerFunc(func(ctx context.Context, conn Conn) error {
		conn.Go(ctx, h)
		<-conn.Done()
		return conn.Err()
	})
}

// ListenAndServe starts a jsonrpc2 server on the given network address.
//
// It listens on network and addr, then serves accepted connections with
// [Serve]. The listener is closed when ListenAndServe returns; for a "unix"
// network the socket file is also removed. If idleTimeout is non-zero,
// ListenAndServe returns [ErrIdleTimeout] after there have been no connections
// for that duration; otherwise it returns only on error or when ctx is canceled.
func ListenAndServe(ctx context.Context, network, addr string, server StreamServer, idleTimeout time.Duration) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("failed to listen %s:%s: %w", network, addr, err)
	}
	defer ln.Close()

	if network == "unix" {
		defer os.Remove(addr)
	}

	return Serve(ctx, ln, server, idleTimeout)
}

// Serve accepts incoming connections from ln and serves each with server on its
// own goroutine.
//
// Serve returns when:
//   - the listener fails to accept (the accept error is returned);
//   - idleTimeout is non-zero and no connection has been active for that
//     duration ([ErrIdleTimeout] is returned); or
//   - ctx is canceled (ctx.Err() is returned).
//
// On any of these, Serve stops accepting, closes ln to unblock the accept
// goroutine, closes the streams of any still-active connections so their
// [StreamServer] goroutines unwind, waits for all of them to return, and only
// then returns. Serve therefore leaks no goroutine: every goroutine it starts has
// exited by the time it returns. Because Serve closes ln on return, the caller
// need not (a second Close is harmless).
func Serve(ctx context.Context, ln net.Listener, server StreamServer, idleTimeout time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Max duration: ~290 years; surely that is long enough.
	const forever = 1<<63 - 1
	if idleTimeout <= 0 {
		idleTimeout = forever
	}
	connTimer := time.NewTimer(idleTimeout)
	defer connTimer.Stop()

	newConns := make(chan net.Conn)
	doneListening := make(chan error, 1)

	// closedConns signals the accept loop that a connection finished and the active
	// set became idle, so it can re-arm the idle timer. A buffer of one plus a
	// non-blocking send means a finishing server goroutine never blocks: if a
	// signal is already pending the send is dropped, and the loop reads the
	// up-to-date active count when it wakes.
	closedConns := make(chan struct{}, 1)

	var (
		acceptWG sync.WaitGroup // the accept goroutine
		connWG   sync.WaitGroup // per-connection server goroutines

		// active tracks the streams of connections still being served so that, on
		// shutdown, Serve can close them to unblock their server goroutines.
		mu     sync.Mutex
		active = map[Stream]struct{}{}
	)

	acceptWG.Go(func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				// Report the accept error unless the accept loop is being torn down,
				// in which case the error is the expected consequence of closing ln.
				select {
				case doneListening <- fmt.Errorf("accept: %w", err):
				case <-ctx.Done():
				}
				return
			}
			select {
			case newConns <- nc:
			case <-ctx.Done():
				// Serve has stopped accepting; do not leak the just-accepted conn.
				nc.Close()
				return
			}
		}
	})

	// shutdown stops accepting, closes ln to unblock ln.Accept, closes every
	// active stream so its server goroutine unwinds, and waits for all goroutines.
	shutdown := func() {
		cancel()
		ln.Close()
		mu.Lock()
		for s := range active {
			s.Close()
		}
		mu.Unlock()
		acceptWG.Wait()
		connWG.Wait()
	}

	for {
		select {
		case netConn := <-newConns:
			connTimer.Stop()
			stream := NewStream(netConn)
			mu.Lock()
			active[stream] = struct{}{}
			mu.Unlock()

			connWG.Go(func() {
				conn := NewConn(stream)
				_ = server.ServeStream(ctx, conn)
				stream.Close()

				mu.Lock()
				delete(active, stream)
				idle := len(active) == 0
				mu.Unlock()

				if idle {
					// Signal the accept loop so it can re-arm the idle timer. The send
					// is non-blocking: if a signal is already pending the loop will
					// observe the up-to-date active count when it wakes.
					select {
					case closedConns <- struct{}{}:
					default:
					}
				}
			})

		case err := <-doneListening:
			shutdown()
			return err

		case <-closedConns:
			mu.Lock()
			idle := len(active) == 0
			mu.Unlock()
			if idle {
				connTimer.Reset(idleTimeout)
			}

		case <-connTimer.C:
			mu.Lock()
			idle := len(active) == 0
			mu.Unlock()
			if idle {
				shutdown()
				return ErrIdleTimeout
			}
			// A connection arrived after the timer fired but before this case ran;
			// the timer is re-armed when the connection set next becomes idle.

		case <-ctx.Done():
			shutdown()
			return ctx.Err()
		}
	}
}
