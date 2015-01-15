#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>
#include <stdio.h>

#define GOLEM_WEB_ERROR golem_web_error_quark()

GQuark
golem_web_error_quark()
{
    return g_quark_from_static_string("golem-web-error-quark");
}

#define GOLEM_WEB_ERROR_NULL_BODY 0

static const gchar introspection_xml[] =
    "<node>"
    "    <interface name='com.github.tkerber.golem.WebExtension'>"
    "        <property type='x' name='ScrollTop' access='readwrite' />"
    "        <property type='x' name='ScrollLeft' access='readwrite' />"
    "        <property type='x' name='ScrollHeight' access='read' />"
    "        <property type='x' name='ScrollWidth' access='read' />"
    "        <property type='x' name='ScrollTargetTop' access='readwrite' />"
    "        <property type='x' name='ScrollTargetLeft' access='readwrite' />"
    "        <property type='x' name='ScrollTargetHeight' access='read' />"
    "        <property type='x' name='ScrollTargetWidth' access='read' />"
    "        <signal name='VerticalPositionChanged'>"
    "            <arg type='x' name='ScrollTop' />"
    "            <arg type='x' name='ScrollHeight' />"
    "        </signal>"
    "        <signal name='InputFocusChanged'>"
    "            <arg type='b' name='InputFocused' />"
    "        </signal>"
    "    </interface>"
    "</node>";

struct Exten {
    WebKitWebPage     *web_page;
    WebKitDOMDocument *document;
    WebKitDOMElement  *active;
    WebKitDOMElement  *scroll_target;
    GDBusConnection   *connection;
    glong              last_top;
    glong              last_height;
    gboolean           last_input_focus;
    gchar             *object_path;
    gchar             *profile;
    gchar             *golem_name;
};

static void
handle_method_call(GDBusConnection       *connection,
                   const gchar           *sender,
                   const gchar           *object_path,
                   const gchar           *interface_name,
                   const gchar           *method_name,
                   GVariant              *parameters,
                   GDBusMethodInvocation *invocation,
                   gpointer               user_data);

static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data);

static gboolean
handle_set_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GVariant        *value,
                    GError         **error,
                    gpointer         user_data);

static void
scroll_delta(gpointer web_page_p, gint64 delta, gboolean vertical);

static void
scroll_to_top(gpointer web_page_p);

static void
scroll_to_bottom(gpointer web_page_p);

static void
watch_document(WebKitDOMDocument *doc,
               struct Exten      *exten);

static GDBusNodeInfo *introspection_data = NULL;
static const GDBusInterfaceVTable interface_vtable =
{
    handle_method_call,
    handle_get_property,
    handle_set_property
};

static void
handle_method_call(GDBusConnection       *connection,
                   const gchar           *sender,
                   const gchar           *object_path,
                   const gchar           *interface_name,
                   const gchar           *method_name,
                   GVariant              *parameters,
                   GDBusMethodInvocation *invocation,
                   gpointer               user_data)
{
    // No methods currently.
}

static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data)
{
    struct Exten *exten = user_data;
    GVariant *ret = NULL;
    WebKitWebPage *wp = exten->web_page;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
    if(dom == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Document element is NULL.");
        return NULL;
    }
    WebKitDOMElement *e = NULL;
    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
        g_strcmp0(property_name, "ScrollLeft") == 0 ||
        g_strcmp0(property_name, "ScrollHeight") == 0 ||
        g_strcmp0(property_name, "ScrollWidth") == 0) {

        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if (g_strcmp0(property_name, "ScrollTargetTop") == 0 ||
        g_strcmp0(property_name, "ScrollTargetLeft") == 0 ||
        g_strcmp0(property_name, "ScrollTargetHeight") == 0||
        g_strcmp0(property_name, "ScrollTargetWidth") == 0) {

        e = exten->scroll_target;
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Scroll element is NULL.");
        return NULL;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
            g_strcmp0(property_name, "ScrollTargetTop") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_top(e));
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0 ||
            g_strcmp0(property_name, "ScrollTargetLeft") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_left(e));
    } else if(g_strcmp0(property_name, "ScrollHeight") == 0 ||
            g_strcmp0(property_name, "ScrollTargetHeight") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_height(e));
    } else if(g_strcmp0(property_name, "ScrollWidth") == 0 ||
            g_strcmp0(property_name, "ScrollTargetWidth") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_width(e));
    }
    return ret;
}

