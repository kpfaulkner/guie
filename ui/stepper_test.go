package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func TestStepperClampsAndStep(t *testing.T) {
	s := NewStepper(StepperRange(0, 5), StepperValue(10)) // over max
	if s.Value() != 5 {
		t.Fatalf("initial value should clamp to max, got %v", s.Value())
	}
	s.Step(-2)
	if s.Value() != 3 {
		t.Fatalf("Step(-2) should give 3, got %v", s.Value())
	}
	s.Step(10) // would exceed max
	if s.Value() != 5 {
		t.Fatalf("Step should clamp to max, got %v", s.Value())
	}
}

func TestStepperOnChangeFiresOnlyOnChange(t *testing.T) {
	s := NewStepper(StepperRange(0, 10))
	calls := 0
	s.OnChange(func(float64) { calls++ })
	s.SetValue(0) // unchanged
	s.SetValue(4) // change
	s.SetValue(4) // unchanged
	if calls != 1 {
		t.Fatalf("OnChange should fire once, got %d", calls)
	}
}

func TestStepperButtonClicks(t *testing.T) {
	app := NewApp()
	s := NewStepper(StepperRange(0, 100), StepperStep(5))
	app.SetContent(s)
	s.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 40})

	// Up button: right column, top half.
	s.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: 190, Y: 8}})
	if s.Value() != 5 {
		t.Fatalf("clicking up should add a step, got %v", s.Value())
	}
	// Down button: right column, bottom half.
	s.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: 190, Y: 32}})
	if s.Value() != 0 {
		t.Fatalf("clicking down should subtract a step, got %v", s.Value())
	}
	// Click in the value area does nothing to the value.
	s.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: 20, Y: 20}})
	if s.Value() != 0 {
		t.Fatalf("clicking the value area should not change it, got %v", s.Value())
	}
}

func TestStepperKeyboardAndWheel(t *testing.T) {
	s := NewStepper(StepperRange(-10, 10), StepperValue(0))
	s.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyUp})
	s.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyUp})
	if s.Value() != 2 {
		t.Fatalf("two Up presses should give 2, got %v", s.Value())
	}
	s.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}})
	if s.Value() != 1 {
		t.Fatalf("wheel down should subtract a step, got %v", s.Value())
	}
	s.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyHome})
	if s.Value() != -10 {
		t.Fatalf("Home should jump to min, got %v", s.Value())
	}
	s.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyEnd})
	if s.Value() != 10 {
		t.Fatalf("End should jump to max, got %v", s.Value())
	}
}

func TestStepperDisabledIgnoresInput(t *testing.T) {
	s := NewStepper(StepperRange(0, 10), StepperValue(5))
	s.SetEnabled(false)
	if s.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeyUp}) {
		t.Fatal("disabled stepper should not consume input")
	}
	if s.Value() != 5 {
		t.Fatalf("disabled stepper value should be unchanged, got %v", s.Value())
	}
}

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner()
	if !s.Spinning() {
		t.Fatal("a new spinner should be spinning")
	}
	s.Stop()
	if s.Spinning() {
		t.Fatal("Stop should halt spinning")
	}
	s.Start()
	if !s.Spinning() {
		t.Fatal("Start should resume spinning")
	}
	if ms := s.MinSize(); ms.W != spinnerDefaultSize || ms.H != spinnerDefaultSize {
		t.Fatalf("default MinSize should be square %v, got %v", spinnerDefaultSize, ms)
	}
}
