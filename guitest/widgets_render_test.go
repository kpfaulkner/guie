package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// fakeImage is a render.Image with a fixed size; the recording canvas only needs
// its size, not real pixels.
type fakeImage struct{ w, h float64 }

func (f fakeImage) Size() geom.Size { return geom.Size{W: f.w, H: f.h} }

func TestImageWidgetDraws(t *testing.T) {
	h := guitest.New(100, 80)
	img := ui.NewImage(fakeImage{w: 40, h: 20})
	img.SetFit(ui.FitStretch)
	h.SetContent(img)

	rec := h.Step()
	if n := rec.Count(guitest.OpDrawImage); n != 1 {
		t.Fatalf("Image widget should draw exactly one image, got %d", n)
	}
	// FitStretch fills the full bounds.
	dst := rec.OpsOfKind(guitest.OpDrawImage)[0].Rect
	if !rectApprox(dst, geom.Rect{W: 100, H: 80}) {
		t.Errorf("FitStretch dst: got %+v, want full bounds", dst)
	}
}

func TestImageWidgetNilDrawsNothing(t *testing.T) {
	h := guitest.New(100, 80)
	h.SetContent(ui.NewImage(nil))
	if n := h.Step().Count(guitest.OpDrawImage); n != 0 {
		t.Errorf("a nil image should draw nothing, got %d DrawImage ops", n)
	}
}

func TestCheckboxCheckedDrawsTick(t *testing.T) {
	h := guitest.New(160, 40)
	pal := h.App.Theme().Palette
	h.SetContent(ui.NewCheckbox("On", ui.Checked(true)))

	rec := h.Step()
	if !rec.HasText("On") {
		t.Error("checkbox should draw its label")
	}
	// The tick is two line segments.
	if n := rec.Count(guitest.OpDrawLine); n != 2 {
		t.Errorf("checked box should draw a 2-segment tick, got %d lines", n)
	}
	// The box is filled with the primary colour when checked.
	if len(rec.FillsOfColour(pal.Primary)) == 0 {
		t.Error("checked box should fill with the primary colour")
	}
}

func TestCheckboxUncheckedHasNoTick(t *testing.T) {
	h := guitest.New(160, 40)
	pal := h.App.Theme().Palette
	h.SetContent(ui.NewCheckbox("Off"))

	rec := h.Step()
	if !rec.HasText("Off") {
		t.Error("checkbox should draw its label")
	}
	if n := rec.Count(guitest.OpDrawLine); n != 0 {
		t.Errorf("an unchecked box should draw no tick, got %d lines", n)
	}
	if len(rec.FillsOfColour(pal.Primary)) != 0 {
		t.Error("an unchecked, unhovered box should not fill with primary")
	}
}

func TestButtonNormalVsFlat(t *testing.T) {
	hn := guitest.New(120, 40)
	pal := hn.App.Theme().Palette
	hn.SetContent(ui.NewButton("OK"))
	hn.MoveMouse(-1, -1) // off-surface, so the button isn't in its hover state
	rec := hn.Step()
	if !rec.HasText("OK") {
		t.Error("button should draw its label")
	}
	if len(rec.FillsOfColour(pal.Primary)) == 0 {
		t.Error("a normal button should fill with the primary colour")
	}

	hf := guitest.New(120, 40)
	hf.SetContent(ui.NewButton("Flat", ui.ButtonFlat()))
	hf.MoveMouse(-1, -1)
	rf := hf.Step()
	if !rf.HasText("Flat") {
		t.Error("flat button should still draw its label")
	}
	if len(rf.FillsOfColour(pal.Primary)) != 0 {
		t.Error("a flat button should not fill until hovered/pressed")
	}
}

func TestSliderDrawsTrackAndHandle(t *testing.T) {
	h := guitest.New(160, 40)
	h.SetContent(ui.NewSlider(ui.SliderValue(0.5)))
	rec := h.Step()
	// Two line segments (full track + filled portion) and one handle circle.
	if n := rec.Count(guitest.OpDrawLine); n != 2 {
		t.Errorf("slider should draw 2 track lines, got %d", n)
	}
	if n := rec.Count(guitest.OpFillCircle); n != 1 {
		t.Errorf("slider should draw 1 handle circle, got %d", n)
	}
}

func TestRadioSelectedVsUnselected(t *testing.T) {
	rg := ui.NewRadioGroup()

	hu := guitest.New(160, 40)
	hu.SetContent(ui.NewRadioButton("A", rg))
	hu.MoveMouse(-1, -1) // avoid the hover-highlight circle
	ru := hu.Step()
	if !ru.HasText("A") {
		t.Error("radio should draw its label")
	}
	// Unselected, unhovered: an outline circle only, no filled dot.
	if n := ru.Count(guitest.OpFillCircle); n != 0 {
		t.Errorf("unselected radio should draw no filled circle, got %d", n)
	}
	if n := ru.Count(guitest.OpStrokeCircle); n != 1 {
		t.Errorf("radio should stroke its outline circle, got %d", n)
	}

	rg2 := ui.NewRadioGroup()
	hs := guitest.New(160, 40)
	rb := ui.NewRadioButton("B", rg2)
	hs.SetContent(rb)
	hs.MoveMouse(-1, -1)
	rb.Select()
	rs := hs.Step()
	if n := rs.Count(guitest.OpFillCircle); n != 1 {
		t.Errorf("a selected radio should draw its filled dot, got %d", n)
	}
}
