// Command table demonstrates the Table widget: a header row over scrollable,
// selectable body rows with weighted columns. Click a row or use Up/Down while
// focused; scroll with the wheel when the rows overflow.
//
// Run with: go run ./examples/table
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — table"),
		ui.WithSize(560, 360),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))

	root.Add(ui.NewLabel("Employees (click a row):"))

	status := ui.NewLabel("Selected: (none)")

	people := [][]string{
		{"Ada Lovelace", "Engineer", "7"},
		{"Bob Ross", "Designer", "4"},
		{"Cy Young", "Manager", "12"},
		{"Dot Matrix", "Engineer", "2"},
		{"Eve Online", "QA", "5"},
		{"Frank Ocean", "Support", "3"},
		{"Grace Hopper", "Architect", "9"},
		{"Hank Hill", "Sales", "6"},
		{"Ivy League", "Intern", "1"},
		{"Jack Black", "Engineer", "8"},
	}

	// "Name" is twice as wide as the other two columns.
	table := ui.NewTable(
		[]ui.Column{{Title: "Name", Weight: 2}, {Title: "Role", Weight: 1}, {Title: "Years", Weight: 1}},
	)
	table.OnSelect(func(i int) {
		status.SetText(fmt.Sprintf("Selected: %s (%s)", people[i][0], people[i][1]))
	})
	table.SetRows(people)

	root.Add(table, ui.Weight(1))
	root.Add(status)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
