package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

func TestImageSetImageAndFit(t *testing.T) {
	i := NewImage(nil)
	if i.MinSize() != (geom.Size{}) {
		t.Error("nil image should have zero MinSize")
	}

	i.SetImage(fakeImage{w: 10, h: 20})
	if got := i.MinSize(); got != (geom.Size{W: 10, H: 20}) {
		t.Errorf("after SetImage, MinSize: got %+v, want {10 20}", got)
	}

	if i.fit != FitContain {
		t.Error("default fit should be FitContain")
	}
	i.SetFit(FitStretch)
	if i.fit != FitStretch {
		t.Error("SetFit(FitStretch) not applied")
	}
}

func TestImageDrawNilIsNoop(t *testing.T) {
	// With no image, Draw returns before touching the canvas, so a nil canvas
	// must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw with nil image panicked: %v", r)
		}
	}()
	NewImage(nil).Draw(nil)
}

func TestLoadImageMissingFile(t *testing.T) {
	if _, err := LoadImage("does/not/exist.png"); err == nil {
		t.Fatal("LoadImage on a missing path should return an error")
	}
}

func TestCentered(t *testing.T) {
	got := centered(geom.Rect{X: 0, Y: 0, W: 100, H: 50}, 20, 20)
	if got != (geom.Rect{X: 40, Y: 15, W: 20, H: 20}) {
		t.Errorf("centered: got %+v, want {40 15 20 20}", got)
	}
}
