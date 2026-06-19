package render

import "testing"

func TestButtonSet(t *testing.T) {
	var s ButtonSet
	if s.Has(MouseLeft) {
		t.Fatal("empty set should have no buttons")
	}
	s = s.Set(MouseLeft).Set(MouseRight)
	if !s.Has(MouseLeft) || !s.Has(MouseRight) {
		t.Fatal("set buttons should be present")
	}
	if s.Has(MouseMiddle) {
		t.Fatal("unset button should be absent")
	}
}

func TestModifierSet(t *testing.T) {
	m := ModifierSet(ModShift | ModControl)
	if !m.Has(ModShift) {
		t.Error("Shift should be present")
	}
	if !m.Has(ModControl) {
		t.Error("Control should be present")
	}
	if m.Has(ModAlt) || m.Has(ModMeta) {
		t.Error("Alt/Meta should be absent")
	}
	if (ModifierSet(0)).Has(ModShift) {
		t.Error("empty modifier set should have nothing")
	}
}
