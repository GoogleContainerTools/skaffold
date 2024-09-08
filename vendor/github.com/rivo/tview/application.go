package tview

import (
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	// The size of the event/update/redraw channels.
	queueSize = 100

	// The minimum time between two consecutive redraws.
	redrawPause = 50 * time.Millisecond
)

// DoubleClickInterval specifies the maximum time between clicks to register a
// double click rather than click.
var DoubleClickInterval = 500 * time.Millisecond

// MouseAction indicates one of the actions the mouse is logically doing.
type MouseAction int16

// Available mouse actions.
const (
	MouseMove MouseAction = iota
	MouseLeftDown
	MouseLeftUp
	MouseLeftClick
	MouseLeftDoubleClick
	MouseMiddleDown
	MouseMiddleUp
	MouseMiddleClick
	MouseMiddleDoubleClick
	MouseRightDown
	MouseRightUp
	MouseRightClick
	MouseRightDoubleClick
	MouseScrollUp
	MouseScrollDown
	MouseScrollLeft
	MouseScrollRight

	// The following special value will not be provided as a mouse action but
	// indicate that an overridden mouse event was consumed. See
	// [Box.SetMouseCapture] for details.
	MouseConsumed
)

// queuedUpdate represented the execution of f queued by
// Application.QueueUpdate(). If "done" is not nil, it receives exactly one
// element after f has executed.
type queuedUpdate struct {
	f    func()
	done chan struct{}
}

// Application represents the top node of an application.
//
// It is not strictly required to use this class as none of the other classes
// depend on it. However, it provides useful tools to set up an application and
// plays nicely with all widgets.
//
// The following command displays a primitive p on the screen until Ctrl-C is
// pressed:
//
//	if err := tview.NewApplication().SetRoot(p, true).Run(); err != nil {
//	    panic(err)
//	}
type Application struct {
	sync.RWMutex

	// The application's screen. Apart from Run(), this variable should never be
	// set directly. Always use the screenReplacement channel after calling
	// Fini(), to set a new screen (or nil to stop the application).
	screen tcell.Screen

	// The primitive which currently has the keyboard focus.
	focus Primitive

	// The root primitive to be seen on the screen.
	root Primitive

	// Whether or not the application resizes the root primitive.
	rootFullscreen bool

	// Set to true if mouse events are enabled.
	enableMouse bool

	// Set to true if paste events are enabled.
	enablePaste bool

	// An optional capture function which receives a key event and returns the
	// event to be forwarded to the default input handler (nil if nothing should
	// be forwarded).
	inputCapture func(event *tcell.EventKey) *tcell.EventKey

	// An optional callback function which is invoked just before the root
	// primitive is drawn.
	beforeDraw func(screen tcell.Screen) bool

	// An optional callback function which is invoked after the root primitive
	// was drawn.
	afterDraw func(screen tcell.Screen)

	// Used to send screen events from separate goroutine to main event loop
	events chan tcell.Event

	// Functions queued from goroutines, used to serialize updates to primitives.
	updates chan queuedUpdate

	// An object that the screen variable will be set to after Fini() was called.
	// Use this channel to set a new screen object for the application
	// (screen.Init() and draw() will be called implicitly). A value of nil will
	// stop the application.
	screenReplacement chan tcell.Screen

	// An optional capture function which receives a mouse event and returns the
	// event to be forwarded to the default mouse handler (nil if nothing should
	// be forwarded).
	mouseCapture func(event *tcell.EventMouse, action MouseAction) (*tcell.EventMouse, MouseAction)

	mouseCapturingPrimitive Primitive        // A Primitive returned by a MouseHandler which will capture future mouse events.
	lastMouseX, lastMouseY  int              // The last position of the mouse.
	mouseDownX, mouseDownY  int              // The position of the mouse when its button was last pressed.
	lastMouseClick          time.Time        // The time when a mouse button was last clicked.
	lastMouseButtons        tcell.ButtonMask // The last mouse button state.
}

// NewApplication creates and returns a new application.
func NewApplication() *Application {
	return &Application{
		events:            make(chan tcell.Event, queueSize),
		updates:           make(chan queuedUpdate, queueSize),
		screenReplacement: make(chan tcell.Screen, 1),
	}
}

