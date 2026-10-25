// Command scroll demonstrates ScrollView: a viewport over content taller than
// the window and wider than it. Scroll with the mouse wheel or by dragging the
// scrollbar thumb on the right edge; Shift+wheel (or the bar along the bottom)
// scrolls across, because horizontal scrolling is switched on. The content is a
// vertical list of checkboxes plus one very wide row of buttons, showing that
// widgets inside a scroll view stay interactive.
//
// Run with: go run ./examples/scroll
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — scroll"),
		ui.WithSize(440, 360),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))

	root.Add(ui.NewLabel("Wheel or drag the thumb to scroll; Shift+wheel or the bottom bar scrolls across:"))

	// Build a tall content column of checkboxes.
	list := ui.NewContainer()
	list.SetLayout(ui.VBox(6))
	for i := 1; i <= 40; i++ {
		list.Add(ui.NewCheckbox(fmt.Sprintf("Item %d", i)))
	}

	// One row far wider than the view: with horizontal scrolling on, the content keeps
	// its natural width and a second bar appears along the bottom.
	wide := ui.NewContainer()
	wide.SetLayout(ui.HBox(8))
	for i := 1; i <= 14; i++ {
		wide.Add(ui.NewButton(fmt.Sprintf("Wide button %d", i)))
	}
	list.Add(wide)

	scroller := ui.NewScrollView()
	scroller.SetHorizontalScroll(true)
	scroller.SetContent(list)
	root.Add(scroller, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
