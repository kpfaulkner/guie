// Command controls demonstrates the form widgets added in step 6: TextField,
// Checkbox, RadioButton/RadioGroup, Slider and ProgressBar. Tab between the
// focusable controls; type in the field; drag or arrow-key the slider to drive
// the progress bar.
//
// Run with: go run ./examples/controls
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

// labeledRow puts a fixed-width caption to the left of a control.
func labeledRow(caption string, control ui.Widget) *ui.Container {
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))
	cap := ui.NewLabel(caption)
	row.Add(cap)
	row.Add(control, ui.Weight(1))
	return row
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — controls"),
		ui.WithSize(560, 420),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Form controls (Tab to move focus)"))

	// Text field with a live echo label.
	echo := ui.NewLabel("you typed: (nothing yet)")
	field := ui.NewTextField(ui.Placeholder("type here..."))
	field.OnChange(func(s string) { echo.SetText("you typed: " + s) })
	root.Add(labeledRow("Name:", field))
	root.Add(echo)

	// Checkbox.
	check := ui.NewCheckbox("Enable feature")
	check.OnChange(func(v bool) {
		echo.SetText(fmt.Sprintf("feature enabled: %v", v))
	})
	root.Add(check)

	// Radio group.
	pick := ui.NewLabel("size: (none)")
	group := ui.NewRadioGroup()
	sizes := []string{"Small", "Medium", "Large"}
	group.OnChange(func(i int) { pick.SetText("size: " + sizes[i]) })
	radios := ui.NewContainer()
	radios.SetLayout(ui.HBox(16))
	for _, s := range sizes {
		radios.Add(ui.NewRadioButton(s, group))
	}
	root.Add(radios)
	root.Add(pick)

	// Slider driving a progress bar.
	progress := ui.NewProgressBar(0.3)
	valLabel := ui.NewLabel("value: 0.30")
	slider := ui.NewSlider(ui.SliderValue(0.3))
	slider.OnChange(func(v float64) {
		progress.SetValue(v)
		valLabel.SetText(fmt.Sprintf("value: %.2f", v))
	})
	root.Add(labeledRow("Level:", slider))
	root.Add(progress)
	root.Add(valLabel, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
