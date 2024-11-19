package tview

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// listItem represents one item in a List.
type listItem struct {
	MainText      string // The main text of the list item.
	SecondaryText string // A secondary text to be shown underneath the main text.
	Shortcut      rune   // The key to select the list item directly, 0 if there is no shortcut.
	Selected      func() // The optional function which is called when the item is selected.
}

// List displays rows of items, each of which can be selected. List items can be
// shown as a single line or as two lines. They can be selected by pressing
// their assigned shortcut key, navigating to them and pressing Enter, or
// clicking on them with the mouse. The following key binds are available:
//
//   - Down arrow / tab: Move down one item.
//   - Up arrow / backtab: Move up one item.
//   - Home: Move to the first item.
//   - End: Move to the last item.
//   - Page down: Move down one page.
//   - Page up: Move up one page.
//   - Enter / Space: Select the current item.
//   - Right / left: Scroll horizontally. Only if the list is wider than the
//     available space.
//
// By default, list item texts can contain style tags. Use
// [List.SetUseStyleTags] to disable this feature.
//
// See [List.SetChangedFunc] for a way to be notified when the user navigates
// to a list item. See [List.SetSelectedFunc] for a way to be notified when a
// list item was selected.
//
// See https://github.com/rivo/tview/wiki/List for an example.
type List struct {
	*Box

	// The items of the list.
	items []*listItem

	// The index of the currently selected item.
	currentItem int

	// Whether or not to show the secondary item texts.
	showSecondaryText bool

	// The item main text style.
	mainTextStyle tcell.Style

	// The item secondary text style.
	secondaryTextStyle tcell.Style

	// The item shortcut text style.
	shortcutStyle tcell.Style

	// The style for selected items.
	selectedStyle tcell.Style

	// If true, the selection is only shown when the list has focus.
	selectedFocusOnly bool

	// If true, the entire row is highlighted when selected.
	highlightFullLine bool

	// Whether or not style tags can be used in the main text.
	mainStyleTags bool

	// Whether or not style tags can be used in the secondary text.
	secondaryStyleTags bool

	// Whether or not navigating the list will wrap around.
	wrapAround bool

	// The number of list items skipped at the top before the first item is
	// drawn.
	itemOffset int

	// The number of cells skipped on the left side of an item text. Shortcuts
	// are not affected.
	horizontalOffset int

	// An optional function which is called when the user has navigated to a
	// list item.
	changed func(index int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when a list item was selected. This
	// function will be called even if the list item defines its own callback.
	selected func(index int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when the user presses the Escape key.
	done func()
}

// NewList returns a new list.
func NewList() *List {
	return &List{
		Box:                NewBox(),
		showSecondaryText:  true,
		wrapAround:         true,
		mainTextStyle:      tcell.StyleDefault.Foreground(Styles.PrimaryTextColor).Background(Styles.PrimitiveBackgroundColor),
		secondaryTextStyle: tcell.StyleDefault.Foreground(Styles.TertiaryTextColor).Background(Styles.PrimitiveBackgroundColor),
		shortcutStyle:      tcell.StyleDefault.Foreground(Styles.SecondaryTextColor).Background(Styles.PrimitiveBackgroundColor),
		selectedStyle:      tcell.StyleDefault.Foreground(Styles.PrimitiveBackgroundColor).Background(Styles.PrimaryTextColor),
		mainStyleTags:      true,
		secondaryStyleTags: true,
	}
}

// SetCurrentItem sets the currently selected item by its index, starting at 0
// for the first item. If a negative index is provided, items are referred to
// from the back (-1 = last item, -2 = second-to-last item, and so on). Out of
// range indices are clamped to the beginning/end.
//
// Calling this function triggers a "changed" event if the selection changes.
func (l *List) SetCurrentItem(index int) *List {
	if index < 0 {
		index = len(l.items) + index
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	if index < 0 {
		index = 0
	}

	if index != l.currentItem && l.changed != nil {
		item := l.items[index]
		l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
	}

	l.currentItem = index

	l.adjustOffset()

	return l
}

// GetCurrentItem returns the index of the currently selected list item,
// starting at 0 for the first item.
func (l *List) GetCurrentItem() int {
	return l.currentItem
}

// SetOffset sets the number of items to be skipped (vertically) as well as the
// number of cells skipped horizontally when the list is drawn. Note that one
// item corresponds to two rows when there are secondary texts. Shortcuts are
// always drawn.
//
// These values may change when the list is drawn to ensure the currently
// selected item is visible and item texts move out of view. Users can also
// modify these values by interacting with the list.
func (l *List) SetOffset(items, horizontal int) *List {
	l.itemOffset = items
	l.horizontalOffset = horizontal
	return l
}

// GetOffset returns the number of items skipped while drawing, as well as the
// number of cells item text is moved to the left. See also SetOffset() for more
// information on these values.
func (l *List) GetOffset() (int, int) {
	return l.itemOffset, l.horizontalOffset
}

// RemoveItem removes the item with the given index (starting at 0) from the
// list. If a negative index is provided, items are referred to from the back
// (-1 = last item, -2 = second-to-last item, and so on). Out of range indices
// are clamped to the beginning/end, i.e. unless the list is empty, an item is
// always removed.
//
// The currently selected item is shifted accordingly. If it is the one that is
// removed, a "changed" event is fired, unless no items are left.
func (l *List) RemoveItem(index int) *List {
	if len(l.items) == 0 {
		return l
	}

	// Adjust index.
	if index < 0 {
		index = len(l.items) + index
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	if index < 0 {
		index = 0
	}

	// Remove item.
	l.items = append(l.items[:index], l.items[index+1:]...)

	// If there is nothing left, we're done.
	if len(l.items) == 0 {
		return l
	}

	// Shift current item.
	previousCurrentItem := l.currentItem
	if l.currentItem > index || l.currentItem == len(l.items) {
		l.currentItem--
	}

	// Fire "changed" event for removed items.
	if previousCurrentItem == index && l.changed != nil {
		item := l.items[l.currentItem]
		l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
	}

	return l
}

// SetMainTextColor sets the color of the items' main text.
func (l *List) SetMainTextColor(color tcell.Color) *List {
	l.mainTextStyle = l.mainTextStyle.Foreground(color)
	return l
}

// SetMainTextStyle sets the style of the items' main text. Note that the
// background color is ignored in order not to override the background color of
// the list itself.
func (l *List) SetMainTextStyle(style tcell.Style) *List {
	l.mainTextStyle = style
	return l
}

// SetSecondaryTextColor sets the color of the items' secondary text.
func (l *List) SetSecondaryTextColor(color tcell.Color) *List {
	l.secondaryTextStyle = l.secondaryTextStyle.Foreground(color)
	return l
}

// SetSecondaryTextStyle sets the style of the items' secondary text. Note that
// the background color is ignored in order not to override the background color
// of the list itself.
func (l *List) SetSecondaryTextStyle(style tcell.Style) *List {
	l.secondaryTextStyle = style
	return l
}

// SetShortcutColor sets the color of the items' shortcut.
func (l *List) SetShortcutColor(color tcell.Color) *List {
	l.shortcutStyle = l.shortcutStyle.Foreground(color)
	return l
}

// SetShortcutStyle sets the style of the items' shortcut. Note that the
// background color is ignored in order not to override the background color of
// the list itself.
func (l *List) SetShortcutStyle(style tcell.Style) *List {
	l.shortcutStyle = style
	return l
}

// SetSelectedTextColor sets the text color of selected items. Note that the
// color of main text characters that are different from the main text color
// (e.g. style tags) is maintained.
func (l *List) SetSelectedTextColor(color tcell.Color) *List {
	l.selectedStyle = l.selectedStyle.Foreground(color)
	return l
}

// SetSelectedBackgroundColor sets the background color of selected items.
func (l *List) SetSelectedBackgroundColor(color tcell.Color) *List {
	l.selectedStyle = l.selectedStyle.Background(color)
	return l
}

// SetSelectedStyle sets the style of the selected items. Note that the color of
// main text characters that are different from the main text color (e.g. color
// tags) is maintained.
func (l *List) SetSelectedStyle(style tcell.Style) *List {
	l.selectedStyle = style
	return l
}

// SetUseStyleTags sets a flag which determines whether style tags are used in
// the main and secondary texts. The default is true.
func (l *List) SetUseStyleTags(mainStyleTags, secondaryStyleTags bool) *List {
	l.mainStyleTags = mainStyleTags
	l.secondaryStyleTags = secondaryStyleTags
	return l
}

// SetSelectedFocusOnly sets a flag which determines when the currently selected
// list item is highlighted. If set to true, selected items are only highlighted
// when the list has focus. If set to false, they are always highlighted.
func (l *List) SetSelectedFocusOnly(focusOnly bool) *List {
	l.selectedFocusOnly = focusOnly
	return l
}

// SetHighlightFullLine sets a flag which determines whether the colored
// background of selected items spans the entire width of the view. If set to
// true, the highlight spans the entire view. If set to false, only the text of
// the selected item from beginning to end is highlighted.
func (l *List) SetHighlightFullLine(highlight bool) *List {
	l.highlightFullLine = highlight
	return l
}

// ShowSecondaryText determines whether or not to show secondary item texts.
func (l *List) ShowSecondaryText(show bool) *List {
	l.showSecondaryText = show
	return l
}

// SetWrapAround sets the flag that determines whether navigating the list will
// wrap around. That is, navigating downwards on the last item will move the
// selection to the first item (similarly in the other direction). If set to
// false, the selection won't change when navigating downwards on the last item
// or navigating upwards on the first item.
func (l *List) SetWrapAround(wrapAround bool) *List {
	l.wrapAround = wrapAround
	return l
}

// SetChangedFunc sets the function which is called when the user navigates to
// a list item. The function receives the item's index in the list of items
// (starting with 0), its main text, secondary text, and its shortcut rune.
//
// This function is also called when the first item is added or when
// SetCurrentItem() is called.
func (l *List) SetChangedFunc(handler func(index int, mainText string, secondaryText string, shortcut rune)) *List {
	l.changed = handler
	return l
}

// SetSelectedFunc sets the function which is called when the user selects a
// list item by pressing Enter on the current selection. The function receives
// the item's index in the list of items (starting with 0), its main text,
// secondary text, and its shortcut rune.
func (l *List) SetSelectedFunc(handler func(int, string, string, rune)) *List {
	l.selected = handler
	return l
}

// GetSelectedFunc returns the function set with [List.SetSelectedFunc] or nil
// if no such function was set.
func (l *List) GetSelectedFunc() func(int, string, string, rune) {
	return l.selected
}

// SetDoneFunc sets a function which is called when the user presses the Escape
// key.
func (l *List) SetDoneFunc(handler func()) *List {
	l.done = handler
	return l
}

// AddItem calls [List.InsertItem] with an index of -1.
func (l *List) AddItem(mainText, secondaryText string, shortcut rune, selected func()) *List {
	l.InsertItem(-1, mainText, secondaryText, shortcut, selected)
	return l
}

// InsertItem adds a new item to the list at the specified index. An index of 0
// will insert the item at the beginning, an index of 1 before the second item,
// and so on. An index of [List.GetItemCount] or higher will insert the item at
// the end of the list. Negative indices are also allowed: An index of -1 will
// insert the item at the end of the list, an index of -2 before the last item,
// and so on. An index of -GetItemCount()-1 or lower will insert the item at the
// beginning.
//
// An item has a main text which will be highlighted when selected. It also has
// a secondary text which is shown underneath the main text (if it is set to
// visible) but which may remain empty.
//
// The shortcut is a key binding. If the specified rune is entered, the item
// is selected immediately. Set to 0 for no binding.
//
// The "selected" callback will be invoked when the user selects the item. You
// may provide nil if no such callback is needed or if all events are handled
// through the selected callback set with [List.SetSelectedFunc].
//
// The currently selected item will shift its position accordingly. If the list
// was previously empty, a "changed" event is fired because the new item becomes
// selected.
func (l *List) InsertItem(index int, mainText, secondaryText string, shortcut rune, selected func()) *List {
	item := &listItem{
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		Selected:      selected,
	}

	// Shift index to range.
	if index < 0 {
		index = len(l.items) + index + 1
	}
	if index < 0 {
		index = 0
	} else if index > len(l.items) {
		index = len(l.items)
	}

	// Shift current item.
	if l.currentItem < len(l.items) && l.currentItem >= index {
		l.currentItem++
	}

	// Insert item (make space for the new item, then shift and insert).
	l.items = append(l.items, nil)
	if index < len(l.items)-1 { // -1 because l.items has already grown by one item.
		copy(l.items[index+1:], l.items[index:])
	}
	l.items[index] = item

	// Fire a "change" event for the first item in the list.
	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.changed(0, item.MainText, item.SecondaryText, item.Shortcut)
	}

	return l
}

// GetItemCount returns the number of items in the list.
func (l *List) GetItemCount() int {
	return len(l.items)
}

// GetItemSelectedFunc returns the function which is called when the user
// selects the item with the given index, if such a function was set. If no
// function was set, nil is returned. Panics if the index is out of range.
func (l *List) GetItemSelectedFunc(index int) func() {
	return l.items[index].Selected
}

// GetItemText returns an item's texts (main and secondary). Panics if the index
// is out of range.
func (l *List) GetItemText(index int) (main, secondary string) {
	return l.items[index].MainText, l.items[index].SecondaryText
}

// SetItemText sets an item's main and secondary text. Panics if the index is
// out of range.
func (l *List) SetItemText(index int, main, secondary string) *List {
	item := l.items[index]
	item.MainText = main
	item.SecondaryText = secondary
	return l
}

// FindItems searches the main and secondary texts for the given strings and
// returns a list of item indices in which those strings are found. One of the
// two search strings may be empty, it will then be ignored. Indices are always
// returned in ascending order.
//
// If mustContainBoth is set to true, mainSearch must be contained in the main
// text AND secondarySearch must be contained in the secondary text. If it is
// false, only one of the two search strings must be contained.
//
// Set ignoreCase to true for case-insensitive search.
func (l *List) FindItems(mainSearch, secondarySearch string, mustContainBoth, ignoreCase bool) (indices []int) {
	if mainSearch == "" && secondarySearch == "" {
		return
	}

	if ignoreCase {
		mainSearch = strings.ToLower(mainSearch)
		secondarySearch = strings.ToLower(secondarySearch)
	}

	for index, item := range l.items {
		mainText := item.MainText
		secondaryText := item.SecondaryText
		if ignoreCase {
			mainText = strings.ToLower(mainText)
			secondaryText = strings.ToLower(secondaryText)
		}

		// strings.Contains() always returns true for a "" search.
		mainContained := strings.Contains(mainText, mainSearch)
		secondaryContained := strings.Contains(secondaryText, secondarySearch)
		if mustContainBoth && mainContained && secondaryContained ||
			!mustContainBoth && (mainSearch != "" && mainContained || secondarySearch != "" && secondaryContained) {
			indices = append(indices, index)
		}
	}

	return
}

// Clear removes all items from the list.
func (l *List) Clear() *List {
	l.items = nil
	l.currentItem = 0
	return l
}

// Draw draws this primitive onto the screen.
func (l *List) Draw(screen tcell.Screen) {
	l.Box.DrawForSubclass(screen, l)

	// Determine the dimensions.
	x, y, width, height := l.GetInnerRect()
	bottomLimit := y + height
	_, totalHeight := screen.Size()
	if bottomLimit > totalHeight {
		bottomLimit = totalHeight
	}

	// Do we show any shortcuts?
	var showShortcuts bool
	for _, item := range l.items {
		if item.Shortcut != 0 {
			showShortcuts = true
			x += 4
			width -= 4
			break
		}
	}

	if l.horizontalOffset < 0 {
		l.horizontalOffset = 0
	}

	// Draw the list items.
	var maxWidth int // The maximum printed item width.
	for index, item := range l.items {
		if index < l.itemOffset {
			continue
		}

		if y >= bottomLimit {
			break
		}

		// Shortcuts.
		if showShortcuts && item.Shortcut != 0 {
			printWithStyle(screen, fmt.Sprintf("(%s)", string(item.Shortcut)), x-5, y, 0, 4, AlignRight, l.shortcutStyle, false)
		}

		// Main text.
		selected := index == l.currentItem && (!l.selectedFocusOnly || l.HasFocus())
		style := l.mainTextStyle
		if selected {
			style = l.selectedStyle
		}
		mainText := item.MainText
		if !l.mainStyleTags {
			mainText = Escape(mainText)
		}
		_, _, printedWidth := printWithStyle(screen, mainText, x, y, l.horizontalOffset, width, AlignLeft, style, false)
		if printedWidth > maxWidth {
			maxWidth = printedWidth
		}

		// Draw until the end of the line if requested.
		if selected && l.highlightFullLine {
			for bx := printedWidth; bx < width; bx++ {
				screen.SetContent(x+bx, y, ' ', nil, style)
			}
		}

		y++
		if y >= bottomLimit {
			break
		}

		// Secondary text.
		if l.showSecondaryText {
			secondaryText := item.SecondaryText
			if !l.secondaryStyleTags {
				secondaryText = Escape(secondaryText)
			}
			_, _, printedWidth := printWithStyle(screen, secondaryText, x, y, l.horizontalOffset, width, AlignLeft, l.secondaryTextStyle, false)
			if printedWidth > maxWidth {
				maxWidth = printedWidth
			}
			y++
		}
	}

	// We don't want the item text to get out of view. If the horizontal offset
	// is too high, we reset it and redraw. (That should be about as efficient
	// as calculating everything up front.)
	if l.horizontalOffset > 0 && maxWidth < width {
		l.horizontalOffset -= width - maxWidth
		l.Draw(screen)
	}
}

// adjustOffset adjusts the vertical offset to keep the current selection in
// view.
func (l *List) adjustOffset() {
	_, _, _, height := l.GetInnerRect()
	if height == 0 {
		return
	}
	if l.currentItem < l.itemOffset {
		l.itemOffset = l.currentItem
	} else if l.showSecondaryText {
		if 2*(l.currentItem-l.itemOffset) >= height-1 {
			l.itemOffset = (2*l.currentItem + 3 - height) / 2
		}
	} else {
		if l.currentItem-l.itemOffset >= height {
			l.itemOffset = l.currentItem + 1 - height
		}
	}
}

// InputHandler returns the handler for this primitive.
func (l *List) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return l.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		if event.Key() == tcell.KeyEscape {
			if l.done != nil {
				l.done()
			}
			return
		} else if len(l.items) == 0 {
			return
		}

		previousItem := l.currentItem

		switch key := event.Key(); key {
		case tcell.KeyTab, tcell.KeyDown:
			l.currentItem++
		case tcell.KeyBacktab, tcell.KeyUp:
			l.currentItem--
		case tcell.KeyRight:
			l.horizontalOffset += 2 // We shift by 2 to account for two-cell characters.
		case tcell.KeyLeft:
			l.horizontalOffset -= 2
		case tcell.KeyHome:
			l.currentItem = 0
		case tcell.KeyEnd:
			l.currentItem = len(l.items) - 1
		case tcell.KeyPgDn:
			_, _, _, height := l.GetInnerRect()
			l.currentItem += height
			if l.currentItem >= len(l.items) {
				l.currentItem = len(l.items) - 1
			}
		case tcell.KeyPgUp:
			_, _, _, height := l.GetInnerRect()
			l.currentItem -= height
			if l.currentItem < 0 {
				l.currentItem = 0
			}
		case tcell.KeyEnter:
			if l.currentItem >= 0 && l.currentItem < len(l.items) {
				item := l.items[l.currentItem]
				if item.Selected != nil {
					item.Selected()
				}
				if l.selected != nil {
					l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
				}
			}
		case tcell.KeyRune:
			ch := event.Rune()
			if ch != ' ' {
				// It's not a space bar. Is it a shortcut?
				var found bool
				for index, item := range l.items {
					if item.Shortcut == ch {
						// We have a shortcut.
						found = true
						l.currentItem = index
						break
					}
				}
				if !found {
					break
				}
			}
			item := l.items[l.currentItem]
			if item.Selected != nil {
				item.Selected()
			}
			if l.selected != nil {
				l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
			}
		}

		if l.currentItem < 0 {
			if l.wrapAround {
				l.currentItem = len(l.items) - 1
			} else {
				l.currentItem = 0
			}
		} else if l.currentItem >= len(l.items) {
			if l.wrapAround {
				l.currentItem = 0
			} else {
				l.currentItem = len(l.items) - 1
			}
		}

		if l.currentItem != previousItem && l.currentItem < len(l.items) {
			if l.changed != nil {
				item := l.items[l.currentItem]
				l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
			}
			l.adjustOffset()
		}
	})
}

