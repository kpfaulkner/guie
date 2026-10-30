package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
	"unicode"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
)

// FileDialogMode picks what a file dialog selects.
type FileDialogMode int

const (
	// FileOpen chooses an existing file.
	FileOpen FileDialogMode = iota
	// FileSave chooses a file name, possibly a new one. Choosing a file that
	// already exists asks before replacing it.
	FileSave
	// FileSelectDir chooses a directory. Only directories are listed.
	FileSelectDir
)

// fileDialogConfig holds the options for ShowFileDialog.
type fileDialogConfig struct {
	title, ok, cancel string
	fsys              FileSystem
	start             string
	filters           []FileFilter
	showHidden        bool
	name              string
	defaultExt        string
	noOverwritePrompt bool
	multiple          bool
}

// FileDialogOption configures ShowFileDialog.
type FileDialogOption func(*fileDialogConfig)

// FileDialogTitle sets the dialog's heading (default "Open file", "Save file"
// or "Select folder", by mode).
func FileDialogTitle(s string) FileDialogOption {
	return func(c *fileDialogConfig) { c.title = s }
}

// FileDialogLabels sets the confirm and cancel button labels (default "Open",
// "Save" or "Select", by mode, and "Cancel").
func FileDialogLabels(ok, cancel string) FileDialogOption {
	return func(c *fileDialogConfig) { c.ok, c.cancel = ok, cancel }
}

// FileDialogFS sets the filesystem to browse (default OSFileSystem). Use
// FSFileSystem to browse an io/fs.FS.
func FileDialogFS(fsys FileSystem) FileDialogOption {
	return func(c *fileDialogConfig) { c.fsys = fsys }
}

// FileDialogStart sets the directory the dialog opens in. If it cannot be read
// the dialog falls back to the filesystem's home directory, then its first
// root. The default is the home directory.
func FileDialogStart(dir string) FileDialogOption {
	return func(c *fileDialogConfig) { c.start = dir }
}

// FileDialogFilters adds a filter dropdown. The first filter is selected when
// the dialog opens. With no filters every file is listed and there is no
// dropdown. It has no effect on FileSelectDir, which lists no files.
func FileDialogFilters(f ...FileFilter) FileDialogOption {
	return func(c *fileDialogConfig) { c.filters = f }
}

// FileDialogName prefills the name field of a FileSave dialog with a suggested
// file name, selected so typing replaces it. It stays there while the user
// browses. Other modes ignore it.
func FileDialogName(name string) FileDialogOption {
	return func(c *fileDialogConfig) { c.name = name }
}

// FileDialogDefaultExt is the extension FileSave adds to a name that has none,
// with or without its leading dot: "txt" turns "notes" into "notes.txt". The
// replace prompt then names the file that would really be written.
func FileDialogDefaultExt(ext string) FileDialogOption {
	return func(c *fileDialogConfig) { c.defaultExt = ext }
}

// FileDialogMultiple lets a FileOpen dialog return several files. Ctrl/Cmd+click
// toggles a row, Shift+click or Shift+arrows select a range, Ctrl/Cmd+A selects
// everything, and OK or Enter returns every selected file (directories in the
// selection are skipped) in listing order. The name field shows the selected
// names, quoted when there are several. Other modes ignore it.
func FileDialogMultiple() FileDialogOption {
	return func(c *fileDialogConfig) { c.multiple = true }
}

// FileDialogNoOverwritePrompt makes FileSave return an existing file without
// asking to replace it, for apps that confirm in their own way.
func FileDialogNoOverwritePrompt() FileDialogOption {
	return func(c *fileDialogConfig) { c.noOverwritePrompt = true }
}

// FileDialogShowHidden lists hidden files when the dialog opens. They are hidden
// by default. On Windows a file is hidden if it has the hidden attribute, as in
// Explorer; elsewhere, and in any io/fs.FS, if its name starts with a dot.
func FileDialogShowHidden(on bool) FileDialogOption {
	return func(c *fileDialogConfig) { c.showHidden = on }
}

const (
	fileDialogW      = 640.0 // preferred panel size, clamped to the surface
	fileDialogH      = 420.0
	fileDialogMargin = 24.0 // space kept clear around the panel on a small surface
	fileDialogMinW   = 320.0
	fileDialogMinH   = 240.0
)

