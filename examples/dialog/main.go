// Command dialog demonstrates modal dialogs. Buttons open modal popups that dim
// the background, block input behind them, and are dismissed only by their own
// buttons or Escape (clicking the scrim does nothing).
//
// Run with: go run ./examples/dialog
package main

import (
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — dialog"),
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

	// A confirm dialog. ShowConfirm reports the choice once: true for Delete,
	// false for Cancel, Escape or any other close. ConfirmDefaultCancel gives
	// Cancel the initial focus so a stray Enter is safe for a destructive action.
	del := ui.NewButton("Delete...")
	del.OnClick(func() {
		app.ShowConfirm("Delete item?", "This cannot be undone.", func(ok bool) {
			if ok {
				status.SetText("Last action: deleted")
			} else {
				status.SetText("Last action: cancelled")
			}
		}, ui.ConfirmLabels("Delete", "Cancel"), ui.ConfirmDefaultCancel())
	})
	row.Add(del)

	root.Add(row, ui.Align(geom.AlignStart), ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
