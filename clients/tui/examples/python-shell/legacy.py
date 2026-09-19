#!/usr/bin/env python3
"""Pixel replica of the legacy default TUI as a TUI v2 layout program.

This program reproduces the layout program that used to drive the old surface
framework (``shell/main.go`` in the repository root: ``build()``/``header``/
``sidebarNode()``/``paneWidths()``/``window()``/``footerText()``/
``pickerOverlay()``/``helpOverlay()``/``promptOverlay()``/``center()``) on top
of the new protocol:

  - the kernel no longer knows borders, so every border, title and focus
    marker is drawn by this program as text rows (the exact legacy rules:
    ``┌─...┐`` with the label embedded at offset 1, content inset by one);
  - terminal panes keep the component-drawn chrome, steered through
    ``content.props`` (chrome.*) so it matches the legacy component styles;
  - styles are explicit theme-free style strings derived from the legacy
    ``render.DefaultTheme`` tokens (see ``LEGACY_STYLES`` and
    ``clients/tui/docs/LEGACY_PARITY.zh-CN.md``).

Interaction replica (M2): tab click/switch, pane focus click, split ``%``/``"``,
close ``x``, divider drag resize (implicit mouse capture), picker
(arrows/enter/click), prompt ``:`` filter + cursor, help ``?``, footer by mode,
``esc`` hierarchy, exit badge and ``Ctrl-E`` restart, scrollback
(``[↑N]`` badge, footer hints, ``y`` copy, ``esc`` live) and toasts.

Modes:
  - default: layout program on the TUI v2 wire (stdin/stdout, ``pb.py``);
  - ``--selftest [--cols N] [--rows N]``: print the rasterized screen and exit
    (used by parity_test.py and troubleshooting);
  - ``--footer-lines``: with the recommended preset, print the 8 v2-scene
    footer rows (``<SCENE>|<line>|``) for the shared golden
    ``golden/recommended_footer_120x32.txt`` (parity_test.py compares them).

Requires Python 3.8+ and the stdlib only.
"""

import json
import os
import sys
import unicodedata

import pb

# ------------------------------------------------------------------- styles
# Legacy theme (tui/render/style.go DefaultTheme) + the token -> SGR mapping
# of tui/program/view.go styleToken(). Colors mixed with mixHostColor are
# inlined here (see LEGACY_PARITY.zh-CN.md "样式规格" for the derivation).
STYLE_FOREGROUND = "fg:#dedbe6"
STYLE_ACCENT = "fg:#a970ff;bold"
STYLE_MUTED = "fg:#b8b1c4;dim"
STYLE_FOOTER = "fg:#b8b1c4"
STYLE_WARNING = "fg:#f0c45c"
# headerWorkspaceFG/BG = mix(StatusFG, Accent, .30) / mix(mix(StatusBG, StatusFG,
# .08), Accent, .24).
STYLE_HEADER = "fg:#d4c0f4;bg:#3c2e55;bold"

# Legacy glyphs. The recommended preset below swaps them for the Nerd Font
# codepoints of tui/docs/tui-v3.recommended.yaml; the defaults stay the exact
# legacy pixel replica (parity_test.py pins them).
GLYPH_WORKSPACE = " WS main "
GLYPH_TAB_CLOSE = "× "
GLYPH_TAB_CREATE = " + "
GLYPH_MARKER = "▎"
GLYPH_TAB_MARKER = "▎"
GLYPH_EXITED = "✕"
GLYPH_ATTACHED = "●"
GLYPH_DETACHED = "○"

# Optional presets: "-config <path>" / ANYTTY_LEGACY_CONFIG, a small JSON file
# {"preset": "recommended"} (or {"icons": {...}, "colors": {...}} overrides).
# Default (no config) reproduces the legacy default UI byte for byte.
LEGACY_CONFIG_ENV = "ANYTTY_LEGACY_CONFIG"
LEGACY_CONFIG_ENV_ALIAS = "ANYTTY_TUI2_LEGACY_CONFIG"

RECOMMENDED_STYLES = {
    "STYLE_FOREGROUND": "fg:#f8f4ff",
    "STYLE_ACCENT": "fg:#f0abfc;bold",
    "STYLE_MUTED": "fg:#9ca3c9;dim",
    "STYLE_FOOTER": "fg:#9ca3c9",
    "STYLE_WARNING": "fg:#fde68a",
    "STYLE_HEADER": "fg:#1b1230;bg:#f0abfc;bold",
}

RECOMMENDED_ICONS = {
    "GLYPH_WORKSPACE": " 󰙅 local ",
    "GLYPH_TAB_CLOSE": "󰅖 ",
    "GLYPH_TAB_CREATE": " 󰐕 ",
    "GLYPH_MARKER": "▎",
    "GLYPH_TAB_MARKER": "⎇",
    "GLYPH_EXITED": "󰅖",
    "GLYPH_ATTACHED": "●",
    "GLYPH_DETACHED": "○",
    # Footer key-group glyphs (RECOMMENDED_CONFIG "footer 规格表").
    "GLYPH_FOCUS": "\U000F0734",
    "GLYPH_PAGE_UP": "\U000F005D",
    "GLYPH_PAGE_DOWN": "\U000F0045",
    "GLYPH_ATTACH": "\U000F02FA",
    "GLYPH_RUN": "\U000F0627",
    "GLYPH_SUMMARY_WORKSPACE": "\U000F0645",
    "GLYPH_SUMMARY_FLOATING": "\U000F0E59",
    "GLYPH_SUMMARY_TERMINALS": "\U0000F489",
    "GLYPH_MODE_NORMAL": "\U000F030C",
    "GLYPH_MODE_PANE": "\uEBEB",
    "GLYPH_MODE_PICKER": "\U000F0C7C",
    "GLYPH_MODE_SCROLL": "\U000F018F",
    "GLYPH_SLOT_CLOSE": "\U000F0156",
    "GLYPH_SLOT_RESTART": "\U000F0450",
    "GLYPH_SLOT_SPLIT_H": "\uEB56",
    "GLYPH_SLOT_SPLIT_V": "\uEB57",
    "GLYPH_SUMMARY_TAB": "\U000F04E9",
    "GLYPH_WORKSPACE_ICON": "\U000F0645",
}

# Recommended-profile footer state: the old recommended shortcut scenes are
# clipped to the actions v2 implements and rendered with the exact old
# spacing. Mirrors clients/tui/cmd/tui2-shell/footer.go; the shared golden is
# clients/tui/examples/python-shell/golden/recommended_footer_120x32.txt.
FOOTER_RECOMMENDED = False
WORKSPACE_NAME = "main"

GLYPH_FOCUS = ""
GLYPH_PAGE_UP = ""
GLYPH_PAGE_DOWN = ""
GLYPH_ATTACH = ""
GLYPH_RUN = ""
GLYPH_SUMMARY_WORKSPACE = ""
GLYPH_SUMMARY_FLOATING = ""
GLYPH_SUMMARY_TERMINALS = ""
GLYPH_MODE_NORMAL = ""
GLYPH_MODE_PANE = ""
GLYPH_MODE_PICKER = ""
GLYPH_MODE_SCROLL = ""
GLYPH_SLOT_CLOSE = ""
GLYPH_SLOT_RESTART = ""
GLYPH_SLOT_SPLIT_H = ""
GLYPH_SLOT_SPLIT_V = ""
GLYPH_SUMMARY_TAB = ""
GLYPH_WORKSPACE_ICON = ""


def load_preset(argv):
    """Reads the optional preset config: -config/--config or the env var."""
    path = None
    for flag in ("-config", "--config"):
        if flag in argv:
            index = argv.index(flag)
            if index + 1 >= len(argv):
                raise ValueError("%s needs a path" % flag)
            path = argv[index + 1]
    if path is None:
        path = os.environ.get(LEGACY_CONFIG_ENV) or os.environ.get(LEGACY_CONFIG_ENV_ALIAS)
    if not path:
        return None
    with open(path, "r", encoding="utf-8") as handle:
        spec = json.load(handle)
    if not isinstance(spec, dict):
        raise ValueError("preset config must be a JSON object")
    return spec


