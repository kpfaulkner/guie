// Command comprehensive is a single-window tour of (nearly) the entire guie
// feature set, intended as a general-purpose demo and a living reference for the
// public API.
//
// It exercises:
//   - a MenuBar (File / View / Help) with actions, including a modal "About"
//   - a TabContainer switching between five panes
//   - Controls:  buttons (primary / flat / disabled / image), checkboxes,
//     a radio group, a slider driving a ProgressBar, a dropdown, and two kinds
//     of modal dialog (ShowMessage and a custom ShowModal prompt)
//   - Text:      a TextField (placeholder + live echo + submit) and a wrapping
//     TextArea, with buttons that read/clear them
//   - Data:      a Table and a List inside a horizontal SplitPane
//   - Layout:    a Grid with a spanning cell and a ScrollView, in a vertical
//     SplitPane; plus a Stack for centering
//   - Media:     a generated image (FitContain) and the theme palette as swatches
//   - cross-cutting: a custom theme, per-widget color overrides, tooltips, a
//     status bar, and a global click counter wired through the event bus
//
// Run with: go run ./examples/comprehensive
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	// A custom theme: start from the default dark palette and tweak the accent
	// colors and corner rounding. Widgets pick these up automatically.
	th := theme.Default()
	th.Palette.Primary = color.RGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
	th.Palette.Accent = color.RGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff}
	th.CornerRadius = 6
	pal := th.Palette

	app := ui.NewApp(
		ui.WithTitle("guie — comprehensive demo"),
		ui.WithSize(960, 680),
		ui.WithResizable(true),
		ui.WithTheme(th),
		ui.WithShadows(true),
	)

	// Shared status line + a tiny reporting helper used throughout.
	status := ui.NewLabel("Ready.")
	say := func(s string) { status.SetText(s) }

	// Global click counter, driven entirely off the event bus: every derived
	// Click in the whole UI is published, so this updates without wiring each
	// widget. Bus handlers run on the UI goroutine, so touching a widget is safe.
	clicks := 0
	clickLabel := ui.NewLabel("clicks: 0", ui.LabelColor(pal.TextMuted))
	app.Events().Subscribe(ui.EventClick, func(ev ui.Event) {
		clicks++
		clickLabel.SetText(fmt.Sprintf("clicks: %d", clicks))
	})

	// A generated PNG, decoded through the public image loader, reused by the
	// image button and the Media tab.
	swatch, err := ui.LoadImageBytes(swatchPNG())
	if err != nil {
		log.Fatal(err)
	}

	// --- Menu bar ---------------------------------------------------------
	bar := ui.NewMenuBar()
	bar.AddMenu("File",
		ui.NewMenuItem("New", func() { say("File ▸ New") }),
		ui.NewMenuItem("Open…", func() { say("File ▸ Open") }),
		ui.NewMenuItem("Save", func() { say("File ▸ Save") }),
		ui.NewMenuItem("Quit", func() { app.Quit() }),
	)
	shadows := true
	bar.AddMenu("View",
		ui.NewMenuItem("Toggle shadows", func() {
			shadows = !shadows
			app.SetShadows(shadows)
			say(fmt.Sprintf("Shadows: %v", shadows))
		}),
	)
	bar.AddMenu("Help",
		ui.NewMenuItem("About", func() {
			app.ShowMessage("About",
				"guie comprehensive demo\nA cross-platform Go UI toolkit.",
				ui.DialogButton{Label: "Nice"})
		}),
	)

	// --- Tabs -------------------------------------------------------------
	tabs := ui.NewTabContainer()
	tabs.OnChange(func(i int) { say(fmt.Sprintf("Switched to tab %d", i)) })
	tabs.AddTab("Controls", controlsTab(app, say, pal, swatch))
	tabs.AddTab("Text", textTab(say, pal))
	tabs.AddTab("Data", dataTab(say, pal))
	tabs.AddTab("Layout", layoutTab(pal))
	tabs.AddTab("Media", mediaTab(pal, swatch))

	// --- Status bar -------------------------------------------------------
	statusBar := ui.NewContainer()
	statusBar.SetBackground(pal.Surface)
	statusBar.SetLayout(ui.HBox(12))
	statusBar.SetPadding(geom.UniformInsets(8))
	statusBar.Add(status, ui.Weight(1))
	statusBar.Add(clickLabel, ui.Align(geom.AlignEnd))

	// --- Root: menu bar / tabs (fill) / status bar ------------------------
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(bar)
	root.Add(tabs, ui.Weight(1))
	root.Add(statusBar)

	app.SetContent(root)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// controlsTab demonstrates buttons, checkboxes, radios, a slider→progress link,
