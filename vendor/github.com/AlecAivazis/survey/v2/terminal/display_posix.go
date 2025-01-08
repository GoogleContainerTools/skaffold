//go:build !windows
// +build !windows

package terminal

import (
	"fmt"
)

func EraseLine(out FileWriter, mode EraseLineMode) error {
	_, err := fmt.Fprintf(out, "\x1b[%dK", mode)
	return err
}
