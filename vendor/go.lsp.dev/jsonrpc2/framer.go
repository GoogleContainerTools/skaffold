// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import (
	"bufio"
	"context"
	"io"
	"sync"
)

// Stream is a bidirectional channel of JSON-RPC messages over a single
// connection. A Stream adapts a byte transport (the framing) to the message
// boundary: Read decodes the next framed [Message] and Write frames and emits
// one.
//
// The int64 returned by Read and Write is the number of wire bytes transferred
// for that message, including any framing overhead (the LSP header, or the
// newline delimiter for ndjson). It is informational and is intended for
// transport accounting; a zero count accompanies an error.
//
// A Stream owns the underlying connection: Close closes it. Read is expected to
// be driven from a single goroutine (the connection's read loop). Write is safe
// for concurrent use; concurrent writers are serialized so that each message is
// emitted as one contiguous frame.
type Stream interface {
	// Read decodes the next message from the stream. It returns [io.EOF] when
	// the peer closes the connection at a clean frame boundary, and
	// [io.ErrUnexpectedEOF] when the connection ends in the middle of a frame.
	//
	// A decoded request's method and params borrow the stream's read buffer
	// and are valid only until the next Read; copy what you retain. Responses
	// own their bytes.
	Read(ctx context.Context) (Message, int64, error)

	// Write frames msg and writes it to the stream as a single contiguous write.
	Write(ctx context.Context, msg Message) (int64, error)

	// Close closes the underlying connection.
	Close() error
}

// Framer adapts a byte-oriented connection into a message-oriented [Stream] by
// supplying the wire framing. The same Framer value can wrap many connections.
type Framer func(conn io.ReadWriteCloser) Stream

// ErrInvalidHeader is returned by the LSP header framer when a frame's header
// block is malformed: it lacks the terminating blank line, contains a field
// without a colon, or declares a missing, non-numeric, zero, negative, or
// oversized Content-Length (above [maxContentLength]).
const ErrInvalidHeader = constError("jsonrpc2: invalid message header")

// NewStream returns a [Stream] over conn using the LSP base-protocol framing
// (an HTTP-like header carrying Content-Length, a blank line, then the JSON
// body). It is the framing used by the Language Server Protocol.
//
// NewStream is an alias for [NewHeaderStream]; the latter name states the
// framing explicitly.
func NewStream(conn io.ReadWriteCloser) Stream { return NewHeaderStream(conn) }

// NewHeaderStream returns a [Stream] over conn using the LSP header framing:
//
//	Content-Length: <n>\r\n
//	[Content-Type: ...\r\n]
//	\r\n
//	<n bytes of JSON>
//
// Unknown header fields are ignored. A frame with a missing, zero, or negative
// Content-Length is rejected. The header and body are composed into a single
// pooled buffer and emitted with one Write per message.
func NewHeaderStream(conn io.ReadWriteCloser) Stream {
	return &headerStream{
		conn: conn,
		in:   bufio.NewReader(conn),
	}
}

// NewRawStream returns a [Stream] over conn using newline-delimited JSON
// (ndjson) framing: each message is one JSON value followed by a single '\n'.
// This framing is compatible with the Model Context Protocol stdio transport.
//
// NewRawStream is an alias for [NewNDJSONStream]; it preserves the name used by
// the gopls-derived API.
func NewRawStream(conn io.ReadWriteCloser) Stream { return NewNDJSONStream(conn) }

// NewNDJSONStream returns a [Stream] over conn using newline-delimited JSON
// framing: each message is one JSON value followed by a single '\n', and the
// payload plus its delimiter are written with one Write per message.
//
// The framing assumes compact payloads: a literal newline inside a message
// would be read as a frame boundary. The wire encoder ([EncodeMessage])
// escapes control characters, so the strings it controls (method names, string
// identifiers, error messages) never embed a raw newline; callers that supply
// pre-encoded params or result bytes must keep them newline-free, which compact
// JSON always is.
func NewNDJSONStream(conn io.ReadWriteCloser) Stream {
	return &ndjsonStream{
		conn: conn,
		in:   bufio.NewReader(conn),
	}
}

// headerStream implements [Stream] with LSP header framing.
type headerStream struct {
	conn    io.ReadWriteCloser
	in      *bufio.Reader
	rbuf    []byte     // reusable body buffer, re-sized per frame
	wbuf    []byte     // reusable header+body compose buffer, guarded by writeMu
	writeMu sync.Mutex // serializes Write so each frame is one contiguous write
}

// ndjsonStream implements [Stream] with newline-delimited JSON framing.
type ndjsonStream struct {
	conn    io.ReadWriteCloser
	in      *bufio.Reader
	wbuf    []byte     // reusable payload+'\n' compose buffer, guarded by writeMu
	writeMu sync.Mutex // serializes Write so payload+'\n' is one contiguous write
}

