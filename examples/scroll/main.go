// Command scroll demonstrates ScrollView: a viewport over content taller than
// the window. Scroll with the mouse wheel or by dragging the scrollbar thumb on
// the right edge. The content is a vertical list of checkboxes, showing that
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

	root.Add(ui.NewLabel("Wheel or drag the thumb to scroll:"))

	// Build a tall content column of checkboxes.
	list := ui.NewContainer()
	list.SetLayout(ui.VBox(6))
	for i := 1; i <= 40; i++ {
		list.Add(ui.NewCheckbox(fmt.Sprintf("Item %d", i)))
	}

	scroller := ui.NewScrollView()
	scroller.SetContent(list)
	root.Add(scroller, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