def apply_preset(spec):
    """Rebinds the legacy style/glyph globals for the requested preset."""
    global STYLE_FOREGROUND, STYLE_ACCENT, STYLE_MUTED, STYLE_FOOTER, STYLE_WARNING, STYLE_HEADER
    global GLYPH_WORKSPACE, GLYPH_TAB_CLOSE, GLYPH_TAB_CREATE, GLYPH_MARKER, GLYPH_TAB_MARKER
    global GLYPH_EXITED, GLYPH_ATTACHED, GLYPH_DETACHED, LEGACY_STYLES
    global GLYPH_FOCUS, GLYPH_PAGE_UP, GLYPH_PAGE_DOWN, GLYPH_ATTACH, GLYPH_RUN
    global GLYPH_SUMMARY_WORKSPACE, GLYPH_SUMMARY_FLOATING, GLYPH_SUMMARY_TERMINALS
    global GLYPH_MODE_NORMAL, GLYPH_MODE_PANE, GLYPH_MODE_PICKER, GLYPH_MODE_SCROLL
    global GLYPH_SLOT_CLOSE, GLYPH_SLOT_RESTART, GLYPH_SLOT_SPLIT_H, GLYPH_SLOT_SPLIT_V, GLYPH_SUMMARY_TAB
    global GLYPH_WORKSPACE_ICON
    global FOOTER_RECOMMENDED, WORKSPACE_NAME
    global ENDPOINTS
    endpoints = spec.get("endpoints") or []
    if not isinstance(endpoints, list):
        raise ValueError("endpoints must be a list")
    ENDPOINTS = [endpoint for endpoint in endpoints if isinstance(endpoint, dict)]
    preset = (spec.get("preset") or "").strip().lower()
    if preset == "recommended":
        for name, value in RECOMMENDED_STYLES.items():
            globals()[name] = value
        for name, value in RECOMMENDED_ICONS.items():
            globals()[name] = value
        FOOTER_RECOMMENDED = True
        WORKSPACE_NAME = "local"
    elif preset not in ("", "legacy", "default"):
        raise ValueError("unknown preset %r (want \"recommended\")" % spec.get("preset"))
    colors = spec.get("colors") or {}
    for name, value in colors.items():
        if name not in RECOMMENDED_STYLES:
            raise ValueError("unknown color slot %r" % name)
        globals()[name] = value
    icons = spec.get("icons") or {}
    for name, value in icons.items():
        if name not in RECOMMENDED_ICONS:
            raise ValueError("unknown icon slot %r" % name)
        globals()[name] = value
    LEGACY_STYLES.update({
        "": STYLE_FOREGROUND,
        "foreground": STYLE_FOREGROUND,
        "accent": STYLE_ACCENT,
        "muted": STYLE_MUTED,
        "footer": STYLE_FOOTER,
        "warning": STYLE_WARNING,
        "header": STYLE_HEADER,
    })


LEGACY_STYLES = {
    "": STYLE_FOREGROUND,
    "foreground": STYLE_FOREGROUND,
    "accent": STYLE_ACCENT,
    "muted": STYLE_MUTED,
    "footer": STYLE_FOOTER,
    "warning": STYLE_WARNING,
    "header": STYLE_HEADER,
}

# Optional configured endpoints (tui2.json "endpoints"). The default replica
# has none, so its pixels stay identical; a daemon endpoint turns on picker
# grouping (the same compatibility rule as the Go shell).
ENDPOINTS = []


def picker_grouping_enabled():
    return any((endpoint.get("kind") or "command") == "daemon" for endpoint in ENDPOINTS)


def endpoint_by_name(name):
    for endpoint in ENDPOINTS:
        if endpoint.get("name") == name:
            return endpoint
    return None


def recommended_footer_spec(scene):
    """The recommended-profile footer spec for one v2 scene.

    Mirrors clients/tui/cmd/tui2-shell/footer.go: old yaml badge/icon/label/order
    clipped to the actions v2 implements. Returns (badge, items, summary)
    where items are (key, icon, label) triples.
    """
    if scene == "PANE":
        return GLYPH_MODE_PANE + " PANE", [
            ("X", GLYPH_SLOT_CLOSE, "CLOSE"),
            ("%", GLYPH_SLOT_SPLIT_H, "VSPLIT"),
            ('"', GLYPH_SLOT_SPLIT_V, "HSPLIT"),
            ("TAB", GLYPH_FOCUS, "FOCUS"),
            ("ESC", "", "BACK"),
        ], False
    if scene == "PICKER":
        return GLYPH_MODE_PICKER + " PICK", [
            ("", "", "↑/↓ SELECT"),
            ("ENTER", GLYPH_ATTACH, "ATTACH"),
            ("ESC", "", "BACK"),
        ], False
    if scene == "PROMPT":
        return "PROMPT", [("ENTER", GLYPH_RUN, "RUN")], False
    if scene == "HELP":
        return "HELP", [], False
    if scene == "SCROLL":
        return GLYPH_MODE_SCROLL + " COPY", [
            ("PGUP", GLYPH_PAGE_UP, "OLDER"),
            ("PGDN", GLYPH_PAGE_DOWN, "NEWER"),
            ("Y", GLYPH_MODE_SCROLL, "COPY"),
            ("ESC", "", "LIVE"),
        ], False
    if scene == "EXITED":
        return GLYPH_MODE_NORMAL + " CTRL", [
            ("E", GLYPH_SLOT_RESTART, "RESTART"),
            ("F", GLYPH_MODE_PICKER, "PICK"),
        ], True
    return GLYPH_MODE_NORMAL + " CTRL", [
        ("P", GLYPH_MODE_PANE, "PANE"),
        ("T", GLYPH_SUMMARY_TAB, "TAB"),
        ("W", GLYPH_WORKSPACE_ICON, "WORKSPACE"),
        ("F", GLYPH_MODE_PICKER, "PICK"),
    ], True

SIDEBAR_WIDTH = 30
PICKER_WIDTH = 56
PICKER_HEIGHT = 12
HELP_WIDTH = 64
PROMPT_WIDTH = 56
MANAGER_WIDTH = 60
MANAGER_ITEMS = 200

INITIAL_LINES = [
    "No terminal connected",
    "Choose a terminal or create one.",
    "Press Ctrl-F to open the terminal picker.",
]

HELP_LINES = [
    "Keys",
    "  ctrl-p   pane mode (% split, \" split, x close, tab focus)",
    "  ctrl-f   terminal picker",
    "  ctrl-t   new tab      ctrl-w  toggle sidebar",
    "  1..9     switch tab    ?       this help",
    "  :        command palette",
    "Mouse",
    "  click tab / tab × / pane / picker item / + create",
    "  wheel over a pane to scroll; drag a divider to resize",
]

PROMPT_COMMANDS = [
    "split row", "split col", "close pane", "kill pane", "new tab", "close tab",
    "toggle sidebar", "open picker", "terminal manager", "create terminal", "help", "quit",
]

# chrome.* prop keys understood by the terminal component (PROTOCOL §5).
P_BORDER = "chrome.border"
P_BORDER_FOCUS = "chrome.border_focus"
P_BORDER_DEAD = "chrome.border_dead"
P_TITLE = "chrome.title"
P_BADGE = "chrome.badge"


# ------------------------------------------------------------ text helpers

def cell_width(ch):
    """Display cells of one character (combining marks are zero width)."""
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1


def display_width(text):
    """Display cells of text (the legacy renderer counted East-Asian wide
    glyphs as two cells; the replica keeps that rule)."""
    return sum(cell_width(ch) for ch in text)


def truncate(text, max_width):
    if max_width <= 0:
        return ""
    out = []
    used = 0
    for ch in text:
        cells = cell_width(ch)
        if used + cells > max_width and cells:
            break
        out.append(ch)
        used += cells
    return "".join(out)


def embed_title(top, title):
    """Legacy embedTitle (tui/program/view.go): copy " title " over the rule
    starting at offset 1; keep the plain rule when it does not fit."""
    runes = list(top)
    label = list(" " + title + " ")
    if len(label) + 2 > len(runes):
        return top
    runes[1:1 + len(label)] = label
    return "".join(runes)


def top_border(width, title):
    if width < 2:
        return "┌" if width == 1 else ""
    return embed_title("┌" + "─" * (width - 2) + "┐", title)


def bottom_border(width):
    if width < 2:
        return "└" if width == 1 else ""
    return "└" + "─" * (width - 2) + "┘"


def window(lines, offset):
    """Legacy window(): drop the first `offset` lines."""
    if offset < 0:
        offset = 0
    if offset >= len(lines):
        return []
    return lines[offset:]


