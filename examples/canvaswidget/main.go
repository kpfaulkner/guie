// Command canvaswidget demonstrates ui.CanvasWidget: a custom-drawn, interactive
// surface built from callbacks alone, with no widget type of its own.
//
// Drag to draw. Click the canvas to give it keyboard focus, then press 1-4 to
// change the pen colour, Z to undo the last stroke and C to clear. The status
// line shows the pointer position relative to the canvas (via ToLocal).
//
// Finished strokes are baked into a RenderTarget and blitted each frame, so the
// per-frame cost stays flat however much has been drawn; only the stroke in
// progress is drawn live. The layer is transparent so the themed surface shows
// through, and it is rebuilt on resize, undo and clear.
//
// Run with: go run ./examples/canvaswidget
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// penWidth is the stroke thickness in logical pixels.
const penWidth = 3

// stroke is one pen-down..pen-up run of points, in canvas-local coordinates so
// it survives a window resize.
type stroke struct {
	col color.Color
	pts []geom.Point
}

// drawStroke paints s onto c, offset by origin (the canvas's top-left on c).
func drawStroke(c render.Canvas, s stroke, origin geom.Point) {
	at := func(p geom.Point) geom.Point { return geom.Point{X: origin.X + p.X, Y: origin.Y + p.Y} }
	for i, p := range s.pts {
		if i == 0 {
			c.FillCircle(at(p), penWidth/2.0, s.col)
			continue
		}
		c.DrawLine(at(s.pts[i-1]), at(p), s.col, penWidth)
	}
}

func main() {
	app := ui.NewApp(ui.WithTitle("guie - canvaswidget"), ui.WithSize(640, 440))
	th := app.Theme()

	pens := []color.Color{
		color.RGBA{R: 0xe0, G: 0x5a, B: 0x5a, A: 0xff},
		color.RGBA{R: 0x5a, G: 0xc0, B: 0x7a, A: 0xff},
		color.RGBA{R: 0x5a, G: 0x9a, B: 0xe0, A: 0xff},
		color.RGBA{R: 0xe0, G: 0xc0, B: 0x4a, A: 0xff},
	}
	var (
		strokes []stroke
		pen     = 0
		drawing bool // the last stroke is still being drawn
		status  = ui.NewLabel("move the pointer over the canvas")
		cw      *ui.CanvasWidget

		layer   render.RenderTarget // finished strokes; transparent background
		layerSz geom.Size           // size the layer was built for
		baked   int                 // strokes already painted into the layer
		stale   bool                // undo/clear: layer must be wiped and rebaked
	)

	// syncLayer brings the layer up to date with the finished strokes. Normally
	// that is just baking the stroke(s) completed since the last frame; a resize,
	// undo or clear rebuilds from scratch.
	syncLayer := func(b geom.Rect, done []stroke) {
		if b.W <= 0 || b.H <= 0 {
			return
		}
		if layer == nil || layerSz != b.Size() {
			if layer != nil {
				layer.Dispose()
			}
			layer, layerSz, baked, stale = ui.NewRenderTarget(int(b.W), int(b.H)), b.Size(), 0, false
		}
		if layer == nil {
			return
		}
		if stale {
			layer.Clear(color.Transparent)
			baked, stale = 0, false
		}
		if baked < len(done) {
			lc := layer.Canvas()
			for ; baked < len(done); baked++ {
				drawStroke(lc, done[baked], geom.Point{})
			}
		}
	}

	cw = ui.NewCanvasWidget(
		// Draw: absolute bounds, already clipped. Colours come from ColourOf so
		// the canvas follows a runtime theme change.
		func(c render.Canvas, b geom.Rect) {
			c.FillRect(b, cw.ColourOf(ui.RoleSurface))
			c.StrokeRect(b, cw.ColourOf(ui.RoleBorder), 1)

			done, live := strokes, stroke{}
			if drawing && len(strokes) > 0 {
				done, live = strokes[:len(strokes)-1], strokes[len(strokes)-1]
			}
			syncLayer(b, done)
			if layer != nil {
				c.DrawImage(layer, b) // everything finished, in one blit
			}
			drawStroke(c, live, b.Min()) // only the stroke in progress is live

			// A swatch of the current pen, so the key presses have visible effect.
			c.FillCircle(geom.Point{X: b.X + 16, Y: b.Y + 16}, 7, pens[pen])
			if len(strokes) == 0 && th.Font != nil {
				c.DrawText("drag to draw", geom.Point{X: b.X + 32, Y: b.Y + 8}, th.Font, cw.ColourOf(ui.RoleTextMuted))
			}
		},
		ui.CanvasMinSize(320, 200),
		ui.CanvasFocusable(),
		ui.CanvasOnEvent(func(ev *ui.Event) bool {
			switch ev.Type {
			case ui.EventPointerDown:
				if ev.Button != render.MouseLeft {
					return false
				}
				drawing = true
				strokes = append(strokes, stroke{col: pens[pen], pts: []geom.Point{cw.ToLocal(ev.Pos)}})
				return true
			case ui.EventPointerMove:
				p := cw.ToLocal(ev.Pos)
				status.SetText(fmt.Sprintf("pointer %.0f, %.0f   strokes %d", p.X, p.Y, len(strokes)))
				if drawing && len(strokes) > 0 {
					s := &strokes[len(strokes)-1]
					s.pts = append(s.pts, p)
					// Input already schedules a frame, so no Invalidate is needed here.
				}
				return true
			case ui.EventPointerUp:
				drawing = false // the stroke is now finished; next frame bakes it
				return true
			case ui.EventKeyDown:
				switch ev.Key {
				case render.Key1, render.Key2, render.Key3, render.Key4:
					pen = int(ev.Key - render.Key1)
				case render.KeyZ:
					if n := len(strokes); n > 0 && !drawing {
						strokes, stale = strokes[:n-1], true
					}
				case render.KeyC:
					if !drawing {
						strokes, stale = nil, true
					}
				default:
					return false // let Tab etc. through
				}
				return true
			}
			return false
		}),
	)
	cw.SetTooltip("Drag to draw. Focus, then 1-4 pen, Z undo, C clear.")

	clearBtn := ui.NewButton("Clear")
	clearBtn.OnClick(func() { strokes, stale = nil, true })

	bar := ui.NewContainer()
	bar.SetLayout(ui.HBox(12))
	bar.Add(clearBtn)
	bar.Add(status, ui.Weight(1))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(cw, ui.Weight(1), ui.Align(geom.AlignStretch))
	root.Add(bar, ui.Align(geom.AlignStretch))
	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
