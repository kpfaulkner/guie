# DESIGN

## GOAL

Make a Golang library that can be used to make cross-platform applications.
This is a user interface library/framework that provides things such as
windows, widgets, events, scrolling, menus, lists, dropdown combos etc.

As a graphical basis it uses the EBiten (`github.com/hajimehoshi/ebiten/v2`)
library for rendering, windowing and input.

**Core principle: EBiten is an internal implementation detail.** A developer
using this framework should never need to know EBiten exists. They do not
import it, do not implement `ebiten.Game`, and do not see `*ebiten.Image` or
`ebiten` input types in any public API. The framework owns the main loop and
exposes its own abstractions for drawing surfaces, colors, input and events. If
we ever swap the rendering backend, application code should not change.

Features that should be available:

- windows
- buttons
- scroll bars
- menus
- lists
- checkboxes
- radio buttons
- events
- themes
- control of window and widget colours
- control of fonts

---

## DESIGN DECISIONS (locked)

| Area          | Decision                                                                 |
|---------------|--------------------------------------------------------------------------|
| Layout        | Layout managers (HBox / VBox / Grid / Flex). Widgets resize with parent. |
| Architecture  | Retained widget tree — persistent, stateful widget objects.              |
| API style     | Struct + methods, constructed via functional options.                    |
| Theming       | Simple named color palette + a default font, to start.                   |
| Events        | Callbacks as the primary API; optional event bus for global listening.   |
| Main loop     | Framework owns the loop. EBiten fully abstracted away.                    |
| Fonts         | Bundle a default TTF so apps work out of the box; allow custom fonts.    |
| Build order   | Finish this design, then implement a core vertical slice first.          |

---

## ARCHITECTURE

### Layered structure

```
+-----------------------------------------------------------+
|  Application code (no ebiten imports)                     |
+-----------------------------------------------------------+
|  Public API:  App, Window, Widgets, Layouts, Events, Theme|
+-----------------------------------------------------------+
|  Core:  widget tree, layout engine, event dispatch, focus |
+-----------------------------------------------------------+
|  Backend abstraction:  Canvas, Input, Clock, FontFace     |  <- interfaces
+-----------------------------------------------------------+
|  EBiten backend (internal)                                |  <- only place that imports ebiten
+-----------------------------------------------------------+
```

The **backend abstraction** layer is the seam that keeps EBiten out of the
public API. The core and public API talk only to interfaces (`Canvas`, `Input`,
etc.); a single internal package implements those interfaces against EBiten.

### Suggested package layout

```
uiframework/
  ui/            # public API: App, Window, widgets, layouts, options
  core/          # widget tree, layout engine, dispatch, focus mgmt (internal)
  render/        # Canvas + drawing primitive interfaces (backend-agnostic)
  internal/ebiten/ # the only package that imports ebiten/v2 (unimportable by users)
  geom/          # Point, Size, Rect, Insets, alignment enums
  theme/         # Palette, Theme, default theme, fonts
  assets/        # bundled default font (go:embed)
```

### Main loop (owned by the framework)

The framework runs the loop internally. Conceptually each frame:

1. **Input poll** — backend collects mouse/keyboard/touch into a backend-neutral
   `InputState`.
2. **Event dispatch** — translate input into UI events (click, hover, key,
   scroll) and route them through the widget tree (hit-testing, focus).
3. **Update** — widgets update animation/state; app callbacks fire.
4. **Layout** — if the tree is dirty, re-run layout from the root.
5. **Draw** — paint the tree onto the `Canvas` back-to-front.

Public entry point is roughly:

```go
app := ui.NewApp(ui.WithTitle("My App"), ui.WithSize(800, 600))
win := app.MainWindow()
win.SetContent(/* root widget */)
app.Run()   // blocks; starts the loop. No ebiten visible anywhere.
```

---

## BACKEND ABSTRACTION

The render seam. Public widgets receive a `Canvas` in their draw method, never
an `*ebiten.Image`.

```go
// render package (no ebiten import)
type Canvas interface {
    Size() geom.Size
    Clip(r geom.Rect)            // push a clip rect (for scroll views, windows)
    ClearClip()
    FillRect(r geom.Rect, c color.Color)
    StrokeRect(r geom.Rect, c color.Color, width float64)
    DrawLine(a, b geom.Point, c color.Color, width float64)
    DrawText(s string, pos geom.Point, face FontFace, c color.Color)
    MeasureText(s string, face FontFace) geom.Size
    DrawImage(img Image, dst geom.Rect)
    SubCanvas(r geom.Rect) Canvas // for compositing windows/widgets
}

type FontFace interface {
    Metrics() (ascent, descent, lineHeight float64)
}

type Image interface { Size() geom.Size }
```

