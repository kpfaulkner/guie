// Command dragdrop demonstrates drag-and-drop: drag the labelled items between
// the two panels. Press an item and move past a few pixels to pick it up; a
// ghost follows the cursor (a snapshot of the item), the panel under the cursor
// highlights when it will accept the drop, and releasing moves the item there.
// Press Escape mid-drag to cancel.
//
// It shows the whole API: SetDragSource (with a payload), SetDropTarget/OnDrop
// to receive it, OnDragEnter/OnDragLeave for the highlight, and OnDragEnd on the
// source. The default ghost is used (a snapshot of the dragged widget).
//
// Run with: go run ./examples/dragdrop
package main

import (
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// item is a draggable labelled chip. It is a custom leaf widget that paints
// itself and acts as a drag source carrying its own widget identity as the
// payload, so a panel can re-parent it on drop.
type item struct {
	ui.BaseWidget
	label string
	col   color.Color
	font  render.FontFace
}

func newItem(label string, col color.Color, font render.FontFace) *item {
	it := &item{BaseWidget: ui.NewBase(), label: label, col: col, font: font}
	it.SetDragSource(func() *ui.DragData {
		return &ui.DragData{Type: "item", Value: ui.Widget(it)}
	})
	return it
}

func (it *item) MinSize() geom.Size { return geom.Size{W: 150, H: 36} }

func (it *item) Draw(c render.Canvas) {
	b := it.Bounds()
	c.FillRoundRect(b, 6, it.col)
	sz := c.MeasureText(it.label, it.font)
	c.DrawText(it.label,
		geom.Point{X: b.X + (b.W-sz.W)/2, Y: b.Y + (b.H-sz.H)/2},
		it.font, color.White)
}

// newPanel builds a drop-target panel that re-parents any "item" dropped on it
// and highlights its border while an acceptable item hovers over it.
func newPanel(th theme.Theme) *ui.Container {
	p := ui.NewContainer()
	p.SetLayout(ui.VBox(8))
	p.SetPadding(geom.UniformInsets(12))
	p.SetBackground(th.Palette.Surface)
	p.SetBorder(th.Palette.Border, 1)
	p.SetCornerRadius(8)

	p.SetDropTarget(func(d ui.DragData) bool { return d.Type == "item" })
	p.OnDragEnter(func(ui.DragData) { p.SetBorder(th.Palette.Accent, 2) })
	p.OnDragLeave(func() { p.SetBorder(th.Palette.Border, 1) })
	p.OnDrop(func(d ui.DragData, _ geom.Point) bool {
		p.SetBorder(th.Palette.Border, 1)
		w, ok := d.Value.(ui.Widget)
		if !ok {
			return false
		}
		// Move the item from its current panel to this one.
		if src, ok := w.Parent().(*ui.Container); ok && src != p {
			src.Remove(w)
			p.Add(w)
		}
		return true
	})
	return p
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — drag and drop"),
		ui.WithSize(560, 360),
	)
	th := app.Theme()

	left := newPanel(th)
	right := newPanel(th)

	palette := []color.Color{
		color.NRGBA{R: 0xE5, G: 0x6B, B: 0x6F, A: 0xff},
		color.NRGBA{R: 0x6B, G: 0xB5, B: 0xE5, A: 0xff},
		color.NRGBA{R: 0x7F, G: 0xC8, B: 0x7F, A: 0xff},
		color.NRGBA{R: 0xE5, G: 0xB5, B: 0x6B, A: 0xff},
	}
	names := []string{"Apple", "Sky", "Leaf", "Sand"}
	for i, n := range names {
		left.Add(newItem(n, palette[i], th.Font))
	}

	panels := ui.NewContainer()
	panels.SetLayout(ui.HBox(16))
	panels.Add(left, ui.Weight(1), ui.Align(geom.AlignStretch))
	panels.Add(right, ui.Weight(1), ui.Align(geom.AlignStretch))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("Drag the chips between the two panels (Esc cancels a drag):"))
	root.Add(panels, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
