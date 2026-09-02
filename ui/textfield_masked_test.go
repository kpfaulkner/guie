package ui

import (
	"image/color"
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// fillingCanvas records FillRect calls on top of recordingCanvas's DrawText
// capture, so selection-highlight geometry can be asserted.
type fillingCanvas struct {
	*recordingCanvas
	fills []filledRect
}

type filledRect struct {
	r geom.Rect
	c color.Color
}

func (c *fillingCanvas) FillRect(r geom.Rect, col color.Color) {
	c.fills = append(c.fills, filledRect{r: r, c: col})
}

func (c *fillingCanvas) SubCanvas(geom.Rect) render.Canvas { return c }

func newFillingCanvas() *fillingCanvas {
	return &fillingCanvas{recordingCanvas: &recordingCanvas{}}
}

// maskedTestField mounts a text field in a real (headless) app so it resolves a
// theme and font, gives it bounds, and returns it with the owning app.
func maskedTestField(text string, masked bool) (*TextField, *App) {
	app := newMemApp()
	var opts []TextFieldOption
	if masked {
		opts = append(opts, Masked())
	}
	tf := NewTextField(opts...)
	app.SetContent(tf)
	tf.SetFont(DefaultFont(16))
	tf.SetBounds(geom.Rect{X: 0, Y: 0, W: 300, H: 30})
	tf.SetText(text)
	return tf, app
}

// bullets returns n mask runes as a string.
func bullets(n int) string { return strings.Repeat(string(maskRune), n) }

// drawnStrings paints the field and returns the strings it drew.
func drawnStrings(tf *TextField) []string {
	c := newFillingCanvas()
	tf.Draw(c)
	out := make([]string, 0, len(c.recordingCanvas.texts))
	for _, t := range c.recordingCanvas.texts {
		out = append(out, t.s)
	}
	return out
}

func TestMaskedFieldDrawsMaskNotText(t *testing.T) {
	tf, _ := maskedTestField("secret", true)
	tf.HandleEvent(&Event{Type: EventFocusGained})

	got := drawnStrings(tf)
	if len(got) != 1 {
		t.Fatalf("expected one DrawText, got %d: %q", len(got), got)
	}
	if got[0] != bullets(6) {
		t.Fatalf("masked field should draw one mask rune per character: got %q want %q", got[0], bullets(6))
	}
	for _, s := range got {
		if strings.Contains(s, "secret") || strings.Contains(s, "sec") {
			t.Fatalf("masked field drew the text (or part of it): %q", s)
		}
	}
}

func TestMaskedFieldTextAndOnChangeAreReal(t *testing.T) {
	tf, _ := maskedTestField("", true)
	var seen string
	tf.OnChange(func(s string) { seen = s })

	tf.SetText("hunter2")
	if tf.Text() != "hunter2" {
		t.Fatalf("Text() should return the real text while masked, got %q", tf.Text())
	}
	if seen != "hunter2" {
		t.Fatalf("OnChange should report the real text, got %q", seen)
	}

	tf.HandleEvent(&Event{Type: EventText, Rune: '!'})
	if tf.Text() != "hunter2!" || seen != "hunter2!" {
		t.Fatalf("typing while masked should edit the real text: Text()=%q OnChange=%q", tf.Text(), seen)
	}
}

func TestSetMaskedTogglesAndRedraws(t *testing.T) {
	tf, app := maskedTestField("abcd", false)

	if got := drawnStrings(tf); got[0] != "abcd" {
		t.Fatalf("unmasked field should draw its text, got %q", got[0])
	}

	app.needsLayout = false
	tf.SetMasked(true)
	if !app.needsLayout {
		t.Error("SetMasked should invalidate so the field is redrawn")
	}
	if got := drawnStrings(tf); got[0] != bullets(4) {
		t.Fatalf("SetMasked(true) should mask: got %q", got[0])
	}

	tf.SetMasked(false)
	if got := drawnStrings(tf); got[0] != "abcd" {
		t.Fatalf("SetMasked(false) should reveal: got %q", got[0])
	}
}

// TestMaskedCaretIndexUsesMaskWidths checks click-to-caret mapping measures the
// mask, not the hidden text. With a proportional font "WWWW" is far wider than
// four bullets, so a routing miss lands on a different index.
func TestMaskedCaretIndexUsesMaskWidths(t *testing.T) {
	tf, _ := maskedTestField("WWWW", true)
	f := tf.face()
	inner := tf.Bounds().Inset(textFieldPadding)

	// Just past the middle of the second mask glyph.
	x := inner.X + f.Measure(bullets(2)).W
	if got := tf.caretIndexAt(x); got != 2 {
		t.Fatalf("masked caret index should follow mask advances: got %d want 2", got)
	}

	plain, _ := maskedTestField("WWWW", false)
	if got := plain.caretIndexAt(x); got == 2 {
		t.Skip("mask and real advances coincide at this x; test cannot discriminate")
	}
}

func TestMaskedSelectionHighlightUsesMaskGeometry(t *testing.T) {
	tf, _ := maskedTestField("WWWW", true)
	tf.HandleEvent(&Event{Type: EventFocusGained})
	tf.selectAll()

	c := newFillingCanvas()
	tf.Draw(c)

	f := tf.face()
	want := f.Measure(bullets(4)).W
	found := false
	for _, fr := range c.fills {
		if fr.r.W > want-0.01 && fr.r.W < want+0.01 {
			found = true
		}
	}
	if !found {
		t.Fatalf("selection highlight should span the mask width %.3f; fills were %+v", want, c.fills)
	}
}

func TestMaskedCopyAndCutAreNoOps(t *testing.T) {
	tf, app := maskedTestField("s3cret", true)
	tf.selectAll()

	primaryKey(tf, render.KeyC)
	if got := app.clipboard.ReadText(); got != "" {
		t.Fatalf("copy should be a no-op while masked, clipboard got %q", got)
	}

	primaryKey(tf, render.KeyX)
	if tf.Text() != "s3cret" {
		t.Fatalf("cut should be a full no-op while masked, text became %q", tf.Text())
	}
	if got := app.clipboard.ReadText(); got != "" {
		t.Fatalf("cut should not write to the clipboard while masked, got %q", got)
	}

	// Paste is untouched: putting a secret in is the point.
	app.clipboard.WriteText("pasted")
	tf.SetText("")
	primaryKey(tf, render.KeyV)
	if tf.Text() != "pasted" {
		t.Fatalf("paste should work while masked, got %q", tf.Text())
	}

	// Revealing re-enables copy and cut through the same rule.
	tf.SetMasked(false)
	tf.selectAll()
	primaryKey(tf, render.KeyC)
	if got := app.clipboard.ReadText(); got != "pasted" {
		t.Fatalf("copy should work once revealed, clipboard got %q", got)
	}
	primaryKey(tf, render.KeyX)
	if tf.Text() != "" {
		t.Fatalf("cut should work once revealed, text is %q", tf.Text())
	}
}

// TestMaskedPreeditIsMasked covers the leak the naive implementation has: the
// composing branch draws pre+preedit+post in one call, and the preedit is not
// in t.runes so display() alone does not cover it.
func TestMaskedPreeditIsMasked(t *testing.T) {
	tf, _ := maskedTestField("ab", true)
	tf.HandleEvent(&Event{Type: EventFocusGained})
	tf.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: "にほ", Caret: 2}})

	got := drawnStrings(tf)
	if len(got) == 0 {
		t.Fatal("composing field drew nothing")
	}
	for _, s := range got {
		if strings.ContainsAny(s, "にほab") {
			t.Fatalf("masked field leaked the composition or the text: %q", s)
		}
	}
	if got[0] != bullets(4) {
		t.Fatalf("composing masked field should draw one mask rune per real and preedit rune: got %q want %q", got[0], bullets(4))
	}
}

