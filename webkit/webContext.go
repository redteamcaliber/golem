package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <stdlib.h>
// #include <webkit2/webkit2.h>
/*

extern void
cgoURISchemeRequestCallback(WebKitURISchemeRequest *req, gpointer f);

static inline void
go_webkit_web_context_register_uri_scheme(
		WebKitWebContext *c,
		gchar *scheme,
		gpointer callback) {
	webkit_web_context_register_uri_scheme(
		c,
		scheme,
		cgoURISchemeRequestCallback,
		callback,
		NULL);
}

*/
import "C"
import (
	"runtime"
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

const (
	// ProcessModelSharedSecondaryProcess specifies that the web process
	// should be shared between all WebViews.
	ProcessModelSharedSecondaryProcess = C.WEBKIT_PROCESS_MODEL_SHARED_SECONDARY_PROCESS
	// ProcessModelMultipleSecondaryProcesses specifies that (most) WebViews
	// should run their own web process.
	ProcessModelMultipleSecondaryProcesses = C.WEBKIT_PROCESS_MODEL_MULTIPLE_SECONDARY_PROCESSES
)

// The defaultWebContext is the WebContext which is used by default for new
// WebViews.
//
// May be nil until the default web context is retrieved.
var defaultWebContext *WebContext

// A WebContext is a wrapper around WebKitWebContext.
//
// It manages aspects common to all WebViews.
type WebContext struct {
	*glib.Object
	uriSchemes map[*func(*URISchemeRequest)]bool
}

// GetDefaultWebContext gets the default web context (i.e. the WebContext used
// by all WebViews by default).
func GetDefaultWebContext() *WebContext {
	if defaultWebContext == nil {
		wc := C.webkit_web_context_get_default()
		if wc == nil {
			panic("Failed to retrieve default web context.")
		}
		obj := &glib.Object{glib.ToGObject(unsafe.Pointer(wc))}
		obj.RefSink()
		runtime.SetFinalizer(obj, (*glib.Object).Unref)
		defaultWebContext = &WebContext{
			obj,
			make(map[*func(*URISchemeRequest)]bool, 5),
		}
	}
	return defaultWebContext
}

// SetWebExtensionsDirectory sets the directory in which web extensions can be
// found.
func (c *WebContext) SetWebExtensionsDirectory(to string) {
	cstr := C.CString(to)
	defer C.free(unsafe.Pointer(cstr))
	C.webkit_web_context_set_web_extensions_directory(
		(*C.WebKitWebContext)(unsafe.Pointer(c.Native())),
		(*C.gchar)(cstr))
}

// SetProcessModel sets the model used for the distribution of web processes
// for WebViews.
//
// Should be one of ProcessModelSharedSecondaryProcess and
// ProcessModelMultipleSecondaryProcesses.
func (c *WebContext) SetProcessModel(to C.WebKitProcessModel) {
	C.webkit_web_context_set_process_model(
		(*C.WebKitWebContext)(unsafe.Pointer(c.Native())),
		to)
}

// RegisterURIScheme registers a custom URI scheme.
func (c *WebContext) RegisterURIScheme(scheme string,
	callback func(*URISchemeRequest)) {

	// Prevents callback from being garbage collected until the WebContext
	// is destroyed.
	c.uriSchemes[&callback] = true
	cstr := C.CString(scheme)
	defer C.free(unsafe.Pointer(cstr))
	C.go_webkit_web_context_register_uri_scheme(
		(*C.WebKitWebContext)(unsafe.Pointer(c.Native())),
		(*C.gchar)(cstr),
		C.gpointer(unsafe.Pointer(&callback)))
}

//export cgoURISchemeRequestCallback
func cgoURISchemeRequestCallback(req *C.WebKitURISchemeRequest, f C.gpointer) {
	goFunc := (*func(req *URISchemeRequest))(unsafe.Pointer(f))
	obj := &glib.Object{glib.ToGObject(unsafe.Pointer(req))}
	obj.RefSink()
	runtime.SetFinalizer(obj, (*glib.Object).Unref)
	goReq := &URISchemeRequest{obj}
	(*goFunc)(goReq)
}