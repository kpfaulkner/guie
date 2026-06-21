package guitest_test

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestColourPickerClickThroughHarness(t *testing.T) {
	h := guitest.New(220, 140)
	var got color.Color
	cp := ui.NewColourPicker(ui.ColourPickerValue(color.NRGBA{R: 255, A: 255}))
	cp.OnChange(func(c color.Color) { got = c })
	h.SetContent(cp) // fills the surface
	h.Step()         // lay out

	// The picker draws gradient strips for its three tracks.
	if h.Frame().Count(guitest.OpFillRect) < 10 {
		t.Fatal("colour picker should draw gradient strips")
	}

	// Drag the value track (third channel) to the far left → value 0 → black.
	// Tracks sit below the 36px swatch + gaps, inside 6px padding.
	const pad, swatch, track, gap = 6.0, 36.0, 18.0, 8.0
	valY := pad + swatch + gap + 2*(track+gap) + track/2
	h.Click(pad+2, valY) // far left of the value track

	if got == nil {
		t.Fatal("OnChange should have fired")
	}
	nc := color.NRGBAModel.Convert(cp.Colour()).(color.NRGBA)
	if nc.R > 5 || nc.G > 5 || nc.B > 5 {
		t.Fatalf("dragging value to the left should give black, got %v", nc)
	}
}
