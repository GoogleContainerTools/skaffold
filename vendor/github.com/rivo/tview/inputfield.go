package tview

import (
	"math"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// InputField is a one-line box (three lines if there is a title) where the
// user can enter text. Use SetAcceptanceFunc() to accept or reject input,
// SetChangedFunc() to listen for changes, and SetMaskCharacter() to hide input
// from onlookers (e.g. for password input).
//
// The following keys can be used for navigation and editing:
//
//   - Left arrow: Move left by one character.
//   - Right arrow: Move right by one character.
//   - Home, Ctrl-A, Alt-a: Move to the beginning of the line.
//   - End, Ctrl-E, Alt-e: Move to the end of the line.
//   - Alt-left, Alt-b: Move left by one word.
//   - Alt-right, Alt-f: Move right by one word.
//   - Backspace: Delete the character before the cursor.
//   - Delete: Delete the character after the cursor.
//   - Ctrl-K: Delete from the cursor to the end of the line.
//   - Ctrl-W: Delete the last word before the cursor.
//   - Ctrl-U: Delete the entire line.
//
// See https://github.com/rivo/tview/wiki/InputField for an example.
type InputField struct {
	*Box

	// The text that was entered.
	text string

	// The text to be displayed before the input area.
	label string

	// The text to be displayed in the input area when "text" is empty.
	placeholder string

	// The label style.
	labelStyle tcell.Style

	// The style of the input area with input text.
	fieldStyle tcell.Style

	// The style of the input area with placeholder text.
	placeholderStyle tcell.Style

	// The screen width of the label area. A value of 0 means use the width of
	// the label text.
	labelWidth int

	// The screen width of the input area. A value of 0 means extend as much as
	// possible.
	fieldWidth int

	// A character to mask entered text (useful for password fields). A value of 0
	// disables masking.
	maskCharacter rune

	// The cursor position as a byte index into the text string.
	cursorPos int

	// An optional autocomplete function which receives the current text of the
	// input field and returns a slice of strings to be displayed in a drop-down
	// selection.
	autocomplete func(text string) []string

	// The List object which shows the selectable autocomplete entries. If not
	// nil, the list's main texts represent the current autocomplete entries.
	autocompleteList      *List
	autocompleteListMutex sync.Mutex

	// The styles of the autocomplete entries.
	autocompleteStyles struct {
		main       tcell.Style
		selected   tcell.Style
		background tcell.Color
	}

	// An optional function which may reject the last character that was entered.
	accept func(text string, ch rune) bool

	// An optional function which is called when the input has changed.
	changed func(text string)

	// An optional function which is called when the user indicated that they
	// are done entering text. The key which was pressed is provided (tab,
	// shift-tab, enter, or escape).
	done func(tcell.Key)

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)

	fieldX int // The x-coordinate of the input field as determined during the last call to Draw().
	offset int // The number of bytes of the text string skipped ahead while drawing.
}

// NewInputField returns a new input field.
func NewInputField() *InputField {
	i := &InputField{
		Box:              NewBox(),
		labelStyle:       tcell.StyleDefault.Foreground(Styles.SecondaryTextColor),
		fieldStyle:       tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.PrimaryTextColor),
		placeholderStyle: tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.ContrastSecondaryTextColor),
	}
	i.autocompleteStyles.main = tcell.StyleDefault.Foreground(Styles.PrimitiveBackgroundColor)
	i.autocompleteStyles.selected = tcell.StyleDefault.Background(Styles.PrimaryTextColor).Foreground(Styles.PrimitiveBackgroundColor)
	i.autocompleteStyles.background = Styles.MoreContrastBackgroundColor
	return i
}

// SetText sets the current text of the input field.
func (i *InputField) SetText(text string) *InputField {
	i.text = text
	i.cursorPos = len(text)
	if i.changed != nil {
		i.changed(text)
	}
	return i
}

// GetText returns the current text of the input field.
func (i *InputField) GetText() string {
	return i.text
}

// SetLabel sets the text to be displayed before the input area.
func (i *InputField) SetLabel(label string) *InputField {
	i.label = label
	return i
}

// GetLabel returns the text to be displayed before the input area.
func (i *InputField) GetLabel() string {
	return i.label
}

// SetLabelWidth sets the screen width of the label. A value of 0 will cause the
// primitive to use the width of the label string.
func (i *InputField) SetLabelWidth(width int) *InputField {
	i.labelWidth = width
	return i
}

