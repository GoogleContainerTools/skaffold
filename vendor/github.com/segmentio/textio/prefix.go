package textio

import (
	"bytes"
	"fmt"
	"io"
)

// PrefixWriter is an implementation of io.Writer which places a prefix before
// every line.
//
// Instances of PrefixWriter are not safe to use concurrently from multiple
// goroutines.
type PrefixWriter struct {
	writer io.Writer
	indent []byte
	buffer []byte
	offset int
}

// NewPrefixWriter constructs a PrefixWriter which outputs to w and prefixes
// every line with s.
func NewPrefixWriter(w io.Writer, s string) *PrefixWriter {
	return &PrefixWriter{
		writer: w,
		indent: copyStringToBytes(s),
		buffer: make([]byte, 0, 256),
	}
}

// Base returns the underlying writer that w outputs to.
func (w *PrefixWriter) Base() io.Writer {
	return w.writer
}

// Buffered returns a byte slice of the data currently buffered in the writer.
func (w *PrefixWriter) Buffered() []byte {
	return w.buffer[w.offset:]
}

// Write writes b to w, satisfies the io.Writer interface.
func (w *PrefixWriter) Write(b []byte) (int, error) {
	var c int
	var n int
	var err error

	forEachLine(b, func(line []byte) bool {
		// Always buffer so the input slice doesn't escape and WriteString won't
		// copy the string (it saves a dynamic memory allocation on every call
		// to WriteString).
		w.buffer = append(w.buffer, line...)

		if chunk := w.Buffered(); isLine(chunk) {
			c, err = w.writeLine(chunk)
			w.discard(c)
		}

		n += len(line)
		return err == nil
	})

	return n, err
}

// WriteString writes s to w.
func (w *PrefixWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Flush forces all buffered data to be flushed to the underlying writer.
func (w *PrefixWriter) Flush() error {
	n, err := w.write(w.buffer)
	w.discard(n)
	return err
}

// Width satisfies the fmt.State interface.
func (w *PrefixWriter) Width() (int, bool) {
	f, ok := Base(w).(fmt.State)
	if ok {
		return f.Width()
	}
	return 0, false
}

// Precision satisfies the fmt.State interface.
func (w *PrefixWriter) Precision() (int, bool) {
	f, ok := Base(w).(fmt.State)
	if ok {
		return f.Precision()
	}
	return 0, false
}

// Flag satisfies the fmt.State interface.
func (w *PrefixWriter) Flag(c int) bool {
	f, ok := Base(w).(fmt.State)
	if ok {
		return f.Flag(c)
	}
	return false
}

func (w *PrefixWriter) writeLine(b []byte) (int, error) {
	if _, err := w.write(w.indent); err != nil {
		return 0, err
	}
	return w.write(b)
}

func (w *PrefixWriter) write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *PrefixWriter) discard(n int) {
	if n > 0 {
		w.offset += n

		switch {
		case w.offset == len(w.buffer):
			w.buffer = w.buffer[:0]
			w.offset = 0

		case w.offset > (cap(w.buffer) / 2):
			copy(w.buffer, w.buffer[w.offset:])
			w.buffer = w.buffer[:len(w.buffer)-w.offset]
			w.offset = 0
		}
	}
}

func copyStringToBytes(s string) []byte {
	b := make([]byte, len(s))
	copy(b, s)
	return b
}

func forEachLine(b []byte, do func([]byte) bool) {
	for len(b) != 0 {
		i := bytes.IndexByte(b, '\n')

		if i < 0 {
			i = len(b)
		} else {
			i++ // include the newline character
		}

		if !do(b[:i]) {
			break
		}

		b = b[i:]
	}
}

func isLine(b []byte) bool {
	return len(b) != 0 && b[len(b)-1] == '\n'
}

var (
	_ fmt.State = (*PrefixWriter)(nil)
)
