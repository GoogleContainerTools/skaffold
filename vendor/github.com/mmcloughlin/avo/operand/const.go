package operand

import (
	"fmt"
	"strconv"
	"strings"
)

// Constant represents a constant literal.
type Constant interface {
	Op
	Bytes() int
	constant()
}

//go:generate go run make_const.go -output zconst.go

// Special cases for floating point string representation.
//
// Issue 387 pointed out that floating point values that happen to be integers
// need to have a decimal point to be parsed correctly.

// String returns a representation the 32-bit float which is guaranteed to be
// parsed as a floating point constant by the Go assembler.
func (f F32) String() string { return asmfloat(float64(f), 32) }

// String returns a representation the 64-bit float which is guaranteed to be
// parsed as a floating point constant by the Go assembler.
func (f F64) String() string { return asmfloat(float64(f), 64) }

// asmfloat represents x as a string such that the assembler scanner will always
// recognize it as a float. Specifically, ensure that when x is an integral
// value, the result will still have a decimal point.
func asmfloat(x float64, bits int) string {
	s := strconv.FormatFloat(x, 'f', -1, bits)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return s
}

// String is a string constant.
type String string

// Asm returns an assembly syntax representation of the string s.
func (s String) Asm() string { return fmt.Sprintf("$%q", s) }

// Bytes returns the length of s.
func (s String) Bytes() int { return len(s) }

func (s String) constant() {}

// Imm returns an unsigned integer constant with size guessed from x.
func Imm(x uint64) Constant {
	switch {
	case uint64(uint8(x)) == x:
		return U8(x)
	case uint64(uint16(x)) == x:
		return U16(x)
	case uint64(uint32(x)) == x:
		return U32(x)
	}
	return U64(x)
}
