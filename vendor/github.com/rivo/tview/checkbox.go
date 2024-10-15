package tview

import (
	"github.com/gdamore/tcell/v2"
)

// Checkbox implements a simple box for boolean values which can be checked and
// unchecked.
//
// See https://github.com/rivo/tview/wiki/Checkbox for an example.
type Checkbox struct {
	*Box

	// Whether or not this checkbox is disabled/read-only.
	disabled bool

	// Whether or not this box is checked.
	checked bool

	// The text to be displayed before the input area.
	label string

	// The screen width of the label area. A value of 0 means use the width of
	// the label text.
	labelWidth int

	// The label style.
	labelStyle tcell.Style

	// The style of the unchecked checkbox.
	uncheckedStyle tcell.Style

	// The style of the checked checkbox.
	checkedStyle tcell.Style

	// Teh style of the checkbox when it is currently focused.
	focusStyle tcell.Style

	// The string used to display an unchecked box.
	uncheckedString string

	// The string used to display a checked box.
	checkedString string

	// An optional function which is called when the user changes the checked
	// state of this checkbox.
	changed func(checked bool)

	// An optional function which is called when the user indicated that they
	// are done entering text. The key which was pressed is provided (tab,
	// shift-tab, or escape).
	done func(tcell.Key)

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)
}

// NewCheckbox returns a new input field.
func NewCheckbox() *Checkbox {
	return &Checkbox{
		Box:             NewBox(),
		labelStyle:      tcell.StyleDefault.Foreground(Styles.SecondaryTextColor),
		uncheckedStyle:  tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.PrimaryTextColor),
		checkedStyle:    tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.PrimaryTextColor),
		focusStyle:      tcell.StyleDefault.Background(Styles.PrimaryTextColor).Foreground(Styles.ContrastBackgroundColor),
		uncheckedString: " ",
		checkedString:   "X",
	}
}

// SetChecked sets the state of the checkbox. This also triggers the "changed"
// callback if the state changes with this call.
func (c *Checkbox) SetChecked(checked bool) *Checkbox {
	if c.checked != checked {
		if c.changed != nil {
			c.changed(checked)
		}
		c.checked = checked
	}
	return c
}

// IsChecked returns whether or not the box is checked.
func (c *Checkbox) IsChecked() bool {
	return c.checked
}

// SetLabel sets the text to be displayed before the input area.
func (c *Checkbox) SetLabel(label string) *Checkbox {
	c.label = label
	return c
}

// GetLabel returns the text to be displayed before the input area.
func (c *Checkbox) GetLabel() string {
	return c.label
}

// SetLabelWidth sets the screen width of the label. A value of 0 will cause the
// primitive to use the width of the label string.
func (c *Checkbox) SetLabelWidth(width int) *Checkbox {
	c.labelWidth = width
	return c
}

// SetLabelColor sets the color of the label.
func (c *Checkbox) SetLabelColor(color tcell.Color) *Checkbox {
	c.labelStyle = c.labelStyle.Foreground(color)
	return c
}

// SetLabelStyle sets the style of the label.
func (c *Checkbox) SetLabelStyle(style tcell.Style) *Checkbox {
	c.labelStyle = style
	return c
}

// SetFieldBackgroundColor sets the background color of the input area.
func (c *Checkbox) SetFieldBackgroundColor(color tcell.Color) *Checkbox {
	c.uncheckedStyle = c.uncheckedStyle.Background(color)
	c.checkedStyle = c.checkedStyle.Background(color)
	c.focusStyle = c.focusStyle.Foreground(color)
	return c
}

// SetFieldTextColor sets the text color of the input area.
func (c *Checkbox) SetFieldTextColor(color tcell.Color) *Checkbox {
	c.uncheckedStyle = c.uncheckedStyle.Foreground(color)
	c.checkedStyle = c.checkedStyle.Foreground(color)
	c.focusStyle = c.focusStyle.Background(color)
	return c
}

// SetUncheckedStyle sets the style of the unchecked checkbox.
func (c *Checkbox) SetUncheckedStyle(style tcell.Style) *Checkbox {
	c.uncheckedStyle = style
	return c
}

// SetCheckedStyle sets the style of the checked checkbox.
func (c *Checkbox) SetCheckedStyle(style tcell.Style) *Checkbox {
	c.checkedStyle = style
	return c
}

// SetActivatedStyle sets the style of the checkbox when it is currently
// focused.
func (c *Checkbox) SetActivatedStyle(style tcell.Style) *Checkbox {
	c.focusStyle = style
	return c
}

// SetCheckedString sets the string to be displayed when the checkbox is
// checked (defaults to "X"). The string may contain color tags (consider
// adapting the checkbox's various styles accordingly). See [Escape] in
// case you want to display square brackets.
func (c *Checkbox) SetCheckedString(checked string) *Checkbox {
	c.checkedString = checked
	return c
}