// Read implements [Stream]. It parses one LSP header block, reads the declared
// body into a reused buffer, and decodes the message. A decoded request's
// method and params borrow that buffer and are valid only until the next
// read; copy what you retain.
func (s *headerStream) Read(ctx context.Context) (Message, int64, error) {
	body, n, err := s.ReadFrame(ctx)
	if err != nil {
		return nil, 0, err
	}

	msg, err := DecodeMessage(body)
	if err != nil {
		return nil, 0, err
	}
	return msg, n, nil
}

// ReadFrame implements the internal frameStream interface. It returns the raw
// JSON body of the next frame without decoding it, so the connection can
// classify single messages from batch arrays on the fast path. The returned
// slice points into a reusable buffer and is valid only until the next read; a
// caller that retains the bytes must copy them.
func (s *headerStream) ReadFrame(ctx context.Context) ([]byte, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	length, headerLen, err := readContentLength(s.in)
	if err != nil {
		return nil, 0, err
	}

	if cap(s.rbuf) < length {
		s.rbuf = make([]byte, length)
	}
	body := s.rbuf[:length]
	if _, err := io.ReadFull(s.in, body); err != nil {
		// A short body after a complete header is a truncated frame.
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, 0, err
	}
	return body, int64(headerLen) + int64(length), nil
}

// Write implements [Stream]. It composes the Content-Length header and encoded
// body directly into the stream's write buffer and emits them with one
// conn.Write. This avoids the intermediate owned body allocation made by
// EncodeMessage, while preserving the single-write framing guarantee.
func (s *headerStream) Write(ctx context.Context, msg Message) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	return s.composeAndWrite(func(buf []byte) []byte {
		return appendMessage(buf, msg)
	})
}

// composeAndWrite frames a single body, where appendBody appends the encoded
// JSON envelope to the body region of the reused, writeMu-guarded compose
// buffer. It reserves room for the Content-Length header, lets appendBody fill
// the body, then writes the header prefix and shifts the body so the header and
// body are emitted with one conn.Write. The caller must hold writeMu.
func (s *headerStream) composeAndWrite(appendBody func(buf []byte) []byte) (int64, error) {
	const (
		contentLengthPrefix = "Content-Length: "
		headerSuffix        = "\r\n\r\n"
		headerReserve       = len(contentLengthPrefix) + 20 + len(headerSuffix)
	)

	buf := s.wbuf[:0]
	if cap(buf) < headerReserve {
		buf = make([]byte, headerReserve)
	} else {
		buf = buf[:headerReserve]
	}
	bodyStart := len(buf)
	buf = appendBody(buf)
	bodyLen := len(buf) - bodyStart

	header := append(buf[:0], contentLengthPrefix...)
	header = appendUint(header, bodyLen)
	header = append(header, headerSuffix...)
	headerLen := len(header)
	copy(buf[headerLen:], buf[bodyStart:])
	buf = buf[:headerLen+bodyLen]
	s.wbuf = buf

	n, err := s.conn.Write(buf)
	return int64(n), err
}

// writeCall implements frameWriter: it frames a call envelope from its concrete
// fields without boxing a wire value into the Message interface.
func (s *headerStream) writeCall(ctx context.Context, id ID, method string, params RawMessage) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendCallFields(buf, id, method, params)
	})
}

// writeNotification implements frameWriter for a notification envelope.
func (s *headerStream) writeNotification(ctx context.Context, method string, params RawMessage) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendNotificationFields(buf, method, params)
	})
}

// writeResponse implements frameWriter for a response envelope.
func (s *headerStream) writeResponse(ctx context.Context, id ID, result RawMessage, err error) (int64, error) {
	if cerr := ctx.Err(); cerr != nil {
		return 0, cerr
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendResponseFields(buf, id, result, err)
	})
}

// WriteFrame implements the internal frameStream interface. It frames the
// already-encoded JSON body data (a single message or a batch array) with the
// LSP header and emits the header and body with one conn.Write, preserving the
// single-write framing guarantee.
func (s *headerStream) WriteFrame(ctx context.Context, data []byte) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	buf := s.wbuf[:0]
	buf = append(buf, "Content-Length: "...)
	buf = appendUint(buf, len(data))
	buf = append(buf, "\r\n\r\n"...)
	buf = append(buf, data...)
	s.wbuf = buf

	n, err := s.conn.Write(buf)
	return int64(n), err
}

// Close implements [Stream].
func (s *headerStream) Close() error { return s.conn.Close() }

// Read implements [Stream]. It reads up to and including the next '\n', decodes
// the JSON value that precedes it, and returns the decoded message. A decoded
// request's method and params borrow the read buffer and are valid only until
// the next read; copy what you retain.
func (s *ndjsonStream) Read(ctx context.Context) (Message, int64, error) {
	body, n, err := s.ReadFrame(ctx)
	if err != nil {
		return nil, 0, err
	}
	msg, derr := DecodeMessage(body)
	if derr != nil {
		return nil, 0, derr
	}
	return msg, n, nil
}

