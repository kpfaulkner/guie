package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

// Step-4 demo: real Label and Button widgets, themed and interactive. Click
// "Click me" to increment the counter; "Reset" sets it back to zero; the third
// button is disabled. Hover and press change the button colors. Still no
// EBiten import in application code.
func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — step 4 (label + button)"),
		ui.WithSize(560, 320),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Step 4: Label + interactive Button"))

	status := ui.NewLabel("clicks: 0")
	root.Add(status)

	count := 0
	render := func() { status.SetText(fmt.Sprintf("clicks: %d", count)) }

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))

	row.Add(ui.NewButton("Click me", ui.OnClick(func() {
		count++
		render()
	})))
	row.Add(ui.NewButton("Reset", ui.OnClick(func() {
		count = 0
		render()
	})))

	disabled := ui.NewButton("Disabled")
	disabled.SetEnabled(false)
	row.Add(disabled)

	// Center the button row horizontally; it keeps its natural height.
	root.Add(row, ui.Align(geom.AlignCenter), ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
