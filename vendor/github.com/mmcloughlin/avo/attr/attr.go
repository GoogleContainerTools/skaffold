// Package attr provides attributes for text and data sections.
package attr

import (
	"fmt"
	"math/bits"
	"strings"
)

// Attribute represents TEXT or DATA flags.
type Attribute uint16

//go:generate go run make_textflag.go -output ztextflag.go

// Asm returns a representation of the attributes in assembly syntax. This may use macros from "textflags.h"; see ContainsTextFlags() to determine if this header is required.
func (a Attribute) Asm() string {
	parts, rest := a.split()
	if len(parts) == 0 || rest != 0 {
		parts = append(parts, fmt.Sprintf("%d", rest))
	}
	return strings.Join(parts, "|")
}

// ContainsTextFlags returns whether the Asm() representation requires macros in "textflags.h".
func (a Attribute) ContainsTextFlags() bool {
	flags, _ := a.split()
	return len(flags) > 0
}

// split splits a into known flags and any remaining bits.
func (a Attribute) split() ([]string, Attribute) {
	var flags []string
	var rest Attribute
	for a != 0 {
		i := uint(bits.TrailingZeros16(uint16(a)))
		bit := Attribute(1) << i
		if flag := attrname[bit]; flag != "" {
			flags = append(flags, flag)
		} else {
			rest |= bit
		}
		a ^= bit
	}
	return flags, rest
}
