package guitest

import (
	"image"
	"testing"

	"github.com/kpfaulkner/guie/ui"
)

func TestConfigRecordsIconAndAppID(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	h := New(200, 100, ui.WithAppID("myapp"), ui.WithIcon(icon))
	h.Step()
	cfg := h.Config()
	if cfg.AppID != "myapp" || len(cfg.Icon) != 1 || cfg.Icon[0] != icon {
		t.Errorf("config %+v", cfg)
	}
}
