// Command splitter demonstrates SplitPane: draggable dividers that resize
// adjacent panes. The layout is a horizontal split (a list on the left, content
// on the right), where the right side is itself a vertical split. Drag any
// divider to resize; it highlights on hover.
//
// Run with: go run ./examples/splitter
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func panel(bg color.Color, heading string) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetLayout(ui.NewStack())
	c.Add(ui.NewLabel(heading), ui.Align(geom.AlignCenter))
	return c
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — splitter"),
		ui.WithSize(640, 420),
	)

	items := make([]string, 20)
	for i := range items {
		items[i] = fmt.Sprintf("Item %d", i+1)
	}
	list := ui.NewList(items)

	// Right side: a vertical split of two panels.
	right := ui.VSplit(
		panel(color.RGBA{0x2b, 0x2b, 0x3a, 0xff}, "Top pane"),
		panel(color.RGBA{0x24, 0x30, 0x2c, 0xff}, "Bottom pane"),
		ui.SplitRatio(0.8),
	)

	// Outer: list on the left (30%), the nested split on the right.
	split := ui.HSplit(list, right, ui.SplitRatio(0.2), ui.SplitMinSizes(80, 120))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.SetPadding(geom.UniformInsets(10))
	root.Add(split, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
