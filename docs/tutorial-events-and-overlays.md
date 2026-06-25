# Tutorial: events and stacked widgets

The [basics tutorial](tutorial-basics.md) built a static layout of containers and
widgets. This one is about the two things that make a UI feel alive:

1. **Events** — how input (clicks, keys, hover, focus) reaches your widgets, and
   the two ways to listen for it.
2. **Stacking** — putting widgets *on top of* one another: overlapping them in a
   layout, and floating dialogs, dropdowns and toasts above the whole window.

These belong in one tutorial because they're entangled: the moment two widgets
share screen space, "which one gets the click?" becomes a real question. By the
end you'll know exactly how guie answers it.

> As always, application code imports only `ui`, `geom`, `render` and `theme`.
> Event types and the event bus live in `ui`; the few value types (points,
> alignment) come from `geom`.

---

## 1. Two ways to listen

Most of the time you wire behaviour onto a single widget with its `OnXxx`
callback. This is the primary API and the one you'll reach for first:

```go
b := ui.NewButton("Save")
b.OnClick(func() { save() })            // runs when this button is clicked
```

But sometimes you want to observe input *across the whole UI* without attaching a
handler to every widget — logging, analytics, a "you have unsaved changes"
flag, a status bar that reacts to any click. That's the **event bus**, reached
through `app.Events()`:

```go
clicks := 0
app.Events().Subscribe(ui.EventClick, func(ev ui.Event) {
    clicks++
    status.SetText(fmt.Sprintf("%d clicks so far", clicks))
})
```

Every event the framework dispatches is *also* published to the bus, so the two
mechanisms coexist: the button's own `OnClick` still fires, and the subscriber
sees the same click. Subscribers run on the UI goroutine, so they must not block,
but they're free to touch other widgets.

> Runnable demo: [`examples/events`](../examples/events) — Tab between buttons,
> activate with Space/Enter, and watch a bus subscriber count every click.

---

## 2. The event types

An `ui.Event` carries a `Type` plus whichever fields are meaningful for that
type. The everyday ones:

| Event type | When it fires | Useful fields |
| --- | --- | --- |
| `EventClick` | a press and release land on the same widget | `Pos`, `Button` |
| `EventPointerDown` / `EventPointerUp` | mouse button pressed / released | `Pos`, `Button` |
| `EventPointerEnter` / `EventPointerLeave` | cursor moves onto / off a widget | `Pos` |
| `EventPointerMove` | each frame, to the widget capturing the pointer (for drags) | `Pos` |
| `EventWheel` | wheel scrolled over the widget under the cursor | `Wheel` |
| `EventKeyDown` / `EventKeyUp` | key pressed / released, sent to the **focused** widget | `Key`, `Modifiers` |
| `EventText` | a rune is typed, sent to the focused widget | `Rune` |
| `EventFocusGained` / `EventFocusLost` | a widget gains or loses keyboard focus | — |

Two distinctions matter:

- **Pointer events** go to the widget *under the cursor*. A press "captures" its
  target, so the matching `EventPointerUp` (and the derived `EventClick`) is
  delivered to the same widget even if the cursor has since moved off it — this
  is what makes dragging work.
- **Keyboard and text events** go to the **focused** widget, wherever the cursor
  is. Clicking a focusable widget focuses it; `Tab` / `Shift+Tab` move focus to
  the next / previous focusable widget in tree order, wrapping around.

---

## 3. How an event finds its widget

When you only ever need `OnClick`, you can skip this section. But once widgets
overlap, it pays to know the rule. Dispatch happens in two steps.

**Hit-testing** picks the target. guie walks the widget tree depth-first and
prefers later (on-top) children, returning the *top-most* visible widget whose
bounds contain the cursor. "Later child wins" is the whole basis of stacking —
keep it in mind for the next section.

**Bubbling** then delivers the event. It's offered to the target first; if the
target doesn't consume it, it bubbles up through the parent chain until some
widget handles it. (The bus is published to regardless, so a subscriber always
sees the event even if a widget consumed it.)

That's the entire model. Everything about overlapping widgets below is just this
rule — "top-most under the cursor wins" — applied to widgets that share space.

