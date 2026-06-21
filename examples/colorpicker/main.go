// Command colorpicker demonstrates the ColorPicker: an HSV picker with a preview
// swatch (showing the hex value) over hue/saturation/value gradient sliders.
// Drag or click a track to change a channel; the chosen color is applied to a
// panel on the right so you can see it in context.
//
// Run with: go run ./examples/colorpicker
package main

import (
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — color picker"),
		ui.WithSize(440, 240),
	)

	// A panel whose background follows the picker.
	preview := ui.NewContainer()
	preview.SetBackground(color.NRGBA{R: 0x4F, G: 0x8A, B: 0xE0, A: 0xff})
	preview.SetCornerRadius(8)
	preview.SetLayout(ui.NewStack())
	preview.Add(ui.NewLabel("Preview"), ui.Align(geom.AlignCenter))

	picker := ui.NewColorPicker(ui.ColorPickerValue(color.NRGBA{R: 0x4F, G: 0x8A, B: 0xE0, A: 0xff}))
	picker.OnChange(func(c color.Color) {
		preview.SetBackground(c)
	})

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(16))
	row.SetPadding(geom.UniformInsets(16))
	row.Add(picker, ui.Align(geom.AlignStretch))
	row.Add(preview, ui.Weight(1), ui.Align(geom.AlignStretch))

	app.SetContent(row)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