// SetUncheckedString sets the string to be displayed when the checkbox is
// not checked (defaults to the empty space " "). The string may contain color
// tags (consider adapting the checkbox's various styles accordingly). See
// [Escape] in case you want to display square brackets.
func (c *Checkbox) SetUncheckedString(unchecked string) *Checkbox {
	c.uncheckedString = unchecked
	return c
}

// SetFormAttributes sets attributes shared by all form items.
func (c *Checkbox) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) FormItem {
	c.labelWidth = labelWidth
	c.SetLabelColor(labelColor)
	c.backgroundColor = bgColor
	c.SetFieldTextColor(fieldTextColor)
	c.SetFieldBackgroundColor(fieldBgColor)
	return c
}

// GetFieldWidth returns this primitive's field width.
func (c *Checkbox) GetFieldWidth() int {
	return 1
}

// GetFieldHeight returns this primitive's field height.
func (c *Checkbox) GetFieldHeight() int {
	return 1
}

// SetDisabled sets whether or not the item is disabled / read-only.
func (c *Checkbox) SetDisabled(disabled bool) FormItem {
	c.disabled = disabled
	if c.finished != nil {
		c.finished(-1)
	}
	return c
}

// SetChangedFunc sets a handler which is called when the checked state of this
// checkbox was changed. The handler function receives the new state.
func (c *Checkbox) SetChangedFunc(handler func(checked bool)) *Checkbox {
	c.changed = handler
	return c
}

// SetDoneFunc sets a handler which is called when the user is done using the
// checkbox. The callback function is provided with the key that was pressed,
// which is one of the following:
//
//   - KeyEscape: Abort text input.
//   - KeyTab: Move to the next field.
//   - KeyBacktab: Move to the previous field.
func (c *Checkbox) SetDoneFunc(handler func(key tcell.Key)) *Checkbox {
	c.done = handler
	return c
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (c *Checkbox) SetFinishedFunc(handler func(key tcell.Key)) FormItem {
	c.finished = handler
	return c
}

// Focus is called when this primitive receives focus.
func (c *Checkbox) Focus(delegate func(p Primitive)) {
	// If we're part of a form and this item is disabled, there's nothing the
	// user can do here so we're finished.
	if c.finished != nil && c.disabled {
		c.finished(-1)
		return
	}

	c.Box.Focus(delegate)
}

// Draw draws this primitive onto the screen.
func (c *Checkbox) Draw(screen tcell.Screen) {
	c.Box.DrawForSubclass(screen, c)

	// Prepare
	x, y, width, height := c.GetInnerRect()
	rightLimit := x + width
	if height < 1 || rightLimit <= x {
		return
	}

	// Draw label.
	_, labelBg, _ := c.labelStyle.Decompose()
	if c.labelWidth > 0 {
		labelWidth := c.labelWidth
		if labelWidth > width {
			labelWidth = width
		}
		printWithStyle(screen, c.label, x, y, 0, labelWidth, AlignLeft, c.labelStyle, labelBg == tcell.ColorDefault)
		x += labelWidth
		width -= labelWidth
	} else {
		_, _, drawnWidth := printWithStyle(screen, c.label, x, y, 0, width, AlignLeft, c.labelStyle, labelBg == tcell.ColorDefault)
		x += drawnWidth
		width -= drawnWidth
	}

	// Draw checkbox.
	str := c.uncheckedString
	style := c.uncheckedStyle
	if c.checked {
		str = c.checkedString
		style = c.checkedStyle
	}
	if c.disabled {
		style = style.Background(c.backgroundColor)
	}
	if c.HasFocus() {
		style = c.focusStyle
	}
	printWithStyle(screen, str, x, y, 0, width, AlignLeft, style, c.disabled)
}

// InputHandler returns the handler for this primitive.
func (c *Checkbox) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return c.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		if c.disabled {
			return
		}

		// Process key event.
		switch key := event.Key(); key {
		case tcell.KeyRune, tcell.KeyEnter: // Check.
			if key == tcell.KeyRune && event.Rune() != ' ' {
				break
			}
			c.checked = !c.checked
			if c.changed != nil {
				c.changed(c.checked)
			}
		case tcell.KeyTab, tcell.KeyBacktab, tcell.KeyEscape: // We're done.
			if c.done != nil {
				c.done(key)
			}
			if c.finished != nil {
				c.finished(key)
			}
		}
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (c *Checkbox) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return c.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		if c.disabled {
			return false, nil
		}

		x, y := event.Position()
		_, rectY, _, _ := c.GetInnerRect()
		if !c.InRect(x, y) {
			return false, nil
		}

		// Process mouse event.
		if y == rectY {
			if action == MouseLeftDown {
				setFocus(c)
				consumed = true
			} else if action == MouseLeftClick {
				c.checked = !c.checked
				if c.changed != nil {
					c.changed(c.checked)
				}
				consumed = true
			}
		}

		return
	})
}
