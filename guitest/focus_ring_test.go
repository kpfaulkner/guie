package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// Every focusable control marks keyboard focus the same way: a 2px stroke in
// the theme's accent colour, and nothing of the kind while unfocused.
func TestFocusRingIsConsistentAcrossControls(t *testing.T) {
	accent := theme.DefaultPalette().Accent
	cases := map[string]func() ui.Widget{
		"button":    func() ui.Widget { return ui.NewButton("b") },
		"flat":      func() ui.Widget { return ui.NewButton("b", ui.ButtonFlat()) },
		"checkbox":  func() ui.Widget { return ui.NewCheckbox("c") },
		"textfield": func() ui.Widget { return ui.NewTextField() },
		"textarea":  func() ui.Widget { return ui.NewTextArea() },
		"dropdown":  func() ui.Widget { return ui.NewDropdown([]string{"a", "b"}) },
		"radio": func() ui.Widget {
			return ui.NewRadioButton("r", ui.NewRadioGroup())
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			h := guitest.New(240, 120)
			w := mk()
			h.SetContent(w)
			h.MoveMouse(-1, -1)

			if hasRing(h.Step(), accent) {
				t.Fatal("unfocused control should not draw a focus ring")
			}
			w.(interface{ RequestFocus() }).RequestFocus()
			h.Step()
			if !hasRing(h.Step(), accent) {
				t.Fatal("focused control should draw a 2px accent ring")
			}
		})
	}
}

func hasRing(rec *guitest.Recording, accent interface{ RGBA() (r, g, b, a uint32) }) bool {
	ar, ag, ab, _ := accent.RGBA()
	for _, k := range []guitest.OpKind{guitest.OpStrokeRoundRect, guitest.OpStrokeCircle} {
		for _, op := range rec.OpsOfKind(k) {
			r, g, b, _ := op.Colour.RGBA()
			if op.Width == 2 && r == ar && g == ag && b == ab {
				return true
			}
		}
	}
	return false
}
