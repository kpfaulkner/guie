package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

// confirmButtons returns the dialog's Cancel and confirm buttons.
func confirmButtons(p *Popup) (cancel, ok *Button) {
	row := p.content.(*Container).Children()[2].(*Container)
	return row.Children()[0].(*Button), row.Children()[1].(*Button)
}

func confirmClick(w Widget) { w.HandleEvent(&Event{Type: EventClick, Button: render.MouseLeft}) }

func TestConfirmButtonsReportOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		do   func(a *App, p *Popup)
		want bool
	}{
		{"confirm", func(a *App, p *Popup) { _, ok := confirmButtons(p); confirmClick(ok) }, true},
		{"cancel", func(a *App, p *Popup) { c, _ := confirmButtons(p); confirmClick(c) }, false},
		{"escape", func(a *App, p *Popup) {
			a.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEscape}})
		}, false},
		{"programmatic close", func(a *App, p *Popup) { a.Close(p) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newDialogApp()
			var got []bool
			p := app.ShowConfirm("Delete?", "Sure?", func(ok bool) { got = append(got, ok) })
			tc.do(app, p)
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("results = %v, want exactly [%v]", got, tc.want)
			}
			if len(app.overlays) != 0 {
				t.Errorf("dialog still open: overlays=%d", len(app.overlays))
			}
		})
	}
}

func TestConfirmNilCallbackIsSafe(t *testing.T) {
	app := newDialogApp()
	p := app.ShowConfirm("t", "m", nil)
	_, ok := confirmButtons(p)
	confirmClick(ok)
	app.ShowConfirm("t", "m", nil)
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEscape}})
}

func TestConfirmLabels(t *testing.T) {
	app := newDialogApp()
	p := app.ShowConfirm("t", "m", nil, ConfirmLabels("Delete", "Keep"))
	c, ok := confirmButtons(p)
	if c.text != "Keep" || ok.text != "Delete" {
		t.Errorf("labels = %q / %q, want Keep / Delete", c.text, ok.text)
	}
	p2 := newDialogApp().ShowConfirm("t", "m", nil)
	c, ok = confirmButtons(p2)
	if c.text != "Cancel" || ok.text != "OK" {
		t.Errorf("default labels = %q / %q, want Cancel / OK", c.text, ok.text)
	}
}

func TestConfirmFocusAndEnter(t *testing.T) {
	app := newDialogApp()
	var got []bool
	p := app.ShowConfirm("t", "m", func(ok bool) { got = append(got, ok) })
	_, ok := confirmButtons(p)
	if app.focused != Widget(ok) {
		t.Fatalf("focus = %v, want the confirm button", app.focused)
	}
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEnter}})
	if len(got) != 1 || !got[0] {
		t.Errorf("Enter results = %v, want [true]", got)
	}
}

func TestConfirmDefaultCancelFocusesCancel(t *testing.T) {
	app := newDialogApp()
	var got []bool
	p := app.ShowConfirm("t", "m", func(ok bool) { got = append(got, ok) }, ConfirmDefaultCancel())
	c, _ := confirmButtons(p)
	if app.focused != Widget(c) {
		t.Fatalf("focus = %v, want the cancel button", app.focused)
	}
	app.dispatchKeyboard(render.InputState{KeysPressed: []render.Key{render.KeyEnter}})
	if len(got) != 1 || got[0] {
		t.Errorf("Enter on default-cancel results = %v, want [false]", got)
	}
}

func TestConfirmRestoresFocus(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	root := NewContainer()
	root.Add(tf)
	app.SetContent(root)
	app.resize(400, 400)
	app.setFocus(tf)

	p := app.ShowConfirm("t", "m", nil)
	if app.focused == Widget(tf) {
		t.Fatal("dialog should take focus")
	}
	app.Close(p)
	if app.focused != Widget(tf) {
		t.Errorf("focus after close = %v, want the text field back", app.focused)
	}
}
