package trace

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

//Printer ...
type Printer interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

//Closer ...
type Closer interface {
	Close() error
}

//PrinterCloser ...
type PrinterCloser interface {
	Printer
	Closer
}

//NullLogger ...
type NullLogger struct{}

func (l *NullLogger) Print(v ...interface{})                 {}
func (l *NullLogger) Printf(format string, v ...interface{}) {}
func (l *NullLogger) Println(v ...interface{})               {}

type loggerImpl struct {
	*log.Logger
	c io.WriteCloser
}

func (loggerImpl *loggerImpl) Close() error {
	if loggerImpl.c != nil {
		return loggerImpl.c.Close()
	}
	return nil
}

func newLoggerImpl(out io.Writer, prefix string, flag int) *loggerImpl {
	l := log.New(out, prefix, flag)
	c := out.(io.WriteCloser)
	return &loggerImpl{
		Logger: l,
		c:      c,
	}
}

//Logger is global logger
var Logger Printer = NewLogger("")

// NewLogger returns a printer for the given trace setting.
func NewLogger(bluemix_trace string) Printer {
	switch strings.ToLower(bluemix_trace) {
	case "", "false":
		return new(NullLogger)
	case "true":
		return NewStdLogger()
	default:
		return NewFileLogger(bluemix_trace)
	}
}

// NewStdLogger return a printer that writes to StdOut.
func NewStdLogger() PrinterCloser {
	return newLoggerImpl(os.Stderr, "", 0)
}

// NewFileLogger return a printer that writes to the given file path.
func NewFileLogger(path string) PrinterCloser {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		logger := NewStdLogger()
		logger.Printf("An error occurred when creating log file '%s':\n%v\n\n", path, err)
		return logger
	}
	return newLoggerImpl(file, "", 0)
}

// Sanitize returns a clean string with sentive user data in the input
// replaced by PRIVATE_DATA_PLACEHOLDER.
func Sanitize(input string) string {
	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized := re.ReplaceAllString(input, "Authorization: "+privateDataPlaceholder())

	re = regexp.MustCompile(`(?m)^X-Auth-Token: .*`)
	sanitized = re.ReplaceAllString(sanitized, "X-Auth-Token: "+privateDataPlaceholder())

	re = regexp.MustCompile(`(?m)^X-Auth-Refresh-Token: .*`)
	sanitized = re.ReplaceAllString(sanitized, "X-Auth-Refresh-Token: "+privateDataPlaceholder())

	re = regexp.MustCompile(`(?m)^X-Auth-Uaa-Token: .*`)
	sanitized = re.ReplaceAllString(sanitized, "X-Auth-Uaa-Token: "+privateDataPlaceholder())

	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+privateDataPlaceholder()+"&")

	re = regexp.MustCompile(`refresh_token=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "refresh_token="+privateDataPlaceholder()+"&")

	re = regexp.MustCompile(`apikey=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "apikey="+privateDataPlaceholder()+"&")

	sanitized = sanitizeJSON("token", sanitized)
	sanitized = sanitizeJSON("password", sanitized)
	sanitized = sanitizeJSON("apikey", sanitized)
	sanitized = sanitizeJSON("passcode", sanitized)

	return sanitized
}

func sanitizeJSON(propertySubstring string, json string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)"([^"]*%s[^"]*)":\s*"[^\,]*"`, propertySubstring))
	return regex.ReplaceAllString(json, fmt.Sprintf(`"$1":"%s"`, privateDataPlaceholder()))
}

// privateDataPlaceholder returns the text to replace the sentive data.
func privateDataPlaceholder() string {
	return "[PRIVATE DATA HIDDEN]"
}
