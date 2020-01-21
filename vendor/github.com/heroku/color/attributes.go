package color

import (
	"fmt"
)

// Attribute defines a single SGR Code.
type Attribute uint64

// String returns color code as a string.
func (a Attribute) String() string {
	return attributeToSGRCode[a]
}

// Name returns a human readable name for an Attribute.
func (a Attribute) Name() string {
	m := map[Attribute]string{
		Reset:        "Reset",
		Bold:         "Bold",
		Faint:        "Faint",
		Italic:       "Italic",
		Underline:    "Underline",
		BlinkSlow:    "BlinkSlow",
		BlinkRapid:   "BlinkRapid",
		ReverseVideo: "ReverseVideo",
		Concealed:    "Concealed",
		CrossedOut:   "CrossedOut",
		FgBlack:      "FgBlack",
		FgRed:        "FgRed",
		FgGreen:      "FgGreen",
		FgYellow:     "FgYellow",
		FgBlue:       "FgBlue",
		FgMagenta:    "FgMagenta",
		FgCyan:       "FgCyan",
		FgWhite:      "FgWhite",
		FgHiBlack:    "FgHiBlack",
		FgHiRed:      "FgHiRed",
		FgHiGreen:    "FgHiGreen",
		FgHiYellow:   "FgHiYellow",
		FgHiBlue:     "FgHiBlue",
		FgHiMagenta:  "FgHiMagenta",
		FgHiCyan:     "FgHiCyan",
		FgHiWhite:    "FgHiWhite",
		BgBlack:      "BgBlack",
		BgRed:        "BgRed",
		BgGreen:      "BgGreen",
		BgYellow:     "BgYellow",
		BgBlue:       "BgBlue",
		BgMagenta:    "BgMagenta",
		BgCyan:       "BgCyan",
		BgWhite:      "BgWhite",
		BgHiBlack:    "BgHiBlack",
		BgHiRed:      "BgHiRed",
		BgHiGreen:    "BgHiGreen",
		BgHiYellow:   "BgHiYellow",
		BgHiBlue:     "BgHiBlue",
		BgHiMagenta:  "BgHiMagenta",
		BgHiCyan:     "BgHiCyan",
		BgHiWhite:    "BgHiWhite",
	}
	if s, ok := m[a]; ok {
		return s
	}
	return fmt.Sprintf("unknown color %d", a)
}

const (
	Reset Attribute = 1 << iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
	FgBlack
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
	FgHiBlack
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
	BgBlack
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
	BgHiBlack
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

var attributeToSGRCode = map[Attribute]string{
	Reset:        "0",
	Bold:         "1",
	Faint:        "2",
	Italic:       "3",
	Underline:    "4",
	BlinkSlow:    "5",
	BlinkRapid:   "6",
	ReverseVideo: "7",
	Concealed:    "8",
	CrossedOut:   "9",
	FgBlack:      "30",
	FgRed:        "31",
	FgGreen:      "32",
	FgYellow:     "33",
	FgBlue:       "34",
	FgMagenta:    "35",
	FgCyan:       "36",
	FgWhite:      "37",
	BgBlack:      "40",
	BgRed:        "41",
	BgGreen:      "42",
	BgYellow:     "43",
	BgBlue:       "44",
	BgMagenta:    "45",
	BgCyan:       "46",
	BgWhite:      "47",
	FgHiBlack:    "90",
	FgHiRed:      "91",
	FgHiGreen:    "92",
	FgHiYellow:   "93",
	FgHiBlue:     "94",
	FgHiMagenta:  "95",
	FgHiCyan:     "96",
	FgHiWhite:    "97",
	BgHiBlack:    "100",
	BgHiRed:      "101",
	BgHiGreen:    "102",
	BgHiYellow:   "103",
	BgHiBlue:     "104",
	BgHiMagenta:  "105",
	BgHiCyan:     "106",
	BgHiWhite:    "107",
}

func to_codes(attrs []Attribute) []string {
	codes := make([]string, len(attrs))
	for i := 0; i < len(attrs); i++ {
		code, ok := attributeToSGRCode[attrs[i]]
		if !ok {
			return nil
		}
		codes[i] = code
	}
	return codes
}

func to_key(attr []Attribute) Attribute {
	var key Attribute
	for i := 0; i < len(attr); i++ {
		key |= attr[i]
	}
	return key
}
