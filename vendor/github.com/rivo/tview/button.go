package tview

import (
	"github.com/gdamore/tcell/v2"
)

// Button is labeled box that triggers an action when selected.
//
// See https://github.com/rivo/tview/wiki/Button for an example.
type Button struct {
	*Box

	// The text to be displayed before the input area.
	label string

	// The label color.
	labelColor tcell.Color

	// The label color when the button is in focus.
	labelColorActivated tcell.Color

	// The background color when the button is in focus.
	backgroundColorActivated tcell.Color

	// An optional function which is called when the button was selected.
	selected func()

	// An optional function which is called when the user leaves the button. A
	// key is provided indicating which key was pressed to leave (tab or
	// backtab).
	exit func(tcell.Key)
}

// NewButton returns a new input field.
func NewButton(label string) *Button {
	box := NewBox().SetBackgroundColor(Styles.ContrastBackgroundColor)
	box.SetRect(0, 0, TaggedStringWidth(label)+4, 1)
	return &Button{
		Box:                      box,
		label:                    label,
		labelColor:               Styles.PrimaryTextColor,
		labelColorActivated:      Styles.InverseTextColor,
		backgroundColorActivated: Styles.PrimaryTextColor,
	}
}

// SetLabel sets the button text.
func (b *Button) SetLabel(label string) *Button {
	b.label = label
	return b
}

// GetLabel returns the button text.
func (b *Button) GetLabel() string {
	return b.label
}

// SetLabelColor sets the color of the button text.
func (b *Button) SetLabelColor(color tcell.Color) *Button {
	b.labelColor = color
	return b
}

// SetLabelColorActivated sets the color of the button text when the button is
// in focus.
func (b *Button) SetLabelColorActivated(color tcell.Color) *Button {
	b.labelColorActivated = color
	return b
}

// SetBackgroundColorActivated sets the background color of the button text when
// the button is in focus.
func (b *Button) SetBackgroundColorActivated(color tcell.Color) *Button {
	b.backgroundColorActivated = color
	return b
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
	borderColor := b.GetBorderColor()
	backgroundColor := b.GetBackgroundColor()
	if b.HasFocus() {
		b.SetBackgroundColor(b.backgroundColorActivated)
		b.SetBorderColor(b.labelColorActivated)
		defer func() {
			b.SetBorderColor(borderColor)
		}()
	}
	b.Box.DrawForSubclass(screen, b)
	b.backgroundColor = backgroundColor

	// Draw label.
	x, y, width, height := b.GetInnerRect()
	if width > 0 && height > 0 {
		y = y + height/2
		labelColor := b.labelColor
		if b.HasFocus() {
			labelColor = b.labelColorActivated
		}
		Print(screen, b.label, x, y, width, AlignCenter, labelColor)
	}
}

// InputHandler returns the handler for this primitive.
func (b *Button) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
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
		if !b.InRect(event.Position()) {
			return false, nil
		}

		// Process mouse event.
		if action == MouseLeftClick {
			setFocus(b)
			if b.selected != nil {
				b.selected()
			}
			consumed = true
		}

		return
	})
}
