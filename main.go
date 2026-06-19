package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.New("uiframework demo", 800, 600)

	// Primary window: a label, a click counter button, and a scrollable list.
	win := ui.NewWindow("Widgets", ui.Rect{X: 60, Y: 60, W: 360, H: 320})
	win.Content.Add(ui.NewLabel(ui.Rect{X: 16, Y: 12, W: 300, H: 20}, "Welcome to uiframework!"))

	status := ui.NewLabel(ui.Rect{X: 150, Y: 52, W: 200, H: 20}, "clicks: 0")
	clicks := 0
	btn := ui.NewButton(ui.Rect{X: 16, Y: 44, W: 120, H: 32}, "Click me")
	btn.OnClick = func() {
		clicks++
		status.Text = fmt.Sprintf("clicks: %d", clicks)
	}
	win.Content.Add(btn)
	win.Content.Add(status)

	// A scroll view whose content is taller than the viewport.
	sv := ui.NewScrollView(ui.Rect{X: 16, Y: 88, W: 320, H: 180})
	list := ui.NewContainer(ui.Rect{X: 0, Y: 0, W: 320, H: 30 * 24})
	for i := 0; i < 30; i++ {
		list.Add(ui.NewLabel(ui.Rect{X: 8, Y: 8 + i*24, W: 280, H: 20}, fmt.Sprintf("Item %d (scroll me)", i+1)))
	}
	sv.SetContent(list)
	win.Content.Add(sv)
	app.AddWindow(win)

	// A second, draggable window to show window stacking and focus.
	win2 := ui.NewWindow("Drag me", ui.Rect{X: 460, Y: 140, W: 260, H: 160})
	win2.Content.Add(ui.NewLabel(ui.Rect{X: 16, Y: 16, W: 240, H: 20}, "Drag my title bar."))
	win2.Content.Add(ui.NewLabel(ui.Rect{X: 16, Y: 40, W: 240, H: 20}, "Click to raise me."))
	app.AddWindow(win2)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
