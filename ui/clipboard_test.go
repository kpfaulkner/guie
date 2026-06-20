package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

func TestTextFieldCopyPasteBetweenFields(t *testing.T) {
	app := NewApp()
	src := NewTextField()
	dst := NewTextField()
	root := NewContainer()
	root.Add(src)
	root.Add(dst)
	app.SetContent(root)

	src.SetText("hello")
	src.selectAll()
	primaryKey(src, render.KeyC)

	dst.SetText("") // caret at 0
	primaryKey(dst, render.KeyV)
	if dst.Text() != "hello" {
		t.Fatalf("paste should copy text across fields, got %q", dst.Text())
	}
}

func TestTextFieldCutClearsAndStores(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)

	tf.SetText("abc")
	tf.selectAll()
	primaryKey(tf, render.KeyX)
	if tf.Text() != "" {
		t.Fatalf("cut should remove the selected text, got %q", tf.Text())
	}
	if app.clipboard.ReadText() != "abc" {
		t.Fatalf("cut should store the text on the clipboard, got %q", app.clipboard.ReadText())
	}
}

func TestTextFieldPasteFlattensNewlines(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)

	app.clipboard.WriteText("a\nb")
	primaryKey(tf, render.KeyV)
	if tf.Text() != "a b" {
		t.Fatalf("a single-line field should flatten newlines on paste, got %q", tf.Text())
	}
}

func TestTextAreaCopyPasteMultiline(t *testing.T) {
	app := NewApp()
	ta := NewTextArea()
	app.SetContent(ta)

	ta.SetText("ab\ncd")
	primaryKey(ta, render.KeyA)
	primaryKey(ta, render.KeyC)
	if app.clipboard.ReadText() != "ab\ncd" {
		t.Fatalf("copy should store the multi-line selection, got %q", app.clipboard.ReadText())
	}

	ta.SetText("") // caret at (0,0)
	primaryKey(ta, render.KeyV)
	if ta.Text() != "ab\ncd" {
		t.Fatalf("paste should restore the multi-line text, got %q", ta.Text())
	}
}
