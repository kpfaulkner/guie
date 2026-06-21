package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

// composition builds an EventComposition for the given preedit.
func composition(text string, caret int) *Event {
	return &Event{Type: EventComposition, Comp: render.Composition{Text: text, Caret: caret}}
}

func TestTextFieldComposeReplacesSelection(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)
	tf.SetText("abc")
	tf.selectAll()

	tf.HandleEvent(composition("x", 1))
	if !tf.composing() {
		t.Fatal("field should be composing after EventComposition")
	}
	if tf.Text() != "" {
		t.Fatalf("starting composition should remove the selection, got %q", tf.Text())
	}

	// A subsequent preedit update must not re-delete anything.
	tf.HandleEvent(composition("xy", 2))
	if tf.Text() != "" {
		t.Fatalf("preedit update should not change committed text, got %q", tf.Text())
	}

	// Clearing the preedit ends composition without committing.
	tf.HandleEvent(composition("", 0))
	if tf.composing() {
		t.Fatal("empty composition should end composing")
	}
}

func TestTextFieldFocusLossClearsPreedit(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)
	tf.HandleEvent(&Event{Type: EventFocusGained})
	tf.HandleEvent(composition("あ", 1))
	if !tf.composing() {
		t.Fatal("should be composing")
	}
	tf.HandleEvent(&Event{Type: EventFocusLost})
	if tf.composing() {
		t.Fatal("losing focus should clear the preedit")
	}
}

func TestTextAreaComposeReplacesSelection(t *testing.T) {
	app := NewApp()
	ta := NewTextArea()
	app.SetContent(ta)
	ta.SetText("hello")
	ta.selectAll()

	ta.HandleEvent(composition("ご", 1))
	if !ta.composing() {
		t.Fatal("area should be composing")
	}
	if ta.Text() != "" {
		t.Fatalf("composition start should remove the selection, got %q", ta.Text())
	}
}