// a dropdown, tooltips, color overrides, and both dialog styles.
func controlsTab(app *ui.App, say func(string), pal theme.Palette, swatch render.Image) ui.Widget {
	// Buttons: primary, flat, disabled, and an image button.
	primary := ui.NewButton("Primary")
	primary.OnClick(func() { say("Primary clicked") })
	primary.SetTooltip("A normal raised button")

	flat := ui.NewButton("Flat", ui.ButtonFlat())
	flat.OnClick(func() { say("Flat clicked") })

	disabled := ui.NewButton("Disabled")
	disabled.SetEnabled(false)

	custom := ui.NewButton("Custom color")
	custom.SetColor(ui.RolePrimary, color.RGBA{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff})
	custom.OnClick(func() { say("Custom recolored button") })

	imgBtn := ui.NewButton("Image", ui.ButtonImage(swatch))
	imgBtn.OnClick(func() { say("Image button clicked") })

	buttons := card("Buttons", pal,
		row(8, primary, flat, disabled),
		row(8, custom, imgBtn),
	)

	// Checkboxes + radio group.
	cb1 := ui.NewCheckbox("Enable feature", ui.Checked(true))
	cb1.OnChange(func(v bool) { say(fmt.Sprintf("Feature: %v", v)) })
	cb2 := ui.NewCheckbox("Verbose logging")
	cb2.OnChange(func(v bool) { say(fmt.Sprintf("Verbose: %v", v)) })

	group := ui.NewRadioGroup()
	group.OnChange(func(i int) { say(fmt.Sprintf("Size option %d", i)) })
	r1 := ui.NewRadioButton("Small", group)
	r2 := ui.NewRadioButton("Medium", group)
	r3 := ui.NewRadioButton("Large", group)
	r2.Select()

	dd := ui.NewDropdown(
		[]string{"Red", "Orange", "Yellow", "Green", "Blue", "Violet"},
		ui.DropdownPlaceholder("choose a color"),
	)
	dd.OnSelect(func(i int) { say(fmt.Sprintf("Dropdown index %d", i)) })

	selection := card("Selection", pal,
		cb1, cb2,
		ui.NewLabel("Size:", ui.LabelColor(pal.TextMuted)),
		r1, r2, r3,
		dd,
	)

	// Slider drives a ProgressBar and a percentage label live.
	prog := ui.NewProgressBar(0.4)
	pct := ui.NewLabel("40%")
	slider := ui.NewSlider(ui.SliderValue(0.4))
	slider.SetTooltip("Drag, or focus and use the arrow keys")
	slider.OnChange(func(v float64) {
		prog.SetValue(v)
		pct.SetText(fmt.Sprintf("%.0f%%", v*100))
	})
	sliderCard := card("Slider → progress", pal, slider, prog, pct)

	// Dialogs: a standard message box and a custom modal prompt.
	msgBtn := ui.NewButton("Message dialog")
	msgBtn.OnClick(func() {
		app.ShowMessage("Save changes?", "Your document has unsaved changes.",
			ui.DialogButton{Label: "Discard", OnClick: func() { say("Discarded") }},
			ui.DialogButton{Label: "Save", OnClick: func() { say("Saved") }},
		)
	})
	promptBtn := ui.NewButton("Custom modal")
	promptBtn.OnClick(func() { showPrompt(app, say, pal) })
	dialogs := card("Dialogs", pal, row(8, msgBtn, promptBtn))

	// Two columns side by side.
	left := ui.NewContainer()
	left.SetLayout(ui.VBox(12))
	left.Add(buttons)
	left.Add(sliderCard)
	left.Add(dialogs)

	rightCol := ui.NewContainer()
	rightCol.SetLayout(ui.VBox(12))
	rightCol.Add(selection)

	cols := ui.NewContainer()
	cols.SetLayout(ui.HBox(12))
	cols.SetPadding(geom.UniformInsets(12))
	cols.Add(left, ui.Weight(1))
	cols.Add(rightCol, ui.Weight(1))
	return cols
}

