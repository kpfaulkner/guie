// Command showcase combines the third-wave widgets: a MenuBar across the top,
// a selectable List, and a DropdownCombo. Menu choices, list selection and the
// dropdown all report into a status line at the bottom. Click a menu title (or
// hover between titles once one is open); click the dropdown to open its popup;
// click outside or press Escape to dismiss popups.
//
// Run with: go run ./examples/showcase
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — showcase"),
		ui.WithSize(720, 480),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))

	status := ui.NewLabel("Ready.")
	say := func(s string) { status.SetText(s) }

	// Menu bar flush across the top.
	bar := ui.NewMenuBar()
	bar.AddMenu("File",
		ui.NewMenuItem("New", func() { say("File ▸ New") }),
		ui.NewMenuItem("Open", func() { say("File ▸ Open") }),
		ui.NewMenuItem("Save", func() { say("File ▸ Save") }),
	)
	bar.AddMenu("Edit",
		ui.NewMenuItem("Cut", func() { say("Edit ▸ Cut") }),
		ui.NewMenuItem("Copy", func() { say("Edit ▸ Copy") }),
		ui.NewMenuItem("Paste", func() { say("Edit ▸ Paste") }),
	)
	bar.AddMenu("Help",
		ui.NewMenuItem("About", func() { say("uiframework showcase") }),
	)
	root.Add(bar)

	// Main content: a list on the left, a dropdown panel on the right.
	content := ui.NewContainer()
	content.SetLayout(ui.HBox(16))
	content.SetPadding(geom.UniformInsets(16))

	fruits := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape", "Kiwi", "Lemon", "Mango"}
	list := ui.NewList(fruits, ui.OnSelect(func(i int) {
		say(fmt.Sprintf("List: %s", fruits[i]))
	}))
	content.Add(list, ui.Weight(1))

	right := ui.NewContainer()
	right.SetLayout(ui.VBox(10))
	right.Add(ui.NewLabel("Pick a color:"))
	colors := []string{"Red", "Orange", "Yellow", "Green", "Blue", "Indigo", "Violet"}
	right.Add(ui.NewDropdown(colors,
		ui.DropdownPlaceholder("choose a color"),
		ui.OnSelectIndex(func(i int) { say("Dropdown: " + colors[i]) }),
	), ui.Align(geom.AlignStart))
	right.Add(ui.NewLabel("(click outside or Esc to close popups)"), ui.Weight(1))
	content.Add(right, ui.Weight(1))

	root.Add(content, ui.Weight(1))

	// Status line.
	statusBar := ui.NewContainer()
	statusBar.SetLayout(ui.VBox(0))
	statusBar.SetPadding(geom.UniformInsets(8))
	statusBar.Add(status)
	root.Add(statusBar)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
