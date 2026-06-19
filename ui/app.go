// Package ui is the public API of the framework: the App, windows, widgets,
// layouts, events and styling helpers that applications use. Application code
// depends only on this package (and geom/render/theme for value types); it
// never imports EBiten. The App drives the main loop through a render.Driver,
// keeping the graphics backend an internal detail.
package ui

import (
	ebitenbackend "github.com/kpfaulkner/uiframework/backend/ebiten"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/theme"
)

// App owns the main loop, the root of the widget tree and top-level
// configuration. Construct one with NewApp, give it content with SetContent,
// and start it with Run.
type App struct {
	driver render.Driver
	cfg    render.Config
	theme  theme.Theme

	root        Widget
	ctx         *treeContext
	needsLayout bool
	surfaceSize geom.Size

	// pointer dispatch state
	hovered     Widget // widget currently under the cursor
	pressTarget Widget // widget that received the active pointer-down (capture)
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

	a.ctx = &treeContext{
		requestLayout: func() { a.needsLayout = true },
		theme:         &a.theme,
	}
	return a
}

// Theme returns the app's active theme.
func (a *App) Theme() theme.Theme { return a.theme }

// SetContent installs w as the root of the widget tree. The root is sized to
// fill the surface. SetContent may be called before or after Run.
func (a *App) SetContent(w Widget) {
	a.root = w
	if w != nil {
		w.mount(nil, a.ctx)
		if a.surfaceSize.W > 0 {
			w.SetBounds(geom.Rect{W: a.surfaceSize.W, H: a.surfaceSize.H})
		}
	}
	a.needsLayout = true
}

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
	// Keep layout current before hit-testing, then dispatch pointer input.
	a.layoutIfNeeded()
	a.dispatchPointer(in)
	return nil
}

// dispatchPointer translates the frame's mouse input into pointer events and
// routes them to widgets. Hover enter/leave follow the widget under the cursor;
// a press captures its target so the matching release (and the click test) goes
// to the same widget. Full bubbling, focus and keyboard handling arrive in
// step 5.
func (a *App) dispatchPointer(in render.InputState) {
	if a.root == nil {
		return
	}
	pos := in.MousePos
	hit := hitTest(a.root, pos)

	if hit != a.hovered {
		if a.hovered != nil {
			ev := Event{Type: EventPointerLeave, Pos: pos}
			a.hovered.HandleEvent(&ev)
		}
		if hit != nil {
			ev := Event{Type: EventPointerEnter, Pos: pos}
			hit.HandleEvent(&ev)
		}
		a.hovered = hit
	}

	if in.MousePressed.Has(render.MouseLeft) && hit != nil {
		ev := Event{Type: EventPointerDown, Pos: pos, Button: render.MouseLeft}
		hit.HandleEvent(&ev)
		a.pressTarget = hit
	}

	if in.MouseReleased.Has(render.MouseLeft) && a.pressTarget != nil {
		ev := Event{Type: EventPointerUp, Pos: pos, Button: render.MouseLeft}
		a.pressTarget.HandleEvent(&ev)
		a.pressTarget = nil
	}
}

func (a *App) draw(c render.Canvas) {
	if a.root != nil && a.root.Visible() {
		a.root.Draw(c)
	}
}

func (a *App) resize(width, height int) {
	a.surfaceSize = geom.Size{W: float64(width), H: float64(height)}
	a.needsLayout = true
}

// layoutIfNeeded re-runs layout from the root when the tree has been marked
// dirty, sizing the root to the full surface first.
func (a *App) layoutIfNeeded() {
	if !a.needsLayout || a.root == nil {
		return
	}
	a.root.SetBounds(geom.Rect{W: a.surfaceSize.W, H: a.surfaceSize.H})
	a.root.Layout()
	a.needsLayout = false
}
