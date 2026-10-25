package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// tree layout constants.
const (
	treeRowPad  = 4  // vertical padding inside a row
	treeIndent  = 16 // horizontal indent added per depth level
	treeChevron = 14 // width reserved for the expand/collapse chevron
	treeGap     = 4  // gap between the chevron and the label
)

// TreeNode is a node in a Tree. Build a hierarchy with NewTreeNode and Add, then
// hand the top-level nodes to NewTree. Value carries arbitrary caller data so a
// selection handler can recover the model object behind a row.
type TreeNode struct {
	Label    string
	Value    any
	children []*TreeNode
	parent   *TreeNode
	expanded bool

	// A cached label width for horizontal scrolling, valid while wOK and the label and
	// font fingerprint still match.
	w      float64
	wLabel string
	wFP    float64
	wOK    bool
}

// NewTreeNode returns a node with the given label and optional children.
func NewTreeNode(label string, children ...*TreeNode) *TreeNode {
	n := &TreeNode{Label: label}
	n.Add(children...)
	return n
}

// Add appends children to the node and returns the node for chaining.
func (n *TreeNode) Add(children ...*TreeNode) *TreeNode {
	for _, c := range children {
		if c == nil {
			continue
		}
		c.parent = n
		n.children = append(n.children, c)
	}
	return n
}

// Children returns the node's child nodes.
func (n *TreeNode) Children() []*TreeNode { return n.children }

// Parent returns the node's parent, or nil for a top-level node.
func (n *TreeNode) Parent() *TreeNode { return n.parent }

// Expanded reports whether the node is showing its children.
func (n *TreeNode) Expanded() bool { return n.expanded }

// Leaf reports whether the node has no children.
func (n *TreeNode) Leaf() bool { return len(n.children) == 0 }

// Tree is a scrollable, selectable hierarchical list of TreeNodes. Rows for
// expanded nodes show their children indented beneath them; a chevron toggles
// expansion. It scrolls with the wheel, selects on click, and supports Up/Down
// to move, Left/Right to collapse/expand (or hop to parent/child) and Enter to
// activate while focused. Like List it draws only the visible rows itself and
// exposes no child widgets.
type Tree struct {
	BaseWidget
	roots      []*TreeNode
	selected   *TreeNode
	hover      *TreeNode
	offset     float64
	bar        barDrag // scrollbar thumb-drag state
	offsetX    float64
	barH       barDrag // horizontal scrollbar thumb-drag state
	hscroll    bool    // rows may be wider than the tree, scrolling across
	focused    bool
	font       render.FontFace
	onSelect   func(*TreeNode)
	onActivate func(*TreeNode)
}

// NewTree returns a Tree showing the given top-level nodes.
func NewTree(roots ...*TreeNode) *Tree {
	return &Tree{BaseWidget: NewBase(), roots: roots}
}

// SetRoots replaces the top-level nodes and clears the selection.
func (t *Tree) SetRoots(roots ...*TreeNode) {
	t.roots = roots
	t.selected = nil
	t.hover = nil
	t.clamp()
	t.Invalidate()
}

// OnSelect registers the handler invoked when the selected node changes.
func (t *Tree) OnSelect(fn func(*TreeNode)) { t.onSelect = fn }

// OnActivate registers the handler invoked when a leaf node is activated
// (Enter while focused, or a double-click on its row). Activating a parent node
// toggles its expansion instead.
func (t *Tree) OnActivate(fn func(*TreeNode)) { t.onActivate = fn }

// activate opens n: a leaf fires OnActivate, a parent toggles its expansion. It is
// what Enter and a double-click on a row both do.
func (t *Tree) activate(n *TreeNode) {
	t.SetSelected(n)
	if !n.Leaf() {
		t.Toggle(n)
	} else if t.onActivate != nil {
		t.onActivate(n)
	}
}

// Selected returns the selected node, or nil.
func (t *Tree) Selected() *TreeNode { return t.selected }

// SetSelected selects n and fires OnSelect if the selection changed.
func (t *Tree) SetSelected(n *TreeNode) {
	if n == t.selected {
		return
	}
	t.selected = n
	if n != nil && t.onSelect != nil {
		t.onSelect(n)
	}
}

// Reveal makes n visible: it expands every ancestor of n (so n is not hidden
// under a collapsed parent) and scrolls the view so n's row is on screen. Use
// it after selecting a node programmatically — e.g. from another view — so the
// selection is actually seen. It is a no-op for a nil node.
func (t *Tree) Reveal(n *TreeNode) {
	if n == nil {
		return
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		p.expanded = true
	}
	t.clamp()
	if i := indexOfNode(t.rows(), n); i >= 0 {
		t.scrollTo(i)
	}
	t.Invalidate()
}

