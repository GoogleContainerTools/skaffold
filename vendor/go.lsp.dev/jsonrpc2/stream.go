// SPDX-FileCopyrightText: 2018 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"bufio"
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/segmentio/encoding/json"
)

const (
	// HdrContentLength is the HTTP header name of the length of the content part in bytes. This header is required.
	// This entity header indicates the size of the entity-body, in bytes, sent to the recipient.
	//
	// RFC 7230, section 3.3.2: Content-Length:
	//  https://tools.ietf.org/html/rfc7230#section-3.3.2
	HdrContentLength = "Content-Length"

	// HeaderContentType is the mime type of the content part. Defaults to "application/vscode-jsonrpc; charset=utf-8".
	// This entity header is used to indicate the media type of the resource.
	//
	// RFC 7231, section 3.1.1.5: Content-Type:
	//  https://tools.ietf.org/html/rfc7231#section-3.1.1.5
	HdrContentType = "Content-Type"

	// HeaderContentSeparator is the header and content part separator.
	HdrContentSeparator = "\r\n\r\n"
)

// Framer wraps a network connection up into a Stream.
//
// It is responsible for the framing and encoding of messages into wire form.
// NewRawStream and NewStream are implementations of a Framer.
type Framer func(conn io.ReadWriteCloser) Stream

// Stream abstracts the transport mechanics from the JSON RPC protocol.
//
// A Conn reads and writes messages using the stream it was provided on
// construction, and assumes that each call to Read or Write fully transfers
// a single message, or returns an error.
//
// A stream is not safe for concurrent use, it is expected it will be used by
// a single Conn in a safe manner.
type Stream interface {
	// Read gets the next message from the stream.
	Read(context.Context) (Message, int64, error)

	// Write sends a message to the stream.
	Write(context.Context, Message) (int64, error)

	// Close closes the connection.
	// Any blocked Read or Write operations will be unblocked and return errors.
	Close() error
}

type rawStream struct {
	conn io.ReadWriteCloser
	in   *stdjson.Decoder
}

// NewRawStream returns a Stream built on top of a io.ReadWriteCloser.
//
// The messages are sent with no wrapping, and rely on json decode consistency
// to determine message boundaries.
func NewRawStream(conn io.ReadWriteCloser) Stream {
	return &rawStream{
		conn: conn,
		in:   stdjson.NewDecoder(conn), // TODO(zchee): why test fail using segmentio json.Decoder?
	}
}

// Read implements Stream.Read.
func (s *rawStream) Read(ctx context.Context) (Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	var raw stdjson.RawMessage
	if err := s.in.Decode(&raw); err != nil {
		return nil, 0, fmt.Errorf("decoding raw message: %w", err)
	}

	msg, err := DecodeMessage(raw)
	return msg, int64(len(raw)), err
}

// Write implements Stream.Write.
func (s *rawStream) Write(ctx context.Context, msg Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshaling message: %w", err)
	}

	n, err := s.conn.Write(data)
	if err != nil {
		return 0, fmt.Errorf("write to stream: %w", err)
	}

	return int64(n), nil
}

// Close implements Stream.Close.
func (s *rawStream) Close() error {
	return s.conn.Close()
}

type stream struct {
	conn io.ReadWriteCloser
	in   *bufio.Reader
}

// NewStream returns a Stream built on top of a io.ReadWriteCloser.
//
// The messages are sent with HTTP content length and MIME type headers.
// This is the format used by LSP and others.
func NewStream(conn io.ReadWriteCloser) Stream {
	return &stream{
		conn: conn,
		in:   bufio.NewReader(conn),
	}
}

// Read implements Stream.Read.
func (s *stream) Read(ctx context.Context) (Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	var total int64
	var length int64
	// read the header, stop on the first empty line
	for {
		line, err := s.in.ReadString('\n')
		total += int64(len(line))
		if err != nil {
			return nil, total, fmt.Errorf("failed reading header line: %w", err)
		}

		line = strings.TrimSpace(line)
		// check we have a header line
		if line == "" {
			break
		}

		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			return nil, total, fmt.Errorf("invalid header line %q", line)
		}

		name, value := line[:colon], strings.TrimSpace(line[colon+1:])
		switch name {
		case HdrContentLength:
			if length, err = strconv.ParseInt(value, 10, 32); err != nil {
				return nil, total, fmt.Errorf("failed parsing %s: %v: %w", HdrContentLength, value, err)
			}
			if length <= 0 {
				return nil, total, fmt.Errorf("invalid %s: %v", HdrContentLength, length)
			}
		default:
			// ignoring unknown headers
		}
	}

	if length == 0 {
		return nil, total, fmt.Errorf("missing %s header", HdrContentLength)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(s.in, data); err != nil {
		return nil, total, fmt.Errorf("read full of data: %w", err)
	}

	total += length
	msg, err := DecodeMessage(data)
	return msg, total, err
}

// Write implements Stream.Write.
func (s *stream) Write(ctx context.Context, msg Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("marshaling message: %w", err)
	}

	n, err := fmt.Fprintf(s.conn, "%s: %v%s", HdrContentLength, len(data), HdrContentSeparator)
	total := int64(n)
	if err != nil {
		return 0, fmt.Errorf("write data to conn: %w", err)
	}

	n, err = s.conn.Write(data)
	total += int64(n)
	if err != nil {
		return 0, fmt.Errorf("write data to conn: %w", err)
	}

	return total, nil
}

// Close implements Stream.Close.
func (s *stream) Close() error {
	return s.conn.Close()
}