// SetInputCapture sets a function which captures all key events before they are
// forwarded to the key event handler of the primitive which currently has
// focus. This function can then choose to forward that key event (or a
// different one) by returning it or stop the key event processing by returning
// nil.
//
// The only default global key event is Ctrl-C which stops the application. It
// requires special handling:
//
//   - If you do not wish to change the default behavior, return the original
//     event object passed to your input capture function.
//   - If you wish to block Ctrl-C from any functionality, return nil.
//   - If you do not wish Ctrl-C to stop the application but still want to
//     forward the Ctrl-C event to primitives down the hierarchy, return a new
//     key event with the same key and modifiers, e.g.
//     tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone).
//
// Pasted key events are not forwarded to the input capture function if pasting
// is enabled (see [Application.EnablePaste]).
func (a *Application) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *Application {
	a.inputCapture = capture
	return a
}

// GetInputCapture returns the function installed with SetInputCapture() or nil
// if no such function has been installed.
func (a *Application) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return a.inputCapture
}

// SetMouseCapture sets a function which captures mouse events (consisting of
// the original tcell mouse event and the semantic mouse action) before they are
// forwarded to the appropriate mouse event handler. This function can then
// choose to forward that event (or a different one) by returning it or stop
// the event processing by returning a nil mouse event. In such a case, the
// event is considered consumed and the screen will be redrawn.
func (a *Application) SetMouseCapture(capture func(event *tcell.EventMouse, action MouseAction) (*tcell.EventMouse, MouseAction)) *Application {
	a.mouseCapture = capture
	return a
}

// GetMouseCapture returns the function installed with SetMouseCapture() or nil
// if no such function has been installed.
func (a *Application) GetMouseCapture() func(event *tcell.EventMouse, action MouseAction) (*tcell.EventMouse, MouseAction) {
	return a.mouseCapture
}

// SetScreen allows you to provide your own tcell.Screen object. For most
// applications, this is not needed and you should be familiar with
// tcell.Screen when using this function.
//
// This function is typically called before the first call to Run(). Init() need
// not be called on the screen.
func (a *Application) SetScreen(screen tcell.Screen) *Application {
	if screen == nil {
		return a // Invalid input. Do nothing.
	}

	a.Lock()
	if a.screen == nil {
		// Run() has not been called yet.
		a.screen = screen
		a.Unlock()
		screen.Init()
		return a
	}

	// Run() is already in progress. Exchange screen.
	oldScreen := a.screen
	a.Unlock()
	oldScreen.Fini()
	a.screenReplacement <- screen

	return a
}

// EnableMouse enables mouse events or disables them (if "false" is provided).
func (a *Application) EnableMouse(enable bool) *Application {
	a.Lock()
	defer a.Unlock()
	if enable != a.enableMouse && a.screen != nil {
		if enable {
			a.screen.EnableMouse()
		} else {
			a.screen.DisableMouse()
		}
	}
	a.enableMouse = enable
	return a
}

// EnablePaste enables the capturing of paste events or disables them (if
// "false" is provided). This must be supported by the terminal.
//
// Widgets won't interpret paste events for navigation or selection purposes.
// Paste events are typically only used to insert a block of text into an
// [InputField] or a [TextArea].
func (a *Application) EnablePaste(enable bool) *Application {
	a.Lock()
	defer a.Unlock()
	if enable != a.enablePaste && a.screen != nil {
		if enable {
			a.screen.EnablePaste()
		} else {
			a.screen.DisablePaste()
		}
	}
	a.enablePaste = enable
	return a
}