// ShowFileDialog shows a modal file dialog. onResult is called exactly once: ok
// is true with at least one path when the user confirmed, and false with a nil
// slice for Cancel, Escape, or the dialog being closed any other way
// (App.Close, App.closeAll). Paths are in the FileSystem's own form: absolute
// for the OS one. Keyboard focus returns to whatever held it before the dialog
// opened. onResult may be nil.
//
// FileOpen returns one path, or every selected file with FileDialogMultiple.
// FileSave returns one path. FileSelectDir returns the directory chosen, or the
// one being browsed if none is selected. An unknown mode reports
// a cancel at once and returns nil.
func (a *App) ShowFileDialog(mode FileDialogMode, onResult func(paths []string, ok bool), opts ...FileDialogOption) *Popup {
	var cfg fileDialogConfig
	switch mode {
	case FileOpen:
		cfg = fileDialogConfig{title: "Open file", ok: "Open", cancel: "Cancel"}
	case FileSave:
		cfg = fileDialogConfig{title: "Save file", ok: "Save", cancel: "Cancel"}
	case FileSelectDir:
		cfg = fileDialogConfig{title: "Select folder", ok: "Select", cancel: "Cancel"}
	default:
		if onResult != nil {
			onResult(nil, false)
		}
		return nil
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.fsys == nil {
		cfg.fsys = OSFileSystem()
	}
	d := &fileDialog{app: a, mode: mode, cfg: cfg, fsys: cfg.fsys, onResult: onResult, sortAsc: true, showHidden: cfg.showHidden}
	d.multi = cfg.multiple && mode == FileOpen
	return d.show()
}

// fileDialog is the state of one open file dialog.
type fileDialog struct {
	app      *App
	mode     FileDialogMode
	cfg      fileDialogConfig
	fsys     FileSystem
	onResult func([]string, bool)
	popup    *Popup
	done     bool

	dir        string
	entries    []fs.FileInfo // dir's contents, as read
	shown      []fs.FileInfo // entries after filter and sort; row i of the table is shown[i]
	sortKey    fileSortKey
	sortAsc    bool
	filter     int
	showHidden bool

	reselecting bool // restoring a selection after a re-sort; selection handlers must not touch the name field

	loadGen  int  // numbers directory reads; only the latest one's result is used
	loadBusy bool // the read numbered loadGen has not come back yet

	typed      typeAhead   // letters typed in the listing, for jumping to an entry
	placeCells []placeCell // the places pane, empty when there is none

	multi    bool   // FileOpen with FileDialogMultiple
	autoName string // the name-field text the dialog itself wrote from the selection

	pathField *TextField
	nameField *TextField
	table     *Table
	status    *Label
	up        *Button
}

// fileDialogPanel is the dialog's root widget. It reports a fixed preferred
// size (ShowModal sizes a popup by MinSize, and a Container's is only as big as
// its children), and catches the keys the focused child does not use.
type fileDialogPanel struct {
	*Container
	size geom.Size
	d    *fileDialog
}

func (p *fileDialogPanel) MinSize() geom.Size { return p.size }

// HandleEvent turns Backspace (outside a text field, which uses it for editing)
// and Alt+Up into "go to the parent directory".
func (p *fileDialogPanel) HandleEvent(ev *Event) bool {
	// Typing in the listing jumps to the entry that starts with what was typed.
	// Text fields consume their own typing, so only the listing's reaches here.
	if ev.Type == EventText && p.d.app.focused == Widget(p.d.table) &&
		!ev.Modifiers.Has(render.ModPrimary) && !ev.Modifiers.Has(render.ModAlt) {
		return p.d.typeAheadRune(ev.Rune)
	}
	if ev.Type != EventKeyDown {
		return false
	}
	if (ev.Key == render.KeyBackspace && ev.Modifiers == 0) || (ev.Key == render.KeyUp && ev.Modifiers.Has(render.ModAlt)) {
		p.d.goUp()
		return true
	}
	return false
}

// columns of the listing, in sortKey order.
var fileDialogColumns = []string{"Name", "Size", "Modified"}

func (d *fileDialog) show() *Popup {
	a := d.app
	pal := a.theme.Palette
	prevFocus := a.focused

	panel := NewContainer()
	panel.SetBackground(pal.Surface)
	panel.SetBorder(pal.Border, a.theme.BorderWidth)
	panel.SetCornerRadius(a.theme.CornerRadius)
	panel.SetLayout(VBox(ThemeGap))
	panel.SetPadding(geom.UniformInsets(16))

	size := geom.Size{
		W: fitDialogSize(fileDialogW, fileDialogMinW, a.surfaceSize.W-2*fileDialogMargin),
		H: fitDialogSize(fileDialogH, fileDialogMinH, a.surfaceSize.H-2*fileDialogMargin),
	}
	root := &fileDialogPanel{Container: panel, size: size, d: d}

	panel.Add(NewLabel(d.cfg.title))

	// Path row: up and the path.
	d.up = NewButton("Up")
	d.up.OnClick(func() {
		d.goUp()
		d.table.RequestFocus() // so typing goes on jumping within the listing
	})
	d.pathField = NewTextField()
	d.pathField.OnSubmit(d.goToTyped)
	pathRow := NewContainer()
	pathRow.SetLayout(HBox(ThemeGap))
	pathRow.Add(d.up)
	// New folder, where making the destination makes sense and the filesystem can.
	if _, can := d.fsys.(DirMaker); can && d.mode != FileOpen {
		newFolder := NewButton("New folder")
		newFolder.OnClick(d.newFolder)
		pathRow.Add(newFolder)
	}
	pathRow.Add(d.pathField, Weight(1))
	panel.Add(pathRow)

	// Listing. Sorting is the dialog's (directories first, real sizes and
	// dates), so the table's own sort is off. Every column's Less is a constant
	// false: SortBy then only draws the indicator and leaves the order alone.
	keep := func(a, b string) bool { return false }
	cols := make([]Column, len(fileDialogColumns))
	for i, t := range fileDialogColumns {
		cols[i] = Column{Title: t, Weight: []int{3, 1, 2}[i], Less: keep}
	}
	d.table = NewTable(cols)
	d.table.SetSortable(false)
	d.table.OnHeaderClick(d.sortByColumn)
	if d.multi {
		d.table.SetMultiSelect(true)
		d.table.OnSelectionChange(d.selectionChanged)
	} else {
		d.table.OnSelect(d.selectRow)
	}
	d.table.OnActivate(d.activateRow)
	if places := d.places(); len(places) > 1 {
		panel.Add(HSplit(d.placesPane(places), d.table, SplitRatio(0.22), SplitMinSizes(90, 200)), Weight(1))
	} else {
		panel.Add(d.table, Weight(1))
	}

	d.status = NewLabel(" ")
	panel.Add(d.status)

	// Name row: file (or folder) name and, if there are filters, the filter
	// dropdown. A folder dialog lists no files, so it has no filters.
	label, hint := "File name:", "File name"
	if d.mode == FileSelectDir {
		label, hint = "Folder:", "Folder (blank for this one)"
	}
	d.nameField = NewTextField(Placeholder(hint))
	d.nameField.OnSubmit(func(string) { d.accept() })
	nameRow := NewContainer()
	nameRow.SetLayout(HBox(ThemeGap))
	nameRow.Add(NewLabel(label))
	nameRow.Add(d.nameField, Weight(1))
	if len(d.cfg.filters) > 0 && d.mode != FileSelectDir {
		names := make([]string, len(d.cfg.filters))
		for i, f := range d.cfg.filters {
			names[i] = f.Name
		}
		dd := NewDropdown(names, DropdownSelected(0))
		dd.OnSelect(d.selectFilter)
		nameRow.Add(dd)
	}
	panel.Add(nameRow)

	// Bottom row: the hidden-files toggle on the left, the buttons on the right.
	hidden := NewCheckbox("Show hidden", Checked(d.showHidden))
	hidden.OnChange(func(on bool) {
		d.showHidden = on
		d.refreshKeepingSelection()
	})
	cancel := NewButton(d.cfg.cancel)
	cancel.OnClick(func() { d.finish(nil, false) })
	ok := NewButton(d.cfg.ok)
	ok.OnClick(d.accept)
	bottom := NewContainer()
	bottom.SetLayout(HBox(ThemeGap))
	bottom.Add(hidden)
	bottom.Add(NewContainer(), Weight(1)) // pushes the buttons to the right
	bottom.Add(cancel)
	bottom.Add(ok)
	panel.Add(bottom)

	d.popup = a.ShowModal(root)
	// A button, Enter or a double-click resolves first (and the close then does
	// nothing); Escape and programmatic closes reach here unresolved and count
	// as Cancel.
	d.popup.onClose = func() {
		d.resolve(nil, false)
		a.setFocus(prevFocus)
	}

	d.openInitialDir()
	if d.mode == FileSave {
		// Saving starts in the name field, with a suggestion selected.
		if d.cfg.name != "" {
			d.nameField.SetText(d.cfg.name)
			d.nameField.selectAll()
		}
		d.nameField.RequestFocus()
	} else {
		d.table.RequestFocus()
	}
	return d.popup
}

// openInitialDir opens the start directory, falling back to home and then the
// first root if it cannot be read. If none can, the dialog stays on the last
// one tried with its error showing.
func (d *fileDialog) openInitialDir() {
	var tries []string
	if d.cfg.start != "" {
		tries = append(tries, d.fsys.Resolve("", d.cfg.start))
	}
	if h := d.fsys.Home(); h != "" {
		tries = append(tries, h)
	}
	tries = append(tries, d.fsys.Roots()[0].Path)
	var try func(i int)
	try = func(i int) {
		d.navigate(tries[i], func(ok bool) {
			if !ok && i+1 < len(tries) {
				try(i + 1)
			}
		})
	}
	try(0)
}

// resolve reports the result to the caller, once.
func (d *fileDialog) resolve(paths []string, ok bool) {
	if d.done {
		return
	}
	d.done = true
	if d.onResult != nil {
		d.onResult(paths, ok)
	}
}

// finish reports the result and closes the dialog.
func (d *fileDialog) finish(paths []string, ok bool) {
	d.resolve(paths, ok)
	d.app.closePopup(d.popup)
}

// setStatus shows a one-line message under the listing; isErr colours it as an
// error. An empty message keeps the line's height so the layout does not jump.
func (d *fileDialog) setStatus(msg string, isErr bool) {
	if msg == "" {
		msg = " "
	}
	pal := d.app.theme.Palette
	col := pal.TextMuted
	if isErr {
		col = orColour(pal.Error, theme.DefaultPalette().Error)
	}
	d.status.SetColour(RoleText, col)
	d.status.SetText(msg)
}

// loadingHintDelay is how long a directory read runs before the dialog says it
// is loading, so a quick read never flashes the message.
const loadingHintDelay = 150 * time.Millisecond

// load reads dir off the UI goroutine, so a slow disk or network share does not
// freeze the window, and calls then with the result on the UI goroutine. The
// dialog keeps showing (and answering to) the old listing meanwhile. A later load
// supersedes an earlier one, whose result is dropped, as is any result that
// arrives after the dialog has closed.
func (d *fileDialog) load(dir string, then func(ents []fs.FileInfo, err error)) {
	d.loadGen++
	gen := d.loadGen
	d.loadBusy = true
	hint := time.AfterFunc(loadingHintDelay, func() {
		d.app.Do(func() {
			if d.loadBusy && d.loadGen == gen && !d.done {
				d.setStatus("Loading "+dir+"...", false)
			}
		})
	})
	var ents []fs.FileInfo
	var err error
	d.app.Async(func() { ents, err = d.fsys.ReadDir(dir) }, func() {
		hint.Stop()
		if d.loadGen != gen || d.done {
			return
		}
		d.loadBusy = false
		then(ents, err)
	})
}

// navigate lists dir and makes it the current directory, then calls then (if not
// nil) with whether that worked. The read is asynchronous, so then is how a
// caller acts on the new listing. If dir cannot be read the dialog stays where it
// was and shows why.
func (d *fileDialog) navigate(dir string, then func(ok bool)) {
	d.load(dir, func(ents []fs.FileInfo, err error) {
		if err != nil {
			d.setStatus(err.Error(), true)
			d.pathField.SetText(d.dir)
			if then != nil {
				then(false)
			}
			return
		}
		d.dir = dir
		d.entries = ents
		d.pathField.SetText(dir)
		if d.mode != FileSave { // a name being typed for Save survives browsing
			d.nameField.SetText("")
			d.autoName = ""
		}
		d.setStatus("", false)
		_, hasParent := d.fsys.Parent(dir)
		d.up.SetEnabled(hasParent)
		d.updatePlaces()
		d.refresh()
		if then != nil {
			then(true)
		}
	})
}

// goUp moves to the parent directory and selects the one just left.
func (d *fileDialog) goUp() {
	parent, ok := d.fsys.Parent(d.dir)
	if !ok {
		return
	}
	from := d.dir
	d.navigate(parent, func(ok bool) {
		if !ok {
			return
		}
		for _, e := range d.shown {
			if e.IsDir() && d.fsys.Join(parent, e.Name()) == from {
				d.selectName(e.Name())
				return
			}
		}
	})
}

// goToTyped handles Enter in the path field: a directory is opened, a file
// opens its directory with the file selected.
func (d *fileDialog) goToTyped(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		d.pathField.SetText(d.dir)
		return
	}
	target := d.fsys.Resolve(d.dir, text)
	info, err := d.fsys.Stat(target)
	switch {
	case err != nil:
		d.setStatus(err.Error(), true)
	case info.IsDir():
		d.navigate(target, nil)
	default:
		if parent, ok := d.fsys.Parent(target); ok {
			d.navigate(parent, func(ok bool) {
				if ok {
					d.selectName(info.Name())
				}
			})
		}
	}
}

