package logging

import (
	"fmt"
	"io"
	"log"
)

// NewSimpleLogger creates a simple logger for the pack library.
func NewSimpleLogger(w io.Writer) Logger {
	return &simpleLogger{
		out: log.New(w, "", log.LstdFlags|log.Lmicroseconds),
	}
}

type simpleLogger struct {
	out *log.Logger
}

const (
	debugPrefix = "DEBUG:"
	infoPrefix  = "INFO:"
	warnPrefix  = "WARN:"
	errorPrefix = "ERROR:"
	prefixFmt   = "%-7s %s"
)

func (l *simpleLogger) Debug(msg string) {
	l.out.Printf(prefixFmt, debugPrefix, msg)
}

func (l *simpleLogger) Debugf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, debugPrefix, fmt.Sprintf(format, v...))
}

func (l *simpleLogger) Info(msg string) {
	l.out.Printf(prefixFmt, infoPrefix, msg)
}

func (l *simpleLogger) Infof(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, infoPrefix, fmt.Sprintf(format, v...))
}

func (l *simpleLogger) Warn(msg string) {
	l.out.Printf(prefixFmt, warnPrefix, msg)
}

func (l *simpleLogger) Warnf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, warnPrefix, fmt.Sprintf(format, v...))
}

func (l *simpleLogger) Error(msg string) {
	l.out.Printf(prefixFmt, errorPrefix, msg)
}

func (l *simpleLogger) Errorf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, errorPrefix, fmt.Sprintf(format, v...))
}

func (l *simpleLogger) Writer() io.Writer {
	return l.out.Writer()
}

func (l *simpleLogger) IsVerbose() bool {
	return false
}
