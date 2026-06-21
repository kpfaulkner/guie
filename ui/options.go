package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
)

// AppOption configures an App during NewApp.
type AppOption func(*App)

// WithTitle sets the host window title.
func WithTitle(title string) AppOption {
	return func(a *App) { a.cfg.Title = title }
}

// WithSize sets the initial logical window size.
func WithSize(width, height int) AppOption {
	return func(a *App) { a.cfg.Width, a.cfg.Height = width, height }
}

// WithBackground sets the color the surface is cleared to each frame,
// overriding the theme background.
func WithBackground(c color.Color) AppOption {
	return func(a *App) { a.cfg.Background = c }
}

// WithResizable controls whether the host window can be resized.
func WithResizable(v bool) AppOption {
	return func(a *App) { a.cfg.Resizable = v }
}

// WithTheme sets the app's theme.
func WithTheme(t theme.Theme) AppOption {
	return func(a *App) { a.theme = t }
}

// WithClipboard supplies a custom clipboard (e.g. an OS-backed one). By default
// the app uses a simple in-process clipboard.
func WithClipboard(c render.Clipboard) AppOption {
	return func(a *App) { a.clipboard = c }
}

// WithDriver replaces the graphics/loop backend. By default the app uses the
// bundled EBiten driver; tests can inject a headless driver (see the guitest
// package) to drive the app frame by frame without a window or GPU.
func WithDriver(d render.Driver) AppOption {
	return func(a *App) {
		if d != nil {
			a.driver = d
		}
	}
}

// WithFont sets the default text face used by widgets that don't override it.
func WithFont(f render.FontFace) AppOption {
	return func(a *App) { a.theme.Font = f }
}

// WithFontSize sets the size of the default bundled font. It has no effect if a
// font face is supplied via WithFont or WithTheme.
func WithFontSize(size float64) AppOption {
	return func(a *App) { a.theme.FontSize = size }
}

// WithShadows enables or disables the soft drop shadows drawn under overlays
// (popups, menus, dialogs) and tooltips. Shadows are on by default.
func WithShadows(v bool) AppOption {
	return func(a *App) { a.shadows = v }
}
