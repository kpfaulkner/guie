// Package clipboard provides an OS-backed render.Clipboard so guie text
// widgets can exchange text with other applications via the system clipboard.
//
// It is deliberately a separate, opt-in package: importing it pulls in a
// platform clipboard dependency, keeping the core ui/render packages
// dependency-free and cross-platform. Wire it into an app with ui.WithClipboard:
//
//	cb, err := clipboard.New()
//	if err != nil {
//	    log.Fatal(err) // OS clipboard unavailable on this platform/environment
//	}
//	app := ui.NewApp(ui.WithClipboard(cb))
//
// Platform notes: CGo-free on Windows; on macOS and Linux it relies on the same
// CGo/X11 toolchain the GUI backend already requires (no external binaries such
// as xclip are needed).
package clipboard

import (
	xclip "golang.design/x/clipboard"

	"github.com/kpfaulkner/guie/render"
)

// osClipboard is an OS-backed render.Clipboard.
type osClipboard struct{}

// New initialises OS clipboard access and returns a render.Clipboard. It
// returns an error if the platform clipboard is unavailable (for example a
// headless environment with no display server), in which case the caller can
// fall back to the default in-process clipboard by not passing ui.WithClipboard.
func New() (render.Clipboard, error) {
	if err := xclip.Init(); err != nil {
		return nil, err
	}
	return osClipboard{}, nil
}

// ReadText returns the current OS clipboard text (empty if none/non-text).
func (osClipboard) ReadText() string {
	return string(xclip.Read(xclip.FmtText))
}

// WriteText replaces the OS clipboard text.
func (osClipboard) WriteText(s string) {
	xclip.Write(xclip.FmtText, []byte(s))
}