static gboolean
handle_set_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GVariant        *value,
                    GError         **error,
                    gpointer         user_data)
{
    struct Exten *exten = user_data;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(exten->web_page);
    if(dom == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Document element is NULL.");
        return TRUE;
    }
    WebKitDOMElement *e = NULL;
    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
        g_strcmp0(property_name, "ScrollLeft") == 0) {

        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if (g_strcmp0(property_name, "ScrollTargetTop") == 0 ||
        g_strcmp0(property_name, "ScrollTargetLeft") == 0) {

        e = exten->scroll_target;
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Scroll element is NULL.");
        return TRUE;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
            g_strcmp0(property_name, "ScrollTargetTop") == 0) {
        webkit_dom_element_set_scroll_top(e, g_variant_get_int64(value));
        return TRUE;
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0 ||
            g_strcmp0(property_name, "ScrollTargetLeft") == 0) {
        webkit_dom_element_set_scroll_left(e, g_variant_get_int64(value));
        return TRUE;
    }
    // Currently no properties exist.
    return FALSE;
}

static gboolean
poll_status(gpointer user_data)
{
    struct Exten *exten = user_data;

    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(exten->web_page);
    WebKitDOMElement *e = NULL;
    if(dom != NULL) {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    }

    // Check for current scroll position. If it has changed, signal DBus.
    if(e != NULL) {
        glong top = webkit_dom_element_get_scroll_top(e);
        glong height = webkit_dom_element_get_scroll_height(e);
        if(top != exten->last_top || height != exten->last_height) {
            exten->last_top = top;
            exten->last_height = height;
            g_dbus_connection_emit_signal(
                    exten->connection,
                    NULL,
                    exten->object_path,
                    "com.github.tkerber.golem.WebExtension",
                    "VerticalPositionChanged",
                    g_variant_new("(xx)", top, height),
                    NULL);
        }
    }

    if(dom != NULL) {
        e = webkit_dom_document_get_active_element(dom);
    }


    return G_SOURCE_CONTINUE;
}