// indexAtPoint returns the index of the list item found at the given position
// or a negative value if there is no such list item.
func (l *List) indexAtPoint(x, y int) int {
	rectX, rectY, width, height := l.GetInnerRect()
	if rectX < 0 || rectX >= rectX+width || y < rectY || y >= rectY+height {
		return -1
	}

	index := y - rectY
	if l.showSecondaryText {
		index /= 2
	}
	index += l.itemOffset

	if index >= len(l.items) {
		return -1
	}
	return index
}

// MouseHandler returns the mouse handler for this primitive.
func (l *List) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return l.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		if !l.InRect(event.Position()) {
			return false, nil
		}

		// Process mouse event.
		switch action {
		case MouseLeftClick:
			setFocus(l)
			index := l.indexAtPoint(event.Position())
			if index != -1 {
				item := l.items[index]
				if item.Selected != nil {
					item.Selected()
				}
				if l.selected != nil {
					l.selected(index, item.MainText, item.SecondaryText, item.Shortcut)
				}
				if index != l.currentItem {
					if l.changed != nil {
						l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
					}
					l.adjustOffset()
				}
				l.currentItem = index
			}
			consumed = true
		case MouseScrollUp:
			if l.itemOffset > 0 {
				l.itemOffset--
			}
			consumed = true
		case MouseScrollDown:
			lines := len(l.items) - l.itemOffset
			if l.showSecondaryText {
				lines *= 2
			}
			if _, _, _, height := l.GetInnerRect(); lines > height {
				l.itemOffset++
			}
			consumed = true
		case MouseScrollLeft:
			l.horizontalOffset--
			consumed = true
		case MouseScrollRight:
			l.horizontalOffset++
			consumed = true
		}

		return
	})
}
