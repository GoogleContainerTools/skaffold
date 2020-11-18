package logging

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/buildpacks/pack/internal/style"
)

// PrefixWriter is a buffering writer that prefixes each new line. Close should be called to properly flush the buffer.
type PrefixWriter struct {
	out           io.Writer
	buf           *bytes.Buffer
	prefix        string
	readerFactory func(data []byte) io.Reader
}

type PrefixWriterOption func(c *PrefixWriter)

func WithReaderFactory(factory func(data []byte) io.Reader) PrefixWriterOption {
	return func(writer *PrefixWriter) {
		writer.readerFactory = factory
	}
}

// NewPrefixWriter writes by w will be prefixed
func NewPrefixWriter(w io.Writer, prefix string, opts ...PrefixWriterOption) *PrefixWriter {
	writer := &PrefixWriter{
		out:    w,
		prefix: fmt.Sprintf("[%s] ", style.Prefix(prefix)),
		buf:    &bytes.Buffer{},
		readerFactory: func(data []byte) io.Reader {
			return bytes.NewReader(data)
		},
	}

	for _, opt := range opts {
		opt(writer)
	}

	return writer
}

// Write writes bytes to the embedded log function
func (w *PrefixWriter) Write(data []byte) (int, error) {
	scanner := bufio.NewScanner(w.readerFactory(data))
	scanner.Split(ScanLinesKeepNewLine)
	for scanner.Scan() {
		newBits := scanner.Bytes()
		if len(newBits) > 0 && newBits[len(newBits)-1] != '\n' { // just append if we don't have a new line
			_, err := w.buf.Write(newBits)
			if err != nil {
				return 0, err
			}
		} else { // write our complete message
			_, err := w.buf.Write(bytes.TrimRight(newBits, "\n"))
			if err != nil {
				return 0, err
			}

			err = w.flush()
			if err != nil {
				return 0, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close writes any pending data in the buffer
func (w *PrefixWriter) Close() error {
	if w.buf.Len() > 0 {
		return w.flush()
	}

	return nil
}

func (w *PrefixWriter) flush() error {
	bits := w.buf.Bytes()
	w.buf.Reset()

	// process any CR in message
	if i := bytes.LastIndexByte(bits, '\r'); i >= 0 {
		bits = bits[i+1:]
	}

	_, err := fmt.Fprint(w.out, w.prefix+string(bits)+"\n")
	return err
}

// A customized implementation of bufio.ScanLines that preserves new line characters.
func ScanLinesKeepNewLine(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// first we'll split by LF (\n)
	// then remove any preceding CR (\r) [due to CR+LF]
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, append(dropCR(data[0:i]), '\n'), nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