// Expand shows n's children (no-op for a leaf) and re-lays-out.
func (t *Tree) Expand(n *TreeNode) {
	if n != nil && !n.Leaf() && !n.expanded {
		n.expanded = true
		t.clamp()
		t.Invalidate()
	}
}

// Collapse hides n's children and re-lays-out.
func (t *Tree) Collapse(n *TreeNode) {
	if n != nil && n.expanded {
		n.expanded = false
		t.clamp()
		t.Invalidate()
	}
}

// Toggle flips n's expansion state.
func (t *Tree) Toggle(n *TreeNode) {
	if n == nil || n.Leaf() {
		return
	}
	if n.expanded {
		t.Collapse(n)
	} else {
		t.Expand(n)
	}
}

// SetFont overrides the tree's font face (nil falls back to the theme font).
func (t *Tree) SetFont(f render.FontFace) {
	t.font = f
	t.Invalidate()
}

func (t *Tree) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// treeRow pairs a visible node with its indentation depth.
type treeRow struct {
	node  *TreeNode
	depth int
}

// rows returns the flattened list of currently visible rows in display order
// (preorder, descending only into expanded nodes).
func (t *Tree) rows() []treeRow {
	var out []treeRow
	var walk func(nodes []*TreeNode, depth int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, n := range nodes {
			out = append(out, treeRow{node: n, depth: depth})
			if n.expanded && len(n.children) > 0 {
				walk(n.children, depth+1)
			}
		}
	}
	walk(t.roots, 0)
	return out
}

// RowHeight returns the pixel height of a single row.
func (t *Tree) RowHeight() float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	return f.Measure("Ag").H + 2*t.px(treeRowPad)
}

// ContentHeight returns the total height of all visible rows.
func (t *Tree) ContentHeight() float64 {
	return t.RowHeight() * float64(len(t.rows()))
}

// Focusable reports whether the tree can take focus (only when enabled).
func (t *Tree) Focusable() bool { return t.Enabled() }

// MinSize returns the widest visible row plus indentation, and one row tall.
func (t *Tree) MinSize() geom.Size {
	f := t.face()
	if f == nil {
		return geom.Size{}
	}
	if t.hscroll {
		// Like its height, the width is the parent's to give: wide rows scroll.
		return geom.Size{W: minViewport + scrollbarWidth, H: t.RowHeight()}
	}
	var w float64
	for _, r := range t.rows() {
		rw := float64(r.depth)*t.px(treeIndent) + treeChevron + t.px(treeGap) + f.Measure(r.node.Label).W
		w = maxF(w, rw)
	}
	return geom.Size{W: w + 2*t.px(treeRowPad) + scrollbarWidth, H: t.RowHeight()}
}

func (t *Tree) maxOffset() float64 { return maxF(0, t.ContentHeight()-t.geomInfo().viewH) }

func (t *Tree) clamp() {
	if t.offset < 0 {
		t.offset = 0
	}
	if m := t.maxOffset(); t.offset > m {
		t.offset = m
	}
	t.offsetX = minF(maxF(t.offsetX, 0), t.maxOffsetX())
}

// rowAt returns the visible-row index at absolute y, or -1.
func (t *Tree) rowAt(rows []treeRow, y float64) int {
	rh := t.RowHeight()
	if rh <= 0 {
		return -1
	}
	if y >= t.Bounds().Y+t.geomInfo().viewH {
		return -1 // the horizontal bar, if any
	}
	idx := int((y - t.Bounds().Y + t.offset) / rh)
	if idx < 0 || idx >= len(rows) {
		return -1
	}
	return idx
}

// scrollTo adjusts the offset so visible row i is fully in view.
func (t *Tree) scrollTo(i int) {
	rh := t.RowHeight()
	top := float64(i) * rh
	bottom := top + rh
	if top < t.offset {
		t.offset = top
	} else if viewH := t.geomInfo().viewH; bottom > t.offset+viewH {
		t.offset = bottom - viewH
	}
	t.clamp()
}

// indexOf returns the index of node n within rows, or -1.
func indexOfNode(rows []treeRow, n *TreeNode) int {
	for i, r := range rows {
		if r.node == n {
			return i
		}
	}
	return -1
}

