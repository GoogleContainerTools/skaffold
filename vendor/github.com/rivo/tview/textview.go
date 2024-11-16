package tview

import (
	"math"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// TabSize is the number of spaces with which a tab character will be replaced.
var TabSize = 4

// textViewLine contains information about a line displayed in the text view.
type textViewLine struct {
	offset  int               // The string position in the buffer where this line starts.
	width   int               // The screen width of this line.
	length  int               // The string length (in bytes) of this line.
	state   *stepState        // The parser state at the beginning of the line, before parsing the first character.
	regions map[string][2]int // The start and end columns of all regions in this line. Only valid for visible lines. May be nil.
}

// TextViewWriter is a writer that can be used to write to and clear a TextView
// in batches, i.e. multiple writes with the lock only being acquired once. Don't
// instantiated this class directly but use the TextView's BatchWriter method
// instead.
type TextViewWriter struct {
	t *TextView
}

// Close implements io.Closer for the writer by unlocking the original TextView.
func (w TextViewWriter) Close() error {
	w.t.Unlock()
	return nil
}

// Clear removes all text from the buffer.
func (w TextViewWriter) Clear() {
	w.t.clear()
}

// Write implements the io.Writer interface. It behaves like the TextView's
// Write() method except that it does not acquire the lock.
func (w TextViewWriter) Write(p []byte) (n int, err error) {
	return w.t.write(p)
}

// HasFocus returns whether the underlying TextView has focus.
func (w TextViewWriter) HasFocus() bool {
	return w.t.hasFocus
}

// TextView is a component to display read-only text. While the text to be
// displayed can be changed or appended to, there is no functionality that
// allows the user to edit it. For that, [TextArea] should be used.
//
// TextView implements the io.Writer interface so you can stream text to it,
// appending to the existing text. This does not trigger a redraw automatically
// but if a handler is installed via [TextView.SetChangedFunc], you can cause it
// to be redrawn. (See [TextView.SetChangedFunc] for more details.)
//
// Tab characters advance the text to the next tab stop at every [TabSize]
// screen columns, but only if the text is left-aligned. If the text is centered
// or right-aligned, tab characters are simply replaced with [TabSize] spaces.
//
// Word wrapping is enabled by default. Use [TextView.SetWrap] and
// [TextView.SetWordWrap] to change this.
//
// # Navigation
//
// If the text view is set to be scrollable (which is the default), text is kept
// in a buffer which may be larger than the screen and can be navigated
// with Vim-like key binds:
//
//   - h, left arrow: Move left.
//   - l, right arrow: Move right.
//   - j, down arrow: Move down.
//   - k, up arrow: Move up.
//   - g, home: Move to the top.
//   - G, end: Move to the bottom.
//   - Ctrl-F, page down: Move down by one page.
//   - Ctrl-B, page up: Move up by one page.
//
// If the text is not scrollable, any text above the top visible line is
// discarded. This can be useful when you want to continuously stream text to
// the text view and only keep the latest lines.
//
// Use [Box.SetInputCapture] to override or modify keyboard input.
//
// # Styles / Colors
//
// If dynamic colors are enabled via [TextView.SetDynamicColors], text style can
// be changed dynamically by embedding color strings in square brackets. This
// works the same way as anywhere else. See the package documentation for more
// information.
//
// # Regions and Highlights
//
// If regions are enabled via [TextView.SetRegions], you can define text regions
// within the text and assign region IDs to them. Text regions start with region
// tags. Region tags are square brackets that contain a region ID in double
// quotes, for example:
//
//	We define a ["rg"]region[""] here.
//
// A text region ends with the next region tag. Tags with no region ID ([""])
// don't start new regions. They can therefore be used to mark the end of a
// region. Region IDs must satisfy the following regular expression:
//
//	[a-zA-Z0-9_,;: \-\.]+
//
// Regions can be highlighted by calling the [TextView.Highlight] function with
// one or more region IDs. This can be used to display search results, for
// example.
//
// The [TextView.ScrollToHighlight] function can be used to jump to the
// currently highlighted region once when the text view is drawn the next time.
//
// # Large Texts
//
// The text view can handle reasonably large texts. It will parse the text as
// needed. For optimal performance, it is best to access or display parts of the
// text very far down only if really needed. For example, call
// [TextView.ScrollToBeginning] before adding the text to the text view, to
// avoid scrolling the text all the way to the bottom, forcing a full-text
// parse.
//
// For even larger texts or "infinite" streams of text such as log files, you
// should consider using [TextView.SetMaxLines] to limit the number of lines in
// the text view buffer. Or disable the text view's scrollability altogether
// (using [TextView.SetScrollable]). This will cause the text view to discard
// lines moving out of the visible area at the top.
//
// See https://github.com/rivo/tview/wiki/TextView for an example.
type TextView struct {
	sync.Mutex
	*Box

	// The size of the text area. If set to 0, the text view will use the entire
	// available space.
	width, height int

	// The text buffer.
	text strings.Builder

	// The line index. It is valid at any time but may not contain trailing
	// lines which are not visible.
	lineIndex []*textViewLine

	// The screen width of the longest line in the index.
	longestLine int

	// Regions mapped by their ID to the line where they start. Regions which
	// cannot be found in [TextView.lineIndex] are not contained.
	regions map[string]int

	// The label text shown, usually when part of a form.
	label string

	// The width of the text area's label.
	labelWidth int

	// The label style.
	labelStyle tcell.Style

	// The text alignment, one of AlignLeft, AlignCenter, or AlignRight.
	align int

	// Currently highlighted regions.
	highlights map[string]struct{}

	// The last width for which the current text view was drawn.
	lastWidth int

	// The height of the content the last time the text view was drawn.
	pageSize int

	// The index of the first line shown in the text view.
	lineOffset int

	// If set to true, the text view will always remain at the end of the
	// content when text is added.
	trackEnd bool

	// The width of the characters to be skipped on each line (not used in wrap
	// mode).
	columnOffset int

	// The maximum number of lines kept in the line index, effectively the
	// latest word-wrapped lines. Ignored if 0.
	maxLines int

	// If set to true, the text view will keep a buffer of text which can be
	// navigated when the text is longer than what fits into the box.
	scrollable bool

	// If set to true, lines that are longer than the available width are
	// wrapped onto the next line. If set to false, any characters beyond the
	// available width are discarded.
	wrap bool

	// If set to true and if wrap is also true, Unicode line breaking is
	// applied.
	wordWrap bool

	// The (starting) style of the text. This also defines the background color
	// of the main text element.
	textStyle tcell.Style

	// Whether or not style tags are used.
	styleTags bool

	// Whether or not region tags are used.
	regionTags bool

	// A temporary flag which, when true, will automatically bring the current
	// highlight(s) into the visible screen the next time the text view is
	// drawn.
	scrollToHighlights bool

	// If true, setting new highlights will be a XOR instead of an overwrite
	// operation.
	toggleHighlights bool

	// An optional function which is called when the content of the text view
	// has changed.
	changed func()

	// An optional function which is called when the user presses one of the
	// following keys: Escape, Enter, Tab, Backtab.
	done func(tcell.Key)

	// An optional function which is called when one or more regions were
	// highlighted.
	highlighted func(added, removed, remaining []string)

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)
}

