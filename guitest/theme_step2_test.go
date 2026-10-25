package guitest_test

import (
	"image/color"
	"reflect"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

func sameColour(a, b color.Color) bool { return to8(a) == to8(b) }

// strokes returns the stroke ops (rounded rect and circle) in a recording.
func strokes(rec *guitest.Recording) []guitest.Op {
	return append(rec.OpsOfKind(guitest.OpStrokeRoundRect), rec.OpsOfKind(guitest.OpStrokeCircle)...)
}

func focusedFrame(th theme.Theme) *guitest.Recording {
	h := guitest.New(240, 120, ui.WithTheme(th))
	b := ui.NewButton("b")
	h.SetContent(b)
	h.MoveMouse(-1, -1)
	b.RequestFocus()
	h.Step()
	return h.Step()
}

func TestFocusRingHaloAndWidthsFollowTheTheme(t *testing.T) {
	for _, c := range []struct {
		name      string
		th        theme.Theme
		ring, hal float64
	}{
		{"default", theme.Default(), 2, 1},
		{"high contrast", theme.HighContrast(), 3, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			pal := c.th.Palette
			var ring, halo *guitest.Op
			for _, op := range strokes(focusedFrame(c.th)) {
				op := op
				switch {
				case sameColour(op.Colour, pal.Accent) && op.Width == c.ring:
					ring = &op
				case sameColour(op.Colour, pal.Background) && op.Width == c.hal:
					halo = &op
				}
			}
			if ring == nil {
				t.Fatalf("no accent ring of width %v", c.ring)
			}
			if halo == nil {
				t.Fatalf("no halo of width %v in the page colour", c.hal)
			}
			// The halo sits inside the ring, so the ring is bounded by the page colour
			// and a filled control's own colour can never swallow it.
			if !(halo.Rect.W < ring.Rect.W && halo.Rect.H < ring.Rect.H) {
				t.Errorf("halo %v should be inside the ring %v", halo.Rect, ring.Rect)
			}
		})
	}
}

func TestUnfocusedControlDrawsNoRingOrHalo(t *testing.T) {
	h := guitest.New(240, 120)
	pal := theme.Default().Palette
	h.SetContent(ui.NewButton("b"))
	h.MoveMouse(-1, -1)
	for _, op := range strokes(h.Step()) {
		if sameColour(op.Colour, pal.Accent) || sameColour(op.Colour, pal.Background) {
			t.Errorf("unfocused button drew a %v stroke", to8(op.Colour))
		}
	}
}

func TestBorderWidthFollowsTheTheme(t *testing.T) {
	th := theme.Default()
	th.BorderWidth = 3
	h := guitest.New(200, 120, ui.WithTheme(th))
	h.SetContent(ui.NewList([]string{"a", "b"}))
	h.MoveMouse(-1, -1)
	var found bool
	for _, op := range h.Step().OpsOfKind(guitest.OpStrokeRect) {
		if sameColour(op.Colour, th.Palette.Border) {
			found = true
			if op.Width != 3 {
				t.Errorf("list outline width = %v, want the theme's 3", op.Width)
			}
		}
	}
	if !found {
		t.Error("the list drew no outline")
	}
}

func TestToastsUseTheThemesStatusColours(t *testing.T) {
	th := theme.HighContrast()
	pal := th.Palette
	fills := func(kind ui.ToastKind) []color.Color {
		h := guitest.New(300, 160, ui.WithTheme(th))
		h.SetContent(ui.NewLabel("page"))
		h.App.ShowToast("hi", ui.WithToastKind(kind))
		var got []color.Color
		for i := 0; i < 20; i++ {
			for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
				got = append(got, op.Colour)
			}
		}
		return got
	}
	has := func(cs []color.Color, want color.Color) bool {
		for _, c := range cs {
			if sameColour(c, want) {
				return true
			}
		}
		return false
	}
	for kind, want := range map[ui.ToastKind]color.Color{
		ui.ToastSuccess: pal.Success, ui.ToastWarning: pal.Warning, ui.ToastError: pal.Error, ui.ToastInfo: pal.Primary,
	} {
		if !has(fills(kind), want) {
			t.Errorf("toast kind %v did not fill with %v", kind, to8(want))
		}
	}

	// An explicit Info colour replaces Primary for info toasts.
	info := th
	info.Palette.Info = color.RGBA{R: 9, G: 99, B: 199, A: 255}
	h := guitest.New(300, 160, ui.WithTheme(info))
	h.SetContent(ui.NewLabel("page"))
	h.App.ShowToast("hi", ui.WithToastKind(ui.ToastInfo))
	var got []color.Color
	for i := 0; i < 20; i++ {
		for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
			got = append(got, op.Colour)
		}
	}
	if !has(got, info.Palette.Info) {
		t.Error("an info toast should use Palette.Info when set")
	}
}

func TestModalScrimComesFromTheTheme(t *testing.T) {
	for _, th := range []theme.Theme{theme.Default(), theme.HighContrast()} {
		h := guitest.New(300, 200, ui.WithTheme(th))
		h.SetContent(ui.NewLabel("page"))
		h.App.ShowMessage("t", "m")
		var got bool
		for _, op := range h.Step().OpsOfKind(guitest.OpFillRect) {
			if sameColour(op.Colour, th.Palette.Scrim) && op.Rect.W == 300 && op.Rect.H == 200 {
				got = true
			}
		}
		if !got {
			t.Errorf("no full-surface scrim in %v", to8(th.Palette.Scrim))
		}
	}
}

