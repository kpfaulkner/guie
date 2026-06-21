// Command widgets2 demonstrates the Stepper (numeric input) and the busy
// Spinner. Adjust the quantity with the stepper's up/down buttons, the arrow
// keys (click it first), or the mouse wheel; press "Work" to show the spinner
// for a couple of seconds.
//
// Run with: go run ./examples/widgets2
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — stepper & spinner"),
		ui.WithSize(420, 240),
	)

	qtyLabel := ui.NewLabel("Quantity: 1")
	qty := ui.NewStepper(ui.StepperRange(1, 99), ui.StepperValue(1))
	qty.OnChange(func(v float64) {
		qtyLabel.SetText(fmt.Sprintf("Quantity: %d", int(v)))
	})

	price := ui.NewStepper(
		ui.StepperRange(0, 1000),
		ui.StepperValue(9.5),
		ui.StepperStep(0.5),
		ui.StepperDecimals(2),
	)

	// Busy spinner, hidden until "Work" is pressed.
	spinner := ui.NewSpinner(ui.SpinnerSize(28))
	spinner.SetVisible(false)

	work := ui.NewButton("Work")
	remaining := 0.0
	work.OnClick(func() {
		remaining = 2.0 // seconds
		spinner.SetVisible(true)
		spinner.Start()
	})
	app.OnFrame(func(dt float64) {
		if remaining > 0 {
			remaining -= dt
			if remaining <= 0 {
				spinner.SetVisible(false)
			}
		}
	})

	stepperRow := ui.NewContainer()
	stepperRow.SetLayout(ui.HBox(12))
	stepperRow.Add(qty)
	stepperRow.Add(ui.NewLabel("Price ($):"))
	stepperRow.Add(price)

	workRow := ui.NewContainer()
	workRow.SetLayout(ui.HBox(12))
	workRow.Add(work)
	workRow.Add(spinner)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(18))
	root.Add(ui.NewLabel("Stepper (buttons, arrow keys, or wheel):"))
	root.Add(stepperRow)
	root.Add(qtyLabel)
	root.Add(workRow)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