// accept handles OK and Enter in the name field, by mode.
func (d *fileDialog) accept() {
	if d.dir == "" {
		return // the first listing has not arrived, so there is no directory to resolve against
	}
	switch d.mode {
	case FileSave:
		d.acceptSave()
	case FileSelectDir:
		d.acceptDir()
	default:
		d.acceptOpen()
	}
}

// acceptSave handles OK for FileSave. A name that is a directory opens it.
// Otherwise the target (with the default extension) is returned, after asking
// whether to replace it if it exists.
func (d *fileDialog) acceptSave() {
	text := strings.TrimSpace(d.nameField.Text())
	if text == "" {
		d.setStatus("Enter a file name.", true)
		return
	}
	if info, err := d.fsys.Stat(d.fsys.Resolve(d.dir, text)); err == nil && info.IsDir() {
		d.navigate(d.fsys.Resolve(d.dir, text), func(ok bool) {
			if ok {
				d.nameField.SetText("")
			}
		})
		return
	}
	target, exists, err := saveTarget(d.fsys, d.dir, text, d.cfg.defaultExt)
	if err != nil {
		d.setStatus(err.Error(), true)
		return
	}
	if !exists || d.cfg.noOverwritePrompt {
		d.finish([]string{target}, true)
		return
	}
	name := withDefaultExt(text, d.cfg.defaultExt)
	d.app.ShowConfirm("Replace file?", fmt.Sprintf("%q already exists.\nDo you want to replace it?", name),
		func(ok bool) {
			if ok {
				d.finish([]string{target}, true)
			}
		},
		ConfirmLabels("Replace", "Cancel"), ConfirmDefaultCancel())
}

