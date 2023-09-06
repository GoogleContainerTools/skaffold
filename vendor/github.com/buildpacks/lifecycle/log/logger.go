// Package log has logging interfaces for convenience in lifecycle
package log

import "github.com/apex/log"

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...interface{})

	Info(msg string)
	Infof(fmt string, v ...interface{})

	Warn(msg string)
	Warnf(fmt string, v ...interface{})

	Error(msg string)
	Errorf(fmt string, v ...interface{})
}

type LoggerHandlerWithLevel interface {
	Logger
	HandleLog(entry *log.Entry) error
	LogLevel() log.Level
}