// Run starts the application and thus the event loop. This function returns
// when [Application.Stop] was called.
//
// Note that while an application is running, it fully claims stdin, stdout, and
// stderr. If you use these standard streams, they may not work as expected.
// Consider stopping the application first or suspending it (using
// [Application.Suspend]) if you have to interact with the standard streams, for
// example when needing to print a call stack during a panic.
func (a *Application) Run() error {
	var (
		err, appErr error
		lastRedraw  time.Time   // The time the screen was last redrawn.
		redrawTimer *time.Timer // A timer to schedule the next redraw.
	)
	a.Lock()

	// Make a screen if there is none yet.
	if a.screen == nil {
		a.screen, err = tcell.NewScreen()
		if err != nil {
			a.Unlock()
			return err
		}
		if err = a.screen.Init(); err != nil {
			a.Unlock()
			return err
		}
		if a.enableMouse {
			a.screen.EnableMouse()
		} else {
			a.screen.DisableMouse()
		}
		if a.enablePaste {
			a.screen.EnablePaste()
		} else {
			a.screen.DisablePaste()
		}
	}

	// We catch panics to clean up because they mess up the terminal.
	defer func() {
		if p := recover(); p != nil {
			if a.screen != nil {
				a.screen.Fini()
			}
			panic(p)
		}
	}()

	// Draw the screen for the first time.
	a.Unlock()
	a.draw()

	// Separate loop to wait for screen events.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			a.RLock()
			screen := a.screen
			a.RUnlock()
			if screen == nil {
				// We have no screen. Let's stop.
				a.QueueEvent(nil)
				break
			}

			// Wait for next event and queue it.
			event := screen.PollEvent()
			if event != nil {
				// Regular event. Queue.
				a.QueueEvent(event)
				continue
			}

			// A screen was finalized (event is nil). Wait for a new screen.
			screen = <-a.screenReplacement
			if screen == nil {
				// No new screen. We're done.
				a.QueueEvent(nil) // Stop the event loop.
				return
			}

			// We have a new screen. Keep going.
			a.Lock()
			a.screen = screen
			enableMouse := a.enableMouse
			enablePaste := a.enablePaste
			a.Unlock()

			// Initialize and draw this screen.
			if err := screen.Init(); err != nil {
				panic(err)
			}
			if enableMouse {
				screen.EnableMouse()
			} else {
				screen.DisableMouse()
			}
			if enablePaste {
				screen.EnablePaste()
			} else {
				screen.DisablePaste()
			}
			a.draw()
		}
	}()

	// Start event loop.
	var (
		pasteBuffer strings.Builder
		pasting     bool // Set to true while we receive paste key events.
	)
EventLoop:
	for {
		select {
		// If we received an event, handle it.
		case event := <-a.events:
			if event == nil {
				break EventLoop
			}

			switch event := event.(type) {
			case *tcell.EventKey:
				// If we are pasting, collect runes, nothing else.
				if pasting {
					switch event.Key() {
					case tcell.KeyRune:
						pasteBuffer.WriteRune(event.Rune())
					case tcell.KeyEnter:
						pasteBuffer.WriteRune('\n')
					case tcell.KeyTab:
						pasteBuffer.WriteRune('\t')
					}
					break
				}

				a.RLock()
				root := a.root
				inputCapture := a.inputCapture
				a.RUnlock()

				// Intercept keys.
				var draw bool
				originalEvent := event
				if inputCapture != nil {
					event = inputCapture(event)
					if event == nil {
						a.draw()
						break // Don't forward event.
					}
					draw = true
				}

				// Ctrl-C closes the application.
				if event == originalEvent && event.Key() == tcell.KeyCtrlC {
					a.Stop()
					break
				}

				// Pass other key events to the root primitive.
				if root != nil && root.HasFocus() {
					if handler := root.InputHandler(); handler != nil {
						handler(event, func(p Primitive) {
							a.SetFocus(p)
						})
						draw = true
					}
				}

				// Redraw.
				if draw {
					a.draw()
				}
			case *tcell.EventPaste:
				if !a.enablePaste {
					break
				}
				if event.Start() {
					pasting = true
					pasteBuffer.Reset()
				} else if event.End() {
					pasting = false
					a.RLock()
					root := a.root
					a.RUnlock()
					if root != nil && root.HasFocus() && pasteBuffer.Len() > 0 {
						// Pass paste event to the root primitive.
						if handler := root.PasteHandler(); handler != nil {
							handler(pasteBuffer.String(), func(p Primitive) {
								a.SetFocus(p)
							})
						}

						// Redraw.
						a.draw()
					}
				}
			case *tcell.EventResize:
				if time.Since(lastRedraw) < redrawPause {
					if redrawTimer != nil {
						redrawTimer.Stop()
					}
					redrawTimer = time.AfterFunc(redrawPause, func() {
						a.events <- event
					})
				}
				a.RLock()
				screen := a.screen
				a.RUnlock()
				if screen == nil {
					break
				}
				lastRedraw = time.Now()
				screen.Clear()
				a.draw()
			case *tcell.EventMouse:
				consumed, isMouseDownAction := a.fireMouseActions(event)
				if consumed {
					a.draw()
				}
				a.lastMouseButtons = event.Buttons()
				if isMouseDownAction {
					a.mouseDownX, a.mouseDownY = event.Position()
				}
			case *tcell.EventError:
				appErr = event
				a.Stop()
			}

		// If we have updates, now is the time to execute them.
		case update := <-a.updates:
			update.f()
			if update.done != nil {
				update.done <- struct{}{}
			}
		}
	}

	// Wait for the event loop to finish.
	wg.Wait()
	a.screen = nil

	return appErr
}