// NewTextView returns a new text view.
func NewTextView() *TextView {
	return &TextView{
		Box:        NewBox(),
		labelStyle: tcell.StyleDefault.Foreground(Styles.SecondaryTextColor),
		highlights: make(map[string]struct{}),
		lineOffset: -1,
		scrollable: true,
		align:      AlignLeft,
		wrap:       true,
		wordWrap:   true,
		textStyle:  tcell.StyleDefault.Background(Styles.PrimitiveBackgroundColor).Foreground(Styles.PrimaryTextColor),
		regionTags: false,
		styleTags:  false,
	}
}

// SetLabel sets the text to be displayed before the text view.
func (t *TextView) SetLabel(label string) *TextView {
	t.label = label
	return t
}

// GetLabel returns the text to be displayed before the text view.
func (t *TextView) GetLabel() string {
	return t.label
}

// SetLabelWidth sets the screen width of the label. A value of 0 will cause the
// primitive to use the width of the label string.
func (t *TextView) SetLabelWidth(width int) *TextView {
	t.labelWidth = width
	return t
}

// SetSize sets the screen size of the main text element of the text view. This
// element is always located next to the label which is always located in the
// top left corner. If any of the values are 0 or larger than the available
// space, the available space will be used.
func (t *TextView) SetSize(rows, columns int) *TextView {
	t.width = columns
	t.height = rows
	return t
}

// GetFieldWidth returns this primitive's field width.
func (t *TextView) GetFieldWidth() int {
	return t.width
}

// GetFieldHeight returns this primitive's field height.
func (t *TextView) GetFieldHeight() int {
	return t.height
}

// SetDisabled sets whether or not the item is disabled / read-only.
func (t *TextView) SetDisabled(disabled bool) FormItem {
	return t // Text views are always read-only.
}

// SetScrollable sets the flag that decides whether or not the text view is
// scrollable. If false, text that moves above the text view's top row will be
// permanently deleted.
func (t *TextView) SetScrollable(scrollable bool) *TextView {
	t.scrollable = scrollable
	if !scrollable {
		t.trackEnd = true
	}
	return t
}

// SetWrap sets the flag that, if true, leads to lines that are longer than the
// available width being wrapped onto the next line. If false, any characters
// beyond the available width are not displayed.
func (t *TextView) SetWrap(wrap bool) *TextView {
	if t.wrap != wrap {
		t.resetIndex() // This invalidates the entire index.
	}
	t.wrap = wrap
	return t
}

// SetWordWrap sets the flag that, if true and if the "wrap" flag is also true
// (see [TextView.SetWrap]), wraps according to [Unicode Standard Annex #14].
//
// This flag is ignored if the "wrap" flag is false.
func (t *TextView) SetWordWrap(wrapOnWords bool) *TextView {
	if t.wrap && t.wordWrap != wrapOnWords {
		t.resetIndex() // This invalidates the entire index.
	}
	t.wordWrap = wrapOnWords
	return t
}

// SetMaxLines sets the maximum number of lines for this text view. Lines at the
// beginning of the text will be discarded when the text view is drawn, so as to
// remain below this value. Only lines above the first visible line are removed.
//
// Broken-over lines via word/character wrapping are counted individually.
//
// Note that [TextView.GetText] will return the shortened text.
//
// A value of 0 (the default) will keep all lines in place.
func (t *TextView) SetMaxLines(maxLines int) *TextView {
	t.maxLines = maxLines
	return t
}

