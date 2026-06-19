package ui

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestDefaultFontScalesWithSize(t *testing.T) {
	small := DefaultFont(10)
	large := DefaultFont(20)
	if small == nil || large == nil {
		t.Fatal("DefaultFont returned nil")
	}
	ws := small.Measure("hello").W
	wl := large.Measure("hello").W
	if !(wl > ws) {
		t.Fatalf("larger font should measure wider: small=%v large=%v", ws, wl)
	}
	if !(large.Metrics().LineHeight > small.Metrics().LineHeight) {
		t.Fatal("larger font should have a taller line height")
	}
}

func TestLoadFontBytes(t *testing.T) {
	f, err := LoadFontBytes(goregular.TTF, 16)
	if err != nil || f == nil {
		t.Fatalf("LoadFontBytes should succeed for a valid font: err=%v", err)
	}
	if _, err := LoadFontBytes([]byte("not a font"), 16); err == nil {
		t.Fatal("LoadFontBytes should error on invalid data")
	}
}

func TestWithFontSizeBuildsLargerDefault(t *testing.T) {
	small := NewApp(WithFontSize(10))
	large := NewApp(WithFontSize(24))
	ws := small.Theme().Font.Measure("X").W
	wl := large.Theme().Font.Measure("X").W
	if !(wl > ws) {
		t.Fatalf("WithFontSize should size the default font: small=%v large=%v", ws, wl)
	}
}

func TestWithFontOverridesDefault(t *testing.T) {
	custom := DefaultFont(30)
	app := NewApp(WithFont(custom))
	if app.Theme().Font != custom {
		t.Fatal("WithFont should set the theme font face")
	}
}

func TestAppSetFontMarksLayout(t *testing.T) {
	app := NewApp()
	app.SetContent(NewContainer())
	app.needsLayout = false
	app.SetFont(DefaultFont(22))
	if !app.needsLayout {
		t.Fatal("SetFont should request a re-layout")
	}
	if app.Theme().Font.Measure("X").W != DefaultFont(22).Measure("X").W {
		t.Fatal("SetFont should update the theme font")
	}
}

func TestLabelSetFontChangesMinSize(t *testing.T) {
	app := NewApp()
	lbl := NewLabel("size me")
	app.SetContent(lbl)

	before := lbl.MinSize()
	lbl.SetFont(DefaultFont(40))
	after := lbl.MinSize()
	if !(after.W > before.W && after.H > before.H) {
		t.Fatalf("a larger per-widget font should grow MinSize: before=%+v after=%+v", before, after)
	}
}