// ReadFrame implements the internal frameStream interface. It reads up to and
// including the next '\n' and returns the JSON body that precedes it, without
// decoding, so the connection can classify single messages from batch arrays.
// The returned slice may point into the bufio window or a temporary buffer and
// is valid only until the next read; a caller that retains the bytes must copy
// them.
func (s *ndjsonStream) ReadFrame(ctx context.Context) ([]byte, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	line, err := s.in.ReadSlice('\n')
	if err != nil {
		switch err {
		case io.EOF:
			if len(line) == 0 {
				// Clean boundary: the peer closed between frames.
				return nil, 0, io.EOF
			}
			// Bytes without a terminating newline: a truncated frame.
			return nil, 0, io.ErrUnexpectedEOF
		case bufio.ErrBufferFull:
			// The line is longer than the bufio buffer; fall back to a growable
			// read that accumulates the remaining bytes of this frame.
			return s.readFrameLong(line)
		default:
			return nil, 0, err
		}
	}

	n := int64(len(line))
	body := line[:len(line)-1] // drop the trailing '\n'
	return body, n, nil
}

// readFrameLong completes a frame whose length exceeds the bufio read buffer.
// prefix holds the bytes already consumed by the initial ReadSlice. It copies
// prefix into a growable buffer because the next ReadSlice invalidates the
// bufio window that backs prefix.
func (s *ndjsonStream) readFrameLong(prefix []byte) ([]byte, int64, error) {
	buf := make([]byte, len(prefix), len(prefix)*2)
	copy(buf, prefix)
	for {
		chunk, err := s.in.ReadSlice('\n')
		buf = append(buf, chunk...)
		if err == nil {
			break
		}
		switch err {
		case bufio.ErrBufferFull:
			continue
		case io.EOF:
			return nil, 0, io.ErrUnexpectedEOF
		default:
			return nil, 0, err
		}
	}
	n := int64(len(buf))
	return buf[:len(buf)-1], n, nil
}

// Write implements [Stream]. It appends the encoded envelope and a single '\n'
// directly into the reusable, writeMu-guarded compose buffer and emits the
// payload and its delimiter with one conn.Write.
//
// Unlike [headerStream.Write], the ndjson framing needs no length prefix before
// the body, so the envelope can be appended straight into the compose buffer
// without the intermediate owned copy that [EncodeMessage] makes. The buffer
// never crosses a goroutine boundary (it is guarded by writeMu for the duration
// of the write), so building the frame in place introduces no aliasing hazard.
func (s *ndjsonStream) Write(ctx context.Context, msg Message) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	return s.composeAndWrite(func(buf []byte) []byte {
		return appendMessage(buf, msg)
	})
}

// composeAndWrite appends one encoded body via appendBody and a single '\n'
// into the reused, writeMu-guarded compose buffer and emits it with one
// conn.Write. The caller must hold writeMu.
func (s *ndjsonStream) composeAndWrite(appendBody func(buf []byte) []byte) (int64, error) {
	buf := appendBody(s.wbuf[:0])
	buf = append(buf, '\n')
	s.wbuf = buf

	n, err := s.conn.Write(buf)
	return int64(n), err
}

// writeCall implements frameWriter: it frames a call envelope from its concrete
// fields without boxing a wire value into the Message interface.
func (s *ndjsonStream) writeCall(ctx context.Context, id ID, method string, params RawMessage) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendCallFields(buf, id, method, params)
	})
}

// writeNotification implements frameWriter for a notification envelope.
func (s *ndjsonStream) writeNotification(ctx context.Context, method string, params RawMessage) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendNotificationFields(buf, method, params)
	})
}

// writeResponse implements frameWriter for a response envelope.
func (s *ndjsonStream) writeResponse(ctx context.Context, id ID, result RawMessage, err error) (int64, error) {
	if cerr := ctx.Err(); cerr != nil {
		return 0, cerr
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.composeAndWrite(func(buf []byte) []byte {
		return appendResponseFields(buf, id, result, err)
	})
}

// WriteFrame implements the internal frameStream interface. It appends a single
// '\n' to the already-encoded JSON body data (a single message or a batch
// array) and emits the payload and its delimiter with one conn.Write,
// preserving the single-write framing guarantee.
func (s *ndjsonStream) WriteFrame(ctx context.Context, data []byte) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	buf := append(s.wbuf[:0], data...)
	buf = append(buf, '\n')
	s.wbuf = buf

	n, err := s.conn.Write(buf)
	return int64(n), err
}

