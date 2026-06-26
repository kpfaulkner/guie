// Command theming demonstrates restyling the whole UI at runtime: swapping the
// colour palette, recolouring the accent, and changing the control corner radius
// (sharp vs. rounded vs. pill). The left column drives the theme; the right card
// is a live preview whose widgets restyle on the next frame.
//
// The key call is app.SetTheme(theme.Theme): standard controls (buttons, fields,
// checkboxes, sliders, dropdowns) resolve their colours, font and corner radius
// from the active theme, so they follow it automatically. A Container's own
// background/border/radius are explicit values, so those are re-applied by hand
// in applyTheme.
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

// namedPalette pairs a label with a palette for the palette picker.
type namedPalette struct {
	name string
	pal  theme.Palette
}

func palettes() []namedPalette {
	return []namedPalette{
		{"Dark", theme.DefaultPalette()},
		{"Light", theme.Palette{
			Background: rgb(0xf4f5f8), Surface: rgb(0xffffff),
			Primary: rgb(0x3b6ea5), OnPrimary: rgb(0xffffff),
			Text: rgb(0x1b1d22), TextMuted: rgb(0x6b6f7a),
			Border: rgb(0xd0d3da), Accent: rgb(0x3b6ea5), Disabled: rgb(0xc8ccd4),
		}},
		{"Ocean", theme.Palette{
			Background: rgb(0x0e1b22), Surface: rgb(0x142a33),
			Primary: rgb(0x2bb3b0), OnPrimary: rgb(0x04201f),
			Text: rgb(0xe6f1f2), TextMuted: rgb(0x88a6ab),
			Border: rgb(0x245058), Accent: rgb(0x53d6d2), Disabled: rgb(0x2a4750),
		}},
		{"Forest", theme.Palette{
			Background: rgb(0x14190f), Surface: rgb(0x1e2716),
			Primary: rgb(0x6fae4a), OnPrimary: rgb(0x0e1707),
			Text: rgb(0xecf2e4), TextMuted: rgb(0x9dae8e),
			Border: rgb(0x3a4a2c), Accent: rgb(0x8fd36a), Disabled: rgb(0x3c4a2e),
		}},
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
		{"Default", color.RGBA{}}, // zero value => keep the palette's own accent
		{"Blue", rgb(0x4a6fa5)}, {"Violet", rgb(0x8a6fe0)},
		{"Rose", rgb(0xe06f9a)}, {"Amber", rgb(0xe0a24a)}, {"Green", rgb(0x4aae6f)},
	}
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — theming"),
		ui.WithSize(760, 520),
	)

	// --- Current theme state, composed from three pickers. ---
	pals, corns, accs := palettes(), corners(), accents()
	curPalette := pals[0].pal // Dark
	curRadius := 6.0          // Soft
	var curAccent *color.RGBA // nil => use the palette's own accent

	// --- The live preview card (its widgets follow the theme automatically). ---
	caption := ui.NewLabel("Muted caption text")
	preview := ui.NewContainer()
	preview.SetLayout(ui.VBox(12))
	preview.SetPadding(geom.UniformInsets(16))
	preview.Add(ui.NewLabel("Preview"))
	preview.Add(ui.NewButton("Primary action"))
	preview.Add(ui.NewButton("Flat button", ui.ButtonFlat()))
	preview.Add(ui.NewTextField(ui.Placeholder("a text field")))
	preview.Add(ui.NewCheckbox("A checkbox"))
	preview.Add(ui.NewSlider(ui.SliderValue(0.4)))
	preview.Add(ui.NewDropdown([]string{"One", "Two", "Three"},
		ui.DropdownPlaceholder("a dropdown")), ui.Align(geom.AlignStart))
	preview.Add(caption)

	// --- The root, whose backdrop also follows the theme. ---
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(16))

	status := ui.NewLabel("")

	// applyTheme rebuilds the theme from the current picker state and pushes it to
	// the app, then re-applies the explicit container styling the theme can't
	// reach on its own.
	applyTheme := func() {
		t := theme.Default()
		t.Palette = curPalette
		if curAccent != nil {
			t.Palette.Primary = *curAccent
			t.Palette.Accent = *curAccent
		}
		t.CornerRadius = curRadius
		app.SetTheme(t)

		// Containers store explicit colours/radius, so restyle them by hand.
		root.SetBackground(t.Palette.Background)
		preview.SetBackground(t.Palette.Surface)
		preview.SetBorder(t.Palette.Border, 1)
		preview.SetCornerRadius(curRadius)
		// LabelColour-style overrides also don't follow the theme; re-apply.
		caption.SetColour(ui.RoleText, t.Palette.TextMuted)
	}

	// --- Pickers (left column). Each button updates state then re-applies. ---
	controls := ui.NewContainer()
	controls.SetLayout(ui.VBox(10))

	controls.Add(ui.NewLabel("Palette"))
	palRow := ui.NewContainer()
	palRow.SetLayout(ui.HBox(8))
	for _, p := range pals {
		p := p
		b := ui.NewButton(p.name)
		b.OnClick(func() {
			curPalette = p.pal
			applyTheme()
			status.SetText("Palette: " + p.name)
		})
		palRow.Add(b)
	}
	controls.Add(palRow)

	controls.Add(ui.NewLabel("Corners"))
	cornRow := ui.NewContainer()
	cornRow.SetLayout(ui.HBox(8))
	for _, c := range corns {
		c := c
		b := ui.NewButton(c.name)
		b.OnClick(func() {
			curRadius = c.radius
			applyTheme()
			status.SetText(fmt.Sprintf("Corners: %s (radius %.0f)", c.name, c.radius))
		})
		cornRow.Add(b)
	}
	controls.Add(cornRow)

	controls.Add(ui.NewLabel("Accent"))
	accRow := ui.NewContainer()
	accRow.SetLayout(ui.HBox(8))
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
			status.SetText("Accent: " + a.name)
		})
		accRow.Add(b)
	}
	controls.Add(accRow)

	controls.Add(ui.NewLabel("← pick a palette, corner style and accent"), ui.Weight(1))

	// --- Assemble: controls beside the preview, status at the bottom. ---
	body := ui.NewContainer()
	body.SetLayout(ui.HBox(16))
	body.Add(controls, ui.Weight(1))
	body.Add(preview, ui.Weight(1))

	root.Add(ui.NewLabel("Runtime theming: restyle the whole UI with app.SetTheme"))
	root.Add(body, ui.Weight(1))
	root.Add(status)

	app.SetContent(root)
	applyTheme() // style everything once before the first frame

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