---

## 4. Stacking widgets in a layout: `Stack`

The simplest kind of "on top of each other" is the **`Stack`** layout. Where
`VBox` and `HBox` give each child its own slice of space, `Stack` puts *every*
child in the **same** rectangle, in add order — so later children draw on top of
earlier ones. Each child is positioned within that rectangle by its `Align`.

The most common use is centering a single child:

```go
root := ui.NewContainer()
root.SetLayout(ui.NewStack())                       // overlay in one area
root.Add(ui.NewLabel("centred"), ui.Align(geom.AlignCenter))
```

But because later children sit on top, `Stack` also lets you overlay widgets
deliberately — a badge over an icon, a caption over an image, a "Loading…"
veil over a panel:

```go
card := ui.NewContainer()
card.SetLayout(ui.NewStack())

card.Add(photo)                                      // bottom layer, stretches to fill
card.Add(
    ui.NewLabel("NEW"),
    ui.Align(geom.AlignStart),                       // pinned top-left, drawn on top
)
```

Events follow the rule from §3: the badge is a later child, so where it overlaps
the photo it's the badge that gets the click. Anywhere the badge *isn't*, the
hit falls through to the photo beneath. There's no special "transparency" — the
top-most widget whose bounds contain the cursor simply wins.

> `Stack` only stacks within its container's own bounds. To float something
> above the *entire window* — a dialog, a menu, a notification — you want an
> overlay, which is next.

---

## 5. Floating above everything: overlays

A `Stack` child is still part of the normal layout. Dialogs, dropdowns and
notifications are different: they're drawn *above* the whole widget tree, in
their own stack managed by the `App`, and they can grab or block input. guie
calls these **overlays** (popups). You rarely build one by hand — the App gives
you ready-made helpers.

### Modal dialogs

A **modal** overlay dims the background with a scrim, blocks all input to the
widgets behind it, and stays open until *it* decides to close (a button, or
`Escape`) — clicking the scrim does nothing. `ShowMessage` is the quick path:

```go
app.ShowMessage("Delete item?", "This cannot be undone.",
    ui.DialogButton{Label: "Cancel", OnClick: func() { /* … */ }},
    ui.DialogButton{Label: "Delete", OnClick: func() { doDelete() }},
)
```

Each `DialogButton` runs its handler (if any) and then closes the dialog. With no
buttons you get a single **OK**.

For a custom dialog — say, a prompt with a text field — build any widget and hand
it to `ShowModal`, which centres it and returns a `*ui.Popup` handle you close
yourself:

```go
panel := ui.NewContainer()
panel.SetLayout(ui.VBox(12))
panel.SetPadding(geom.UniformInsets(16))
panel.Add(ui.NewLabel("What's your name?"))
field := ui.NewTextField(ui.Placeholder("type here"))
panel.Add(field)

var p *ui.Popup
ok := ui.NewButton("OK")
ok.OnClick(func() {
    greet(field.Text())
    app.Close(p)                                     // dismiss the modal
})
panel.Add(ok, ui.Align(geom.AlignEnd))

p = app.ShowModal(panel)
```

While the modal is open, the focus traversal from §2 is **confined to the
dialog** — `Tab` can't reach the blocked widgets behind it.

> Runnable demo: [`examples/dialog`](../examples/dialog).

### Non-modal popups

Dropdowns and menus are overlays too, but **non-modal**: they float above the UI
without a scrim and are dismissed by clicking *outside* them or pressing
`Escape`. You get this behaviour for free just by using the widgets — a
`ui.NewDropdown(...)` or a menu manages its own popup internally. The stacking
rule still holds: while the popup is open, the top-most overlay wins the hit, and
a press outside it closes it.

### Toasts

A **toast** is a transient, non-interactive notification that fades in near a
window edge, holds, then removes itself. Toasts stack: raise several and they
queue up rather than overlap.

```go
app.ShowToast("Saved", ui.WithToastKind(ui.ToastSuccess))
app.ShowToast("Could not connect", ui.WithToastKind(ui.ToastError),
    ui.ToastDuration(6))
```

Because toasts are non-interactive, they never steal a click — input passes
straight through to whatever is beneath them.