// SetPlaceholder sets the text to be displayed when the input text is empty.
func (i *InputField) SetPlaceholder(text string) *InputField {
	i.placeholder = text
	return i
}

// SetLabelColor sets the text color of the label.
func (i *InputField) SetLabelColor(color tcell.Color) *InputField {
	i.labelStyle = i.labelStyle.Foreground(color)
	return i
}

// SetLabelStyle sets the style of the label.
func (i *InputField) SetLabelStyle(style tcell.Style) *InputField {
	i.labelStyle = style
	return i
}

// GetLabelStyle returns the style of the label.
func (i *InputField) GetLabelStyle() tcell.Style {
	return i.labelStyle
}

// SetFieldBackgroundColor sets the background color of the input area.
func (i *InputField) SetFieldBackgroundColor(color tcell.Color) *InputField {
	i.fieldStyle = i.fieldStyle.Background(color)
	return i
}

// SetFieldTextColor sets the text color of the input area.
func (i *InputField) SetFieldTextColor(color tcell.Color) *InputField {
	i.fieldStyle = i.fieldStyle.Foreground(color)
	return i
}

// SetFieldStyle sets the style of the input area (when no placeholder is
// shown).
func (i *InputField) SetFieldStyle(style tcell.Style) *InputField {
	i.fieldStyle = style
	return i
}

// GetFieldStyle returns the style of the input area (when no placeholder is
// shown).
func (i *InputField) GetFieldStyle() tcell.Style {
	return i.fieldStyle
}

// SetPlaceholderTextColor sets the text color of placeholder text.
func (i *InputField) SetPlaceholderTextColor(color tcell.Color) *InputField {
	i.placeholderStyle = i.placeholderStyle.Foreground(color)
	return i
}

// SetPlaceholderStyle sets the style of the input area (when a placeholder is
// shown).
func (i *InputField) SetPlaceholderStyle(style tcell.Style) *InputField {
	i.placeholderStyle = style
	return i
}

// GetPlaceholderStyle returns the style of the input area (when a placeholder
// is shown).
func (i *InputField) GetPlaceholderStyle() tcell.Style {
	return i.placeholderStyle
}

// SetAutocompleteStyles sets the colors and style of the autocomplete entries.
// For details, see List.SetMainTextStyle(), List.SetSelectedStyle(), and
// Box.SetBackgroundColor().
func (i *InputField) SetAutocompleteStyles(background tcell.Color, main, selected tcell.Style) *InputField {
	i.autocompleteStyles.background = background
	i.autocompleteStyles.main = main
	i.autocompleteStyles.selected = selected
	return i
}

// SetFormAttributes sets attributes shared by all form items.
func (i *InputField) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) FormItem {
	i.labelWidth = labelWidth
	i.backgroundColor = bgColor
	i.SetLabelColor(labelColor).
		SetFieldTextColor(fieldTextColor).
		SetFieldBackgroundColor(fieldBgColor)
	return i
}

// SetFieldWidth sets the screen width of the input area. A value of 0 means
// extend as much as possible.
func (i *InputField) SetFieldWidth(width int) *InputField {
	i.fieldWidth = width
	return i
}

// GetFieldWidth returns this primitive's field width.
func (i *InputField) GetFieldWidth() int {
	return i.fieldWidth
}

// SetMaskCharacter sets a character that masks user input on a screen. A value
// of 0 disables masking.
func (i *InputField) SetMaskCharacter(mask rune) *InputField {
	i.maskCharacter = mask
	return i
}

// SetAutocompleteFunc sets an autocomplete callback function which may return
// strings to be selected from a drop-down based on the current text of the
// input field. The drop-down appears only if len(entries) > 0. The callback is
// invoked in this function and whenever the current text changes or when
// Autocomplete() is called. Entries are cleared when the user selects an entry
// or presses Escape.
func (i *InputField) SetAutocompleteFunc(callback func(currentText string) (entries []string)) *InputField {
	i.autocomplete = callback
	i.Autocomplete()
	return i
}