// Draw paints the background, the visible rows (chevrons, indented labels, hover
// and selection highlights) and a scrollbar when the content overflows.
func (t *Tree) Draw(canvas render.Canvas) {
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	canvas.FillRect(b, t.ColourOf(RoleSurface))

	rows := t.rows()
	rh := t.RowHeight()
	g := t.geomInfo()
	// Rows are as wide as the widest when scrolling across, so a highlight spans the
	// whole scrolled row; the clip keeps them off the gutter.
	rowW := g.viewW
	if t.hscroll {
		rowW = maxF(t.widest(), g.viewW)
	}
	ox := b.X - t.offsetX

	canvas.PushClip(geom.Rect{X: b.X, Y: b.Y, W: g.viewW, H: g.viewH})
	for i, r := range rows {
		y := b.Y - t.offset + float64(i)*rh
		if y+rh < b.Y || y > b.Y+g.viewH {
			continue // not visible
		}
		row := geom.Rect{X: ox, Y: y, W: rowW, H: rh}
		switch {
		case r.node == t.selected:
			canvas.FillRect(row, t.ColourOf(RolePrimary))
		case r.node == t.hover:
			canvas.FillRect(row, tint(t.ColourOf(RoleSurface), t.tintOf(tintHover)))
		}

		textColour := t.ColourOf(RoleText)
		if r.node == t.selected {
			textColour = t.ColourOf(RoleOnPrimary)
		}

		indent := ox + t.px(treeRowPad) + float64(r.depth)*t.px(treeIndent)
		if !r.node.Leaf() {
			t.drawChevron(canvas, indent, y, rh, r.node.expanded, textColour)
		}
		labelX := indent + treeChevron + t.px(treeGap)
		canvas.DrawText(r.node.Label, geom.Point{X: labelX, Y: vCenterY(f, y, rh)}, f, textColour)
	}
	canvas.PopClip()

	canvas.StrokeRect(b, t.ColourOf(RoleBorder), t.borderW())

	if g.vOver {
		t.drawScrollbar(canvas, b, rh*float64(len(rows)))
	}
	if g.hOver {
		drawBar(canvas, t.scrollTrackH(), false, t.scrollMetricsH(), t.ColourOf(RoleBackground), t.ColourOf(RoleAccent))
		if g.vOver { // the corner where the two gutters meet
			canvas.FillRect(geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y + b.H - scrollbarWidth, W: scrollbarWidth, H: scrollbarWidth}, t.ColourOf(RoleBackground))
		}
	}
}

// drawChevron draws a small right-pointing (collapsed) or down-pointing
// (expanded) chevron centered in the chevron column of a row.
func (t *Tree) drawChevron(canvas render.Canvas, x, y, rh float64, expanded bool, col color.Color) {
	cx := x + treeChevron/2
	cy := y + rh/2
	const s = 3.5
	if expanded {
		// pointing down: \/
		canvas.DrawLine(geom.Point{X: cx - s, Y: cy - s/2}, geom.Point{X: cx, Y: cy + s/2}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx + s, Y: cy - s/2}, geom.Point{X: cx, Y: cy + s/2}, col, 1.5)
	} else {
		// pointing right: >
		canvas.DrawLine(geom.Point{X: cx - s/2, Y: cy - s}, geom.Point{X: cx + s/2, Y: cy}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx - s/2, Y: cy + s}, geom.Point{X: cx + s/2, Y: cy}, col, 1.5)
	}
}

// scrollTrack is the scrollbar gutter along the right edge.
func (t *Tree) scrollTrack() geom.Rect {
	b := t.Bounds()
	return geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: t.geomInfo().viewH}
}

func (t *Tree) scrollMetrics() scrollMetrics {
	return scrollMetrics{content: t.ContentHeight(), view: t.geomInfo().viewH, offset: t.offset}
}

func (t *Tree) drawScrollbar(canvas render.Canvas, b geom.Rect, contentH float64) {
	drawBar(canvas, t.scrollTrack(), true, t.scrollMetrics(), t.ColourOf(RoleBackground), t.ColourOf(RoleAccent))
}

// chevronHit reports whether absolute x falls within the chevron column for a
// node at the given depth.
func (t *Tree) chevronHit(x float64, depth int) bool {
	start := t.Bounds().X - t.offsetX + t.px(treeRowPad) + float64(depth)*t.px(treeIndent)
	return x >= start && x < start+treeChevron
}

