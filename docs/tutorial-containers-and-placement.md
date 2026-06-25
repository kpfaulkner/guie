# Tutorial: containers and widget placement

The [basics tutorial](tutorial-basics.md) introduced containers and layouts just
far enough to build a window. This one goes deep on the *placement* model: how a
container decides where each child goes, what each layout actually does with
weight, alignment, padding and spans, and how to compose them into real screens.

If you've ever wondered "why is my button stretched across the whole window?" or
"how do I make this panel take the leftover space?", this is the tutorial that
answers it.

> Everything here lives in `ui`, with value types (insets, alignment, direction)
> from `geom`. No Ebiten imports in application code.

---

## 1. The placement model

A `Container` doesn't position its own children. It holds them and delegates the
*where* to a **layout**:

```go
c := ui.NewContainer()
c.SetLayout(ui.VBox(8))   // the layout decides positions
c.Add(child)              // the container just holds children
```

Every layout does two things, and understanding them explains every surprise you
will hit:

- **Measure** — "how much space do these children need at minimum?" This is the
  container's `MinSize`. A label measures its text; a container measures its
  layout, which measures *its* children, all the way down.
- **Arrange** — "given this rectangle, where does each child go?" The layout
  receives the **content rectangle** (the container's bounds minus its padding)
  and sets each child's bounds inside it.

So sizing flows *up* the tree (children tell parents their minimums) and
positioning flows *down* (parents hand rectangles to children). Keep that
two-way flow in mind and layouts stop being mysterious.

### Padding and the content rectangle

A container has **padding** — inner space reserved around its children. The area
left for children is `Bounds` inset by padding, called the **content rectangle**:

```go
c.SetPadding(geom.UniformInsets(16))                 // 16px on all sides
c.SetPadding(geom.Insets{Top: 8, Left: 16, Right: 16, Bottom: 8}) // per-side
```

`geom.UniformInsets(16)` is shorthand for the same value on all four sides; set
the `Insets` fields directly when you want different amounts per edge. Padding is
the container's own breathing room; **spacing** (below) is the gap *between*
children — they're different knobs.

---

## 2. The two per-child controls: weight and alignment

When you `Add` a child you can attach per-child options. Two of them drive almost
all placement, and they act on **different axes**:

- **`ui.Weight(n)`** acts on the layout's **main axis** — the direction it
  stacks. It's the child's share of *leftover* space. `Weight(0)` (the default)
  means "just take your minimum size"; a positive weight grabs a proportional
  slice of whatever's left after the minimums are satisfied.
- **`ui.Align(a)`** acts on the **cross axis** — across the stacking direction.
  It decides how the child sits in the space the layout gives it:
  `geom.AlignStart`, `AlignCenter`, `AlignEnd`, or `AlignStretch` (the default).

The single most common confusion: **"my button is stretched."** That's
`AlignStretch` — the default — filling the cross axis. Add it with
`ui.Align(geom.AlignStart)` and it shrinks to its natural size, hugging the
leading edge.

```go
col := ui.NewContainer()
col.SetLayout(ui.VBox(8))                 // main axis = vertical
col.Add(header)                           // natural height, stretched full width
col.Add(body, ui.Weight(1))               // grabs all leftover height
col.Add(ui.NewButton("OK"), ui.Align(geom.AlignStart)) // natural size, left-aligned
```

In a `VBox`, `Weight` controls **height** and `Align` controls **horizontal**
placement. In an `HBox` it's the mirror image: `Weight` controls width, `Align`
controls vertical placement. The axes swap with the direction.

---

## 3. Box: rows and columns

`VBox(spacing)` and `HBox(spacing)` are the workhorses. They lay children out in
a single line along the main axis with `spacing` pixels between each, then
distribute leftover main-axis space by weight.

### How leftover space is shared

Each child first claims its minimum. Then any space left over (after minimums and
spacing) is divided among children *in proportion to their weights*:

```go
row := ui.NewContainer()
row.SetLayout(ui.HBox(12))
row.Add(left,  ui.Weight(1))   // gets 1/3 of the leftover
row.Add(right, ui.Weight(2))   // gets 2/3 of the leftover
```

Weight is a *ratio*, not a fixed size — `Weight(1)` and `Weight(2)` mean the same
as `Weight(10)` and `Weight(20)`. A child with no weight never grows; it stays at
its measured minimum no matter how much room there is.

### The classic "label + field" row

A label at its natural width and an input that eats the rest is just an `HBox`
where only the input has weight:

```go
nameRow := ui.NewContainer()
nameRow.SetLayout(ui.HBox(8))
nameRow.Add(ui.NewLabel("Name:"))               // natural width
nameRow.Add(ui.NewTextField(), ui.Weight(1))    // stretches to fill the row
```

### Pushing items apart (a spacer)

There's no special "spring" widget — an empty, weighted container *is* the
spacer. Put one between two groups and it shoves them to opposite ends:

```go
bar := ui.NewContainer()
bar.SetLayout(ui.HBox(8))
bar.Add(ui.NewLabel("Title"))
bar.Add(ui.NewContainer(), ui.Weight(1))        // invisible spacer
bar.Add(ui.NewButton("Settings"))               // pushed to the right edge
```

---

## 4. Grid: rows and columns at once

`ui.NewGrid(columns, spacing)` flows children left-to-right into a fixed number
of equal columns, wrapping to a new row when it runs out. Columns and rows
**fill** the content rectangle — by default every column is equal width and every
row equal height.

```go
grid := ui.NewContainer()
grid.SetLayout(ui.NewGrid(3, 8))    // 3 columns, 8px gaps
for i := 1; i <= 6; i++ {
    grid.Add(ui.NewButton(fmt.Sprintf("Cell %d", i)))
}
```

### Spanning cells

A child can cover several columns and/or rows with `ui.Span(cols, rows)`.
Auto-flow steps over the cells a span already occupies, so a header that spans
the full width is simply:

```go
grid.Add(header, ui.Span(3, 1))     // full-width header across 3 columns
grid.Add(cellA)                     // flows into row 2
grid.Add(wide, ui.Span(2, 1))       // covers two columns
```

`Align` still applies *inside* a cell (or cell block): the default `AlignStretch`
makes the child fill its cell, while `AlignCenter` centres it at natural size.

> Uneven columns or rows? The underlying `Grid` struct has `ColWeights` and
> `RowWeights` fields that weight individual tracks the way `Box` weights
> children — construct the `Grid` directly when you need that.

---

## 5. Stack: children on top of each other

`ui.NewStack()` puts **every** child in the same rectangle, in add order, so
later children draw on top. Each child is placed by its `Align` on *both* axes.

The everyday use is centering one child:

```go
c := ui.NewContainer()
c.SetLayout(ui.NewStack())
c.Add(ui.NewLabel("centred"), ui.Align(geom.AlignCenter))
```

A coloured **panel with a centred label** — the pattern you'll see all over the
examples — is exactly this:

```go
func panel(bg color.Color, text string) *ui.Container {
    c := ui.NewContainer()
    c.SetBackground(bg)
    c.SetLayout(ui.NewStack())
    c.Add(ui.NewLabel(text), ui.Align(geom.AlignCenter))
    return c
}
```

Stacking also lets you overlay widgets deliberately (a badge over an image, a
caption over a photo). For *that* — and for how clicks resolve between
overlapping widgets, plus floating dialogs and menus — see the
[events and stacked widgets tutorial](tutorial-events-and-overlays.md).

---

## 6. Nesting: the one idea that scales

A container *is* a widget, so containers go inside containers. Every real layout
is a tree of boxes: a vertical column whose rows are themselves horizontal rows
is just a `VBox` holding `HBox` children.

The trick to building any screen is to **decompose it outside-in**:

1. What's the outermost direction? (A toolbar on top, content below → `VBox`.)
2. Which piece eats the slack? (The content → `Weight(1)`.)
3. Repeat inside each piece.

```go
// A window: toolbar across the top, then a sidebar beside a content area.
toolbar := ui.NewContainer()
toolbar.SetLayout(ui.HBox(8))
toolbar.Add(ui.NewButton("New"))
toolbar.Add(ui.NewButton("Open"))

body := ui.NewContainer()
body.SetLayout(ui.HBox(12))
body.Add(sidebar, ui.Weight(1))       // 1/4 of the width
body.Add(content, ui.Weight(3))       // 3/4 of the width

root := ui.NewContainer()
root.SetLayout(ui.VBox(10))
root.SetPadding(geom.UniformInsets(12))
root.Add(toolbar)                     // natural height, full width
root.Add(body, ui.Weight(1))          // eats the rest of the height
```

Read it top-down: a `VBox` of `[toolbar, body]`, where `body` is an `HBox` of
`[sidebar, content]`. That nesting — boxes inside boxes — is the whole game.

---

## 7. Drawing panels: background, border, radius

A container can paint itself, which turns any group into a visible card or panel:

```go
card := ui.NewContainer()
card.SetBackground(pal.Surface)            // fill behind the children
card.SetBorder(pal.Border, 1)              // 1px outline
card.SetCornerRadius(8)                     // rounded corners
card.SetPadding(geom.UniformInsets(12))    // keep children off the edges
card.SetLayout(ui.VBox(8))
card.Add(ui.NewLabel("Card title"))
card.Add(body)
```

Children are clipped to the content rectangle, so content never spills past the
padding. Colours typically come from the theme palette (`app.Theme().Palette`) so
panels match the rest of the UI — `Surface`, `Border`, `Text`, `Primary`, and so
on.

---

## 8. Container-like widgets you don't build by hand

Two common arrangements ship as dedicated widgets rather than layouts, because
they need their own interaction:

**`ScrollView`** — a viewport over content taller than the space available. Give
it any widget (usually a tall `VBox`) and it adds a draggable scrollbar; the
content stays fully interactive.

```go
list := ui.NewContainer()
list.SetLayout(ui.VBox(6))
for i := 1; i <= 40; i++ {
    list.Add(ui.NewCheckbox(fmt.Sprintf("Item %d", i)))
}

scroller := ui.NewScrollView()
scroller.SetContent(list)
root.Add(scroller, ui.Weight(1))           // the viewport eats the leftover height
```

**`SplitPane`** — two panes separated by a divider the user can drag to resize.
`HSplit`/`VSplit` take the two children, an optional starting `SplitRatio`, and
optional `SplitMinSizes`. Splits nest like anything else:

```go
right := ui.VSplit(topPane, bottomPane, ui.SplitRatio(0.8))
split := ui.HSplit(list, right, ui.SplitRatio(0.2), ui.SplitMinSizes(80, 120))
root.Add(split, ui.Weight(1))
```

> Runnable demos: [`examples/scroll`](../examples/scroll) and
> [`examples/splitter`](../examples/splitter).

---

## 9. Putting it together

A small app exercising boxes, weight, alignment, a grid with a span, a stacked
panel and a card — resize the window and watch every piece reflow.

```go
// Run with: go run .
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func panel(bg color.Color, text string) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetLayout(ui.NewStack())
	c.Add(ui.NewLabel(text), ui.Align(geom.AlignCenter))
	return c
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — containers & placement"),
		ui.WithSize(720, 480),
	)
	pal := app.Theme().Palette

	// A toolbar: a title on the left, a spacer, a button shoved to the right.
	toolbar := ui.NewContainer()
	toolbar.SetLayout(ui.HBox(8))
	toolbar.Add(ui.NewLabel("Dashboard"))
	toolbar.Add(ui.NewContainer(), ui.Weight(1))           // spacer
	toolbar.Add(ui.NewButton("Settings"), ui.Align(geom.AlignStart))

	// A weighted row: the right panel is twice as wide as the left.
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(12))
	row.Add(panel(pal.Surface, "weight 1"), ui.Weight(1))
	row.Add(panel(pal.Primary, "weight 2"), ui.Weight(2))

	// A 3-column grid with a full-width header that spans all columns.
	grid := ui.NewContainer()
	grid.SetLayout(ui.NewGrid(3, 8))
	grid.Add(panel(pal.Primary, "header (spans 3)"), ui.Span(3, 1))
	for i := 1; i <= 5; i++ {
		grid.Add(panel(pal.Surface, fmt.Sprintf("cell %d", i)))
	}

	// A card: a self-drawn panel grouping a couple of widgets.
	card := ui.NewContainer()
	card.SetBackground(pal.Surface)
	card.SetBorder(pal.Border, 1)
	card.SetCornerRadius(8)
	card.SetPadding(geom.UniformInsets(12))
	card.SetLayout(ui.VBox(8))
	card.Add(ui.NewLabel("A card"))
	card.Add(ui.NewLabel("background + border + radius + padding",
		ui.LabelColour(pal.TextMuted)))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))
	root.Add(toolbar)                 // natural height
	root.Add(row, ui.Weight(2))       // weighted band
	root.Add(grid, ui.Weight(3))      // taller weighted band
	root.Add(card)                    // natural height

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

What to notice:

- The **toolbar** uses a weighted empty container as a spacer to push the button
  to the right edge.
- The **row** splits its width 1:2 by weight; the **grid** bands below take 2 and
  3 shares of the leftover *height* (the `Weight` on `row`/`grid` in the root
  `VBox`).
- The **grid header** spans all three columns via `ui.Span(3, 1)`.
- The **card** has no weight, so it sits at its natural height at the bottom while
  everything above flexes.

---

## 10. Where to go next

- **Events and overlays** — [events and stacked widgets](tutorial-events-and-overlays.md)
  covers input handling and floating dialogs/menus, the natural follow-on to
  stacking.
- **The layout engine in motion** — [`examples/layouts`](../examples/layouts)
  packs VBox, HBox, Grid, Stack, weights and spans into one resizable window.
- **Composite containers** — [`examples/scroll`](../examples/scroll) and
  [`examples/splitter`](../examples/splitter).
- **Back to basics** — if `App`, `Run` and the widget constructors are still
  fuzzy, start with the [basics tutorial](tutorial-basics.md).
