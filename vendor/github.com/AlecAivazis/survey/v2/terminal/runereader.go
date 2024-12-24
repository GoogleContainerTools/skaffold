package terminal

import (
	"fmt"
	"unicode"

	"golang.org/x/text/width"
)

type RuneReader struct {
	stdio Stdio
	state runeReaderState
}

func NewRuneReader(stdio Stdio) *RuneReader {
	return &RuneReader{
		stdio: stdio,
		state: newRuneReaderState(stdio.In),
	}
}

func (rr *RuneReader) printChar(char rune, mask rune) error {
	// if we don't need to mask the input
	if mask == 0 {
		// just print the character the user pressed
		_, err := fmt.Fprintf(rr.stdio.Out, "%c", char)
		return err
	}
	// otherwise print the mask we were given
	_, err := fmt.Fprintf(rr.stdio.Out, "%c", mask)
	return err
}

type OnRuneFn func(rune, []rune) ([]rune, bool, error)

func (rr *RuneReader) ReadLine(mask rune, onRunes ...OnRuneFn) ([]rune, error) {
	return rr.ReadLineWithDefault(mask, []rune{}, onRunes...)
}

func (rr *RuneReader) ReadLineWithDefault(mask rune, d []rune, onRunes ...OnRuneFn) ([]rune, error) {
	line := []rune{}
	// we only care about horizontal displacements from the origin so start counting at 0
	index := 0

	cursor := &Cursor{
		In:  rr.stdio.In,
		Out: rr.stdio.Out,
	}

	onRune := func(r rune, line []rune) ([]rune, bool, error) {
		return line, false, nil
	}

	// if the user pressed a key the caller was interested in capturing
	if len(onRunes) > 0 {
		onRune = onRunes[0]
	}

	// we get the terminal width and height (if resized after this point the property might become invalid)
	terminalSize, _ := cursor.Size(rr.Buffer())
	// we set the current location of the cursor once
	cursorCurrent, _ := cursor.Location(rr.Buffer())

	increment := func() {
		if cursorCurrent.CursorIsAtLineEnd(terminalSize) {
			cursorCurrent.X = COORDINATE_SYSTEM_BEGIN
			cursorCurrent.Y++
		} else {
			cursorCurrent.X++
		}
	}
	decrement := func() {
		if cursorCurrent.CursorIsAtLineBegin() {
			cursorCurrent.X = terminalSize.X
			cursorCurrent.Y--
		} else {
			cursorCurrent.X--
		}
	}

	if len(d) > 0 {
		index = len(d)
		if _, err := fmt.Fprint(rr.stdio.Out, string(d)); err != nil {
			return d, err
		}
		line = d
		for range d {
			increment()
		}
	}

	for {
		// wait for some input
		r, _, err := rr.ReadRune()
		if err != nil {
			return line, err
		}

		if l, stop, err := onRune(r, line); stop || err != nil {
			return l, err
		}

		// if the user pressed enter or some other newline/termination like ctrl+d
		if r == '\r' || r == '\n' || r == KeyEndTransmission {
			// delete what's printed out on the console screen (cleanup)
			for index > 0 {
				if cursorCurrent.CursorIsAtLineBegin() {
					EraseLine(rr.stdio.Out, ERASE_LINE_END)
					cursor.PreviousLine(1)
					cursor.Forward(int(terminalSize.X))
				} else {
					cursor.Back(1)
				}
				decrement()
				index--
			}
			// move the cursor the a new line
			cursor.MoveNextLine(cursorCurrent, terminalSize)

			// we're done processing the input
			return line, nil
		}
		// if the user interrupts (ie with ctrl+c)
		if r == KeyInterrupt {
			// go to the beginning of the next line
			if _, err := fmt.Fprint(rr.stdio.Out, "\r\n"); err != nil {
				return line, err
			}

			// we're done processing the input, and treat interrupt like an error
			return line, InterruptErr
		}

		// allow for backspace/delete editing of inputs
		if r == KeyBackspace || r == KeyDelete {
			// and we're not at the beginning of the line
			if index > 0 && len(line) > 0 {
				// if we are at the end of the word
				if index == len(line) {
					// just remove the last letter from the internal representation
					// also count the number of cells the rune before the cursor occupied
					cells := runeWidth(line[len(line)-1])
					line = line[:len(line)-1]
					// go back one
					if cursorCurrent.X == 1 {
						cursor.PreviousLine(1)
						cursor.Forward(int(terminalSize.X))
					} else {
						cursor.Back(cells)
					}

					// clear the rest of the line
					EraseLine(rr.stdio.Out, ERASE_LINE_END)
				} else {
					// we need to remove a character from the middle of the word

					cells := runeWidth(line[index-1])

					// remove the current index from the list
					line = append(line[:index-1], line[index:]...)

					// save the current position of the cursor, as we have to move the cursor one back to erase the current symbol
					// and then move the cursor for each symbol in line[index-1:] to print it out, afterwards we want to restore
					// the cursor to its previous location.
					cursor.Save()

					// clear the rest of the line
					cursor.Back(cells)

					// print what comes after
					for _, char := range line[index-1:] {
						//Erase symbols which are left over from older print
						EraseLine(rr.stdio.Out, ERASE_LINE_END)
						// print characters to the new line appropriately
						if err := rr.printChar(char, mask); err != nil {
							return line, err
						}
					}
					// erase what's left over from last print
					if cursorCurrent.Y < terminalSize.Y {
						cursor.NextLine(1)
						EraseLine(rr.stdio.Out, ERASE_LINE_END)
					}
					// restore cursor
					cursor.Restore()
					if cursorCurrent.CursorIsAtLineBegin() {
						cursor.PreviousLine(1)
						cursor.Forward(int(terminalSize.X))
					} else {
						cursor.Back(cells)
					}
				}

				// decrement the index
				index--
				decrement()
			} else {
				// otherwise the user pressed backspace while at the beginning of the line
				_ = soundBell(rr.stdio.Out)
			}

			// we're done processing this key
			continue
		}

		// if the left arrow is pressed
		if r == KeyArrowLeft {
			// if we have space to the left
			if index > 0 {
				//move the cursor to the prev line if necessary
				if cursorCurrent.CursorIsAtLineBegin() {
					cursor.PreviousLine(1)
					cursor.Forward(int(terminalSize.X))
				} else {
					cursor.Back(runeWidth(line[index-1]))
				}
				//decrement the index
				index--
				decrement()

			} else {
				// otherwise we are at the beginning of where we started reading lines
				// sound the bell
				_ = soundBell(rr.stdio.Out)
			}

			// we're done processing this key press
			continue
		}

		// if the right arrow is pressed
		if r == KeyArrowRight {
			// if we have space to the right
			if index < len(line) {
				// move the cursor to the next line if necessary
				if cursorCurrent.CursorIsAtLineEnd(terminalSize) {
					cursor.NextLine(1)
				} else {
					cursor.Forward(runeWidth(line[index]))
				}
				index++
				increment()

			} else {
				// otherwise we are at the end of the word and can't go past
				// sound the bell
				_ = soundBell(rr.stdio.Out)
			}

			// we're done processing this key press
			continue
		}
		// the user pressed one of the special keys
		if r == SpecialKeyHome {
			for index > 0 {
				if cursorCurrent.CursorIsAtLineBegin() {
					cursor.PreviousLine(1)
					cursor.Forward(int(terminalSize.X))
					cursorCurrent.Y--
					cursorCurrent.X = terminalSize.X
				} else {
					cursor.Back(runeWidth(line[index-1]))
					cursorCurrent.X -= Short(runeWidth(line[index-1]))
				}
				index--
			}
			continue
			// user pressed end
		} else if r == SpecialKeyEnd {
			for index != len(line) {
				if cursorCurrent.CursorIsAtLineEnd(terminalSize) {
					cursor.NextLine(1)
					cursorCurrent.Y++
					cursorCurrent.X = COORDINATE_SYSTEM_BEGIN
				} else {
					cursor.Forward(runeWidth(line[index]))
					cursorCurrent.X += Short(runeWidth(line[index]))
				}
				index++
			}
			continue
			// user pressed forward delete key
		} else if r == SpecialKeyDelete {
			// if index at the end of the line nothing to delete
			if index != len(line) {
				// save the current position of the cursor, as we have to  erase the current symbol
				// and then move the cursor for each symbol in line[index:] to print it out, afterwards we want to restore
				// the cursor to its previous location.
				cursor.Save()
				// remove the symbol after the cursor
				line = append(line[:index], line[index+1:]...)
				// print the updated line
				for _, char := range line[index:] {
					EraseLine(rr.stdio.Out, ERASE_LINE_END)
					// print out the character
					if err := rr.printChar(char, mask); err != nil {
						return line, err
					}
				}
				// erase what's left on last line
				if cursorCurrent.Y < terminalSize.Y {
					cursor.NextLine(1)
					EraseLine(rr.stdio.Out, ERASE_LINE_END)
				}
				// restore cursor
				cursor.Restore()
				if len(line) == 0 || index == len(line) {
					EraseLine(rr.stdio.Out, ERASE_LINE_END)
				}
			}
			continue
		}

		// if the letter is another escape sequence
		if unicode.IsControl(r) || r == IgnoreKey {
			// ignore it
			continue
		}

		// the user pressed a regular key

		// if we are at the end of the line
		if index == len(line) {
			// just append the character at the end of the line
			line = append(line, r)
			// save the location of the cursor
			index++
			increment()
			// print out the character
			if err := rr.printChar(r, mask); err != nil {
				return line, err
			}
		} else {
			// we are in the middle of the word so we need to insert the character the user pressed
			line = append(line[:index], append([]rune{r}, line[index:]...)...)
			// save the current position of the cursor, as we have to move the cursor back to erase the current symbol
			// and then move for each symbol in line[index:] to print it out, afterwards we want to restore
			// cursor's location to its previous one.
			cursor.Save()
			EraseLine(rr.stdio.Out, ERASE_LINE_END)
			// remove the symbol after the cursor
			// print the updated line
			for _, char := range line[index:] {
				EraseLine(rr.stdio.Out, ERASE_LINE_END)
				// print out the character
				if err := rr.printChar(char, mask); err != nil {
					return line, err
				}
				increment()
			}
			// if we are at the last line, we want to visually insert a new line and append to it.
			if cursorCurrent.CursorIsAtLineEnd(terminalSize) && cursorCurrent.Y == terminalSize.Y {
				// add a new line to the terminal
				if _, err := fmt.Fprintln(rr.stdio.Out); err != nil {
					return line, err
				}
				// restore the position of the cursor horizontally
				cursor.Restore()
				// restore the position of the cursor vertically
				cursor.PreviousLine(1)
			} else {
				// restore cursor
				cursor.Restore()
			}
			// check if cursor needs to move to next line
			cursorCurrent, _ = cursor.Location(rr.Buffer())
			if cursorCurrent.CursorIsAtLineEnd(terminalSize) {
				cursor.NextLine(1)
			} else {
				cursor.Forward(runeWidth(r))
			}
			// increment the index
			index++
			increment()

		}
	}
}

// runeWidth returns the number of columns spanned by a rune when printed to the terminal
func runeWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}

	if !unicode.IsPrint(r) {
		return 0
	}
	return 1
}

// isAnsiMarker returns if a rune denotes the start of an ANSI sequence
func isAnsiMarker(r rune) bool {
	return r == '\x1B'
}

// isAnsiTerminator returns if a rune denotes the end of an ANSI sequence
func isAnsiTerminator(r rune) bool {
	return (r >= 0x40 && r <= 0x5a) || (r == 0x5e) || (r >= 0x60 && r <= 0x7e)
}

// StringWidth returns the visible width of a string when printed to the terminal
func StringWidth(str string) int {
	w := 0
	ansi := false

	for _, r := range str {
		// increase width only when outside of ANSI escape sequences
		if ansi || isAnsiMarker(r) {
			ansi = !isAnsiTerminator(r)
		} else {
			w += runeWidth(r)
		}
	}
	return w
}
