package log

import "time"

// Interface represents the API of both Logger and Entry.
type Interface interface {
	WithFields(Fielder) *Entry
	WithField(string, interface{}) *Entry
	WithDuration(time.Duration) *Entry
	WithError(error) *Entry
	Debug(string)
	Info(string)
	Warn(string)
	Error(string)
	Fatal(string)
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Trace(string) *Entry
}
