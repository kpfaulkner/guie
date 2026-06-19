# INTERNALS

Detailed design notes for the `guie` library: how the pieces fit
together, the invariants they rely on, and why things are the way they are.
This complements `design.md` (the goals/plan) by documenting the *implemented*
architecture.

Audience: contributors to the framework itself. Application authors only need
the `ui`, `geom`, `render` and `theme` packages.

---

## 1. Overview

`guie` is a **retained-mode** GUI toolkit for Go, rendered with EBiten.
The defining constraint is that **EBiten is an internal detail**: the public
API never exposes an EBiten type, and the EBiten-backed package is physically
unimportable by users (it lives under `internal/`). The framework owns the main
loop; applications build a tree of widgets and register callbacks.

Key characteristics:

- **Retained widget tree** — persistent, stateful widget objects, re-laid-out
  only when marked dirty, redrawn every frame.
- **Absolute coordinates** — every widget's `Bounds()` are in surface space;
  there is no per-draw origin/transform threaded through the tree.
- **Backend seam** — the `render` package defines interfaces (`Canvas`,
  `FontFace`, `Image`, `Driver`, `Clipboard`); `internal/ebiten` is the sole
  implementation and the sole importer of `github.com/hajimehoshi/ebiten/v2`.
- **Single-threaded** — all UI work happens on the loop goroutine; `App.Do`
  marshals work in from other goroutines.

---

## 2. Package layout & dependency graph

```
geom      ── no deps (pure value types)
render    ── imports geom                      (backend-neutral interfaces)
theme     ── imports render                    (palette + font defaults)
internal/ebiten ── imports geom, render, ebiten/v2, x/image   (the backend)
ui        ── imports geom, render, theme, internal/ebiten     (public API)
```

Rules that keep the seam intact:

- **Only `internal/ebiten` imports EBiten.** Verified by grep; nothing else
  references `hajimehoshi/ebiten`.
- **`ui` is the only importer of `internal/ebiten`** (in `app.go` and the
  `font.go`/`image.go` loaders). Because the backend is under `internal/`, Go's
  compiler forbids any *external* program from importing it — the encapsulation
  is enforced, not just conventional.
- **No EBiten type appears in any exported signature.** `ui` re-exports the
  backend's capabilities through neutral types: e.g. `ui.DefaultFont` returns
  `render.FontFace`, `ui.LoadImage` returns `render.Image`.

There is no separate `core/` package (the design sketched one). The widget tree
lives in `ui` because the `Widget` interface has an unexported method (`mount`),
which forces all implementations into one package — see §7.

---

## 3. The backend seam (`render`)

`render` is interfaces + plain data only; it imports just `geom` and
`image/color`.

### 3.1 `Canvas` (`render/canvas.go`)

The drawing surface handed to widgets each frame. All coordinates are logical
pixels in surface space. Methods:

- Clip stack: `PushClip(Rect)` / `PopClip()` — `PushClip` **intersects** with the
  current region. Backends must honour the active clip for every draw call.
- Shapes: `Fill`, `FillRect`, `StrokeRect`, `DrawLine`, `FillCircle`,
  `StrokeCircle`.
- Text: `DrawText(s, pos, face, color)` (pos = top-left), `MeasureText`.
- Images: `DrawImage(img, dst)` (scaled to `dst`).
- `SubCanvas(r)` — a clipped sub-surface (used for compositing; rarely needed).

### 3.2 `FontFace` (`render/canvas.go`)

Opaque, backend-specific, **fixed-size** handle. Exposes `Metrics()` and
`Measure(s) geom.Size`. `Measure` is deliberately frame-independent so widgets
can call it from `MinSize()` during layout (not just during `Draw`).

### 3.3 `Image` (`render/canvas.go`)

Opaque bitmap handle; only `Size()`.

### 3.4 Input (`render/input.go`)

`InputState` is the per-frame, backend-neutral snapshot:

- `MousePos geom.Point`
- `MouseDown`/`MousePressed`/`MouseReleased ButtonSet` — held / edge-down /
  edge-up this frame
