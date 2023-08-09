// Package logging defines the minimal interface that loggers must support to be used by client.
package logging

import (
	"io"
	"io/ioutil"

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

type isSelectableWriter interface {
	WriterForLevel(level Level) io.Writer
}

// GetWriterForLevel retrieves the appropriate Writer for the log level provided.
//
// See isSelectableWriter
func GetWriterForLevel(logger Logger, level Level) io.Writer {
	if w, ok := logger.(isSelectableWriter); ok {
		return w.WriterForLevel(level)
	}

	return logger.Writer()
}

// IsQuiet defines whether a pack logger is set to quiet mode
func IsQuiet(logger Logger) bool {
	if writer := GetWriterForLevel(logger, InfoLevel); writer == ioutil.Discard {
		return true
	}

	return false
}

// Tip logs a tip.
func Tip(l Logger, format string, v ...interface{}) {
	l.Infof(style.Tip("Tip: ")+format, v...)
}