// SetTextAlign sets the text alignment within the text view. This must be
// either AlignLeft, AlignCenter, or AlignRight.
func (t *TextView) SetTextAlign(align int) *TextView {
	t.align = align
	return t
}

// SetTextColor sets the initial color of the text.
func (t *TextView) SetTextColor(color tcell.Color) *TextView {
	t.textStyle = t.textStyle.Foreground(color)
	t.resetIndex()
	return t
}

// SetBackgroundColor overrides its implementation in Box to set the background
// color of this primitive. For backwards compatibility reasons, it also sets
// the background color of the main text element.
func (t *TextView) SetBackgroundColor(color tcell.Color) *Box {
	t.Box.SetBackgroundColor(color)
	t.textStyle = t.textStyle.Background(color)
	t.resetIndex()
	return t.Box
}

// SetTextStyle sets the initial style of the text. This style's background
// color also determines the background color of the main text element.
func (t *TextView) SetTextStyle(style tcell.Style) *TextView {
	t.textStyle = style
	t.resetIndex()
	return t
}

// SetText sets the text of this text view to the provided string. Previously
// contained text will be removed. As with writing to the text view io.Writer
// interface directly, this does not trigger an automatic redraw but it will
// trigger the "changed" callback if one is set.
func (t *TextView) SetText(text string) *TextView {
	t.Lock()
	defer t.Unlock()
	t.text.Reset()
	t.text.WriteString(text)
	t.resetIndex()
	if t.changed != nil {
		go t.changed()
	}
	return t
}

// GetText returns the current text of this text view. If "stripAllTags" is set
// to true, any region/style tags are stripped from the text. Note that any text
// that has been discarded due to [TextView.SetMaxLines] or
// [TextView.SetScrollable] will not be part of the returned text.
func (t *TextView) GetText(stripAllTags bool) string {
	if !stripAllTags || (!t.styleTags && !t.regionTags) {
		return t.text.String()
	}

	var (
		str   strings.Builder
		state *stepState
		text  = t.text.String()
		opts  stepOptions
		ch    string
	)
	if t.styleTags {
		opts = stepOptionsStyle
	}
	if t.regionTags {
		opts |= stepOptionsRegion
	}
	for len(text) > 0 {
		ch, text, state = step(text, state, opts)
		str.WriteString(ch)
	}
	return str.String()
}

// GetOriginalLineCount returns the number of lines in the original text buffer,
// without applying any wrapping. This is an expensive call as it needs to
// iterate over the entire text. Note that any text that has been discarded due
// to [TextView.SetMaxLines] or [TextView.SetScrollable] will not be part of the
// count.
func (t *TextView) GetOriginalLineCount() int {
	if t.text.Len() == 0 {
		return 0
	}

	var (
		state *stepState
		str       = t.text.String()
		lines int = 1
	)
	for len(str) > 0 {
		_, str, state = step(str, state, stepOptionsNone)
		if lineBreak, optional := state.LineBreak(); lineBreak && !optional {
			lines++
		}
	}

	return lines
}

// GetWrappedLineCount returns the number of lines in the text view, taking
// wrapping into account (if activated). This is an even more expensive call
// than [TextView.GetOriginalLineCount] as it needs to parse the text until the
// end and calculate the line breaks. It will also allocate memory for each
// line. Note that any text that has been discarded due to
// [TextView.SetMaxLines] or [TextView.SetScrollable] will not be part of the
// count. Calling this method before the text view was drawn for the first time
// will assume no wrapping.
func (t *TextView) GetWrappedLineCount() int {
	t.parseAhead(t.width, func(int, *textViewLine) bool {
		return false
	})
	return len(t.lineIndex)
}

// SetDynamicColors sets the flag that allows the text color to be changed
// dynamically with style tags. See class description for details.
func (t *TextView) SetDynamicColors(dynamic bool) *TextView {
	if t.styleTags != dynamic {
		t.resetIndex() // This invalidates the entire index.
	}
	t.styleTags = dynamic
	return t
}

// SetRegions sets the flag that allows to define regions in the text. See class
// description for details.
func (t *TextView) SetRegions(regions bool) *TextView {
	if t.regionTags != regions {
		t.resetIndex() // This invalidates the entire index.
	}
	t.regionTags = regions
	return t
}

// SetChangedFunc sets a handler function which is called when the text of the
// text view has changed. This is useful when text is written to this
// [io.Writer] in a separate goroutine. Doing so does not automatically cause
// the screen to be refreshed so you may want to use the "changed" handler to
// redraw the screen.
//
// Note that to avoid race conditions or deadlocks, there are a few rules you
// should follow:
//
//   - You can call [Application.Draw] from this handler.
//   - You can call [TextView.HasFocus] from this handler.
//   - During the execution of this handler, access to any other variables from
//     this primitive or any other primitive must be queued using
//     [Application.QueueUpdate].
//
// See package description for details on dealing with concurrency.
func (t *TextView) SetChangedFunc(handler func()) *TextView {
	t.changed = handler
	return t
}

// SetDoneFunc sets a handler which is called when the user presses on the
// following keys: Escape, Enter, Tab, Backtab. The key is passed to the
// handler.
func (t *TextView) SetDoneFunc(handler func(key tcell.Key)) *TextView {
	t.done = handler
	return t
}