- `WheelDelta geom.Point`
- `KeysDown`/`KeysPressed`/`KeysReleased []Key`
- `Runes []rune` (text input)
- `Modifiers ModifierSet`

`ButtonSet`/`ModifierSet` are bitsets with `Set`/`Has`. `Key`, `MouseButton`,
`Modifier` are framework-defined enums (the backend maps native codes onto
them; unmapped keys become `KeyUnknown`).

### 3.5 Loop seam (`render/driver.go`)

- `Config` — `Title`, `Width`, `Height`, `Background`, `Resizable`.
- `Hooks` — `Update(InputState) error`, `Draw(Canvas)`, `Resize(w, h int)`.
- `Driver.Run(Config, Hooks) error` — sets up the window and runs the blocking
  loop, invoking hooks each frame.
- `ErrTerminated` — a sentinel a hook may return from `Update` to request a
  clean shutdown; the driver must treat it as a normal stop (`Run` returns
  `nil`, not the error). This is how `App.Quit()` works (§8).

### 3.6 Clipboard (`render/clipboard.go`)

`Clipboard{ ReadText() string; WriteText(string) }`. The default is an
in-process implementation in `ui` (`memClipboard`); apps can inject an
OS-backed one via `ui.WithClipboard`. This keeps the core dependency-free and
cross-platform; the design's OPEN QUESTIONS flagged clipboard as TBD.

---

## 4. The EBiten backend (`internal/ebiten`)

Package name `ebitenbackend`. The only place that touches EBiten.

### 4.1 Canvas (`canvas.go`)

`canvas` wraps an `*ebiten.Image` and implements `render.Canvas`.

- **Clip stack via sub-images.** The canvas holds a stack of `clipEntry{target
  *ebiten.Image, rect geom.Rect}`. `PushClip(r)` intersects `r` with the current
  clip, takes `base.SubImage(intRect)` as the new target, and pushes it. Drawing
  goes to the top-of-stack target, so anything outside the clip is discarded by
  EBiten's sub-image bounds. `reset(surface)` rebinds for a new frame.
- Shapes use `ebiten/v2/vector` (`FillRect`, `StrokeRect`, `StrokeLine`,
  `FillCircle`, `StrokeCircle`) — note `vector.FillRect` (not the deprecated
  `DrawFilledRect`).
- Text uses `ebiten/v2/text/v2`; `DrawText` translates to `pos` and color-scales;
  `MeasureText` delegates to the face's `Measure`.
- `toImageRect` rounds a float `Rect` outward to integer pixel bounds.

### 4.2 Fonts (`font.go`)

- `fontFace` wraps a `*text.GoTextFace` and caches `render.FontMetrics`.
- The bundled Go font (`x/image/font/gofont/goregular`) source is parsed once
  (`sync.Once`) and reused for all sizes — making faces at new sizes is cheap.
- `DefaultFont(size)` returns a face from the bundled source. `NewFontFace(ttf,
  size)` builds one from arbitrary TTF/OTF bytes.
- `Measure` uses `text.Measure` (no target image required), so layout can size
  text outside a frame.

### 4.3 Images (`image.go`)

- `imageHandle` wraps an `*ebiten.Image`.
- `LoadImageBytes(data)` decodes PNG/JPEG/GIF (`image.Decode`, with the standard
  decoders registered via blank imports) and uploads via
  `ebiten.NewImageFromImage`. Decode errors return *before* any GPU call, which
  is what makes the error path unit-testable headlessly.

### 4.4 Input (`input.go`, `keymap.go`)

`pollInput()` builds an `InputState` from EBiten: cursor position,
`inpututil` edge detection for mouse buttons and keys, `ebiten.Wheel`,
`ebiten.AppendInputChars` for runes, and modifier flags. `keymap.go` maps
`ebiten.Key` → `render.Key` for the supported subset.

### 4.5 Driver / loop (`driver.go`)

