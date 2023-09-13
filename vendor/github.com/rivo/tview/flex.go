package tview

import (
	"github.com/gdamore/tcell/v2"
)

// Configuration values.
const (
	FlexRow       = 0 // One item per row.
	FlexColumn    = 1 // One item per column.
	FlexRowCSS    = 1 // As defined in CSS, items distributed along a row.
	FlexColumnCSS = 0 // As defined in CSS, items distributed within a column.
)

// flexItem holds layout options for one item.
type flexItem struct {
	Item       Primitive // The item to be positioned. May be nil for an empty item.
	FixedSize  int       // The item's fixed size which may not be changed, 0 if it has no fixed size.
	Proportion int       // The item's proportion.
	Focus      bool      // Whether or not this item attracts the layout's focus.
}

// Flex is a basic implementation of the Flexbox layout. The contained
// primitives are arranged horizontally or vertically. The way they are
// distributed along that dimension depends on their layout settings, which is
// either a fixed length or a proportional length. See AddItem() for details.
//
// See https://github.com/rivo/tview/wiki/Flex for an example.
type Flex struct {
	*Box

	// The items to be positioned.
	items []*flexItem

	// FlexRow or FlexColumn.
	direction int

	// If set to true, Flex will use the entire screen as its available space
	// instead its box dimensions.
	fullScreen bool
}

// NewFlex returns a new flexbox layout container with no primitives and its
// direction set to FlexColumn. To add primitives to this layout, see AddItem().
// To change the direction, see SetDirection().
//
// Note that Box, the superclass of Flex, will not clear its contents so that
// any nil flex items will leave their background unchanged. To clear a Flex's
// background before any items are drawn, set it to a box with the desired
// color:
//
//   flex.Box = NewBox()
func NewFlex() *Flex {
	f := &Flex{
		direction: FlexColumn,
	}
	f.Box = NewBox()
	f.Box.dontClear = true
	return f
}

// SetDirection sets the direction in which the contained primitives are
// distributed. This can be either FlexColumn (default) or FlexRow. Note that
// these are the opposite of what you would expect coming from CSS. You may also
// use FlexColumnCSS or FlexRowCSS, to remain in line with the CSS definition.
func (f *Flex) SetDirection(direction int) *Flex {
	f.direction = direction
	return f
}

// SetFullScreen sets the flag which, when true, causes the flex layout to use
// the entire screen space instead of whatever size it is currently assigned to.
func (f *Flex) SetFullScreen(fullScreen bool) *Flex {
	f.fullScreen = fullScreen
	return f
}

// AddItem adds a new item to the container. The "fixedSize" argument is a width
// or height that may not be changed by the layout algorithm. A value of 0 means
// that its size is flexible and may be changed. The "proportion" argument
// defines the relative size of the item compared to other flexible-size items.
// For example, items with a proportion of 2 will be twice as large as items
// with a proportion of 1. The proportion must be at least 1 if fixedSize == 0
// (ignored otherwise).
//
// If "focus" is set to true, the item will receive focus when the Flex
// primitive receives focus. If multiple items have the "focus" flag set to
// true, the first one will receive focus.
//
// You can provide a nil value for the primitive. This will still consume screen
// space but nothing will be drawn.
func (f *Flex) AddItem(item Primitive, fixedSize, proportion int, focus bool) *Flex {
	f.items = append(f.items, &flexItem{Item: item, FixedSize: fixedSize, Proportion: proportion, Focus: focus})
	return f
}

// RemoveItem removes all items for the given primitive from the container,
// keeping the order of the remaining items intact.
func (f *Flex) RemoveItem(p Primitive) *Flex {
	for index := len(f.items) - 1; index >= 0; index-- {
		if f.items[index].Item == p {
			f.items = append(f.items[:index], f.items[index+1:]...)
		}
	}
	return f
}

// GetItemCount returns the number of items in this container.
func (f *Flex) GetItemCount() int {
	return len(f.items)
}

// GetItem returns the primitive at the given index, starting with 0 for the
// first primitive in this container.
//
// This function will panic for out of range indices.
func (f *Flex) GetItem(index int) Primitive {
	return f.items[index].Item
}

// Clear removes all items from the container.
func (f *Flex) Clear() *Flex {
	f.items = nil
	return f
}

// ResizeItem sets a new size for the item(s) with the given primitive. If there
// are multiple Flex items with the same primitive, they will all receive the
// same size. For details regarding the size parameters, see AddItem().
func (f *Flex) ResizeItem(p Primitive, fixedSize, proportion int) *Flex {
	for _, item := range f.items {
		if item.Item == p {
			item.FixedSize = fixedSize
			item.Proportion = proportion
		}
	}
	return f
}

// Draw draws this primitive onto the screen.
func (f *Flex) Draw(screen tcell.Screen) {
	f.Box.DrawForSubclass(screen, f)

	// Calculate size and position of the items.

	// Do we use the entire screen?
	if f.fullScreen {
		width, height := screen.Size()
		f.SetRect(0, 0, width, height)
	}

	// How much space can we distribute?
	x, y, width, height := f.GetInnerRect()
	var proportionSum int
	distSize := width
	if f.direction == FlexRow {
		distSize = height
	}
	for _, item := range f.items {
		if item.FixedSize > 0 {
			distSize -= item.FixedSize
		} else {
			proportionSum += item.Proportion
		}
	}

	// Calculate positions and draw items.
	pos := x
	if f.direction == FlexRow {
		pos = y
	}
	for _, item := range f.items {
		size := item.FixedSize
		if size <= 0 {
			if proportionSum > 0 {
				size = distSize * item.Proportion / proportionSum
				distSize -= size
				proportionSum -= item.Proportion
			} else {
				size = 0
			}
		}
		if item.Item != nil {
			if f.direction == FlexColumn {
				item.Item.SetRect(pos, y, size, height)
			} else {
				item.Item.SetRect(x, pos, width, size)
			}
		}
		pos += size

		if item.Item != nil {
			if item.Item.HasFocus() {
				defer item.Item.Draw(screen)
			} else {
				item.Item.Draw(screen)
			}
		}
	}
}

// Focus is called when this primitive receives focus.
func (f *Flex) Focus(delegate func(p Primitive)) {
	for _, item := range f.items {
		if item.Item != nil && item.Focus {
			delegate(item.Item)
			return
		}
	}
	f.Box.Focus(delegate)
}

// HasFocus returns whether or not this primitive has focus.
func (f *Flex) HasFocus() bool {
	for _, item := range f.items {
		if item.Item != nil && item.Item.HasFocus() {
			return true
		}
	}
	return f.Box.HasFocus()
}

// MouseHandler returns the mouse handler for this primitive.
func (f *Flex) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return f.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		if !f.InRect(event.Position()) {
			return false, nil
		}

		// Pass mouse events along to the first child item that takes it.
		for _, item := range f.items {
			if item.Item == nil {
				continue
			}
			consumed, capture = item.Item.MouseHandler()(action, event, setFocus)
			if consumed {
				return
			}
		}

		return
	})
}

// InputHandler returns the handler for this primitive.
func (f *Flex) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return f.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		for _, item := range f.items {
			if item.Item != nil && item.Item.HasFocus() {
				if handler := item.Item.InputHandler(); handler != nil {
					handler(event, setFocus)
					return
				}
			}
		}
	})
}