// SetHighlightedFunc sets a handler which is called when the list of currently
// highlighted regions change. It receives a list of region IDs which were newly
// highlighted, those that are not highlighted anymore, and those that remain
// highlighted.
//
// Note that because regions are only determined when drawing the text view,
// this function can only fire for regions that have existed when the text view
// was last drawn.
func (t *TextView) SetHighlightedFunc(handler func(added, removed, remaining []string)) *TextView {
	t.highlighted = handler
	return t
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (t *TextView) SetFinishedFunc(handler func(key tcell.Key)) FormItem {
	t.finished = handler
	return t
}

// SetFormAttributes sets attributes shared by all form items.
func (t *TextView) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) FormItem {
	t.labelWidth = labelWidth
	t.backgroundColor = bgColor
	t.labelStyle = t.labelStyle.Foreground(labelColor)
	// We ignore the field background color because this is a read-only element.
	t.textStyle = tcell.StyleDefault.Foreground(fieldTextColor).Background(bgColor)
	return t
}

// ScrollTo scrolls to the specified row and column (both starting with 0).
func (t *TextView) ScrollTo(row, column int) *TextView {
	if !t.scrollable {
		return t
	}
	t.lineOffset = row
	t.columnOffset = column
	t.trackEnd = false
	return t
}

// ScrollToBeginning scrolls to the top left corner of the text if the text view
// is scrollable.
func (t *TextView) ScrollToBeginning() *TextView {
	if !t.scrollable {
		return t
	}
	t.trackEnd = false
	t.lineOffset = 0
	t.columnOffset = 0
	return t
}

// ScrollToEnd scrolls to the bottom left corner of the text if the text view
// is scrollable. Adding new rows to the end of the text view will cause it to
// scroll with the new data.
func (t *TextView) ScrollToEnd() *TextView {
	if !t.scrollable {
		return t
	}
	t.trackEnd = true
	t.columnOffset = 0
	return t
}

// GetScrollOffset returns the number of rows and columns that are skipped at
// the top left corner when the text view has been scrolled.
func (t *TextView) GetScrollOffset() (row, column int) {
	return t.lineOffset, t.columnOffset
}

// Clear removes all text from the buffer. This triggers the "changed" callback.
func (t *TextView) Clear() *TextView {
	t.Lock()
	defer t.Unlock()
	t.clear()
	if t.changed != nil {
		go t.changed()
	}
	return t
}

// clear is the internal implementation of clear. It is used by TextViewWriter
// and anywhere that we need to perform a write without locking the buffer.
func (t *TextView) clear() {
	t.text.Reset()
	t.resetIndex()
}

// Highlight specifies which regions should be highlighted. If highlight
// toggling is set to true (see [TextView.SetToggleHighlights]), the highlight
// of the provided regions is toggled (i.e. highlighted regions are
// un-highlighted and vice versa). If toggling is set to false, the provided
// regions are highlighted and all other regions will not be highlighted (you
// may also provide nil to turn off all highlights).
//
// For more information on regions, see class description. Empty region strings
// or regions not contained in the text are ignored.
//
// Text in highlighted regions will be drawn inverted, i.e. with their
// background and foreground colors swapped.
//
// If toggling is set to false, clicking outside of any region will remove all
// highlights.
//
// This function is expensive if a specified region is in a part of the text
// that has not yet been parsed.
func (t *TextView) Highlight(regionIDs ...string) *TextView {
	// Make sure we know these regions.
	t.parseAhead(t.lastWidth, func(lineNumber int, line *textViewLine) bool {
		for _, regionID := range regionIDs {
			if _, ok := t.regions[regionID]; !ok {
				return false
			}
		}
		return true
	})

	// Remove unknown regions.
	newRegions := make([]string, 0, len(regionIDs))
	for _, regionID := range regionIDs {
		if _, ok := t.regions[regionID]; ok {
			newRegions = append(newRegions, regionID)
		}
	}
	regionIDs = newRegions

	// Toggle highlights.
	if t.toggleHighlights {
		var newIDs []string
	HighlightLoop:
		for regionID := range t.highlights {
			for _, id := range regionIDs {
				if regionID == id {
					continue HighlightLoop
				}
			}
			newIDs = append(newIDs, regionID)
		}
		for _, regionID := range regionIDs {
			if _, ok := t.highlights[regionID]; !ok {
				newIDs = append(newIDs, regionID)
			}
		}
		regionIDs = newIDs
	} // Now we have a list of region IDs that end up being highlighted.

	// Determine added and removed regions.
	var added, removed, remaining []string
	if t.highlighted != nil {
		for _, regionID := range regionIDs {
			if _, ok := t.highlights[regionID]; ok {
				remaining = append(remaining, regionID)
				delete(t.highlights, regionID)
			} else {
				added = append(added, regionID)
			}
		}
		for regionID := range t.highlights {
			removed = append(removed, regionID)
		}
	}

	// Make new selection.
	t.highlights = make(map[string]struct{})
	for _, id := range regionIDs {
		if id == "" {
			continue
		}
		t.highlights[id] = struct{}{}
	}

	// Notify.
	if t.highlighted != nil && (len(added) > 0 || len(removed) > 0) {
		t.highlighted(added, removed, remaining)
	}

	return t
}