- `Driver.Run` sets window title/size/resizable and calls `ebiten.RunGame(g)`.
- `game` implements `ebiten.Game`:
  - `Update()` polls input, calls `hooks.Update`. **If the hook returns
    `render.ErrTerminated`, it is mapped to `ebiten.Termination`** so the loop
    stops cleanly and `RunGame` returns nil. Other errors propagate.
  - `Draw(screen)` rebinds the canvas to `screen`, clears to the background, then
    calls `hooks.Draw`.
  - `Layout(outsideW, outsideH)` returns the logical size = device size (1:1, no
    HiDPI scaling yet) and calls `hooks.Resize` when the size changes.

**Tick model (important):** EBiten calls `Update` at a fixed TPS (default 60),
decoupled from render FPS. The framework relies on this for time-based behaviour
(tooltip delay) without a clock abstraction — counting `Update` calls is
wall-clock-stable.

---

## 5. `geom`

Pure value types, `float64` logical pixels: `Point`, `Size`, `Rect`, `Insets`,
and the `Alignment` (`Start`/`Center`/`End`/`Stretch`) and `Direction`
(`Horizontal`/`Vertical`) enums. `Rect` helpers: `Min`/`Max`/`Size`/`Center`/
`Add`/`Inset`/`Intersect`/`Contains`/`Empty`, plus `FromMinMax`. `Contains` is
inclusive of the top-left, exclusive of the bottom-right (so adjacent rects
don't both claim a boundary pixel — matters for hit-testing).

---

## 6. `theme` & the color model

`theme.Theme` = `Palette` + `Font` + `FontSize` (nothing else — the earlier
`Spacing`/`Padding`/`CornerRadius` fields were removed because no widget read
them). `Palette` is nine named `color.Color` roles: `Background`, `Surface`,
`Primary`, `OnPrimary`, `Text`, `TextMuted`, `Border`, `Accent`, `Disabled`.
`Default()` returns the dark palette at size 14 with a nil `Font` (the App fills
in the bundled font, keeping `theme` backend-independent).

See §12 for how widgets resolve colors through these roles with per-widget
overrides.

---

## 7. The widget model

### 7.1 `Widget` interface (`ui/widget.go`)

Methods: tree (`Parent`, `Children`), layout (`Bounds`, `SetBounds`, `MinSize`,
`Layout`), paint (`Draw`), input (`HandleEvent`), state (`Visible`, `Enabled`,
`Focusable`, `Tooltip`), and the unexported `mount(self, parent, ctx)`.

The unexported `mount` means **external types cannot satisfy `Widget` without
embedding `BaseWidget`** — which is exactly the intended constraint, and the
reason all widgets live in one package. Custom widgets embed `ui.BaseWidget`
(exported) and get `mount` for free.

### 7.2 `BaseWidget` (`ui/base.go`)

Stores bounds, visible/enabled flags, tooltip text, per-widget color overrides,
the widget's own `self Widget` identity, its `parent`, and the shared
`*treeContext`. Provides default implementations of every interface method
(leaf defaults: no children, zero `MinSize`, no-op `Layout`/`Draw`,
non-consuming `HandleEvent`, not focusable) plus `Invalidate`, `SetVisible`,
`SetEnabled`, `SetTooltip`, `RequestFocus`, `SetColor`/`ColorOf`, and the
internal `appTheme()`/`clipboard()` accessors.

### 7.3 The `mount` self-identity mechanism

`mount(self, parent, ctx)` records `self` so that **containers pass the correct
outer `Widget` as the parent of their children**, even when a custom type embeds
`Container`. Without this, an embedded `*Container`'s methods would be recorded
as the parent and event bubbling would reach the wrong `HandleEvent`. Call
chain:

- `App.SetContent(w)` → `w.mount(w, nil, ctx)`.
- A container's `mount(self, parent, ctx)` mounts each child with
  `ch.mount(ch, self, ctx)` — child's parent is `self` (the outer widget).
- `Container.Add(w)` on an already-mounted container calls `w.mount(w, c.self,
  c.ctx)`.

`RequestFocus()` uses the stored `self`, so callers don't pass themselves.

### 7.4 `treeContext`

Shared state injected at mount: `requestLayout` (sets the App dirty flag),
`requestFocus`, `openPopup`/`closePopup`, `theme` (pointer to the App's theme,
so `SetFont`/theme edits are seen live), and `clipboard`. Widgets reach app-wide
services only through this context, never the App directly.

### 7.5 Dirty / invalidation

`Invalidate()` → `ctx.markNeedsLayout()` → sets `App.needsLayout`. The App
re-runs layout from the root before the next frame (`layoutIfNeeded`). Drawing
happens every frame regardless (no dirty-region optimization yet).

---

## 8. The `App` and the frame loop (`ui/app.go`)

### 8.1 Construction

`NewApp(opts...)` builds defaults (ebiten driver via `ebitenbackend.New()`,
default theme, 800×600 resizable), applies options, then: fills the theme font
from the backend if unset, defaults the clear color to the palette background,
defaults the clipboard to `memClipboard`, creates the `EventBus`, and wires the
`treeContext` (closures over the App's dirty flag, focus, popups; pointers to
its theme and clipboard).

### 8.2 Lifecycle

- `SetContent(w)` mounts `w` as the root and marks dirty (sizes it to the
  surface if known).
- `Run()` calls `driver.Run(cfg, Hooks{update, draw, resize})` — blocks.
- `Quit()` (goroutine-safe via `atomic.Bool`) makes the next `update` return
  `render.ErrTerminated`.

### 8.3 Per-frame `update(in)`

Order matters:

1. `runPending()` — drain and run work queued by `App.Do` (on the loop
   goroutine).
2. If `quit` is set, return `render.ErrTerminated` (clean stop).
3. `layoutIfNeeded()` — if dirty, size root to surface, `root.Layout()`, then
   `layoutOverlays()`.
4. `dispatchPointer(in)` (§10).
5. `dispatchKeyboard(in)` (§10).

### 8.4 Per-frame `draw(c)`

`root.Draw` → `drawOverlays` (popups, each modal preceded by a scrim) →
`drawTooltip`. So overlays paint above content, and the tooltip above
everything.

### 8.5 Concurrency model

`App` and the widget tree are **not safe for concurrent use**. The only
goroutine-safe methods are `Do` (queues a func, mutex-guarded) and `Quit`.
Background goroutines update the UI via `app.Do(func(){ ... })`, which runs at
the top of the next frame on the loop goroutine. This is documented on `App`.

---

## 9. Layout engine

### 9.1 Two-phase, top-down

1. **Measure** — `MinSize()` bottom-up (a parent calls children's `MinSize`).
2. **Arrange** — a container assigns child `Bounds` via its `Layout` manager,
   then calls each child's `Layout()` so nested containers arrange their own
   children.

Layout re-runs from the root on resize or any `Invalidate()`.

### 9.2 `Layout` interface & per-child data (`ui/layout.go`)

```go
type Layout interface {
    Measure(items []Item) geom.Size            // content min size (excl. padding)
    Arrange(items []Item, content geom.Rect)   // assign each item's Bounds
}
type Item struct { Widget Widget; Data LayoutData }
type LayoutData struct { Weight int; Align geom.Alignment }
```

`Container` records a `LayoutData` per child (default `Align: Stretch`),
configured at `Add(w, opts...)` time with `ItemOption`s (`Weight(n)`,
`Align(a)`). Shared helpers: `mainExtent`/`crossExtent`/`sizeFor` (axis
projection), `alignSpan` (position+length within available span), and
`distributeTracks` (split a total into weighted tracks — used by Box weights and
Grid columns/rows).

### 9.3 Managers

- **`Box`** (`HBox`/`VBox`): sequential along the main axis with `Spacing`; each
  child takes its min main size, leftover distributed by `Weight`; cross axis per
  `Align` (Stretch fills). This is also the "flex".
- **`Grid`**: fixed `Columns`, auto-flow wrapping; column/row tracks distributed
  by optional weights (equal by default) across the content rect; per-cell
  `Align`. Cell spanning is not implemented (a noted TODO).
- **`Stack`**: overlays children in the content rect per `Align` on both axes;
  basis for centering, dialogs, popups.

### 9.4 `Container` (`ui/container.go`)

Holds children + parallel `LayoutData`, an optional `Layout`, an optional
background fill and border (explicit instance colors with `Background()`/
`BorderColor()` getters), and padding. `MinSize` = layout's measure (or max
child extents if no layout) + padding. `Layout` arranges via the layout over
`ContentRect()` (bounds inset by padding), then recurses. `Draw` fills
background, clips to the content rect, draws visible children, pops the clip,
then strokes the border.

### 9.5 The viewport rule

Scrollable/viewport widgets (`ScrollView`, `TextArea`) must report a **small
intrinsic `MinSize`** and rely on layout (weight/stretch) to size them.
Reporting their content size would let them grow to fit the content, leaving
nothing to scroll. This bit `ScrollView` once and is now a deliberate invariant.

---

## 10. Event system

### 10.1 Events (`ui/event.go`)

`Event{Type, Pos, Button, Wheel, Key, Rune, Modifiers}`. `EventType`:
`PointerEnter/Leave/Move/Down/Up`, `Click`, `Wheel`, `KeyDown/Up`, `Text`,
`FocusGained/Lost`. Which fields matter depends on `Type`.

### 10.2 Dispatch & bubbling (`ui/app.go`)

- `dispatch(target, ev)` sends to `target` and **bubbles up via `Parent()`**
  until a widget consumes it (`HandleEvent` returns true). Every dispatched
  event is also published to the `EventBus`.
- `sendTo(w, ev)` delivers to a single widget without bubbling (enter/leave,
  focus) and publishes.

### 10.3 Hit-testing & pointer dispatch

`hitTest(w, pos)` returns the top-most visible widget whose bounds contain
`pos`, depth-first, preferring later (on-top) children. `dispatchPointer`:

1. Resolve `hit` honouring overlays/modals (§11).
2. A left-press outside all popups is swallowed; non-modal stacks also dismiss.
3. Hover enter/leave when `hit` changes.
4. **Tooltip timing** update (§13).
5. **Pointer move** is dispatched to the **capture target** (press target) if
   any, else the hovered widget — so drags (sliders, scrollbar/splitter thumbs)
   get moves, and self-drawing multi-row widgets (List/Table/Menu) can track the
   cursor row.
6. Wheel → `hit` (bubbles).
7. Press → focus the nearest focusable in the hit chain, dispatch `PointerDown`,
   set the capture target.
8. Release → `PointerUp` to the capture target; if released within its bounds,
   also dispatch a derived `Click`. (So *click derivation* lives in the
   dispatcher, not the widget: press+release on the same target = click;
   release-outside = no click.)

### 10.4 Focus

One `focused` widget receives keyboard events. `setFocus` sends `FocusLost`/
`FocusGained`. `moveFocus(±1)` walks `appendFocusables(root)` (tree order),
wrapping; **Tab/Shift+Tab** drive it. Clicking focuses the nearest `Focusable()`
in the hit chain (clicking empty space clears focus).

### 10.5 Keyboard (`dispatchKeyboard`)

Per `KeysPressed`: `Escape` closes the top popup; `Tab` moves focus; otherwise
`KeyDown` → focused widget (bubbles). `KeysReleased` → `KeyUp`; `Runes` →
`Text`. Modifiers (Shift/Ctrl/...) ride on every event.

### 10.6 Event bus (`ui/bus.go`)

`app.Events().Subscribe(EventType, func(Event))`. Every dispatched event is
published. Per-widget callbacks remain the primary API; the bus is for
cross-cutting listeners. Runs on the UI goroutine (handlers must not block).

### 10.7 Callback convention

Callbacks are **methods** named by event, uniform across widgets: `OnClick`,
`OnChange`, `OnSelect`, `OnSubmit`. Methods (rather than package-level option
funcs) avoid the namespace collision that would otherwise force per-widget names
and give one verb per event type. Construction *options* remain for
non-callback config (`Placeholder`, `Checked`, `SliderValue`, `DropdownSelected`,
`ListSelected`, color/font options, `TextAreaWrap`, ...).

---

## 11. Overlay / popup layer (`ui/popup.go`)

`Popup{content, bounds, onClose, modal}`. The App keeps an `overlays []*Popup`
stack drawn above the root.

- **Open**: `openPopup` mounts the content (`content.mount(content, nil, ctx)`),
  lays it out at its bounds, pushes it. `closePopup` removes it (and anything
  above), calling `onClose`. `closeTop`/`closeAll` for Escape / outside-click.
- **Hit-testing order**: overlays top-most first, then the root. A **modal**
  (top overlay with `modal=true`) confines hit-testing to its content — the
  background is fully blocked, a full-screen scrim is drawn behind it, and
  outside clicks are swallowed *without* dismissing (only its own buttons or
  Escape close it). Non-modal popups (dropdown list, menu) dismiss on
  outside-click.
- **Built on this**: `DropdownCombo` opens a `List` popup below itself;
  `MenuBar` opens a `Container` of `menuRow`s; `App.ShowModal`/`ShowMessage`
  center a panel as a modal dialog. `DialogButton` wraps an action + auto-close.

(There is no in-surface "window" layer yet; multi-window is parked because
EBiten is single-window — see §16.)

---

## 12. Color system (`ui/color.go`)

Every widget resolves the colors it draws through **roles**, not the palette
directly:

- `ColorRole` enum mirrors the palette (`RoleBackground`…`RoleDisabled`).
- `BaseWidget.SetColor(role, c)` stores a per-widget override (nil clears it).
- `BaseWidget.ColorOf(role)` returns the **effective** color: the override if
  set, else `paletteColor(theme.Palette, role)`.

Because this lives on `BaseWidget`, *every* widget supports per-widget color
overrides and read-back uniformly. Widget `Draw` methods call
`w.ColorOf(RoleX)` instead of reading `theme.Palette.X`, so an override actually
changes appearance. Derived state colors (button hover/pressed) are computed
from the base role via `lighten`/`darken`, so overriding `RolePrimary` recolors
the hover/pressed states too. App-level chrome that isn't a widget (modal scrim,
tooltip box) reads the theme palette directly.

