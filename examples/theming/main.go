// Command theming demonstrates restyling the whole UI at runtime with
// app.SetTheme(theme.Theme). The left column drives the theme; the right card is a
// live preview whose widgets restyle on the next frame.
//
//   - Theme: the Dark, Light and High contrast presets, or a custom palette
//     (Ocean, Forest) on the default look. High contrast also brings 2 px borders,
//     a 3 px focus ring, stronger hover and pressed shading, no shadows and no
//     fades. Press Tab to walk the preview and see the focus ring.
//   - Corners and Accent: the control corner radius, and a recoloured Primary and
//     Accent.
//   - Density: scales the padding inside every widget (not the text). Spacing: the
//     gap between widgets, which only layouts built with ui.ThemeGap follow, as
//     every row and column here is.
//   - Reduce motion: hover and press jump straight to their end state instead of
//     fading. (Toasts still slide, because that is the application's own motion.)
//   - Try it: a toast of each kind (their fills come from the theme's status
//     colours), and a confirm dialog (its backdrop is the theme's scrim).
//
// Standard controls resolve their colours, font, radius, borders and padding from
// the theme, so they follow it automatically. A Container's own background and
// border are explicit values, so those are re-applied by hand in applyTheme.
//
// Run with: go run ./examples/theming
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// namedTheme pairs a label with the base theme it selects. A preset brings its own
// borders, focus width, state tints and motion; a custom palette sits on the
// default look.
type namedTheme struct {
	name string
	base func() theme.Theme
}

func custom(p theme.Palette) func() theme.Theme {
	return func() theme.Theme {
		t := theme.Default()
		t.Palette = p // status colours and scrim are left nil: the app fills in defaults
		return t
	}
}

func themes() []namedTheme {
	return []namedTheme{
		{"Dark", theme.Dark},
		{"Light", theme.Light},
		{"High contrast", theme.HighContrast},
		{"Ocean", custom(theme.Palette{
			Background: rgb(0x0e1b22), Surface: rgb(0x142a33),
			Primary: rgb(0x2bb3b0), OnPrimary: rgb(0x04201f),
			Text: rgb(0xe6f1f2), TextMuted: rgb(0x88a6ab),
			Border: rgb(0x245058), Accent: rgb(0x53d6d2), Disabled: rgb(0x2a4750),
		})},
		{"Forest", custom(theme.Palette{
			Background: rgb(0x14190f), Surface: rgb(0x1e2716),
			Primary: rgb(0x6fae4a), OnPrimary: rgb(0x0e1707),
			Text: rgb(0xecf2e4), TextMuted: rgb(0x9dae8e),
			Border: rgb(0x3a4a2c), Accent: rgb(0x8fd36a), Disabled: rgb(0x3c4a2e),
		})},
	}
}

// cornerStyle pairs a label with a corner radius for the corner picker.
type cornerStyle struct {
	name   string
	radius float64
}

func corners() []cornerStyle {
	return []cornerStyle{
		{"Sharp", 0}, {"Soft", 6}, {"Rounded", 14}, {"Pill", 999},
	}
}

// accentSwatch pairs a label with a colour for the accent picker.
type accentSwatch struct {
	name   string
	colour color.RGBA
}

func accents() []accentSwatch {
	return []accentSwatch{
		{"Default", color.RGBA{}}, // zero value => keep the theme's own accent
		{"Blue", rgb(0x4a6fa5)}, {"Violet", rgb(0x8a6fe0)},
		{"Rose", rgb(0xe06f9a)}, {"Amber", rgb(0xe0a24a)}, {"Green", rgb(0x4aae6f)},
	}
}

// The sliders run 0..1; these map them onto the ranges they control.
const (
	densityMin, densityMax = 0.7, 1.5
	spacingMin, spacingMax = 2.0, 20.0
)