// Autocomplete invokes the autocomplete callback (if there is one). If the
// length of the returned autocomplete entries slice is greater than 0, the
// input field will present the user with a corresponding drop-down list the
// next time the input field is drawn.
//
// It is safe to call this function from any goroutine. Note that the input
// field is not redrawn automatically unless called from the main goroutine
// (e.g. in response to events).
func (i *InputField) Autocomplete() *InputField {
	i.autocompleteListMutex.Lock()
	defer i.autocompleteListMutex.Unlock()
	if i.autocomplete == nil {
		return i
	}

	// Do we have any autocomplete entries?
	entries := i.autocomplete(i.text)
	if len(entries) == 0 {
		// No entries, no list.
		i.autocompleteList = nil
		return i
	}

	// Make a list if we have none.
	if i.autocompleteList == nil {
		i.autocompleteList = NewList()
		i.autocompleteList.ShowSecondaryText(false).
			SetMainTextStyle(i.autocompleteStyles.main).
			SetSelectedStyle(i.autocompleteStyles.selected).
			SetHighlightFullLine(true).
			SetBackgroundColor(i.autocompleteStyles.background)
	}

	// Fill it with the entries.
	currentEntry := -1
	suffixLength := 9999 // I'm just waiting for the day somebody opens an issue with this number being too small.
	i.autocompleteList.Clear()
	for index, entry := range entries {
		i.autocompleteList.AddItem(entry, "", 0, nil)
		if strings.HasPrefix(entry, i.text) && len(entry)-len(i.text) < suffixLength {
			currentEntry = index
			suffixLength = len(i.text) - len(entry)
		}
	}

	// Set the selection if we have one.
	if currentEntry >= 0 {
		i.autocompleteList.SetCurrentItem(currentEntry)
	}

	return i
}

// SetAcceptanceFunc sets a handler which may reject the last character that was
// entered (by returning false).
//
// This package defines a number of variables prefixed with InputField which may
// be used for common input (e.g. numbers, maximum text length).
func (i *InputField) SetAcceptanceFunc(handler func(textToCheck string, lastChar rune) bool) *InputField {
	i.accept = handler
	return i
}

// SetChangedFunc sets a handler which is called whenever the text of the input
// field has changed. It receives the current text (after the change).
func (i *InputField) SetChangedFunc(handler func(text string)) *InputField {
	i.changed = handler
	return i
}

