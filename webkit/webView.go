package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #cgo pkg-config: gtk+-3.0
// #include <webkit2/webkit2.h>
// #include <gtk/gtk.h>
// #include <stdlib.h>
/*
static GtkWidget* toGtkWidget(void* p) {
	return (GTK_WIDGET(p));
}
*/
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
)

// WebView represents a WebKitWebView widget.
type WebView struct {
	gtk.Container
	// The settings of the WebView, may be nil if they were never set or
	// retrieved.
	settings *Settings
}

// NewWebView creates and returns a new webkit webview.
func NewWebView() (*WebView, error) {
	w := C.webkit_web_view_new()
	if w == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(w))}
	webView := wrapWebView(obj)
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return webView, nil
}

// NewWebViewWithUserContentManager creates a new WebView, using a specific
// UserContentManager.
func NewWebViewWithUserContentManager(ucm *UserContentManager) (*WebView, error) {
	w := C.webkit_web_view_new_with_user_content_manager(
		(*C.WebKitUserContentManager)(unsafe.Pointer(ucm.Native())))
	if w == nil {
		return nil, errNilPtr
	}
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(w))}
	webView := wrapWebView(obj)
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	return webView, nil
}

// wrapWebView wraps a creates web view object in the appropriate classes.
func wrapWebView(obj *glib.Object) *WebView {
	return &WebView{gtk.Container{gtk.Widget{glib.InitiallyUnowned{obj}}}, nil}
}

// native retrieves (a properly casted) pointer the native C WebKitWebView.
func (w *WebView) native() *C.WebKitWebView {
	return (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
}

// GetPageID gets the ID of the web page corresponding to the web view.
func (w *WebView) GetPageID() uint64 {
	return uint64(C.webkit_web_view_get_page_id(w.native()))
}

// LoadURI requests loading of the speicified URI string.
func (w *WebView) LoadURI(uri string) {
	cURI := (*C.gchar)(C.CString(uri))
	defer C.free(unsafe.Pointer(cURI))
	C.webkit_web_view_load_uri(w.native(), cURI)
}

// LoadRequest loads a specified URI request.
func (w *WebView) LoadRequest(req *UriRequest) {
	C.webkit_web_view_load_request(
		w.native(),
		(*C.WebKitURIRequest)(unsafe.Pointer(req.Native())))
}

// IsLoading checks if a WebView is currently loading.
func (w *WebView) IsLoading() bool {
	return gobool(C.webkit_web_view_is_loading(w.native()))
}

// Reload request the WebView to reload.
func (w *WebView) Reload() {
	C.webkit_web_view_reload(w.native())
}

// GetEstimatedLoadProgress gets an estimation for the progress of a load
// operation.
func (w *WebView) GetEstimatedLoadProgress() float64 {
	return float64(C.webkit_web_view_get_estimated_load_progress(w.native()))
}

// GetTitle gets the webviews current title.
func (w *WebView) GetTitle() string {
	cstr := C.webkit_web_view_get_title(w.native())
	return C.GoString((*C.char)(cstr))
}

// GetURI gets the currently displayed URI.
func (w *WebView) GetURI() string {
	cstr := C.webkit_web_view_get_uri(w.native())
	return C.GoString((*C.char)(cstr))
}

// CanGoBack checks whether it is possible to currently go back.
func (w *WebView) CanGoBack() bool {
	return gobool(C.webkit_web_view_can_go_back(w.native()))
}

// GoBack goes back one step in browser history.
func (w *WebView) GoBack() {
	C.webkit_web_view_go_back(w.native())
}

// CanGoForward checks whether it is possible to currently go forward.
func (w *WebView) CanGoForward() bool {
	return gobool(C.webkit_web_view_can_go_forward(w.native()))
}

// GoForward goes forward one step in browser history.
func (w *WebView) GoForward() {
	C.webkit_web_view_go_forward(w.native())
}

// GetBackForwardList gets the views list of back/forward steps in history.
//
// Note that this call is fairly expensive and takes several conversions.
// Keep a reference if you use it more often.
func (w *WebView) GetBackForwardList() *BackForwardList {
	bfl := C.webkit_web_view_get_back_forward_list(w.native())
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(bfl))}
	return &BackForwardList{obj}
}

// SetSettings sets the settings used for this WebView.
func (w *WebView) SetSettings(s *Settings) {
	w.settings = s
	C.webkit_web_view_set_settings(
		w.native(),
		(*C.WebKitSettings)(unsafe.Pointer(s.Native())))
}

// GetSettings retrieves the settings used for this WebView.
func (w *WebView) GetSettings() *Settings {
	if w.settings == nil {
		w.settings = wrapSettings(C.webkit_web_view_get_settings(w.native()))
	}
	return w.settings
}
