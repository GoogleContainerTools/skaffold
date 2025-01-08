// Package src provides types for working with source files.
package src

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Position represents a position in a source file.
type Position struct {
	Filename string
	Line     int // 1-up
}

// FramePosition returns the Position of the given stack frame.
func FramePosition(f runtime.Frame) Position {
	return Position{
		Filename: f.File,
		Line:     f.Line,
	}
}

// IsValid reports whether the position is valid: Line must be positive, but
// Filename may be empty.
func (p Position) IsValid() bool {
	return p.Line > 0
}

// String represents Position as a string.
func (p Position) String() string {
	if !p.IsValid() {
		return "-"
	}
	var s string
	if p.Filename != "" {
		s += p.Filename + ":"
	}
	s += strconv.Itoa(p.Line)
	return s
}

// Rel returns Position relative to basepath. If the given filename cannot be
// expressed relative to basepath the position will be returned unchanged.
func (p Position) Rel(basepath string) Position {
	q := p
	if rel, err := filepath.Rel(basepath, q.Filename); err == nil {
		q.Filename = rel
	}
	return q
}

// Relwd returns Position relative to the current working directory. Returns p
// unchanged if the working directory cannot be determined, or the filename
// cannot be expressed relative to the working directory.
func (p Position) Relwd() Position {
	if wd, err := os.Getwd(); err == nil {
		return p.Rel(wd)
	}
	return p
}