// SetDoneFunc sets a handler which is called when the user is done entering
// text. The callback function is provided with the key that was pressed, which
// is one of the following:
//
//   - KeyEnter: Done entering text.
//   - KeyEscape: Abort text input.
//   - KeyTab: Move to the next field.
//   - KeyBacktab: Move to the previous field.
func (i *InputField) SetDoneFunc(handler func(key tcell.Key)) *InputField {
	i.done = handler
	return i
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (i *InputField) SetFinishedFunc(handler func(key tcell.Key)) FormItem {
	i.finished = handler
	return i
}

// Draw draws this primitive onto the screen.
func (i *InputField) Draw(screen tcell.Screen) {
	i.Box.DrawForSubclass(screen, i)

	// Prepare
	x, y, width, height := i.GetInnerRect()
	rightLimit := x + width
	if height < 1 || rightLimit <= x {
		return
	}

	// Draw label.
	_, labelBg, _ := i.labelStyle.Decompose()
	if i.labelWidth > 0 {
		labelWidth := i.labelWidth
		if labelWidth > rightLimit-x {
			labelWidth = rightLimit - x
		}
		printWithStyle(screen, i.label, x, y, 0, labelWidth, AlignLeft, i.labelStyle, labelBg == tcell.ColorDefault)
		x += labelWidth
	} else {
		_, drawnWidth, _, _ := printWithStyle(screen, i.label, x, y, 0, rightLimit-x, AlignLeft, i.labelStyle, labelBg == tcell.ColorDefault)
		x += drawnWidth
	}

	// Draw input area.
	i.fieldX = x
	fieldWidth := i.fieldWidth
	text := i.text
	inputStyle := i.fieldStyle
	placeholder := text == "" && i.placeholder != ""
	if placeholder {
		inputStyle = i.placeholderStyle
	}
	_, inputBg, _ := inputStyle.Decompose()
	if fieldWidth == 0 {
		fieldWidth = math.MaxInt32
	}
	if rightLimit-x < fieldWidth {
		fieldWidth = rightLimit - x
	}
	if inputBg != tcell.ColorDefault {
		for index := 0; index < fieldWidth; index++ {
			screen.SetContent(x+index, y, ' ', nil, inputStyle)
		}
	}

	// Text.
	var cursorScreenPos int
	if placeholder {
		// Draw placeholder text.
		printWithStyle(screen, Escape(i.placeholder), x, y, 0, fieldWidth, AlignLeft, i.placeholderStyle, true)
		i.offset = 0
	} else {
		// Draw entered text.
		if i.maskCharacter > 0 {
			text = strings.Repeat(string(i.maskCharacter), utf8.RuneCountInString(i.text))
		}
		if fieldWidth >= stringWidth(text) {
			// We have enough space for the full text.
			printWithStyle(screen, Escape(text), x, y, 0, fieldWidth, AlignLeft, i.fieldStyle, true)
			i.offset = 0
			iterateString(text, func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				if textPos >= i.cursorPos {
					return true
				}
				cursorScreenPos += screenWidth
				return false
			})
		} else {
			// The text doesn't fit. Where is the cursor?
			if i.cursorPos < 0 {
				i.cursorPos = 0
			} else if i.cursorPos > len(text) {
				i.cursorPos = len(text)
			}
			// Shift the text so the cursor is inside the field.
			var shiftLeft int
			if i.offset > i.cursorPos {
				i.offset = i.cursorPos
			} else if subWidth := stringWidth(text[i.offset:i.cursorPos]); subWidth > fieldWidth-1 {
				shiftLeft = subWidth - fieldWidth + 1
			}
			currentOffset := i.offset
			iterateString(text, func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				if textPos >= currentOffset {
					if shiftLeft > 0 {
						i.offset = textPos + textWidth
						shiftLeft -= screenWidth
					} else {
						if textPos+textWidth > i.cursorPos {
							return true
						}
						cursorScreenPos += screenWidth
					}
				}
				return false
			})
			printWithStyle(screen, Escape(text[i.offset:]), x, y, 0, fieldWidth, AlignLeft, i.fieldStyle, true)
		}
	}

	// Draw autocomplete list.
	i.autocompleteListMutex.Lock()
	defer i.autocompleteListMutex.Unlock()
	if i.autocompleteList != nil {
		// How much space do we need?
		lheight := i.autocompleteList.GetItemCount()
		lwidth := 0
		for index := 0; index < lheight; index++ {
			entry, _ := i.autocompleteList.GetItemText(index)
			width := TaggedStringWidth(entry)
			if width > lwidth {
				lwidth = width
			}
		}

		// We prefer to drop down but if there is no space, maybe drop up?
		lx := x
		ly := y + 1
		_, sheight := screen.Size()
		if ly+lheight >= sheight && ly-2 > lheight-ly {
			ly = y - lheight
			if ly < 0 {
				ly = 0
			}
		}
		if ly+lheight >= sheight {
			lheight = sheight - ly
		}
		i.autocompleteList.SetRect(lx, ly, lwidth, lheight)
		i.autocompleteList.Draw(screen)
	}

	// Set cursor.
	if i.HasFocus() {
		screen.ShowCursor(x+cursorScreenPos, y)
	}
}

