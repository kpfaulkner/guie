package ui

import (
	"testing"
	"time"
)

func TestDateFieldDefaultsToPlaceholder(t *testing.T) {
	f := NewDateField(DateFieldPlaceholder("pick a day"))
	if _, ok := f.Value(); ok {
		t.Fatal("a new field should have no value")
	}
	if f.label() != "pick a day" {
		t.Fatalf("label should be the placeholder, got %q", f.label())
	}
}

func TestDateFieldSetValueFormatsAndFires(t *testing.T) {
	f := NewDateField(DateFieldFormat("2006-01-02"))
	calls := 0
	f.OnChange(func(time.Time) { calls++ })

	f.SetValue(time.Date(2024, time.March, 15, 9, 30, 0, 0, time.Local))
	if v, ok := f.Value(); !ok || v.Day() != 15 {
		t.Fatalf("value should be set to the 15th, got %v ok=%v", v, ok)
	}
	if f.label() != "2024-03-15" {
		t.Fatalf("label should use the format, got %q", f.label())
	}
	f.SetValue(time.Date(2024, time.March, 15, 0, 0, 0, 0, time.Local)) // same day
	if calls != 1 {
		t.Fatalf("OnChange should fire once for a real change, got %d", calls)
	}
}

func TestDateFieldOpensPopupAndPicksDate(t *testing.T) {
	app := NewApp()
	f := NewDateField()
	app.SetContent(f)

	f.openCalendar()
	if !f.open || len(app.overlays) != 1 {
		t.Fatalf("openCalendar should push a popup; open=%v overlays=%d", f.open, len(app.overlays))
	}

	// The popup content is the calendar; selecting a day there should update the
	// field and close the popup (the calendar's OnChange wiring).
	cal, ok := f.popup.content.(*DatePicker)
	if !ok {
		t.Fatalf("popup content should be a *DatePicker, got %T", f.popup.content)
	}
	cal.SetValue(time.Date(2030, time.January, 2, 0, 0, 0, 0, time.Local))

	if v, ok := f.Value(); !ok || v.Year() != 2030 || v.Day() != 2 {
		t.Fatalf("picking a date should set the field value, got %v ok=%v", v, ok)
	}
	if f.open || len(app.overlays) != 0 {
		t.Fatalf("picking a date should close the popup; open=%v overlays=%d", f.open, len(app.overlays))
	}
}

func TestDateFieldToggleClosesPopup(t *testing.T) {
	app := NewApp()
	f := NewDateField()
	app.SetContent(f)

	f.toggle() // open
	if !f.open {
		t.Fatal("first toggle should open")
	}
	f.toggle() // close
	if f.open || len(app.overlays) != 0 {
		t.Fatalf("second toggle should close; open=%v overlays=%d", f.open, len(app.overlays))
	}
}
