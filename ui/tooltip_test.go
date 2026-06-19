package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// hoverApp builds an app with a single tooltipped button filling the surface.
func hoverApp(tip string) (*App, *Button) {
	app := NewApp()
	b := NewButton("ok")
	b.SetTooltip(tip)
	app.SetContent(b)
	b.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 100})
	return app, b
}

func hoverFrames(app *App, pos geom.Point, n int) {
	for i := 0; i < n; i++ {
		app.dispatchPointer(render.InputState{MousePos: pos})
	}
}

func TestTooltipAppearsAfterDelay(t *testing.T) {
	app, _ := hoverApp("Save the file")
	pos := geom.Point{X: 20, Y: 20}

	// Just before the threshold: not yet visible. (First tick only records the
	// position, so allow one extra.)
	hoverFrames(app, pos, tooltipDelayTicks)
	if app.tooltipText != "" {
		t.Fatalf("tooltip should not show before the delay elapses, got %q", app.tooltipText)
	}
	hoverFrames(app, pos, 3)
	if app.tooltipText != "Save the file" {
		t.Fatalf("tooltip should appear after resting, got %q", app.tooltipText)
	}
}

func TestTooltipHidesOnMove(t *testing.T) {
	app, _ := hoverApp("hint")
	hoverFrames(app, geom.Point{X: 20, Y: 20}, tooltipDelayTicks+3)
	if app.tooltipText == "" {
		t.Fatal("precondition: tooltip should be visible")
	}
	// Moving the pointer hides it and restarts the timer.
	app.dispatchPointer(render.InputState{MousePos: geom.Point{X: 60, Y: 40}})
	if app.tooltipText != "" {
		t.Fatalf("moving should hide the tooltip, got %q", app.tooltipText)
	}
}

func TestTooltipHidesOnPress(t *testing.T) {
	app, _ := hoverApp("hint")
	pos := geom.Point{X: 20, Y: 20}
	hoverFrames(app, pos, tooltipDelayTicks+3)
	if app.tooltipText == "" {
		t.Fatal("precondition: tooltip should be visible")
	}
	app.dispatchPointer(render.InputState{MousePos: pos, MousePressed: render.ButtonSet(0).Set(render.MouseLeft)})
	if app.tooltipText != "" {
		t.Fatalf("pressing should hide the tooltip, got %q", app.tooltipText)
	}
}

func TestNoTooltipWithoutText(t *testing.T) {
	app := NewApp()
	b := NewButton("ok") // no tooltip set
	app.SetContent(b)
	b.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 100})

	hoverFrames(app, geom.Point{X: 20, Y: 20}, tooltipDelayTicks+5)
	if app.tooltipText != "" {
		t.Fatalf("a widget without tooltip text should never show one, got %q", app.tooltipText)
	}
}
