// Package prnt provides common functionality for code generators.
package prnt

import (
	"bytes"
	"fmt"
	"go/build/constraint"
	"io"
	"strings"
)

// Generator provides convenience methods for code generators. In particular it
// provides fmt-like methods which print to an internal buffer. It also allows
// any errors to be stored so they can be checked at the end, rather than having
// error checks obscuring the code generation.
type Generator struct {
	buf     bytes.Buffer
	level   int    // current indentation level
	indent  string // indentation string
	pending bool   // if there's a pending indentation
	err     error  // saved error from printing
}

// Raw provides direct access to the underlying output stream.
func (g *Generator) Raw() io.Writer {
	return &g.buf
}

// SetIndentString sets the string used for one level of indentation. Use
// Indent() and Dedent() to control indent level.
func (g *Generator) SetIndentString(indent string) {
	g.indent = indent
}

// Indent increments the indent level.
func (g *Generator) Indent() {
	g.level++
}

// Dedent decrements the indent level.
func (g *Generator) Dedent() {
	g.level--
}

// Linef prints formatted output terminated with a new line.
func (g *Generator) Linef(format string, args ...any) {
	g.Printf(format, args...)
	g.NL()
}

// Printf prints to the internal buffer.
func (g *Generator) Printf(format string, args ...any) {
	if g.err != nil {
		return
	}
	if g.pending {
		indent := strings.Repeat(g.indent, g.level)
		format = indent + format
		g.pending = false
	}
	_, err := fmt.Fprintf(&g.buf, format, args...)
	g.AddError(err)
}

// NL prints a new line.
func (g *Generator) NL() {
	g.Printf("\n")
	g.pending = true
}

// Comment writes comment lines prefixed with "// ".
func (g *Generator) Comment(lines ...string) {
	for _, line := range lines {
		line = strings.TrimSpace("// " + line)
		g.Printf("%s\n", line)
	}
}

// BuildConstraint outputs a build constraint.
func (g *Generator) BuildConstraint(expr string) {
	line := fmt.Sprintf("//go:build %s", expr)
	if _, err := constraint.Parse(line); err != nil {
		g.AddError(err)
	}
	g.Linef(line)
}

// AddError records an error in code generation. The first non-nil error will
// prevent printing operations from writing anything else, and the error will be
// returned from Result().
func (g *Generator) AddError(err error) {
	if err != nil && g.err == nil {
		g.err = err
	}
}

// Result returns the printed bytes. If any error was recorded with AddError
// during code generation, the first such error will be returned here.
func (g *Generator) Result() ([]byte, error) {
	return g.buf.Bytes(), g.err
}
