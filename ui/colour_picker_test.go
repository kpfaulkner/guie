package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func near8(a, b uint8) bool {
	d := int(a) - int(b)
	return d >= -1 && d <= 1
}

func TestHSVRoundTrip(t *testing.T) {
	cases := []color.NRGBA{
		{R: 255, A: 255},
		{G: 128, A: 255},
		{B: 200, A: 255},
		{R: 64, G: 128, B: 192, A: 255},
		{R: 200, G: 200, B: 200, A: 255}, // gray
		{A: 255},                         // black
	}
	for _, c := range cases {
		h, s, v := rgbToHSV(c)
		got := hsvToRGB(h, s, v)
		if !near8(got.R, c.R) || !near8(got.G, c.G) || !near8(got.B, c.B) {
			t.Fatalf("round-trip %v -> %v (hsv %.3f,%.3f,%.3f)", c, got, h, s, v)
		}
	}
}

func TestColourPickerValueAndColour(t *testing.T) {
	p := NewColourPicker(ColourPickerValue(color.NRGBA{R: 255, A: 255}))
	got := color.NRGBAModel.Convert(p.Colour()).(color.NRGBA)
	if !near8(got.R, 255) || !near8(got.G, 0) || !near8(got.B, 0) {
		t.Fatalf("value should be red, got %v", got)
	}
}

func TestColourPickerClickSetsHue(t *testing.T) {
	p := NewColourPicker(ColourPickerValue(color.NRGBA{R: 255, A: 255}))
	p.SetBounds(geom.Rect{X: 0, Y: 0, W: 220, H: 140})
	fired := 0
	p.OnChange(func(color.Color) { fired++ })

	// Click the middle of the hue track (channel 0).
	tr := p.trackRect(0)
	p.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: tr.X + tr.W/2, Y: tr.Y + tr.H/2}})

	if p.h < 0.45 || p.h > 0.55 {
		t.Fatalf("clicking the track middle should set hue ~0.5, got %v", p.h)
	}
	if fired == 0 {
		t.Fatal("OnChange should fire on a click that changes the colour")
	}
}

func TestColourPickerKeyboard(t *testing.T) {
	p := NewColourPicker(ColourPickerValue(color.NRGBA{R: 255, G: 255, B: 255, A: 255})) // h0 s0 v1
	// Down moves the active channel to saturation (index 1).
	p.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyDown})
	if p.active != 1 {
		t.Fatalf("Down should select channel 1, got %d", p.active)
	}
	// Right increases saturation.
	before := p.s
	p.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyRight})
	if p.s <= before {
		t.Fatalf("Right should increase saturation, %v -> %v", before, p.s)
	}
}

func TestColourPickerAlphaChannel(t *testing.T) {
	// Initial alpha is read from the value.
	p := NewColourPicker(ColourPickerValue(color.NRGBA{R: 255, A: 128}))
	if got := color.NRGBAModel.Convert(p.Colour()).(color.NRGBA); !near8(got.A, 128) {
		t.Fatalf("alpha should round-trip, got A=%d", got.A)
	}

	// Channel 3 is alpha; dragging it to 0 makes the colour fully transparent.
	p.SetBounds(geom.Rect{X: 0, Y: 0, W: 220, H: 170})
	tr := p.trackRect(3)
	p.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: tr.X, Y: tr.Y + tr.H/2}})
	if got := color.NRGBAModel.Convert(p.Colour()).(color.NRGBA); got.A != 0 {
		t.Fatalf("dragging alpha to the left should give A=0, got %d", got.A)
	}
}

func TestColourPickerSetColourFiresOnChange(t *testing.T) {
	p := NewColourPicker(ColourPickerValue(color.NRGBA{R: 255, A: 255}))
	calls := 0
	p.OnChange(func(color.Color) { calls++ })
	p.SetColour(color.NRGBA{R: 255, A: 255}) // unchanged
	p.SetColour(color.NRGBA{B: 255, A: 255}) // change to blue
	if calls != 1 {
		t.Fatalf("OnChange should fire once for a real change, got %d", calls)
	}
}
