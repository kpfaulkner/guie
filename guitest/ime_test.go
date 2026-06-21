package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func TestIMEComposeShowsPreeditThenCommit(t *testing.T) {
	h := guitest.New(240, 60)
	tf := ui.NewTextField()
	h.SetContent(tf)
	h.Click(40, 30) // focus

	// Composing shows the preedit inline but does not change the committed text.
	rec := h.Compose("にほ", 2)
	if tf.Text() != "" {
		t.Fatalf("preedit must not commit text yet, got %q", tf.Text())
	}
	if !rec.HasText("にほ") {
		t.Fatalf("preedit should be drawn; texts = %v", rec.Texts())
	}

	// Accepting the candidate commits the final text and clears the preedit.
	h.CommitText("日本")
	if tf.Text() != "日本" {
		t.Fatalf("commit should set the text, got %q", tf.Text())
	}
	frame := h.Step()
	if frame.TextContaining("にほ") {
		t.Fatal("preedit should be gone after commit")
	}
}

func TestIMECancelDiscardsPreedit(t *testing.T) {
	h := guitest.New(240, 60)
	tf := ui.NewTextField()
	h.SetContent(tf)
	h.Click(40, 30)

	h.Compose("か", 1)
	h.CancelComposition()
	if tf.Text() != "" {
		t.Fatalf("cancel should commit nothing, got %q", tf.Text())
	}
	if h.Step().TextContaining("か") {
		t.Fatal("preedit should be cleared after cancel")
	}
}

func TestIMEKeysGatedWhileComposing(t *testing.T) {
	h := guitest.New(240, 60)
	tf := ui.NewTextField()
	h.SetContent(tf)
	h.Click(40, 30)

	h.CommitText("abc") // committed text, caret at end
	if tf.Text() != "abc" {
		t.Fatalf("setup commit failed, got %q", tf.Text())
	}

	h.Compose("x", 1)              // composition active
	h.TypeKey(render.KeyBackspace) // should be swallowed by the IME, not edit
	if tf.Text() != "abc" {
		t.Fatalf("editing keys must be gated while composing, got %q", tf.Text())
	}

	// After cancelling, keys edit again.
	h.CancelComposition()
	h.TypeKey(render.KeyBackspace)
	if tf.Text() != "ab" {
		t.Fatalf("Backspace should work after composition ends, got %q", tf.Text())
	}
}

func TestIMEEnabledFollowsFocus(t *testing.T) {
	h := guitest.New(240, 120)
	tf := ui.NewTextField()
	lbl := ui.NewLabel("not editable")
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(tf, ui.Weight(1), ui.Align(geom.AlignStretch))
	root.Add(lbl, ui.Weight(1), ui.Align(geom.AlignStretch))
	h.SetContent(root)
	h.Step() // lay out

	if h.IMEEnabled() {
		t.Fatal("IME should be off before focusing a text widget")
	}
	h.Click(40, 30) // top half → the text field
	if !h.IMEEnabled() {
		t.Fatal("focusing a text field should enable IME")
	}
	if r := h.IMERect(); r.H <= 0 {
		t.Fatalf("a caret rect should be reported while focused, got %+v", r)
	}
	h.Click(40, 90) // bottom half → the label (clears focus to a non-editable)
	if h.IMEEnabled() {
		t.Fatal("focusing a non-editable widget should disable IME")
	}
}

func TestIMETextAreaCommit(t *testing.T) {
	h := guitest.New(240, 120)
	ta := ui.NewTextArea()
	h.SetContent(ta)
	h.Click(40, 30)

	rec := h.Compose("ご", 1)
	if !rec.HasText("ご") {
		t.Fatalf("textarea preedit should be drawn; texts = %v", rec.Texts())
	}
	h.CommitText("語")
	if ta.Text() != "語" {
		t.Fatalf("textarea commit should set text, got %q", ta.Text())
	}
}