// GetHighlights returns the IDs of all currently highlighted regions.
func (t *TextView) GetHighlights() (regionIDs []string) {
	for id := range t.highlights {
		regionIDs = append(regionIDs, id)
	}
	return
}

// SetToggleHighlights sets a flag to determine how regions are highlighted.
// When set to true, the [TextView.Highlight] function (or a mouse click) will
// toggle the provided/selected regions. When set to false, [TextView.Highlight]
// (or a mouse click) will simply highlight the provided regions.
func (t *TextView) SetToggleHighlights(toggle bool) *TextView {
	t.toggleHighlights = toggle
	return t
}

// ScrollToHighlight will cause the visible area to be scrolled so that the
// highlighted regions appear in the visible area of the text view. This
// repositioning happens the next time the text view is drawn. It happens only
// once so you will need to call this function repeatedly to always keep
// highlighted regions in view.
//
// Nothing happens if there are no highlighted regions or if the text view is
// not scrollable.
func (t *TextView) ScrollToHighlight() *TextView {
	if len(t.highlights) == 0 || !t.scrollable || !t.regionTags {
		return t
	}
	t.scrollToHighlights = true
	t.trackEnd = false
	return t
}

// GetRegionText returns the text of the first region with the given ID. If
// dynamic colors are enabled, style tags are stripped from the text.
//
// If the region does not exist or if regions are turned off, an empty string
// is returned.
//
// This function can be expensive if the specified region is way beyond the
// visible area of the text view as the text needs to be parsed until the region
// can be found, or if the region does not contain any text.
func (t *TextView) GetRegionText(regionID string) string {
	if !t.regionTags || regionID == "" {
		return ""
	}

	// Parse until we find the region.
	lineNumber, ok := t.regions[regionID]
	if !ok {
		lineNumber = -1
		t.parseAhead(t.lastWidth, func(number int, line *textViewLine) bool {
			lineNumber, ok = t.regions[regionID]
			return ok
		})
		if lineNumber < 0 {
			return "" // We couldn't find this region.
		}
	}

	// Extract text from region.
	var (
		line       = t.lineIndex[lineNumber]
		text       = t.text.String()[line.offset:]
		st         = *line.state
		state      = &st
		options    = stepOptionsRegion
		regionText strings.Builder
	)
	if t.styleTags {
		options |= stepOptionsStyle
	}
	for len(text) > 0 {
		var ch string
		ch, text, state = step(text, state, options)
		if state.region == regionID {
			regionText.WriteString(ch)
		} else if regionText.Len() > 0 {
			break
		}
	}

	return regionText.String()
}

// Focus is called when this primitive receives focus.
func (t *TextView) Focus(delegate func(p Primitive)) {
	// Implemented here with locking because this is used by layout primitives.
	t.Lock()
	defer t.Unlock()

	// But if we're part of a form and not scrollable, there's nothing the user
	// can do here so we're finished.
	if t.finished != nil && !t.scrollable {
		t.finished(-1)
		return
	}

	t.Box.Focus(delegate)
}

// HasFocus returns whether or not this primitive has focus.
func (t *TextView) HasFocus() bool {
	// Implemented here with locking because this may be used in the "changed"
	// callback.
	t.Lock()
	defer t.Unlock()
	return t.Box.HasFocus()
}

// Write lets us implement the io.Writer interface.
func (t *TextView) Write(p []byte) (n int, err error) {
	t.Lock()
	defer t.Unlock()

	return t.write(p)
}

// write is the internal implementation of Write. It is used by [TextViewWriter]
// and anywhere that we need to perform a write without locking the buffer.
func (t *TextView) write(p []byte) (n int, err error) {
	// Notify at the end.
	changed := t.changed
	if changed != nil {
		defer func() {
			// We always call the "changed" function in a separate goroutine to avoid
			// deadlocks.
			go changed()
		}()
	}

	return t.text.Write(p)
}

// BatchWriter returns a new writer that can be used to write into the buffer
// but without Locking/Unlocking the buffer on every write, as [TextView.Write]
// and [TextView.Clear] do. The lock will be acquired once when BatchWriter is
// called, and will be released when the returned writer is closed. Example:
//
//	tv := tview.NewTextView()
//	w := tv.BatchWriter()
//	defer w.Close()
//	w.Clear()
//	fmt.Fprintln(w, "To sit in solemn silence")
//	fmt.Fprintln(w, "on a dull, dark, dock")
//	fmt.Println(tv.GetText(false))
//
// Note that using the batch writer requires you to manage any issues that may
// arise from concurrency yourself. See package description for details on
// dealing with concurrency.
func (t *TextView) BatchWriter() TextViewWriter {
	t.Lock()
	return TextViewWriter{
		t: t,
	}
}

// resetIndex resets all indexed data, including the line index.
func (t *TextView) resetIndex() {
	t.lineIndex = nil
	t.regions = make(map[string]int)
	t.longestLine = 0
}