// acceptDir handles OK for FileSelectDir: the typed or selected folder, or the
// one being browsed if the name field is empty.
func (d *fileDialog) acceptDir() {
	text := strings.TrimSpace(d.nameField.Text())
	if text == "" {
		d.finish([]string{d.dir}, true)
		return
	}
	target := d.fsys.Resolve(d.dir, text)
	info, err := d.fsys.Stat(target)
	switch {
	case err != nil && isNotExist(err):
		d.setStatus(fmt.Sprintf("%q does not exist.", text), true)
	case err != nil:
		d.setStatus(err.Error(), true)
	case !info.IsDir():
		d.setStatus(fmt.Sprintf("%q is not a folder.", text), true)
	default:
		d.finish([]string{target}, true)
	}
}

// acceptOpen handles OK for FileOpen. A file is returned; a directory is opened;
// with nothing typed, a selected directory is opened.
func (d *fileDialog) acceptOpen() {
	text := strings.TrimSpace(d.nameField.Text())
	// With several files selected the name field holds the dialog's own summary of
	// them. If the user has not edited it, the selection is the answer.
	if d.multi && text != "" && text == d.autoName {
		if paths := d.selectedFilePaths(); len(paths) > 0 {
			d.finish(paths, true)
			return
		}
	}
	if text == "" {
		if e := d.selectedEntry(); e != nil && e.IsDir() {
			d.navigate(d.fsys.Join(d.dir, e.Name()), nil)
			return
		}
		d.setStatus("Choose a file.", true)
		return
	}
	target := d.fsys.Resolve(d.dir, text)
	info, err := d.fsys.Stat(target)
	switch {
	case err != nil && isNotExist(err):
		d.setStatus(fmt.Sprintf("%q does not exist.", text), true)
	case err != nil:
		d.setStatus(err.Error(), true)
	case info.IsDir():
		d.navigate(target, nil)
	default:
		d.finish([]string{target}, true)
	}
}

