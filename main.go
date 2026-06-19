package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

// Step-5 demo: the full event system. Click or use Tab / Shift+Tab to move
// focus between the buttons (focused button shows an accent ring), then press
// Space or Enter to activate the focused button. A global event-bus subscriber
// watches every click and reports the running total. No EBiten import here.
func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — step 5 (events)"),
		ui.WithSize(600, 340),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Step 5: bubbling, focus, keyboard, event bus"))
	root.Add(ui.NewLabel("Tab / Shift+Tab to focus, Space/Enter to activate."))

	status := ui.NewLabel("clicks: 0")
	root.Add(status)

	busLabel := ui.NewLabel("(bus) no events yet")
	root.Add(busLabel)

	count := 0
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	row.Add(ui.NewButton("Increment", ui.OnClick(func() {
		count++
		status.SetText(fmt.Sprintf("clicks: %d", count))
	})))
	row.Add(ui.NewButton("Reset", ui.OnClick(func() {
		count = 0
		status.SetText("clicks: 0")
	})))
	disabled := ui.NewButton("Disabled")
	disabled.SetEnabled(false)
	row.Add(disabled)
	root.Add(row, ui.Align(geom.AlignCenter), ui.Weight(1))

	// Global listener: fires for every click anywhere in the UI.
	busClicks := 0
	app.Events().Subscribe(ui.EventClick, func(ui.Event) {
		busClicks++
		busLabel.SetText(fmt.Sprintf("(bus) observed %d click(s)", busClicks))
	})

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
