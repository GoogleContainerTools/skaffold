package cmd

import (
	"io"
	"os"
	"sync"

	"github.com/apex/log"
	"github.com/heroku/color"
)

const (
	errorLevelText = "ERROR: "
	warnLevelText  = "Warning: "
)

// Default logger
var (
	Logger = &log.Logger{
		Handler: &handler{
			writer: os.Stdout,
		},
	}
	warnStyle  = color.New(color.FgYellow, color.Bold).SprintfFunc()
	errorStyle = color.New(color.FgRed, color.Bold).SprintfFunc()
)

func SetLogLevel(level string) *ErrorFail {
	var err error
	Logger.Level, err = log.ParseLevel(level)
	if err != nil {
		return FailErrCode(err, CodeInvalidArgs, "parse log level")
	}

	return nil
}

type handler struct {
	mu     sync.Mutex
	writer io.Writer
}

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
