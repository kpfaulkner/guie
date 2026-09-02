package ui

import (
	"sync"

	osclipboard "github.com/kpfaulkner/guie/clipboard"
	"github.com/kpfaulkner/guie/render"
)

// memClipboard is a plain in-process Clipboard: text copied in a guie widget can
// be pasted into another guie widget, and nothing crosses the process boundary.
// It is the fallback when the OS clipboard is unavailable, and what tests use.
type memClipboard struct{ text string }

func (c *memClipboard) ReadText() string   { return c.text }
func (c *memClipboard) WriteText(s string) { c.text = s }

// defaultClipboard is the clipboard an App installs when the caller passes no
// WithClipboard. It is the OS clipboard, so Ctrl+V pastes text copied from a
// browser or an editor and Ctrl+C is visible to other applications.
//
// Resolution is lazy and happens on the first copy or paste, not at NewApp:
// initialising the platform clipboard can fail (a headless machine with no
// display server, a browser that withholds clipboard access) and can be
// comparatively expensive, and an app that never touches text should pay
// neither cost. A failure is not fatal - the in-process clipboard takes over,
// which is exactly the behaviour guie had before, so copy and paste keep
// working within the app.
type defaultClipboard struct {
	once sync.Once
	os   render.Clipboard
	mem  memClipboard
}

// resolve returns the OS clipboard if it could be initialised, else the
// in-process fallback.
func (d *defaultClipboard) resolve() render.Clipboard {
	d.once.Do(func() { d.os = tryOSClipboard() })
	if d.os != nil {
		return d.os
	}
	return &d.mem
}

func (d *defaultClipboard) ReadText() string   { return d.resolve().ReadText() }
func (d *defaultClipboard) WriteText(s string) { d.resolve().WriteText(s) }

// tryOSClipboard initialises the platform clipboard, returning nil if it is
// unavailable. The platform library panics rather than erroring on some
// platforms, so a panic is treated as unavailable too.
//
// It is a var so tests can inject an unavailable platform clipboard and
// exercise the fallback for real.
var tryOSClipboard = func() (cb render.Clipboard) {
	defer func() {
		if recover() != nil {
			cb = nil
		}
	}()
	c, err := osclipboard.New()
	if err != nil {
		return nil
	}
	return c
}
