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

### 3.3 `Image` & `RenderTarget` (`render/canvas.go`)

`Image` is an opaque bitmap handle; only `Size()`.

`RenderTarget` is an **offscreen drawing surface whose contents persist between
frames**. It embeds `Image` (so it can be blitted with `Canvas.DrawImage`) and
adds `Canvas()` (a canvas that draws *into* the target, additively), `Clear(c)`,
and `Dispose()`. It exists so callers can draw expensive, slowly-changing
content once and cheaply blit the result each frame instead of re-issuing every
draw call per frame. Backend impl is `renderTarget` in `internal/ebiten/image.go`
(an `*ebiten.Image` plus a persistent `canvas` bound at scale 1); created via
`ui.NewRenderTarget(w, h)` → `ebitenbackend.NewRenderTarget`. `DrawImage` blits
either an `imageHandle` or a `renderTarget` through the shared `ebitenImager`
interface. The target draws at 1:1 (logical=physical), so on a HiDPI surface its
contents are upscaled like any bitmap (a noted limitation; a future option could
size it to physical pixels). The paint example uses this: finished strokes are
baked into a target and only the in-progress stroke is drawn live, so per-frame
cost is roughly constant regardless of how much has been drawn.

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
them; unmapped keys become `KeyUnknown`). Besides the concrete `ModShift`/
`ModControl`/`ModAlt`/`ModMeta`, there is a synthetic **`ModPrimary`** — the
platform's primary shortcut modifier, which the backend sets to Command (Meta)
on macOS and Control elsewhere (`primaryIsMeta = runtime.GOOS == "darwin"`),
alongside the concrete bit. Widgets test `ModPrimary` for clipboard/select-all
shortcuts (`TextField`/`TextArea`) so chords follow the host convention (⌘C on
macOS, Ctrl+C on Windows/Linux).

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
OS-backed one via `ui.WithClipboard`. An OS-backed implementation ships in the
opt-in top-level `clipboard` package (`clipboard.New() (render.Clipboard,
error)`), backed by `golang.design/x/clipboard`. It lives in its own package —
not wired in by default — so the core `ui`/`render` packages stay
dependency-free and cross-platform; only apps that import `clipboard` pull in
the platform dependency. CGo-free on Windows; reuses the CGo/X11 toolchain
EBiten already requires on macOS/Linux (no external `xclip` binary).

---

## 4. The EBiten backend (`internal/ebiten`)

Package name `ebitenbackend`. The only place that touches EBiten.

### 4.1 Canvas (`canvas.go`)

`canvas` wraps an `*ebiten.Image` and implements `render.Canvas`.

- **HiDPI scaling.** The canvas carries a `scale` (device scale factor) bound
  each frame by `reset(surface, scale)`. Framework coordinates are **logical**
  pixels; the surface is **physical** pixels. Every draw multiplies its
  coordinates (and line widths / radii) by `scale`, so widgets stay
  device-independent while rendering happens at native resolution — crisp on
  Retina. `Size()` reports the logical size (physical ÷ scale). See §4.5 for how
  the scale reaches here and §19/`design.md` task 19 for the overall HiDPI model.
- **Clip stack via sub-images.** The canvas holds a stack of `clipEntry{target
  *ebiten.Image, rect geom.Rect}` where `rect` is **logical**. `PushClip(r)`
  intersects `r` (logical) with the current clip, then takes
  `base.SubImage(phys(clip))` — `clip` scaled to physical pixel bounds — as the
  new target and pushes it. Drawing goes to the top-of-stack target, so anything
  outside the clip is discarded by EBiten's sub-image bounds.
- Shapes use `ebiten/v2/vector` (`FillRect`, `StrokeRect`, `StrokeLine`,
  `FillCircle`, `StrokeCircle`) — note `vector.FillRect` (not the deprecated
  `DrawFilledRect`).
- Text uses `ebiten/v2/text/v2`; `DrawText` translates to `pos×scale` and
  color-scales, **rasterizing glyphs at physical size** (a shallow copy of the
  face at `Size×scale`) so text stays sharp; `MeasureText` delegates to the
  face's `Measure`, which stays at logical size so layout is unaffected.
