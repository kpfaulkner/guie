package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/render"
)

// newMemApp builds an App pinned to the in-process clipboard. Tests must never
// use the default clipboard: it is the OS one, so a test would read and clobber
// whatever the developer running it had copied.
func newMemApp(opts ...AppOption) *App {
	return NewApp(append(opts, WithClipboard(&memClipboard{}))...)
}

// TestDefaultClipboardFallsBackInProcess covers the path taken when the
// platform clipboard cannot be initialised (a headless machine, a browser that
// withholds access): copy and paste must keep working within the app rather
// than silently doing nothing. The OS branch is not exercised here - it would
// read and clobber the developer's real clipboard.
func TestDefaultClipboardFallsBackInProcess(t *testing.T) {
	orig := tryOSClipboard
	tryOSClipboard = func() render.Clipboard { return nil } // platform clipboard unavailable
	defer func() { tryOSClipboard = orig }()

	var d defaultClipboard
	d.WriteText("in-process")
	if got := d.ReadText(); got != "in-process" {
		t.Fatalf("fallback clipboard should round-trip, got %q", got)
	}
	if _, ok := d.resolve().(*memClipboard); !ok {
		t.Errorf("resolve should yield the in-process clipboard, got %T", d.resolve())
	}
}

func TestTextFieldCopyPasteBetweenFields(t *testing.T) {
	app := newMemApp()
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
	app := newMemApp()
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
	app := newMemApp()
	tf := NewTextField()
	app.SetContent(tf)

	app.clipboard.WriteText("a\nb")
	primaryKey(tf, render.KeyV)
	if tf.Text() != "a b" {
		t.Fatalf("a single-line field should flatten newlines on paste, got %q", tf.Text())
	}
}

func TestTextAreaCopyPasteMultiline(t *testing.T) {
	app := newMemApp()
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
