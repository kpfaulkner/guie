// Command windowstate demonstrates remembering the window across runs. Move and
// resize the window (or maximize it), close it, and run it again: it reopens
// where you left it. The state is a small JSON file in the user's config
// directory; delete it to start fresh.
//
// Run with: go run ./examples/windowstate
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	// Where the file lives is the application's choice; the framework only reads
	// and writes the path it is given.
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "guie-windowstate-example", "window.json")

	app := ui.NewApp(
		ui.WithTitle("guie - window state"),
		ui.WithSize(520, 320), // used the first time, before anything is saved
		ui.WithResizable(true),
		ui.WithWindowStateFile(path),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(10))
	root.SetPadding(geom.UniformInsets(20))
	root.Add(ui.NewLabel("Move, resize or maximize this window, then close it and run it again."))
	root.Add(ui.NewLabel("State file: " + path))

	live := ui.NewLabel("")
	show := ui.NewButton("Show geometry")
	// App.WindowGeometry is the last restorable geometry (it keeps the normal size
	// while the window is maximized), which is exactly what gets saved on exit.
	// It is read on demand here: polling it from OnFrame would keep the app
	// presenting every frame.
	show.OnClick(func() {
		if g, ok := app.WindowGeometry(); ok {
			live.SetText(fmt.Sprintf("%dx%d at (%d, %d)  maximized=%v", g.Width, g.Height, g.X, g.Y, g.Maximized))
		}
	})
	root.Add(show, ui.Align(geom.AlignStart))
	root.Add(live)

	app.SetContent(root)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
