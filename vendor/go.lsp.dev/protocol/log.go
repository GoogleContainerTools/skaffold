// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"go.lsp.dev/jsonrpc2"
)

// loggingStream represents a logging of jsonrpc2.Stream.
type loggingStream struct {
	stream jsonrpc2.Stream
	log    io.Writer
	logMu  sync.Mutex
}

// LoggingStream returns a stream that does LSP protocol logging.
func LoggingStream(stream jsonrpc2.Stream, w io.Writer) jsonrpc2.Stream {
	return &loggingStream{
		stream: stream,
		log:    w,
	}
}

// Read implements jsonrpc2.Stream.Read.
func (s *loggingStream) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	msg, count, err := s.stream.Read(ctx)
	if err == nil {
		s.logCommon(msg, true)
	}

	return msg, count, err
}

// Write implements jsonrpc2.Stream.Write.
func (s *loggingStream) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	s.logCommon(msg, false)
	count, err := s.stream.Write(ctx, msg)

	return count, err
}

// Close implements jsonrpc2.Stream.Close.
func (s *loggingStream) Close() error {
	return s.stream.Close()
}

type req struct {
	method string
	start  time.Time
}

type mapped struct {
	mu          sync.Mutex
	clientCalls map[string]req
	serverCalls map[string]req
}

var maps = &mapped{
	mu:          sync.Mutex{},
	clientCalls: make(map[string]req),
	serverCalls: make(map[string]req),
}

// these 4 methods are each used exactly once, but it seemed
// better to have the encapsulation rather than ad hoc mutex
// code in 4 places.
func (m *mapped) client(id string) req {
	m.mu.Lock()
	v := m.clientCalls[id]
	delete(m.clientCalls, id)
	m.mu.Unlock()

	return v
}

func (m *mapped) server(id string) req {
	m.mu.Lock()
	v := m.serverCalls[id]
	delete(m.serverCalls, id)
	m.mu.Unlock()

	return v
}

func (m *mapped) setClient(id string, r req) {
	m.mu.Lock()
	m.clientCalls[id] = r
	m.mu.Unlock()
}

func (m *mapped) setServer(id string, r req) {
	m.mu.Lock()
	m.serverCalls[id] = r
	m.mu.Unlock()
}

const eor = "\r\n\r\n\r\n"

func (s *loggingStream) logCommon(msg jsonrpc2.Message, isRead bool) {
	if msg == nil || s.log == nil {
		return
	}

	s.logMu.Lock()

	direction, pastTense := "Received", "Received"
	get, set := maps.client, maps.setServer
	if isRead {
		direction, pastTense = "Sending", "Sent"
		get, set = maps.server, maps.setClient
	}

	tm := time.Now()
	tmfmt := tm.Format("15:04:05.000 PM")

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "[Trace - %s] ", tmfmt) // common beginning

	switch msg := msg.(type) {
	case *jsonrpc2.Call:
		id := fmt.Sprint(msg.ID())
		fmt.Fprintf(&buf, "%s request '%s - (%s)'.\n", direction, msg.Method(), id)
		fmt.Fprintf(&buf, "Params: %s%s", msg.Params(), eor)
		set(id, req{method: msg.Method(), start: tm})

	case *jsonrpc2.Notification:
		fmt.Fprintf(&buf, "%s notification '%s'.\n", direction, msg.Method())
		fmt.Fprintf(&buf, "Params: %s%s", msg.Params(), eor)

	case *jsonrpc2.Response:
		id := fmt.Sprint(msg.ID())
		if err := msg.Err(); err != nil {
			fmt.Fprintf(s.log, "[Error - %s] %s #%s %s%s", pastTense, tmfmt, id, err, eor)

			return
		}

		cc := get(id)
		elapsed := tm.Sub(cc.start)
		fmt.Fprintf(&buf, "%s response '%s - (%s)' in %dms.\n",
			direction, cc.method, id, elapsed/time.Millisecond)
		fmt.Fprintf(&buf, "Result: %s%s", msg.Result(), eor)
	}

	s.log.Write(buf.Bytes())

	s.logMu.Unlock()
}
