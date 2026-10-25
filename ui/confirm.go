package ui

// confirmConfig holds the options for ShowConfirm.
type confirmConfig struct {
	ok, cancel    string
	focusesCancel bool
}

// ConfirmOption configures ShowConfirm.
type ConfirmOption func(*confirmConfig)

// ConfirmLabels sets the button labels (default "OK" and "Cancel"), for example
// "Delete" and "Keep".
func ConfirmLabels(ok, cancel string) ConfirmOption {
	return func(c *confirmConfig) { c.ok, c.cancel = ok, cancel }
}

// ConfirmDefaultCancel gives the Cancel button initial keyboard focus instead of
// the confirm button, so a stray Enter or Space cancels. Use it for destructive
// actions.
func ConfirmDefaultCancel() ConfirmOption {
	return func(c *confirmConfig) { c.focusesCancel = true }
}

// ShowConfirm shows a modal yes/no dialog and reports the choice to onResult
// exactly once: true for the confirm button, false for Cancel, Escape, or the
// dialog being closed any other way (App.Close, App.closeAll). The confirm
// button is on the right and has keyboard focus, so Enter confirms; keyboard
// focus returns to whatever held it before the dialog opened. onResult may be
// nil.
//
// It is ShowMessage with the bookkeeping done once: apps that need more than two
// choices, or no focus handling, use ShowMessage with their own DialogButtons.
func (a *App) ShowConfirm(title, message string, onResult func(ok bool), opts ...ConfirmOption) *Popup {
	cfg := confirmConfig{ok: "OK", cancel: "Cancel"}
	for _, o := range opts {
		o(&cfg)
	}
	prevFocus := a.focused
	done := false
	resolve := func(ok bool) {
		if done {
			return
		}
		done = true
		if onResult != nil {
			onResult(ok)
		}
	}

	p := a.ShowMessage(title, message,
		DialogButton{Label: cfg.cancel, OnClick: func() { resolve(false) }},
		DialogButton{Label: cfg.ok, OnClick: func() { resolve(true) }},
	)
	// A button click resolves first (and the close then does nothing); Escape and
	// programmatic closes reach here unresolved and count as Cancel.
	p.onClose = func() {
		resolve(false)
		a.setFocus(prevFocus)
	}

	row := p.content.(*Container).Children()[2].(*Container)
	pick := 1
	if cfg.focusesCancel {
		pick = 0
	}
	if f, ok := row.Children()[pick].(interface{ RequestFocus() }); ok {
		f.RequestFocus()
	}
	return p
}
