// Command fonts demonstrates adjusting both font type and size at runtime. A
// dropdown picks the font family and the A-/A+ buttons change the size; each
// change rebuilds the font from the selected family's TTF and calls
// App.SetFont, which re-lays out the whole UI. One label uses a fixed
// per-widget font to show overrides are independent of the global font.
//
// The font families come from golang.org/x/image/font/gofont (loaded via
// ui.LoadFontBytes) — application code never imports the rendering backend.
//
// Run with: go run ./examples/fonts
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
)

type family struct {
	name string
	ttf  []byte
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — fonts"),
		ui.WithSize(600, 360),
	)

	families := []family{
		{"Go Regular", goregular.TTF},
		{"Go Mono", gomono.TTF},
		{"Go Bold", gobold.TTF},
		{"Go Italic", goitalic.TTF},
		{"Go Smallcaps", gosmallcaps.TTF},
	}

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	familyIdx := 0
	size := 16.0
	info := ui.NewLabel("")

	apply := func() {
		face, err := ui.LoadFontBytes(families[familyIdx].ttf, size)
		if err != nil {
			log.Printf("load font: %v", err)
			return
		}
		app.SetFont(face) // global font: family + size
		info.SetText(fmt.Sprintf("%s, size %.0f", families[familyIdx].name, size))
	}

	// Row 1: font-family dropdown.
	names := make([]string, len(families))
	for i, f := range families {
		names[i] = f.name
	}
	pickerRow := ui.NewContainer()
	pickerRow.SetLayout(ui.HBox(10))
	pickerRow.Add(ui.NewLabel("Font:"))
	picker := ui.NewDropdown(names, ui.DropdownSelected(0))
	picker.OnSelect(func(i int) {
		familyIdx = i
		apply()
	})
	pickerRow.Add(picker, ui.Align(geom.AlignStart))
	root.Add(pickerRow, ui.Align(geom.AlignStart))

	// Row 2: size stepper.
	button := func(label string, fn func()) *ui.Button {
		b := ui.NewButton(label)
		b.OnClick(fn)
		return b
	}
	sizeRow := ui.NewContainer()
	sizeRow.SetLayout(ui.HBox(10))
	sizeRow.Add(button("A-", func() {
		if size > 8 {
			size -= 2
			apply()
		}
	}))
	sizeRow.Add(button("A+", func() {
		if size < 48 {
			size += 2
			apply()
		}
	}))
	sizeRow.Add(info)
	root.Add(sizeRow, ui.Align(geom.AlignStart))

	// These pick up the global font, so they change with the dropdown/stepper.
	root.Add(ui.NewLabel("The quick brown fox jumps over the lazy dog."))
	root.Add(ui.NewTextField(ui.Placeholder("type here — text uses the chosen font")))

	// A label with a fixed per-widget font: unaffected by the global choice.
	fixed := ui.NewLabel("This line is pinned at Go Regular 24.")
	fixed.SetFont(ui.DefaultFont(24))
	root.Add(fixed, ui.Weight(1))

	app.SetContent(root)
	apply() // sync everything to the initial family/size

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
