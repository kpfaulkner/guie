package guitest_test

import (
	"fmt"
	"image/color"
	"sort"
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// rgba8 is a colour as 8-bit channels, comparable and printable.
type rgba8 [4]uint8

func to8(c color.Color) rgba8 {
	r, g, b, a := c.RGBA()
	return rgba8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func (c rgba8) String() string { return fmt.Sprintf("#%02x%02x%02x/%d", c[0], c[1], c[2], c[3]) }

// paletteOf builds a palette whose every entry is distinct and which shares no
// colour with the palette of any other seed, so a colour that turns up under two
// seeds cannot have come from the palette.
func paletteOf(seed int) theme.Palette {
	c := func(i int) color.Color {
		// Spread across the cube, away from black and white, and different per seed.
		n := seed*61 + i*17
		return color.RGBA{R: uint8(40 + (n*37)%170), G: uint8(40 + (n*91)%170), B: uint8(40 + (n*53)%170), A: 255}
	}
	return theme.Palette{
		Background: c(0), Surface: c(1), Primary: c(2), OnPrimary: c(3), Text: c(4),
		TextMuted: c(5), Border: c(6), Accent: c(7), Disabled: c(8),
		Success: c(9), Warning: c(10), Error: c(11), Info: c(12), Scrim: color.RGBA{A: 0x90},
	}
}

// scrimOK lists colours that are allowed to be the same under every theme: a dim
// over the page is "black at some opacity", and the modal case uses the same alpha
// in both themes below, so it is covered by the transparent-black rule.
func allowed(c rgba8) bool {
	if c[3] == 0 {
		return true // fully transparent: draws nothing
	}
	if c[0] == 0 && c[1] == 0 && c[2] == 0 && c[3] < 255 {
		return true // black at partial opacity: shadows and the scrim
	}
	return false
}

type auditCase struct {
	name string
	run  func(h *guitest.Harness, step func())
}

func content(h *guitest.Harness, w ui.Widget) { h.SetContent(w); h.MoveMouse(-1, -1) }

// auditCases is one entry per widget (and per state worth distinguishing). Each draws
// frames through step, which records the colours of every one.
func auditCases() []auditCase {
	focus := func(w ui.Widget) { w.(interface{ RequestFocus() }).RequestFocus() }
	rows := func(n int) []string {
		s := make([]string, n)
		for i := range s {
			s[i] = fmt.Sprintf("row %d", i)
		}
		return s
	}
	return []auditCase{
		{"label", func(h *guitest.Harness, step func()) { content(h, ui.NewLabel("hello")); step() }},
		{"button", func(h *guitest.Harness, step func()) {
			b := ui.NewButton("ok")
			content(h, b)
			step()
			h.MoveMouse(20, 12) // hover, then press
			step()
			h.PressMouse(0)
			step()
			focus(b)
			step()
		}},
		{"flat button", func(h *guitest.Harness, step func()) {
			b := ui.NewButton("ok", ui.ButtonFlat())
			content(h, b)
			h.MoveMouse(20, 12)
			step()
			h.PressMouse(0)
			step()
		}},
		{"disabled button", func(h *guitest.Harness, step func()) {
			b := ui.NewButton("ok")
			b.SetEnabled(false)
			content(h, b)
			step()
		}},
		{"checkbox", func(h *guitest.Harness, step func()) {
			c := ui.NewCheckbox("c")
			c.SetChecked(true)
			content(h, c)
			step()
			focus(c)
			step()
		}},
		{"radio", func(h *guitest.Harness, step func()) {
			g := ui.NewRadioGroup()
			r := ui.NewRadioButton("r", g)
			content(h, r)
			step()
			focus(r)
			step()
		}},
		{"textfield", func(h *guitest.Harness, step func()) {
			tf := ui.NewTextField(ui.Placeholder("type"))
			content(h, tf)
			step()
			tf.SetText("hello world")
			focus(tf)
			h.PressKey(65) // any key; selection below
			step()
		}},
		{"masked textfield", func(h *guitest.Harness, step func()) {
			tf := ui.NewTextField(ui.Masked())
			tf.SetText("secret")
			content(h, tf)
			step()
		}},
		{"textarea", func(h *guitest.Harness, step func()) {
			ta := ui.NewTextArea(ui.TextAreaPlaceholder("notes"))
			content(h, ta)
			step()
			ta.SetText("one\ntwo\nthree")
			focus(ta)
			ta.SelectRange(1, 6)
			step()
		}},
		{"slider", func(h *guitest.Harness, step func()) {
			s := ui.NewSlider()
			content(h, s)
			step()
			focus(s)
			step()
		}},
		{"stepper", func(h *guitest.Harness, step func()) {
			s := ui.NewStepper()
			content(h, s)
			step()
			h.MoveMouse(100, 8)
			step()
			focus(s)
			step()
		}},
		{"progress bar", func(h *guitest.Harness, step func()) { content(h, ui.NewProgressBar(0.4)); step() }},
		{"spinner", func(h *guitest.Harness, step func()) { content(h, ui.NewSpinner()); step() }},
		{"list", func(h *guitest.Harness, step func()) {
			l := ui.NewList(rows(30))
			content(h, l)
			step()
			l.SetSelected(2)
			h.MoveMouse(30, 50)
			step()
			focus(l)
			step()
		}},
		{"tree", func(h *guitest.Harness, step func()) {
			root := ui.NewTreeNode("root", ui.NewTreeNode("a"), ui.NewTreeNode("b"))
			tr := ui.NewTree(root)
			content(h, tr)
			tr.Expand(root)
			tr.SetSelected(root.Children()[0])
			step()
			h.MoveMouse(30, 40)
			step()
		}},
		{"table", func(h *guitest.Harness, step func()) {
			tb := ui.NewTable([]ui.Column{{Title: "A"}, {Title: "B"}})
			tb.SetStriped(true)
			for i := 0; i < 8; i++ {
				tb.AddRow(fmt.Sprint(i), "x")
			}
			content(h, tb)
			tb.SetSelected(1)
			step()
			h.MoveMouse(30, 60)
			step()
			h.MoveMouse(30, 8) // header hover
			step()
		}},
		{"dropdown open", func(h *guitest.Harness, step func()) {
			d := ui.NewDropdown([]string{"a", "b", "c"})
			content(h, d)
			step()
			h.Click(40, 12)
			step()
		}},
		{"menu bar open", func(h *guitest.Harness, step func()) {
			bar := ui.NewMenuBar()
			bar.AddMenu("File", ui.NewMenuItem("Open", func() {}), ui.NewMenuItem("Save", func() {}))
			content(h, bar)
			step()
			h.Click(20, 10)
			step()
			h.MoveMouse(30, 36)
			step()
		}},
		{"tabs", func(h *guitest.Harness, step func()) {
			tc := ui.NewTabContainer()
			tc.AddTab("One", ui.NewLabel("1"))
			tc.AddTab("Two", ui.NewLabel("2"))
			content(h, tc)
			step()
			h.MoveMouse(80, 10)
			step()
			focus(tc)
			step()
		}},
		{"split pane", func(h *guitest.Harness, step func()) {
			content(h, ui.HSplit(ui.NewLabel("a"), ui.NewLabel("b")))
			step()
		}},
		{"scroll view", func(h *guitest.Harness, step func()) {
			sv := ui.NewScrollView()
			c := ui.NewContainer()
			c.SetLayout(ui.VBox(4))
			for i := 0; i < 30; i++ {
				c.Add(ui.NewLabel(fmt.Sprint("line ", i)))
			}
			sv.SetContent(c)
			content(h, sv)
			step()
		}},
		{"scroll bar", func(h *guitest.Harness, step func()) {
			sb := ui.NewScrollBar(geom.Vertical)
			sb.SetRange(500, 100)
			content(h, sb)
			step()
		}},
		{"date picker", func(h *guitest.Harness, step func()) {
			dp := ui.NewDatePicker()
			dp.SetValue(time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC))
			content(h, dp)
			step()
			h.MoveMouse(60, 60)
			step()
			focus(dp)
			step()
		}},
		{"date field", func(h *guitest.Harness, step func()) {
			df := ui.NewDateField()
			content(h, df)
			step()
			h.Click(40, 12)
			step()
		}},
		{"tooltip", func(h *guitest.Harness, step func()) {
			b := ui.NewButton("tip")
			b.SetTooltip("a tooltip")
			content(h, b)
			h.MoveMouse(20, 12)
			for i := 0; i < 45; i++ {
				step()
			}
		}},
		{"toasts", func(h *guitest.Harness, step func()) {
			content(h, ui.NewLabel("page"))
			for _, k := range []ui.ToastKind{ui.ToastInfo, ui.ToastSuccess, ui.ToastWarning, ui.ToastError} {
				h.App.ShowToast("message", ui.WithToastKind(k))
			}
			for i := 0; i < 20; i++ {
				step()
			}
		}},
		{"modal dialog", func(h *guitest.Harness, step func()) {
			content(h, ui.NewLabel("page"))
			h.App.ShowConfirm("Title", "Message", nil)
			step()
		}},
	}
}

// coloursUsed runs one case under a theme and returns every colour it drew.
func coloursUsed(th theme.Theme, c auditCase) map[rgba8]bool {
	h := guitest.New(260, 160, ui.WithTheme(th))
	used := map[rgba8]bool{}
	step := func() {
		for _, op := range h.Step().Ops {
			if op.Colour != nil {
				used[to8(op.Colour)] = true
			}
		}
	}
	c.run(h, step)
	return used
}

// Every colour a widget draws must come from the theme. Draw each widget under two
// themes whose palettes share no colour: anything that turns up under both is a
// literal baked into the widget (or a colour cached before SetTheme), and would
// look the same in every theme. Colours derived from the palette (tints, mixes)
// differ between the two, so they do not trip it.
func TestEveryWidgetColourComesFromTheTheme(t *testing.T) {
	a := theme.Theme{Palette: paletteOf(1), FontSize: 14, CornerRadius: 5}
	b := theme.Theme{Palette: paletteOf(2), FontSize: 14, CornerRadius: 5}
	// The two palettes must really be disjoint, or the test proves nothing.
	pa, pb := map[rgba8]bool{}, map[rgba8]bool{}
	for _, c := range paletteColours(a.Palette) {
		pa[to8(c)] = true
	}
	for _, c := range paletteColours(b.Palette) {
		pb[to8(c)] = true
	}
	for c := range pa {
		if pb[c] {
			t.Fatalf("test palettes share %v", c)
		}
	}

	for _, c := range auditCases() {
		t.Run(c.name, func(t *testing.T) {
			ua, ub := coloursUsed(a, c), coloursUsed(b, c)
			if len(ua) == 0 {
				t.Fatal("the case drew nothing")
			}
			var shared []string
			for col := range ua {
				if ub[col] && !allowed(col) {
					shared = append(shared, col.String())
				}
			}
			sort.Strings(shared)
			if len(shared) > 0 {
				t.Errorf("colours drawn identically under two unrelated themes (hard-coded or cached): %v", shared)
			}
		})
	}
}

func paletteColours(p theme.Palette) []color.Color {
	return []color.Color{p.Background, p.Surface, p.Primary, p.OnPrimary, p.Text, p.TextMuted,
		p.Border, p.Accent, p.Disabled, p.Success, p.Warning, p.Error, p.Info}
}
