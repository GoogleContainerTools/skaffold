// Package color produces colored output in terms of ANSI Escape Codes. Posix and Windows platforms are supported.
package color

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
)

var noColor bool
var noColorLock sync.RWMutex

// Disable is used to turn color output on and off globally.
func Disable(flag bool) {
	noColorLock.Lock()
	defer noColorLock.Unlock()
	noColor = flag
}

// Enabled returns flag indicating whether colors are enabled or not.
func Enabled() bool {
	noColorLock.RLock()
	defer noColorLock.RUnlock()
	return !noColor
}

var stdout *Console // Don't use directly use Stdout() instead.
var stdoutOnce sync.Once

// Stdout returns an io.Writer that writes colored text to standard out.
func Stdout() *Console {
	stdoutOnce.Do(func() {
		stdout = NewConsole(os.Stdout)
	})
	return stdout
}

var stderr *Console // Don't use directly use Stderr() instead.
var stderrOnce sync.Once

// Stderr returns an io.Writer that writes colored text to standard error.
func Stderr() *Console {
	stderrOnce.Do(func() {
		stderr = NewConsole(os.Stderr)
	})
	return stderr
}

// Console manages state for output, typically stdout or stderr.
type Console struct {
	sync.Mutex
	colored        io.Writer
	noncolored     io.Writer
	current        io.Writer
	fileDescriptor uintptr
}

// NewConsole creates a wrapper around out which will output platform independent colored text.
func NewConsole(out *os.File) *Console {
	c := &Console{
		colored:        colorable.NewColorable(out),
		noncolored:     colorable.NewNonColorable(out),
		fileDescriptor: out.Fd(),
	}
	if Enabled() {
		c.current = c.colored
		return c
	}
	c.current = c.noncolored
	return c
}

func (c *Console) Fd() uintptr {
	c.Lock()
	defer c.Unlock()
	return c.fileDescriptor
}

// DisableColors if true ANSI color information will be removed for this console object. Passing true will enable
// colors for this Console, even if colors are disabled globally.
func (c *Console) DisableColors(strip bool) {
	c.Lock()
	defer c.Unlock()
	if strip {
		c.current = c.noncolored
		return
	}
	c.current = c.colored
}

// Set will cause the color passed in as an argument to be written until Unset is called.
func (c *Console) Set(color *Color) {
	c.Lock()
	defer c.Unlock()
	_, _ = c.current.Write([]byte(color.colorStart))
}

// Unset will restore console output to default. It will undo colored console output defined from a call to Set.
func (c *Console) Unset() {
	c.Lock()
	defer c.Unlock()
	_, _ = c.current.Write([]byte(colorReset))
}

// Write so we can treat a console as a Writer
func (c *Console) Write(b []byte) (int, error) {
	c.Lock()
	n, err := c.current.Write(b)
	c.Unlock()
	return n, err
}

// Print writes colored text to the console. The number of bytes written
// is returned.
func (c *Console) Print(col *Color, args ...string) (int, error) {
	return c.Write([]byte(col.wrap(args...)))
}

// Printf formats according to a format specifier and writes colored text to the console.
func (c *Console) Printf(col *Color, format string, args ...interface{}) (int, error) {
	return c.Write([]byte(col.wrap(fmt.Sprintf(format, args...))))
}

// Println writes colored text to console, appending input with a line feed.
// The number of bytes written is returned.
func (c *Console) Println(col *Color, args ...string) (int, error) {
	s := col.wrap(args...)
	if !strings.HasSuffix(s, lineFeed) {
		s += lineFeed
	}
	return c.Write([]byte(s))

}

// PrintFunc returns a wrapper function for Print.
func (c *Console) PrintFunc(col *Color) func(a ...string) {
	return func(a ...string) {
		_, _ = c.Print(col, a...)
	}
}

// PrintfFunc returns a wrapper function for Printf.
func (c *Console) PrintfFunc(col *Color) func(format string, args ...interface{}) {
	return func(format string, s ...interface{}) {
		_, _ = c.Printf(col, format, s...)
	}
}

// PrintlnFunc returns a wrapper function for Println.
func (c *Console) PrintlnFunc(col *Color) func(a ...string) {
	return func(a ...string) {
		_, _ = c.Println(col, a...)
	}
}

func (c *Console) colorPrint(format string, attr Attribute, a ...interface{}) {
	col := cache().value(attr)
	if !strings.HasSuffix(format, lineFeed) {
		_, _ = c.Println(col, fmt.Sprintf(format, a...))
		return
	}
	_, _ = c.Printf(col, format, fmt.Sprint(a...))
}