`Container` background/border are *explicit instance colors* (default
transparent / none), not roles — they have their own setters/getters.

---

## 13. Tooltips (`ui/tooltip.go`)

Time the hover delay by **counting `Update` ticks** (no clock needed; TPS is
fixed at 60). `App` tracks `lastPointer`, `tooltipTicks`, `tooltipText`,
`tooltipPos`. In `dispatchPointer`, `updateTooltip(pos)`: if the pointer moved,
reset and hide; otherwise increment; once `tooltipTicks >= tooltipDelayTicks`
(~0.5s) and the hovered widget has non-empty `Tooltip()`, show it. Press hides
it. `drawTooltip` paints a bordered box near the cursor, clamped to the surface,
above all overlays. Tooltip text is per-widget via `SetTooltip` on `BaseWidget`,
so any widget can have one.

---

## 14. Text editing internals

### 14.1 `TextField` (`ui/textfield.go`) — single line

State: `runes []rune`, `caret int`, `anchor int` (selection = `[min,max)`,
empty when `anchor==caret`), `scrollX` (horizontal offset to keep the caret
visible), `dragging`. `caretIndexForX(line, x, face)` maps a pixel x to a caret
index by measuring prefixes (shared with `TextArea`). Selection: click+drag,
Shift+navigation, Ctrl+A; editing replaces the selection. Clipboard:
Ctrl+C/X/V via the context clipboard; paste flattens newlines (single-line).

