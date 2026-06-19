package ui

import (
	"image/color"

	"github.com/kpfaulkner/uiframework/theme"
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