Input is likewise neutral:

```go
type InputState struct {
    MousePos      geom.Point
    MouseButtons  ButtonSet      // pressed this frame
    MousePressed  ButtonSet      // just went down
    MouseReleased ButtonSet      // just went up
    WheelDelta    geom.Point
    KeysDown      []Key
    KeysPressed   []Key
    Runes         []rune         // text input
    Modifiers     ModifierSet    // shift/ctrl/alt/meta
}
```

The EBiten backend translates `ebiten` input/keys into these types and
implements `Canvas` over `*ebiten.Image`. Nothing else imports `ebiten`.

---

## WIDGET MODEL

### The Widget interface

```go
type Widget interface {
    // identity / tree
    Parent() Widget
    Children() []Widget

    // layout
    MinSize() geom.Size            // intrinsic minimum
    Bounds() geom.Rect             // assigned by parent during layout
    SetBounds(geom.Rect)
    Layout()                       // position children within Bounds

    // paint
    Draw(c render.Canvas)

    // input / events
    HandleEvent(ev Event) bool     // returns true if consumed

    // state
    Visible() bool
    Enabled() bool
}
```

Most widgets embed a `BaseWidget` that provides bounds storage, visibility,
enabled state, parent/child bookkeeping, and default event handling — so a
concrete widget overrides only what it needs.

### Construction (struct + functional options)

```go
btn := ui.NewButton(
    ui.WithText("Save"),
    ui.OnClick(func() { save() }),
    ui.Disabled(false),
)
btn.SetText("Saved")   // methods mutate the retained widget at runtime
```

`NewButton` returns a concrete `*Button` (so methods are discoverable). Options
are `func(*Button)` values; shared options (size, padding, visibility, colours)
are generic helpers that apply to `BaseWidget`.

### Retained tree

Widgets are persistent objects. The framework holds the tree between frames,
re-laying-out only when something is marked dirty (`Invalidate()`), and
redrawing each frame (a later optimization can add dirty-region redraw).

---

## LAYOUT SYSTEM

Two-phase, top-down layout over the retained tree:

1. **Measure** — each widget reports `MinSize()` (bottom-up).
2. **Arrange** — each container assigns `Bounds` to children via its layout
   manager (top-down), then calls each child's `Layout()`.

Containers and layout managers:

- `Box` with direction `Horizontal`/`Vertical` — sequential, with spacing,
  per-child stretch weight, and alignment (start/center/end/stretch).
- `Grid` — rows × columns, per-cell span, column/row weights.
- `Flex` — weighted distribution of free space along one axis.
- `Stack` / `Overlay` — children share the same area (z-ordered) for things
  like dialogs and dropdown popups.
- `Padding` / `Insets` wrapper for margins.

Sizing model per child: a minimum size plus optional stretch weight; containers
distribute leftover space by weight. Layout re-runs on window resize or when a
widget calls `Invalidate()`.

---

## EVENT SYSTEM

Two complementary mechanisms (both requested):

### 1. Callbacks (primary)

Per-widget handlers registered at construction or via setters:

```go
ui.OnClick(func())
ui.OnChange(func(newValue))
ui.OnHover(func(entered bool))
ui.OnFocus(func(focused bool))
ui.OnKey(func(ev KeyEvent) bool)
```

### 2. Event bus (optional, global)

For cross-cutting / global listening without wiring every widget:

```go
app.Events().Subscribe(EventClick, func(ev Event) { ... })
```

Backed by a channel/subscriber model internally; callbacks remain the default
path so simple apps never touch it.

### Dispatch rules

- **Hit-testing**: pointer events go to the top-most widget under the cursor,
  bubbling up to ancestors until consumed (`HandleEvent` returns true).
- **Focus**: one focused widget receives keyboard events; Tab/Shift-Tab move
  focus; clicking a focusable widget focuses it.
- **Capture**: a widget (e.g. a scrollbar thumb or a button being pressed) can
  capture the pointer so it keeps receiving move/release events until release.

Event types (initial): `Click`, `DoubleClick`, `PointerDown/Up/Move`,
`PointerEnter/Leave`, `Wheel`, `KeyDown/KeyUp`, `TextInput`, `FocusGained/Lost`,
`Resize`.

---

## THEMING & COLOUR CONTROL

Start simple: a named color palette plus a default font, with per-widget
overrides so any window or widget colour can be controlled individually.

```go
type Palette struct {
    Background   color.Color
    Surface      color.Color
    Primary      color.Color
    OnPrimary    color.Color
    Text         color.Color
    TextMuted    color.Color
    Border       color.Color
    Accent       color.Color
    Disabled     color.Color
}

type Theme struct {
    Palette      Palette
    Font         render.FontFace
    FontSize     float64
    Spacing      float64   // default gap between widgets
    Padding      geom.Insets
    CornerRadius float64
}
```