// selectedEntry returns the entry behind the selected row, or nil.
func (d *fileDialog) selectedEntry() fs.FileInfo {
	if i := d.table.Selected(); i >= 0 && i < len(d.shown) {
		return d.shown[i]
	}
	return nil
}

// selectRow runs when a row is selected, and puts its name in the name field:
//   - Open: a file's name; a directory clears it, so OK then opens it.
//   - Save: a file's name; a directory leaves the name being typed alone.
//   - Select folder: the directory's name.
func (d *fileDialog) selectRow(i int) {
	if i < 0 || i >= len(d.shown) || d.reselecting {
		return
	}
	d.setStatus("", false)
	e := d.shown[i]
	switch {
	case d.mode == FileSelectDir:
		d.nameField.SetText(e.Name())
	case e.IsDir() && d.mode == FileOpen:
		d.nameField.SetText("")
	case !e.IsDir():
		d.nameField.SetText(e.Name())
	}
}

// activateRow runs on double-click or Enter in the listing. A directory is
// opened. A file is chosen: Open returns it, Save goes through the same checks
// as OK (so an existing file still asks before it is replaced).
func (d *fileDialog) activateRow(i int) {
	if i < 0 || i >= len(d.shown) {
		return
	}
	e := d.shown[i]
	target := d.fsys.Join(d.dir, e.Name())
	switch {
	case e.IsDir():
		d.navigate(target, nil)
	case d.mode == FileSave:
		d.nameField.SetText(e.Name())
		d.acceptSave()
	case d.multi:
		// Enter on a multiple selection takes all of it; a double-click has already
		// collapsed the selection to the row clicked.
		if paths := d.selectedFilePaths(); len(paths) > 0 && d.table.isSelected(i) {
			d.finish(paths, true)
			return
		}
		d.finish([]string{target}, true)
	default:
		d.finish([]string{target}, true)
	}
}

