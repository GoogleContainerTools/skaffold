/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Includes code from github.com/sirupsen/logrus (MIT License)

package log

import (
	"context"
	"fmt"
	"io"
	stdlog "log"

	ggcrlogs "github.com/google/go-containerregistry/pkg/logs"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// Logging levels. Defining our own so we can encapsulate the underlying logger implementation.
const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

// Level type for logging levels
type Level uint32

// AllLevels exposes all logging levels
var AllLevels = []Level{
	PanicLevel,
	FatalLevel,
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
	TraceLevel,
}

// DefaultLogLevel for the global Skaffold logger
const DefaultLogLevel = WarnLevel

type contextKey struct{}

var ContextKey = contextKey{}

// logger is the global logrus.Logger for Skaffold
// TODO: Make this not global.
var logger = New()

type EventContext struct {
	Task    constants.Phase
	Subtask string
}

// String converts the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	switch level {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}
	return "unknown"
}

// Entry takes an context.Context and constructs a logrus.Entry from it, adding
// fields for task and subtask information
func Entry(ctx context.Context) *logrus.Entry {
	val := ctx.Value(ContextKey)
	if eventContext, ok := val.(EventContext); ok {
		return logger.WithFields(logrus.Fields{
			"task":    eventContext.Task,
			"subtask": eventContext.Subtask,
		})
	}

	// Use constants.DevLoop as the default task, as it's the highest level task we
	// can default to if one isn't specified.
	return logger.WithFields(logrus.Fields{
		"task":    constants.DevLoop,
		"subtask": constants.SubtaskIDNone,
	})
}

// IsDebugLevelEnabled returns true if debug level log is enabled.
func IsDebugLevelEnabled() bool {
	return logger.IsLevelEnabled(logrus.DebugLevel)
}

// IsTraceLevelEnabled returns true if trace level log is enabled.
func IsTraceLevelEnabled() bool {
	return logger.IsLevelEnabled(logrus.TraceLevel)
}

// New returns a new logrus.Logger.
// We use a new instance instead of the default logrus singleton to avoid clashes with dependencies that also use logrus.
func New() *logrus.Logger {
	return logrus.New()
}

// KanikoLogLevel makes sure kaniko logs at least at Info level and at most Debug level (trace doesn't work with Kaniko)
func KanikoLogLevel() logrus.Level {
	if GetLevel() <= InfoLevel {
		return logrus.InfoLevel
	}
	return logrus.DebugLevel
}

// SetupLogs sets up logrus logger for skaffold command line
func SetupLogs(stdErr io.Writer, level string, timestamp bool, hook logrus.Hook) error {
	logger.SetOutput(stdErr)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	logger.SetLevel(lvl)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: timestamp,
	})
	logger.AddHook(hook)
	setupStdLog(logger, lvl, stdlog.Default())
	setupGGCRLogging(logger, lvl)
	return nil
}

// AddHook adds a hook to the global Skaffold logger.
func AddHook(hook logrus.Hook) {
	logger.AddHook(hook)
}

// SetLevel sets the global Skaffold logger level.
func SetLevel(level Level) {
	logger.SetLevel(logrus.AllLevels[level])
}

// GetLevel returns the global Skaffold logger level.
func GetLevel() Level {
	return AllLevels[logger.GetLevel()]
}

// setupStdLog writes Go's standard library `log` messages to logrus at Info level.
//
// This function uses SetFlags() to standardize the output format.
func setupStdLog(logger *logrus.Logger, lvl logrus.Level, stdlogger *stdlog.Logger) {
	stdlogger.SetFlags(0)
	if lvl >= logrus.InfoLevel {
		stdlogger.SetOutput(logger.WriterLevel(logrus.InfoLevel))
	}
}

// setupGGCRLogging enables go-containerregistry logging, mapping its levels to our levels.
//
// The mapping is:
// - ggcr Warn -> Skaffold Error
// - ggcr Progress -> Skaffold Info
// - ggcr Debug -> Skaffold Trace
//
// The reasons for this mapping are:
// - `ggcr` defines `Warn` as "non-fatal errors": https://github.com/google/go-containerregistry/blob/main/pkg/logs/logs.go#L24
// - `ggcr` `Debug` logging is _very_ verbose and includes HTTP requests and responses, with non-sensitive headers and non-binary payloads.
//
// This function uses SetFlags() to standardize the output format.
func setupGGCRLogging(logger *logrus.Logger, lvl logrus.Level) {
	if lvl >= logrus.ErrorLevel {
		ggcrlogs.Warn.SetOutput(logger.WriterLevel(logrus.ErrorLevel))
		ggcrlogs.Warn.SetFlags(0)
	}
	if lvl >= logrus.InfoLevel {
		ggcrlogs.Progress.SetOutput(logger.WriterLevel(logrus.InfoLevel))
		ggcrlogs.Progress.SetFlags(0)
	}
	if lvl >= logrus.TraceLevel {
		ggcrlogs.Debug.SetOutput(logger.WriterLevel(logrus.TraceLevel))
		ggcrlogs.Debug.SetFlags(0)
	}
}