def clamp(value, low, high):
    if value < low:
        return low
    if value > high:
        return high
    return value


def pane_extents(available, weights):
    """Legacy paneWidths() arithmetic: proportional floor split, the last
    extent absorbs the remainder, every extent is at least one cell."""
    count = len(weights)
    if count == 0:
        return []
    if available < count:
        available = count
    total = 0
    for weight in weights:
        total += weight if weight > 0 else 1
    if total <= 0:
        total = count
    out = []
    used = 0
    for weight in weights:
        weight = weight if weight > 0 else 1
        part = available * weight // total
        if part < 1:
            part = 1
        out.append(part)
        used += part
    out[-1] += available - used
    if out[-1] < 1:
        out[-1] = 1
    return out


# --------------------------------------------------------------- box trees

def text_box(node_id, text, style, width=0, height=0, input_kinds=None, cursor=None):
    box = {"id": node_id, "content": {"text": text}}
    if style:
        box["style"] = style
    if width or height:
        box["size"] = (width, height, 0)
    if input_kinds:
        box["input"] = list(input_kinds)
    if cursor is not None:
        box["cursor"] = cursor
    return box


def flow_box(node_id, flow, children, width=0, height=0, flex=0, pos=None):
    box = {"id": node_id, "flow": flow, "children": children}
    if width or height or flex:
        box["size"] = (width, height, flex)
    if pos is not None:
        box["pos"] = pos
    return box


def self_box(node_id, source_id, props, width, height, focused, input_kinds):
    box = {
        "id": node_id,
        "size": (width, height, 0),
        "content": {"self": source_id, "props": dict(props)},
    }
    if input_kinds:
        box["input"] = list(input_kinds)
    if focused:
        box["focused"] = True
    return box


def panel_rows(width, height, title, border_style, lines, line_style):
    """A legacy bordered panel as a column of full-width text rows.

    Legacy ``borderLines`` draws every row itself (top rule with the label,
    ``│`` + spaces + ``│`` middle rows, bottom rule) and the content frame is
    blitted on top of the middle rows, clearing the content rectangle. The
    replica therefore draws the two border cells and the content text per row;
    the cells after the text keep the framebuffer default, exactly like the
    cleared legacy content rectangle.
    """
    if width < 2 or height < 2:
        first = lines[0] if lines else ""
        return [text_box("", first, line_style, width=max(0, width), height=max(0, height))]
    rows = [text_box("", top_border(width, title), border_style, width=width, height=1)]
    for index in range(height - 2):
        text = lines[index] if index < len(lines) else ""
        rows.append(flow_box("", "row", [
            text_box("", "│", border_style, width=1),
            text_box("", text, line_style, width=width - 2),
            text_box("", "│", border_style, width=1),
        ], height=1))
    rows.append(text_box("", bottom_border(width), border_style, width=width, height=1))
    return rows


def overlay_panel(node_id, title, width, height, x, y, rows):
    """A centered overlay: pos subtree with border/rule rows, every inner row
    carrying its own style (unselected picker rows keep the accent ``│``)."""
    children = [text_box("", top_border(width, title), STYLE_ACCENT, width=width, height=1)]
    for index in range(height - 2):
        row = rows[index] if index < len(rows) else {"text": "", "style": STYLE_FOREGROUND}
        cursor = row.get("cursor")
        children.append(flow_box("", "row", [
            text_box("", "│", STYLE_ACCENT, width=1),
            text_box(row.get("id", ""), row["text"], row.get("style", STYLE_FOREGROUND),
                     width=width - 2, input_kinds=row.get("input"), cursor=cursor),
            text_box("", "│", STYLE_ACCENT, width=1),
        ], height=1))
    children.append(text_box("", bottom_border(width), STYLE_ACCENT, width=width, height=1))
    return flow_box(node_id, "col", children, width=width, height=height, pos=(x, y))


# ------------------------------------------------------------------ model

class Pane:
    __slots__ = ("id", "title", "lines", "scroll", "source_id")

    def __init__(self, pane_id, title, lines):
        self.id = pane_id
        self.title = title
        self.lines = list(lines)
        self.scroll = 0
        self.source_id = ""


class Tab:
    __slots__ = ("title", "panes", "focus", "weights", "flow")

    def __init__(self, title, panes, flow="row"):
        self.title = title
        self.panes = panes
        self.focus = 0
        self.weights = [1] * len(panes)
        self.flow = flow


