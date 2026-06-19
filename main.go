package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/ui"
)

// textWidget is a minimal leaf widget used by the demo. It reports its size via
// the font so the layout engine can measure it. (Real Label/Button widgets
// arrive in step 4.)
type textWidget struct {
	ui.BaseWidget
	str   string
	font  render.FontFace
	color color.Color
}

func newText(s string, font render.FontFace, c color.Color) *textWidget {
	return &textWidget{BaseWidget: ui.NewBase(), str: s, font: font, color: c}
}

func (t *textWidget) MinSize() geom.Size { return t.font.Measure(t.str) }

func (t *textWidget) Draw(c render.Canvas) {
	b := t.Bounds()
	c.DrawText(t.str, geom.Point{X: b.X, Y: b.Y}, t.font, t.color)
}

// panel is a colored box with a single centered label, built from a Stack.
func panel(bg color.Color, label string, font render.FontFace, txt color.Color) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetLayout(ui.NewStack())
	c.Add(newText(label, font, txt), ui.Align(geom.AlignCenter))
	return c
}

// Step-3 demo: a VBox root holds a title, a weighted HBox row of two panels,
// and a 3-column grid of cells. Resize the window — everything reflows because
// layout re-runs on resize. Still no EBiten import.
func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — step 3 (layouts)"),
		ui.WithSize(720, 480),
	)
	th := app.Theme()
	pal := th.Palette

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))

	title := newText("Step 3: layout engine (Box / Grid / Stack)", th.Font, pal.Text)
	root.Add(title) // weight 0 → natural height, stretched across width

	// Weighted row: panelB is twice as wide as panelA.
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(12))
	row.Add(panel(pal.Surface, "HBox weight 1", th.Font, pal.Text), ui.Weight(1))
	row.Add(panel(pal.Primary, "HBox weight 2", th.Font, pal.OnPrimary), ui.Weight(2))
	root.Add(row, ui.Weight(2))

	// A 3-column grid of colored cells that fills the remaining height.
	grid := ui.NewContainer()
	grid.SetLayout(ui.NewGrid(3, 8))
	cellColors := []color.Color{
		color.RGBA{0x8a, 0x4a, 0x4a, 0xff}, color.RGBA{0x4a, 0x8a, 0x5a, 0xff}, color.RGBA{0x4a, 0x5a, 0x8a, 0xff},
		color.RGBA{0x8a, 0x7a, 0x4a, 0xff}, color.RGBA{0x6a, 0x4a, 0x8a, 0xff}, color.RGBA{0x4a, 0x8a, 0x8a, 0xff},
	}
	for i, cc := range cellColors {
		grid.Add(panel(cc, fmt.Sprintf("cell %d", i+1), th.Font, pal.Text))
	}
	root.Add(grid, ui.Weight(3))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
