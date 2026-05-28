// Package log has logging interfaces for convenience in lifecycle
package log

import "github.com/apex/log"

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...any)

	Info(msg string)
	Infof(fmt string, v ...any)

	Warn(msg string)
	Warnf(fmt string, v ...any)

	Error(msg string)
	Errorf(fmt string, v ...any)
}

type LoggerHandlerWithLevel interface {
	Logger
	HandleLog(entry *log.Entry) error
	LogLevel() log.Level
}
