// Command table demonstrates the Table widget: a header row over scrollable,
// selectable body rows with weighted columns. Click a row or use Up/Down while
// focused; scroll with the wheel when the rows overflow. Click a column header
// to sort by it (click again to reverse) - the "Years" column sorts numerically.
//
// Horizontal scrolling is on (SetHorizontalScroll): the Email and Location
// columns have a MinWidth, so when the window is too narrow for every column a
// bar appears along the bottom, and the header and rows scroll across together.
// Drag the bar, use a sideways wheel or Shift+wheel, or the Left/Right keys.
// Narrow the window to see it.
//
// Double-click a row, or press Enter on it, to activate it (Table.OnActivate).
//
// Run with: go run ./examples/table
package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie - table"),
		ui.WithSize(560, 360),
		ui.WithResizable(true),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))

	root.Add(ui.NewLabel("Employees (click to select, double-click to open; narrow the window to scroll across):"))

	status := ui.NewLabel("Selected: (none)")

	people := [][]string{
		{"Ada Lovelace", "Engineer", "7", "ada.lovelace@analytical-engine.example", "London"},
		{"Bob Ross", "Designer", "4", "bob.ross@happy-little-trees.example", "Daytona Beach"},
		{"Cy Young", "Manager", "12", "cy.young@baseball-hall-of-fame.example", "Cleveland"},
		{"Dot Matrix", "Engineer", "2", "dot.matrix@printers.example", "Austin"},
		{"Eve Online", "QA", "5", "eve.online@new-eden.example", "Reykjavik"},
		{"Frank Ocean", "Support", "3", "frank.ocean@channel-orange.example", "Long Beach"},
		{"Grace Hopper", "Architect", "9", "grace.hopper@cobol.example", "New York"},
		{"Hank Hill", "Sales", "6", "hank.hill@strickland-propane.example", "Arlen"},
		{"Ivy League", "Intern", "1", "ivy.league@university.example", "Cambridge"},
		{"Jack Black", "Engineer", "8", "jack.black@tenacious.example", "Santa Monica"},
	}

	// "Name" is twice as wide as Role and Years. "Years" sorts numerically. Email
	// and Location refuse to shrink below their MinWidth, which is what makes the
	// table scroll across when the window is narrower than all the columns.
	table := ui.NewTable([]ui.Column{
		{Title: "Name", Weight: 2},
		{Title: "Role", Weight: 1},
		{Title: "Years", Weight: 1, Less: func(a, b string) bool {
			ai, _ := strconv.Atoi(a)
			bi, _ := strconv.Atoi(b)
			return ai < bi
		}},
		{Title: "Email", Weight: 3, MinWidth: 260},
		{Title: "Location", Weight: 1, MinWidth: 140},
	})
	table.SetHorizontalScroll(true)

	// Read the selected row from the table (its order changes when sorted), not
	// from the original slice.
	table.OnSelect(func(i int) {
		r := table.Row(i)
		status.SetText(fmt.Sprintf("Selected: %s (%s)", r[0], r[1]))
	})
	table.OnActivate(func(i int) {
		r := table.Row(i)
		app.ShowToast("Opened "+r[0], ui.WithToastKind(ui.ToastSuccess))
	})
	table.SetStriped(true)
	table.SetRows(people)

	root.Add(table, ui.Weight(1))
	root.Add(status)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
