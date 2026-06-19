package render

import (
	"errors"
	"image/color"
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
	// Background is the color the surface is cleared to each frame. If nil, the
	// surface is not cleared by the driver.
	Background color.Color
	// Resizable allows the user to resize the host window.
	Resizable bool
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