// showPrompt builds a custom modal (ShowModal) with a text field and two
// buttons, closing itself via App.Close.
func showPrompt(app *ui.App, say func(string), pal theme.Palette) {
	panel := ui.NewContainer()
	panel.SetBackground(pal.Surface)
	panel.SetBorder(pal.Border, 1)
	panel.SetCornerRadius(8)
	panel.SetPadding(geom.UniformInsets(16))
	panel.SetLayout(ui.VBox(12))

	panel.Add(ui.NewLabel("What's your name?"))
	field := ui.NewTextField(ui.Placeholder("type here"))
	panel.Add(field)

	var p *ui.Popup
	ok := ui.NewButton("OK")
	cancel := ui.NewButton("Cancel", ui.ButtonFlat())
	ok.OnClick(func() {
		name := field.Text()
		if name == "" {
			name = "stranger"
		}
		say("Hello, " + name)
		app.Close(p)
	})
	cancel.OnClick(func() { app.Close(p) })

	buttons := ui.NewContainer()
	buttons.SetLayout(ui.HBox(8))
	buttons.Add(cancel)
	buttons.Add(ok)
	panel.Add(buttons, ui.Align(geom.AlignEnd))

	p = app.ShowModal(panel)
}

// textTab shows a TextField with live echo + submit and a wrapping TextArea.
func textTab(say func(string), pal theme.Palette) ui.Widget {
	echo := ui.NewLabel("(type above)", ui.LabelColor(pal.TextMuted))
	field := ui.NewTextField(ui.Placeholder("Type something and press Enter"))
	field.OnChange(func(s string) { echo.SetText("echo: " + s) })
	field.OnSubmit(func(s string) { say("Submitted: " + s) })

	area := ui.NewTextArea(ui.TextAreaWrap(), ui.TextAreaPlaceholder("A multi-line, word-wrapping editor…"))
	area.SetText("guie supports a multi-line text area with soft word wrap, " +
		"selection, and clipboard shortcuts (Ctrl/Cmd + C/X/V/A).\n\n" +
		"Try selecting text with the mouse or Shift+arrows.")
	area.OnChange(func(s string) { say(fmt.Sprintf("Text area: %d chars", len(s))) })

	getBtn := ui.NewButton("Report length")
	getBtn.OnClick(func() { say(fmt.Sprintf("Field=%q  Area=%d chars", field.Text(), len(area.Text()))) })
	clearBtn := ui.NewButton("Clear", ui.ButtonFlat())
	clearBtn.OnClick(func() {
		field.SetText("")
		area.SetText("")
		echo.SetText("(cleared)")
	})

	fieldCard := card("TextField", pal, field, echo, row(8, getBtn, clearBtn))

	areaCard := card("TextArea (wrapping)", pal)
	areaCard.Add(area, ui.Weight(1))

	col := ui.NewContainer()
	col.SetLayout(ui.VBox(12))
	col.SetPadding(geom.UniformInsets(12))
	col.Add(fieldCard)
	col.Add(areaCard, ui.Weight(1))
	return col
}

// dataTab shows a Table and a List inside a horizontal SplitPane.
func dataTab(say func(string), pal theme.Palette) ui.Widget {
	tbl := ui.NewTable([]ui.Column{
		{Title: "Name", Weight: 2},
		{Title: "Role", Weight: 2},
		{Title: "Score", Weight: 1},
	})
	tbl.SetRows([][]string{
		{"Ada", "Engineer", "98"},
		{"Linus", "Maintainer", "95"},
		{"Grace", "Admiral", "99"},
		{"Dennis", "Architect", "97"},
		{"Margaret", "Lead", "96"},
	})
	tbl.OnSelect(func(i int) { say(fmt.Sprintf("Table row %d", i)) })
	tableCard := card("Table", pal)
	tableCard.Add(tbl, ui.Weight(1))

	items := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry",
		"Fig", "Grape", "Kiwi", "Lemon", "Mango", "Nectarine", "Orange"}
	list := ui.NewList(items, ui.ListSelected(0))
	list.OnSelect(func(i int) { say("List: " + items[i]) })
	listCard := card("List", pal)
	listCard.Add(list, ui.Weight(1))

	split := ui.HSplit(tableCard, listCard, ui.SplitRatio(0.62), ui.SplitMinSizes(220, 140))

	wrap := ui.NewContainer()
	wrap.SetLayout(ui.VBox(0))
	wrap.SetPadding(geom.UniformInsets(12))
	wrap.Add(split, ui.Weight(1))
	return wrap
}

