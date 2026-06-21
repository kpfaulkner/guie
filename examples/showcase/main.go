// Command showcase combines the third-wave widgets: a MenuBar across the top,
// a selectable List, and a DropdownCombo. Menu choices, list selection and the
// dropdown all report into a status line at the bottom. Click a menu title (or
// hover between titles once one is open); click the dropdown to open its popup;
// click outside or press Escape to dismiss popups.
//
// It also demonstrates keyboard accelerators (Ctrl/Cmd+N, Ctrl/Cmd+S,
// Ctrl/Cmd+Shift+S) and a context menu — right-click the list.
//
// Run with: go run ./examples/showcase
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — showcase"),
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
		ui.NewMenuItem("About", func() { say("guie showcase") }),
	)
	root.Add(bar)

	// Main content: a list on the left, a dropdown panel on the right.
	content := ui.NewContainer()
	content.SetLayout(ui.HBox(16))
	content.SetPadding(geom.UniformInsets(16))

	fruits := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape", "Kiwi", "Lemon", "Mango"}
	list := ui.NewList(fruits)
	list.OnSelect(func(i int) { say(fmt.Sprintf("List: %s", fruits[i])) })
	// Right-click the list for a context menu acting on the current selection.
	list.SetContextMenu(
		ui.NewMenuItem("Select first", func() { list.SetSelected(0) }),
		ui.NewMenuItem("Select last", func() { list.SetSelected(len(fruits) - 1) }),
		ui.NewMenuItem("Show selection", func() { say(fmt.Sprintf("Selected: %s", fruits[list.Selected()])) }),
	)
	content.Add(list, ui.Weight(1))

	right := ui.NewContainer()
	right.SetLayout(ui.VBox(10))
	right.Add(ui.NewLabel("Pick a colour:"))
	colours := []string{"Red", "Orange", "Yellow", "Green", "Blue", "Indigo", "Violet"}
	dd := ui.NewDropdown(colours, ui.DropdownPlaceholder("choose a colour"))
	dd.OnSelect(func(i int) { say("Dropdown: " + colours[i]) })
	right.Add(dd, ui.Align(geom.AlignStart))
	right.Add(ui.NewLabel("(right-click the list · try Ctrl/Cmd+N, +S, +Shift+S)"), ui.Weight(1))
	content.Add(right, ui.Weight(1))

	root.Add(content, ui.Weight(1))

	// Status line.
	statusBar := ui.NewContainer()
	statusBar.SetLayout(ui.VBox(0))
	statusBar.SetPadding(geom.UniformInsets(8))
	statusBar.Add(status)
	root.Add(statusBar)

	// Keyboard accelerators (Cmd on macOS, Ctrl elsewhere via ModPrimary).
	primary := func(extra ...render.Modifier) render.ModifierSet {
		m := render.ModPrimary
		for _, e := range extra {
			m |= e
		}
		return render.ModifierSet(m)
	}
	app.AddAccelerator(render.KeyN, primary(), func() { say("Accelerator: New (Ctrl/Cmd+N)") })
	app.AddAccelerator(render.KeyS, primary(), func() { say("Accelerator: Save (Ctrl/Cmd+S)") })
	app.AddAccelerator(render.KeyS, primary(render.ModShift), func() { say("Accelerator: Save As (Ctrl/Cmd+Shift+S)") })

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