static gboolean
golem_is_blocked(const char *uri, struct Exten *exten)
{
    GError *err = NULL;
    GVariant *ret = g_dbus_connection_call_sync(
            exten->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "Blocks",
            g_variant_new("(s)", uri),
            G_VARIANT_TYPE("(b)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            &err);
    if(err != NULL) {
        printf("%s\n", err->message);
        g_error_free(err);
        return false;
    }
    gboolean blocked = g_variant_get_boolean(g_variant_get_child_value(ret, 0));
    g_variant_unref(ret);
    return blocked;
}

static gboolean
golem_request_handler(WebKitWebPage     *page,
                      WebKitURIRequest  *req,
                      WebKitURIResponse *resp,
                      gpointer           exten)
{
    const gchar *uri = webkit_uri_request_get_uri(req);
    return golem_is_blocked(uri, exten);
}

static gboolean
is_scroll_target(WebKitDOMElement *elem)
{
    WebKitDOMElement *parent = webkit_dom_element_get_offset_parent(elem);
    if(parent == NULL) {
        return true;
    }
    glong height = webkit_dom_element_get_scroll_height(elem);
    glong width = webkit_dom_element_get_scroll_width(elem);
    glong parentHeight = webkit_dom_element_get_scroll_height(parent);
    glong parentWidth = webkit_dom_element_get_scroll_width(parent);
    return parentHeight < height || parentWidth < width;
}

static WebKitDOMElement *
get_scroll_target(WebKitDOMElement *elem)
{
    WebKitDOMElement *prev = elem;
    while(!is_scroll_target(elem)) {
        elem = webkit_dom_element_get_offset_parent(elem);
    }
    return elem;
}

static void
active_element_change_cb(WebKitDOMEventTarget *target,
                         WebKitDOMEvent       *event,
                         gpointer              user_data)
{
    struct Exten *exten = user_data;
    WebKitDOMDocument *document;
    g_object_get(target, "document", &document, NULL);
    WebKitDOMElement *active = webkit_dom_document_get_active_element(document);
    if(active == NULL || active == exten->active) {
        return;
    }
    if(WEBKIT_DOM_IS_HTML_IFRAME_ELEMENT(active)) {
        WebKitDOMDocument *doc =
            webkit_dom_html_iframe_element_get_content_document(
                    WEBKIT_DOM_HTML_IFRAME_ELEMENT(active));
        watch_document(doc, exten);
        active_element_change_cb(
                WEBKIT_DOM_EVENT_TARGET(webkit_dom_document_get_default_view(doc)),
                NULL,
                exten);
        return;
    }
    exten->active = active;
    exten->scroll_target = get_scroll_target(active);

    // Check whether the currently active element is an input element.
    // If this has changed, signal DBus.
    //
    // Input elements:
    //
    // WebKitDOMHTMLAppletElement
    // WebKitDOMHTMLEmbedElement
    // WebKitDOMHTMLInputElement
    // WebKitDOMHTMLTextAreaElement
    gboolean input_focus = (
            WEBKIT_DOM_IS_HTML_APPLET_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_EMBED_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_TEXT_AREA_ELEMENT(active));
    if(input_focus != exten->last_input_focus) {
        exten->last_input_focus = input_focus;
        g_dbus_connection_emit_signal(
                exten->connection,
                NULL,
                exten->object_path,
                "com.github.tkerber.golem.WebExtension",
                "InputFocusChanged",
                g_variant_new("(b)", input_focus),
                NULL);
    }
}

static void
watch_document(WebKitDOMDocument *doc,
               struct Exten      *exten)
{
    WebKitDOMEventTarget *target = WEBKIT_DOM_EVENT_TARGET(
            webkit_dom_document_get_default_view(doc));
    webkit_dom_event_target_add_event_listener(
            target,
            "blur",
            G_CALLBACK(active_element_change_cb),
            true,
            exten);
    webkit_dom_event_target_add_event_listener(
            target,
            "focus",
            G_CALLBACK(active_element_change_cb),
            true,
            exten);
    active_element_change_cb(target, NULL, exten);
}

static void
document_loaded_cb(WebKitWebPage *page,
                   gpointer       user_data)
{
    struct Exten *exten = user_data;
    exten->document = webkit_web_page_get_dom_document(page);
    watch_document(exten->document, exten);
}

static void
on_bus_acquired(GDBusConnection *connection,
                const gchar     *name,
                gpointer         user_data)
{
    struct Exten *exten = user_data;
    exten->connection = connection;
    exten->last_top = 0;
    exten->last_height = 0;
    exten->last_input_focus = FALSE;
    exten->object_path = g_strdup_printf(
            "/com/github/tkerber/golem/WebExtension/%s/page%d", 
            exten->profile,
            webkit_web_page_get_id(exten->web_page));
    // Register DBus methods
    gint registration_id = g_dbus_connection_register_object(
            connection,
            exten->object_path,
            introspection_data->interfaces[0],
            &interface_vtable,
            exten,
            NULL,
            NULL);
    g_assert(registration_id > 0);
    // Register 100ms loop polling the current status and sending updates as
    // required.
    g_timeout_add(100, poll_status, exten);
    g_signal_connect(
            exten->web_page,
            "document-loaded",
            G_CALLBACK(document_loaded_cb),
            exten);
    // Register the request signal...
    g_signal_connect(
            exten->web_page,
            "send-request",
            G_CALLBACK(golem_request_handler),
            exten);
}

static void
on_name_lost(GDBusConnection *connection,
             const gchar     *name,
             gpointer         user_data)
{
    g_printerr("Lost DBus connection to main proccess.\n");
    exit(1);
}

// NOTE: There appears to be no way to attach to a web page being destroyed.
// I'm not sure if this means they *aren't* destroyed, or just that it wasn't
// planned for. Either way, it spews errors on the regular update if used
// with a destroyed page.
//
// As there is only one page per process, this isn't a problem, however it is
// worthy of note should this ever change for any reason.
static void
web_page_created_callback(WebKitWebExtension *extension,
                          WebKitWebPage      *web_page,
                          gpointer            user_data)
{
    struct Exten *exten = malloc(sizeof(struct Exten));
    exten->web_page = web_page;
    exten->document = NULL;
    exten->active = NULL;
    exten->scroll_target = NULL;
    exten->profile = user_data;
    exten->golem_name = g_strdup_printf(
            "com.github.tkerber.Golem.%s", exten->profile);
    guint owner_id;

    introspection_data = g_dbus_node_info_new_for_xml(introspection_xml, NULL);
    g_assert(introspection_data != NULL);
    gchar *bus_name = g_strdup_printf(
            "com.github.tkerber.golem.WebExtension.%s.Page%d", 
            exten->profile,
            webkit_web_page_get_id(web_page));
    owner_id = g_bus_own_name(G_BUS_TYPE_SESSION,
            bus_name,
            G_BUS_NAME_OWNER_FLAGS_NONE,
            on_bus_acquired,
            NULL,
            on_name_lost,
            exten,
            NULL);
    free(bus_name);
}

G_MODULE_EXPORT void
webkit_web_extension_initialize_with_user_data(WebKitWebExtension *extension,
                                               GVariant           *data)
{
    gchar *profile = g_variant_dup_string(data, NULL);
    g_signal_connect(extension, "page-created",
        G_CALLBACK(web_page_created_callback), profile);
}