// layoutTab demonstrates a Grid with a spanning cell and a ScrollView, split
// vertically.
func layoutTab(pal theme.Palette) ui.Widget {
	// Grid: 3 columns; the first cell spans two columns.
	grid := ui.NewContainer()
	grid.SetLayout(ui.NewGrid(3, 8))
	grid.Add(tile("Span 2×1", pal.Primary), ui.Span(2, 1))
	grid.Add(tile("A", pal.Surface))
	grid.Add(tile("B", pal.Surface))
	grid.Add(tile("C", pal.Surface))
	grid.Add(tile("Tall 1×2", pal.Accent), ui.Span(1, 2))
	grid.Add(tile("D", pal.Surface))
	grid.Add(tile("E", pal.Surface))
	grid.Add(tile("F", pal.Surface))
	gridCard := card("Grid (with cell spans)", pal)
	gridCard.Add(grid, ui.Weight(1))

	// ScrollView over a tall column.
	inner := ui.NewContainer()
	inner.SetLayout(ui.VBox(4))
	inner.SetPadding(geom.UniformInsets(8))
	for i := 1; i <= 40; i++ {
		inner.Add(ui.NewLabel(fmt.Sprintf("Scrollable row %02d — wheel or drag the thumb", i)))
	}
	sv := ui.NewScrollView()
	sv.SetContent(inner)
	scrollCard := card("ScrollView", pal)
	scrollCard.Add(sv, ui.Weight(1))

	split := ui.VSplit(gridCard, scrollCard, ui.SplitRatio(0.5))

	wrap := ui.NewContainer()
	wrap.SetLayout(ui.VBox(0))
	wrap.SetPadding(geom.UniformInsets(12))
	wrap.Add(split, ui.Weight(1))
	return wrap
}

// mediaTab shows a generated image and the active theme palette as swatches.
func mediaTab(pal theme.Palette, swatch render.Image) ui.Widget {
	img := ui.NewImage(swatch)
	img.SetFit(ui.FitContain)
	imgCard := card("Image (FitContain)", pal)
	imgCard.Add(img, ui.Weight(1))

	// Palette swatches in a grid.
	swatches := ui.NewContainer()
	swatches.SetLayout(ui.NewGrid(3, 8))
	roles := []struct {
		name string
		c    color.Color
	}{
		{"Background", pal.Background}, {"Surface", pal.Surface}, {"Primary", pal.Primary},
		{"OnPrimary", pal.OnPrimary}, {"Text", pal.Text}, {"TextMuted", pal.TextMuted},
		{"Border", pal.Border}, {"Accent", pal.Accent}, {"Disabled", pal.Disabled},
	}
	for _, r := range roles {
		swatches.Add(tile(r.name, r.c))
	}
	swatchCard := card("Theme palette", pal)
	swatchCard.Add(swatches, ui.Weight(1))

	split := ui.HSplit(imgCard, swatchCard, ui.SplitRatio(0.45))

	wrap := ui.NewContainer()
	wrap.SetLayout(ui.VBox(0))
	wrap.SetPadding(geom.UniformInsets(12))
	wrap.Add(split, ui.Weight(1))
	return wrap
}

// --- small helpers --------------------------------------------------------

// card wraps body widgets in a titled, bordered, rounded surface panel.
func card(title string, pal theme.Palette, body ...ui.Widget) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(pal.Surface)
	c.SetBorder(pal.Border, 1)
	c.SetCornerRadius(6)
	c.SetPadding(geom.UniformInsets(12))
	c.SetLayout(ui.VBox(8))
	c.Add(ui.NewLabel(title, ui.LabelColor(pal.TextMuted)))
	for _, w := range body {
		c.Add(w)
	}
	return c
}

// row lays widgets out horizontally with the given spacing.
func row(spacing float64, ws ...ui.Widget) *ui.Container {
	c := ui.NewContainer()
	c.SetLayout(ui.HBox(spacing))
	for _, w := range ws {
		c.Add(w)
	}
	return c
}

// tile is a colored cell with a centered label, used for grid demos.
func tile(text string, bg color.Color) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetCornerRadius(4)
	c.SetLayout(ui.NewStack())
	c.Add(ui.NewLabel(text, ui.LabelAlign(geom.AlignCenter)), ui.Align(geom.AlignCenter))
	return c
}

// swatchPNG generates a small gradient PNG so the demo needs no asset files.
func swatchPNG() []byte {
	const s = 96
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / s), G: uint8(y * 255 / s), B: 0xb0, A: 0xff})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
