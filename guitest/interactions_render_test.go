package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestTabsSelectSwitchesContent(t *testing.T) {
	h := guitest.New(300, 200)
	tabs := ui.NewTabContainer()
	tabs.AddTab("First", ui.NewLabel("content-one"))
	tabs.AddTab("Second", ui.NewLabel("content-two"))
	h.SetContent(tabs)

	rec := h.Step()
	// Both titles are always drawn; only the active tab's content shows.
	if !rec.HasText("First") || !rec.HasText("Second") {
		t.Error("both tab titles should be drawn")
	}
	if !rec.HasText("content-one") {
		t.Error("first tab's content should be visible initially")
	}
	if rec.HasText("content-two") {
		t.Error("second tab's content should be hidden initially")
	}

	tabs.Select(1)
	rec = h.Step()
	if !rec.HasText("content-two") || rec.HasText("content-one") {
		t.Error("after Select(1) the second tab's content should show")
	}
	if tabs.Selected() != 1 {
		t.Errorf("Selected(): got %d, want 1", tabs.Selected())
	}
}

func TestToastRendersMessage(t *testing.T) {
	h := guitest.New(300, 200)
	h.SetContent(ui.NewLabel("backdrop"))
	h.App.ShowToast("Saved!", ui.WithToastKind(ui.ToastSuccess))

	if !h.Step().TextContaining("Saved!") {
		t.Error("a shown toast should draw its message")
	}
}

func TestDropdownOpensOnClick(t *testing.T) {
	h := guitest.New(220, 240)
	dd := ui.NewDropdown([]string{"Red", "Green", "Blue"},
		ui.DropdownPlaceholder("pick one"))
	h.SetContent(dd)

	// Closed: shows the placeholder, options are not drawn.
	if rec := h.Step(); !rec.TextContaining("pick one") {
		t.Fatal("closed dropdown should show its placeholder")
	}

	// Click the dropdown to open its popup list of options.
	h.Click(20, 12)
	rec := h.Step()
	if !rec.HasText("Green") {
		t.Errorf("opened dropdown should draw its options; texts=%v", rec.Texts())
	}
}