- A `DefaultTheme()` ships with the library (light palette + bundled font).
- Theme lives on the `App`; widgets read it during `Draw`.
- **Window and widget colour control**: every widget accepts colour overrides
  via options/setters (`ui.WithBackground(c)`, `ui.WithTextColor(c)`,
  `ui.WithBorderColor(c)`); a window has its own background colour. Unset
  colours fall back to the theme palette.
- Room to grow into multiple themes / dark mode later without API churn.

---

## FONTS & FONT CONTROL

- Bundle a default TTF (the Go font, `golang.org/x/image/font/gofont`, or an
  embedded TTF via `go:embed`) so apps render text with zero setup.
- Expose font loading: `ui.LoadFont(path)` / `ui.LoadFontBytes([]byte)` returning
  a `render.FontFace`.
- **Font control**: font family and size are settable globally via the theme and
  per-widget via options (`ui.WithFont(face)`, `ui.WithFontSize(n)`).
- Text measurement goes through `Canvas.MeasureText` so layout can size labels,
  buttons and list rows correctly.

---

## WIDGET CATALOG

Grouped by build priority.

### Core (built first — the vertical slice)

- **Window** — top-level surface; for now a single OS window with content,
  optional title bar, settable background colour. (Multiple/child windows later.)
- **Container / Box / Grid / Flex** — layout primitives (see Layout System).
- **Label** — static text.
- **Button** — text (and later icon); `OnClick`, pressed/hover/disabled states.
- **Events + focus + theme** wired through these.

### Second wave

- **Checkbox** — boolean toggle, `OnChange(bool)`.
- **RadioButton** / **RadioGroup** — mutually exclusive selection.
- **TextField** — single-line text entry (uses `TextInput`/focus/caret).
- **ScrollBar** + **ScrollView** — clip + offset content; vertical & horizontal.

### Third wave

- **List** — virtualized vertical list of selectable rows.
- **DropdownCombo** — collapsed selector that opens a popup list (uses Overlay).
- **Menu / MenuBar** — popup menus, items, separators, submenus.
- **Slider**, **ProgressBar** — convenience widgets.

### Later / nice-to-have

- Tabs, tooltips, modal dialogs, splitters, tables, drag-and-drop, multi-window,
  dirty-region redraw optimization, accessibility hooks.

---

## OPEN QUESTIONS / TBD

- High-DPI / scaling strategy (logical vs physical pixels).
- Clipboard and IME support for text fields.
- Animation/transition primitives (timeline vs per-frame).
- Persistence of window size/position.
- Testing approach (headless backend implementing `Canvas`/`Input` for unit
  tests of layout and event dispatch without a real window).

---

## IMPLEMENTATION PLAN

1. **Skeleton & backend seam** — `geom` types, `render.Canvas`/`Input`
   interfaces, EBiten backend implementing them, `App.Run()` owning the loop and
   clearing the screen.
2. **Widget tree & base** — `Widget` interface, `BaseWidget`, root container,
   dirty/invalidate, draw traversal.
3. **Layout engine** — `Box` (H/V) first, then `Grid`/`Flex`; resize handling.
4. **Core widgets** — `Label`, `Button`; theme + default font; click/hover/focus.
5. **Event system** — dispatch, hit-testing, focus traversal, callbacks, then
   the optional event bus.
6. **Expand widgets** — second and third waves per the catalog.
7. **Add Modal windows** - allow modal dialogs
8. **Add text area widget** - add widget that has multi line text support
9. **Tabs** — a tabbed container that switches between panes via a tab strip.
10. **Tooltips** — hover-delayed floating hints anchored to a widget.
11. **Splitters** — draggable dividers that resize adjacent panes (H/V split panes).
12. **Tables** — a grid of rows/columns with headers and selectable rows.
13. **Drag-and-drop** — pick up, drag and drop widgets/data between targets.
14. **Multi-window** — more than one top-level OS window, each with its own tree.
15. **Dirty-region redraw** — only repaint changed regions instead of every frame.
16. **Accessibility hooks** — expose roles/labels/state for assistive technology.
17. **Text editing enhancements** — soft word-wrap and selection/clipboard for
    `TextField` and `TextArea`.
18. **Grid cell-spanning** — per-cell column/row spans in the `Grid` layout.
19. **High-DPI scaling** — logical vs physical pixels, device scale factor.
20. **Button has images instead of text** - allow image buttons (as well as text buttons)
21. **Display Images** - Can images be displayed in canvases? Or something else?

Build a tiny example app alongside each step to exercise the new pieces.