func TestMaskedPlaceholderIsDrawnUnmasked(t *testing.T) {
	app := newMemApp()
	tf := NewTextField(Masked(), Placeholder("Password"))
	app.SetContent(tf)
	tf.SetFont(DefaultFont(16))
	tf.SetBounds(geom.Rect{X: 0, Y: 0, W: 300, H: 30})

	got := drawnStrings(tf)
	if len(got) != 1 || got[0] != "Password" {
		t.Fatalf("placeholder should be drawn unmasked, got %q", got)
	}
}

// TestMaskedCaretRectUsesMaskWidths pins caretVisualWidth (and through it
// updateScroll and imeCaretRect), which the highlight and click tests do not
// reach: the caret is drawn with DrawLine, which the recording canvas ignores.
func TestMaskedCaretRectUsesMaskWidths(t *testing.T) {
	tf, _ := maskedTestField("WWWW", true)
	tf.HandleEvent(&Event{Type: EventFocusGained})
	tf.setCaret(2, false)

	f := tf.face()
	inner := tf.Bounds().Inset(textFieldPadding)
	r, ok := tf.imeCaretRect()
	if !ok {
		t.Fatal("imeCaretRect should report a rect while focused")
	}
	want := inner.X + f.Measure(bullets(2)).W
	if r.X < want-0.01 || r.X > want+0.01 {
		t.Fatalf("caret x should follow mask advances: got %.3f want %.3f", r.X, want)
	}
	if real := inner.X + f.Measure("WW").W; real > want+0.01 || real < want-0.01 {
		return // the two differ, as expected: the assertion above discriminates
	}
	t.Skip("mask and real advances coincide; test cannot discriminate")
}

