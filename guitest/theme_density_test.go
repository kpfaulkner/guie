package guitest_test

import (
	"math"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// minSizeAt builds a widget under a theme of the given density and returns its
// MinSize. The widget has to be mounted for the theme to reach it.
func minSizeAt(mk func() ui.Widget, density float64) geom.Size {
	h := guitest.New(400, 300, densityOpts(density)...)
	w := mk()
	h.SetContent(w)
	h.Step()
	return w.MinSize()
}

// Every widget's size is "text part + padding part x density". The table gives, for
// each widget, the padding it adds at density 1, worked out by hand from its
// constants; the size must change by exactly that much per unit of density. A
// padding left unscaled makes the slope come out short, which a mere "is it linear"
// check cannot see (a constant is linear too).
func TestDensityScalesPaddingAndOnlyPadding(t *testing.T) {
	type entry struct {
		mk           func() ui.Widget
		wantW, wantH float64
	}
	widgets := map[string]entry{
		"button":        {func() ui.Widget { return ui.NewButton("Save") }, 28, 12},
		"checkbox":      {func() ui.Widget { return ui.NewCheckbox("Check") }, 8, 0},
		"radio":         {func() ui.Widget { return ui.NewRadioButton("Radio", ui.NewRadioGroup()) }, 8, 0},
		"textfield":     {func() ui.Widget { return ui.NewTextField() }, 0, 10}, // its width is a fixed default
		"textarea":      {func() ui.Widget { return ui.NewTextArea() }, 0, 12},
		"stepper":       {func() ui.Widget { return ui.NewStepper() }, 14, 10},
		"dropdown":      {func() ui.Widget { return ui.NewDropdown([]string{"one", "two"}) }, 10, 10},
		"date field":    {func() ui.Widget { return ui.NewDateField() }, 16, 10},
		"date picker":   {func() ui.Widget { return ui.NewDatePicker() }, 12, 12},
		"list":          {func() ui.Widget { return ui.NewList([]string{"one", "two"}) }, 10, 10},
		"tree":          {func() ui.Widget { return ui.NewTree(ui.NewTreeNode("root", ui.NewTreeNode("kid"))) }, 12, 8},
		"table":         {func() ui.Widget { return ui.NewTable([]ui.Column{{Title: "A"}, {Title: "B"}}) }, 24, 24},
		"menu bar":      {func() ui.Widget { m := ui.NewMenuBar(); m.AddMenu("File"); return m }, 24, 12},
		"colour picker": {func() ui.Widget { return ui.NewColourPicker() }, 0, 44},
		"tabs": {func() ui.Widget {
			tc := ui.NewTabContainer()
			tc.AddTab("One", ui.NewLabel("1"))
			return tc
		}, 24, 12},
	}
	for name, e := range widgets {
		t.Run(name, func(t *testing.T) {
			s05, s1, s2 := minSizeAt(e.mk, 0.5), minSizeAt(e.mk, 1), minSizeAt(e.mk, 2)
			for axis, c := range map[string]struct{ lo, mid, hi, want float64 }{
				"width":  {s05.W, s1.W, s2.W, e.wantW},
				"height": {s05.H, s1.H, s2.H, e.wantH},
			} {
				// f(d) = text + pad*d: the slope is the padding, from either side of 1.
				if !approx(c.hi-c.mid, c.want) || !approx(2*(c.mid-c.lo), c.want) {
					t.Errorf("%s: size at density 0.5 / 1 / 2 is %.3f / %.3f / %.3f; the padding slope should be %v", axis, c.lo, c.mid, c.hi, c.want)
				}
			}
		})
	}
}

func TestDensityOneIsExact(t *testing.T) {
	// 1 is the default, and Default() carries it explicitly: scaling by it must not
	// move a single pixel, or every existing app would shift.
	for _, v := range []float64{0, 0.5, 3, 5.25, 6, 14, 12.123456789} {
		if got := v * theme.Default().Density; got != v {
			t.Errorf("%v * default density = %v", v, got)
		}
	}
}

// textX returns the x of the first drawn string equal to s.
func drawnX(rec *guitest.Recording, s string) float64 {
	for _, op := range rec.OpsOfKind(guitest.OpDrawText) {
		if op.Text == s {
			return op.A.X
		}
	}
	return -1e9
}

// What is drawn must agree with what is measured: the text sits one scaled
// padding in from the edge, at every density. A widget that scaled MinSize but not
// its drawing (or the reverse) would be clipped or off-centre.
func TestDrawingUsesTheScaledPadding(t *testing.T) {
	cases := []struct {
		name string
		mk   func() ui.Widget
		text string
		pad  float64 // the base padding the text is inset by, before density
	}{
		{"textfield", func() ui.Widget { tf := ui.NewTextField(); tf.SetText("hello"); return tf }, "hello", 8},
		{"textarea", func() ui.Widget { ta := ui.NewTextArea(); ta.SetText("hello"); return ta }, "hello", 8},
		{"list", func() ui.Widget { return ui.NewList([]string{"hello"}) }, "hello", 5},
		{"table cell", func() ui.Widget {
			tb := ui.NewTable([]ui.Column{{Title: "Head"}})
			tb.AddRow("hello")
			return tb
		}, "hello", 6},
		{"dropdown", func() ui.Widget { return ui.NewDropdown([]string{"hello"}, ui.DropdownPlaceholder("hello")) }, "hello", 5},
		{"menu title", func() ui.Widget { m := ui.NewMenuBar(); m.AddMenu("hello"); return m }, "hello", 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, d := range []float64{0.5, 1, 2} {
				h := guitest.New(300, 120, densityOpts(d)...)
				w := c.mk()
				h.SetContent(w)
				h.MoveMouse(-1, -1)
				x := drawnX(h.Step(), c.text)
				want := w.Bounds().X + c.pad*d
				if !approx(x, want) {
					t.Errorf("density %v: text at x=%v, want %v (padding %v x %v from the left edge)", d, x, want, c.pad, d)
				}
			}
		})
	}
}

