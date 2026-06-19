// Command images demonstrates displaying images and image buttons. Images are
// generated in memory as PNG bytes and loaded via ui.LoadImageBytes (the same
// path ui.LoadImage uses for files) — application code never touches the
// rendering backend. The Image widget scales a picture to fit; buttons can show
// an icon with or without a label.
//
// Run with: go run ./examples/images
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// makePNG builds a w×h image by calling f for each pixel and encodes it as PNG.
func makePNG(w, h int, f func(x, y int) color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, f(x, y))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func mustLoad(data []byte) render.Image {
	img, err := ui.LoadImageBytes(data)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — images"),
		ui.WithSize(560, 420),
	)

	// A 160x100 diagonal gradient "logo".
	logo := mustLoad(makePNG(160, 100, func(x, y int) color.Color {
		return color.RGBA{R: uint8(x * 255 / 160), G: uint8(y * 255 / 100), B: 0x80, A: 0xff}
	}))

	// A 20x20 round icon on a transparent background.
	icon := mustLoad(makePNG(20, 20, func(x, y int) color.Color {
		dx, dy := float64(x-10)+0.5, float64(y-10)+0.5
		if math.Hypot(dx, dy) <= 9 {
			return color.RGBA{R: 0x5d, G: 0x86, B: 0xc4, A: 0xff}
		}
		return color.RGBA{}
	}))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Image widget (scaled to fit) and image buttons:"))

	// The Image widget fills the remaining space, scaled with FitContain.
	pic := ui.NewImage(logo)
	root.Add(pic, ui.Weight(1))

	// Buttons: icon + label, and icon-only.
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	row.Add(ui.NewButton("Save", ui.ButtonImage(icon)))
	row.Add(ui.NewButton("Open", ui.ButtonImage(icon)))
	row.Add(ui.NewButton("", ui.ButtonImage(icon))) // icon-only
	root.Add(row, ui.Align(geom.AlignStart))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
