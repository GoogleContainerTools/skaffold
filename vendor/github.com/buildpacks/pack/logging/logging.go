// Package logging defines the minimal interface that loggers must support to be used by pack.
package logging

import (
	"fmt"
	"io"

	"github.com/buildpacks/pack/internal/style"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
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

// WithSelectableWriter is an optional interface for loggers that want to support a separate writer per log level.
type WithSelectableWriter interface {
	WriterForLevel(level Level) io.Writer
}

// GetWriterForLevel retrieves the appropriate Writer for the log level provided.
//
// See WithSelectableWriter
func GetWriterForLevel(logger Logger, level Level) io.Writer {
	if er, ok := logger.(WithSelectableWriter); ok {
		return er.WriterForLevel(level)
	}

	return logger.Writer()
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
