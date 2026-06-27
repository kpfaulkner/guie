package ui

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

// nrgbaOf converts any color to non-premultiplied RGBA for field comparison.
func nrgbaOf(c color.Color) color.NRGBA {
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}

func wantRGBA(t *testing.T, name string, got color.Color, r, g, b, a uint8) {
	t.Helper()
	n := nrgbaOf(got)
	if n.R != r || n.G != g || n.B != b || n.A != a {
		t.Errorf("%s: got {%d %d %d %d}, want {%d %d %d %d}",
			name, n.R, n.G, n.B, n.A, r, g, b, a)
	}
}

// --- button.go: scaleRGB / darken / lighten ---

func TestScaleRGB(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}

	// Halving scales each channel toward black; alpha is preserved.
	wantRGBA(t, "darken 0.5", darken(base, 0.5), 50, 75, 100, 255)

	// factor 0 collapses to black but keeps alpha.
	wantRGBA(t, "darken 0", darken(base, 0), 0, 0, 0, 255)

	// Lightening past white clamps each channel at 255.
	wantRGBA(t, "lighten 2", lighten(base, 2), 200, 255, 255, 255)

	// Identity factor returns the same opaque colour.
	wantRGBA(t, "scale 1", scaleRGB(base, 1), 100, 150, 200, 255)
}

func TestScaleRGBPreservesAlpha(t *testing.T) {
	// A semi-transparent input: RGBA() is premultiplied, but the alpha byte
	// itself must pass through unscaled.
	got := scaleRGB(color.RGBA{R: 80, G: 80, B: 80, A: 128}, 1)
	if a := nrgbaOf(got).A; a != 128 {
		t.Errorf("alpha: got %d, want 128", a)
	}
}

// --- colour_picker.go: hexOf / contrastColour / gradientColour ---

func TestHexOf(t *testing.T) {
	cases := []struct {
		c    color.Color
		want string
	}{
		{color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xFF}, "#123456"},
		{color.RGBA{R: 0xAB, G: 0xCD, B: 0xEF, A: 0xFF}, "#ABCDEF"},
		{color.Black, "#000000"},
		{color.White, "#FFFFFF"},
	}
	for _, c := range cases {
		if got := hexOf(c.c); got != c.want {
			t.Errorf("hexOf(%v): got %q, want %q", c.c, got, c.want)
		}
	}
}

func TestContrastColour(t *testing.T) {
	cases := []struct {
		name string
		in   color.Color
		want color.Color
	}{
		{"white -> black", color.White, color.Black},
		{"black -> white", color.Black, color.White},
		{"green is bright -> black", color.RGBA{G: 255, A: 255}, color.Black},
		{"blue is dark -> white", color.RGBA{B: 255, A: 255}, color.White},
		{"red is dark -> white", color.RGBA{R: 255, A: 255}, color.White},
	}
	for _, c := range cases {
		if got := contrastColour(c.in); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestGradientColourHueTrack(t *testing.T) {
	// Track 0 is the hue gradient and ignores picker state: t=0 is pure red.
	p := NewColourPicker()
	wantRGBA(t, "hue at 0", p.gradientColour(0, 0), 255, 0, 0, 255)
}

func TestGradientColourAlphaTrack(t *testing.T) {
	// Track 3 holds the current colour and varies only alpha by t.
	p := NewColourPicker(ColourPickerValue(color.RGBA{R: 255, A: 255})) // opaque red
	got := p.gradientColour(3, 0.5)
	// 0.5*255 + 0.5 rounds to 128.
	wantRGBA(t, "alpha at 0.5", got, 255, 0, 0, 128)

	wantRGBA(t, "alpha at 1", p.gradientColour(3, 1), 255, 0, 0, 255)
	wantRGBA(t, "alpha at 0", p.gradientColour(3, 0), 255, 0, 0, 0)
}

// --- progressbar.go: Value / MinSize / clamping ---

func TestProgressBarValueClampAndMinSize(t *testing.T) {
	if v := NewProgressBar(0.4).Value(); !approx(v, 0.4) {
		t.Errorf("Value: got %v, want 0.4", v)
	}
	if v := NewProgressBar(2).Value(); !approx(v, 1) {
		t.Errorf("Value clamp high: got %v, want 1", v)
	}
	if v := NewProgressBar(-1).Value(); !approx(v, 0) {
		t.Errorf("Value clamp low: got %v, want 0", v)
	}

	p := NewProgressBar(0)
	p.SetValue(5) // clamps to 1
	if v := p.Value(); !approx(v, 1) {
		t.Errorf("SetValue clamp: got %v, want 1", v)
	}

	if ms := NewProgressBar(0).MinSize(); !approx(ms.W, 120) || !approx(ms.H, progressHeight) {
		t.Errorf("MinSize: got %+v, want {120 %d}", ms, progressHeight)
	}
}

// --- box.go: NewBox ---

func TestNewBox(t *testing.T) {
	h := NewBox(geom.Horizontal)
	if h.Direction != geom.Horizontal || h.Spacing != 0 {
		t.Errorf("NewBox(Horizontal): got dir=%v spacing=%v", h.Direction, h.Spacing)
	}
	v := NewBox(geom.Vertical)
	if v.Direction != geom.Vertical {
		t.Errorf("NewBox(Vertical): got dir=%v", v.Direction)
	}
}

// --- layout.go: Span item option ---

func TestSpanOption(t *testing.T) {
	d := defaultLayoutData()
	Span(3, 2)(&d)
	if d.ColSpan != 3 || d.RowSpan != 2 {
		t.Errorf("Span(3,2): got col=%d row=%d, want 3,2", d.ColSpan, d.RowSpan)
	}

	// Values below 1 are clamped to 1.
	d = defaultLayoutData()
	Span(0, -5)(&d)
	if d.ColSpan != 1 || d.RowSpan != 1 {
		t.Errorf("Span(0,-5): got col=%d row=%d, want 1,1", d.ColSpan, d.RowSpan)
	}
}

// --- grid.go: Measure ---

func TestGridMeasure(t *testing.T) {
	// 2 columns, 8px spacing, four 50x20 cells -> a 2x2 grid.
	g := NewGrid(2, 8)
	its := items(newStub(50, 20), newStub(50, 20), newStub(50, 20), newStub(50, 20))
	got := g.Measure(its)
	// width  = 50 + 50 + 8 (one gap between 2 columns)
	// height = 20 + 20 + 8 (one gap between 2 rows)
	if !approx(got.W, 108) || !approx(got.H, 48) {
		t.Errorf("Grid.Measure: got %+v, want {108 48}", got)
	}
}

func TestGridMeasureEmpty(t *testing.T) {
	if got := NewGrid(3, 8).Measure(nil); got != (geom.Size{}) {
		t.Errorf("Grid.Measure(nil): got %+v, want zero", got)
	}
}

func TestGridMeasureSpan(t *testing.T) {
	// A single cell spanning both columns: its width is split across the two
	// tracks, so the measured width is the cell width plus one inter-column gap.
	g := NewGrid(2, 10)
	its := []Item{{
		Widget: newStub(100, 20),
		Data:   LayoutData{Align: geom.AlignStretch, ColSpan: 2, RowSpan: 1},
	}}
	got := g.Measure(its)
	// Each column gets 100/2 = 50; total = 50 + 50 + 10 gap = 110.
	if !approx(got.W, 110) || !approx(got.H, 20) {
		t.Errorf("Grid.Measure span: got %+v, want {110 20}", got)
	}
}
