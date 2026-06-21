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
// (Enter while focused). Activating a parent node toggles its expansion instead.
func (t *Tree) OnActivate(fn func(*TreeNode)) { t.onActivate = fn }

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
	return f.Measure("Ag").H + 2*treeRowPad
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
	var w float64
	for _, r := range t.rows() {
		rw := float64(r.depth)*treeIndent + treeChevron + treeGap + f.Measure(r.node.Label).W
		w = maxF(w, rw)
	}
	return geom.Size{W: w + 2*treeRowPad + scrollbarWidth, H: t.RowHeight()}
}

func (t *Tree) maxOffset() float64 { return maxF(0, t.ContentHeight()-t.Bounds().H) }

func (t *Tree) clamp() {
	if t.offset < 0 {
		t.offset = 0
	}
	if m := t.maxOffset(); t.offset > m {
		t.offset = m
	}
}

// rowAt returns the visible-row index at absolute y, or -1.
func (t *Tree) rowAt(rows []treeRow, y float64) int {
	rh := t.RowHeight()
	if rh <= 0 {
		return -1
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
	} else if bottom > t.offset+t.Bounds().H {
		t.offset = bottom - t.Bounds().H
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
	canvas.FillRect(b, t.ColorOf(RoleSurface))

	rows := t.rows()
	rh := t.RowHeight()
	overflow := rh*float64(len(rows)) > b.H
	rowW := b.W
	if overflow {
		rowW -= scrollbarWidth
	}

	canvas.PushClip(geom.Rect{X: b.X, Y: b.Y, W: rowW, H: b.H})
	for i, r := range rows {
		y := b.Y - t.offset + float64(i)*rh
		if y+rh < b.Y || y > b.Y+b.H {
			continue // not visible
		}
		row := geom.Rect{X: b.X, Y: y, W: rowW, H: rh}
		switch {
		case r.node == t.selected:
			canvas.FillRect(row, t.ColorOf(RolePrimary))
		case r.node == t.hover:
			canvas.FillRect(row, lighten(t.ColorOf(RoleSurface), 1.25))
		}

		textColor := t.ColorOf(RoleText)
		if r.node == t.selected {
			textColor = t.ColorOf(RoleOnPrimary)
		}

		indent := b.X + treeRowPad + float64(r.depth)*treeIndent
		if !r.node.Leaf() {
			t.drawChevron(canvas, indent, y, rh, r.node.expanded, textColor)
		}
		labelX := indent + treeChevron + treeGap
		canvas.DrawText(r.node.Label, geom.Point{X: labelX, Y: vCenterY(f, y, rh)}, f, textColor)
	}
	canvas.PopClip()

	canvas.StrokeRect(b, t.ColorOf(RoleBorder), 1)

	if overflow {
		t.drawScrollbar(canvas, b, rh*float64(len(rows)))
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

func (t *Tree) drawScrollbar(canvas render.Canvas, b geom.Rect, contentH float64) {
	thumbH := maxF(minThumb, b.H*b.H/contentH)
	var off float64
	if m := t.maxOffset(); m > 0 {
		off = (t.offset / m) * (b.H - thumbH)
	}
	gutter := geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: b.H}
	canvas.FillRect(gutter, t.ColorOf(RoleBackground))
	canvas.FillRect(geom.Rect{X: gutter.X, Y: b.Y + off, W: scrollbarWidth, H: thumbH}, t.ColorOf(RoleAccent))
}

// chevronHit reports whether absolute x falls within the chevron column for a
// node at the given depth.
func (t *Tree) chevronHit(x float64, depth int) bool {
	start := t.Bounds().X + treeRowPad + float64(depth)*treeIndent
	return x >= start && x < start+treeChevron
}

// HandleEvent handles hover tracking, clicking (chevron toggles, row selects),
// wheel scrolling, focus and keyboard navigation.
func (t *Tree) HandleEvent(ev *Event) bool {
	if !t.Enabled() {
		return false
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
	case EventWheel:
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
		if t.selected.Leaf() {
			if t.onActivate != nil {
				t.onActivate(t.selected)
			}
		} else {
			t.Toggle(t.selected)
		}
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
