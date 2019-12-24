// Package logging defines the minimal interface that loggers must support to be used by pack.
package logging

import (
	"fmt"
	"io"

	"github.com/buildpacks/pack/internal/style"
)

// Logger defines behavior required by a logging package used by pack libraries
type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...interface{})

	Info(msg string)
	Infof(fmt string, v ...interface{})

	Warn(msg string)
	Warnf(fmt string, v ...interface{})

	Error(msg string)
	Errorf(fmt string, v ...interface{})

	Writer() io.Writer

	IsVerbose() bool
}

// WithInfoErrorWriter is an optional interface for loggers that want to support a separate writer for errors and standard logging.
// the DebugInfoWriter should write to stderr if quiet is false.
type WithInfoErrorWriter interface {
	InfoErrorWriter() io.Writer
}

// WithInfoWriter is an optional interface what will return a writer that will write raw output if quiet is false.
type WithInfoWriter interface {
	InfoWriter() io.Writer
}

// GetInfoErrorWriter will return an ErrorWriter, typically stderr if one exists, otherwise the standard logger writer
// will be returned.
func GetInfoErrorWriter(l Logger) io.Writer {
	if er, ok := l.(WithInfoErrorWriter); ok {
		return er.InfoErrorWriter()
	}
	return l.Writer()
}

// GetInfoWriter returns a writer
// See WithInfoWriter
func GetInfoWriter(l Logger) io.Writer {
	if ew, ok := l.(WithInfoWriter); ok {
		return ew.InfoWriter()
	}
	return l.Writer()
}

// PrefixWriter will prefix writes
type PrefixWriter struct {
	out    io.Writer
	prefix string
}

// NewPrefixWriter writes by w will be prefixed
func NewPrefixWriter(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		out:    w,
		prefix: fmt.Sprintf("[%s] ", style.Prefix(prefix)),
	}
}

// Writes bytes to the embedded log function
func (w *PrefixWriter) Write(buf []byte) (int, error) {
	_, _ = fmt.Fprint(w.out, w.prefix+string(buf))
	return len(buf), nil
}

// Tip logs a tip.
func Tip(l Logger, format string, v ...interface{}) {
	l.Infof(style.Tip("Tip: ")+format, v...)
}
