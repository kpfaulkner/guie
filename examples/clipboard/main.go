// Command clipboard demonstrates OS clipboard integration: Ctrl/Cmd+C / X / V in
// guie text widgets exchange text with other applications (your browser, editor,
// etc.), not just within this app.
//
// An app gets this for free - ui.NewApp installs an OS-backed clipboard by
// default, falling back to an in-process one where the platform clipboard is
// unavailable. This example wires it up explicitly only so it can *report* that
// fallback in the status line instead of degrading silently.
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
