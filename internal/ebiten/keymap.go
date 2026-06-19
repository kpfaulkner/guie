package ebitenbackend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/uiframework/render"
)

// keyMap translates EBiten key codes into backend-neutral render.Key values.
// Keys absent from the map are reported as render.KeyUnknown (i.e. dropped).
var keyMap = map[ebiten.Key]render.Key{
	ebiten.KeyEnter:      render.KeyEnter,
	ebiten.KeyEscape:     render.KeyEscape,
	ebiten.KeyBackspace:  render.KeyBackspace,
	ebiten.KeyTab:        render.KeyTab,
	ebiten.KeySpace:      render.KeySpace,
	ebiten.KeyDelete:     render.KeyDelete,
	ebiten.KeyHome:       render.KeyHome,
	ebiten.KeyEnd:        render.KeyEnd,
	ebiten.KeyPageUp:     render.KeyPageUp,
	ebiten.KeyPageDown:   render.KeyPageDown,
	ebiten.KeyArrowLeft:  render.KeyLeft,
	ebiten.KeyArrowRight: render.KeyRight,
	ebiten.KeyArrowUp:    render.KeyUp,
	ebiten.KeyArrowDown:  render.KeyDown,

	ebiten.KeyA: render.KeyA,
	ebiten.KeyB: render.KeyB,
	ebiten.KeyC: render.KeyC,
	ebiten.KeyD: render.KeyD,
	ebiten.KeyE: render.KeyE,
	ebiten.KeyF: render.KeyF,
	ebiten.KeyG: render.KeyG,
	ebiten.KeyH: render.KeyH,
	ebiten.KeyI: render.KeyI,
	ebiten.KeyJ: render.KeyJ,
	ebiten.KeyK: render.KeyK,
	ebiten.KeyL: render.KeyL,
	ebiten.KeyM: render.KeyM,
	ebiten.KeyN: render.KeyN,
	ebiten.KeyO: render.KeyO,
	ebiten.KeyP: render.KeyP,
	ebiten.KeyQ: render.KeyQ,
	ebiten.KeyR: render.KeyR,
	ebiten.KeyS: render.KeyS,
	ebiten.KeyT: render.KeyT,
	ebiten.KeyU: render.KeyU,
	ebiten.KeyV: render.KeyV,
	ebiten.KeyW: render.KeyW,
	ebiten.KeyX: render.KeyX,
	ebiten.KeyY: render.KeyY,
	ebiten.KeyZ: render.KeyZ,

	ebiten.KeyDigit0: render.Key0,
	ebiten.KeyDigit1: render.Key1,
	ebiten.KeyDigit2: render.Key2,
	ebiten.KeyDigit3: render.Key3,
	ebiten.KeyDigit4: render.Key4,
	ebiten.KeyDigit5: render.Key5,
	ebiten.KeyDigit6: render.Key6,
	ebiten.KeyDigit7: render.Key7,
	ebiten.KeyDigit8: render.Key8,
	ebiten.KeyDigit9: render.Key9,
}
