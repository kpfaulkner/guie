// Command dialog demonstrates modal dialogs. Buttons open modal popups that dim
// the background, block input behind them, and are dismissed only by their own
// buttons or Escape (clicking the scrim does nothing).
//
// Run with: go run ./examples/dialog
package main

import (
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — dialog"),
		ui.WithSize(560, 320),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Modal dialogs (background is blocked while open)"))

	status := ui.NewLabel("Last action: (none)")
	root.Add(status)

	row := ui.NewContainer()
	row.SetLayout(ui.HBox(10))

	// A simple message dialog with a single OK button.
	showMsg := ui.NewButton("Show message")
	showMsg.OnClick(func() {
		app.ShowMessage("Hello", "This is a modal message dialog.")
	})
	row.Add(showMsg)

	// A confirm dialog with Cancel / Delete choices.
	del := ui.NewButton("Delete...")
	del.OnClick(func() {
		app.ShowMessage("Delete item?", "This cannot be undone.",
			ui.DialogButton{Label: "Cancel", OnClick: func() {
				status.SetText("Last action: cancelled")
			}},
			ui.DialogButton{Label: "Delete", OnClick: func() {
				status.SetText("Last action: deleted")
			}},
		)
	})
	row.Add(del)

	root.Add(row, ui.Align(geom.AlignStart), ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
