package ebitenbackend

import (
	"errors"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// Driver implements render.Driver on top of EBiten. Construct one with New and
// hand it to the framework's App; the framework never touches EBiten directly.
type Driver struct{}

// New returns an EBiten-backed render.Driver.
func New() render.Driver { return &Driver{} }

// SetIMEEnabled and SetIMERect satisfy render.IMEController. EBiten (v2.9.9)
// exposes no API to toggle the IME or position the candidate window, and does
// not surface preedit text, so these are no-ops today: only committed IME text
// flows (via AppendInputChars in pollInput). The methods exist so the framework
// can light up IME support once a backend provides it, without API churn above.
func (d *Driver) SetIMEEnabled(on bool)  {}
func (d *Driver) SetIMERect(r geom.Rect) {}

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
	if len(cfg.Icon) > 0 {
		ebiten.SetWindowIcon(cfg.Icon)
	}
	if cfg.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}

	g := &game{
		hooks:  hooks,
		bg:     cfg.Background,
		canvas: newCanvas(),
		width:  cfg.Width,
		height: cfg.Height,
		scale:  1,
	}
	return ebiten.RunGame(g)
}

// game is the internal ebiten.Game that drives the framework's hooks. It is the
// bridge between EBiten's Update/Draw/Layout model and the render.Hooks model.
//
// width/height track the last reported logical size (device-independent
// pixels); scale is the current device scale factor. The framework works in
// logical pixels, while the offscreen surface is sized in physical pixels so
// rendering stays crisp on HiDPI displays — the canvas bridges the two.
type game struct {
	hooks         render.Hooks
	bg            color.Color
	canvas        *canvas
	width, height int
	scale         float64
	sized         bool
}

// Update polls input and forwards it to the framework's Update hook. A
// render.ErrTerminated result is mapped to EBiten's clean-termination sentinel
// so the loop stops and RunGame returns nil.
func (g *game) Update() error {
	if g.hooks.Update == nil {
		return nil
	}
	if err := g.hooks.Update(pollInput()); err != nil {
		if errors.Is(err, render.ErrTerminated) {
			return ebiten.Termination
		}
		return err
	}
	return nil
}

// Draw clears the surface to the background colour and runs the Draw hook. The
// canvas is bound with the current device scale so logical draw coordinates map
// to the physical-resolution surface.
func (g *game) Draw(screen *ebiten.Image) {
	g.canvas.reset(screen, g.scale)
	if g.bg != nil {
		screen.Fill(g.bg)
	}
	if g.hooks.Draw != nil {
		g.hooks.Draw(g.canvas)
	}
}

// layout records the device scale factor, notifies the framework of any change
// to the logical surface size (including once at startup), and returns the
// physical surface size. Sizing the surface to logical×scale makes EBiten map
// it 1:1 to the window's physical pixels, so rendering is crisp on HiDPI/Retina
// displays. The logical size reported to the framework stays device-independent.
func (g *game) layout(outsideWidth, outsideHeight float64) (float64, float64) {
	g.scale = ebiten.Monitor().DeviceScaleFactor()
	if g.scale <= 0 {
		g.scale = 1
	}
	w, h := int(outsideWidth), int(outsideHeight)
	if !g.sized || w != g.width || h != g.height {
		g.width, g.height = w, h
		g.sized = true
		if g.hooks.Resize != nil {
			g.hooks.Resize(w, h)
		}
	}
	return outsideWidth * g.scale, outsideHeight * g.scale
}

// LayoutF is the floating-point layout EBiten prefers when implemented, giving
// fractional device scale factors (e.g. 1.25× or 1.5×) full precision.
func (g *game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return g.layout(outsideWidth, outsideHeight)
}

// Layout satisfies ebiten.Game; EBiten calls LayoutF instead when present.
func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.layout(float64(outsideWidth), float64(outsideHeight))
	return int(w), int(h)
}