// fireMouseActions analyzes the provided mouse event, derives mouse actions
// from it and then forwards them to the corresponding primitives.
func (a *Application) fireMouseActions(event *tcell.EventMouse) (consumed, isMouseDownAction bool) {
	// We want to relay follow-up events to the same target primitive.
	var targetPrimitive Primitive

	// Helper function to fire a mouse action.
	fire := func(action MouseAction) {
		switch action {
		case MouseLeftDown, MouseMiddleDown, MouseRightDown:
			isMouseDownAction = true
		}

		// Intercept event.
		if a.mouseCapture != nil {
			event, action = a.mouseCapture(event, action)
			if event == nil {
				consumed = true
				return // Don't forward event.
			}
		}

		// Determine the target primitive.
		var primitive, capturingPrimitive Primitive
		if a.mouseCapturingPrimitive != nil {
			primitive = a.mouseCapturingPrimitive
			targetPrimitive = a.mouseCapturingPrimitive
		} else if targetPrimitive != nil {
			primitive = targetPrimitive
		} else {
			primitive = a.root
		}
		if primitive != nil {
			if handler := primitive.MouseHandler(); handler != nil {
				var wasConsumed bool
				wasConsumed, capturingPrimitive = handler(action, event, func(p Primitive) {
					a.SetFocus(p)
				})
				if wasConsumed {
					consumed = true
				}
			}
		}
		a.mouseCapturingPrimitive = capturingPrimitive
	}

	x, y := event.Position()
	buttons := event.Buttons()
	clickMoved := x != a.mouseDownX || y != a.mouseDownY
	buttonChanges := buttons ^ a.lastMouseButtons

	if x != a.lastMouseX || y != a.lastMouseY {
		fire(MouseMove)
		a.lastMouseX = x
		a.lastMouseY = y
	}

	for _, buttonEvent := range []struct {
		button                  tcell.ButtonMask
		down, up, click, dclick MouseAction
	}{
		{tcell.ButtonPrimary, MouseLeftDown, MouseLeftUp, MouseLeftClick, MouseLeftDoubleClick},
		{tcell.ButtonMiddle, MouseMiddleDown, MouseMiddleUp, MouseMiddleClick, MouseMiddleDoubleClick},
		{tcell.ButtonSecondary, MouseRightDown, MouseRightUp, MouseRightClick, MouseRightDoubleClick},
	} {
		if buttonChanges&buttonEvent.button != 0 {
			if buttons&buttonEvent.button != 0 {
				fire(buttonEvent.down)
			} else {
				fire(buttonEvent.up) // A user override might set event to nil.
				if !clickMoved && event != nil {
					if a.lastMouseClick.Add(DoubleClickInterval).Before(time.Now()) {
						fire(buttonEvent.click)
						a.lastMouseClick = time.Now()
					} else {
						fire(buttonEvent.dclick)
						a.lastMouseClick = time.Time{} // reset
					}
				}
			}
		}
	}

	for _, wheelEvent := range []struct {
		button tcell.ButtonMask
		action MouseAction
	}{
		{tcell.WheelUp, MouseScrollUp},
		{tcell.WheelDown, MouseScrollDown},
		{tcell.WheelLeft, MouseScrollLeft},
		{tcell.WheelRight, MouseScrollRight}} {
		if buttons&wheelEvent.button != 0 {
			fire(wheelEvent.action)
		}
	}

	return consumed, isMouseDownAction
}

