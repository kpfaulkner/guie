package main

import (
	"image/color"
	"log"

	ebitenbackend "github.com/kpfaulkner/uiframework/backend/ebiten"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/ui"
)

// Step-1 demo: proves the backend seam and the framework-owned loop end to end.
// Note there is no EBiten import here — the app talks only to ui/geom/render.
func main() {
	font := ebitenbackend.DefaultFont(18)

	white := color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
	blue := color.RGBA{R: 0x4a, G: 0x6f, B: 0xa5, A: 0xff}
	accent := color.RGBA{R: 0x5d, G: 0x86, B: 0xc4, A: 0xff}

	app := ui.NewApp(
		ui.WithTitle("uiframework — step 1"),
		ui.WithSize(640, 400),
		ui.WithRootDraw(func(c render.Canvas) {
			panel := geom.Rect{X: 40, Y: 40, W: 240, H: 120}
			c.FillRect(panel, blue)
			c.StrokeRect(panel, white, 2)
			c.DrawText("Hello from the Canvas", geom.Point{X: 56, Y: 72}, font, white)

			c.DrawLine(geom.Point{X: 40, Y: 200}, geom.Point{X: 600, Y: 200}, accent, 3)
			c.DrawText("Step 1: backend seam + framework-owned loop", geom.Point{X: 40, Y: 220}, font, white)
		}),
	)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
