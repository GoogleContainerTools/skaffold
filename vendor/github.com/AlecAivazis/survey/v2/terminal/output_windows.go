package terminal

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mattn/go-isatty"
)

const (
	foregroundBlue      = 0x1
	foregroundGreen     = 0x2
	foregroundRed       = 0x4
	foregroundIntensity = 0x8
	foregroundMask      = (foregroundRed | foregroundBlue | foregroundGreen | foregroundIntensity)
	backgroundBlue      = 0x10
	backgroundGreen     = 0x20
	backgroundRed       = 0x40
	backgroundIntensity = 0x80
	backgroundMask      = (backgroundRed | backgroundBlue | backgroundGreen | backgroundIntensity)
)

type Writer struct {
	out     FileWriter
	handle  syscall.Handle
	orgAttr word
}

func NewAnsiStdout(out FileWriter) io.Writer {
	var csbi consoleScreenBufferInfo
	if !isatty.IsTerminal(out.Fd()) {
		return out
	}
	handle := syscall.Handle(out.Fd())
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	return &Writer{out: out, handle: handle, orgAttr: csbi.attributes}
}

func NewAnsiStderr(out FileWriter) io.Writer {
	var csbi consoleScreenBufferInfo
	if !isatty.IsTerminal(out.Fd()) {
		return out
	}
	handle := syscall.Handle(out.Fd())
	procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	return &Writer{out: out, handle: handle, orgAttr: csbi.attributes}
}

func (w *Writer) Write(data []byte) (n int, err error) {
	r := bytes.NewReader(data)

	for {
		var ch rune
		var size int
		ch, size, err = r.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		n += size

		switch ch {
		case '\x1b':
			size, err = w.handleEscape(r)
			n += size
			if err != nil {
				return
			}
		default:
			_, err = fmt.Fprint(w.out, string(ch))
			if err != nil {
				return
			}
		}
	}
}

func (w *Writer) handleEscape(r *bytes.Reader) (n int, err error) {
	buf := make([]byte, 0, 10)
	buf = append(buf, "\x1b"...)

	var ch rune
	var size int
	// Check '[' continues after \x1b
	ch, size, err = r.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		fmt.Fprint(w.out, string(buf))
		return
	}
	n += size
	if ch != '[' {
		fmt.Fprint(w.out, string(buf))
		return
	}

	// Parse escape code
	var code rune
	argBuf := make([]byte, 0, 10)
	for {
		ch, size, err = r.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			fmt.Fprint(w.out, string(buf))
			return
		}
		n += size
		if ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') {
			code = ch
			break
		}
		argBuf = append(argBuf, string(ch)...)
	}

	err = w.applyEscapeCode(buf, string(argBuf), code)
	return
}

func (w *Writer) applyEscapeCode(buf []byte, arg string, code rune) error {
	c := &Cursor{Out: w.out}

	switch arg + string(code) {
	case "?25h":
		return c.Show()
	case "?25l":
		return c.Hide()
	}

	if code >= 'A' && code <= 'G' {
		if n, err := strconv.Atoi(arg); err == nil {
			switch code {
			case 'A':
				return c.Up(n)
			case 'B':
				return c.Down(n)
			case 'C':
				return c.Forward(n)
			case 'D':
				return c.Back(n)
			case 'E':
				return c.NextLine(n)
			case 'F':
				return c.PreviousLine(n)
			case 'G':
				return c.HorizontalAbsolute(n)
			}
		}
	}

	switch code {
	case 'm':
		return w.applySelectGraphicRendition(arg)
	default:
		buf = append(buf, string(code)...)
		_, err := fmt.Fprint(w.out, string(buf))
		return err
	}
}

// Original implementation: https://github.com/mattn/go-colorable
func (w *Writer) applySelectGraphicRendition(arg string) error {
	if arg == "" {
		_, _, err := procSetConsoleTextAttribute.Call(uintptr(w.handle), uintptr(w.orgAttr))
		return normalizeError(err)
	}

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(w.handle), uintptr(unsafe.Pointer(&csbi))); normalizeError(err) != nil {
		return err
	}
	attr := csbi.attributes

	for _, param := range strings.Split(arg, ";") {
		n, err := strconv.Atoi(param)
		if err != nil {
			continue
		}

		switch {
		case n == 0 || n == 100:
			attr = w.orgAttr
		case 1 <= n && n <= 5:
			attr |= foregroundIntensity
		case 30 <= n && n <= 37:
			attr = (attr & backgroundMask)
			if (n-30)&1 != 0 {
				attr |= foregroundRed
			}
			if (n-30)&2 != 0 {
				attr |= foregroundGreen
			}
			if (n-30)&4 != 0 {
				attr |= foregroundBlue
			}
		case 40 <= n && n <= 47:
			attr = (attr & foregroundMask)
			if (n-40)&1 != 0 {
				attr |= backgroundRed
			}
			if (n-40)&2 != 0 {
				attr |= backgroundGreen
			}
			if (n-40)&4 != 0 {
				attr |= backgroundBlue
			}
		case 90 <= n && n <= 97:
			attr = (attr & backgroundMask)
			attr |= foregroundIntensity
			if (n-90)&1 != 0 {
				attr |= foregroundRed
			}
			if (n-90)&2 != 0 {
				attr |= foregroundGreen
			}
			if (n-90)&4 != 0 {
				attr |= foregroundBlue
			}
		case 100 <= n && n <= 107:
			attr = (attr & foregroundMask)
			attr |= backgroundIntensity
			if (n-100)&1 != 0 {
				attr |= backgroundRed
			}
			if (n-100)&2 != 0 {
				attr |= backgroundGreen
			}
			if (n-100)&4 != 0 {
				attr |= backgroundBlue
			}
		}
	}

	_, _, err := procSetConsoleTextAttribute.Call(uintptr(w.handle), uintptr(attr))
	return normalizeError(err)
}

func normalizeError(err error) error {
	if syserr, ok := err.(syscall.Errno); ok && syserr == 0 {
		return nil
	}
	return err
}
