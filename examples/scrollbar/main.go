// Command scrollbar demonstrates the standalone ui.ScrollBar widget: it scrolls
// nothing by itself, so here a CanvasWidget draws a big grid "map" and two
// ScrollBars (one vertical, one horizontal) move the view over it.
//
// The pattern is the whole point: the owner tells each bar how much content there
// is and how much is visible (SetRange), the bar reports where the user scrolled
// to (OnScroll), and the owner draws accordingly. Drag a thumb, click the gutter
// beside it to page, or use the wheel (Shift+wheel or a sideways wheel scrolls
// across). Dragging the map itself pans it, and the bars follow (SetOffset).
//
// Run with: go run ./examples/scrollbar
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

const (
	worldW = 3000 // the size of the map being scrolled, in logical pixels
	worldH = 2000
	cell   = 100 // grid cell size
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie - scrollbar"),
		ui.WithSize(640, 440),
		ui.WithResizable(true),
	)
	th := app.Theme()

	var (
		offX, offY float64 // the top-left of the visible part of the map
		vbar       = ui.NewScrollBar(geom.Vertical)
		hbar       = ui.NewScrollBar(geom.Horizontal)
		status     = ui.NewLabel("")
		canvas     *ui.CanvasWidget
	)
	report := func() {
		status.SetText(fmt.Sprintf("view origin: %.0f, %.0f   of %d x %d", offX, offY, worldW, worldH))
	}

	// The bars are the source of truth for where the user scrolled to.
	vbar.OnScroll(func(o float64) { offY = o; report() })
	hbar.OnScroll(func(o float64) { offX = o; report() })

	var panning bool
	var lastX, lastY float64

	canvas = ui.NewCanvasWidget(
		func(c render.Canvas, b geom.Rect) {
			// Tell the bars how big the world and the view are. The bars clamp the
			// offsets to the new range (the window may have been resized), so read
			// them back.
			hbar.SetRange(worldW, b.W)
			vbar.SetRange(worldH, b.H)
			offX, offY = hbar.Offset(), vbar.Offset()

			c.FillRect(b, canvas.ColourOf(ui.RoleSurface))

			// Draw only the cells in view.
			first := func(off float64) int { return int(math.Floor(off / cell)) }
			for gy := first(offY); float64(gy*cell) < offY+b.H && gy*cell < worldH; gy++ {
				for gx := first(offX); float64(gx*cell) < offX+b.W && gx*cell < worldW; gx++ {
					r := geom.Rect{X: b.X + float64(gx*cell) - offX, Y: b.Y + float64(gy*cell) - offY, W: cell, H: cell}
					if (gx+gy)%2 == 0 {
						c.FillRect(r, canvas.ColourOf(ui.RoleBackground))
					}
					c.StrokeRect(r, canvas.ColourOf(ui.RoleBorder), 1)
					if th.Font != nil {
						c.DrawText(fmt.Sprintf("%d,%d", gx*cell, gy*cell), geom.Point{X: r.X + 6, Y: r.Y + 6}, th.Font, canvas.ColourOf(ui.RoleTextMuted))
					}
				}
			}
		},
		ui.CanvasMinSize(200, 120),
		ui.CanvasOnEvent(func(ev *ui.Event) bool {
			switch ev.Type {
			case ui.EventWheel:
				// Shift+wheel or a sideways wheel scrolls across, as in the built-in widgets.
				if ev.Wheel.X != 0 || ev.Modifiers.Has(render.ModShift) {
					d := ev.Wheel.X
					if d == 0 {
						d = ev.Wheel.Y
					}
					hbar.SetOffset(hbar.Offset() - d*32)
					offX = hbar.Offset()
				} else {
					vbar.SetOffset(vbar.Offset() - ev.Wheel.Y*32)
					offY = vbar.Offset()
				}
				report()
				return true
			case ui.EventPointerDown:
				if ev.Button == render.MouseLeft {
					panning, lastX, lastY = true, ev.Pos.X, ev.Pos.Y
					return true
				}
			case ui.EventPointerMove:
				if panning { // grab the map and drag it: the view moves the other way
					hbar.SetOffset(hbar.Offset() - (ev.Pos.X - lastX))
					vbar.SetOffset(vbar.Offset() - (ev.Pos.Y - lastY))
					offX, offY = hbar.Offset(), vbar.Offset()
					lastX, lastY = ev.Pos.X, ev.Pos.Y
					report()
					return true
				}
			case ui.EventPointerUp:
				panning = false
				return true
			}
			return false
		}),
	)
	report()

	// The map and the horizontal bar stack on the left; the vertical bar sits on the
	// right, with a square spacer under it so the two bars meet at a corner.
	left := ui.NewContainer()
	left.SetLayout(ui.VBox(0))
	left.Add(canvas, ui.Weight(1), ui.Align(geom.AlignStretch))
	left.Add(hbar, ui.Align(geom.AlignStretch))

	right := ui.NewContainer()
	right.SetLayout(ui.VBox(0))
	right.Add(vbar, ui.Weight(1), ui.Align(geom.AlignStretch))
	corner := ui.NewContainer()
	corner.SetPadding(geom.UniformInsets(6)) // 12 x 12, the bar thickness
	right.Add(corner)

	view := ui.NewContainer()
	view.SetLayout(ui.HBox(0))
	view.Add(left, ui.Weight(1), ui.Align(geom.AlignStretch))
	view.Add(right, ui.Align(geom.AlignStretch))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("A standalone ScrollBar pair scrolling a CanvasWidget (drag the map, the thumbs, or use the wheel):"))
	root.Add(view, ui.Weight(1), ui.Align(geom.AlignStretch))
	root.Add(status)
	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