// Close implements [Stream].
func (s *ndjsonStream) Close() error { return s.conn.Close() }

// readContentLength reads the LSP header block from r and returns the parsed
// Content-Length, the number of header bytes consumed (including the trailing
// blank line), and any error.
//
// Header field names are matched case-insensitively against "Content-Length";
// every other field, including Content-Type, is read and ignored. The blank
// line that terminates the header is required. A missing, zero, negative, or
// oversized Content-Length is reported as [ErrInvalidHeader].
func readContentLength(r *bufio.Reader) (length, consumed int, err error) {
	haveLength := false
	for {
		line, err := readHeaderLine(r)
		if err != nil {
			// A clean io.EOF before any header byte means the peer closed between
			// frames. Once at least one header line has been consumed, the header
			// block is truncated mid-frame.
			if err == io.EOF && consumed > 0 {
				err = io.ErrUnexpectedEOF
			}
			return 0, 0, err
		}
		consumed += len(line)
		// A blank line (just the CRLF) terminates the header block.
		if isBlankLine(line) {
			break
		}
		name, value, ok := splitHeaderField(line)
		if !ok {
			return 0, 0, ErrInvalidHeader
		}
		if asciiEqualFold(name, "content-length") {
			n, ok := parseContentLength(value)
			if !ok {
				return 0, 0, ErrInvalidHeader
			}
			length = n
			haveLength = true
		}
	}
	if !haveLength || length <= 0 {
		return 0, 0, ErrInvalidHeader
	}
	return length, consumed, nil
}

// readHeaderLine reads a single header line terminated by '\n', returning the
// raw line including its line terminator. It returns [io.ErrUnexpectedEOF] when
// the connection ends before the line terminator at a clean boundary, and
// [io.EOF] when the connection is closed before any header byte is read.
func readHeaderLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			if len(line) == 0 {
				return nil, io.EOF
			}
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return line, nil
}

// isBlankLine reports whether line is the header-terminating blank line: either
// "\r\n" or a bare "\n".
func isBlankLine(line []byte) bool {
	switch len(line) {
	case 1:
		return line[0] == '\n'
	case 2:
		return line[0] == '\r' && line[1] == '\n'
	default:
		return false
	}
}

// splitHeaderField splits a header line of the form "Name: value" into its name
// and value, trimming the surrounding CRLF and the optional space after the
// colon. It returns ok=false when the line has no colon separator.
func splitHeaderField(line []byte) (name, value []byte, ok bool) {
	// Trim the trailing line terminator.
	end := len(line)
	if end > 0 && line[end-1] == '\n' {
		end--
	}
	if end > 0 && line[end-1] == '\r' {
		end--
	}
	line = line[:end]

	colon := -1
	for i := range len(line) {
		if line[i] == ':' {
			colon = i
			break
		}
	}
	if colon < 0 {
		return nil, nil, false
	}
	name = line[:colon]
	value = line[colon+1:]
	// Trim a single leading space (the conventional "Name: value" form) and any
	// surrounding whitespace.
	for len(value) > 0 && (value[0] == ' ' || value[0] == '\t') {
		value = value[1:]
	}
	for len(value) > 0 && (value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return name, value, true
}

// maxContentLength caps the Content-Length a header frame may declare. It bounds
// the body buffer a single frame can force the reader to allocate, so a peer
// cannot trigger an out-of-memory allocation, and it keeps the accumulation in
// parseContentLength below the point where it could overflow int on any
// platform (including 32-bit). The limit is 1 GiB, far above any practical
// JSON-RPC message yet small enough to reject a hostile declaration outright.
const maxContentLength = 1 << 30

// parseContentLength parses value as a base-10 non-negative integer by scanning
// digits directly, avoiding a substring allocation. It returns ok=false when
// value is empty, contains a non-digit byte, or declares a length above
// [maxContentLength]. The overflow guard checks each digit before accumulating
// it, so n*10 never exceeds the cap and the running total cannot wrap int on a
// 32-bit platform.
func parseContentLength(value []byte) (n int, ok bool) {
	if len(value) == 0 {
		return 0, false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, false
		}
		// Reject before accumulating so n*10+digit can never exceed the cap or
		// wrap int. Equivalent to n*10+digit > maxContentLength without the
		// intermediate multiply that could overflow.
		if n > (maxContentLength-int(c-'0'))/10 {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// asciiEqualFold reports whether name equals want under ASCII case folding. want
// must be lowercase ASCII.
func asciiEqualFold(name []byte, want string) bool {
	if len(name) != len(want) {
		return false
	}
	for i := range name {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != want[i] {
			return false
		}
	}
	return true
}

// appendUint appends the base-10 representation of v (v >= 0) to dst and returns
// the extended slice, without allocating an intermediate string.
func appendUint(dst []byte, v int) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, tmp[i:]...)
}