// selectedFilePaths returns the paths of the selected files, in listing order.
// Directories in the selection are skipped.
func (d *fileDialog) selectedFilePaths() []string {
	var paths []string
	for _, i := range d.table.SelectedRows() {
		if i < len(d.shown) && !d.shown[i].IsDir() {
			paths = append(paths, d.fsys.Join(d.dir, d.shown[i].Name()))
		}
	}
	return paths
}

// selectionChanged is the multiple-selection counterpart of selectRow: the name
// field shows the selected files, one name as it is, several quoted and
// space-separated. Directories do not appear in it.
func (d *fileDialog) selectionChanged(rows []int) {
	if d.reselecting {
		return
	}
	d.setStatus("", false)
	var names []string
	for _, i := range rows {
		if i < len(d.shown) && !d.shown[i].IsDir() {
			names = append(names, d.shown[i].Name())
		}
	}
	text := ""
	switch len(names) {
	case 0:
	case 1:
		text = names[0]
	default:
		for i, n := range names {
			if i > 0 {
				text += " "
			}
			text += `"` + n + `"`
		}
	}
	d.autoName = text
	d.nameField.SetText(text)
}

// sortByColumn runs when a header is clicked: a new column sorts ascending, the
// same column flips direction.
func (d *fileDialog) sortByColumn(col int) {
	key := fileSortKey(col)
	if key == d.sortKey {
		d.sortAsc = !d.sortAsc
	} else {
		d.sortKey, d.sortAsc = key, true
	}
	d.refreshKeepingSelection()
}

// selectFilter runs when the filter dropdown changes.
func (d *fileDialog) selectFilter(i int) {
	d.filter = i
	d.refreshKeepingSelection()
}

func (d *fileDialog) refreshKeepingSelection() {
	var names []string
	for _, i := range d.table.SelectedRows() {
		if i < len(d.shown) {
			names = append(names, d.shown[i].Name())
		}
	}
	active := ""
	if e := d.selectedEntry(); e != nil {
		active = e.Name()
	}
	// With several files selected the name field is the dialog's own summary of
	// them, which must follow the selection if a filter drops some of the files.
	summary := d.multi && d.nameField.Text() == d.autoName
	d.refresh()
	if len(names) == 0 {
		return
	}
	// Putting the selection back must not rewrite a name the user typed since
	// selecting the row.
	d.reselecting = true
	defer func() { d.reselecting = false }()
	var rows []int
	for _, n := range names {
		for i, e := range d.shown {
			if e.Name() == n {
				rows = append(rows, i)
			}
		}
	}
	if len(rows) == 0 {
		return
	}
	// Make the row that was active last, so it stays the cursor row.
	for k, i := range rows {
		if d.shown[i].Name() == active {
			rows[k], rows[len(rows)-1] = rows[len(rows)-1], rows[k]
			break
		}
	}
	d.table.SetSelectedRows(rows...)
	d.table.scrollTo(rows[len(rows)-1])
	if summary {
		d.reselecting = false
		d.selectionChanged(d.table.SelectedRows())
	}
}

