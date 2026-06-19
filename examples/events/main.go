// Command events demonstrates the event system: keyboard focus traversal with
// Tab / Shift+Tab (the focused button shows an accent ring), activation with
// Space or Enter, and a global event-bus subscriber that observes every click
// in the UI.
//
// Run with: go run ./examples/events
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — events"),
		ui.WithSize(600, 320),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Tab / Shift+Tab to move focus; Space/Enter to activate."))

	picked := ui.NewLabel("picked: (none)")
	root.Add(picked)

	busLabel := ui.NewLabel("(bus) observed 0 clicks")
	root.Add(busLabel)

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	for _, name := range []string{"Red", "Green", "Blue"} {
		n := name
		b := ui.NewButton(n)
		b.OnClick(func() { picked.SetText("picked: " + n) })
		row.Add(b)
	}
	root.Add(row, ui.Align(geom.AlignCenter), ui.Weight(1))

	// Global listener: counts clicks anywhere, without per-widget wiring.
	clicks := 0
	app.Events().Subscribe(ui.EventClick, func(ui.Event) {
		clicks++
		busLabel.SetText(fmt.Sprintf("(bus) observed %d clicks", clicks))
	})

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
