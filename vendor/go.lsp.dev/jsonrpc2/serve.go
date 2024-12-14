// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

// NOTE: This file provides an experimental API for serving multiple remote
// jsonrpc2 clients over the network. For now, it is intentionally similar to
// net/http, but that may change in the future as we figure out the correct
// semantics.

// StreamServer is used to serve incoming jsonrpc2 clients communicating over
// a newly created connection.
type StreamServer interface {
	ServeStream(context.Context, Conn) error
}

// ServerFunc is an adapter that implements the StreamServer interface
// using an ordinary function.
type ServerFunc func(context.Context, Conn) error

// ServeStream implements StreamServer.
//
// ServeStream calls f(ctx, s).
func (f ServerFunc) ServeStream(ctx context.Context, c Conn) error {
	return f(ctx, c)
}

// HandlerServer returns a StreamServer that handles incoming streams using the
// provided handler.
func HandlerServer(h Handler) StreamServer {
	return ServerFunc(func(ctx context.Context, conn Conn) error {
		conn.Go(ctx, h)
		<-conn.Done()
		return conn.Err()
	})
}

// ListenAndServe starts an jsonrpc2 server on the given address.
//
// If idleTimeout is non-zero, ListenAndServe exits after there are no clients for
// this duration, otherwise it exits only on error.
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

// Serve accepts incoming connections from the network, and handles them using
// the provided server. If idleTimeout is non-zero, ListenAndServe exits after
// there are no clients for this duration, otherwise it exits only on error.
func Serve(ctx context.Context, ln net.Listener, server StreamServer, idleTimeout time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Max duration: ~290 years; surely that's long enough.
	const forever = 1<<63 - 1
	if idleTimeout <= 0 {
		idleTimeout = forever
	}
	connTimer := time.NewTimer(idleTimeout)

	newConns := make(chan net.Conn)
	doneListening := make(chan error)
	closedConns := make(chan error)

	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				select {
				case doneListening <- fmt.Errorf("accept: %w", err):
				case <-ctx.Done():
				}
				return
			}

			newConns <- nc
		}
	}()

	activeConns := 0
	for {
		select {
		case netConn := <-newConns:
			activeConns++
			connTimer.Stop()
			stream := NewStream(netConn)
			go func() {
				conn := NewConn(stream)
				closedConns <- server.ServeStream(ctx, conn)
				stream.Close()
			}()

		case err := <-doneListening:
			return err

		case <-closedConns:
			// if !isClosingError(err) {
			// }

			activeConns--
			if activeConns == 0 {
				connTimer.Reset(idleTimeout)
			}

		case <-connTimer.C:
			return ErrIdleTimeout

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