func TestNoShadowsThemeTurnsShadowsOff(t *testing.T) {
	shadowOps := func(th theme.Theme) int {
		h := guitest.New(300, 200, ui.WithTheme(th))
		h.SetContent(ui.NewLabel("page"))
		h.App.ShowMessage("t", "m")
		n := 0
		for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
			if r, g, b, a := op.Colour.RGBA(); r == 0 && g == 0 && b == 0 && a > 0 && a < 0xffff {
				n++
			}
		}
		return n
	}
	if shadowOps(theme.Default()) == 0 {
		t.Fatal("the default theme should draw a soft shadow under a dialog")
	}
	if n := shadowOps(theme.HighContrast()); n != 0 {
		t.Errorf("high contrast drew %d shadow layers", n)
	}
	// App.SetShadows(false) still wins on its own.
	h := guitest.New(300, 200)
	h.App.SetShadows(false)
	h.SetContent(ui.NewLabel("page"))
	h.App.ShowMessage("t", "m")
	for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
		if r, g, b, a := op.Colour.RGBA(); r == 0 && g == 0 && b == 0 && a > 0 && a < 0xffff {
			t.Error("SetShadows(false) should still turn shadows off")
			break
		}
	}
}

// The state tints are the theme's, not constants: a theme with stronger hover makes
// a hovered row measurably different from the resting colour.
func TestHoverStrengthFollowsTheTheme(t *testing.T) {
	hovered := func(th theme.Theme) color.Color {
		h := guitest.New(200, 120, ui.WithTheme(th))
		l := ui.NewList([]string{"a", "b", "c"})
		h.SetContent(l)
		h.MoveMouse(-1, -1)
		h.Step()
		h.MoveMouse(30, 10)
		var row color.Color
		for _, op := range h.Step().OpsOfKind(guitest.OpFillRect) {
			if !sameColour(op.Colour, th.Palette.Surface) && !sameColour(op.Colour, th.Palette.Primary) {
				row = op.Colour
			}
		}
		return row
	}
	weak, strong := theme.Default(), theme.Default()
	weak.States.Hover, strong.States.Hover = 0.02, 0.4
	w, s := hovered(weak), hovered(strong)
	if w == nil || s == nil {
		t.Fatal("no hover fill drawn")
	}
	dist := func(c color.Color) uint32 {
		r, g, b, _ := c.RGBA()
		sr, sg, sb, _ := theme.Default().Palette.Surface.RGBA()
		d := func(a, b uint32) uint32 {
			if a > b {
				return a - b
			}
			return b - a
		}
		return d(r, sr) + d(g, sg) + d(b, sb)
	}
	if !(dist(s) > 4*dist(w)) {
		t.Errorf("hover at 0.4 (%v) should be far from the surface compared with 0.02 (%v)", to8(s), to8(w))
	}
}

// Filled-button states now come from the theme's tints. The default must stay close
// to the old fixed factors: pressed was exactly 0.8 of the fill, hover was 1.15x.
func TestFilledButtonStatesStayCloseToTheOldLook(t *testing.T) {
	base := theme.Default().Palette.Primary
	br, bg, bb, _ := base.RGBA()
	h := guitest.New(240, 120)
	b := ui.NewButton("ok")
	h.SetContent(b)
	h.MoveMouse(-1, -1)
	fill := func() color.Color {
		var c color.Color
		for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
			c = op.Colour
		}
		return c
	}
	if got := fill(); !sameColour(got, base) {
		t.Fatalf("resting fill %v, want Primary %v", to8(got), to8(base))
	}
	b.RequestFocus()
	h.MoveMouse(20, 12)
	h.PressMouse(0)
	var pressed color.Color
	for i := 0; i < 20; i++ { // let the fade finish
		pressed = fill()
	}
	pr, pg, pb, _ := pressed.RGBA()
	want := func(v uint32) float64 { return float64(v>>8) * 0.8 }
	for i, pair := range [][2]float64{{float64(pr >> 8), want(br)}, {float64(pg >> 8), want(bg)}, {float64(pb >> 8), want(bb)}} {
		if d := pair[0] - pair[1]; d > 2 || d < -2 {
			t.Errorf("pressed channel %d = %v, want about %v (the old 0.8 shade)", i, pair[0], pair[1])
		}
	}
}

// A theme written before the new fields existed must draw exactly what the
// complete default draws.
func TestOldStyleThemeDrawsLikeTheDefault(t *testing.T) {
	old := theme.Theme{Palette: theme.DefaultPalette(), FontSize: 14, CornerRadius: 5}
	build := map[string]func() ui.Widget{
		"button":   func() ui.Widget { return ui.NewButton("ok") },
		"list":     func() ui.Widget { l := ui.NewList([]string{"a", "b", "c"}); l.SetSelected(1); return l },
		"textarea": func() ui.Widget { ta := ui.NewTextArea(); ta.SetText("x\ny"); return ta },
		"table": func() ui.Widget {
			tb := ui.NewTable([]ui.Column{{Title: "A"}})
			tb.SetStriped(true)
			tb.AddRow("1")
			tb.AddRow("2")
			return tb
		},
	}
	for name, mk := range build {
		draw := func(th theme.Theme) []guitest.Op {
			h := guitest.New(240, 140, ui.WithTheme(th))
			w := mk()
			h.SetContent(w)
			h.MoveMouse(-1, -1)
			w.(interface{ RequestFocus() }).RequestFocus()
			h.Step()
			return h.Step().Ops
		}
		if a, b := draw(theme.Default()), draw(old); !reflect.DeepEqual(a, b) {
			t.Errorf("%s: an old-style theme draws differently from the default (%d vs %d ops)", name, len(a), len(b))
		}
	}
}
