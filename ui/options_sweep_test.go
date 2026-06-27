package ui

import (
	"image"
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
)

// mountForTest attaches w to a real (headless) App so it resolves the theme font,
// letting font-dependent methods like MinSize measure text.
func mountForTest(w Widget) {
	a := NewApp()
	root := NewContainer()
	root.Add(w)
	a.SetContent(root)
}

func TestButtonOptionsAndSetters(t *testing.T) {
	f := DefaultFont(12)
	b := NewButton("x", ButtonFont(f), ButtonImage(fakeImage{w: 16, h: 16}))
	if b.face() != f {
		t.Error("ButtonFont should set the face")
	}
	if b.icon == nil {
		t.Error("ButtonImage should set the icon")
	}
	b.SetText("y")
	if b.text != "y" {
		t.Errorf("SetText: got %q", b.text)
	}
	b.SetFont(DefaultFont(13))
	if b.face() == f {
		t.Error("SetFont should replace the face")
	}

	// flatHighlight has a hover branch and a pressed+hover branch.
	bf := NewButton("z", ButtonFlat())
	mountForTest(bf)
	bf.hover = true
	if bf.flatHighlight() == nil {
		t.Error("flatHighlight (hover) should return a colour")
	}
	bf.pressed = true
	if bf.flatHighlight() == nil {
		t.Error("flatHighlight (pressed) should return a colour")
	}
}

func TestLabelOptionsAndSetters(t *testing.T) {
	f := DefaultFont(12)
	l := NewLabel("hi", LabelFont(f), LabelAlign(geom.AlignCenter))
	if l.align != geom.AlignCenter {
		t.Error("LabelAlign should set alignment")
	}
	if l.font != f {
		t.Error("LabelFont should set the font")
	}
	l.SetText("bye")
	if l.Text() != "bye" {
		t.Errorf("SetText/Text: got %q", l.Text())
	}
}

func TestListSetFontMinSize(t *testing.T) {
	l := NewList([]string{"a", "b"})
	l.SetFont(DefaultFont(12))
	if l.MinSize().H <= 0 {
		t.Error("list MinSize should be positive with a font")
	}
}

func TestDropdownOptionsMinSize(t *testing.T) {
	d := NewDropdown([]string{"a", "b", "c"}, DropdownSelected(1))
	if d.Selected() != 1 {
		t.Errorf("DropdownSelected: got %d, want 1", d.Selected())
	}
	d.SetFont(DefaultFont(12))
	if d.MinSize().W <= 0 {
		t.Error("dropdown MinSize should be positive with a font")
	}
}

func TestColourPickerSetFontMinSize(t *testing.T) {
	p := NewColourPicker()
	p.SetFont(DefaultFont(12))
	if p.MinSize().W <= 0 {
		t.Error("colour picker MinSize should be positive with a font")
	}
}

func TestDateOptionsAndSetters(t *testing.T) {
	when := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	d := NewDatePicker(DatePickerToday(when))
	d.SetFont(DefaultFont(12))
	d.ShowMonth(2025, time.March) // navigate; should not panic

	df := NewDateField(DateFieldFirstWeekday(time.Monday))
	df.SetFont(DefaultFont(12))
}

func TestStepperOptionsMinSize(t *testing.T) {
	s := NewStepper(StepperDecimals(2))
	s.SetFont(DefaultFont(12))
	if s.MinSize().W <= 0 {
		t.Error("stepper MinSize should be positive with a font")
	}
}

func TestSpinnerOptions(t *testing.T) {
	if sp := NewSpinner(SpinnerSize(40), SpinnerSpeed(2)); sp == nil {
		t.Error("NewSpinner with options should construct")
	}
}

func TestSplitterRatio(t *testing.T) {
	s := HSplit(NewLabel("a"), NewLabel("b"), SplitRatio(0.3))
	if !approx(s.Ratio(), 0.3) {
		t.Errorf("SplitRatio/Ratio: got %v, want 0.3", s.Ratio())
	}
}

func TestTabsSetFontFocusableMinSize(t *testing.T) {
	tc := NewTabContainer()
	tc.AddTab("A", NewLabel("x"))
	tc.SetFont(DefaultFont(12))
	if !tc.Focusable() {
		t.Error("an enabled tab container should be focusable")
	}
	if tc.MinSize().W <= 0 {
		t.Error("tabs MinSize should be positive with a font")
	}
}

func TestTextAreaPlaceholderSetFontMinSize(t *testing.T) {
	ta := NewTextArea(TextAreaPlaceholder("hint"))
	ta.SetFont(DefaultFont(12))
	if ta.MinSize().H <= 0 {
		t.Error("textarea MinSize should be positive with a font")
	}
}

func TestTextFieldPlaceholderOnChangeSetFont(t *testing.T) {
	tf := NewTextField(Placeholder("hint"))
	tf.SetFont(DefaultFont(12))
	got := ""
	tf.OnChange(func(s string) { got = s })
	typeRune(tf, 'q')
	if got != "q" {
		t.Errorf("OnChange should fire on typing: got %q", got)
	}
}

func TestCheckboxRadioSliderFocusableMinSize(t *testing.T) {
	cb := NewCheckbox("x")
	mountForTest(cb)
	if !cb.Focusable() || cb.MinSize().H <= 0 {
		t.Error("checkbox should be focusable with a positive MinSize")
	}

	rg := NewRadioGroup()
	rb := NewRadioButton("y", rg)
	mountForTest(rb)
	if !rb.Focusable() || rb.MinSize().H <= 0 {
		t.Error("radio should be focusable with a positive MinSize")
	}

	sl := NewSlider()
	if !sl.Focusable() || sl.MinSize().W <= 0 {
		t.Error("slider should be focusable with a positive MinSize")
	}
}

func TestWithIconOption(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 16, 16))
	a := NewApp(WithIcon(icon))
	if len(a.cfg.Icon) != 1 {
		t.Errorf("WithIcon should record the icon, got %d", len(a.cfg.Icon))
	}
}
