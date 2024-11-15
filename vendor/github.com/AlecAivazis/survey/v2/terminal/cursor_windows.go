package terminal

import (
	"bytes"
	"syscall"
	"unsafe"
)

var COORDINATE_SYSTEM_BEGIN Short = 0

// shared variable to save the cursor location from CursorSave()
var cursorLoc Coord

type Cursor struct {
	In  FileReader
	Out FileWriter
}

func (c *Cursor) Up(n int) error {
	return c.cursorMove(0, n)
}

func (c *Cursor) Down(n int) error {
	return c.cursorMove(0, -1*n)
}

func (c *Cursor) Forward(n int) error {
	return c.cursorMove(n, 0)
}

func (c *Cursor) Back(n int) error {
	return c.cursorMove(-1*n, 0)
}

// save the cursor location
func (c *Cursor) Save() error {
	loc, err := c.Location(nil)
	if err != nil {
		return err
	}
	cursorLoc = *loc
	return nil
}

func (c *Cursor) Restore() error {
	handle := syscall.Handle(c.Out.Fd())
	// restore it to the original position
	_, _, err := procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursorLoc))))
	return normalizeError(err)
}

func (cur Coord) CursorIsAtLineEnd(size *Coord) bool {
	return cur.X == size.X
}

func (cur Coord) CursorIsAtLineBegin() bool {
	return cur.X == 0
}

func (c *Cursor) cursorMove(x int, y int) error {
	handle := syscall.Handle(c.Out.Fd())

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi))); normalizeError(err) != nil {
		return err
	}

	var cursor Coord
	cursor.X = csbi.cursorPosition.X + Short(x)
	cursor.Y = csbi.cursorPosition.Y + Short(y)

	_, _, err := procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
	return normalizeError(err)
}

func (c *Cursor) NextLine(n int) error {
	if err := c.Up(n); err != nil {
		return err
	}
	return c.HorizontalAbsolute(0)
}

func (c *Cursor) PreviousLine(n int) error {
	if err := c.Down(n); err != nil {
		return err
	}
	return c.HorizontalAbsolute(0)
}

// for comparability purposes between windows
// in windows we don't have to print out a new line
func (c *Cursor) MoveNextLine(cur *Coord, terminalSize *Coord) error {
	return c.NextLine(1)
}

func (c *Cursor) HorizontalAbsolute(x int) error {
	handle := syscall.Handle(c.Out.Fd())

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi))); normalizeError(err) != nil {
		return err
	}

	var cursor Coord
	cursor.X = Short(x)
	cursor.Y = csbi.cursorPosition.Y

	if csbi.size.X < cursor.X {
		cursor.X = csbi.size.X
	}

	_, _, err := procSetConsoleCursorPosition.Call(uintptr(handle), uintptr(*(*int32)(unsafe.Pointer(&cursor))))
	return normalizeError(err)
}

func (c *Cursor) Show() error {
	handle := syscall.Handle(c.Out.Fd())

	var cci consoleCursorInfo
	if _, _, err := procGetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci))); normalizeError(err) != nil {
		return err
	}
	cci.visible = 1

	_, _, err := procSetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
	return normalizeError(err)
}

func (c *Cursor) Hide() error {
	handle := syscall.Handle(c.Out.Fd())

	var cci consoleCursorInfo
	if _, _, err := procGetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci))); normalizeError(err) != nil {
		return err
	}
	cci.visible = 0

	_, _, err := procSetConsoleCursorInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&cci)))
	return normalizeError(err)
}

func (c *Cursor) Location(buf *bytes.Buffer) (*Coord, error) {
	handle := syscall.Handle(c.Out.Fd())

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi))); normalizeError(err) != nil {
		return nil, err
	}

	return &csbi.cursorPosition, nil
}

func (c *Cursor) Size(buf *bytes.Buffer) (*Coord, error) {
	handle := syscall.Handle(c.Out.Fd())

	var csbi consoleScreenBufferInfo
	if _, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi))); normalizeError(err) != nil {
		return nil, err
	}
	// windows' coordinate system begins at (0, 0)
	csbi.size.X--
	csbi.size.Y--
	return &csbi.size, nil
}