> Runnable demo: [`examples/tree`](../examples/tree) raises toasts of each kind.

### Tooltips

The lightest overlay of all: set hover-hint text on *any* widget and the App
shows it after the pointer rests on the widget for a moment.

```go
save := ui.NewButton("Save")
save.SetTooltip("Save the current document (Ctrl+S)")
```

> Runnable demo: [`examples/tooltips`](../examples/tooltips).

---

## 6. Putting it together

A small program that uses both halves of this tutorial: a `Stack` overlays a
"DRAFT" badge on a panel; the event bus keeps a running count of every click;
buttons raise a toast and a confirm dialog. Run it, then click around and watch
the bus counter tick on *every* click — including the dialog's own buttons.

```go
// Run with: go run .
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — events & overlays"),
		ui.WithSize(520, 360),
	)

	// --- A Stack: a panel with a badge overlaid in its top-left corner. ---
	panel := ui.NewContainer()
	panel.SetLayout(ui.NewStack())
	body := ui.NewLabel("A panel. The DRAFT badge is stacked on top of it.")
	panel.Add(body, ui.Align(geom.AlignCenter))         // bottom layer
	panel.Add(ui.NewLabel("DRAFT"), ui.Align(geom.AlignStart)) // on top

	// --- Buttons that raise overlays. ---
	toastBtn := ui.NewButton("Notify")
	toastBtn.SetTooltip("Raise a toast notification")
	toastBtn.OnClick(func() {
		app.ShowToast("Hello from a toast", ui.WithToastKind(ui.ToastSuccess))
	})

	delBtn := ui.NewButton("Delete…")
	delBtn.OnClick(func() {
		app.ShowMessage("Delete item?", "This cannot be undone.",
			ui.DialogButton{Label: "Cancel"},
			ui.DialogButton{Label: "Delete", OnClick: func() {
				app.ShowToast("Deleted", ui.WithToastKind(ui.ToastWarning))
			}},
		)
	})

	buttons := ui.NewContainer()
	buttons.SetLayout(ui.HBox(10))
	buttons.Add(toastBtn)
	buttons.Add(delBtn)

	// --- A bus subscriber: counts every click anywhere in the UI. ---
	busLabel := ui.NewLabel("(bus) 0 clicks observed")
	clicks := 0
	app.Events().Subscribe(ui.EventClick, func(ui.Event) {
		clicks++
		busLabel.SetText(fmt.Sprintf("(bus) %d clicks observed", clicks))
	})

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(ui.NewLabel("Stacking + events"))
	root.Add(panel, ui.Weight(1))                       // panel eats the slack
	root.Add(buttons, ui.Align(geom.AlignStart))
	root.Add(busLabel)

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

What to notice while it runs:

- The **badge** overlaps the panel body, yet both are just children of a
  `Stack` — the later child draws on top, exactly as §4 described.
- The **bus counter** increments on every click, including clicks on the
  dialog's buttons, because every dispatched event is published to the bus.
- Opening the **dialog** dims and blocks the window: the `Notify` button behind
  it won't respond, and `Tab` stays inside the dialog until you close it.
- The **toast** never blocks anything — it's non-interactive, so clicks pass
  straight through to whatever is beneath it.

---

## 7. Where to go next

- **Focus and keyboard** — [`examples/events`](../examples/events) for Tab
  traversal and Space/Enter activation; try giving a custom widget keyboard
  behaviour by handling `EventKeyDown`.
- **Drag interactions** — `EventPointerDown` → `EventPointerMove` (capture) →
  `EventPointerUp` is the whole drag lifecycle; see
  [`examples/dragdrop`](../examples/dragdrop) and
  [`examples/paint`](../examples/paint).
- **More overlays** — [`examples/dialog`](../examples/dialog),
  [`examples/tooltips`](../examples/tooltips), and the dropdown/menu widgets in
  [`examples/comprehensive`](../examples/comprehensive).
- **Containers and placement** — if containers, layouts and per-child options
  are still fuzzy, read the
  [containers and widget placement tutorial](tutorial-containers-and-placement.md),
  or start over with the [basics tutorial](tutorial-basics.md).
