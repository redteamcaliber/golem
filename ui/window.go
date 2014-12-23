// Package ui contains golem's user-interface implementation.
package ui

import (
	"fmt"

	"github.com/conformal/gotk3/gtk"
	"github.com/tkerber/golem/cmd"
	"github.com/tkerber/golem/webkit"
)

// A Window is one of golem's windows.
type Window struct {
	StatusBar
	*webkit.WebView
	*gtk.Window
	// How far from the top the active web view is scrolled.
	Top int64
	// The height of the active web view.
	Height int64
}

// NewWindow creates a new window containing the given WebView.
func NewWindow(webView *webkit.WebView) (*Window, error) {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	win.SetTitle("Golem")

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	cmdStatus.OverrideFont("monospace")

	locationStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	locationStatus.OverrideFont("monospace")

	// TODO: long urls/commands cause these to overlap. fix.
	statusBar.PackStart(cmdStatus, false, false, 0)
	statusBar.PackEnd(locationStatus, false, false, 0)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	box.PackStart(webView, true, true, 0)
	box.PackStart(statusBar, false, false, 0)
	win.Add(box)

	// TODO sensible default size. (Default to screen size?)
	win.SetDefaultSize(800, 600)

	w := &Window{
		StatusBar{cmdStatus, locationStatus, statusBar.Container},
		webView,
		win,
		0,
		0,
	}

	return w, nil
}

// Show shows the window.
func (w *Window) Show() {
	w.Window.ShowAll()
}

// HideUI hides all UI (non-webkit) elements.
func (w *Window) HideUI() {
	w.StatusBar.container.Hide()
}

// ShowUI shows all UI elements.
func (w *Window) ShowUI() {
	w.StatusBar.container.Show()
}

// UpdateState updates the (command) state display of the window.
func (w *Window) UpdateState(state cmd.State) {
	var newStatus string
	switch s := state.(type) {
	case *cmd.NormalMode:
		// The status is either empty, or [current_binding] if it exists.
		if len(s.CurrentKeys) == 0 {
			newStatus = ""
		} else {
			newStatus = fmt.Sprintf("[%v]", cmd.KeysString(s.CurrentKeys))
		}
	case *cmd.InsertMode:
		newStatus = "-- INSERT --"
	case *cmd.CommandLineMode:
		newStatus = fmt.Sprintf(
			":%v",
			cmd.KeysStringSelective(s.CurrentKeys, false))
	}
	w.SetCmdLabel(newStatus)
}

// UpdateLocation updates the location display of the window.
func (w *Window) UpdateLocation() {
	locStr := w.GetURI()
	locStr += " "

	backForward := ""
	if w.CanGoBack() {
		backForward += "-"
	}
	if w.CanGoForward() {
		backForward += "+"
	}
	if backForward != "" {
		locStr += "[" + backForward + "]"
	}

	var pos string
	visible := int64(w.WebView.GetAllocatedHeight())
	if int64(visible) >= w.Height {
		pos = "ALL"
	} else if w.Top == 0 {
		pos = "TOP"
	} else if w.Top == w.Height-visible {
		pos = "BOT"
	} else {
		percent := w.Top * 100 / (w.Height - visible)
		pos = fmt.Sprintf("%02d%%", percent)
	}
	locStr += "[" + pos + "]"

	w.SetLocationLabel(locStr)
}