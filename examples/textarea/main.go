// Command textarea demonstrates the multi-line TextArea widget: type across
// multiple lines (Enter for a new line), navigate with the arrow keys, and
// scroll with the wheel when the text outgrows the box. Soft word-wrap is on,
// so long lines wrap to the width. Select with click+drag or Shift+arrows, and
// cut/copy/paste with Ctrl+X / Ctrl+C / Ctrl+V (Ctrl+A selects all). A status
// line reports the line and character counts as you edit.
//
// Run with: go run ./examples/textarea
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — textarea"),
		ui.WithSize(560, 400),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))

	root.Add(ui.NewLabel("Notes (multi-line; Enter for a new line):"))

	stats := ui.NewLabel("0 lines, 0 chars")
	update := func(s string) {
		lines := strings.Count(s, "\n") + 1
		stats.SetText(fmt.Sprintf("%d lines, %d chars", lines, len(s)))
	}

	area := ui.NewTextArea(
		ui.TextAreaWrap(),
		ui.TextAreaPlaceholder("Start typing your notes here..."),
	)
	area.OnChange(update)
	area.SetText("The quick brown fox jumps over the lazy dog, and keeps on running well past the right edge so you can see soft word-wrap in action.\nSecond paragraph.")
	update(area.Text())

	root.Add(area, ui.Weight(1))
	root.Add(stats)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
