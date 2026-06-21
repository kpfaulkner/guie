package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestStepperClickThroughHarness(t *testing.T) {
	h := guitest.New(200, 40)
	got := -1.0
	s := ui.NewStepper(ui.StepperRange(0, 10), ui.StepperStep(2))
	s.OnChange(func(v float64) { got = v })
	h.SetContent(s) // fills the 200x40 surface

	// Up button lives in the right ~18px column; click its top half.
	h.Click(192, 10)
	if s.Value() != 2 || got != 2 {
		t.Fatalf("clicking up should step to 2, value=%v onChange=%v", s.Value(), got)
	}
	// Bottom half steps down.
	h.Click(192, 30)
	if s.Value() != 0 {
		t.Fatalf("clicking down should step to 0, got %v", s.Value())
	}
}

func TestSpinnerAnimatesWhileSpinning(t *testing.T) {
	h := guitest.New(60, 60)
	sp := ui.NewSpinner()
	h.SetContent(sp)

	// Each frame draws the dot ring.
	rec := h.Step()
	if n := rec.Count(guitest.OpFillCircle); n == 0 {
		t.Fatal("spinner should draw dots")
	}

	// While spinning, the head dot's colour (alpha) advances frame to frame.
	a := firstCircleColour(h.Step())
	b := firstCircleColour(h.Step())
	if a == b {
		t.Fatal("a spinning spinner should change between frames")
	}

	// Stopped, it freezes: consecutive frames are identical.
	sp.Stop()
	c := firstCircleColour(h.Step())
	d := firstCircleColour(h.Step())
	if c != d {
		t.Fatal("a stopped spinner should not change between frames")
	}
}

// firstCircleColour returns the RGBA of the first FillCircle op, or zeros.
func firstCircleColour(rec *guitest.Recording) [4]uint32 {
	for _, op := range rec.Ops {
		if op.Kind == guitest.OpFillCircle && op.Colour != nil {
			r, g, b, a := op.Colour.RGBA()
			return [4]uint32{r, g, b, a}
		}
	}
	return [4]uint32{}
}
