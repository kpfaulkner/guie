// Command tabs demonstrates the TabContainer: a tab strip that switches between
// panes. Click a tab title (or focus the tabs and use Left/Right) to switch.
// Each pane keeps its own state while hidden.
//
// Run with: go run ./examples/tabs
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

// pane builds a simple padded container with a background and a heading.
func pane(bg color.Color, heading string, body ...ui.Widget) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetLayout(ui.VBox(10))
	c.SetPadding(geom.UniformInsets(16))
	c.Add(ui.NewLabel(heading))
	for _, w := range body {
		c.Add(w)
	}
	return c
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — tabs"),
		ui.WithSize(560, 360),
	)

	tabs := ui.NewTabContainer()

	// Tab 1: a counter button (keeps its count when you switch away and back).
	count := 0
	countLabel := ui.NewLabel("clicks: 0")
	counterBtn := ui.NewButton("Click me", ui.OnClick(func() {
		count++
		countLabel.SetText(fmt.Sprintf("clicks: %d", count))
	}))
	tabs.AddTab("Counter", pane(color.RGBA{0x2b, 0x2b, 0x3a, 0xff}, "A stateful counter:", counterBtn, countLabel))

	// Tab 2: a small form.
	tabs.AddTab("Form", pane(color.RGBA{0x24, 0x30, 0x2c, 0xff}, "A tiny form:",
		ui.NewTextField(ui.Placeholder("your name")),
		ui.NewCheckbox("Subscribe"),
	))

	// Tab 3: static text.
	tabs.AddTab("About", pane(color.RGBA{0x30, 0x28, 0x24, 0xff}, "About",
		ui.NewLabel("TabContainer switches panes via the strip above."),
		ui.NewLabel("Only the active pane draws and gets events."),
	))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.SetPadding(geom.UniformInsets(12))
	root.Add(tabs, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
