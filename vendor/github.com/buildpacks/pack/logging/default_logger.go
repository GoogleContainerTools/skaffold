package logging

import (
	"fmt"
	"io"
	"log"
)

// New creates a default logger for the pack library. Note that the pack CLI has it's own logger.
func New(w io.Writer) Logger {
	return &defaultLogger{
		out: log.New(w, "", log.LstdFlags|log.Lmicroseconds),
	}
}

type defaultLogger struct {
	out *log.Logger
}

const (
	debugPrefix = "DEBUG:"
	infoPrefix  = "INFO:"
	warnPrefix  = "WARN:"
	errorPrefix = "ERROR:"
	prefixFmt   = "%-7s %s"
)

func (l *defaultLogger) Debug(msg string) {
	l.out.Printf(prefixFmt, debugPrefix, msg)
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, debugPrefix, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Info(msg string) {
	l.out.Printf(prefixFmt, infoPrefix, msg)
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, infoPrefix, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Warn(msg string) {
	l.out.Printf(prefixFmt, warnPrefix, msg)
}

func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, warnPrefix, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Error(msg string) {
	l.out.Printf(prefixFmt, errorPrefix, msg)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.out.Printf(prefixFmt, errorPrefix, fmt.Sprintf(format, v...))
}

func (l *defaultLogger) Writer() io.Writer {
	return l.out.Writer()
}

func (l *defaultLogger) IsVerbose() bool {
	return false
}
