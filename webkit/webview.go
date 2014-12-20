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
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"go/build"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
)

// WebView represents a webkit webview widget.
type WebView struct {
	gtk.Container
}

func init() {
	// TODO figure out a better way to reference this. (i.e. without the source)
	extenPath := ""
	for _, src := range build.Default.SrcDirs() {
		p := filepath.Join(src, "github.com", "tkerber", "golem", "web_extension")
		if _, err := os.Stat(p); err == nil {
			extenPath = p
			break
		}
	}
	if extenPath == "" {
		panic("Failed to find source files!")
	}

	DefaultWebContext.SetWebExtensionsDirectory(extenPath)
	// TODO this is temporary.
	DefaultWebContext.RegisterURIScheme("golem", &golemSchemeHandler)
}

var golemSchemeHandler = func(req *URISchemeRequest) {
	req.Finish([]byte("<html><head><title>Golem</title></head><body><h1>Golem Home Page</h1><p>And stuff.</p></body></html>"), "text/html")
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

func wrapWebView(obj *glib.Object) *WebView {
	return &WebView{gtk.Container{gtk.Widget{glib.InitiallyUnowned{obj}}}}
}

func (w *WebView) native() *C.WebKitWebView {
	return (*C.WebKitWebView)(unsafe.Pointer(w.Native()))
}

// LoadURI requests loading of the speicified URI string.
func (w *WebView) LoadURI(uri string) {
	cURI := (*C.gchar)(C.CString(uri))
	defer C.free(unsafe.Pointer(cURI))
	C.webkit_web_view_load_uri(w.native(), cURI)
}

// IsLoading checks if a WebView is currently loading.
func (w *WebView) IsLoading() bool {
	if C.webkit_web_view_is_loading(w.native()) != 0 {
		return true
	}
	return false
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

func (w *WebView) GetURI() string {
	cstr := C.webkit_web_view_get_uri(w.native())
	return C.GoString((*C.char)(cstr))
}

func (w *WebView) CanGoBack() bool {
	return C.webkit_web_view_can_go_back(w.native()) != 0
}

// GoBack goes back one step in browser history.
func (w *WebView) GoBack() {
	C.webkit_web_view_go_back(w.native())
}

func (w *WebView) CanGoForward() bool {
	return C.webkit_web_view_can_go_forward(w.native()) != 0
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
