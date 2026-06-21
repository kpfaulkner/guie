package guitest_test

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestDatePickerClickSelectsDay(t *testing.T) {
	const w, h = 280, 240
	hn := guitest.New(w, h)
	dp := ui.NewDatePicker(
		ui.DatePickerValue(time.Date(2024, time.March, 10, 0, 0, 0, 0, time.Local)),
		ui.DatePickerFirstWeekday(time.Sunday),
	)
	got := time.Time{}
	dp.OnChange(func(d time.Time) { got = d })
	hn.SetContent(dp)
	hn.Step() // lay out so the grid fills the surface

	// Compute the cell for March 20, 2024 the same way the widget does, then
	// click its center. The grid is 8 rows x 7 cols inside a 6px padding; weeks
	// start at row 2.
	visible := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.Local)
	offset := int(visible.Weekday()) // Sunday-first
	idx := offset + (20 - 1)
	row, col := 2+idx/7, idx%7

	const pad = 6.0
	cellW := (w - 2*pad) / 7.0
	rowH := (h - 2*pad) / 8.0
	x := pad + float64(col)*cellW + cellW/2
	y := pad + float64(row)*rowH + rowH/2

	hn.Click(x, y)

	if dp.Value().Day() != 20 || dp.Value().Month() != time.March {
		t.Fatalf("clicking the cell for the 20th should select March 20, got %v", dp.Value())
	}
	if got.Day() != 20 {
		t.Fatalf("OnChange should report the 20th, got %v", got)
	}
}

func TestDatePickerHeaderNavigatesMonths(t *testing.T) {
	const w, h = 280, 240
	hn := guitest.New(w, h)
	dp := ui.NewDatePicker(ui.DatePickerValue(time.Date(2024, time.March, 10, 0, 0, 0, 0, time.Local)))
	hn.SetContent(dp)
	hn.Step()

	const pad = 6.0
	cellW := (w - 2*pad) / 7.0
	rowH := (h - 2*pad) / 8.0
	// The next-month arrow sits in the top-right header cell.
	hn.Click(pad+cellW*6+cellW/2, pad+rowH/2)

	frame := hn.Step()
	if !frame.HasText("April 2024") {
		t.Fatalf("clicking the right arrow should show April 2024; texts = %v", frame.Texts())
	}
	// Navigation must not change the selection.
	if dp.Value().Month() != time.March {
		t.Fatalf("month nav must not change selection, got %v", dp.Value())
	}
}