### 14.2 `TextArea` (`ui/textarea.go`) — multi line

State: `lines [][]rune`, `caretRow`/`caretCol`, `anchorRow`/`anchorCol`,
`scrollY`, `wrap`, `wrapWidth`. Editing handles newline insertion, line joining
on backspace/delete at edges, and selection across lines.

**Soft word-wrap** uses a unified **visual-rows** model: `rows()` maps logical
lines to `visRow{lr, start, end}` segments via `wrapSegments` (greedy break at
spaces, hard-break overlong words). With wrap **off**, each logical line is one
visual row, so the non-wrap path is just the general case. Rendering, caret
position, click-to-caret, vertical scrolling, and Up/Down navigation all operate
on visual rows; Up/Down preserves the caret's x. Selection highlighting is
computed per visual row.

---

## 15. Widget catalogue (quick reference)

| Widget | File | Notes |
|---|---|---|
| `Container` | container.go | grouping, layout host, bg/border/padding |
| `Label` | label.go | single line text, `Align`, color/font overrides |
| `Button` | button.go | text and/or icon, hover/press/focus, keyboard activate |
| `Checkbox` | checkbox.go | check glyph, `OnChange(bool)` |
| `RadioButton`/`RadioGroup` | radio.go | circular indicators, group exclusivity |
| `TextField` | textfield.go | single-line edit + selection + clipboard |
| `TextArea` | textarea.go | multi-line, soft-wrap, selection, clipboard |
| `ScrollView` | scrollview.go | viewport + wheel + draggable thumb |
| `List` | list.go | selectable rows, wheel, keyboard, per-row hover |
| `Table` | table.go | header + weighted columns, selectable scrollable body |
| `DropdownCombo` | dropdown.go | opens a popup `List` |
| `MenuBar`/`Menu`/`MenuItem` | menu.go | popup menus, hover-switch between titles |
| `Slider` | slider.go | drag + arrow keys, `[0,1]` |
| `ProgressBar` | progressbar.go | non-interactive fill |
| `TabContainer` | tabs.go | tab strip; all panes mounted (keep state), only active shown |
| `SplitPane` | splitter.go | draggable divider, ratio + min sizes, nests |
| `Image` | image.go | displays `render.Image` with `FitContain/Stretch/None` |

