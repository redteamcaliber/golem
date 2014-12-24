package webkit

// #cgo pkg-config: webkit2gtk-4.0
// #include <stdlib.h>
// #include <webkit2/webkit2.h>
import "C"
import (
	"unsafe"

	"github.com/conformal/gotk3/glib"
)

// A URISchemeRequest is a wrapper around a WebKitURISchemeRequest.
//
// This is an object passes to registered scheme handlers when a request to
// that scheme is made. The scheme handler is responsible for completing the
// request.
type URISchemeRequest struct {
	*glib.Object
}

// GetScheme retrieves the scheme of the request.
func (r *URISchemeRequest) GetScheme() string {
	cstr := C.webkit_uri_scheme_request_get_scheme(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.Native())))
	return C.GoString((*C.char)(cstr))
}

// GetURI retrieves the URI of the request.
func (r *URISchemeRequest) GetURI() string {
	cstr := C.webkit_uri_scheme_request_get_uri(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.Native())))
	return C.GoString((*C.char)(cstr))
}

// GetPath retrieves the URI path of the request.
func (r *URISchemeRequest) GetPath() string {
	cstr := C.webkit_uri_scheme_request_get_path(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.Native())))
	return C.GoString((*C.char)(cstr))
}

// Finish completes the request with given data and mimeType.
//
// This method is temporary and should be considered unstable. It will be
// replaced with a method using a Reader for retrieving data instead of the
// raw bytes.
func (r *URISchemeRequest) Finish(data []byte, mimeType string) {
	// TODO: use a reader instead of data; and think of some way to transform it
	// into a GInputStream.
	cstr := C.CString(mimeType)
	defer C.free(unsafe.Pointer(cstr))
	s := C.g_memory_input_stream_new_from_data(
		unsafe.Pointer(&data[0]),
		C.gssize(len(data)),
		nil)
	C.webkit_uri_scheme_request_finish(
		(*C.WebKitURISchemeRequest)(unsafe.Pointer(r.Native())),
		s,
		C.gint64(len(data)),
		(*C.gchar)(cstr))
	C.g_object_unref(C.gpointer(s))
}
