package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

func newDialogApp() *App {
	app := NewApp()
	app.SetContent(NewContainer())
	app.resize(400, 400)
	return app
}

func TestShowMessageOpensModal(t *testing.T) {
	app := newDialogApp()
	p := app.ShowMessage("Title", "Body")
	if len(app.overlays) != 1 || !p.modal {
		t.Fatalf("ShowMessage should open one modal popup")
	}
}

func TestModalBlocksOutsideClick(t *testing.T) {
	app := newDialogApp()
	app.ShowMessage("Title", "Body")
	// A click in the far corner (on the scrim) must not dismiss the modal.
	clickAt(app, geom.Point{X: 5, Y: 5})
	if len(app.overlays) != 1 {
		t.Fatalf("modal should stay open on outside click, overlays=%d", len(app.overlays))
	}
}

func TestDialogButtonRunsAndCloses(t *testing.T) {
	app := newDialogApp()
	ran := ""
	p := app.ShowMessage("Title", "Body",
		DialogButton{Label: "Cancel", OnClick: func() { ran = "cancel" }},
		DialogButton{Label: "OK", OnClick: func() { ran = "ok" }},
	)
	panel := p.content.(*Container)
	row := panel.Children()[2].(*Container)
	ok := row.Children()[1] // second button

	ev := Event{Type: EventClick}
	ok.HandleEvent(&ev)

	if ran != "ok" {
		t.Fatalf("OK button should run its handler, got %q", ran)
	}
	if len(app.overlays) != 0 {
		t.Fatalf("choosing a dialog button should close the modal, overlays=%d", len(app.overlays))
	}
}

func TestEscapeClosesModal(t *testing.T) {
	app := newDialogApp()
	app.ShowMessage("Title", "Body")
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEscape}})
	if len(app.overlays) != 0 {
		t.Fatalf("Escape should close the modal, overlays=%d", len(app.overlays))
	}
}

func TestShowModalCenters(t *testing.T) {
	app := newDialogApp()
	p := app.ShowModal(newStub(100, 80))
	// Width is clamped up to the minimum modal width (240); centered in 400x400.
	if p.bounds.W != 240 {
		t.Fatalf("width should clamp to %d, got %v", minModalWidth, p.bounds.W)
	}
	if p.bounds.X != (400-240)/2 || p.bounds.Y != (400-80)/2 {
		t.Fatalf("dialog should be centered, got %+v", p.bounds)
	}
}
