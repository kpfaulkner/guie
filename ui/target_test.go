package ui

import "testing"

// TestNewRenderTargetRejectsNonPositive verifies the dimension guard returns nil
// before allocating any backend resource. (Positive-size targets allocate a GPU
// image, so they're exercised by the examples rather than headless tests, per
// the project's no-GPU-in-tests convention.)
func TestNewRenderTargetRejectsNonPositive(t *testing.T) {
	cases := [][2]int{{0, 10}, {10, 0}, {0, 0}, {-1, 5}, {5, -1}}
	for _, c := range cases {
		if got := NewRenderTarget(c[0], c[1]); got != nil {
			t.Errorf("NewRenderTarget(%d, %d) = %v, want nil", c[0], c[1], got)
		}
	}
}
