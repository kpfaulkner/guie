package guitest_test

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestDateFieldClickOpensCalendarPopup(t *testing.T) {
	h := guitest.New(320, 360)
	field := ui.NewDateField(
		ui.DateFieldValue(time.Date(2024, time.March, 10, 0, 0, 0, 0, time.Local)),
	)

	// Put the field at the top so the popup has room to open below it.
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(8))
	root.SetPadding(geom.UniformInsets(12))
	root.Add(field)
	root.Add(ui.NewLabel("below"))
	h.SetContent(root)
	h.Step() // lay out

	// Before opening, the calendar's month title is not drawn.
	if h.Frame().HasText("March 2024") {
		t.Fatal("calendar should not be visible before opening")
	}

	// Click the field (top-left area) to open the calendar popup.
	h.Click(40, 24)
	if !h.Step().HasText("March 2024") {
		t.Fatalf("clicking the field should open the calendar; texts = %v", h.Frame().Texts())
	}

	// Clicking far outside the popup dismisses it (non-modal).
	h.Click(300, 350)
	if h.Step().HasText("March 2024") {
		t.Fatal("clicking outside should dismiss the calendar popup")
	}
}