// parseAhead parses the text buffer starting at the last line in
// [TextView.lineIndex] until either the end of the buffer or until stop returns
// true for the last complete line that was parsed. If wrapping is enabled,
// width will be used as the available screen width. If width is 0, it is
// assumed that there is no wrapping. This can happen when this function is
// called before the first time [TextView.Draw] is called.
//
// There is no guarantee that stop will ever be called.
//
// The function adds entries to the [TextView.lineIndex] slice and the
// [TextView.regions] map and adjusts [TextView.longestLine].
func (t *TextView) parseAhead(width int, stop func(lineNumber int, line *textViewLine) bool) {
	if t.text.Len() == 0 {
		return // No text. Nothing to parse.
	}

	// If width is 0, make it infinite.
	if width == 0 {
		width = math.MaxInt
	}

	// What kind of tags do we scan for?
	var options stepOptions
	if t.styleTags {
		options |= stepOptionsStyle
	}
	if t.regionTags {
		options |= stepOptionsRegion
	}

	// Start parsing at the last line in the index.
	var lastLine *textViewLine
	str := t.text.String()
	if len(t.lineIndex) == 0 {
		// Insert the first line.
		lastLine = &textViewLine{
			state: &stepState{
				unisegState: -1,
				style:       t.textStyle,
			},
		}
		t.lineIndex = append(t.lineIndex, lastLine)
	} else {
		// Reset the last line.
		lastLine = t.lineIndex[len(t.lineIndex)-1]
		lastLine.width = 0
		lastLine.length = 0
		str = str[lastLine.offset:]
	}

	// Parse.
	var (
		lastOption      int               // Text index of the last optional split point.
		lastOptionWidth int               // Line width at last optional split point.
		lastOptionState *stepState        // State at last optional split point.
		leftPos         int               // The current position in the line (only for left-alignment).
		offset          = lastLine.offset // Text index of the current position.
		st              = *lastLine.state // Current state.
		state           = &st             // Pointer to current state.
	)
	for len(str) > 0 {
		var c string
		region := state.region
		c, str, state = step(str, state, options)
		w := state.Width()
		if c == "\t" {
			if t.align == AlignLeft {
				w = TabSize - leftPos%TabSize
			} else {
				w = TabSize
			}
		}
		length := state.GrossLength()

		// Would it exceed the line width?
		if t.wrap && lastLine.width+w > width {
			if lastOptionWidth == 0 {
				// No split point so far. Just split at the current position.
				if stop(len(t.lineIndex)-1, lastLine) {
					return
				}
				st := *state
				lastLine = &textViewLine{
					offset: offset,
					state:  &st,
				}
				lastOption, lastOptionWidth, leftPos = 0, 0, 0
			} else {
				// Split at the last split point.
				newLine := &textViewLine{
					offset: lastLine.offset + lastOption,
					width:  lastLine.width - lastOptionWidth,
					length: lastLine.length - lastOption,
					state:  lastOptionState,
				}
				lastLine.width = lastOptionWidth
				lastLine.length = lastOption
				if stop(len(t.lineIndex)-1, lastLine) {
					return
				}
				lastLine = newLine
				lastOption, lastOptionWidth = 0, 0
				leftPos -= lastOptionWidth
			}
			t.lineIndex = append(t.lineIndex, lastLine)
		}

		// Move ahead.
		lastLine.width += w
		lastLine.length += length
		offset += length
		leftPos += w

		// Do we have a new longest line?
		if lastLine.width > t.longestLine {
			t.longestLine = lastLine.width
		}

		// Check for split points.
		if lineBreak, optional := state.LineBreak(); lineBreak {
			if optional {
				if t.wrap && t.wordWrap {
					// Remember this split point.
					lastOption = offset - lastLine.offset
					lastOptionWidth = lastLine.width
					st := *state
					lastOptionState = &st
				}
			} else {
				// We must split here.
				if stop(len(t.lineIndex)-1, lastLine) {
					return
				}
				st := *state
				lastLine = &textViewLine{
					offset: offset,
					state:  &st,
				}
				t.lineIndex = append(t.lineIndex, lastLine)
				lastOption, lastOptionWidth, leftPos = 0, 0, 0
			}
		}

		// Add new regions if any.
		if t.regionTags && state.region != "" && state.region != region {
			if _, ok := t.regions[state.region]; !ok {
				t.regions[state.region] = len(t.lineIndex) - 1
			}
		}
	}
}

