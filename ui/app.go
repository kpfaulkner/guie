package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// App owns the window stack and drives the Ebiten game loop. It implements
// ebiten.Game: each frame it polls input into Events, dispatches them to the
// windows (topmost first), then updates and draws every window.
type App struct {
	Background color.Color

	width, height int
	windows       []*Window

	prevCursor    Point
	hasPrevCursor bool
}

// New creates an App with a logical screen size of width x height and applies
// the window title to the host OS window.
func New(title string, width, height int) *App {
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return &App{
		Background: DefaultBackground,
		width:      width,
		height:     height,
	}
}

// AddWindow pushes w onto the top of the window stack.
func (a *App) AddWindow(w *Window) {
	a.windows = append(a.windows, w)
}

// Run starts the game loop. It blocks until the window is closed.
func (a *App) Run() error {
	return ebiten.RunGame(a)
}

// Update polls input, dispatches events, and advances every window.
func (a *App) Update() error {
	for _, ev := range a.pollEvents() {
		for i := len(a.windows) - 1; i >= 0; i-- {
			if a.windows[i].HandleEvent(&ev, Point{}) {
				a.bringToFront(i)
				break
			}
		}
	}
	for _, w := range a.windows {
		if err := w.Update(); err != nil {
			return err
		}
	}
	return nil
}

// Draw clears the screen and renders every window bottom-to-top.
func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(a.Background)
	for _, w := range a.windows {
		w.Draw(screen, Point{})
	}
}

// Layout reports the fixed logical screen size to Ebiten.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return a.width, a.height
}

// bringToFront moves the window at index i to the top of the draw/event stack.
func (a *App) bringToFront(i int) {
	if i == len(a.windows)-1 {
		return
	}
	w := a.windows[i]
	a.windows = append(a.windows[:i], a.windows[i+1:]...)
	a.windows = append(a.windows, w)
}

// pollEvents converts the current frame's raw Ebiten input into Events.
func (a *App) pollEvents() []Event {
	var events []Event

	cx, cy := ebiten.CursorPosition()
	pos := Point{X: cx, Y: cy}
	if !a.hasPrevCursor || pos != a.prevCursor {
		events = append(events, Event{Type: MouseMove, Pos: pos})
		a.prevCursor = pos
		a.hasPrevCursor = true
	}

	for _, btn := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonMiddle, ebiten.MouseButtonRight} {
		if inpututil.IsMouseButtonJustPressed(btn) {
			events = append(events, Event{Type: MouseDown, Pos: pos, Button: btn})
		}
		if inpututil.IsMouseButtonJustReleased(btn) {
			events = append(events, Event{Type: MouseUp, Pos: pos, Button: btn})
		}
	}

	if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
		events = append(events, Event{Type: MouseWheel, Pos: pos, WheelX: wx, WheelY: wy})
	}

	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		events = append(events, Event{Type: KeyDown, Pos: pos, Key: k})
	}
	for _, k := range inpututil.AppendJustReleasedKeys(nil) {
		events = append(events, Event{Type: KeyUp, Pos: pos, Key: k})
	}

	return events
}
