/*
Package tview implements rich widgets for terminal based user interfaces. The
widgets provided with this package are useful for data exploration and data
entry.

# Widgets

The package implements the following widgets:

  - [TextView]: A scrollable window that display multi-colored text. Text may
    also be highlighted.
  - [TextArea]: An editable multi-line text area.
  - [Table]: A scrollable display of tabular data. Table cells, rows, or columns
    may also be highlighted.
  - [TreeView]: A scrollable display for hierarchical data. Tree nodes can be
    highlighted, collapsed, expanded, and more.
  - [List]: A navigable text list with optional keyboard shortcuts.
  - [InputField]: One-line input fields to enter text.
  - [DropDown]: Drop-down selection fields.
  - [Checkbox]: Selectable checkbox for boolean values.
  - [Image]: Displays images.
  - [Button]: Buttons which get activated when the user selects them.
  - [Form]: Forms composed of input fields, drop down selections, checkboxes,
    and buttons.
  - [Modal]: A centered window with a text message and one or more buttons.
  - [Grid]: A grid based layout manager.
  - [Flex]: A Flexbox based layout manager.
  - [Pages]: A page based layout manager.

The package also provides Application which is used to poll the event queue and
draw widgets on screen.

# Hello World

The following is a very basic example showing a box with the title "Hello,
world!":

	package main

	import (
		"github.com/rivo/tview"
	)

	func main() {
		box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
		if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
			panic(err)
		}
	}

First, we create a box primitive with a border and a title. Then we create an
application, set the box as its root primitive, and run the event loop. The
application exits when the application's [Application.Stop] function is called
or when Ctrl-C is pressed.

# More Demos

You will find more demos in the "demos" subdirectory. It also contains a
presentation (written using tview) which gives an overview of the different
widgets and how they can be used.

# Styles, Colors, and Hyperlinks

Throughout this package, styles are specified using the [tcell.Style] type.
Styles specify colors with the [tcell.Color] type. Functions such as
[tcell.GetColor], [tcell.NewHexColor], and [tcell.NewRGBColor] can be used to
create colors from W3C color names or RGB values. The [tcell.Style] type also
allows you to specify text attributes such as "bold" or "underline" or a URL
which some terminals use to display hyperlinks.

Almost all strings which are displayed may contain style tags. A style tag's
content is always wrapped in square brackets. In its simplest form, a style tag
specifies the foreground color of the text. Colors in these tags are W3C color
names or six hexadecimal digits following a hash tag. Examples:

	This is a [red]warning[white]!
	The sky is [#8080ff]blue[#ffffff].

A style tag changes the style of the characters following that style tag. There
is no style stack and no nesting of style tags.

Style tags are used in almost everything from box titles, list text, form item
labels, to table cells. In a [TextView], this functionality has to be switched
on explicitly. See the [TextView] documentation for more information.

A style tag's full format looks like this:

	[<foreground>:<background>:<attribute flags>:<url>]

Each of the four fields can be left blank and trailing fields can be omitted.
(Empty square brackets "[]", however, are not considered style tags.) Fields
that are not specified will be left unchanged. A field with just a dash ("-")
means "reset to default".

You can specify the following flags to turn on certain attributes (some flags
may not be supported by your terminal):

	l: blink
	b: bold
	i: italic
	d: dim
	r: reverse (switch foreground and background color)
	u: underline
	s: strike-through

Use uppercase letters to turn off the corresponding attribute, for example,
"B" to turn off bold. Uppercase letters have no effect if the attribute was not
previously set.

Setting a URL allows you to turn a piece of text into a hyperlink in some
terminals. Specify a dash ("-") to specify the end of the hyperlink. Hyperlinks
must only contain single-byte characters (e.g. ASCII) and they may not contain
bracket characters ("[" or "]").

Examples:

	[yellow]Yellow text
	[yellow:red]Yellow text on red background
	[:red]Red background, text color unchanged
	[yellow::u]Yellow text underlined
	[::bl]Bold, blinking text
	[::-]Colors unchanged, flags reset
	[-]Reset foreground color
	[::i]Italic and [::I]not italic
	Click [:::https://example.com]here[:::-] for example.com.
	Send an email to [:::mailto:her@example.com]her/[:::mail:him@example.com]him/[:::mail:them@example.com]them[:::-].
	[-:-:-:-]Reset everything
	[:]No effect
	[]Not a valid style tag, will print square brackets as they are

In the rare event that you want to display a string such as "[red]" or
"[#00ff1a]" without applying its effect, you need to put an opening square
bracket before the closing square bracket. Note that the text inside the
brackets will be matched less strictly than region or colors tags. I.e. any
character that may be used in color or region tags will be recognized. Examples:

	[red[]      will be output as [red]
	["123"[]    will be output as ["123"]
	[#6aff00[[] will be output as [#6aff00[]
	[a#"[[[]    will be output as [a#"[[]
	[]          will be output as [] (see style tags above)
	[[]         will be output as [[] (not an escaped tag)

You can use the Escape() function to insert brackets automatically where needed.

# Styles

When primitives are instantiated, they are initialized with colors taken from
the global [Styles] variable. You may change this variable to adapt the look and
feel of the primitives to your preferred style.

Note that most terminals will not report information about their color theme.
This package therefore does not support using the terminal's color theme. The
default style is a dark theme and you must change the [Styles] variable to
switch to a light (or other) theme.

# Unicode Support

This package supports all unicode characters supported by your terminal.

# Mouse Support

If your terminal supports mouse events, you can enable mouse support for your
application by calling [Application.EnableMouse]. Note that this may interfere
with your terminal's default mouse behavior. Mouse support is disabled by
default.

# Concurrency

Many functions in this package are not thread-safe. For many applications, this
is not an issue: If your code makes changes in response to key events, the
corresponding callback function will execute in the main goroutine and thus will
not cause any race conditions. (Exceptions to this are documented.)

If you access your primitives from other goroutines, however, you will need to
synchronize execution. The easiest way to do this is to call
[Application.QueueUpdate] or [Application.QueueUpdateDraw] (see the function
documentation for details):

	go func() {
	  app.QueueUpdateDraw(func() {
	    table.SetCellSimple(0, 0, "Foo bar")
	  })
	}()

One exception to this is the io.Writer interface implemented by [TextView]. You
can safely write to a [TextView] from any goroutine. See the [TextView]
documentation for details.

You can also call [Application.Draw] from any goroutine without having to wrap
it in [Application.QueueUpdate]. And, as mentioned above, key event callbacks
are executed in the main goroutine and thus should not use
[Application.QueueUpdate] as that may lead to deadlocks. It is also not
necessary to call [Application.Draw] from such callbacks as it will be called
automatically.

# Type Hierarchy

All widgets listed above contain the [Box] type. All of [Box]'s functions are
therefore available for all widgets, too. Please note that if you are using the
functions of [Box] on a subclass, they will return a *Box, not the subclass.
This is a Golang limitation. So while tview supports method chaining in many
places, these chains must be broken when using [Box]'s functions. Example:

	// This will cause "textArea" to be an empty Box.
	textArea := tview.NewTextArea().
		SetMaxLength(256).
		SetPlaceholder("Enter text here").
		SetBorder(true)

You will need to call [Box.SetBorder] separately:

	textArea := tview.NewTextArea().
		SetMaxLength(256).
		SetPlaceholder("Enter text here")
	texArea.SetBorder(true)

All widgets also implement the [Primitive] interface.

The tview package's rendering is based on version 2 of
https://github.com/gdamore/tcell. It uses types and constants from that package
(e.g. colors, styles, and keyboard values).
*/
package tview
