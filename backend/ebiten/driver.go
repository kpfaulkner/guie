package ebitenbackend

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/uiframework/render"
)

// Driver implements render.Driver on top of EBiten. Construct one with New and
// hand it to the framework's App; the framework never touches EBiten directly.
type Driver struct{}

// New returns an EBiten-backed render.Driver.
func New() render.Driver { return &Driver{} }

// Run configures the host window and runs the EBiten game loop, invoking the
// framework's hooks each frame. It blocks until the window closes or a hook
// returns an error.
func (d *Driver) Run(cfg render.Config, hooks render.Hooks) error {
	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}
	ebiten.SetWindowSize(cfg.Width, cfg.Height)
	ebiten.SetWindowTitle(cfg.Title)
	if cfg.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}

	g := &game{
		hooks:  hooks,
		bg:     cfg.Background,
		canvas: newCanvas(),
		width:  cfg.Width,
		height: cfg.Height,
	}
	return ebiten.RunGame(g)
}

// game is the internal ebiten.Game that drives the framework's hooks. It is the
// bridge between EBiten's Update/Draw/Layout model and the render.Hooks model.
type game struct {
	hooks         render.Hooks
	bg            color.Color
	canvas        *canvas
	width, height int
	sized         bool
}

// Update polls input and forwards it to the framework's Update hook.
func (g *game) Update() error {
	if g.hooks.Update == nil {
		return nil
	}
	return g.hooks.Update(pollInput())
}

// Draw clears the surface to the background color and runs the Draw hook.
func (g *game) Draw(screen *ebiten.Image) {
	g.canvas.reset(screen)
	if g.bg != nil {
		screen.Fill(g.bg)
	}
	if g.hooks.Draw != nil {
		g.hooks.Draw(g.canvas)
	}
}

// Layout maps the host window size to the logical surface size (1:1 for now)
// and notifies the framework when the size changes, including once at startup.
func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if !g.sized || outsideWidth != g.width || outsideHeight != g.height {
		g.width, g.height = outsideWidth, outsideHeight
		g.sized = true
		if g.hooks.Resize != nil {
			g.hooks.Resize(outsideWidth, outsideHeight)
		}
	}
	return outsideWidth, outsideHeight
}