// Stop stops the application, causing Run() to return.
func (a *Application) Stop() {
	a.Lock()
	defer a.Unlock()
	screen := a.screen
	if screen == nil {
		return
	}
	a.screen = nil
	screen.Fini()
	a.screenReplacement <- nil
}

// Suspend temporarily suspends the application by exiting terminal UI mode and
// invoking the provided function "f". When "f" returns, terminal UI mode is
// entered again and the application resumes.
//
// A return value of true indicates that the application was suspended and "f"
// was called. If false is returned, the application was already suspended,
// terminal UI mode was not exited, and "f" was not called.
func (a *Application) Suspend(f func()) bool {
	a.RLock()
	screen := a.screen
	a.RUnlock()
	if screen == nil {
		return false // Screen has not yet been initialized.
	}

	// Enter suspended mode.
	if err := screen.Suspend(); err != nil {
		return false // Suspension failed.
	}

	// Wait for "f" to return.
	f()

	// If the screen object has changed in the meantime, we need to do more.
	a.RLock()
	defer a.RUnlock()
	if a.screen != screen {
		// Calling Stop() while in suspend mode currently still leads to a
		// panic, see https://github.com/gdamore/tcell/issues/440.
		screen.Fini()
		if a.screen == nil {
			return true // If stop was called (a.screen is nil), we're done already.
		}
	} else {
		// It hasn't changed. Resume.
		screen.Resume() // Not much we can do in case of an error.
	}

	// Continue application loop.
	return true
}

// Draw refreshes the screen (during the next update cycle). It calls the Draw()
// function of the application's root primitive and then syncs the screen
// buffer. It is almost never necessary to call this function. It can actually
// deadlock your application if you call it from the main thread (e.g. in a
// callback function of a widget). Please see
// https://github.com/rivo/tview/wiki/Concurrency for details.
func (a *Application) Draw() *Application {
	a.QueueUpdate(func() {
		a.draw()
	})
	return a
}

// ForceDraw refreshes the screen immediately. Use this function with caution as
// it may lead to race conditions with updates to primitives in other
// goroutines. It is always preferable to call [Application.Draw] instead.
// Never call this function from a goroutine.
//
// It is safe to call this function during queued updates and direct event
// handling.
func (a *Application) ForceDraw() *Application {
	return a.draw()
}

// draw actually does what Draw() promises to do.
func (a *Application) draw() *Application {
	a.Lock()
	defer a.Unlock()

	screen := a.screen
	root := a.root
	fullscreen := a.rootFullscreen
	before := a.beforeDraw
	after := a.afterDraw

	// Maybe we're not ready yet or not anymore.
	if screen == nil || root == nil {
		return a
	}

	// Resize if requested.
	if fullscreen { // root is not nil here.
		width, height := screen.Size()
		root.SetRect(0, 0, width, height)
	}

	// Clear screen to remove unwanted artifacts from the previous cycle.
	screen.Clear()

	// Call before handler if there is one.
	if before != nil {
		if before(screen) {
			screen.Show()
			return a
		}
	}

	// Draw all primitives.
	root.Draw(screen)

	// Call after handler if there is one.
	if after != nil {
		after(screen)
	}

	// Sync screen.
	screen.Show()

	return a
}

// Sync forces a full re-sync of the screen buffer with the actual screen during
// the next event cycle. This is useful for when the terminal screen is
// corrupted so you may want to offer your users a keyboard shortcut to refresh
// the screen.
func (a *Application) Sync() *Application {
	a.updates <- queuedUpdate{f: func() {
		a.RLock()
		screen := a.screen
		a.RUnlock()
		if screen == nil {
			return
		}
		screen.Sync()
	}}
	return a
}

