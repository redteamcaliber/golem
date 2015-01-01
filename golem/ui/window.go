// Package ui contains golem's user-interface implementation.
package ui

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"

	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/pango"
)

// A Window is one of golem's windows.
type Window struct {
	*StatusBar
	*TabBar
	WebView
	*gtk.Window
	*ColorScheme
	webViewBox *gtk.Box
	// The number of the active tab.
	TabNumber int
	// The number of total tabs in this window.
	TabCount int
}

// NewWindow creates a new window containing the given WebView.
func NewWindow(webView WebView) (*Window, error) {
	colors := NewColorScheme(
		0xffffff,
		0x888888,
		0xff8888,
		0xaaffaa,
		0xffaa88,
		0xff8888,
		0x66aaaa,
		0xdddddd,
		0x225588,
		0xdd9955,
		0x333333,
		0x222222,
	)

	w := &Window{
		nil,
		nil,
		webView,
		nil,
		colors,
		nil,
		1,
		1,
	}

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	win.SetTitle("Golem")
	w.Window = win

	sp := C.gtk_css_provider_new()
	css := colors.CSS
	gErr := new(*C.GError)
	cCss := C.CString(css)
	defer C.free(unsafe.Pointer(cCss))
	C.gtk_css_provider_load_from_data(
		sp,
		(*C.gchar)(cCss),
		-1,
		gErr)
	if *gErr != nil {
		goStr := C.GoString((*C.char)((**gErr).message))
		C.g_error_free(*gErr)
		return nil, errors.New(goStr)
	}
	screen, err := win.GetScreen()
	if err != nil {
		return nil, err
	}
	C.gtk_style_context_add_provider_for_screen(
		(*C.GdkScreen)(unsafe.Pointer(screen.Native())),
		(*C.GtkStyleProvider)(unsafe.Pointer(sp)),
		C.GTK_STYLE_PROVIDER_PRIORITY_APPLICATION)

	statusBar, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	if err != nil {
		return nil, err
	}
	statusBar.SetName("statusbar")

	cmdStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	cmdStatus.OverrideFont("monospace")
	cmdStatus.SetUseMarkup(true)
	cmdStatus.SetEllipsize(pango.ELLIPSIZE_START)

	locationStatus, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	locationStatus.OverrideFont("monospace")
	locationStatus.SetUseMarkup(true)
	locationStatus.SetEllipsize(pango.ELLIPSIZE_START)

	statusBar.PackStart(cmdStatus, false, false, 0)
	statusBar.PackEnd(locationStatus, false, false, 0)
	w.StatusBar = &StatusBar{cmdStatus, locationStatus, statusBar.Container}

	tabBar, err := NewTabBar(w)
	if err != nil {
		return nil, err
	}
	w.TabBar = tabBar

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	webViewBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}
	w.webViewBox = webViewBox
	webViewBox.PackStart(webView.GetWebView(), true, true, 0)

	contentBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	contentBox.PackStart(tabBar, false, false, 0)
	contentBox.PackStart(webViewBox, true, true, 0)

	box.PackStart(contentBox, true, true, 0)
	box.PackStart(statusBar, false, false, 0)
	win.Add(box)

	// TODO sensible default size. (Default to screen size?)
	win.SetDefaultSize(800, 600)

	return w, nil
}

// Show shows the window.
func (w *Window) Show() {
	w.Window.ShowAll()
}

// HideUI hides all UI (non-webkit) elements.
func (w *Window) HideUI() {
	w.StatusBar.container.Hide()
	w.TabBar.Box.Hide()
}

// ShowUI shows all UI elements.
func (w *Window) ShowUI() {
	w.StatusBar.container.Show()
	w.TabBar.Box.Show()
}

// SetTitle wraps gtk.Window.SetTitle in glib's main context.
func (w *Window) SetTitle(title string) {
	GlibMainContextInvoke(w.Window.SetTitle, title)
}

// ReplaceWebView replaces the web view being shown by the UI.
//
// This replacing occurs in the glib main context.
func (w *Window) ReplaceWebView(wv WebView) {
	GlibMainContextInvoke(w.replaceWebView, wv)
}

// replaceWebView replaces the web view being shown by the UI.
//
// MUST ONLY BE INVOKED THROUGH GlibMainContextInvoke!
func (w *Window) replaceWebView(wv WebView) {
	wvWidget := wv.GetWebView()
	w.GetWebView().Hide()
	if p, _ := wvWidget.GetParent(); p == nil {
		w.webViewBox.PackStart(wvWidget, true, true, 0)
	}
	wvWidget.Show()
	w.WebView = wv
	wvWidget.QueueDraw()
	wvWidget.GrabFocus()
}