// InputHandler returns the handler for this primitive.
func (i *InputField) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return i.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		// Trigger changed events.
		currentText := i.text
		defer func() {
			if i.text != currentText {
				i.Autocomplete()
				if i.changed != nil {
					i.changed(i.text)
				}
			}
		}()

		// Movement functions.
		home := func() { i.cursorPos = 0 }
		end := func() { i.cursorPos = len(i.text) }
		moveLeft := func() {
			iterateStringReverse(i.text[:i.cursorPos], func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				i.cursorPos -= textWidth
				return true
			})
		}
		moveRight := func() {
			iterateString(i.text[i.cursorPos:], func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				i.cursorPos += textWidth
				return true
			})
		}
		moveWordLeft := func() {
			i.cursorPos = len(regexp.MustCompile(`\S+\s*$`).ReplaceAllString(i.text[:i.cursorPos], ""))
		}
		moveWordRight := func() {
			i.cursorPos = len(i.text) - len(regexp.MustCompile(`^\s*\S+\s*`).ReplaceAllString(i.text[i.cursorPos:], ""))
		}

		// Add character function. Returns whether or not the rune character is
		// accepted.
		add := func(r rune) bool {
			newText := i.text[:i.cursorPos] + string(r) + i.text[i.cursorPos:]
			if i.accept != nil && !i.accept(newText, r) {
				return false
			}
			i.text = newText
			i.cursorPos += len(string(r))
			return true
		}

		// Change the autocomplete selection.
		autocompleteSelect := func(offset int) {
			count := i.autocompleteList.GetItemCount()
			newEntry := i.autocompleteList.GetCurrentItem() + offset
			if newEntry >= count {
				newEntry = 0
			} else if newEntry < 0 {
				newEntry = count - 1
			}
			i.autocompleteList.SetCurrentItem(newEntry)
			currentText, _ = i.autocompleteList.GetItemText(newEntry) // Don't trigger changed function twice.
			currentText = stripTags(currentText)
			i.SetText(currentText)
		}

		// Finish up.
		finish := func(key tcell.Key) {
			if i.done != nil {
				i.done(key)
			}
			if i.finished != nil {
				i.finished(key)
			}
		}

		// Process key event.
		i.autocompleteListMutex.Lock()
		defer i.autocompleteListMutex.Unlock()
		switch key := event.Key(); key {
		case tcell.KeyRune: // Regular character.
			if event.Modifiers()&tcell.ModAlt > 0 {
				// We accept some Alt- key combinations.
				switch event.Rune() {
				case 'a': // Home.
					home()
				case 'e': // End.
					end()
				case 'b': // Move word left.
					moveWordLeft()
				case 'f': // Move word right.
					moveWordRight()
				default:
					if !add(event.Rune()) {
						return
					}
				}
			} else {
				// Other keys are simply accepted as regular characters.
				if !add(event.Rune()) {
					return
				}
			}
		case tcell.KeyCtrlU: // Delete all.
			i.text = ""
			i.cursorPos = 0
		case tcell.KeyCtrlK: // Delete until the end of the line.
			i.text = i.text[:i.cursorPos]
		case tcell.KeyCtrlW: // Delete last word.
			lastWord := regexp.MustCompile(`\S+\s*$`)
			newText := lastWord.ReplaceAllString(i.text[:i.cursorPos], "") + i.text[i.cursorPos:]
			i.cursorPos -= len(i.text) - len(newText)
			i.text = newText
		case tcell.KeyBackspace, tcell.KeyBackspace2: // Delete character before the cursor.
			iterateStringReverse(i.text[:i.cursorPos], func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				i.text = i.text[:textPos] + i.text[textPos+textWidth:]
				i.cursorPos -= textWidth
				return true
			})
			if i.offset >= i.cursorPos {
				i.offset = 0
			}
		case tcell.KeyDelete, tcell.KeyCtrlD: // Delete character after the cursor.
			iterateString(i.text[i.cursorPos:], func(main rune, comb []rune, textPos, textWidth, screenPos, screenWidth int) bool {
				i.text = i.text[:i.cursorPos] + i.text[i.cursorPos+textWidth:]
				return true
			})
		case tcell.KeyLeft:
			if event.Modifiers()&tcell.ModAlt > 0 {
				moveWordLeft()
			} else {
				moveLeft()
			}
		case tcell.KeyCtrlB:
			moveLeft()
		case tcell.KeyRight:
			if event.Modifiers()&tcell.ModAlt > 0 {
				moveWordRight()
			} else {
				moveRight()
			}
		case tcell.KeyCtrlF:
			moveRight()
		case tcell.KeyHome, tcell.KeyCtrlA:
			home()
		case tcell.KeyEnd, tcell.KeyCtrlE:
			end()
		case tcell.KeyEnter:
			if i.autocompleteList != nil {
				autocompleteSelect(0)
				i.autocompleteList = nil
			} else {
				finish(key)
			}
		case tcell.KeyEscape:
			if i.autocompleteList != nil {
				i.autocompleteList = nil
			} else {
				finish(key)
			}
		case tcell.KeyTab:
			if i.autocompleteList != nil {
				autocompleteSelect(0)
			} else {
				finish(key)
			}
		case tcell.KeyDown:
			if i.autocompleteList != nil {
				autocompleteSelect(1)
			} else {
				finish(key)
			}
		case tcell.KeyUp, tcell.KeyBacktab: // Autocomplete selection.
			if i.autocompleteList != nil {
				autocompleteSelect(-1)
			} else {
				finish(key)
			}
		}
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (i *InputField) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return i.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		x, y := event.Position()
		_, rectY, _, _ := i.GetInnerRect()
		if !i.InRect(x, y) {
			return false, nil
		}

		// Process mouse event.
		if action == MouseLeftClick && y == rectY {
			// Determine where to place the cursor.
			if x >= i.fieldX {
				if !iterateString(i.text[i.offset:], func(main rune, comb []rune, textPos int, textWidth int, screenPos int, screenWidth int) bool {
					if x-i.fieldX < screenPos+screenWidth {
						i.cursorPos = textPos + i.offset
						return true
					}
					return false
				}) {
					i.cursorPos = len(i.text)
				}
			}
			setFocus(i)
			consumed = true
		}

		return
	})
}
