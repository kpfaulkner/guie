// Command datepicker demonstrates the DatePicker: an inline month calendar.
// Click a day to select it (dimmed days belong to the adjacent month), use the
// ‹ › header arrows or the mouse wheel to change months, or click the calendar
// and navigate with the arrow keys (PageUp/PageDown change months). The selected
// date is echoed below.
//
// Run with: go run ./examples/datepicker
package main

import (
	"log"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — date picker"),
		ui.WithSize(320, 320),
	)

	chosen := ui.NewLabel("Selected: (none)")

	cal := ui.NewDatePicker()
	cal.OnChange(func(d time.Time) {
		chosen.SetText("Selected: " + d.Format("Mon 2 Jan 2006"))
	})
	chosen.SetText("Selected: " + cal.Value().Format("Mon 2 Jan 2006"))

	// The popup variant: a compact field that drops the calendar down on click.
	field := ui.NewDateField(ui.DateFieldPlaceholder("Click to pick a date"))
	field.OnChange(func(d time.Time) {
		chosen.SetText("Selected: " + d.Format("Mon 2 Jan 2006"))
		cal.SetValue(d)
	})

	fieldRow := ui.NewContainer()
	fieldRow.SetLayout(ui.HBox(8))
	fieldRow.Add(ui.NewLabel("Popup field:"))
	fieldRow.Add(field)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(fieldRow)
	root.Add(ui.NewLabel("Inline calendar:"))
	root.Add(cal, ui.Weight(1), ui.Align(geom.AlignStretch))
	root.Add(chosen)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
