package ui

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func mar(day int) time.Time { return time.Date(2024, time.March, day, 0, 0, 0, 0, time.Local) }

func TestDatePickerInitialMonthAndValue(t *testing.T) {
	d := NewDatePicker(DatePickerValue(mar(15)))
	if !sameDay(d.Value(), mar(15)) {
		t.Fatalf("value = %v, want 2024-03-15", d.Value())
	}
	if d.visible.Month() != time.March || d.visible.Day() != 1 {
		t.Fatalf("visible should be first of March, got %v", d.visible)
	}
}

func TestDatePickerCellRoundTrip(t *testing.T) {
	d := NewDatePicker(DatePickerValue(mar(15)), DatePickerFirstWeekday(time.Sunday))
	// dateForCell and cellForDate must be inverses on the visible grid.
	for day := 1; day <= 31; day++ {
		date := mar(day)
		i := d.cellForDate(date)
		if i < 0 {
			t.Fatalf("day %d should be on the grid", day)
		}
		if !sameDay(d.dateForCell(i), date) {
			t.Fatalf("cell %d -> %v, want %v", i, d.dateForCell(i), date)
		}
	}
	// March 1, 2024 is a Friday → with Sunday first, offset 5.
	if got := d.offset(); got != 5 {
		t.Fatalf("offset for March 2024 (Sunday-first) = %d, want 5", got)
	}
}

func TestDatePickerKeyboardNavigation(t *testing.T) {
	d := NewDatePicker(DatePickerValue(mar(15)))
	d.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight})
	if !sameDay(d.Value(), mar(16)) {
		t.Fatalf("Right should go to the 16th, got %v", d.Value())
	}
	d.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyDown})
	if !sameDay(d.Value(), mar(23)) {
		t.Fatalf("Down should add a week (23rd), got %v", d.Value())
	}
	d.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyPageDown})
	if d.Value().Month() != time.April || d.Value().Day() != 23 {
		t.Fatalf("PageDown should go to April 23, got %v", d.Value())
	}
}

func TestDatePickerStepMonthKeepsSelection(t *testing.T) {
	d := NewDatePicker(DatePickerValue(mar(15)))
	d.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}}) // wheel down → next month
	if d.visible.Month() != time.April {
		t.Fatalf("wheel should advance the view to April, got %v", d.visible)
	}
	if !sameDay(d.Value(), mar(15)) {
		t.Fatalf("stepping the month must not change the selection, got %v", d.Value())
	}
}

func TestDatePickerOnChangeOnlyOnChange(t *testing.T) {
	d := NewDatePicker(DatePickerValue(mar(15)))
	calls := 0
	d.OnChange(func(time.Time) { calls++ })
	d.SetValue(mar(15)) // unchanged
	d.SetValue(mar(20)) // change
	if calls != 1 {
		t.Fatalf("OnChange should fire once, got %d", calls)
	}
}