class Program:
    """Legacy state machine + view builder + wire program."""

    def __init__(self, out=None, cols=120, rows=30, view_id="", epoch=0):
        self.out = out
        self.view_id = view_id
        self.epoch = epoch
        self.rev = 0
        self.cols = cols
        self.rows = rows
        self.tab_seq = 1
        self.pane_seq = 0
        self.tabs = [Tab("main", [self.new_pane("unconnected", INITIAL_LINES)])]
        self.active = 0
        self.mode = ""
        self.overlay = ""
        self.picker = 0
        self.sidebar = True
        self.toast = ""
        self.dragging = 0
        self.prompt = ""
        self.prompt_sel = 0
        self.manager_pos = 0
        self.sources = []
        self.sources_ready = False
        self.scroll_source = ""
        self.scroll_offset = 0
        self.request_id = 0
        self.pending = {}
        self.cursor = None

    # ------------------------------------------------------------- state

    def new_pane(self, title, lines):
        self.pane_seq += 1
        return Pane("p%d" % self.pane_seq, title, lines)

    def active_tab(self):
        return self.tabs[self.active]

    def focus_pane(self):
        tab = self.active_tab()
        if tab.focus < 0 or tab.focus >= len(tab.panes):
            tab.focus = 0
        return tab.panes[tab.focus] if tab.panes else None

    def pane_by_id(self, pane_id):
        for tab in self.tabs:
            for pane in tab.panes:
                if pane.id == pane_id:
                    return pane
        return None

    def source_by_id(self, source_id):
        for source in self.sources:
            if source.get("id") == source_id:
                return source
        return None

    def sources_of_kind(self, kind):
        return [source for source in self.sources if source.get("kind") == kind]

    def terminals(self):
        return self.sources_of_kind("terminal")

    def bound_pane(self):
        pane = self.focus_pane()
        if pane is None or not pane.source_id:
            return None, None
        return pane, self.source_by_id(pane.source_id)

    def focused_terminal(self):
        pane, source = self.bound_pane()
        if source is None or not source.get("attached"):
            return None, None
        return pane, source

    def focused_exited(self):
        pane, source = self.bound_pane()
        if source is None or not source.get("attached"):
            return None, None
        if source.get("exited"):
            return pane, source
        return None, None

    def accepts_terminal_input(self):
        return self.overlay == "" and self.mode == ""

    def pane_area_width(self):
        return self.cols - (SIDEBAR_WIDTH if self.sidebar else 0)

    # -------------------------------------------------------- view build

    def view(self):
        tab = self.active_tab()
        body_height = max(1, self.rows - 2)
        main = flow_box("", "col", [
            self.header_node(),
            self.body_node(tab, body_height),
            self.footer_node(tab),
        ])
        children = [main]
        overlay = self.overlay_node()
        if overlay is not None:
            children.append(overlay)
        toast = self.toast_node()
        if toast is not None:
            children.append(toast)
        return flow_box("root", "stack", children)

    def header_node(self):
        children = [text_box("ws", GLYPH_WORKSPACE, STYLE_HEADER)]
        for index, tab in enumerate(self.tabs):
            if index == self.active:
                marker, style = GLYPH_TAB_MARKER, STYLE_ACCENT
            else:
                marker, style = " ", STYLE_FOREGROUND
            children.append(text_box("tab:%d" % index, " %s%d %s " % (marker, index + 1, tab.title),
                                     style, input_kinds=["mouse"]))
            children.append(text_box("tabclose:%d" % index, GLYPH_TAB_CLOSE, STYLE_MUTED, input_kinds=["mouse"]))
        children.append(text_box("tabcreate", GLYPH_TAB_CREATE, STYLE_MUTED, input_kinds=["mouse"]))
        return flow_box("header", "row", children, height=1)

    def body_node(self, tab, height):
        """The sidebar is always the left column of the body row; the pane
        area is one container whose flow follows the split direction."""
        children = []
        if self.sidebar:
            children.append(self.sidebar_node())
        children.append(self.panes_node(tab, height))
        return flow_box("body", "row", children, flex=1)

    def panes_node(self, tab, height):
        area_width = self.pane_area_width()
        count = len(tab.panes)
        children = []
        if tab.flow == "col":
            extents = pane_extents(height - (count - 1), list(tab.weights))
            for index, pane in enumerate(tab.panes):
                if index > 0:
                    children.append(text_box("divider:%d" % index, "─" * area_width, STYLE_MUTED,
                                             width=area_width, height=1, input_kinds=["mouse"]))
                children.append(self.pane_node(pane, area_width, extents[index], index == tab.focus))
        else:
            extents = pane_extents(area_width - (count - 1), list(tab.weights))
            for index, pane in enumerate(tab.panes):
                if index > 0:
                    children.append(text_box("divider:%d" % index, "│", STYLE_MUTED,
                                             width=1, height=height, input_kinds=["mouse"]))
                children.append(self.pane_node(pane, extents[index], height, index == tab.focus))
        return flow_box("panes", tab.flow, children, width=area_width, height=height)

    def sidebar_node(self):
        status = flow_box("status", "col", panel_rows(
            SIDEBAR_WIDTH, 6, "status", STYLE_MUTED, self.status_lines(), STYLE_MUTED,
        ), width=SIDEBAR_WIDTH, height=6)
        return flow_box("sidebar", "col", [status], width=SIDEBAR_WIDTH)

    def status_lines(self):
        tab = self.active_tab()
        return [
            "tabs      %d" % len(self.tabs),
            "panes     %d" % len(tab.panes),
            "mode      " + self.mode,
            self.world_line(),
        ]

    def world_line(self):
        if not self.sources_ready:
            return "host      (no world yet)"
        terminals = self.terminals()
        endpoints = len({(source.get("endpoint") or "local") for source in terminals}) or 1
        return "host panes %d · endpoints %d · focus %s" % (
            len(terminals), endpoints, self.view_id or "-")

    def pane_node(self, pane, width, height, focused):
        source = self.source_by_id(pane.source_id) if pane.source_id else None
        if pane.source_id:
            if source is None:
                # The bind response can arrive before the sources snapshot:
                # keep rendering the component chrome (and the focus flag) so
                # keys reach the PTY as soon as the terminal is bound.
                source = {"id": pane.source_id, "terminal_id": short_source_id(pane.source_id),
                          "title": "", "endpoint": "local", "attached": True, "exited": False}
            return self.terminal_node(pane, source, width, height, focused)
        marker, style = (GLYPH_MARKER + " ", STYLE_ACCENT) if focused else ("  ", STYLE_MUTED)
        lines = window(pane.lines, pane.scroll)
        rows = panel_rows(width, height, marker + pane.title, style, lines, STYLE_MUTED)
        return flow_box(pane.id, "col", rows, width=width, height=height)

    def terminal_node(self, pane, source, width, height, focused):
        """Component-drawn chrome steered by props so it matches the legacy
        component (border: exited->warning, focused->accent, else muted)."""
        exited = bool(source.get("exited"))
        if exited:
            label = STYLE_WARNING
        elif focused and self.accepts_terminal_input():
            label = STYLE_ACCENT
        else:
            label = STYLE_MUTED
        props = {
            P_BORDER: STYLE_MUTED,
            P_BORDER_FOCUS: STYLE_ACCENT,
            P_BORDER_DEAD: STYLE_WARNING,
            P_TITLE: label,
            P_BADGE: label,
        }
        return self_box(pane.id, source.get("id", ""), props, width, height,
                        focused and self.accepts_terminal_input(),
                        ["key", "paste", "wheel"])

    def footer_node(self, tab):
        footer = text_box("footer", self.footer_text(tab), STYLE_FOOTER, height=1)
        return footer

    def overlay_node(self):
        if self.overlay == "picker":
            return self.picker_overlay()
        if self.overlay == "help":
            return self.help_overlay()
        if self.overlay == "prompt":
            return self.prompt_overlay()
        if self.overlay == "manager":
            return self.manager_overlay()
        return None

    def toast_node(self):
        if not self.toast:
            return None
        y = max(0, self.rows - 2)
        box = text_box("toast", " " + self.toast + "  ·  any key dismiss", STYLE_WARNING, height=1)
        box["pos"] = (1, y)
        return box

    def center(self, width, height):
        width = min(width, self.cols)
        height = min(height, self.rows)
        x = max(0, (self.cols - width) // 2)
        y = max(0, (self.rows - height) // 2)
        return width, height, x, y

    def source_picker_entry(self, source, endpoint_label="local"):
        label = (source.get("title") or "").strip() or source.get("terminal_id") or ""
        if source.get("exited"):
            marker = GLYPH_EXITED
        elif source.get("attached"):
            marker = GLYPH_ATTACHED
        else:
            marker = GLYPH_DETACHED
        return {"text": marker + " " + endpoint_label + "  " + label,
                "selectable": True, "source": source}

    def picker_entries(self):
        """Structured picker rows.

        Without a configured daemon endpoint this is the exact legacy list
        (sources then "+ New terminal", pinned by the parity scenarios). A
        daemon endpoint turns on group headers and configured endpoint rows,
        mirroring the Go shell's compatibility rule.
        """
        entries = []
        if not picker_grouping_enabled():
            for source in self.terminals():
                entries.append(self.source_picker_entry(source))
            entries.append({"text": "+ New terminal", "selectable": True})
            return entries
        last = None
        for source in self.terminals():
            endpoint = source.get("endpoint") or "local"
            if endpoint != last:
                entries.append({"text": "  " + endpoint, "selectable": False, "header": True})
                last = endpoint
            entries.append(self.source_picker_entry(source, endpoint_label=endpoint))
        for endpoint in ENDPOINTS:
            name = endpoint.get("name") or ""
            if not name:
                continue
            if (endpoint.get("kind") or "command") == "daemon" and name != last:
                entries.append({"text": "  " + name, "selectable": False, "header": True})
                last = name
            label = endpoint.get("label") or name
            entries.append({"text": "+ " + label, "selectable": True, "endpoint": endpoint})
        entries.append({"text": "  local", "selectable": False, "header": True})
        entries.append({"text": "+ New terminal", "selectable": True})
        return entries

    def picker_selectables(self):
        return [entry for entry in self.picker_entries() if entry.get("selectable")]

    def picker_items(self):
        return [entry["text"] for entry in self.picker_entries()]

    def picker_overlay(self):
        rows = []
        selected = 0
        for entry in self.picker_entries():
            if entry.get("header"):
                rows.append({"text": entry["text"], "style": STYLE_MUTED})
                continue
            marker, style = ("▸ ", STYLE_ACCENT) if selected == self.picker else ("  ", STYLE_MUTED)
            rows.append({"text": marker + entry["text"], "style": style,
                         "id": "picker:%d" % selected, "input": ["mouse"]})
            selected += 1
        rows.append({"text": "↑↓ select · enter/click attach · esc close", "style": STYLE_MUTED})
        width, height, x, y = self.center(PICKER_WIDTH, PICKER_HEIGHT)
        return overlay_panel("picker", "Terminal Picker", width, height, x, y, rows)

    def prompt_matches(self):
        query = self.prompt.strip().lower()
        if not query:
            return list(PROMPT_COMMANDS)
        return [command for command in PROMPT_COMMANDS if query in command.lower()]

    def prompt_overlay(self):
        text = ": " + self.prompt
        rows = [{"text": text, "style": STYLE_ACCENT,
                 "cursor": {"row": 0, "col": display_width(text)}}]
        matches = self.prompt_matches()
        for index, command in enumerate(matches):
            marker, style = ("▸ ", STYLE_ACCENT) if index == self.prompt_sel else ("  ", STYLE_MUTED)
            rows.append({"text": marker + command, "style": style, "id": "prompt:%d" % index})
        rows.append({"text": "enter run · ↑↓ select · esc close", "style": STYLE_MUTED})
        width, height, x, y = self.center(PROMPT_WIDTH, len(matches) + 5)
        return overlay_panel("prompt", "Command", width, height, x, y, rows)

    def help_overlay(self):
        rows = [{"text": "Help", "style": STYLE_ACCENT}]
        for line in HELP_LINES:
            rows.append({"text": line, "style": STYLE_FOREGROUND})
        rows.append({"text": "esc close", "style": STYLE_MUTED})
        width, height, x, y = self.center(HELP_WIDTH, len(HELP_LINES) + 4)
        return overlay_panel("help", "Help", width, height, x, y, rows)

    def manager_overlay(self):
        rows = []
        for index in range(13):
            item = self.manager_pos + index
            if item >= MANAGER_ITEMS:
                break
            status = "exited" if item % 5 == 4 else "running"
            rows.append({"text": "terminal-%03d   %-8s  local" % (item, status),
                         "style": STYLE_FOREGROUND})
        width, height, x, y = self.center(MANAGER_WIDTH, 13 + 2)
        return overlay_panel("manager", "Terminal Manager", width, height, x, y, rows)

    def footer_hints(self):
        if self.scroll_source:
            return "[PgUp/PgDn] SCROLL +%d │ [y] COPY │ [esc] LIVE" % self.scroll_offset
        pane, source = self.focused_exited()
        if source is not None:
            return "[Ctrl+E] RESTART │ [Ctrl+F] PICKER │ terminal exited"
        if self.mode == "PANE":
            return "[%] SPLIT H │ [\"] SPLIT V │ [x] CLOSE │ [tab] NEXT │ [Ctrl+F] PICKER │ [:] CMD │ [esc] EXIT"
        if self.mode == "RESIZE":
            return "[Ctrl+P] PANE │ [Ctrl+G] GLOBAL │ [?] HELP │ [esc] BACK"
        if self.mode == "GLOBAL":
            return "[Ctrl+T] NEW TAB │ [Ctrl+W] SIDEBAR │ [Ctrl+E] RESTART │ [Ctrl+F] PICKER │ [esc] BACK"
        return "[Ctrl+P] PANE │ [Ctrl+R] RESIZE │ [Ctrl+G] GLOBAL │ [Ctrl+F] PICKER │ [?] HELP │ [:] CMD"

    def recommended_footer_scene(self):
        """Scene precedence mirrors tui2's footerSceneFor (overlays win)."""
        if self.overlay == "picker":
            return "PICKER"
        if self.overlay == "prompt":
            return "PROMPT"
        if self.overlay == "help":
            return "HELP"
        if self.scroll_source:
            return "SCROLL"
        if self.focused_exited()[1] is not None:
            return "EXITED"
        if self.mode == "PANE":
            return "PANE"
        return "NORMAL"

    def recommended_footer_line(self):
        scene = self.recommended_footer_scene()
        badge, items, summary = recommended_footer_spec(scene)
        left = ""
        if badge:
            left = " " + badge + " "
        for key, icon, label in items:
            if left:
                left += " · "
            left += " " + " ".join(part for part in (key, icon, label) if part)
        parts = []
        if self.toast:
            parts.append(self.toast)
        if summary:
            parts.append(GLYPH_SUMMARY_WORKSPACE + " " + WORKSPACE_NAME)
            parts.append(GLYPH_SUMMARY_FLOATING + " 0")
            parts.append(GLYPH_SUMMARY_TERMINALS + " " + str(len(self.terminals())))
        right = (" " + " ".join(parts) + " ") if parts else ""
        pad = self.cols - display_width(left) - display_width(right)
        if pad < 1:
            pad = 1
        return left + " " * pad + right

    def footer_text(self, tab):
        if FOOTER_RECOMMENDED:
            return self.recommended_footer_line()
        left = " " + self.footer_hints()
        right = "ws:%s tabs:%d panes:%d " % (tab.title, len(self.tabs), len(tab.panes))
        pad = self.cols - display_width(left) - display_width(right)
        if pad < 1:
            pad = 1
        return left + " " * pad + right

    # ------------------------------------------------------------- wire

    def on_hello(self, hello):
        self.view_id = hello.get("view_id", "")
        self.epoch = hello.get("epoch", 0)
        self.cols = hello.get("cols") or self.cols
        self.rows = hello.get("rows") or self.rows
        for endpoint in ENDPOINTS:
            if (endpoint.get("kind") or "command") != "daemon":
                continue
            self.emit_result("endpoint.sync", {
                "endpoint": endpoint.get("name") or "",
                "kind": "daemon",
                "socket": endpoint.get("socket") or "",
                "address": endpoint.get("address") or "",
                "connect_mode": endpoint.get("connect_mode") or "local-unix",
            })
        self.commit()

    def on_event(self, event):
        kind = event.get("kind")
        if kind == "resize":
            self.cols = event.get("cols") or self.cols
            self.rows = event.get("rows") or self.rows
        elif kind == "key":
            self.handle_key(event.get("key") or event.get("char") or "",
                            event.get("char") or "")
        elif kind == "mouse":
            self.handle_mouse(event)
        elif kind == "wheel":
            self.handle_wheel(event)
        elif kind == "paste":
            self.handle_paste(event.get("text") or "")
        elif kind == "sources":
            self.on_sources(event.get("items") or [])
        elif kind == "notice":
            level = event.get("level") or "notice"
            self.toast = level + ": " + (event.get("message") or "")
        self.commit()

    def on_response(self, response):
        callback = self.pending.pop(response.get("request_id"), None)
        if callback is not None:
            callback(response)
        self.commit()

    def on_sources(self, items):
        self.sources = [source for source in items if source.get("id")]
        present = {source.get("id") for source in self.terminals()}
        for tab in self.tabs:
            for pane in tab.panes:
                if pane.source_id and pane.source_id not in present:
                    pane.source_id = ""
                    pane.scroll = 0
        if not self.sources_ready:
            self.sources_ready = True
            if not self.terminals() and self.overlay == "":
                self.open_picker()

    def emit_result(self, method, params=None, on_response=None):
        self.request_id += 1
        request_id = self.request_id
        if on_response is not None:
            self.pending[request_id] = on_response
        if self.out is None:
            return request_id
        payload = pb.encode_result(request_id, self.epoch, method, params or {})
        self.out.write(pb.frame(pb.RESULT, payload))
        self.out.flush()
        return request_id

    def commit(self):
        if self.out is None or not self.view_id:
            return
        self.rev += 1
        claim, all_keys = self.claim()
        payload = pb.encode_view(self.epoch, self.rev, claim, all_keys, self.view())
        self.out.write(pb.frame(pb.VIEW, payload))
        self.out.flush()

    def claim(self):
        if self.overlay:
            return [], True
        return ["ctrl-p", "ctrl-r", "ctrl-g", "ctrl-t", "ctrl-w", "ctrl-f", "ctrl-e", "tab"], False

    # ------------------------------------------------------------- input

    def open_picker(self):
        self.overlay = "picker"
        self.picker = 0

    def close_overlay(self):
        self.overlay = ""

    def handle_key(self, key, char):
        if self.toast:
            self.toast = ""
        if self.overlay:
            self.handle_overlay_key(key, char)
            return
        if self.mode == "PANE":
            self.handle_pane_key(key, char)
            return
        self.handle_normal_key(key, char)

    def handle_normal_key(self, key, char):
        if key == "ctrl-p":
            self.mode = "PANE"
        elif key == "ctrl-r":
            self.mode = "RESIZE"
        elif key == "ctrl-g":
            self.mode = "GLOBAL"
        elif key == "esc":
            self.mode = ""
        elif key == "ctrl-f":
            self.open_picker()
        elif key == "ctrl-t":
            self.new_tab()
        elif key == "ctrl-w":
            self.sidebar = not self.sidebar
        elif key == "ctrl-e":
            self.restart_focused()
        elif key == "tab":
            self.focus_next()
        elif len(key) == 1 and key.isdigit() and key != "0":
            index = ord(key) - ord("1")
            if 0 <= index < len(self.tabs):
                self.active = index
        elif len(key) == 1:
            if key == "?":
                self.overlay = "help"
            elif key == ":":
                self.overlay = "prompt"
                self.prompt = ""
                self.prompt_sel = 0
            else:
                self.append_to_pane(key)

    def handle_pane_key(self, key, char):
        if key == "ctrl-p":
            if self.scroll_source:
                self.scroll_end()
            self.mode = ""
        elif key == "ctrl-f":
            self.mode = ""
            self.open_picker()
        elif key == "ctrl-t":
            self.mode = ""
            self.new_tab()
        elif key == "ctrl-w":
            self.mode = ""
            self.sidebar = not self.sidebar
        elif key == "ctrl-r":
            self.mode = "RESIZE"
        elif key == "ctrl-g":
            self.mode = "GLOBAL"
        elif key == "esc":
            if self.scroll_source:
                self.scroll_end()
                return
            self.mode = ""
        elif key == "page-up":
            self.scroll_focused(10)
        elif key == "page-down":
            self.scroll_focused(-10)
        elif key == "tab":
            self.focus_next()
        elif key == "ctrl-e":
            self.restart_focused()
        elif len(key) == 1:
            if key == "y":
                self.copy_visible()
            elif key == ":":
                self.mode = ""
                self.overlay = "prompt"
                self.prompt = ""
                self.prompt_sel = 0
            elif key == "?":
                self.mode = ""
                self.overlay = "help"
            elif key == "%":
                self.split_pane("row")
            elif key == '"':
                self.split_pane("col")
            elif key == "x":
                self.close_pane()
            elif key.isdigit() and key != "0":
                index = ord(key) - ord("1")
                if 0 <= index < len(self.tabs):
                    self.active = index

    def handle_overlay_key(self, key, char):
        if self.overlay == "picker":
            if key == "up":
                self.picker = clamp(self.picker - 1, 0, max(0, len(self.picker_selectables()) - 1))
            elif key == "down":
                self.picker = clamp(self.picker + 1, 0, max(0, len(self.picker_selectables()) - 1))
            elif key == "enter":
                self.attach(self.picker)
            elif key == "esc":
                self.close_overlay()
        elif self.overlay == "help":
            if key == "esc" or key == "?":
                self.close_overlay()
        elif self.overlay == "manager":
            if key == "esc":
                self.close_overlay()
            elif key == "up":
                self.manager_pos = clamp(self.manager_pos - 1, 0, MANAGER_ITEMS - 1)
            elif key == "down":
                self.manager_pos = clamp(self.manager_pos + 1, 0, MANAGER_ITEMS - 1)
            elif key == "page-up":
                self.manager_pos = clamp(self.manager_pos - 10, 0, MANAGER_ITEMS - 1)
            elif key == "page-down":
                self.manager_pos = clamp(self.manager_pos + 10, 0, MANAGER_ITEMS - 1)
        elif self.overlay == "prompt":
            if key == "esc":
                self.close_overlay()
            elif key == "up":
                self.prompt_sel = clamp(self.prompt_sel - 1, 0, max(0, len(self.prompt_matches()) - 1))
            elif key == "down":
                self.prompt_sel = clamp(self.prompt_sel + 1, 0, max(0, len(self.prompt_matches()) - 1))
            elif key == "backspace":
                if self.prompt:
                    self.prompt = self.prompt[:-1]
                    self.prompt_sel = 0
            elif key == "enter":
                matches = self.prompt_matches()
                if 0 <= self.prompt_sel < len(matches):
                    self.run_command(matches[self.prompt_sel])
            elif len(key) == 1:
                self.prompt += char or key
                self.prompt_sel = 0

    def handle_mouse(self, event):
        action = event.get("action")
        node = event.get("node") or ""
        if action == "release":
            self.dragging = 0
            return
        if action == "drag":
            self.drag_resize(event.get("x", 0), event.get("y", 0))
            return
        if node.startswith("divider:"):
            self.dragging = atoi_node(node, "divider:")
            return
        if node == "tabcreate":
            self.new_tab()
        elif node.startswith("tabclose:"):
            self.close_tab(atoi_node(node, "tabclose:"))
        elif node.startswith("tab:"):
            index = atoi_node(node, "tab:")
            if 0 <= index < len(self.tabs):
                self.active = index
        elif node == "ws":
            self.open_picker()
        elif node.startswith("picker:"):
            index = atoi_node(node, "picker:")
            self.picker = index
            self.attach(index)
        elif node.startswith("prompt:"):
            index = atoi_node(node, "prompt:")
            self.prompt_sel = index
            matches = self.prompt_matches()
            if 0 <= index < len(matches):
                self.run_command(matches[index])
        elif node == "toast":
            self.toast = ""
        else:
            pane = self.pane_by_id(node)
            if pane is None:
                return
            for tab_index, tab in enumerate(self.tabs):
                for index, candidate in enumerate(tab.panes):
                    if candidate is pane:
                        self.active = tab_index
                        tab.focus = index
                        return

    def handle_wheel(self, event):
        node = event.get("node") or ""
        delta = event.get("delta", 0)
        if node == "manager":
            self.manager_pos = clamp(self.manager_pos - delta, 0, MANAGER_ITEMS - 1)
            return
        pane = self.pane_by_id(node)
        if pane is None:
            return
        source = self.source_by_id(pane.source_id) if pane.source_id else None
        if source is not None and not source.get("exited"):
            self.scroll_terminal(pane, source, delta)
        else:
            pane.scroll = clamp(pane.scroll - delta, 0, len(pane.lines))

    def handle_paste(self, text):
        pane = self.focus_pane()
        if pane is not None and not pane.source_id and text:
            pane.lines.append("> " + text)

    # ------------------------------------------------------------ actions

    def focus_next(self):
        tab = self.active_tab()
        if tab.panes:
            tab.focus = (tab.focus + 1) % len(tab.panes)

    def append_to_pane(self, char):
        pane = self.focus_pane()
        if pane is None or pane.source_id:
            return
        line = "> " + char
        if len(pane.lines) < 12:
            pane.lines.append(line)
        else:
            pane.lines = pane.lines[1:] + [line]

    def restart_focused(self):
        pane, source = self.focused_exited()
        if source is None or not source.get("terminal_id"):
            return
        params = {"endpoint": source.get("endpoint") or "local", "id": source["terminal_id"]}
        self.emit_result("terminal.restart", params)
        self.toast = "restart requested · " + source["terminal_id"]

    def copy_visible(self):
        pane, source = self.focused_terminal()
        if source is None or source.get("exited") or not source.get("terminal_id"):
            return
        pane_id = pane.id
        params = {"endpoint": source.get("endpoint") or "local", "id": source["terminal_id"]}

        def done(response):
            if response.get("ok"):
                self.toast = "copied visible screen"
            else:
                self.toast = "copy failed: " + (response.get("error") or "")
        self.emit_result("terminal.copy", params, done)

    def scroll_focused(self, delta):
        pane, source = self.focused_terminal()
        if source is None:
            return
        self.scroll_terminal(pane, source, delta)

    def scroll_terminal(self, pane, source, delta):
        if not source.get("terminal_id") or delta == 0:
            return
        params = {"endpoint": source.get("endpoint") or "local",
                  "id": source["terminal_id"], "delta": delta}
        self.emit_result("terminal.scroll", params)
        if self.scroll_source != pane.id:
            self.scroll_source = pane.id
            self.scroll_offset = 0
            pane.scroll = 0
        self.scroll_offset += delta
        pane.scroll = max(0, pane.scroll + delta)
        if self.scroll_offset <= 0:
            self.scroll_source = ""
            self.scroll_offset = 0
            pane.scroll = 0

    def scroll_end(self):
        if not self.scroll_source:
            return
        pane = self.pane_by_id(self.scroll_source)
        if pane is not None:
            pane.scroll = 0
            source = self.source_by_id(pane.source_id) if pane.source_id else None
            if source is not None and source.get("terminal_id"):
                self.emit_result("terminal.scrollEnd", {
                    "endpoint": source.get("endpoint") or "local",
                    "id": source["terminal_id"]})
        self.scroll_source = ""
        self.scroll_offset = 0

    def split_pane(self, direction):
        pane = self.focus_pane()
        if pane is None:
            return
        clone = self.new_pane(pane.title + " copy", ["split " + direction])
        tab = self.active_tab()
        tab.panes.append(clone)
        tab.weights.append(1)
        tab.focus = len(tab.panes) - 1
        tab.flow = "row" if direction == "row" else "col"

    def close_pane(self):
        tab = self.active_tab()
        if len(tab.panes) <= 1:
            return
        del tab.panes[tab.focus]
        del tab.weights[tab.focus]
        if tab.focus >= len(tab.panes):
            tab.focus = len(tab.panes) - 1

    def new_tab(self):
        self.tab_seq += 1
        pane = self.new_pane("unconnected", ["new tab"])
        self.tabs.append(Tab("tab%d" % self.tab_seq, [pane]))
        self.active = len(self.tabs) - 1

    def close_tab(self, index):
        if index < 0 or index >= len(self.tabs) or len(self.tabs) == 1:
            return
        del self.tabs[index]
        if self.active >= len(self.tabs):
            self.active = len(self.tabs) - 1

    def drag_resize(self, x, y):
        tab = self.active_tab()
        if self.dragging == 0 or len(tab.panes) != 2:
            return
        if tab.flow == "row":
            sidebar = SIDEBAR_WIDTH if self.sidebar else 0
            avail = self.cols - sidebar - 1
            if avail < 4:
                return
            left = clamp(x - sidebar, 2, avail - 2)
            tab.weights = [left, avail - left]
        else:
            avail = max(1, self.rows - 2) - 1
            if avail < 4:
                return
            top = clamp(y - 2, 2, avail - 2)
            tab.weights = [top, avail - top]

    def attach(self, index):
        entries = self.picker_selectables()
        if index < 0 or index >= len(entries):
            return
        entry = entries[index]
        pane = self.focus_pane()
        endpoint = entry.get("endpoint")
        if endpoint is not None:
            # Configured endpoint row: create through the endpoint (command or
            # daemon connect strategy).
            params = {
                "endpoint": endpoint.get("name") or "",
                "argv": list(endpoint.get("argv") or []),
                "cwd": endpoint.get("cwd") or "",
                "env": endpoint.get("env") or {},
            }
            if (endpoint.get("kind") or "command") == "daemon":
                params["kind"] = "daemon"
                params["socket"] = endpoint.get("socket") or ""
                params["address"] = endpoint.get("address") or ""
                params["connect_mode"] = endpoint.get("connect_mode") or "local-unix"
            self.bind_pending(pane.id, None, params)
            self.close_overlay()
            self.toast = "create requested · " + (endpoint.get("label") or endpoint.get("name") or "")
            return
        source = entry.get("source")
        if source is not None:
            terminal_id = source.get("terminal_id") or ""
            if terminal_id and source.get("exited"):
                self.emit_result("terminal.restart", {
                    "endpoint": source.get("endpoint") or "local", "id": terminal_id})
                self.close_overlay()
                self.toast = "restart requested · " + terminal_id
                return
            if terminal_id:
                self.bind_pending(pane.id, source.get("id"), {
                    "endpoint": source.get("endpoint") or "local", "id": terminal_id, "fit": True})
                self.close_overlay()
                self.toast = "attach requested · " + terminal_id
                return
        self.bind_pending(pane.id, None, {"endpoint": "local"})
        self.close_overlay()
        self.toast = "create requested"

    def bind_pending(self, pane_id, source_id, params):
        method = "terminal.attach" if source_id else "terminal.create"

        def done(response):
            if not response.get("ok"):
                self.toast = method + " failed: " + (response.get("error") or "")
                return
            if source_id:
                self.bind_pane(pane_id, source_id)
                self.toast = "bound " + short_source_id(source_id)
                return
            data = response.get("data") or {}
            endpoint = data.get("endpoint") or "local"
            terminal_id = data.get("id") or ""
            self.bind_pane(pane_id, "terminal:%s:%s" % (endpoint, terminal_id))
            self.toast = "bound " + terminal_id
        self.emit_result(method, params, done)

    def bind_pane(self, pane_id, source_id):
        pane = self.pane_by_id(pane_id)
        if pane is None:
            return
        pane.source_id = source_id
        pane.scroll = 0

    def run_command(self, command):
        self.close_overlay()
        self.toast = "ran: " + command
        if command == "split row":
            self.split_pane("row")
        elif command == "split col":
            self.split_pane("col")
        elif command == "close pane":
            self.close_pane()
        elif command == "kill pane":
            pane, source = self.bound_pane()
            if source is not None and source.get("terminal_id"):
                self.emit_result("terminal.kill", {
                    "endpoint": source.get("endpoint") or "local", "id": source["terminal_id"]})
        elif command == "new tab":
            self.new_tab()
        elif command == "close tab":
            self.close_tab(self.active)
        elif command == "toggle sidebar":
            self.sidebar = not self.sidebar
        elif command == "open picker":
            self.open_picker()
        elif command == "terminal manager":
            self.overlay = "manager"
            self.manager_pos = 0
        elif command == "create terminal":
            self.bind_pending(self.focus_pane().id, None, {"endpoint": "local"})
        elif command == "help":
            self.overlay = "help"
        elif command == "quit":
            self.emit_result("system.quit", {})

    # --------------------------------------------------------- rasterizer

    def screen(self):
        """Rasterize the current view with the tui2 kernel semantics (flow,
        intrinsic sizes, pos overlays, truncation) plus the legacy terminal
        component chrome. Returns (lines, styles): styles is a grid of style
        strings aligned with the characters ("" = default)."""
        grid = [[" " for _ in range(self.cols)] for _ in range(self.rows)]
        styles = [["" for _ in range(self.cols)] for _ in range(self.rows)]
        self.cursor = None
        self.draw_box(self.view(), (0, 0, self.cols, self.rows), grid, styles)
        return ["".join(row) for row in grid], styles

    def draw_box(self, box, rect, grid, styles):
        x, y, width, height = rect
        if width <= 0 or height <= 0:
            return
        content = box.get("content")
        if content:
            if content.get("self"):
                self.draw_component(box, rect, grid, styles)
            else:
                lines = content.get("lines") or []
                if not lines and content.get("text"):
                    lines = content["text"].split("\n")
                style = box.get("style", "")
                for index, line in enumerate(lines):
                    if index >= height:
                        break
                    self.put_text(grid, styles, x, y + index, width, truncate(line, width), style)
        cursor = box.get("cursor")
        if cursor is not None:
            self.cursor = (x + cursor.get("col", 0), y + cursor.get("row", 0))
        flow = box.get("flow") or "col"
        regular = []
        overlays = []
        for child in box.get("children") or []:
            if child.get("visible") is False:
                continue
            if child.get("pos") is not None:
                overlays.append(child)
            else:
                regular.append(child)
        for child, child_rect in zip(regular, assign_flow(flow, rect, regular)):
            self.draw_box(child, child_rect, grid, styles)
        for child in overlays:
            px, py = child["pos"]
            cw = (child.get("size") or (0, 0, 0))[0]
            ch = (child.get("size") or (0, 0, 0))[1]
            iw, ih = intrinsic_size(child)
            if cw <= 0:
                cw = iw if iw > 0 else width
            if ch <= 0:
                ch = ih if ih > 0 else height
            child_rect = (x + px, y + py, cw, ch)
            clear_rect(grid, styles, child_rect, self.cols, self.rows)
            self.draw_box(child, child_rect, grid, styles)

    def draw_component(self, box, rect, grid, styles):
        x, y, width, height = rect
        source = self.source_by_id(box.get("content", {}).get("self", ""))
        focused = bool(box.get("focused"))
        exited = bool(source and source.get("exited"))
        code = int(source.get("exit_code") or 0) if source else 0
        title = ""
        if source:
            title = (source.get("title") or "").strip() or (source.get("terminal_id") or "")
        if not title:
            title = "terminal"
        if exited:
            title += " [exited %d]" % code if code else " [exited]"
        pane = self.pane_by_id(box.get("id"))
        if pane is not None and pane.scroll > 0:
            title += " [↑%d]" % pane.scroll
        props = box.get("content", {}).get("props") or {}
        border = props.get(P_BORDER_DEAD) if exited else (
            props.get(P_BORDER_FOCUS) if focused else props.get(P_BORDER))
        border = border or STYLE_MUTED
        label_style = props.get(P_TITLE) or border
        # The legacy component only draws chrome when W >= 3 and H >= 3.
        if width < 3 or height < 3:
            return
        prefix = " " + (GLYPH_MARKER if focused and not exited else "")
        available = width - 2 - display_width(prefix) - 1
        label = ""
        if width >= 4 and available > 0:
            label = prefix + truncate(title, available) + " "
        self.put_text(grid, styles, x, y, 1, "┌", border)
        cursor_x = x + 1
        if label:
            self.put_text(grid, styles, cursor_x, y, width - 2, label, label_style)
            cursor_x += display_width(label)
        dashes = max(0, width - 2 - display_width(label))
        self.put_text(grid, styles, cursor_x, y, dashes + 1, "─" * dashes + "┐", border)
        for row in range(1, height - 1):
            self.put_text(grid, styles, x, y + row, 1, "│", border)
            self.put_text(grid, styles, x + width - 1, y + row, 1, "│", border)
        self.put_text(grid, styles, x, y + height - 1, width, bottom_border(width), border)

    def put_text(self, grid, styles, x, y, width, text, style):
        if y < 0 or y >= self.rows or width <= 0:
            return
        cursor = x
        remaining = width
        for ch in text:
            cells = cell_width(ch)
            if cells == 0:
                continue
            if cells > remaining or cursor >= self.cols:
                break
            if cursor >= 0:
                grid[y][cursor] = ch
                styles[y][cursor] = style
                for offset in range(1, cells):
                    if cursor + offset < self.cols:
                        grid[y][cursor + offset] = ""
                        styles[y][cursor + offset] = style
            cursor += cells
            remaining -= cells


def clear_rect(grid, styles, rect, cols, rows):
    """The kernel blits a pos subtree opaquely: blank its bounding rect first
    (mirrors compositor.blitOpaqueOverlay)."""
    x, y, width, height = rect
    for row in range(max(0, y), min(rows, y + height)):
        for col in range(max(0, x), min(cols, x + width)):
            grid[row][col] = " "
            styles[row][col] = ""


def intrinsic_size(box):
    content = box.get("content")
    if not content or content.get("self"):
        return 0, 0
    lines = content.get("lines") or []
    if not lines and content.get("text"):
        lines = content["text"].split("\n")
    if not lines:
        return 0, 0
    width = 0
    for line in lines:
        width = max(width, display_width(line))
    return width, len(lines)


def assign_flow(flow, rect, children):
    """tui2 kernel assignFlow: fixed sizes first, then flex, then stretch."""
    x, y, width, height = rect
    if flow == "stack":
        out = []
        for child in children:
            size = child.get("size") or (0, 0, 0)
            cw = size[0] if size[0] > 0 else width
            ch = size[1] if size[1] > 0 else height
            out.append((x, y, cw, ch))
        return out
    row = flow == "row"
    plans = []
    for child in children:
        size = child.get("size") or (0, 0, 0)
        iw, ih = intrinsic_size(child)
        main_spec = size[0] if row else size[1]
        main_intrinsic = iw if row else ih
        cross_spec = size[1] if row else size[0]
        cross_intrinsic = ih if row else iw
        if cross_spec > 0:
            cross = cross_spec
        elif cross_intrinsic > 0:
            cross = cross_intrinsic
        else:
            cross = height if row else width
        if main_spec > 0:
            plans.append({"fixed": main_spec, "flex": 0, "base": 0, "flexible": False, "cross": cross})
        elif size[2] > 0:
            plans.append({"fixed": 0, "flex": size[2], "base": main_intrinsic, "flexible": True, "cross": cross})
        elif main_intrinsic > 0:
            plans.append({"fixed": main_intrinsic, "flex": 0, "base": 0, "flexible": False, "cross": cross})
        else:
            plans.append({"fixed": 0, "flex": 0, "base": 0, "flexible": True, "cross": cross})
    main_total = width if row else height
    used = 0
    flex_sum = 0
    for plan in plans:
        if plan["flexible"]:
            used += plan["base"]
            flex_sum += plan["flex"]
        else:
            used += plan["fixed"]
    remaining = max(0, main_total - used)
    if flex_sum > 0:
        distributed = 0
        last = -1
        for index, plan in enumerate(plans):
            if not plan["flexible"] or plan["flex"] <= 0:
                continue
            extra = remaining * plan["flex"] // flex_sum
            plan["base"] += extra
            distributed += extra
            last = index
        if last >= 0:
            plans[last]["base"] += remaining - distributed
    else:
        stretch = [index for index, plan in enumerate(plans) if plan["flexible"]]
        if stretch:
            each = remaining // len(stretch)
            for index in stretch:
                plans[index]["base"] += each
            plans[stretch[-1]]["base"] += remaining - each * len(stretch)
    out = []
    position = y if not row else x
    for plan in plans:
        size = plan["base"] if plan["flexible"] else plan["fixed"]
        size = max(0, size)
        cross = max(0, plan["cross"])
        if row:
            out.append((position, y, size, cross))
        else:
            out.append((x, position, cross, size))
        position += size
    return out


def short_source_id(source_id):
    parts = source_id.split(":", 2)
    return parts[2] if len(parts) == 3 else source_id


def atoi_node(node, prefix):
    value = node[len(prefix):]
    if not value:
        return -1
    for ch in value:
        if ch < "0" or ch > "9":
            return -1
    return int(value)


# --------------------------------------------------------------- main loop

def run_stream(stdin, stdout):
    program = Program(stdout)
    while True:
        frame = pb.read_frame(stdin)
        if frame is None:
            return 0
        frame_type, payload = frame
        if frame_type == pb.HELLO:
            program.on_hello(pb.decode_hello(payload))
        elif frame_type == pb.EVENT:
            program.on_event(pb.decode_event(payload))
        elif frame_type == pb.RESPONSE:
            program.on_response(pb.decode_response(payload))


def recommended_footer_lines(cols=120, rows=32):
    """Canonical recommended-profile footer rows for every v2 scene.

    The lines are the shared golden's content
    (golden/recommended_footer_120x32.txt): the Go shell composes the same
    rows through its own spec table (clients/tui/cmd/tui2-shell/footer.go).
    """
    def new_program():
        program = Program(None, cols=cols, rows=rows, view_id="view:local:1")
        program.on_hello({"view_id": "view:local:1", "epoch": 1, "cols": cols, "rows": rows})
        return program

    def press(program, key):
        program.on_event({"kind": "key", "key": key, "char": key})

    out = {}

    program = new_program()
    program.mode = ""
    out["NORMAL"] = program.footer_text(program.active_tab())

    program = new_program()
    program.mode = ""
    press(program, "ctrl-p")
    press(program, "%")
    program.mode = ""
    out["NORMAL_2PANE"] = program.footer_text(program.active_tab())

    program = new_program()
    press(program, "ctrl-p")
    press(program, "%")
    out["PANE"] = program.footer_text(program.active_tab())

    program = new_program()
    program.overlay = "picker"
    out["PICKER"] = program.footer_text(program.active_tab())

    program = new_program()
    program.overlay = "prompt"
    out["PROMPT"] = program.footer_text(program.active_tab())

    program = new_program()
    program.overlay = "help"
    out["HELP"] = program.footer_text(program.active_tab())

    program = new_program()
    program.scroll_source = program.focus_pane().id
    out["SCROLL"] = program.footer_text(program.active_tab())

    program = new_program()
    exited = {"id": "terminal:local:term-1", "kind": "terminal", "title": "term-1",
              "endpoint": "local", "terminal_id": "term-1", "attached": True,
              "exited": True, "exit_code": 3}
    program.sources = [exited]
    program.focus_pane().source_id = exited["id"]
    program.mode = ""
    out["EXITED"] = program.footer_text(program.active_tab())

    return out


def main(argv):
    try:
        spec = load_preset(argv)
        if spec is not None:
            apply_preset(spec)
    except (OSError, ValueError) as error:
        sys.stderr.write("legacy: preset config: %s\n" % error)
        return 2
    if "--footer-lines" in argv:
        if not FOOTER_RECOMMENDED:
            sys.stderr.write("legacy: --footer-lines needs the recommended preset\n")
            return 2
        cols, rows = 120, 32
        if "--cols" in argv:
            cols = int(argv[argv.index("--cols") + 1])
        if "--rows" in argv:
            rows = int(argv[argv.index("--rows") + 1])
        for name, line in sorted(recommended_footer_lines(cols, rows).items()):
            sys.stdout.write("%s|%s|\n" % (name, line))
        return 0
    if "--selftest" in argv:
        cols, rows = 120, 32
        if "--cols" in argv:
            cols = int(argv[argv.index("--cols") + 1])
        if "--rows" in argv:
            rows = int(argv[argv.index("--rows") + 1])
        program = Program(None, cols=cols, rows=rows, view_id="selftest")
        program.on_sources([])
        lines, _styles = program.screen()
        sys.stdout.write("\n".join(lines) + "\n")
        return 0
    try:
        return run_stream(sys.stdin.buffer, sys.stdout.buffer)
    except (BrokenPipeError, KeyboardInterrupt):
        return 0
    except EOFError:
        return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