// HandleEvent handles hover tracking, clicking (chevron toggles, row selects),
// wheel scrolling, focus and keyboard navigation.
func (t *Tree) HandleEvent(ev *Event) bool {
	if !t.Enabled() {
		return false
	}
	g := t.geomInfo()
	if off, changed, ok := t.bar.handle(ev, t.scrollTrack(), g.vOver, true, t.scrollMetrics()); ok {
		if changed {
			t.offset = off
			t.clamp()
		}
		return true
	}
	if off, changed, ok := t.barH.handle(ev, t.scrollTrackH(), g.hOver, false, t.scrollMetricsH()); ok {
		if changed {
			t.offsetX = off
			t.clamp()
		}
		return true
	}
	switch ev.Type {
	case EventPointerMove:
		rows := t.rows()
		if i := t.rowAt(rows, ev.Pos.Y); i >= 0 {
			t.hover = rows[i].node
		} else {
			t.hover = nil
		}
		return true
	case EventPointerLeave:
		t.hover = nil
		return true
	case EventClick:
		rows := t.rows()
		i := t.rowAt(rows, ev.Pos.Y)
		if i < 0 {
			return true
		}
		r := rows[i]
		if !r.node.Leaf() && t.chevronHit(ev.Pos.X, r.depth) {
			t.Toggle(r.node)
		} else {
			t.SetSelected(r.node)
		}
		return true
	case EventDoubleClick:
		rows := t.rows()
		if i := t.rowAt(rows, ev.Pos.Y); i >= 0 && ev.Button == render.MouseLeft {
			r := rows[i]
			// A double-click on the chevron is two toggles already; leave it.
			if !r.node.Leaf() && t.chevronHit(ev.Pos.X, r.depth) {
				return true
			}
			t.activate(r.node)
		}
		return true
	case EventWheel:
		// A sideways wheel, or Shift with the ordinary one, scrolls across when there
		// is somewhere to go.
		if dx := wheelAcross(ev); dx != 0 && t.maxOffsetX() > 0 {
			t.offsetX -= dx * wheelStep
			t.clamp()
			return true
		}
		t.offset -= ev.Wheel.Y * wheelStep
		t.clamp()
		return true
	case EventFocusGained:
		t.focused = true
		return true
	case EventFocusLost:
		t.focused = false
		t.hover = nil
		return true
	case EventKeyDown:
		return t.handleKey(ev.Key)
	}
	return false
}

func (t *Tree) handleKey(k render.Key) bool {
	rows := t.rows()
	switch k {
	case render.KeyDown:
		t.moveSelection(rows, 1)
	case render.KeyUp:
		t.moveSelection(rows, -1)
	case render.KeyHome:
		t.selectIndex(rows, 0)
	case render.KeyEnd:
		t.selectIndex(rows, len(rows)-1)
	case render.KeyRight:
		t.expandKey(rows)
	case render.KeyLeft:
		t.collapseKey(rows)
	case render.KeyEnter:
		if t.selected == nil {
			return true
		}
		t.activate(t.selected)
	default:
		return false
	}
	return true
}

func (t *Tree) moveSelection(rows []treeRow, delta int) {
	if len(rows) == 0 {
		return
	}
	idx := indexOfNode(rows, t.selected)
	if idx < 0 {
		if delta > 0 {
			t.selectIndex(rows, 0)
		} else {
			t.selectIndex(rows, len(rows)-1)
		}
		return
	}
	t.selectIndex(rows, idx+delta)
}

func (t *Tree) selectIndex(rows []treeRow, i int) {
	if i < 0 || i >= len(rows) {
		return
	}
	t.SetSelected(rows[i].node)
	t.scrollTo(i)
}

// expandKey expands a collapsed parent, or moves to the first child if already
// expanded.
func (t *Tree) expandKey(rows []treeRow) {
	n := t.selected
	if n == nil {
		t.selectIndex(rows, 0)
		return
	}
	if !n.Leaf() && !n.expanded {
		t.Expand(n)
		return
	}
	if n.expanded && len(n.children) > 0 {
		t.SetSelected(n.children[0])
		t.scrollTo(indexOfNode(t.rows(), n.children[0]))
	}
}

// collapseKey collapses an expanded parent, or moves to the parent node.
func (t *Tree) collapseKey(rows []treeRow) {
	n := t.selected
	if n == nil {
		return
	}
	if !n.Leaf() && n.expanded {
		t.Collapse(n)
		return
	}
	if n.parent != nil {
		t.SetSelected(n.parent)
		t.scrollTo(indexOfNode(t.rows(), n.parent))
	}
}