// SetBeforeDrawFunc installs a callback function which is invoked just before
// the root primitive is drawn during screen updates. If the function returns
// true, drawing will not continue, i.e. the root primitive will not be drawn
// (and an after-draw-handler will not be called).
//
// Note that the screen is not cleared by the application. To clear the screen,
// you may call screen.Clear().
//
// Provide nil to uninstall the callback function.
func (a *Application) SetBeforeDrawFunc(handler func(screen tcell.Screen) bool) *Application {
	a.beforeDraw = handler
	return a
}

// GetBeforeDrawFunc returns the callback function installed with
// SetBeforeDrawFunc() or nil if none has been installed.
func (a *Application) GetBeforeDrawFunc() func(screen tcell.Screen) bool {
	return a.beforeDraw
}

// SetAfterDrawFunc installs a callback function which is invoked after the root
// primitive was drawn during screen updates.
//
// Provide nil to uninstall the callback function.
func (a *Application) SetAfterDrawFunc(handler func(screen tcell.Screen)) *Application {
	a.afterDraw = handler
	return a
}

// GetAfterDrawFunc returns the callback function installed with
// SetAfterDrawFunc() or nil if none has been installed.
func (a *Application) GetAfterDrawFunc() func(screen tcell.Screen) {
	return a.afterDraw
}

// SetRoot sets the root primitive for this application. If "fullscreen" is set
// to true, the root primitive's position will be changed to fill the screen.
//
// This function must be called at least once or nothing will be displayed when
// the application starts.
//
// It also calls SetFocus() on the primitive.
func (a *Application) SetRoot(root Primitive, fullscreen bool) *Application {
	a.Lock()
	a.root = root
	a.rootFullscreen = fullscreen
	if a.screen != nil {
		a.screen.Clear()
	}
	a.Unlock()

	a.SetFocus(root)

	return a
}

// ResizeToFullScreen resizes the given primitive such that it fills the entire
// screen.
func (a *Application) ResizeToFullScreen(p Primitive) *Application {
	a.RLock()
	width, height := a.screen.Size()
	a.RUnlock()
	p.SetRect(0, 0, width, height)
	return a
}

// SetFocus sets the focus to a new primitive. All key events will be directed
// down the hierarchy (starting at the root) until a primitive handles them,
// which per default goes towards the focused primitive.
//
// Blur() will be called on the previously focused primitive. Focus() will be
// called on the new primitive.
func (a *Application) SetFocus(p Primitive) *Application {
	a.Lock()
	if a.focus != nil {
		a.focus.Blur()
	}
	a.focus = p
	if a.screen != nil {
		a.screen.HideCursor()
	}
	a.Unlock()
	if p != nil {
		p.Focus(func(p Primitive) {
			a.SetFocus(p)
		})
	}

	return a
}

// GetFocus returns the primitive which has the current focus. If none has it,
// nil is returned.
func (a *Application) GetFocus() Primitive {
	a.RLock()
	defer a.RUnlock()
	return a.focus
}

// QueueUpdate is used to synchronize access to primitives from non-main
// goroutines. The provided function will be executed as part of the event loop
// and thus will not cause race conditions with other such update functions or
// the Draw() function.
//
// Note that Draw() is not implicitly called after the execution of f as that
// may not be desirable. You can call Draw() from f if the screen should be
// refreshed after each update. Alternatively, use QueueUpdateDraw() to follow
// up with an immediate refresh of the screen.
//
// This function returns after f has executed.
func (a *Application) QueueUpdate(f func()) *Application {
	ch := make(chan struct{})
	a.updates <- queuedUpdate{f: f, done: ch}
	<-ch
	return a
}

// QueueUpdateDraw works like QueueUpdate() except it refreshes the screen
// immediately after executing f.
func (a *Application) QueueUpdateDraw(f func()) *Application {
	a.QueueUpdate(func() {
		f()
		a.draw()
	})
	return a
}

// QueueEvent sends an event to the Application event loop.
//
// It is not recommended for event to be nil.
func (a *Application) QueueEvent(event tcell.Event) *Application {
	a.events <- event
	return a
}
