package tview

import (
	"github.com/gdamore/tcell/v2"
)

// Button is labeled box that triggers an action when selected.
//
// See https://github.com/rivo/tview/wiki/Button for an example.
type Button struct {
	*Box

	// If set to true, the button cannot be activated.
	disabled bool

	// The text to be displayed inside the button.
	text string

	// The button's style (when deactivated).
	style tcell.Style

	// The button's style (when activated).
	activatedStyle tcell.Style

	// The button's style (when disabled).
	disabledStyle tcell.Style

	// An optional function which is called when the button was selected.
	selected func()

	// An optional function which is called when the user leaves the button. A
	// key is provided indicating which key was pressed to leave (tab or
	// backtab).
	exit func(tcell.Key)
}

// NewButton returns a new input field.
func NewButton(label string) *Button {
	box := NewBox()
	box.SetRect(0, 0, TaggedStringWidth(label)+4, 1)
	return &Button{
		Box:            box,
		text:           label,
		style:          tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.PrimaryTextColor),
		activatedStyle: tcell.StyleDefault.Background(Styles.PrimaryTextColor).Foreground(Styles.InverseTextColor),
		disabledStyle:  tcell.StyleDefault.Background(Styles.ContrastBackgroundColor).Foreground(Styles.ContrastSecondaryTextColor),
	}
}

// SetLabel sets the button text.
func (b *Button) SetLabel(label string) *Button {
	b.text = label
	return b
}

// GetLabel returns the button text.
func (b *Button) GetLabel() string {
	return b.text
}

// SetLabelColor sets the color of the button text.
func (b *Button) SetLabelColor(color tcell.Color) *Button {
	b.style = b.style.Foreground(color)
	return b
}

// SetStyle sets the style of the button used when it is not focused.
func (b *Button) SetStyle(style tcell.Style) *Button {
	b.style = style
	return b
}

// SetLabelColorActivated sets the color of the button text when the button is
// in focus.
func (b *Button) SetLabelColorActivated(color tcell.Color) *Button {
	b.activatedStyle = b.activatedStyle.Foreground(color)
	return b
}

// SetBackgroundColorActivated sets the background color of the button text when
// the button is in focus.
func (b *Button) SetBackgroundColorActivated(color tcell.Color) *Button {
	b.activatedStyle = b.activatedStyle.Background(color)
	return b
}

// SetActivatedStyle sets the style of the button used when it is focused.
func (b *Button) SetActivatedStyle(style tcell.Style) *Button {
	b.activatedStyle = style
	return b
}

// SetDisabledStyle sets the style of the button used when it is disabled.
func (b *Button) SetDisabledStyle(style tcell.Style) *Button {
	b.disabledStyle = style
	return b
}

// SetDisabled sets whether or not the button is disabled. Disabled buttons
// cannot be activated.
//
// If the button is part of a form, you should set focus to the form itself
// after calling this function to set focus to the next non-disabled form item.
func (b *Button) SetDisabled(disabled bool) *Button {
	b.disabled = disabled
	return b
}

// IsDisabled returns whether or not the button is disabled.
func (b *Button) IsDisabled() bool {
	return b.disabled
}

// SetSelectedFunc sets a handler which is called when the button was selected.
func (b *Button) SetSelectedFunc(handler func()) *Button {
	b.selected = handler
	return b
}

// SetExitFunc sets a handler which is called when the user leaves the button.
// The callback function is provided with the key that was pressed, which is one
// of the following:
//
//   - KeyEscape: Leaving the button with no specific direction.
//   - KeyTab: Move to the next field.
//   - KeyBacktab: Move to the previous field.
func (b *Button) SetExitFunc(handler func(key tcell.Key)) *Button {
	b.exit = handler
	return b
}

// Draw draws this primitive onto the screen.
func (b *Button) Draw(screen tcell.Screen) {
	// Draw the box.
	style := b.style
	if b.disabled {
		style = b.disabledStyle
	}
	if b.HasFocus() && !b.disabled {
		style = b.activatedStyle
	}
	_, backgroundColor, _ := style.Decompose()
	b.SetBackgroundColor(backgroundColor)
	b.Box.DrawForSubclass(screen, b)

	// Draw label.
	x, y, width, height := b.GetInnerRect()
	if width > 0 && height > 0 {
		y = y + height/2
		printWithStyle(screen, b.text, x, y, 0, width, AlignCenter, style, true)
	}
}

// InputHandler returns the handler for this primitive.
func (b *Button) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		if b.disabled {
			return
		}

		// Process key event.
		switch key := event.Key(); key {
		case tcell.KeyEnter: // Selected.
			if b.selected != nil {
				b.selected()
			}
		case tcell.KeyBacktab, tcell.KeyTab, tcell.KeyEscape: // Leave. No action.
			if b.exit != nil {
				b.exit(key)
			}
		}
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (b *Button) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return b.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		if b.disabled {
			return false, nil
		}

		if !b.InRect(event.Position()) {
			return false, nil
		}

		// Process mouse event.
		if action == MouseLeftDown {
			setFocus(b)
			consumed = true
		} else if action == MouseLeftClick {
			if b.selected != nil {
				b.selected()
			}
			consumed = true
		}

		return
	})
}
