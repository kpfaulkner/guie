// Command tree demonstrates the Tree widget and Toast notifications. Expand and
// collapse folders with the chevrons (or Left/Right keys), select a node with a
// click (or Up/Down), and activate a leaf with Enter or the button — each action
// raises a transient toast in the corner.
//
// Run with: go run ./examples/tree
package main

import (
	"fmt"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — tree & toast"),
		ui.WithSize(560, 420),
	)

	// Build a small file-system-like hierarchy.
	src := ui.NewTreeNode("src",
		ui.NewTreeNode("ui",
			ui.NewTreeNode("tree.go"),
			ui.NewTreeNode("toast.go"),
			ui.NewTreeNode("list.go"),
		),
		ui.NewTreeNode("render",
			ui.NewTreeNode("canvas.go"),
			ui.NewTreeNode("input.go"),
		),
	)
	docs := ui.NewTreeNode("docs",
		ui.NewTreeNode("design.md"),
		ui.NewTreeNode("internals.md"),
	)
	root := ui.NewTreeNode("project", src, docs)

	tree := ui.NewTree(root)
	tree.Expand(root)
	tree.Expand(src)

	status := ui.NewLabel("Select a node, or activate a file (Enter).")

	tree.OnSelect(func(n *ui.TreeNode) {
		status.SetText("Selected: " + n.Label)
	})
	tree.OnActivate(func(n *ui.TreeNode) {
		app.ShowToast("Opened "+n.Label, ui.WithToastKind(ui.ToastSuccess))
	})

	// A row of buttons that raise toasts of each kind.
	buttons := ui.NewContainer()
	buttons.SetLayout(ui.HBox(8))
	kinds := []struct {
		label string
		kind  ui.ToastKind
	}{
		{"Info", ui.ToastInfo},
		{"Success", ui.ToastSuccess},
		{"Warning", ui.ToastWarning},
		{"Error", ui.ToastError},
	}
	for _, k := range kinds {
		kk := k
		b := ui.NewButton(kk.label)
		b.OnClick(func() {
			app.ShowToast(fmt.Sprintf("%s toast raised", kk.label), ui.WithToastKind(kk.kind))
		})
		buttons.Add(b)
	}

	root2 := ui.NewContainer()
	root2.SetLayout(ui.VBox(12))
	root2.SetPadding(geom.UniformInsets(16))
	root2.Add(ui.NewLabel("Project tree (chevrons or Left/Right to fold; Enter opens a file):"))
	root2.Add(tree, ui.Weight(1))
	root2.Add(status)
	root2.Add(buttons)

	app.SetContent(root2)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