// Draw draws this primitive onto the screen.
func (t *TextView) Draw(screen tcell.Screen) {
	t.Box.DrawForSubclass(screen, t)
	t.Lock()
	defer t.Unlock()

	// Get the available size.
	x, y, width, height := t.GetInnerRect()
	t.pageSize = height

	// Draw label.
	_, labelBg, _ := t.labelStyle.Decompose()
	if t.labelWidth > 0 {
		labelWidth := t.labelWidth
		if labelWidth > width {
			labelWidth = width
		}
		printWithStyle(screen, t.label, x, y, 0, labelWidth, AlignLeft, t.labelStyle, labelBg == tcell.ColorDefault)
		x += labelWidth
		width -= labelWidth
	} else {
		_, _, drawnWidth := printWithStyle(screen, t.label, x, y, 0, width, AlignLeft, t.labelStyle, labelBg == tcell.ColorDefault)
		x += drawnWidth
		width -= drawnWidth
	}

	// What's the space for the text element?
	if t.width > 0 && t.width < width {
		width = t.width
	}
	if t.height > 0 && t.height < height {
		height = t.height
	}
	if width <= 0 {
		return // No space left for the text area.
	}

	// Draw the text element if necessary.
	_, bg, _ := t.textStyle.Decompose()
	if bg != t.backgroundColor {
		for row := 0; row < height; row++ {
			for column := 0; column < width; column++ {
				screen.SetContent(x+column, y+row, ' ', nil, t.textStyle)
			}
		}
	}

	// If the width has changed, we need to reindex.
	if width != t.lastWidth && t.wrap {
		t.resetIndex()
	}
	t.lastWidth = width

	// What are our parse options?
	var options stepOptions
	if t.styleTags {
		options |= stepOptionsStyle
	}
	if t.regionTags {
		options |= stepOptionsRegion
	}

	// Scroll to highlighted regions.
	if t.regionTags && t.scrollToHighlights {
		// Make sure we know all highlighted regions.
		t.parseAhead(width, func(lineNumber int, line *textViewLine) bool {
			for regionID := range t.highlights {
				if _, ok := t.regions[regionID]; !ok {
					return false
				}
				t.highlights[regionID] = struct{}{}
			}
			return true
		})

		// What is the line range for all highlighted regions?
		var (
			firstRegion                string
			fromHighlight, toHighlight int
		)
		for regionID := range t.highlights {
			// We can safely assume that the region is known.
			line := t.regions[regionID]
			if firstRegion == "" || line > toHighlight {
				toHighlight = line
			}
			if firstRegion == "" || line < fromHighlight {
				fromHighlight = line
				firstRegion = regionID
			}
		}
		if firstRegion != "" {
			// Do we fit the entire height?
			if toHighlight-fromHighlight+1 < height {
				// Yes, let's center the highlights.
				t.lineOffset = (fromHighlight + toHighlight - height) / 2
			} else {
				// No, let's move to the start of the highlights.
				t.lineOffset = fromHighlight
			}

			// If the highlight is too far to the right, move it to the middle.
			if t.wrap {
				// Find the first highlight's column in screen space.
				line := t.lineIndex[fromHighlight]
				st := *line.state
				state := &st
				str := t.text.String()[line.offset:]
				var posHighlight int
				for len(str) > 0 && posHighlight < line.width && state.region != firstRegion {
					_, str, state = step(str, state, options)
					posHighlight += state.Width()
				}

				if posHighlight-t.columnOffset > 3*width/4 {
					t.columnOffset = posHighlight - width/2
				}

				// If the highlight is off-screen on the left, move it on-screen.
				if posHighlight-t.columnOffset < 0 {
					t.columnOffset = posHighlight - width/4
				}
			}
		}
	}
	t.scrollToHighlights = false

	// Make sure our index has enough lines.
	t.parseAhead(width, func(lineNumber int, line *textViewLine) bool {
		return lineNumber >= t.lineOffset+height
	})

	// Adjust line offset.
	if t.trackEnd {
		t.parseAhead(width, func(lineNumber int, line *textViewLine) bool {
			return false
		})
		t.lineOffset = len(t.lineIndex) - height
	}
	if t.lineOffset > len(t.lineIndex)-height {
		t.lineOffset = len(t.lineIndex) - height
	}
	if t.lineOffset < 0 {
		t.lineOffset = 0
	}

	// Adjust column offset.
	if t.align == AlignLeft || t.align == AlignRight {
		if t.columnOffset+width > t.longestLine {
			t.columnOffset = t.longestLine - width
		}
		if t.columnOffset < 0 {
			t.columnOffset = 0
		}
	} else { // AlignCenter.
		half := (t.longestLine - width) / 2
		if half > 0 {
			if t.columnOffset > half {
				t.columnOffset = half
			}
			if t.columnOffset < -half {
				t.columnOffset = -half
			}
		} else {
			t.columnOffset = 0
		}
	}

	// Draw visible lines.
	for line := t.lineOffset; line < len(t.lineIndex); line++ {
		// Are we done?
		if line-t.lineOffset >= height {
			break
		}

		info := t.lineIndex[line]
		info.regions = nil

		// Determine starting point of the text and the screen.
		var skipWidth, xPos int
		switch t.align {
		case AlignLeft:
			skipWidth = t.columnOffset
		case AlignCenter:
			skipWidth = t.columnOffset + (info.width-width)/2
			if skipWidth < 0 {
				skipWidth = 0
				xPos = (width-info.width)/2 - t.columnOffset
			}
		case AlignRight:
			maxWidth := width
			if t.longestLine > width {
				maxWidth = t.longestLine
			}
			skipWidth = t.columnOffset - (maxWidth - info.width)
			if skipWidth < 0 {
				skipWidth = 0
				xPos = maxWidth - info.width - t.columnOffset
			}
		}

		// Draw the line text.
		str := t.text.String()[info.offset:]
		st := *info.state
		state := &st
		var processed int
		for len(str) > 0 && xPos < width && processed < info.length {
			var ch string
			ch, str, state = step(str, state, options)
			w := state.Width()
			if ch == "\t" {
				if t.align == AlignLeft {
					w = TabSize - xPos%TabSize
				} else {
					w = TabSize
				}
			}
			processed += state.GrossLength()

			// Don't draw anything while we skip characters.
			if skipWidth > 0 {
				skipWidth -= w
				continue
			}

			// Draw this character.
			if w > 0 {
				style := state.Style()

				// Do we highlight this character?
				var highlighted bool
				if state.region != "" {
					if _, ok := t.highlights[state.region]; ok {
						highlighted = true
					}
				}
				if highlighted {
					fg, bg, _ := style.Decompose()
					if bg == t.backgroundColor {
						r, g, b := fg.RGB()
						c := colorful.Color{R: float64(r) / 255, G: float64(g) / 255, B: float64(b) / 255}
						_, _, li := c.Hcl()
						if li < .5 {
							bg = tcell.ColorWhite
						} else {
							bg = tcell.ColorBlack
						}
					}
					style = style.Background(fg).Foreground(bg)
				}

				// Paint on screen.
				for offset := w - 1; offset >= 0; offset-- {
					runes := []rune(ch)
					if offset == 0 {
						screen.SetContent(x+xPos+offset, y+line-t.lineOffset, runes[0], runes[1:], style)
					} else {
						screen.SetContent(x+xPos+offset, y+line-t.lineOffset, ' ', nil, style)
					}
				}

				// Register this region.
				if state.region != "" {
					if info.regions == nil {
						info.regions = make(map[string][2]int)
					}
					fromTo, ok := info.regions[state.region]
					if !ok {
						fromTo = [2]int{xPos, xPos + w}
					} else {
						if xPos < fromTo[0] {
							fromTo[0] = xPos
						}
						if xPos+w > fromTo[1] {
							fromTo[1] = xPos + w
						}
					}
					info.regions[state.region] = fromTo
				}
			}

			xPos += w
		}
	}

	// If this view is not scrollable, we'll purge the buffer of lines that have
	// scrolled out of view.
	var purgeStart int
	if !t.scrollable && t.lineOffset > 0 {
		purgeStart = t.lineOffset
	}

	// If we reached the maximum number of lines, we'll purge the buffer of the
	// oldest lines.
	if t.maxLines > 0 && len(t.lineIndex) > t.maxLines {
		purgeStart = len(t.lineIndex) - t.maxLines
	}

	// Purge.
	if purgeStart > 0 && purgeStart < len(t.lineIndex) {
		newText := t.text.String()[t.lineIndex[purgeStart].offset:]
		t.text.Reset()
		t.text.WriteString(newText)
		t.resetIndex()
		t.lineOffset = 0
	}
}

