package ebitenbackend

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// Drawing on a scaled display reuses one scaled face instead of making one per
// call, and makes it again when the scale changes.
func TestScaledFaceIsReused(t *testing.T) {
	ff := DefaultFont(14).(*fontFace)
	if ff.scaledFace(1) != ff.face {
		t.Fatal("scale 1 must draw with the face itself")
	}
	a := ff.scaledFace(2)
	if a.Size != 28 || ff.face.Size != 14 {
		t.Fatalf("scaled size %v, face size %v", a.Size, ff.face.Size)
	}
	if b := ff.scaledFace(2); b != a {
		t.Error("a second call at the same scale made a new face")
	}
	if n := testing.AllocsPerRun(100, func() { ff.scaledFace(2) }); n != 0 {
		t.Errorf("%v allocations per call", n)
	}
	if c := ff.scaledFace(1.5); c == a || c.Size != 21 {
		t.Errorf("scale change: size %v, same face %v", c.Size, c == a)
	}
}

// A font file is read and parsed once: a second face, at another size, shares
// the source, and is made even after the file has gone.
func TestLoadFontFileParsesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "font.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadFontFile(path, 12)
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(path)
	b, err := LoadFontFile(path, 20)
	if err != nil {
		t.Fatalf("second load read the file again: %v", err)
	}
	fa, fb := a.(*fontFace), b.(*fontFace)
	if fa.face.Source != fb.face.Source || fb.face.Size != 20 {
		t.Errorf("faces do not share a source (size %v)", fb.face.Size)
	}
	if _, err := LoadFontFile(filepath.Join(t.TempDir(), "missing.ttf"), 12); err == nil {
		t.Error("a missing file loaded")
	}
}
