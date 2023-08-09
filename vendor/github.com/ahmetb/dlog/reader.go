// Package dlog provides utilities to read Docker Logs API stream format.
package dlog

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// these should match https://github.com/docker/docker/blob/master/pkg/stdcopy/stdcopy.go
	stdWriterPrefixLen = 8 // len of header
	stdWriterSizeIndex = 4 // size byte index in header

	initialBufLen = 1024 * 2
	maxMsgLen     = 1024 * 64
)

type reader struct {
	r io.Reader // original reader

	// reader state
	inMsg     bool
	msgLen    uint32
	cursor    uint32
	buf       []byte
	prefixBuf []byte
}

// NewReader returns a reader that strips off the message headers from the
// underlying raw docker logs stream and returns the messages.
func NewReader(r io.Reader) io.Reader {
	return &reader{
		r:         r,
		prefixBuf: make([]byte, stdWriterPrefixLen),
		buf:       make([]byte, initialBufLen)}
}

func (r *reader) Read(p []byte) (n int, err error) {
	// at the beginning of a message, parse and store the message
	if !r.inMsg {
		if err := r.parse(); err != nil {
			return 0, err
		}
		r.inMsg = true
	}

	n, err = r.readMsg(p) // serve from buf
	if err == io.EOF {
		err = nil // continue next msg (parse() handles the EOF from r.r)
		r.inMsg = false
	}
	return
}

func (r *reader) readMsg(p []byte) (int, error) {
	if r.cursor >= r.msgLen {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.cursor:r.msgLen])
	r.cursor += uint32(n)
	return n, nil
}

func (r *reader) parse() error {
	n, err := io.ReadFull(r.r, r.prefixBuf)
	if err != nil {
		switch err {
		case io.EOF:
			return err // end of the underlying logs stream
		case io.ErrUnexpectedEOF:
			return fmt.Errorf("dlog: corrupted prefix. read %d bytes", n)
		default:
			return fmt.Errorf("dlog: error reading prefix: %v", err)
		}
	}

	if r.prefixBuf[0] != 0x1 && r.prefixBuf[0] != 0x2 {
		return fmt.Errorf("dlog: unexpected stream byte: %#x", r.prefixBuf[0])
	}

	size := binary.BigEndian.Uint32(r.prefixBuf[stdWriterSizeIndex : stdWriterSizeIndex+4])
	if size > maxMsgLen { // safeguard to prevent reading garbage
		return fmt.Errorf("dlog: parsed msg too large: %d (max: %d) suspected garbage", size, maxMsgLen)
	}

	// grow buf if necessary
	if int(size) > len(r.buf) {
		r.buf = make([]byte, size)
	}

	// read the message body into buf
	m, err := io.ReadFull(r.r, r.buf[:int(size)])
	if err != nil {
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			return fmt.Errorf("dlog: corrupt message read %d out of %d bytes: %v", m, size, err)
		default:
			return fmt.Errorf("dlog: failed to read message: %v", err)
		}
	}

	// reset cursors for the new message
	r.msgLen = size
	r.cursor = 0
	return nil
}
