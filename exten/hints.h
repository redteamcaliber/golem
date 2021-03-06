#ifndef GOLEM_HINTS_H
#define GOLEM_HINTS_H

#include <webkitdom/webkitdom.h>
#include <glib.h>
#include "libgolem.h"

// Returns a GList of WebKitDOMNodes, which have been ref'd.
typedef GList *(*NodeSelecter)(GHashTable*, Exten*);

// Do something with a node. Return true to continue hints mode.
typedef gboolean (*NodeExecuter)(WebKitDOMNode*, Exten*);

typedef struct _Hint {
    gchar            *text;
    WebKitDOMElement *div;
} Hint;

struct _HintsMode {
    NodeExecuter executer;
    GHashTable  *hints;
};

gboolean hint_call_by_form_variable_get(WebKitDOMNode*, Exten*);

gboolean hint_call_by_href(WebKitDOMNode*, Exten*);

gboolean hint_call_by_click(WebKitDOMNode*, Exten*);

GList *select_form_text_variables(GHashTable*, Exten*);

GList *select_clickable(GHashTable*, Exten*);

GList *select_links(GHashTable*, Exten*);

gint64
start_hints_mode(NodeSelecter ns, NodeExecuter ne, Exten *exten);

gboolean
filter_hints_mode(const gchar *hints, Exten *exten);

void
end_hints_mode(Exten *exten);

#endif /* GOLEM_HINTS_H */
