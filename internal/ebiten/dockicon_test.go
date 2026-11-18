package ebitenbackend

import (
	"image"
	"testing"

	"github.com/kpfaulkner/guie/render"
)

func TestLargestIcon(t *testing.T) {
	small := image.NewRGBA(image.Rect(0, 0, 16, 16))
	big := image.NewRGBA(image.Rect(0, 0, 256, 256))
	mid := image.NewRGBA(image.Rect(0, 0, 48, 48))
	if got := largestIcon([]image.Image{small, big, nil, mid}); got != big {
		t.Errorf("largest is %v, want the 256 one", got.Bounds())
	}
	if largestIcon(nil) != nil {
		t.Error("no icons gave an image")
	}
}

func TestRunOptions(t *testing.T) {
	if runOptions(render.Config{}) != nil {
		t.Error("no app id should keep Ebitengine's defaults (nil options)")
	}
	op := runOptions(render.Config{AppID: "tterm"})
	if op == nil || op.X11ClassName != "tterm" || op.X11InstanceName != "tterm" {
		t.Errorf("options %+v, want WM_CLASS tterm", op)
	}
}