func TestRowHeightsScaleWithDensityAndHitTestingFollows(t *testing.T) {
	for _, d := range []float64{0.5, 2} {
		h := guitest.New(240, 200, densityOpts(d)...)
		var picked int = -1
		l := ui.NewList([]string{"a", "b", "c", "d"})
		l.OnSelect(func(i int) { picked = i })
		h.SetContent(l)
		h.Step()
		rh := l.RowHeight()
		base := guitest.New(240, 200)
		l1 := ui.NewList([]string{"a", "b"})
		base.SetContent(l1)
		base.Step()
		if want := (l1.RowHeight()-2*5)*1 + 2*5*d; !approx(rh, want) {
			t.Errorf("density %v: row height %v, want %v (padding scaled, text height not)", d, rh, want)
		}
		// A click in the middle of row 2 must select row 2: the hit test and the
		// drawing agree on where rows are.
		h.Click(30, l.Bounds().Y+2.5*rh)
		if picked != 2 {
			t.Errorf("density %v: click in row 2 selected %d", d, picked)
		}
	}
}

func TestTextFieldClickLandsOnTheRightCharacterWhenScaled(t *testing.T) {
	for _, d := range []float64{0.5, 2} {
		h := guitest.New(300, 80, densityOpts(d)...)
		tf := ui.NewTextField()
		tf.SetText("abcdef")
		h.SetContent(tf)
		h.Step()
		// The headless font is a fixed advance per rune; find it from the drawn text.
		adv := h.Font.Measure("a").W
		inner := tf.Bounds().X + 8*d
		h.Click(inner+2*adv+1, tf.Bounds().Y+5) // just past 2 characters
		h.TypeText("X")
		h.Step()
		if got := tf.Text(); got != "abXcdef" {
			t.Errorf("density %v: typed after a click past two characters gave %q, want abXcdef", d, got)
		}
	}
}

