// Command widgets demonstrates the Label and Button widgets and pointer
// interaction: a counter driven by buttons, plus a toggle that enables and
// disables another button at runtime.
//
// Run with: go run ./examples/widgets
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — widgets"),
		ui.WithSize(560, 320),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Labels, buttons and runtime state changes"))

	count := 0
	status := ui.NewLabel("count: 0")
	root.Add(status)
	setStatus := func() { status.SetText(fmt.Sprintf("count: %d", count)) }

	// The button whose enabled state we toggle below.
	step := ui.NewButton("+1", ui.OnClick(func() {
		count++
		setStatus()
	}))

	controls := ui.NewContainer()
	controls.SetLayout(ui.HBox(10))
	controls.Add(step)
	controls.Add(ui.NewButton("-1", ui.OnClick(func() {
		count--
		setStatus()
	})))
	controls.Add(ui.NewButton("Reset", ui.OnClick(func() {
		count = 0
		setStatus()
	})))
	root.Add(controls, ui.Align(geom.AlignStart))

	// A second row: toggle the "+1" button's enabled state.
	enabled := true
	var toggle *ui.Button
	toggle = ui.NewButton("Disable +1", ui.OnClick(func() {
		enabled = !enabled
		step.SetEnabled(enabled)
		if enabled {
			toggle.SetText("Disable +1")
		} else {
			toggle.SetText("Enable +1")
		}
	}))
	root.Add(toggle, ui.Align(geom.AlignStart), ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