// currentFilter is the filter chosen in the dropdown, or the match-all filter.
// A folder dialog has none.
func (d *fileDialog) currentFilter() FileFilter {
	if d.mode != FileSelectDir && d.filter >= 0 && d.filter < len(d.cfg.filters) {
		return d.cfg.filters[d.filter]
	}
	return FileFilter{}
}

// refresh rebuilds the listing from entries with the current filter and sort.
func (d *fileDialog) refresh() {
	d.shown = visibleEntries(d.entries, d.currentFilter(), d.showHidden, d.mode == FileSelectDir)
	sortEntries(d.shown, d.sortKey, d.sortAsc)
	rows := make([][]string, len(d.shown))
	for i, e := range d.shown {
		name, size := e.Name(), humanSize(e.Size())
		if e.IsDir() {
			name, size = name+"/", ""
		}
		rows[i] = []string{name, size, formatModTime(e.ModTime())}
	}
	d.table.SetRows(rows)
	d.table.SortBy(int(d.sortKey), d.sortAsc)
}

// selectName selects the row for the entry called name, if it is listed, and
// scrolls it into view.
func (d *fileDialog) selectName(name string) {
	for i, e := range d.shown {
		if e.Name() == name {
			d.table.SetSelected(i)
			d.table.scrollTo(i)
			return
		}
	}
}

// newFolder asks for a name and creates a folder in the directory being browsed.
// The prompt is a small modal over the dialog; its error line shows why a name
// was refused, and it stays open until the name works or it is cancelled.
func (d *fileDialog) newFolder() {
	mk, ok := d.fsys.(DirMaker)
	if !ok {
		return
	}
	a := d.app
	pal := a.theme.Palette
	prevFocus := a.focused

	panel := NewContainer()
	panel.SetBackground(pal.Surface)
	panel.SetBorder(pal.Border, a.theme.BorderWidth)
	panel.SetCornerRadius(a.theme.CornerRadius)
	panel.SetLayout(VBox(12))
	panel.SetPadding(geom.UniformInsets(16))

	field := NewTextField()
	status := NewLabel(" ")
	fail := func(msg string) {
		status.SetColour(RoleText, orColour(pal.Error, theme.DefaultPalette().Error))
		status.SetText(msg)
	}
	var popup *Popup
	create := func() {
		name, err := validFolderName(field.Text())
		if err != nil {
			fail(err.Error())
			return
		}
		if err := mk.Mkdir(d.fsys.Join(d.dir, name)); err != nil {
			if errors.Is(err, fs.ErrExist) {
				fail(fmt.Sprintf("%q already exists.", name))
			} else {
				fail(err.Error())
			}
			return
		}
		a.closePopup(popup)
		d.reloadAndSelect(name)
		d.table.RequestFocus()
	}
	field.OnSubmit(func(string) { create() })

	cancel := NewButton("Cancel")
	cancel.OnClick(func() { a.closePopup(popup) })
	ok2 := NewButton("Create")
	ok2.OnClick(create)
	row := NewContainer()
	row.SetLayout(HBox(10))
	row.Add(cancel)
	row.Add(ok2)

	panel.Add(NewLabel("New folder name:"))
	panel.Add(field)
	panel.Add(status)
	panel.Add(row, Align(geom.AlignEnd))

	popup = a.ShowModal(panel)
	popup.onClose = func() { a.setFocus(prevFocus) }
	field.SetText(uniqueName("New folder", func(n string) bool {
		for _, e := range d.entries {
			if e.Name() == n {
				return true
			}
		}
		return false
	}))
	field.selectAll()
	field.RequestFocus()
}

// reloadAndSelect re-reads the current directory, which something outside the
// dialog has changed, and selects name in it.
func (d *fileDialog) reloadAndSelect(name string) {
	dir := d.dir
	d.load(dir, func(ents []fs.FileInfo, err error) {
		if err != nil {
			d.setStatus(err.Error(), true)
			return
		}
		d.entries = ents
		d.refresh()
		d.selectName(name)
		d.setStatus(fmt.Sprintf("Created folder %q.", name), false)
	})
}

// fileDialogPlace is one shortcut in the places pane.
type fileDialogPlace struct{ name, path string }

