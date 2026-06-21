package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestTableHeaderClickSortsThroughHarness(t *testing.T) {
	const w, h = 240, 160
	hn := guitest.New(w, h)
	tbl := ui.NewTable([]ui.Column{{Title: "Name"}, {Title: "Role"}})
	tbl.SetRows([][]string{
		{"Cy", "Manager"},
		{"Ada", "Engineer"},
		{"Bob", "Designer"},
	})
	hn.SetContent(tbl)
	hn.Step() // lay out

	// Click the "Name" header (left column, header row at the very top).
	hn.Click(40, 8)

	if c, asc := tbl.SortColumn(); c != 0 || !asc {
		t.Fatalf("header click should sort col 0 ascending, got (%d,%v)", c, asc)
	}

	// The first body row should now be the alphabetically-first name. The header
	// "Name" is drawn first, so the first data cell text after it is the top row.
	texts := hn.Frame().Texts()
	// Find the first occurrence of a name cell after the headers.
	var firstName string
	for _, s := range texts {
		if s == "Ada" || s == "Bob" || s == "Cy" {
			firstName = s
			break
		}
	}
	if firstName != "Ada" {
		t.Fatalf("ascending sort should put Ada first; drawn texts = %v", texts)
	}

	// Clicking again flips to descending.
	hn.Click(40, 8)
	if _, asc := tbl.SortColumn(); asc {
		t.Fatal("second header click should sort descending")
	}
}