// TestMaskedPreeditCaretRectUsesMaskWidths pins the preedit half of
// caretVisualWidth, which the masked-IME path depends on.
func TestMaskedPreeditCaretRectUsesMaskWidths(t *testing.T) {
	tf, _ := maskedTestField("ab", true)
	tf.HandleEvent(&Event{Type: EventFocusGained})
	tf.setCaret(2, false)
	tf.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: "WWW", Caret: 2}})

	f := tf.face()
	inner := tf.Bounds().Inset(textFieldPadding)
	r, ok := tf.imeCaretRect()
	if !ok {
		t.Fatal("imeCaretRect should report a rect while composing")
	}
	// Two committed runes plus two preedit runes, all masked.
	want := inner.X + f.Measure(bullets(2)).W + f.Measure(bullets(2)).W
	if r.X < want-0.01 || r.X > want+0.01 {
		t.Fatalf("caret x should measure the masked preedit: got %.3f want %.3f", r.X, want)
	}
}

// TestMaskedCompositionCommitsRealText finishes the compose row: what the IME
// commits is stored and reported as the real text, masked only on screen.
func TestMaskedCompositionCommitsRealText(t *testing.T) {
	tf, _ := maskedTestField("", true)
	var seen string
	tf.OnChange(func(s string) { seen = s })
	tf.HandleEvent(&Event{Type: EventFocusGained})

	tf.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: "にほ", Caret: 2}})
	// A commit frame clears the preedit, then delivers the committed runes.
	tf.HandleEvent(&Event{Type: EventComposition, Comp: render.Composition{Text: ""}})
	for _, r := range "日本" {
		tf.HandleEvent(&Event{Type: EventText, Rune: r})
	}

	if tf.Text() != "日本" {
		t.Fatalf("committed text should be the real text, got %q", tf.Text())
	}
	if seen != "日本" {
		t.Fatalf("OnChange should report the committed real text, got %q", seen)
	}
	for _, s := range drawnStrings(tf) {
		if strings.ContainsAny(s, "日本") {
			t.Fatalf("masked field drew the committed text: %q", s)
		}
	}
}
