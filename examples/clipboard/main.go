// Command clipboard demonstrates OS clipboard integration. By passing an
// OS-backed clipboard via ui.WithClipboard, Ctrl/Cmd+C / X / V in guie text
// widgets exchange text with other applications (your browser, editor, etc.),
// not just within this app.
//
// The clipboard package is opt-in and lives outside the dependency-free core,
// so apps that don't need system-wide copy/paste pay no extra dependency.
//
// Run with: go run ./examples/clipboard
package main

import (
	"log"

	"github.com/kpfaulkner/guie/clipboard"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	opts := []ui.AppOption{
		ui.WithTitle("guie — OS clipboard"),
		ui.WithSize(520, 260),
	}

	// Try to use the OS clipboard; fall back to the default in-process
	// clipboard if it's unavailable (e.g. no display server).
	status := "OS clipboard active — copy/paste works with other apps."
	if cb, err := clipboard.New(); err != nil {
		status = "OS clipboard unavailable (" + err.Error() + "); using in-process clipboard."
	} else {
		opts = append(opts, ui.WithClipboard(cb))
	}

	app := ui.NewApp(opts...)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(16))

	root.Add(ui.NewLabel("Copy text from another app, then paste here (Ctrl/Cmd+V):"))

	field := ui.NewTextField(ui.Placeholder("Paste here..."))
	root.Add(field)

	root.Add(ui.NewLabel("Type here and copy it into another app (Ctrl/Cmd+C):"))

	area := ui.NewTextArea(ui.TextAreaWrap())
	area.SetText("Select this text and copy it out to your browser or editor.")
	root.Add(area, ui.Weight(1))

	root.Add(ui.NewLabel(status))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
