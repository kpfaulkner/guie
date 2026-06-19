// Package ui is the public API of the framework: the App, windows, widgets,
// layouts, events and styling helpers that applications use. Application code
// depends only on this package (and geom/render/theme for value types); it
// never imports EBiten. The App drives the main loop through a render.Driver,
// keeping the graphics backend an internal detail.
package ui

import (
	ebitenbackend "github.com/kpfaulkner/uiframework/backend/ebiten"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/theme"
)

// App owns the main loop and top-level configuration. Construct one with
// NewApp, configure it with options, and start it with Run.
type App struct {
	driver render.Driver
	cfg    render.Config
	theme  theme.Theme

	// rootDraw and rootUpdate are temporary step-1 scaffolding: until the
	// retained widget tree lands (step 2), they expose the frame canvas and
	// per-frame input directly. They will be replaced by SetContent.
	rootDraw   func(render.Canvas)
	rootUpdate func(render.InputState)
}

// NewApp creates an App with default configuration, then applies opts.
func NewApp(opts ...AppOption) *App {
	a := &App{
		driver: ebitenbackend.New(),
		theme:  theme.Default(),
		cfg: render.Config{
			Title:     "uiframework",
			Width:     800,
			Height:    600,
			Resizable: true,
		},
	}
	for _, o := range opts {
		o(a)
	}

	// Default the theme font and the clear color from the theme if unset.
	if a.theme.Font == nil {
		a.theme.Font = ebitenbackend.DefaultFont(a.theme.FontSize)
	}
	if a.cfg.Background == nil {
		a.cfg.Background = a.theme.Palette.Background
	}
	return a
}

// Theme returns the app's active theme.
func (a *App) Theme() theme.Theme { return a.theme }

// Run starts the main loop. It blocks until the window is closed or an error
// occurs.
func (a *App) Run() error {
	return a.driver.Run(a.cfg, render.Hooks{
		Update: a.update,
		Draw:   a.draw,
		Resize: a.resize,
	})
}

func (a *App) update(in render.InputState) error {
	if a.rootUpdate != nil {
		a.rootUpdate(in)
	}
	return nil
}

func (a *App) draw(c render.Canvas) {
	if a.rootDraw != nil {
		a.rootDraw(c)
	}
}

func (a *App) resize(width, height int) {
	// Step 2 will re-layout the widget tree against the new surface size.
}
