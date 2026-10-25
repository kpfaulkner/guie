package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// gapBetween returns the empty space between a and b along the main axis.
func gapH(a, b ui.Widget) float64 { return b.Bounds().X - (a.Bounds().X + a.Bounds().W) }
func gapV(a, b ui.Widget) float64 { return b.Bounds().Y - (a.Bounds().Y + a.Bounds().H) }

// row builds a container of two buttons laid out by l and returns the buttons.
func twoButtons(l ui.Layout) (*ui.Container, *ui.Button, *ui.Button) {
	c := ui.NewContainer()
	c.SetLayout(l)
	a, b := ui.NewButton("a"), ui.NewButton("b")
	c.Add(a)
	c.Add(b)
	return c, a, b
}

func themed(th theme.Theme) []ui.AppOption {
	return []ui.AppOption{ui.WithTheme(th), ui.WithFont(guitest.NewFont(th.FontSize))}
}

func TestThemeGapFollowsSpacingAndDensity(t *testing.T) {
	for _, c := range []struct {
		name     string
		spacing  float64 // 0 = the default (8)
		density  float64
		wantGap  float64
		explicit bool
	}{
		{"default", 0, 0, 8, false},
		{"custom spacing", 12, 0, 12, false},
		{"density scales it", 0, 2, 16, false},
		{"both", 12, 1.5, 18, false},
		{"compact", 0, 0.5, 4, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			th := theme.Default()
			if c.spacing != 0 {
				th.Spacing = c.spacing
			}
			if c.density != 0 {
				th = th.WithDensity(c.density)
			}
			for dir, mk := range map[string]func() (*ui.Container, *ui.Button, *ui.Button){
				"hbox": func() (*ui.Container, *ui.Button, *ui.Button) { return twoButtons(ui.HBox(ui.ThemeGap)) },
				"vbox": func() (*ui.Container, *ui.Button, *ui.Button) { return twoButtons(ui.VBox(ui.ThemeGap)) },
			} {
				h := guitest.New(300, 300, themed(th)...)
				root, a, b := mk()
				h.SetContent(root)
				h.Step()
				got := gapH(a, b)
				if dir == "vbox" {
					got = gapV(a, b)
				}
				if !approx(got, c.wantGap) {
					t.Errorf("%s: gap = %v, want %v", dir, got, c.wantGap)
				}
			}
		})
	}
}

func TestExplicitSpacingIsNotScaled(t *testing.T) {
	th := theme.Default().WithDensity(2)
	th.Spacing = 20
	h := guitest.New(300, 300, themed(th)...)
	root, a, b := twoButtons(ui.HBox(10))
	h.SetContent(root)
	h.Step()
	if got := gapH(a, b); !approx(got, 10) {
		t.Errorf("HBox(10) gap = %v under density 2 and spacing 20, want exactly 10", got)
	}
	// Zero is also explicit: no gap.
	h2 := guitest.New(300, 300, themed(th)...)
	root2, a2, b2 := twoButtons(ui.HBox(0))
	h2.SetContent(root2)
	h2.Step()
	if got := gapH(a2, b2); !approx(got, 0) {
		t.Errorf("HBox(0) gap = %v, want 0", got)
	}
}

func TestThemeGapInAGrid(t *testing.T) {
	th := theme.Default()
	th.Spacing = 10
	h := guitest.New(300, 300, themed(th)...)
	g := ui.NewGrid(2, ui.ThemeGap)
	c := ui.NewContainer()
	c.SetLayout(g)
	bs := []*ui.Button{ui.NewButton("a"), ui.NewButton("b"), ui.NewButton("c"), ui.NewButton("d")}
	for _, b := range bs {
		c.Add(b)
	}
	h.SetContent(c)
	h.Step()
	// Columns and rows are stretched to fill, so measure the gap from cell edges.
	if got := gapH(bs[0], bs[1]); !approx(got, 10) {
		t.Errorf("grid column gap = %v, want 10", got)
	}
	if got := gapV(bs[0], bs[2]); !approx(got, 10) {
		t.Errorf("grid row gap = %v, want 10", got)
	}
	// The tracks share what is left after the gaps, so the last column and row end
	// exactly at the container's edge: a gap that the track sums did not know about
	// would leave them short or long.
	if r := bs[1].Bounds().X + bs[1].Bounds().W; !approx(r, 300) {
		t.Errorf("last column ends at %v, want 300", r)
	}
	if b := bs[3].Bounds().Y + bs[3].Bounds().H; !approx(b, 300) {
		t.Errorf("last row ends at %v, want 300", b)
	}
}

func TestThemeGapFollowsSetTheme(t *testing.T) {
	h := guitest.New(300, 300)
	root, a, b := twoButtons(ui.HBox(ui.ThemeGap))
	h.SetContent(root)
	h.Step()
	if got := gapH(a, b); !approx(got, 8) {
		t.Fatalf("initial gap = %v, want the default 8", got)
	}

	th := theme.Default()
	th.Spacing = 20
	h.App.SetTheme(th)
	h.Step()
	if got := gapH(a, b); !approx(got, 20) {
		t.Errorf("gap after SetTheme = %v, want 20", got)
	}
	h.App.SetTheme(th.WithDensity(0.5))
	h.Step()
	if got := gapH(a, b); !approx(got, 10) {
		t.Errorf("gap after a density change = %v, want 10", got)
	}
}

func TestThemeGapCountsInMinSize(t *testing.T) {
	size := func(l ui.Layout, th theme.Theme) geom.Size {
		h := guitest.New(300, 300, themed(th)...)
		root, _, _ := twoButtons(l)
		h.SetContent(root)
		h.Step()
		return root.MinSize()
	}
	th := theme.Default()
	th.Spacing = 8
	none := size(ui.HBox(0), th)
	themed8 := size(ui.HBox(ui.ThemeGap), th)
	if !approx(themed8.W-none.W, 8) {
		t.Errorf("the gap should add 8 to the width: %v vs %v", themed8.W, none.W)
	}
	th2 := th.WithDensity(2)
	if got := size(ui.HBox(ui.ThemeGap), th2).W; !approx(got-size(ui.HBox(0), th2).W, 16) {
		t.Errorf("at density 2 the gap should add 16, got %v", got-size(ui.HBox(0), th2).W)
	}
}

func TestThemeGapInNestedContainersAndOldThemes(t *testing.T) {
	// A theme written before Spacing existed still gives the default 8.
	old := theme.Theme{Palette: theme.DefaultPalette(), FontSize: 14, CornerRadius: 5}
	h := guitest.New(300, 300, ui.WithTheme(old), ui.WithFont(guitest.NewFont(14)))
	inner, a, b := twoButtons(ui.HBox(ui.ThemeGap))
	outer := ui.NewContainer()
	outer.SetLayout(ui.VBox(ui.ThemeGap))
	label := ui.NewLabel("x")
	outer.Add(label)
	outer.Add(inner)
	h.SetContent(outer)
	h.Step()
	if got := gapH(a, b); !approx(got, 8) {
		t.Errorf("inner gap = %v, want 8", got)
	}
	if got := gapV(label, inner); !approx(got, 8) {
		t.Errorf("outer gap = %v, want 8", got)
	}
}

func TestAnyNegativeSpacingMeansTheTheme(t *testing.T) {
	h := guitest.New(300, 300)
	root, a, b := twoButtons(ui.HBox(-7))
	h.SetContent(root)
	h.Step()
	if got := gapH(a, b); !approx(got, 8) {
		t.Errorf("HBox(-7) gap = %v, want the theme's 8", got)
	}
}
