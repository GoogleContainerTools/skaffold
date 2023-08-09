package tview

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// The states of the ANSI escape code parser.
const (
	ansiText = iota
	ansiEscape
	ansiSubstring
	ansiControlSequence
)

// ansi is a io.Writer which translates ANSI escape codes into tview color
// tags.
type ansi struct {
	io.Writer

	// Reusable buffers.
	buffer                        *bytes.Buffer // The entire output text of one Write().
	csiParameter, csiIntermediate *bytes.Buffer // Partial CSI strings.
	attributes                    string        // The buffer's current text attributes (a tview attribute string).

	// The current state of the parser. One of the ansi constants.
	state int
}

// ANSIWriter returns an io.Writer which translates any ANSI escape codes
// written to it into tview color tags. Other escape codes don't have an effect
// and are simply removed. The translated text is written to the provided
// writer.
func ANSIWriter(writer io.Writer) io.Writer {
	return &ansi{
		Writer:          writer,
		buffer:          new(bytes.Buffer),
		csiParameter:    new(bytes.Buffer),
		csiIntermediate: new(bytes.Buffer),
		state:           ansiText,
	}
}

// Write parses the given text as a string of runes, translates ANSI escape
// codes to color tags and writes them to the output writer.
func (a *ansi) Write(text []byte) (int, error) {
	defer func() {
		a.buffer.Reset()
	}()

	for _, r := range string(text) {
		switch a.state {

		// We just entered an escape sequence.
		case ansiEscape:
			switch r {
			case '[': // Control Sequence Introducer.
				a.csiParameter.Reset()
				a.csiIntermediate.Reset()
				a.state = ansiControlSequence
			case 'c': // Reset.
				fmt.Fprint(a.buffer, "[-:-:-]")
				a.state = ansiText
			case 'P', ']', 'X', '^', '_': // Substrings and commands.
				a.state = ansiSubstring
			default: // Ignore.
				a.state = ansiText
			}

		// CSI Sequences.
		case ansiControlSequence:
			switch {
			case r >= 0x30 && r <= 0x3f: // Parameter bytes.
				if _, err := a.csiParameter.WriteRune(r); err != nil {
					return 0, err
				}
			case r >= 0x20 && r <= 0x2f: // Intermediate bytes.
				if _, err := a.csiIntermediate.WriteRune(r); err != nil {
					return 0, err
				}
			case r >= 0x40 && r <= 0x7e: // Final byte.
				switch r {
				case 'E': // Next line.
					count, _ := strconv.Atoi(a.csiParameter.String())
					if count == 0 {
						count = 1
					}
					fmt.Fprint(a.buffer, strings.Repeat("\n", count))
				case 'm': // Select Graphic Rendition.
					var background, foreground string
					params := a.csiParameter.String()
					fields := strings.Split(params, ";")
					if len(params) == 0 || len(fields) == 1 && fields[0] == "0" {
						// Reset.
						a.attributes = ""
						if _, err := a.buffer.WriteString("[-:-:-]"); err != nil {
							return 0, err
						}
						break
					}
					lookupColor := func(colorNumber int) string {
						if colorNumber < 0 || colorNumber > 15 {
							return "black"
						}
						return []string{
							"black",
							"maroon",
							"green",
							"olive",
							"navy",
							"purple",
							"teal",
							"silver",
							"gray",
							"red",
							"lime",
							"yellow",
							"blue",
							"fuchsia",
							"aqua",
							"white",
						}[colorNumber]
					}
				FieldLoop:
					for index, field := range fields {
						switch field {
						case "1", "01":
							if !strings.ContainsRune(a.attributes, 'b') {
								a.attributes += "b"
							}
						case "2", "02":
							if !strings.ContainsRune(a.attributes, 'd') {
								a.attributes += "d"
							}
						case "4", "04":
							if !strings.ContainsRune(a.attributes, 'u') {
								a.attributes += "u"
							}
						case "5", "05":
							if !strings.ContainsRune(a.attributes, 'l') {
								a.attributes += "l"
							}
						case "22":
							if i := strings.IndexRune(a.attributes, 'b'); i >= 0 {
								a.attributes = a.attributes[:i] + a.attributes[i+1:]
							}
							if i := strings.IndexRune(a.attributes, 'd'); i >= 0 {
								a.attributes = a.attributes[:i] + a.attributes[i+1:]
							}
						case "24":
							if i := strings.IndexRune(a.attributes, 'u'); i >= 0 {
								a.attributes = a.attributes[:i] + a.attributes[i+1:]
							}
						case "25":
							if i := strings.IndexRune(a.attributes, 'l'); i >= 0 {
								a.attributes = a.attributes[:i] + a.attributes[i+1:]
							}
						case "30", "31", "32", "33", "34", "35", "36", "37":
							colorNumber, _ := strconv.Atoi(field)
							foreground = lookupColor(colorNumber - 30)
						case "39":
							foreground = "-"
						case "40", "41", "42", "43", "44", "45", "46", "47":
							colorNumber, _ := strconv.Atoi(field)
							background = lookupColor(colorNumber - 40)
						case "49":
							background = "-"
						case "90", "91", "92", "93", "94", "95", "96", "97":
							colorNumber, _ := strconv.Atoi(field)
							foreground = lookupColor(colorNumber - 82)
						case "100", "101", "102", "103", "104", "105", "106", "107":
							colorNumber, _ := strconv.Atoi(field)
							background = lookupColor(colorNumber - 92)
						case "38", "48":
							var color string
							if len(fields) > index+1 {
								if fields[index+1] == "5" && len(fields) > index+2 { // 8-bit colors.
									colorNumber, _ := strconv.Atoi(fields[index+2])
									if colorNumber <= 15 {
										color = lookupColor(colorNumber)
									} else if colorNumber <= 231 {
										red := (colorNumber - 16) / 36
										green := ((colorNumber - 16) / 6) % 6
										blue := (colorNumber - 16) % 6
										color = fmt.Sprintf("#%02x%02x%02x", 255*red/5, 255*green/5, 255*blue/5)
									} else if colorNumber <= 255 {
										grey := 255 * (colorNumber - 232) / 23
										color = fmt.Sprintf("#%02x%02x%02x", grey, grey, grey)
									}
								} else if fields[index+1] == "2" && len(fields) > index+4 { // 24-bit colors.
									red, _ := strconv.Atoi(fields[index+2])
									green, _ := strconv.Atoi(fields[index+3])
									blue, _ := strconv.Atoi(fields[index+4])
									color = fmt.Sprintf("#%02x%02x%02x", red, green, blue)
								}
							}
							if len(color) > 0 {
								if field == "38" {
									foreground = color
								} else {
									background = color
								}
							}
							break FieldLoop
						}
					}
					var colon string
					if len(a.attributes) > 0 {
						colon = ":"
					}
					if len(foreground) > 0 || len(background) > 0 || len(a.attributes) > 0 {
						fmt.Fprintf(a.buffer, "[%s:%s%s%s]", foreground, background, colon, a.attributes)
					}
				}
				a.state = ansiText
			default: // Undefined byte.
				a.state = ansiText // Abort CSI.
			}

			// We just entered a substring/command sequence.
		case ansiSubstring:
			if r == 27 { // Most likely the end of the substring.
				a.state = ansiEscape
			} // Ignore all other characters.

			// "ansiText" and all others.
		default:
			if r == 27 {
				// This is the start of an escape sequence.
				a.state = ansiEscape
			} else {
				// Just a regular rune. Send to buffer.
				if _, err := a.buffer.WriteRune(r); err != nil {
					return 0, err
				}
			}
		}
	}

	// Write buffer to target writer.
	n, err := a.buffer.WriteTo(a.Writer)
	if err != nil {
		return int(n), err
	}
	return len(text), nil
}

// TranslateANSI replaces ANSI escape sequences found in the provided string
// with tview's color tags and returns the resulting string.
func TranslateANSI(text string) string {
	var buffer bytes.Buffer
	writer := ANSIWriter(&buffer)
	writer.Write([]byte(text))
	return buffer.String()
}
