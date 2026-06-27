package guitest_test

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// These tests double as a template for render-assertion tests: drive a widget
// through the headless harness, then query the recorded draw ops to assert what
// was actually painted (rather than just its state).

const tol = 0.01

func colourEq(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestProgressBarDrawsTrackFillAndBorder(t *testing.T) {
	h := guitest.New(200, 40)
	pal := h.App.Theme().Palette

	pb := ui.NewProgressBar(0.5)
	h.SetContent(pb) // fills the 200x40 surface, so Bounds = {0,0,200,40}

	rec := h.Step()

	// The track: a single surface-coloured rounded rect over the full bounds.
	track := rec.FillsOfColour(pal.Surface)
	if len(track) != 1 {
		t.Fatalf("expected 1 surface fill (the track), got %d", len(track))
	}
	if got := track[0]; !rectApprox(got, geom.Rect{W: 200, H: 40}) {
		t.Errorf("track rect: got %+v, want full bounds {0 0 200 40}", got)
	}

	// The fill: a primary-coloured rect whose width is value * bounds width.
	fill := rec.FillsOfColour(pal.Primary)
	if len(fill) != 1 {
		t.Fatalf("expected 1 primary fill, got %d", len(fill))
	}
	if got := fill[0]; !rectApprox(got, geom.Rect{W: 100, H: 40}) {
		t.Errorf("fill rect at value 0.5: got %+v, want width 100", got)
	}

	// A border is stroked in the border colour.
	bordered := false
	for _, op := range rec.OpsOfKind(guitest.OpStrokeRoundRect) {
		if colourEq(op.Colour, pal.Border) {
			bordered = true
		}
	}
	if !bordered {
		t.Error("expected a border-coloured StrokeRoundRect")
	}
}

func TestProgressBarFillIsProportional(t *testing.T) {
	cases := []struct {
		value float64
		wantW float64
	}{
		{0.25, 50},
		{0.5, 100},
		{1.0, 200},
	}
	for _, c := range cases {
		h := guitest.New(200, 40)
		pal := h.App.Theme().Palette
		h.SetContent(ui.NewProgressBar(c.value))

		fill := h.Step().FillsOfColour(pal.Primary)
		if len(fill) != 1 {
			t.Fatalf("value %v: expected 1 primary fill, got %d", c.value, len(fill))
		}
		if got := fill[0].W; !approxF(got, c.wantW) {
			t.Errorf("value %v: fill width got %v, want %v", c.value, got, c.wantW)
		}
	}
}

func TestProgressBarZeroDrawsNoFill(t *testing.T) {
	h := guitest.New(200, 40)
	pal := h.App.Theme().Palette
	h.SetContent(ui.NewProgressBar(0))

	rec := h.Step()

	// No primary fill is painted at value 0...
	if fill := rec.FillsOfColour(pal.Primary); len(fill) != 0 {
		t.Errorf("value 0 should paint no fill, got %d", len(fill))
	}
	// ...and the clip used to mask the fill is skipped entirely.
	if n := rec.Count(guitest.OpPushClip); n != 0 {
		t.Errorf("value 0 should not push a clip, got %d", n)
	}
	// The track is still drawn.
	if track := rec.FillsOfColour(pal.Surface); len(track) != 1 {
		t.Errorf("value 0 should still draw the track, got %d", len(track))
	}
}

func TestProgressBarSetValueRedraws(t *testing.T) {
	h := guitest.New(200, 40)
	pal := h.App.Theme().Palette
	pb := ui.NewProgressBar(0.2)
	h.SetContent(pb)

	if w := h.Step().FillsOfColour(pal.Primary)[0].W; !approxF(w, 40) {
		t.Fatalf("initial fill width got %v, want 40", w)
	}

	pb.SetValue(0.8)
	if w := h.Step().FillsOfColour(pal.Primary)[0].W; !approxF(w, 160) {
		t.Errorf("after SetValue(0.8) fill width got %v, want 160", w)
	}
}

func approxF(a, b float64) bool {
	d := a - b
	return d < tol && d > -tol
}

func rectApprox(got, want geom.Rect) bool {
	return approxF(got.X, want.X) && approxF(got.Y, want.Y) &&
		approxF(got.W, want.W) && approxF(got.H, want.H)
}
