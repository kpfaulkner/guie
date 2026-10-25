// Command textarea demonstrates the multi-line TextArea widget: type across
// multiple lines (Enter for a new line), navigate with the arrow keys, and
// scroll with the wheel when the text outgrows the box. Soft word-wrap is on,
// so long lines wrap to the width. Select with click+drag or Shift+arrows, and
// cut/copy/paste with Ctrl+X / Ctrl+C / Ctrl+V (Ctrl+A selects all). A status
// line reports the line and character counts as you edit.
//
// Ctrl+Z undoes and Ctrl+Shift+Z (or Ctrl+Y) redoes, and a double-click selects a
// word. Untick "Word wrap" and the long lines scroll sideways instead: a bar
// appears along the bottom, Shift+wheel scrolls across, and the view follows the
// caret as you type or move along a line.
//
// Run with: go run ./examples/textarea
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — textarea"),
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
	area.SetText("The quick brown fox jumps over the lazy dog, and keeps on running well past the right edge so you can see soft word-wrap in action.\nSecond paragraph.\nA line with no break points: " + strings.Repeat("0123456789", 14))
	update(area.Text())

	wrap := ui.NewCheckbox("Word wrap")
	wrap.SetChecked(true)
	wrap.OnChange(func(on bool) { area.SetWrap(on) })
	root.Add(wrap)
	root.Add(area, ui.Weight(1))
	root.Add(stats)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