// places lists the shortcuts: the home directory and those of its usual folders
// that exist, then every root (the drives on Windows).
func (d *fileDialog) places() []fileDialogPlace {
	var out []fileDialogPlace
	if h := d.fsys.Home(); h != "" {
		out = append(out, fileDialogPlace{"Home", h})
		for _, name := range []string{"Desktop", "Documents", "Downloads"} {
			p := d.fsys.Join(h, name)
			if info, err := d.fsys.Stat(p); err == nil && info.IsDir() {
				out = append(out, fileDialogPlace{name, p})
			}
		}
	}
	for _, r := range d.fsys.Roots() {
		out = append(out, fileDialogPlace{r.Name, r.Path})
	}
	return out
}

// placeCell is one entry in the places pane: a flat button in a container whose
// background shows when this is the current place.
type placeCell struct {
	place fileDialogPlace
	cell  *Container
	btn   *Button
}

// placesPane builds the left-hand pane: one flat button per place, in a
// scroll view in case there are more than fit. Buttons rather than a List,
// because a List cannot clear its selection and would keep showing the last
// place picked after the user had browsed elsewhere. The current place is
// marked by updatePlaces instead, from where the dialog actually is.
func (d *fileDialog) placesPane(places []fileDialogPlace) Widget {
	col := NewContainer()
	col.SetLayout(VBox(2))
	d.placeCells = d.placeCells[:0]
	for _, p := range places {
		p := p
		b := NewButton(p.name, ButtonFlat())
		b.OnClick(func() {
			d.navigate(p.path, nil)
			d.table.RequestFocus()
		})
		cell := NewContainer()
		cell.SetLayout(VBox(0))
		cell.SetCornerRadius(d.app.theme.CornerRadius)
		cell.Add(b)
		col.Add(cell)
		d.placeCells = append(d.placeCells, placeCell{place: p, cell: cell, btn: b})
	}
	sv := NewScrollView()
	sv.SetContent(col)
	return sv
}

// updatePlaces marks the place the dialog is in, and clears the mark from the
// rest. It runs after every navigation, so the pane follows the user however
// they got there: a click, a double-click, Up, the path bar or New folder.
func (d *fileDialog) updatePlaces() {
	if len(d.placeCells) == 0 {
		return
	}
	places := make([]fileDialogPlace, len(d.placeCells))
	for i, pc := range d.placeCells {
		places[i] = pc.place
	}
	current := currentPlaceIndex(d.fsys, places, d.dir)
	for i, pc := range d.placeCells {
		if i == current {
			pc.cell.SetBackground(pc.btn.ColourOf(RolePrimary))
			pc.btn.SetColour(RoleText, pc.btn.ColourOf(RoleOnPrimary))
		} else {
			pc.cell.SetBackground(nil)
			pc.btn.SetColour(RoleText, nil)
		}
	}
}

// typeAheadRune handles a character typed in the listing: it selects the next
// entry whose name starts with the letters typed so far, and reports whether it
// took the character.
func (d *fileDialog) typeAheadRune(r rune) bool {
	if !unicode.IsPrint(r) || len(d.shown) == 0 {
		return false
	}
	now := time.Now()
	// A space is not the start of a name worth jumping to, and in a multiple
	// selection it is the key that toggles the row, so there it never belongs to
	// the prefix. Otherwise it can continue one ("holiday photos").
	if r == ' ' && (d.multi || !d.typed.pending(now)) {
		return true
	}
	prefix, fresh := d.typed.add(r, now)
	start := d.table.Selected()
	switch {
	case start < 0:
		start = 0
	case fresh:
		start++ // a new first letter looks beyond the current entry
	}
	// The same letter repeated cycles through the entries that start with it.
	if rs := []rune(prefix); len(rs) > 1 && allSameRune(rs) {
		prefix = string(rs[0])
		start = d.table.Selected() + 1
	}
	names := make([]string, len(d.shown))
	for i, e := range d.shown {
		names[i] = e.Name()
	}
	if i := typeAheadMatch(names, start, prefix); i >= 0 {
		d.table.SetSelected(i)
		d.table.scrollTo(i)
	}
	return true
}

// fitDialogSize returns the preferred size, shrunk to fit avail but never below
// min: a surface too small for min gets min (and the panel overflows) rather
// than an unusably tiny dialog.
func fitDialogSize(pref, min, avail float64) float64 {
	if pref > avail {
		pref = avail
	}
	if pref < min {
		pref = min
	}
	return pref
}
