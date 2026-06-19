package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
)

// fakeImage is a render.Image with a fixed size, for tests that don't need real
// pixels.
type fakeImage struct{ w, h float64 }

func (f fakeImage) Size() geom.Size { return geom.Size{W: f.w, H: f.h} }

func TestFitRectStretch(t *testing.T) {
	got := fitRect(FitStretch, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, geom.Size{W: 20, H: 20})
	want := geom.Rect{X: 10, Y: 10, W: 100, H: 50}
	if got != want {
		t.Fatalf("stretch: got %+v want %+v", got, want)
	}
}

func TestFitRectNoneCenters(t *testing.T) {
	got := fitRect(FitNone, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, geom.Size{W: 20, H: 20})
	want := geom.Rect{X: 50, Y: 25, W: 20, H: 20} // native size, centered
	if got != want {
		t.Fatalf("none: got %+v want %+v", got, want)
	}
}

func TestFitRectContainPreservesAspect(t *testing.T) {
	// 20x20 into 100x50: limiting axis is height → scale 2.5 → 50x50, centered.
	got := fitRect(FitContain, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, geom.Size{W: 20, H: 20})
	want := geom.Rect{X: 35, Y: 10, W: 50, H: 50}
	if got != want {
		t.Fatalf("contain: got %+v want %+v", got, want)
	}
}

func TestImageMinSize(t *testing.T) {
	if got := NewImage(fakeImage{w: 40, h: 30}).MinSize(); got != (geom.Size{W: 40, H: 30}) {
		t.Fatalf("MinSize with image: got %+v", got)
	}
	if got := NewImage(nil).MinSize(); got != (geom.Size{}) {
		t.Fatalf("MinSize with no image should be zero, got %+v", got)
	}
}

func TestLoadImageBytesRejectsGarbage(t *testing.T) {
	if _, err := LoadImageBytes([]byte("definitely not an image")); err == nil {
		t.Fatal("LoadImageBytes should error on undecodable data")
	}
}

func TestButtonIconOnlyMinSize(t *testing.T) {
	b := NewButton("") // no text
	b.SetImage(fakeImage{w: 16, h: 16})
	got := b.MinSize()
	want := geom.Size{
		W: 16 + buttonPadding.Left + buttonPadding.Right,
		H: 16 + buttonPadding.Top + buttonPadding.Bottom,
	}
	if got != want {
		t.Fatalf("icon-only button MinSize: got %+v want %+v", got, want)
	}
}

func TestButtonLabelColorByState(t *testing.T) {
	normal := NewButton("x")
	if !sameColor(normal.labelColor(), normal.ColorOf(RoleOnPrimary)) {
		t.Error("a normal button should use OnPrimary for its label")
	}
	flat := NewButton("y", ButtonFlat())
	if !sameColor(flat.labelColor(), flat.ColorOf(RoleText)) {
		t.Error("a flat button should use Text for its label")
	}
	disabled := NewButton("z", ButtonFlat())
	disabled.SetEnabled(false)
	if !sameColor(disabled.labelColor(), disabled.ColorOf(RoleTextMuted)) {
		t.Error("a disabled button should mute its label (TextMuted)")
	}
}

func TestButtonIconAddsWidth(t *testing.T) {
	app := NewApp()
	textOnly := NewButton("Go")
	withIcon := NewButton("Go")
	app.SetContent(NewContainer()) // mount something so theme font is available
	// Mount both buttons via a container so they resolve the theme font.
	root := NewContainer()
	root.Add(textOnly)
	root.Add(withIcon)
	app.SetContent(root)
	withIcon.SetImage(fakeImage{w: 16, h: 16})

	delta := withIcon.MinSize().W - textOnly.MinSize().W
	if !approx(delta, 16+buttonIconGap) {
		t.Fatalf("adding a 16px icon should widen by icon+gap (%v), got %v", 16.0+buttonIconGap, delta)
	}
}