// InputHandler returns the handler for this primitive.
func (t *TextView) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return t.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		key := event.Key()

		if key == tcell.KeyEscape || key == tcell.KeyEnter || key == tcell.KeyTab || key == tcell.KeyBacktab {
			if t.done != nil {
				t.done(key)
			}
			if t.finished != nil {
				t.finished(key)
			}
			return
		}

		if !t.scrollable {
			return
		}

		switch key {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g': // Home.
				t.trackEnd = false
				t.lineOffset = 0
				t.columnOffset = 0
			case 'G': // End.
				t.trackEnd = true
				t.columnOffset = 0
			case 'j': // Down.
				t.lineOffset++
			case 'k': // Up.
				t.trackEnd = false
				t.lineOffset--
			case 'h': // Left.
				t.columnOffset--
			case 'l': // Right.
				t.columnOffset++
			}
		case tcell.KeyHome:
			t.trackEnd = false
			t.lineOffset = 0
			t.columnOffset = 0
		case tcell.KeyEnd:
			t.trackEnd = true
			t.columnOffset = 0
		case tcell.KeyUp:
			t.trackEnd = false
			t.lineOffset--
		case tcell.KeyDown:
			t.lineOffset++
		case tcell.KeyLeft:
			t.columnOffset--
		case tcell.KeyRight:
			t.columnOffset++
		case tcell.KeyPgDn, tcell.KeyCtrlF:
			t.lineOffset += t.pageSize
		case tcell.KeyPgUp, tcell.KeyCtrlB:
			t.trackEnd = false
			t.lineOffset -= t.pageSize
		}
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (t *TextView) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return t.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		x, y := event.Position()
		if !t.InRect(x, y) {
			return false, nil
		}

		rectX, rectY, width, height := t.GetInnerRect()
		switch action {
		case MouseLeftDown:
			setFocus(t)
			consumed = true
		case MouseLeftClick:
			if t.regionTags && t.InInnerRect(x, y) {
				// Find a region to highlight.
				x -= rectX
				y -= rectY
				var highlightedID string
				if y+t.lineOffset < len(t.lineIndex) {
					line := t.lineIndex[y+t.lineOffset]
					for regionID, fromTo := range line.regions {
						if x >= fromTo[0] && x < fromTo[1] {
							highlightedID = regionID
							break
						}
					}
				}
				if highlightedID != "" {
					t.Highlight(highlightedID)
				} else if !t.toggleHighlights {
					t.Highlight()
				}
			}
			consumed = true
		case MouseScrollUp:
			if !t.scrollable {
				break
			}
			t.trackEnd = false
			t.lineOffset--
			consumed = true
		case MouseScrollDown:
			if !t.scrollable {
				break
			}
			t.lineOffset++
			if len(t.lineIndex)-t.lineOffset < height {
				// If we scroll to the end, turn on tracking.
				t.parseAhead(width, func(lineNumber int, line *textViewLine) bool {
					return len(t.lineIndex)-t.lineOffset < height
				})
				if len(t.lineIndex)-t.lineOffset < height {
					t.trackEnd = true
				}
			}
			consumed = true
		}

		return
	})
}