Self-drawing multi-row widgets (`List`, `Table`, `MenuBar`, `TabContainer`,
`SplitPane`) draw their "rows/cells/divider" themselves and return only their
real child widgets (or none) from `Children()`, so hit-testing returns the
widget itself for the self-drawn regions and it handles those clicks. They get
per-row hover via the pointer-move-to-hovered dispatch (§10.3).

---

## 16. Known limitations & invariants

- **Single OS window.** EBiten runs one window per process; true multi-window
  is out of scope without a different backend. An in-surface "window" layer
  could be added (parked).
- **Full redraw every frame.** No dirty-region optimization; fine for typical
  UIs and simplest against EBiten's model.
- **HiDPI** is 1:1 logical=physical; no device scale factor yet.
- **Grid** has no cell spanning.
- **No accessibility** bridge (EBiten surfaces have no OS a11y tree to feed).
- **Clipboard** default is in-process; OS integration is opt-in via
  `WithClipboard`.
- **IME / complex text input** is not handled (relies on `AppendInputChars`).
- **Coordinates are absolute**; a widget that overflows its parent's bounds
  won't be found by `hitTest` through the parent (the parent's bounds gate the
  recursion).
- **Viewport widgets must under-report `MinSize`** (§9.5).

---

## 17. Testing approach

- **White-box** tests (`package ui`) cover layout math, event dispatch, focus,
  selection, wrap, color roles, tooltip timing, etc. Pure logic, no GPU.
- **Black-box** tests (`package ui_test`) exercise the exported API as a
  consumer would, plus godoc `Example` functions that compile-check usage.
- **Headless-safety**: tests avoid creating GPU resources. Text/layout use the
  bundled font (CPU metrics). Image tests use a `fakeImage` stub of
  `render.Image` and only assert the decode **error** path (which returns before
  any EBiten call). Widgets needing a font are mounted via a real `App` (whose
  default font is available without a window).
- **Tick-based timing** (tooltips) is deterministic in tests because it counts
  `Update` calls rather than wall-clock.
- The `geom`, `render`, and `theme` packages have their own focused tests.

GUI examples can't run in CI (no display) but all compile; they're the manual
"does it actually render" check, run locally with `go run ./examples/<name>`.
```
