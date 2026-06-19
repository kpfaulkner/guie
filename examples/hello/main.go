// Command hello is the smallest possible uiframework program: a window with a
// single centered label.
//
// Run with: go run ./examples/hello
package main

import (
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — hello"),
		ui.WithSize(400, 200),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.NewStack()) // Stack + center → centers its single child
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("Hello, uiframework!"), ui.Align(geom.AlignCenter))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
