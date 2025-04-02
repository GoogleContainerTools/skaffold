package log

import (
	"fmt"
	"io"
	"sync"

	"github.com/apex/log"
	"github.com/heroku/color"
)

var _ Logger = &DefaultLogger{}
var _ LoggerHandlerWithLevel = &DefaultLogger{}

// DefaultLogger extends `github.com/apex/log` `log.Logger`
type DefaultLogger struct {
	*log.Logger
}

func NewDefaultLogger(writer io.Writer) *DefaultLogger {
	return &DefaultLogger{
		Logger: &log.Logger{
			Handler: &handler{
				writer: writer,
			},
		},
	}
}

func (l *DefaultLogger) HandleLog(entry *log.Entry) error {
	return l.Handler.HandleLog(entry)
}

func (l *DefaultLogger) LogLevel() log.Level {
	return l.Level
}

func (l *DefaultLogger) Phase(name string) {
	l.Infof(phaseStyle("===> %s", name))
}

func (l *DefaultLogger) SetLevel(requested string) error {
	var err error
	l.Level, err = log.ParseLevel(requested)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}
	return nil
}

var _ log.Handler = &handler{}

type handler struct {
	mu     sync.Mutex
	writer io.Writer
}

const (
	errorLevelText = "ERROR: "
	warnLevelText  = "Warning: "
)

var (
	warnStyle  = color.New(color.FgYellow, color.Bold).SprintfFunc()
	errorStyle = color.New(color.FgRed, color.Bold).SprintfFunc()
	phaseStyle = color.New(color.FgCyan).SprintfFunc()
)

func (h *handler) HandleLog(entry *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var err error
	switch entry.Level {
	case log.WarnLevel:
		_, err = h.writer.Write([]byte(warnStyle(warnLevelText) + appendMissingLineFeed(entry.Message)))
	case log.ErrorLevel:
		_, err = h.writer.Write([]byte(errorStyle(errorLevelText) + appendMissingLineFeed(entry.Message)))
	default:
		_, err = h.writer.Write([]byte(appendMissingLineFeed(entry.Message)))
	}
	return err
}

func appendMissingLineFeed(msg string) string {
	buff := []byte(msg)
	if buff[len(buff)-1] != '\n' {
		buff = append(buff, '\n')
	}
	return string(buff)
}
