// Command tooltips demonstrates hover tooltips: rest the pointer on a widget for
// about half a second and a hint appears near the cursor. Moving the pointer or
// clicking hides it. Tooltips are set with SetTooltip and work on any widget.
//
// Run with: go run ./examples/tooltips
package main

import (
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — tooltips"),
		ui.WithSize(520, 260),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Hover over a control and wait ~0.5s for its tooltip."))

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))

	save := ui.NewButton("Save")
	save.SetTooltip("Save the current document (Ctrl+S)")
	row.Add(save)

	open := ui.NewButton("Open")
	open.SetTooltip("Open an existing file")
	row.Add(open)

	del := ui.NewButton("Delete")
	del.SetTooltip("Permanently delete the selection")
	row.Add(del)

	root.Add(row, ui.Align(geom.AlignStart))

	check := ui.NewCheckbox("Enable feature")
	check.SetTooltip("Toggles the experimental feature")
	root.Add(check)

	field := ui.NewTextField(ui.Placeholder("name"))
	field.SetTooltip("Your full name, as it should appear")
	root.Add(field, ui.Align(geom.AlignStart), ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
