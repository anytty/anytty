"""Program-side widget toolkit (port of the Go ``sdk/widgets`` package).

Every module mirrors one Go file and is pure: widgets hold no timers, threads
or protocol state, and ``build()`` returns a composable ``tui2sdk.builder.Node``.

    from tui2sdk.widgets import layout, list, theme

    view = layout.SplitLayout(weights=[1, 3]).build()

``chrome`` keeps the original ``ChromeApp``-style helpers (and their flat
top-level names below) for backward compatibility; the Go-parity modules are
addressed by module name (``widgets.list.List``, ``widgets.theme.Theme``) to
avoid clashing with those legacy names.
"""

from . import (  # noqa: F401
    basics,
    border,
    chart,
    chrome,
    contextmenu,
    datepicker,
    format,
    form,
    hover,
    input,
    layout,
    list,
    modal,
    mouse,
    progress,
    richtext,
    scrollbar,
    select,
    table,
    theme,
    tokens,
    validate,
)
from .chrome import (  # noqa: F401  legacy flat names (v3ui and older code)
    ChromeApp,
    Floating,
    Leaf,
    Pane,
    Split,
    Tab,
    Theme,
    atoi_node,
    cell_width,
    center_pad,
    clamp,
    clear_rect,
    display_width,
    flow_box,
    footer_action_style,
    footer_fallback_style,
    leaf_nodes,
    pad_right,
    put_text,
    self_box,
    short_source_id,
    text_box,
    truncate,
    visible_leaf_count,
)
