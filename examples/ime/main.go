// Command ime demonstrates guie's inline IME preedit rendering.
//
// Important: EBiten (the current backend) does not expose a real IME — it only
// delivers *committed* text. So this example drives a *simulated* composition
// through the same EventComposition / EventText path a real backend would use,
// purely so you can see the feature: pressing the button animates a preedit
// ("n" → "ni" → "に" → "にほ" → "にほん") drawn inline and underlined at the
// caret, then commits "日本". Your OS IME's committed output also works if you
// just type into the fields.
//
// Application code does not normally call HandleEvent directly; that is done
// here only to stand in for the backend's missing IME feed. The real preedit
// behavior is covered by the headless tests in the guitest package.
//
// Run with: go run ./examples/ime
package main

import (
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

// imeStep is one beat of the simulated composition: either a preedit update or,
// when commit is set, the accepted text that ends the composition.
type imeStep struct {
	preedit string
	caret   int
	commit  string
}

var script = []imeStep{
	{preedit: "n", caret: 1},
	{preedit: "ni", caret: 2},
	{preedit: "に", caret: 1},
	{preedit: "にほ", caret: 2},
	{preedit: "にほん", caret: 3},
	{commit: "日本"},
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — IME preedit (simulated)"),
		ui.WithSize(520, 300),
	)

	field := ui.NewTextField(ui.Placeholder("click here, then press the button"))

	// Simulated-IME state machine, advanced by OnFrame.
	const beats = 24 // frames between steps (~0.4s at 60fps)
	idx, tick, running := 0, 0, false

	apply := func(s imeStep) {
		if s.commit != "" {
			// End the composition (empty preedit) then deliver committed runes,
			// exactly as a backend would on accept.
			field.HandleEvent(&ui.Event{Type: ui.EventComposition, Comp: render.Composition{}})
			for _, r := range s.commit {
				field.HandleEvent(&ui.Event{Type: ui.EventText, Rune: r})
			}
			return
		}
		field.HandleEvent(&ui.Event{
			Type: ui.EventComposition,
			Comp: render.Composition{Text: s.preedit, Caret: s.caret},
		})
	}

	app.OnFrame(func(dt float64) {
		if !running {
			return
		}
		if tick%beats == 0 {
			apply(script[idx])
			idx++
			if idx >= len(script) {
				running = false
			}
		}
		tick++
	})

	start := ui.NewButton("Simulate composition → 日本")
	start.OnClick(func() { idx, tick, running = 0, 0, true })

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("Inline IME preedit (drawn underlined at the caret):"))
	root.Add(field)
	root.Add(start)
	root.Add(ui.NewLabel("Note: simulated — EBiten exposes no real IME, so this drives"))
	root.Add(ui.NewLabel("the same EventComposition path a backend would. Typing also works."))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