func fromSlider(v, lo, hi float64) float64 { return lo + v*(hi-lo) }
func toSlider(x, lo, hi float64) float64   { return (x - lo) / (hi - lo) }

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie - theming"),
		ui.WithSize(980, 700),
	)

	// --- Current theme state, composed from the pickers. ---
	ths, corns, accs := themes(), corners(), accents()
	curTheme := ths[0]        // Dark
	curRadius := 6.0          // Soft
	var curAccent *color.RGBA // nil => use the theme's own accent
	curDensity := 1.0         // Theme.Density
	curSpacing := 8.0         // Theme.Spacing
	reduceMotion := false     // Theme.Motion.Reduced
	status := ui.NewLabel("") // summary of what is applied
	densityLbl := ui.NewLabel("")
	spacingLbl := ui.NewLabel("")

	// --- The live preview card. Everything in it follows the theme, and its own
	// layout uses ThemeGap, so Spacing and Density move its rows apart or together. ---
	caption := ui.NewLabel("Muted caption text")
	list := ui.NewList([]string{"List row one", "List row two", "List row three"})
	list.SetSelected(1)
	table := ui.NewTable([]ui.Column{{Title: "Name", Weight: 2}, {Title: "Role"}})
	table.SetStriped(true)
	table.SetRows([][]string{{"Ada", "Engineer"}, {"Bob", "Designer"}, {"Cy", "Manager"}})
	table.SetSelected(0)

	preview := ui.NewContainer()
	preview.SetLayout(ui.VBox(ui.ThemeGap))
	preview.SetPadding(geom.UniformInsets(16))
	preview.Add(ui.NewLabel("Preview (press Tab to see focus)"))
	preview.Add(ui.NewButton("Primary action"))
	preview.Add(ui.NewButton("Flat button", ui.ButtonFlat()))
	preview.Add(ui.NewTextField(ui.Placeholder("a text field")))
	preview.Add(ui.NewCheckbox("A checkbox"))
	preview.Add(ui.NewSlider(ui.SliderValue(0.4)))
	preview.Add(ui.NewStepper(ui.StepperRange(0, 10), ui.StepperValue(3)), ui.Align(geom.AlignStart))
	preview.Add(ui.NewDropdown([]string{"One", "Two", "Three"},
		ui.DropdownPlaceholder("a dropdown")), ui.Align(geom.AlignStart))
	preview.Add(list, ui.Weight(1))
	preview.Add(table, ui.Weight(1))
	preview.Add(caption)

	// --- The root, whose backdrop also follows the theme. ---
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(ui.ThemeGap))
	root.SetPadding(geom.UniformInsets(16))

	// applyTheme rebuilds the theme from the picker state and pushes it to the app,
	// then re-applies the explicit container styling the theme cannot reach.
	applyTheme := func() {
		t := curTheme.base()
		if curAccent != nil {
			t.Palette.Primary = *curAccent
			t.Palette.Accent = *curAccent
		}
		t.CornerRadius = curRadius
		t.Density = curDensity
		t.Spacing = curSpacing
		if reduceMotion {
			t.Motion.Reduced = true // a preset that already reduces motion keeps it
		}
		app.SetTheme(t)
		t = app.Theme() // the app's copy has every optional field filled in

		// Containers store explicit colours/radius, so restyle them by hand.
		root.SetBackground(t.Palette.Background)
		preview.SetBackground(t.Palette.Surface)
		preview.SetBorder(t.Palette.Border, t.BorderWidth)
		preview.SetCornerRadius(curRadius)
		// Per-widget overrides do not follow the theme either; re-apply.
		caption.SetColour(ui.RoleText, t.Palette.TextMuted)

		densityLbl.SetText(fmt.Sprintf("Density %.2f (padding inside widgets)", curDensity))
		spacingLbl.SetText(fmt.Sprintf("Spacing %.0f (gap between widgets, ThemeGap)", curSpacing))
		status.SetText(fmt.Sprintf("%s | border %.0f focus %.0f | motion %s",
			curTheme.name, t.BorderWidth, t.FocusWidth,
			map[bool]string{true: "reduced", false: "on"}[t.Motion.Reduced]))
	}

	// --- Pickers (left column). Each updates state, then re-applies. ---
	controls := ui.NewContainer()
	controls.SetLayout(ui.VBox(ui.ThemeGap))

	controls.Add(ui.NewLabel("Theme"))
	thRow := ui.NewContainer()
	thRow.SetLayout(ui.HBox(ui.ThemeGap))
	for _, p := range ths {
		p := p
		b := ui.NewButton(p.name)
		b.OnClick(func() { curTheme = p; applyTheme() })
		thRow.Add(b)
	}
	controls.Add(thRow)

	controls.Add(ui.NewLabel("Corners"))
	cornRow := ui.NewContainer()
	cornRow.SetLayout(ui.HBox(ui.ThemeGap))
	for _, c := range corns {
		c := c
		b := ui.NewButton(c.name)
		b.OnClick(func() { curRadius = c.radius; applyTheme() })
		cornRow.Add(b)
	}
	controls.Add(cornRow)

	controls.Add(ui.NewLabel("Accent"))
	accRow := ui.NewContainer()
	accRow.SetLayout(ui.HBox(ui.ThemeGap))
	for _, a := range accs {
		a := a
		b := ui.NewButton(a.name)
		b.OnClick(func() {
			if a.name == "Default" {
				curAccent = nil
			} else {
				col := a.colour
				curAccent = &col
			}
			applyTheme()
		})
		accRow.Add(b)
	}
	controls.Add(accRow)

	// Density and spacing: Theme.Density scales padding inside widgets; Theme.Spacing
	// is the gap ThemeGap resolves to (then scaled by Density).
	controls.Add(densityLbl)
	density := ui.NewSlider(ui.SliderValue(toSlider(curDensity, densityMin, densityMax)))
	density.OnChange(func(v float64) { curDensity = fromSlider(v, densityMin, densityMax); applyTheme() })
	controls.Add(density)

	controls.Add(spacingLbl)
	spacing := ui.NewSlider(ui.SliderValue(toSlider(curSpacing, spacingMin, spacingMax)))
	spacing.OnChange(func(v float64) { curSpacing = fromSlider(v, spacingMin, spacingMax); applyTheme() })
	controls.Add(spacing)

	reduce := ui.NewCheckbox("Reduce motion (no hover or press fades)")
	reduce.OnChange(func(on bool) { reduceMotion = on; applyTheme() })
	controls.Add(reduce)

	// Try it: toasts take their fills from the status colours, the dialog's backdrop
	// is the scrim, and everything has a focus ring to look at.
	controls.Add(ui.NewLabel("Try it: a toast of each kind, and a dialog"))
	toastRow := ui.NewContainer()
	toastRow.SetLayout(ui.HBox(ui.ThemeGap))
	for _, k := range []struct {
		label string
		kind  ui.ToastKind
	}{
		{"Info", ui.ToastInfo}, {"Success", ui.ToastSuccess},
		{"Warning", ui.ToastWarning}, {"Error", ui.ToastError},
	} {
		k := k
		b := ui.NewButton(k.label)
		b.OnClick(func() { app.ShowToast(k.label+" toast raised", ui.WithToastKind(k.kind)) })
		toastRow.Add(b)
	}
	controls.Add(toastRow)

	dialog := ui.NewButton("Confirm dialog...")
	dialog.OnClick(func() {
		app.ShowConfirm("Apply this theme?", "The backdrop behind this dialog is the theme's scrim.",
			func(ok bool) { app.ShowToast(fmt.Sprintf("Confirmed: %v", ok)) },
			ui.ConfirmLabels("Apply", "Cancel"), ui.ConfirmDefaultCancel())
	})
	controls.Add(dialog, ui.Align(geom.AlignStart))

	// --- Assemble: controls beside the preview, status at the bottom. ---
	body := ui.NewContainer()
	body.SetLayout(ui.HBox(16))
	// The controls scroll too, both ways: at the largest density and spacing their rows of
	// buttons are wider and taller than the window.
	controlScroll := ui.NewScrollView()
	controlScroll.SetHorizontalScroll(true)
	controlScroll.SetContent(controls)
	body.Add(controlScroll, ui.Weight(3))
	// The preview scrolls: at the largest density and spacing it is taller than the window.
	previewScroll := ui.NewScrollView()
	previewScroll.SetHorizontalScroll(true) // so the 3:2 split below is honoured, not its content width
	previewScroll.SetContent(preview)
	body.Add(previewScroll, ui.Weight(2))

	root.Add(ui.NewLabel("Runtime theming: restyle the whole UI with app.SetTheme"))
	root.Add(body, ui.Weight(1))
	root.Add(status)

	app.SetContent(root)
	applyTheme() // style everything once before the first frame

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