- `toImageRect` rounds a float `Rect` outward to integer pixel bounds;
  `scaleRect`/`phys` map a logical rect to physical coords / pixel bounds.

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
  - `LayoutF(outsideW, outsideH)` (with `Layout` kept as the integer fallback)
    reads `ebiten.Monitor().DeviceScaleFactor()`, reports the **logical**
    (device-independent) size to `hooks.Resize` when it changes, and returns the
    **physical** surface size (`outside × scale`) so EBiten maps the offscreen
    surface 1:1 to the window's physical pixels — the basis for crisp HiDPI
    rendering. The scale is handed to the canvas each `Draw` (§4.1).

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
then strokes the border. `Add` mounts the child; `Remove` detaches it (leaving
the widget intact so it can be re-`Add`ed elsewhere — used to move an item
between panels on a drag-and-drop drop, §11b).

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
7. **Right-press** over a widget with a context menu (found by walking up the hit
   chain via `contextTarget`) opens that menu at the cursor and consumes the
   press — see §11. Widgets without one fall through to normal press handling
   (so e.g. paint's right-drag-erase still works).
8. Press → focus the nearest focusable in the hit chain, dispatch `PointerDown`,
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

Per `KeysPressed`: `Escape` closes the top popup; `Tab` moves focus; then
**global accelerators** are checked (§10.8); otherwise `KeyDown` → focused widget
(bubbles). `KeysReleased` → `KeyUp`; `Runes` → `Text`. Modifiers (Shift/Ctrl/...)
ride on every event.

### 10.8 Accelerators & context menus (`ui/accelerator.go`)

- **Accelerators.** `App.AddAccelerator(key, mods, action)` registers a global
  chord, checked in `dispatchKeyboard` *before* the focused widget — so it takes
  precedence (don't bind chords a focused widget needs, like Primary+C, unless
  intended). Matching is exact over a **normalized** modifier set (`normalizeMods`
  keeps only Primary/Shift/Alt and drops the redundant concrete Control/Meta
  bit), so an accelerator registered with `ModPrimary` fires for Ctrl+key on
  Windows/Linux and Cmd+key on macOS, but Primary+Shift+key does *not* fire a
  Primary-only binding.
- **Context menus.** Any widget gets one via `BaseWidget.SetContextMenu(items…)`
  (stored like the tooltip; exposed through the new `Widget.ContextMenu()` method,
  defaulted by `BaseWidget`). On a right-press the dispatcher finds the nearest
  hit-chain widget with a menu and calls `App.ShowContextMenu(at, items…)`, which
  opens a non-modal popup (built by the shared `menuPanel` helper, also used by
  `MenuBar`) at the cursor, clamped on-screen. Choosing a row runs its action
  after closing the menu (so an action that opens a dialog isn't torn down with
  it); clicking elsewhere or Escape dismisses it.

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
  `MenuBar` opens a `Container` of `menuRow`s; `App.ShowContextMenu` opens the
  same kind of menu at the cursor on right-click (§10.8); `App.ShowModal`/
  `ShowMessage` center a panel as a modal dialog. `DialogButton` wraps an action
  + auto-close.

(There is no in-surface "window" layer yet; multi-window is parked because
EBiten is single-window — see §16.)

---

## 11a. Frame hook & animations (`ui/animation.go`)

The App drives per-frame work from `update`, before layout and dispatch, so any
value changes are laid out and drawn the same frame:

- **`App.OnFrame(func(dt float64))`** — register callbacks run once per frame with
  the per-frame delta. `dt` is a **fixed** `nominalFrameDelta` (1/60 s). EBiten
  ticks `Update` at a constant TPS (catching up under load), so a fixed step is
  both wall-clock-accurate and deterministic for tests — the same rationale as
  tooltip tick-timing (§13). Real elapsed time is deliberately *not* used.
- **`App.Animate(duration, ease, apply)`** and **`App.Tween(duration, from, to,
  ease, set)`** — start an `*Animation`. Each frame `advanceFrame` adds `dt` to
  the animation's elapsed time, computes `t∈[0,1]`, applies `ease`, and calls
  `apply(t)` (Tween maps `t` to `from→to`). On completion `apply` is called once
  with `t=1` and `OnDone` fires; `Stop()` cancels (no `OnDone`). `Done()` reports
  finished-or-stopped. Easings: `Linear`, `EaseIn`, `EaseOut`, `EaseInOut`
  (custom `Easing` funcs may overshoot for spring-like motion).
- **Reentrancy:** `advanceFrame` snapshots the active list and collects newly
  started animations into a fresh slice, so an `apply`/`OnDone` that starts or
  stops animations is safe; new ones first advance the next frame.

Animations don't need to mark the tree dirty to show: the App redraws every
frame, so a tweened paint property is visible immediately; a tweened layout-
affecting property still calls `Invalidate` through its setter and re-lays-out
the same frame. Widgets can already be animated via their public setters from
app code; exposing the animator to widgets internally (through `treeContext`) is
a straightforward future extension. See `examples/animation`.

---

## 11b. Drag-and-drop (`ui/dragdrop.go`)

In-process DnD layered on the existing pointer capture (§10.3), not a new input
path. The pieces:

- **Payload.** `DragData{Type string; Value any}` — `Type` is a free-form tag
  drop targets match on; `Value` is the in-process payload.
- **Configuration on `BaseWidget`** (read by the dispatcher via accessors, the
  same pattern as `ContextMenu()`/tooltip — not routed through `HandleEvent`):
  `SetDragSource(func() *DragData)`, `SetDragGhost(DragGhost)`,
  `SetDropTarget(func(DragData) bool)`, and the `OnDragEnd`/`OnDragEnter`/
  `OnDragLeave`/`OnDragOver`/`OnDrop` callbacks. Accessors (`dragSourceFn()`,
  `dropAcceptFn()`, …) default to nil on `BaseWidget`, so any widget opts in by
  calling a setter.
- **Session** (`*dragSession` on the `App`, one at a time, stored beside
  `pressTarget`): `{data, source, over, ghost, origin, started}`. `origin` is the
  press point; `started` flips once movement passes `dragThreshold` (4px).

**Dispatcher hooks (`dispatchPointer`).** All inside the existing capture block:

1. *Press* — when `pressTarget` is set, if the hit chain has a drag source, a
   *pending* session is recorded (`started=false`). Normal `PointerDown` still
   fires, so the widget stays clickable.
2. *Move* — if pending and not yet `started` and `|pos-origin| > dragThreshold`,
   call the source provider; a nil return aborts, otherwise `started=true` and the
   ghost is built. While `started`, the App **intercepts** moves: it resolves the
   nearest accepting drop target under the cursor (`dropTarget`, walks `Parent()`
   like `contextTarget`), fires `OnDragLeave`/`OnDragEnter` on change and
   `OnDragOver` each move, and **suppresses the normal `PointerMove` to the
   source** (so a dragged slider doesn't also slide).
3. *Release* — if `started`: `over != nil && over.OnDrop(data,pos)` → success →
   `source.OnDragEnd(true)`, else `OnDragEnd(false)`; the derived `Click`
   (§10.3) is **suppressed** for that release. If never `started`, behaves exactly
   as before (normal click).

**Cancel.** `Escape` in `dispatchKeyboard` cancels an active drag
(`OnDragEnd(false)`) and takes precedence over closing popups while a drag runs.

**Ghost rendering.** `App.draw` calls `drawDrag(c)` between `drawOverlays` and
`drawTooltip`, so the ghost sits above the root and overlays but below tooltips.
Default ghost is a translucent snapshot of the source captured into a
`render.RenderTarget` (§3.3) and blitted at the cursor offset by `pos-origin`;
overridable via `SetDragGhost`, or nil so targets indicate the drop via
`OnDragOver`.

Only the left button initiates a drag. Deferred: edge auto-scroll, copy/move drop
effects. See `examples/dragdrop`.

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

## 13a. Tree (`ui/tree.go`)

A self-drawing hierarchical list, structured like `List` (§15) but over a node
tree instead of a flat slice.

- **Model.** `TreeNode{Label, Value any, children, parent, expanded}`. Build with
  `NewTreeNode(label, children…)` / `Add`; hand the top-level nodes to
  `NewTree(roots…)`. `Value` carries caller data so a handler can recover the
  model object behind a row.
- **Flattening.** `rows()` walks the forest preorder, descending only into
  `expanded` parents, returning `[]treeRow{node, depth}` — the visible rows in
  display order. Everything (draw, hit-testing, keyboard) works off this list, so
  collapsed subtrees simply don't appear. It is recomputed on demand (trees are
  small); no cached flat list to invalidate.
- **Layout.** Row height mirrors `List` (`"Ag"` + padding). Each row reserves a
  fixed chevron column then indents the label by `depth*treeIndent`. The chevron
  (a 2-line `>`/`⌄` drawn with `DrawLine`) is painted only for parents.
- **Input.** Click in the chevron column toggles expansion; click elsewhere
  selects (so selecting never collapses). Wheel scrolls; Up/Down move the
  selection, Home/End jump, **Right** expands (or steps to the first child if
  already open), **Left** collapses (or steps to the parent), Enter toggles a
  parent or fires `OnActivate` on a leaf. `OnSelect` fires only on change.
- Like other self-drawing widgets it exposes no child `Widget`s, so hit-testing
  resolves to the `Tree` itself and it handles all row interaction.

## 13b. Toasts (`ui/toast.go`)

Transient, non-interactive notifications owned by the `App` (not tree widgets),
in the same family as tooltips/popups.

- **Lifecycle.** `App.ShowToast(msg, opts…)` appends a `Toast{message, kind,
  duration, elapsed}` to `App.toasts` and returns the handle. `advanceToasts(dt)`
  runs each frame from `update` (right after `advanceFrame`), ages every toast and
  drops expired ones (compacting in place and niling the tail so dropped toasts
  aren't pinned by the backing array). `Dismiss()` shortens `duration` to
  `elapsed+fade-out` so an early dismiss still fades rather than cutting.
- **Opacity.** `alpha()` ramps 0→1 over `toastFadeIn`, holds at 1, then 1→0 over
  the final `toastFadeOut`. Timing counts fixed frame deltas (like tooltips/
  animations), so it is deterministic in tests.
- **Rendering.** `drawToasts` (called in `App.draw` between `drawDrag` and
  `drawTooltip`) stacks them from the bottom-right upward, newest at the bottom,
  each sized to its (possibly multi-line) text. Color comes from `ToastKind`
  (info → theme primary/on-primary; success/warning/error → fixed green/amber/red
  with white text); every color is run through `withAlpha` for the fade. Toasts
  are never hit-tested, so input passes through to the widgets beneath them.

## 13c. Stepper & Spinner

- **`Stepper` (`ui/stepper.go`)** is a self-drawn numeric input — no free-form
  text entry, so values stay valid by construction. State is `value`, `min`,
  `max`, `step`, `decimals`; `SetValue` clamps via `clampF` and fires `OnChange`
  only on a real change. It draws the formatted value (`strconv.FormatFloat`)
  plus an up/down button column on the right; `HandleEvent` steps on a click in
  the column's top/bottom half, on Up/Down keys (Home/End jump to min/max), and
  on the wheel, tracking per-half hover. Focusable like the other inputs.
- **`Spinner` (`ui/spinner.go`)** is an indeterminate busy indicator: a ring of
  `spinnerDots` dots with a bright rotating head and fading tail. It **advances
  in `Draw`** (by `speed·nominalFrameDelta` per frame) rather than via a timer —
  the framework redraws every frame, so this stays smooth and is deterministic
  under the `guitest` step model. `Start`/`Stop` toggle the animation (a stopped
  spinner freezes; hide it with `SetVisible(false)`). Dots are drawn with
  `FillCircle` and `withAlpha` for the fade, in `RolePrimary`.

## 13d. DatePicker (`ui/date.go`)

An inline month calendar over `time.Time`, self-drawn like `List`/`Tree`.

- **Model.** `selected`, `visible` (first of the displayed month) and `today` are
  all date-only (`dayOf` truncates to midnight in the value's location). Options
  set the value, the today marker (handy for deterministic tests) and the
  `firstWeekday` (Sunday default). `SetValue` clamps the view to the new month
  and fires `OnChange` only on a real day change; `ShowMonth`/`stepMonth` move the
  view without touching the selection.
- **Grid.** A fixed 8×7 layout (header, weekday labels, 6 week rows) sized to the
  content rect (`grid()` returns origin + cell size), so it scales with the
  widget. `offset()` is the count of leading cells before day 1;
  `dateForCell`/`cellForDate` convert between grid index and date via
  `AddDate`, so prev/next-month days fall out naturally (drawn dimmed, still
  selectable).
- **Input.** Click selects a day (or the corner header cells step the month);
  the wheel steps months; Left/Right move ±1 day, Up/Down ±1 week, PageUp/
  PageDown ±1 month (all via `SetValue`, so the view follows the selection).
  Selected day filled `RolePrimary`; today outlined `RoleAccent`; hover lightened.

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

### 14.3 IME / preedit

Input method editors have three concerns; the framework machinery for all three
is wired, but only committed text is deliverable on EBiten v2.9.9 (which exposes
no preedit or candidate-window API — see §16).

- **Seam.** `render.Composition{Text, Caret, SelLo, SelHi}` is a per-frame field
  on `InputState` (zero = not composing); it is a **level**, not an edge —
  supporting backends report the current preedit every frame while composing.
  `render.IMEController{SetIMEEnabled, SetIMERect}` is an optional capability the
  framework type-asserts on the `Driver`.
- **Dispatch (`dispatchKeyboard`).** Before the key/rune loops, if the frame has
  a composition (or one was active last frame) an `EventComposition` carrying
  `Event.Comp` is sent to the focused widget, and `App.composing` is updated.
  **While composing, KeyDown/KeyUp are not forwarded** to the focused widget (the
  IME owns those keys); committed runes still flow as `EventText`. A commit frame
  carries `Comp.Text==""` (clears the preedit) followed by the committed `Runes`.
- **App wiring.** `App.ime` caches the driver if it implements `IMEController`.
  `setFocus` calls `SetIMEEnabled(true/false)` as an editable widget gains/loses
  focus; `update` reports the focused widget's `imeCaretRect()` via `SetIMERect`
  each frame. Editable widgets are detected by the unexported `imeEditable`
  interface (`imeCaretRect() (geom.Rect, bool)`), implemented by `TextField`/
  `TextArea`.
- **Widgets.** Both hold `preedit`/`preeditCaret`. `setPreedit` replaces the
  selection on composition start (once), then stores the preedit; focus-loss
  clears it. `Draw` inserts the preedit inline at the caret (in `TextField`, into
  the displayed string; in `TextArea`, into the caret's visual row), underlines
  it, and places the caret within the preedit. The preedit is never written into
  the committed `runes`/`lines`; the commit arrives separately as `EventText`.
- **Testing.** Because composition is just `InputState` fields, the `guitest`
  harness drives the whole lifecycle headlessly via `Compose`/`CommitText`/
  `CancelComposition`, independent of EBiten's limitations.

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
| `Tree` | tree.go | hierarchical `TreeNode`s, expand/collapse chevrons, keyboard nav, scroll |
| `Table` | table.go | header + weighted columns, selectable scrollable body |
| `DropdownCombo` | dropdown.go | opens a popup `List` |
| `MenuBar`/`Menu`/`MenuItem` | menu.go | popup menus, hover-switch between titles |
| `Slider` | slider.go | drag + arrow keys, `[0,1]` |
| `Stepper` | stepper.go | numeric value, up/down buttons + arrows + wheel, min/max/step |
| `ProgressBar` | progressbar.go | non-interactive fill |
| `Spinner` | spinner.go | indeterminate busy indicator (animated dot ring) |
| `DatePicker` | date.go | inline month calendar over `time.Time`, month nav, click + keyboard |
| `TabContainer` | tabs.go | tab strip; all panes mounted (keep state), only active shown |
| `SplitPane` | splitter.go | draggable divider, ratio + min sizes, nests |
| `Image` | image.go | displays `render.Image` with `FitContain/Stretch/None` |

Self-drawing multi-row widgets (`List`, `Tree`, `Table`, `MenuBar`,
`TabContainer`, `SplitPane`) draw their "rows/cells/divider" themselves and return only their
real child widgets (or none) from `Children()`, so hit-testing returns the
widget itself for the self-drawn regions and it handles those clicks. They get
per-row hover via the pointer-move-to-hovered dispatch (§10.3).

---

## 16. Known limitations & invariants

- **Single OS window.** EBiten runs one window per process; true multi-window
  is out of scope without a different backend. An in-surface "window" layer
  could be added (parked).
- **Full redraw every frame.** No dirty-region optimization; fine for typical
  UIs and simplest against EBiten's model. Apps with expensive, slowly-changing
  content can opt into caching it themselves via `render.RenderTarget`
  (`ui.NewRenderTarget`) — draw once, blit each frame (§3.3). This is also the
  building block a future framework-level dirty-region pass (design task 15)
  would use.
- **HiDPI** is handled: the driver sizes the offscreen surface to
  logical×`DeviceScaleFactor` and the canvas scales all drawing (including
  physical-size glyph rasterization), so rendering is crisp on Retina while
  widgets work in logical pixels. Not yet covered: reacting to the scale factor
  changing mid-run when a window is dragged between monitors of different DPI
  (the next frame picks it up, but cached scaled state isn't pre-warmed).
- **Grid** has no cell spanning.
- **No accessibility** bridge (EBiten surfaces have no OS a11y tree to feed).
- **Clipboard** default is in-process; OS integration is opt-in via
  `WithClipboard` using the `clipboard` package (§3.6).
- **IME / complex text input**: the framework seam (`render.Composition`,
  `render.IMEController`, `EventComposition`) and inline preedit rendering in the
  text widgets are implemented and tested via `guitest` (§14.3). But **EBiten
  v2.9.9 exposes no preedit or candidate-window API**, so on the EBiten backend
  only *committed* IME text flows (via `AppendInputChars`); preedit is never fed
  and `SetIMERect` is a no-op. The wiring activates without API churn once a
  backend supplies composition.
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

### 17.1 Headless harness (`guitest`)

The `guitest` package is a public, GPU-free implementation of the render seam
plus a `Harness` that drives an `App` frame by frame, so **app authors** (not
just the framework) can integration-test their UIs. It is the headless backend
the design's testing TBD called for.

- **Injection.** `ui.WithDriver(render.Driver)` lets an alternate backend be
  supplied at `NewApp`. `guitest.New(w, h, opts…)` wires a headless `driver`
  plus a deterministic `font` (via `ui.WithFont`), builds the `App`, and calls
  `App.Run` — which the headless driver makes **non-blocking**: it captures the
  `render.Hooks`, fires the initial `Resize`, and returns. The harness then
  invokes those hooks on demand.
- **Stepping.** `Harness.Step()` builds an `InputState` from accumulated input,
  runs the framework's `Update` then `Draw` onto a fresh `Recording`, clears the
  per-frame edges (held buttons/keys persist), and returns the recording. Input
  helpers are low-level (`MoveMouse`/`PressMouse`/`PressKey`/`TypeText`/…) and
  high-level gestures that self-step (`Click`, `RightClick`, `Drag`, `TypeKey`).
- **Recording.** The headless `canvas` records each draw call as an `Op` rather
  than rasterizing (clipping is recorded but not enforced; coordinates are
  absolute). `Recording` exposes query helpers (`HasText`, `TextContaining`,
  `Count`, `OpsOfKind`, `FillsOfColor`, `TextAt`) so a test asserts what was
  painted without a surface.
- **Deterministic font.** `guitest.NewFont(size)` has synthetic, stable metrics
  (advance `0.6·size`/rune, ascent/descent/line-height fixed fractions), so text
  measurement and layout are predictable in assertions and independent of the
  bundled TTF.
- **Caveat.** `ui.NewRenderTarget` still routes to the real backend (it is not
  part of the `Driver` seam), so headless tests must avoid it — e.g. a drag must
  set a custom `DragGhost` rather than rely on the default snapshot ghost. See
  `guitest/harness_test.go` for worked examples.

GUI examples can't run in CI (no display) but all compile; they're the manual
"does it actually render" check, run locally with `go run ./examples/<name>`.
```