func TestToastAndTooltipPaddingScale(t *testing.T) {
	toastW := func(d float64) float64 {
		h := guitest.New(400, 200, densityOpts(d)...)
		h.SetContent(ui.NewLabel("page"))
		h.App.ShowToast("hello", ui.WithToastKind(ui.ToastSuccess))
		var w float64
		for i := 0; i < 20; i++ {
			for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
				r, g, b, _ := op.Colour.RGBA()
				sr, sg, sb, _ := theme.Default().Palette.Success.RGBA()
				if r == sr && g == sg && b == sb {
					w = op.Rect.W
				}
			}
		}
		return w
	}
	w1, w2 := toastW(1), toastW(2)
	if w1 == 0 || w2 == 0 {
		t.Fatal("no toast drawn")
	}
	// Width = text + 2*pad(10): doubling the density adds 2*10.
	if !approx(w2-w1, 2*10) {
		t.Errorf("toast width %v -> %v, want it to grow by 20 (2 x the 10px padding)", w1, w2)
	}
}

func TestDialogsAreNotBlownUpByDensity(t *testing.T) {
	// A modal built under a high density must still fit on a small surface once its
	// padding is scaled: it is centred from its (scaled) MinSize, so it stays inside.
	h := guitest.New(300, 200, densityOpts(1.25)...)
	h.SetContent(ui.NewLabel("page"))
	h.App.ShowMessage("Title", "Message")
	var maxX, maxY float64
	for _, op := range h.Step().OpsOfKind(guitest.OpFillRoundRect) {
		if op.Colour != nil {
			if r, g, b, a := op.Colour.RGBA(); a == 0xffff && !(r == 0 && g == 0 && b == 0) {
				maxX = math.Max(maxX, op.Rect.X+op.Rect.W)
				maxY = math.Max(maxY, op.Rect.Y+op.Rect.H)
			}
		}
	}
	if maxX > 300 || maxY > 200 {
		t.Errorf("dialog extends to (%v,%v) on a 300x200 surface", maxX, maxY)
	}
}

// densityOpts is a default theme at the given density, with the harness's
// deterministic font put back: WithTheme replaces the whole theme, font included,
// and the real font would make the arithmetic below depend on font metrics.
func densityOpts(d float64) []ui.AppOption {
	th := theme.Default().WithDensity(d)
	return []ui.AppOption{ui.WithTheme(th), ui.WithFont(guitest.NewFont(th.FontSize))}
}

func TestTreeChevronHitFollowsDensity(t *testing.T) {
	for _, d := range []float64{0.5, 2} {
		h := guitest.New(260, 120, densityOpts(d)...)
		kid := ui.NewTreeNode("kid")
		root := ui.NewTreeNode("root", kid)
		tr := ui.NewTree(root)
		h.SetContent(tr)
		h.Step()
		// Just outside the scaled chevron but inside where an unscaled one would be: a
		// click there must NOT toggle. (2 px left of the scaled start at 2x; 1 px past the
		// scaled end at 0.5x.)
		miss := tr.Bounds().X + 6
		if d < 1 {
			miss = tr.Bounds().X + 17
		}
		h.Click(miss, tr.Bounds().Y+tr.RowHeight()/2)
		if root.Expanded() {
			t.Errorf("density %v: a click outside the scaled chevron expanded the node", d)
		}
		// The root's chevron starts one scaled row padding from the left edge.
		h.Click(tr.Bounds().X+4*d+2, tr.Bounds().Y+tr.RowHeight()/2)
		if !root.Expanded() {
			t.Errorf("density %v: a click on the chevron did not expand the node", d)
		}
		// Expanded: the child's label is indented by a scaled step.
		h.Step()
		kidX := drawnX(h.Frame(), "kid")
		want := tr.Bounds().X + 4*d + 16*d + 14 + 4*d // pad + one indent + chevron column + gap
		if !approx(kidX, want) {
			t.Errorf("density %v: child label at x=%v, want %v", d, kidX, want)
		}
	}
}
