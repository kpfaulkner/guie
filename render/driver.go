package render

import (
	"errors"
	"image"
	"image/color"

	"github.com/kpfaulkner/guie/geom"
)

// ErrTerminated may be returned from Hooks.Update to request a clean shutdown of
// the main loop. A Driver must treat it as a normal stop: Run returns nil, not
// this error.
var ErrTerminated = errors.New("render: terminated")

// Config describes the host window and loop parameters a Driver should set up.
type Config struct {
	// Title is the OS window title.
	Title string
	// Width and Height are the initial logical window size.
	Width, Height int
	// Background is the colour the surface is cleared to each frame. If nil, the
	// surface is not cleared by the driver.
	Background color.Color
	// Resizable allows the user to resize the host window.
	Resizable bool
	// Icon holds one or more window/taskbar icon images, ordered by preference
	// (typically the same icon at several sizes). The driver/OS picks the
	// best-matching size. If empty, the platform's default icon is used. These are
	// standard image.Image values, so applications set the icon without importing
	// any graphics backend.
	Icon []image.Image
}

// Hooks are the per-frame callbacks a Driver invokes while running the loop.
// The framework supplies these; the Driver owns when they fire.
type Hooks struct {
	// Update is called once per frame with the latest input. Returning a non-nil
	// error stops the loop and is propagated out of Driver.Run.
	Update func(in InputState) error
	// Draw is called once per frame to paint the surface.
	Draw func(c Canvas)
	// Resize is called when the logical surface size changes, including once at
	// startup. It is invoked before the first Update.
	Resize func(width, height int)
	// CloseRequested is called when the user asks the OS to close the window (the
	// title-bar close button, Alt+F4, the dock menu). Returning true lets the
	// close proceed and the loop stop; returning false vetoes it and the loop
	// keeps running, which is how an application prompts about unsaved work and
	// quits later on its own terms.
	//
	// A Driver must call it before that frame's Update, so no frame is processed
	// on the way out. A veto is only honoured while close interception is on (see
	// CloseInterceptor); without that capability the platform closes the window
	// itself and the hook is advisory.
	CloseRequested func() bool
}

// CloseInterceptor is an optional capability a Driver may implement to let the
// framework veto window closing. The framework type-asserts a Driver for it and
// toggles it as an application installs or clears its close handler; a Driver
// without it simply lets the platform close the window, and Hooks.CloseRequested
// cannot hold it back.
//
// It may be called before Run (while there is no window yet) and at any point
// afterwards, so implementations must tolerate both.
type CloseInterceptor interface {
	// SetCloseHandled asks the platform to stop closing the window by itself
	// (true) or to resume doing so (false). While handled, the Driver reports
	// close requests through Hooks.CloseRequested and stops the loop only if that
	// hook allows it.
	SetCloseHandled(handled bool)
}

// Driver runs the platform main loop and bridges it to the backend-neutral
// world. A Driver implementation is the only component permitted to import a
// concrete graphics library. The framework holds a Driver but never sees the
// backend's types.
type Driver interface {
	// Run sets up the host window per cfg and runs the main loop, invoking hooks
	// each frame. It blocks until the window is closed or a hook returns an
	// error.
	Run(cfg Config, hooks Hooks) error
}

// FrameScheduler is an optional capability a Driver may implement to let the
// framework choose when frames happen instead of presenting one every display
// refresh. The framework type-asserts a Driver for it; a Driver without it
// simply keeps presenting continuously, which is correct, just not free.
//
// It exists because presenting is what a frame costs. On macOS an ebiten window
// with an empty Draw burns roughly 14% of a core at 60Hz and 0.6% presenting
// only on demand, and the same window drawing 400 strings drops from 44% to
// 3.4% - the drawing is charged per frame, so not running the frame is what
// removes it.
//
// A Driver must start in continuous mode: the framework switches it off once
// the first frame is on screen, so a backend that needs frames to get a window
// up is not stopped before it has one.
type FrameScheduler interface {
	// SetContinuousFrames chooses between presenting every display refresh
	// (true) and presenting only on input or on RequestFrame (false). It is
	// called from the UI goroutine, both before Run and during the loop.
	SetContinuousFrames(on bool)
	// RequestFrame asks for one frame, for a change the Driver cannot see for
	// itself - work finishing on a background goroutine, say. It must be safe
	// to call from any goroutine, and before Run, where it may be dropped.
	RequestFrame()
}

// IMEController is an optional capability a Driver may implement to support
// input method editors. The framework type-asserts a Driver for it; when absent,
// IME degrades to committed-text-only (no inline preedit, no candidate-window
// positioning). A Driver that implements it should also report the preedit each
// frame via InputState.Composition.
type IMEController interface {
	// SetIMEEnabled turns IME on or off, called as an editable widget gains or
	// loses focus.
	SetIMEEnabled(on bool)
	// SetIMERect reports the focused widget's caret rectangle in absolute logical
	// pixels, so the OS can place the candidate window beside it.
	SetIMERect(r geom.Rect)
}
