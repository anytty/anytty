"""Program-side chrome widgets (stdlib only).

The kernel has no border/tab-bar/footer/picker concepts (PROTOCOL §9.6):
chrome is the program's job. This module is the generic toolkit behind the
reference v3 replica and is built from small named widgets:

  - ``Card``: a full pane card (frame + lock/title/state/owner/action chrome)
    via :meth:`ChromeApp.pane_runs` / :meth:`ChromeApp.card_nodes`;
  - ``TitleBar``/``TabBar``: the top chip strip (:meth:`ChromeApp.header_runs`);
  - ``Footer``/``StatusBar``: scene badge + key groups + right summary
    (:meth:`ChromeApp.footer_runs` / :meth:`ChromeApp.footer_line`);
  - ``SplitLayout``: the recursive split tree (:class:`Split`, :class:`Leaf`,
    :class:`Tab`, :meth:`ChromeApp.pane_rects`, :meth:`ChromeApp.split_pane`);
  - ``FloatingLayer``: floating windows (:class:`Floating`,
    :meth:`ChromeApp.floating_runs`);
  - ``Picker``: selectable overlay rows (:meth:`ChromeApp.overlay_rows`);
  - ``Toast``: transient one-line notice (:meth:`ChromeApp.toast_nodes`);
  - ``Button``/``KeyHint``: clickable runs and footer key text.

Everything is pure presentation over plain dicts: the widgets never emit
protocol methods, clicks/keys are handled by the program after hit-testing
box ids, exactly like every other box ("a button is a box plus program-side
hit handling", CUSTOMIZE §5).

:class:`ChromeApp` is the composition used by the v3 replica: it owns the
program-side state machine (modes, splits, floats, overlays), builds the view
tree and can rasterize it (``screen()``) for golden tests. A theme object
supplies every style string and glyph; see ``v3ui.py`` for the reference
coralline-candy theme.
"""

import unicodedata

from .core import App


class Theme:
    """Neutral default token set; programs override every field they use."""

    def __init__(self):
        CENTER = "o"
        CLOSE = "x"
        COLLAPSE = "v"
        COLLAPSE_HINT = "Click to collapse"
        EDGE_L = " "
        EDGE_R = " "
        EDGE_ROUND_R = " "
        FLOAT_SUMMARY = "f"
        GUTTER = "|"
        LOCKED = "\u25a0"
        RUNNING = "*"
        SPLIT_H = "-"
        SPLIT_V = "|"
        TAB_CLOSE = "x"
        TAB_CREATE = "+ "
        TAB_ICON = "\u2387"
        TERM_SUMMARY = "t"
        UNLOCKED = "\u25a1"
        WS_ICON = ""
        ZOOM = "[]"
        TABS_SUMMARY = "T"
        PANES_SUMMARY = "P"
        CHECK = "v"
        UNZOOM = "~"
        ST_ACCENT = ""
        ST_ACCENT_GROUP = ""
        ST_CONTENT = ""
        ST_CREATE_BODY = ""
        ST_CREATE_L = ""
        ST_CREATE_R = ""
        ST_DANGER = ""
        ST_FOOTER = ""
        ST_FOOTER_ACCENT = ""
        ST_FOOTER_FILL = ""
        ST_HEADER_FILL = ""
        ST_HINT = ""
        ST_INACTIVE_GROUP = ""
        ST_ITAB_BODY = ""
        ST_ITAB_CLOSE = ""
        ST_ITAB_L = ""
        ST_ITAB_R = ""
        ST_ITAB_SPACE = ""
        ST_MUTED = ""
        ST_OVERLAY = ""
        ST_PANEL_BORDER = ""
        ST_SUCCESS = ""
        ST_TAB_BODY = ""
        ST_TAB_CLOSE = ""
        ST_TAB_L = ""
        ST_TAB_R = ""
        ST_TAB_SPACE = ""
        ST_TOAST = ""
        ST_WARNING = ""
        ST_WS_BODY = ""
        ST_WS_L = ""
        ST_WS_R = ""
        MODE_ICONS = {}
        MODE_STYLES = {}
        SCENES = {}
        SHORTCUT_ACTION_STYLES = {}
        FOOTER_STYLE_TOKENS = {}
        PROMPT_COMMANDS = []
        HELP_LINES = []


# ------------------------------------------------------------ text metrics

def cell_width(ch):
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1


def display_width(text):
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


def pad_right(text, cells):
    width = display_width(text)
    if width >= cells:
        return text
    return text + " " * (cells - width)


def center_pad(text, cells):
    text = truncate(text, cells)
    pad = cells - display_width(text)
    left = pad // 2
    return " " * left + text + " " * (pad - left)


def clamp(value, low, high):
    if value < low:
        return low
    if value > high:
        return high
    return value


# --------------------------------------------------------------------- model

class Pane:
    __slots__ = ("id", "title", "source_id", "lines", "collapsed", "scroll",
                 "locked", "pending")

    def __init__(self, pane_id, title="", lines=None):
        self.id = pane_id
        self.title = title
        self.source_id = ""
        self.lines = list(lines or [])
        self.collapsed = False
        self.scroll = 0
        self.locked = False
        self.pending = ""


class Leaf:
    """One pane card: a leaf of the recursive split tree."""

    __slots__ = ("pane",)

    def __init__(self, pane):
        self.pane = pane


class Split:
    """A recursive split node.

    ``orient`` is "row" (a | b side by side) or "col" (a above b); ``ratio``
    is the share of the split extent (minus the 1-cell separator and the two
    card borders) given to ``a``.  A split only partitions the rect of the
    focused leaf it was created from; sibling splits are untouched.
    """

    __slots__ = ("orient", "ratio", "a", "b", "seq", "rect")

    def __init__(self, orient, ratio, a, b, seq):
        self.orient = orient
        self.ratio = ratio
        self.a = a
        self.b = b
        self.seq = seq
        self.rect = (0, 0, 0, 0)


def leaf_nodes(node):
    """In-order leaf list of a split subtree."""
    if node is None:
        return []
    if isinstance(node, Leaf):
        return [node]
    return leaf_nodes(node.a) + leaf_nodes(node.b)


def visible_leaf_count(node):
    if node is None:
        return 0
    if isinstance(node, Leaf):
        return 0 if node.pane.collapsed else 1
    return visible_leaf_count(node.a) + visible_leaf_count(node.b)


class Tab:
    __slots__ = ("id", "title", "panes", "root", "focus", "split_seq")

    def __init__(self, tab_id, title, panes=None, flow="row"):
        self.id = tab_id
        self.title = title
        self.split_seq = 0
        leaves = [pane if isinstance(pane, Leaf) else Leaf(pane)
                  for pane in (panes or [])]
        self.panes = [leaf.pane for leaf in leaves]
        self.focus = 0
        root = leaves[0] if leaves else None
        for leaf in leaves[1:]:
            self.split_seq += 1
            root = Split(flow, 0.5, root, leaf, self.split_seq)
        self.root = root

    def next_split_seq(self):
        self.split_seq += 1
        return self.split_seq

    def leaf_of(self, pane):
        for leaf in leaf_nodes(self.root):
            if leaf.pane is pane:
                return leaf
        return None

    def rebuild(self):
        self.panes = [leaf.pane for leaf in leaf_nodes(self.root)]


class Floating:
    __slots__ = ("id", "title", "x", "y", "w", "h", "collapsed", "pane")

    def __init__(self, float_id, title, x, y, w, h, pane):
        self.id = float_id
        self.title = title
        self.x = x
        self.y = y
        self.w = w
        self.h = h
        self.collapsed = False
        self.pane = pane


def footer_fallback_style(theme, key, label):
    """Old footerActionKeyStyle() heuristic on the raw key + label text."""
    upper = (key + " " + label).upper()
    if "X" in upper or "theme.CLOSE" in upper or "KILL" in upper:
        return theme.ST_FOOTER_KEY_PICKER
    if "W" in upper or "WORKSPACE" in upper:
        return theme.ST_FOOTER_KEY_WORKSPACE
    if "F" in upper or "PICK" in upper:
        return theme.ST_FOOTER_KEY_PICKER
    if "O" in upper or "FLOAT" in upper:
        return theme.ST_FOOTER_KEY_FLOAT
    if "V" in upper or "COPY" in upper:
        return theme.ST_FOOTER_KEY_COPY
    if "G" in upper or "GLOBAL" in upper:
        return theme.ST_FOOTER_KEY_GLOBAL
    if "R" in upper or "RESIZE" in upper or "SIZE" in upper:
        return theme.ST_FOOTER_KEY_RESIZE
    if "P" in upper or "PANE" in upper:
        return theme.ST_FOOTER_KEY_PANE
    if "T" in upper or "TAB" in upper or "TREE" in upper:
        return theme.ST_FOOTER_KEY_TAB
    return theme.ST_FOOTER_ACCENT


def footer_action_style(theme, action_id, key, label):
    """Resolution chain of the old app: yaml override -> scene default
    (StatusAccent / StatusWarning) -> footerActionDisplayStyle -> heuristic.

    Without a yaml override the old shortcutActionStyle() yields StatusAccent
    (StatusWarning for destructive ids), and footerActionDisplayStyle() sends
    both through footerActionKeyStyle() -- the heuristic below.
    """
    token = theme.SHORTCUT_ACTION_STYLES.get(action_id)
    if token is not None:
        return theme.FOOTER_STYLE_TOKENS[token]
    return footer_fallback_style(theme, key, label)


class ChromeApp(App):
    """Program-side chrome composition: state machine, view builder, raster."""

    def __init__(self, out=None, cols=120, rows=32,
                 view_id="", epoch=0, demo=False, theme=None):
        self.t = theme if theme is not None else Theme()
        self.client = None
        self.out = out
        self.view_id = view_id
        self.epoch = epoch
        self.rev = 0
        self.cols = cols
        self.rows = rows
        self.demo = demo
        self.host = out is not None
        self.workspace = "main"
        self.tab_seq = 1
        self.pane_seq = 0
        self.float_seq = 0
        self.tabs = [Tab("tab-1", "main", [self.new_pane("empty")])]
        self.active = 0
        self.mode = "live"
        self.overlay = ""
        self.picker = 0
        self.prompt = ""
        self.prompt_sel = 0
        self.help_sel = 0
        self.clipboard = []
        self.clip_sel = 0
        self.floatings = []
        self.active_float = ""
        self.toast = ""
        self.sources = []
        self.sources_ready = False
        self.terminal_count = 0
        self.scroll_source = ""
        self.scroll_offset = 0
        self.dragging = ""
        self.drag_last = (0, 0)
        self.request_id = 0
        self.pending = {}
        self.cursor = None
        if demo:
            self.load_demo()

    # ------------------------------------------------------------- state

    def new_pane(self, title, lines=None):
        self.pane_seq += 1
        return Pane("pane-%d" % self.pane_seq, title, lines)

    def source_by_id(self, source_id):
        for source in self.sources:
            if source.get("id") == source_id:
                return source
        return None

    def terminals(self):
        return [s for s in self.sources if s.get("kind") == "terminal" and s.get("id")]

    def active_tab(self):
        return self.tabs[self.active]

    def focus_pane(self):
        tab = self.active_tab()
        if not tab.panes:
            return None
        tab.focus = clamp(tab.focus, 0, len(tab.panes) - 1)
        return tab.panes[tab.focus]

    def pane_by_id(self, pane_id):
        for tab in self.tabs:
            for pane in tab.panes:
                if pane.id == pane_id:
                    return pane
        for floating in self.floatings:
            if floating.pane.id == pane_id:
                return floating.pane
        return None

    def tab_of_pane(self, pane):
        for index, tab in enumerate(self.tabs):
            if pane in tab.panes:
                return index, tab
        return -1, None

    def load_demo(self):
        """Program hook: preload a demo state (base: none)."""

    # ------------------------------------------------------- pane chrome

    def pane_source(self, pane):
        if not pane.source_id:
            return None
        return self.source_by_id(pane.source_id)

    def pane_title(self, pane):
        source = self.pane_source(pane)
        if source is None:
            return pane.title or pane.id
        terminal = (source.get("title") or source.get("terminal_id") or "").strip()
        if not terminal:
            terminal = "terminal"
        endpoint = (source.get("endpoint") or "local").strip() or "local"
        return truncate(terminal, 18) + "@" + endpoint

    def pane_state(self, pane, active):
        source = self.pane_source(pane)
        if source is not None and source.get("exited"):
            return "x", self.t.ST_DANGER
        if pane.pending:
            return "\u25cc", self.t.ST_WARNING
        if source is not None:
            return self.t.RUNNING, (self.t.ST_SUCCESS if active else self.t.ST_MUTED)
        return ("active" if active else "idle"), (self.t.ST_SUCCESS if active else self.t.ST_MUTED)

    def pane_owner(self, pane):
        source = self.pane_source(pane)
        if source is None:
            return "", self.t.ST_MUTED, ""
        if pane.pending == "owner":
            return "owner?", self.t.ST_WARNING, ""
        owner = (source.get("resize_owner") or "").strip()
        if owner:
            return "owner", self.t.ST_ACCENT, "pane:%s:take-owner" % pane.id
        return "follow", self.t.ST_MUTED, "pane:%s:take-owner" % pane.id

    def pane_runs(self, pane, active, width, collapsed=False):
        """One pane top-border row as (text, style, node, inputs) runs.

        Geometry mirrors tui/render: ``paneChromeTopSlots`` +
        ``paneChromeTerminalLabelSlots`` + ``paneChromeActionRendered...``.
        """
        frame = self.t.ST_ACCENT if active else self.t.ST_PANEL_BORDER
        runs = [("\u250c", frame, "", None), ("\u2500", frame, "", None)]
        if width < 4:
            return runs
        inner_right = width - 1
        terminal = self.pane_source(pane) is not None
        actions = []
        if terminal:
            full = [("zoom", self.t.ZOOM), ("split-v", self.t.SPLIT_V),
                    ("split-h", self.t.SPLIT_H), ("close", self.t.CLOSE)]
            if self.action_group_width(full) <= width - 6:
                actions = full
            elif self.action_group_width(full[-1:]) <= width - 5:
                actions = full[-1:]
        action_width = self.action_group_width(actions)
        action_x = inner_right - action_width - 1 if actions else inner_right
        right_limit = action_x - 1 if actions else inner_right

        if terminal:
            owner_text, owner_style, owner_node = self.pane_owner(pane)
            state_text, state_style = self.pane_state(pane, active)
            if collapsed:
                state_text = self.t.COLLAPSE
            right = [
                (center_pad(state_text, 3), state_style, "", None),
                (center_pad("x" + str(self.attach_count(pane)), 4), frame, "", None),
                (center_pad(owner_text, max(8, display_width(owner_text))),
                 owner_style, owner_node, ["mouse"] if owner_node else None),
            ]
        else:
            state_text, state_style = self.pane_state(pane, active)
            right = [(center_pad(" " + state_text + " ", 12), state_style, "", None)]
            owner_text = ""

        right_width = sum(display_width(run[0]) for run in right)
        left_width = max(0, right_limit - 2)
        prefix = " " + (self.t.LOCKED if pane.locked else self.t.UNLOCKED) + " "
        lock_node = "pane:%s:lock" % pane.id
        title = self.pane_title(pane)
        min_title = 3 if title else 0
        if display_width(prefix) + min_title + right_width > left_width:
            prefix = ""
        if display_width(prefix) + min_title + right_width > left_width:
            right = []
            right_width = 0
        title_width = max(0, left_width - display_width(prefix) - right_width)
        title_text = ""
        if title:
            if title_width > 2:
                title_text = " " + truncate(title, title_width - 2) + " "
            elif title_width > 0:
                title_text = truncate(title, title_width)

        # v3 order: prefix -> title (advance eats the gap) -> right slots ->
        # one rule cell (rightLimit) -> action group -> trailing rule/corner.
        x = 2
        if prefix:
            node = lock_node if terminal else ""
            runs.append((prefix, frame, node, ["mouse"] if node else None))
            x += display_width(prefix)
        gap = title_width - display_width(title_text)
        if title_text:
            runs.append((title_text, self.t.ST_ACCENT if active else self.t.ST_MUTED, "", None))
            x += display_width(title_text)
        if gap > 0:
            runs.append(("\u2500" * gap, frame, "", None))
            x += gap
        for text, text_style, node, inputs in right:
            runs.append((text, text_style, node, inputs))
            x += display_width(text)
        if actions:
            if x < action_x:
                runs.append(("\u2500" * (action_x - x), frame, "", None))
            for index, (action, glyph) in enumerate(actions):
                if index == 0:
                    runs.append((self.t.EDGE_L, frame, "", None))
                item_style = self.t.ST_ACCENT_GROUP if active else self.t.ST_INACTIVE_GROUP
                runs.append((" " + glyph + " ", item_style,
                             "pane:%s:%s" % (pane.id, action), ["mouse"]))
            runs.append((self.t.EDGE_ROUND_R, frame, "", None))
            x = action_x + action_width
        tail = width - x
        if tail > 1:
            runs.append(("\u2500" * (tail - 1), frame, "", None))
        runs.append(("\u2510", frame, "", None))
        return runs

    @staticmethod
    def action_group_width(actions):
        if not actions:
            return 0
        return 2 + 3 * len(actions)

    def attach_count(self, pane):
        source = self.pane_source(pane)
        if source is None:
            return 0
        count = int(source.get("attach_count") or 0)
        return max(0, count or 1)

    # -------------------------------------------------------------- header

    def add_run(self, out, x, y, text, text_style, node="", inputs=None, width=None):
        if width is None:
            width = display_width(text)
        if width <= 0:
            return
        box = text_box(node or "", text, text_style, width=width, height=1)
        if inputs:
            box["input"] = list(inputs)
        box["pos"] = (x, y)
        out.append(box)

    def header_runs(self):
        runs = [(self.t.EDGE_L, self.t.ST_WS_L, "", None),
                (" " + self.t.WS_ICON + " " + self.workspace + " ", self.t.ST_WS_BODY,
                 "hdr:workspace", ["mouse"]),
                (self.t.EDGE_R, self.t.ST_WS_R, "hdr:workspace", ["mouse"])]
        for index, tab in enumerate(self.tabs):
            active = index == self.active
            if active:
                edge_l, body, edge_r = self.t.ST_TAB_L, self.t.ST_TAB_BODY, self.t.ST_TAB_R
                space, close_style = self.t.ST_TAB_SPACE, self.t.ST_TAB_CLOSE
            else:
                edge_l, body, edge_r = self.t.ST_ITAB_L, self.t.ST_ITAB_BODY, self.t.ST_ITAB_R
                space, close_style = self.t.ST_ITAB_SPACE, self.t.ST_ITAB_CLOSE
            select = "hdr:tab:%d" % index
            close = "hdr:tabclose:%d" % index
            label = " " + self.t.TAB_ICON + " " + truncate(tab.title, 14) + " \u00b7 T" + str(index + 1) + " "
            runs.append((self.t.EDGE_R, edge_l, select, ["mouse"]))
            runs.append((label, body, select, ["mouse"]))
            runs.append((self.t.EDGE_R, edge_r, select, ["mouse"]))
            runs.append((" ", space, select, ["mouse"]))
            runs.append((self.t.TAB_CLOSE, close_style, close, ["mouse"]))
            runs.append((" ", space, select, ["mouse"]))
        runs.append((self.t.EDGE_R, self.t.ST_CREATE_L, "hdr:create", ["mouse"]))
        runs.append((" " + self.t.TAB_CREATE + " ", self.t.ST_CREATE_BODY, "hdr:create", ["mouse"]))
        runs.append((self.t.EDGE_ROUND_R, self.t.ST_CREATE_R, "hdr:create", ["mouse"]))
        return runs

    def header_nodes(self, out):
        x = 0
        for text, text_style, node, inputs in self.header_runs():
            width = display_width(text)
            if x >= self.cols:
                break
            if x + width > self.cols:
                text = truncate(text, self.cols - x)
                width = display_width(text)
            self.add_run(out, x, 0, text, text_style, node, inputs)
            x += width
        if x < self.cols:
            self.add_run(out, x, 0, "", self.t.ST_HEADER_FILL, "", None,
                         width=self.cols - x)

    # -------------------------------------------------------------- footer

    def scene(self):
        if self.overlay == "picker":
            return "terminal-picker"
        if self.overlay == "help":
            return "help"
        if self.overlay == "prompt":
            return "prompt"
        if self.overlay == "clipboard":
            return "copy"
        if self.mode in ("pane", "resize", "tab", "workspace", "system", "floating"):
            return self.mode
        if self.scroll_source:
            return "copy"
        return "live"

    def footer_right_runs(self):
        # Go appends a final muted space segment after the summary tokens.
        runs = [(" " + self.t.WS_ICON + " " + self.workspace, self.t.ST_FOOTER, "", 2),
                (" " + self.t.FLOAT_SUMMARY + " " + str(len(self.floatings)),
                 self.t.ST_FOOTER_ACCENT, "", 1),
                (" " + self.t.TERM_SUMMARY + " " + str(self.terminal_count),
                 self.t.ST_FOOTER, "", 4),
                (" ", self.t.ST_FOOTER, "", 4)]
        return runs

    def select_actions(self, actions, limit):
        selected = []
        used = 0
        truncated = False
        for action in actions:
            token_width = 1 + display_width(action[0])
            if selected:
                token_width += 3
            if selected and used + token_width > limit:
                truncated = True
                break
            if not selected and token_width > limit:
                truncated = True
                break
            selected.append(action)
            used += token_width
        if not truncated or len(selected) == len(actions):
            return selected
        tail = actions[-1]
        tail_width = 1 + display_width(tail[0]) + (3 if selected else 0)
        while selected and used + tail_width > limit:
            dropped = selected.pop()
            used -= 1 + display_width(dropped[0])
            if selected:
                used -= 3
            tail_width = 1 + display_width(tail[0]) + (3 if selected else 0)
        if tail_width <= limit and used + tail_width <= limit:
            selected.append(tail)
        return selected

    def action_style(self, action):
        label, _node, action_id, key = action
        return footer_action_style(self.t, action_id, key, label)

    def footer_runs(self):
        scene = self.scene()
        icon, label, actions = self.t.SCENES.get(scene, ("", "", []))
        runs = []
        if icon or label:
            badge = " " + icon + " " + label + " " if label else " " + icon + " "
            token = self.t.MODE_STYLES.get(scene, "footer-accent")
            runs.append((badge, self.t.FOOTER_STYLE_TOKENS[token], "", 1))
        right = list(self.footer_right_runs())
        reserve = 0
        for text, _style, _node, _priority in right:
            reserve += display_width(text)
        limit = self.cols
        if self.cols >= 120 and scene == "live":
            limit = max(0, self.cols - reserve)
        selected = self.select_actions(actions, limit)
        for action in selected:
            if runs:
                runs.append((" \u00b7 ", self.t.ST_FOOTER, "", 1))
            runs.append((" " + action[0], self.action_style(action),
                         action[1], 1))
        return runs, right

    @staticmethod
    def trim_runs(runs, width):
        """shell_bar.trimBarSegments: drop the highest-priority-number run."""
        out = list(runs)
        while sum(display_width(run[0]) for run in out) > width and out:
            index = 0
            for i, run in enumerate(out):
                if run[3] >= out[index][3]:
                    index = i
            del out[index]
        return out

    def footer_nodes(self, out):
        y = self.rows - 1
        runs, right = self.footer_runs()
        runs = self.trim_runs(runs, self.cols)
        left_width = sum(display_width(run[0]) for run in runs)
        right = self.trim_runs(right, self.cols - left_width)
        right_width = sum(display_width(run[0]) for run in right)
        pad = max(0, self.cols - left_width - right_width)
        x = 0
        for text, text_style, node, _priority in runs:
            width = display_width(text)
            if width <= 0:
                continue
            self.add_run(out, x, y, text, text_style, node,
                         ["mouse"] if node else None)
            x += width
        if pad > 0:
            self.add_run(out, x, y, "", self.t.ST_FOOTER_FILL, "", None, width=pad)
        x += pad
        for text, text_style, node, _priority in right:
            width = display_width(text)
            if width <= 0:
                continue
            self.add_run(out, x, y, text, text_style, "", None)
            x += width

    def footer_line(self):
        runs, right = self.footer_runs()
        runs = self.trim_runs(runs, self.cols)
        left_width = sum(display_width(run[0]) for run in runs)
        right = self.trim_runs(right, self.cols - left_width)
        right_width = sum(display_width(run[0]) for run in right)
        pad = max(0, self.cols - left_width - right_width)
        left = "".join(run[0] for run in runs)
        right_text = "".join(run[0] for run in right)
        return left + " " * pad + right_text

    # ------------------------------------------------------------ floating

    def floating_runs(self, floating, active, width):
        frame = self.t.ST_ACCENT if active else self.t.ST_PANEL_BORDER
        actions = [("center", self.t.CENTER), ("collapse", self.t.COLLAPSE),
                   ("zoom", self.t.ZOOM), ("close", self.t.CLOSE)]
        action_width = self.action_group_width(actions)
        if action_width > width - 6:
            actions = actions[-2:]
            action_width = self.action_group_width(actions)
        if action_width > width - 5:
            actions = []
            action_width = 0
        inner_right = width - 1
        action_x = inner_right - action_width - 1 if actions else inner_right
        right_limit = action_x - 1 if actions else inner_right
        lock = " " + (self.t.LOCKED if floating.pane.locked else self.t.UNLOCKED) + " "
        pane_title = self.pane_title(floating.pane)
        title = truncate(pane_title or floating.title or floating.pane.title,
                         max(0, right_limit - 2 - display_width(lock)))
        title_text = (" " + title + " ") if title else ""
        runs = [("\u250c", frame, "", None), ("\u2500", frame, "", None),
                (lock, frame, "float:%s:lock" % floating.id, ["mouse"]),
                (title_text, frame, "float:%s:title" % floating.id, ["mouse"])]
        drag_node = "float:%s:title" % floating.id
        x = 2 + display_width(lock) + display_width(title_text)
        if x < right_limit:
            runs.append(("\u2500" * (right_limit - x), frame, drag_node,
                         ["mouse"]))
            x = right_limit
        for index, (action, glyph) in enumerate(actions):
            if x < action_x:
                runs.append(("\u2500" * (action_x - x), frame, "", None))
                x = action_x
            if index == 0:
                runs.append((self.t.EDGE_L, frame, "", None))
                x += 1
            item_style = self.t.ST_ACCENT_GROUP if active else self.t.ST_INACTIVE_GROUP
            runs.append((" " + glyph + " ", item_style,
                         "float:%s:%s" % (floating.id, action), ["mouse"]))
            x += 3
        if actions:
            runs.append((self.t.EDGE_ROUND_R, frame, "", None))
            x += 1
        tail = width - x
        if tail > 1:
            runs.append(("\u2500" * (tail - 1), frame, "", None))
        runs.append(("\u2510", frame, "", None))
        return runs

    def floating_nodes(self, out, floating):
        active = self.active_float == floating.id
        focused = active and self.mode == "live" and not floating.collapsed
        frame = self.t.ST_ACCENT if active else self.t.ST_PANEL_BORDER
        x, y, w, h = floating.x, floating.y, floating.w, floating.h
        rx = x
        for text, text_style, node, inputs in self.floating_runs(floating, active, w):
            width = display_width(text)
            if rx >= self.cols:
                break
            if rx + width > self.cols:
                text = truncate(text, self.cols - rx)
                width = display_width(text)
            self.add_run(out, rx, y, text, text_style, node, inputs)
            rx += width
        if floating.collapsed:
            return
        for row in range(1, h - 1):
            self.add_run(out, x, y + row, "\u2502", frame, "", None)
            self.add_run(out, x + w - 1, y + row, "\u2502", frame, "", None)
        self.add_run(out, x, y + h - 1,
                     "\u2514" + "\u2500" * max(0, w - 2) + "\u2518", frame, "", None)
        inner_w, inner_h = max(0, w - 2), max(0, h - 2)
        source = self.pane_source(floating.pane)
        if source is not None and not self.demo and self.host:
            box = self_box(floating.pane.id, floating.pane.source_id,
                           {"chrome.inset": "0"}, inner_w, inner_h,
                           focused,
                           ["key", "paste", "wheel"])
            box["pos"] = (x + 1, y + 1)
            out.append(box)
        else:
            lines = floating.pane.lines or [""]
            for index in range(inner_h):
                text = truncate(lines[index] if index < len(lines) else "", inner_w)
                self.add_run(out, x + 1, y + 1 + index, text, self.t.ST_OVERLAY,
                             floating.pane.id, ["mouse"], width=inner_w)
            if source is None and not floating.pane.lines:
                self.add_hint(out, floating.pane, x, y + 1, w, h - 1)

    # ------------------------------------------------------------- overlay

    def center_rect(self, width, height):
        width = min(width, self.cols)
        height = min(height, self.rows)
        x = max(0, (self.cols - width) // 2)
        y = max(0, (self.rows - height) // 2)
        return width, height, x, y

    def overlay_rows(self):
        if self.overlay == "picker":
            rows = []
            for source in self.terminals():
                label = (source.get("title") or source.get("terminal_id") or "").strip()
                endpoint = source.get("endpoint") or "local"
                glyph = "\u00d7" if source.get("exited") else (
                    self.t.RUNNING if source.get("attached") else "\u25cb")
                rows.append((glyph + " " + endpoint + "  " + label, True))
            rows.append(("+ New terminal", True))
            return rows
        if self.overlay == "prompt":
            matches = self.prompt_matches()
            rows = [(": " + self.prompt, False, "cursor")]
            for command in matches:
                rows.append((command, True))
            return rows
        if self.overlay == "help":
            return [(line, False) for line in self.t.HELP_LINES]
        if self.overlay == "clipboard":
            return [("[%d] %s" % (i, item), True) for i, item in enumerate(self.clipboard)]
        return []

    def overlay_nodes(self, out):
        if self.overlay == "picker":
            title, width, min_height = "Terminals", 64, 8
        elif self.overlay == "prompt":
            title, width, min_height = "Command", 56, 6
        elif self.overlay == "help":
            title, width, min_height = "Help", 62, 8
        elif self.overlay == "clipboard":
            title, width, min_height = "Clipboard", 60, 8
        else:
            return
        rows = self.overlay_rows()
        height = min(self.rows, max(min_height, len(rows) + 4))
        width, height, x, y = self.center_rect(width, height)
        frame = self.t.ST_ACCENT
        inner_w = width - 2
        title_text = " " + title + " "
        top = "\u250c" + title_text + "\u2500" * max(0, width - 2 - display_width(title_text)) + "\u2510"
        self.add_run(out, x, y, top, frame, "", None)
        for row in range(1, height - 1):
            self.add_run(out, x, y + row, "\u2502", frame, "", None)
            self.add_run(out, x + width - 1, y + row, "\u2502", frame, "", None)
        self.add_run(out, x, y + height - 1,
                     "\u2514" + "\u2500" * max(0, width - 2) + "\u2518", frame, "", None)
        selectable = 0
        for index, row in enumerate(rows):
            if index >= height - 2:
                break
            text = row[0]
            if self.overlay == "picker":
                selectable_row = row[1]
            elif self.overlay == "prompt":
                selectable_row = row[1]
            else:
                selectable_row = row[1]
            selected = False
            marker = "  "
            row_style = self.t.ST_OVERLAY
            if self.overlay == "picker" and selectable_row:
                selected = selectable == self.picker
                selectable += 1
            elif self.overlay == "prompt" and selectable_row:
                selected = (selectable == self.prompt_sel)
                selectable += 1
            elif self.overlay == "clipboard" and selectable_row:
                selected = selectable == self.clip_sel
                selectable += 1
            if selected:
                marker = "\u25b8 "
                row_style = self.t.ST_ACCENT
            node = ""
            if self.overlay == "picker" and selectable_row:
                node = "picker:%d" % (selectable - 1)
            elif self.overlay == "prompt" and selectable_row:
                node = "prompt:%d" % (selectable - 1)
            self.add_run(out, x + 1, y + 1 + index, truncate(marker + text, inner_w),
                         row_style, node, ["mouse"] if node else None, width=inner_w)
        # Cursor for the prompt input row.
        if self.overlay == "prompt":
            self.cursor = (x + 1 + display_width(": " + self.prompt), y + 1)

    # ------------------------------------------------------------ toasts

    def toast_nodes(self, out):
        if not self.toast:
            return
        text = " " + self.toast + "  \u00b7  Ctrl-Q quit "
        width = min(self.cols, display_width(text))
        x = max(1, self.cols - width - 1)
        y = max(0, self.rows - 2)
        self.add_run(out, x, y, truncate(text, width), self.t.ST_TOAST, "toast",
                     ["mouse"], width=width)

    # --------------------------------------------------------------- view

    def pane_rects(self, tab):
        """Recursive split-tree layout: every visible leaf is a full card.

        A Split gives child ``a`` ``ratio`` of the available extent (the rect
        minus the 1-cell separator and both card borders); its separator is
        the draggable Divider cell between the two child rects.  Splits nest
        arbitrarily: a split only partitions its own rect, so splitting the
        focused leaf into "left 1 / right 2" keeps the left card intact.
        """
        body = (0, 1, self.cols, max(0, self.rows - 2))
        entries = []
        if tab.root is not None:
            self.layout_node(tab.root, body, entries)
        return body, entries

    def layout_node(self, node, rect, entries):
        if isinstance(node, Leaf):
            if not node.pane.collapsed:
                x, y, w, h = rect
                entries.append((node.pane, x, y, w, h))
            return
        visible_a = visible_leaf_count(node.a)
        visible_b = visible_leaf_count(node.b)
        if visible_a == 0 and visible_b == 0:
            return
        if visible_a == 0:
            self.layout_node(node.b, rect, entries)
            return
        if visible_b == 0:
            self.layout_node(node.a, rect, entries)
            return
        node.rect = rect
        x, y, w, h = rect
        if node.orient == "row":
            avail = max(2, w - 5)
            first = clamp(int(avail * node.ratio + 0.5), 1, avail - 1)
            a_rect = (x, y, first + 2, h)
            divider = (x + first + 2, y, 1, h)
            b_rect = (x + first + 3, y, w - first - 3, h)
        else:
            avail = max(2, h - 5)
            first = clamp(int(avail * node.ratio + 0.5), 1, avail - 1)
            a_rect = (x, y, w, first + 2)
            divider = (x, y + first + 2, w, 1)
            b_rect = (x, y + first + 3, w, h - first - 3)
        self.layout_node(node.a, a_rect, entries)
        entries.append((node, divider[0], divider[1], divider[2], divider[3]))
        self.layout_node(node.b, b_rect, entries)

    def split_entries(self, tab):
        """Every visible Split with its current rect (layout order)."""
        _, entries = self.pane_rects(tab)
        return [(node, x, y, w, h) for node, x, y, w, h in entries
                if isinstance(node, Split)]

    def view(self):
        tab = self.active_tab()
        out = []
        self.header_nodes(out)
        _body, entries = self.pane_rects(tab)
        focused = self.focus_pane()
        pane_focus = self.mode == "live" and not self.active_float
        visible = [entry for entry in entries
                   if not isinstance(entry[0], Split)]
        if not visible and focused is not None:
            # Every leaf collapsed: only the focused card's title row stays.
            self.card_nodes(out, focused, 0, 1, self.cols, 1,
                            True, pane_focus, True)
        else:
            for entry in entries:
                if isinstance(entry[0], Split):
                    split, dx, dy, dw, dh = entry
                    self.divider_nodes(out, split, dx, dy, dw, dh)
                    continue
                pane, px, py, pw, ph = entry
                self.card_nodes(out, pane, px, py, pw, ph,
                                pane is focused,
                                pane_focus and pane is focused,
                                pane.collapsed)
        for floating in self.floatings:
            self.floating_nodes(out, floating)
        if self.overlay:
            self.overlay_nodes(out)
        self.toast_nodes(out)
        self.footer_nodes(out)
        return flow_box("root", "stack", out)

    def card_nodes(self, out, pane, x, y, w, h, active=True, content_focus=None,
                   collapsed=False):
        """One leaf card: full border + pane title/action chrome + content.

        The focused card keeps its accent frame/title in every mode (old
        card_nodes hardcoded active=True); ``content_focus`` only controls
        the ``focused`` flag of the embedded terminal box.
        """
        if content_focus is None:
            content_focus = active
        runs = self.pane_runs(pane, active, w, collapsed)
        rx = x
        for text, text_style, node, inputs in runs:
            width = display_width(text)
            if rx >= self.cols:
                break
            if rx + width > self.cols:
                text = truncate(text, self.cols - rx)
                width = display_width(text)
            self.add_run(out, rx, y, text, text_style, node, inputs)
            rx += width
        if collapsed or h <= 0 or w <= 0:
            return
        frame = self.t.ST_ACCENT if active else self.t.ST_PANEL_BORDER
        for row in range(1, h - 1):
            self.add_run(out, x, y + row, "\u2502", frame, "", None)
            self.add_run(out, x + w - 1, y + row, "\u2502", frame, "", None)
        self.add_run(out, x, y + h - 1,
                     "\u2514" + "\u2500" * max(0, w - 2) + "\u2518", frame, "", None)
        self.sub_pane_nodes(out, pane, x + 1, y + 1,
                            max(0, w - 2), max(0, h - 2), content_focus)

    def divider_nodes(self, out, split, x, y, w, h):
        node = "divider:%s:%d" % (self.active_tab().id, split.seq)
        if split.orient == "row":
            for row in range(h):
                self.add_run(out, x, y + row, "\u2502", self.t.ST_PANEL_BORDER,
                             node, ["mouse"])
            return
        self.add_run(out, x, y, "\u2500" * max(0, w), self.t.ST_PANEL_BORDER,
                     node, ["mouse"], width=max(0, w))

    def sub_pane_nodes(self, out, pane, x, y, w, h, active):
        source = self.pane_source(pane)
        if source is not None and not self.demo and self.host:
            box = self_box(pane.id, pane.source_id,
                           {"chrome.inset": "0"}, w, h,
                           active,
                           ["key", "paste", "wheel"])
            box["pos"] = (x, y)
            out.append(box)
            return
        lines = pane.lines or [""]
        for index in range(h):
            text = truncate(lines[index] if index < len(lines) else "", w)
            self.add_run(out, x, y + index, text, self.t.ST_CONTENT,
                         "pane:%s:focus" % pane.id, ["mouse"], width=w)
        if source is None and not pane.lines:
            self.add_hint(out, pane, x - 1, y - 1, w + 2, h + 2)

    def add_hint(self, out, pane, x, y, w, h):
        if h < 2 or w < 8:
            return
        # Exact-width boxes: they stay smaller than the pane content box so
        # the hit test prefers the hint over the focus target.  The rendered
        # hint is two lines (gutter + label); both lines carry the collapse
        # target so clicking either of them toggles the pane.
        node = "pane:%s:hint" % pane.id
        self.add_run(out, x + 1, y + 1, "  " + self.t.GUTTER, self.t.ST_HINT, node, ["mouse"])
        self.add_run(out, x + 1, y + 2, "  " + self.t.GUTTER + "  " + self.t.COLLAPSE_HINT,
                     self.t.ST_HINT, node, ["mouse"])

    # --------------------------------------------------------- rasterizer

    def screen(self):
        grid = [[" " for _ in range(self.cols)] for _ in range(self.rows)]
        styles = [["" for _ in range(self.cols)] for _ in range(self.rows)]
        self.cursor = None
        for box in self.view().get("children") or []:
            if box.get("visible") is False:
                continue
            px, py = box.get("pos") or (0, 0)
            size = box.get("size") or (0, 0, 0)
            width = size[0] if size[0] > 0 else self.cols
            height = size[1] if size[1] > 0 else 1
            clear_rect(grid, styles, (px, py, width, height), self.cols, self.rows)
            self.draw_box(box, (px, py, width, height), grid, styles)
        return ["".join(row) for row in grid], styles

    def draw_box(self, box, rect, grid, styles):
        x, y, width, height = rect
        if width <= 0 or height <= 0:
            return
        content = box.get("content")
        if content:
            if content.get("self"):
                lines = self.self_lines(box, content.get("self") or "")
                for index in range(height):
                    text = truncate(lines[index] if index < len(lines) else "", width)
                    put_text(grid, styles, x, y + index, width, text, self.t.ST_CONTENT, self)
            else:
                lines = content.get("lines") or []
                if not lines and content.get("text"):
                    lines = content["text"].split("\n")
                text_style = box.get("style", "")
                for index, line in enumerate(lines):
                    if index >= height:
                        break
                    put_text(grid, styles, x, y + index, width,
                             truncate(line, width), text_style, self)
        cursor = box.get("cursor")
        if cursor is not None:
            self.cursor = (x + cursor.get("col", 0), y + cursor.get("row", 0))
        for child in box.get("children") or []:
            if child.get("visible") is False:
                continue
            child_pos = child.get("pos")
            if child_pos is None:
                continue
            px, py = child_pos
            size = child.get("size") or (0, 0, 0)
            cw = size[0] if size[0] > 0 else width
            ch = size[1] if size[1] > 0 else height
            clear_rect(grid, styles, (x + px, y + py, cw, ch), self.cols, self.rows)
            self.draw_box(child, (x + px, y + py, cw, ch), grid, styles)

    def self_lines(self, box, source_id):
        if self.demo:
            pane = self.pane_by_id(box.get("id"))
            if pane is not None and pane.lines:
                return pane.lines
        return []

    # ------------------------------------------------------------- claims

    def claim(self):
        if self.overlay:
            return [], True
        base = ["ctrl-p", "ctrl-r", "ctrl-o", "ctrl-t", "ctrl-w", "ctrl-f",
                "ctrl-shift-c", "ctrl-shift-h", "ctrl-shift-v", "ctrl-g"]
        base += ["ctrl-alt-%d" % n for n in range(1, 6)]
        if self.mode == "pane":
            base += ["x", "%", '"', "q", "h", "l", "d", "r", "z", "a", "s",
                     "t", "k", "b", "c", "p", "up", "down", "left", "right",
                     "ctrl-d", "ctrl-e", "esc"]
        elif self.mode == "resize":
            base += ["h", "l", "k", "j", "r", "space", "=", "a", "s", "esc",
                     "left", "right", "up", "down"]
        elif self.mode == "tab":
            base += ["c", "n", "p", "x", "r", "k", "esc"] + [str(n) for n in range(1, 10)]
        elif self.mode == "workspace":
            base += ["c", "n", "p", "r", "x", "t", "esc"]
        elif self.mode == "system":
            base += ["h", "f", "c", "p", "e", "m", "t", "w", "l", "o", "?",
                     "q", "x", "esc"]
        elif self.mode == "floating":
            base += ["n", "o", "f", "a", "x", "z", "m", "c", "v", "=", "s",
                     "h", "j", "k", "l", "left", "right", "up", "down", "esc"]
            base += [str(n) for n in range(1, 10)]
        return base, False

    # ------------------------------------------------------------- input

    def handle_key(self, key, char):
        if self.toast:
            self.toast = ""
        if self.overlay:
            self.handle_overlay_key(key, char)
            return
        if self.mode == "pane":
            self.handle_pane_key(key, char)
            return
        if self.mode == "resize":
            self.handle_resize_key(key, char)
            return
        if self.mode == "tab":
            self.handle_tab_key(key, char)
            return
        if self.mode == "workspace":
            self.handle_workspace_key(key, char)
            return
        if self.mode == "system":
            self.handle_system_key(key, char)
            return
        if self.mode == "floating":
            self.handle_floating_key(key, char)
            return
        self.handle_live_key(key, char)

    def handle_live_key(self, key, char):
        if key in ("ctrl-p",):
            self.mode = "pane"
        elif key == "ctrl-r":
            self.mode = "resize"
        elif key == "ctrl-o":
            self.mode = "floating"
            self.open_float_menu()
        elif key == "ctrl-t":
            self.mode = "tab"
        elif key == "ctrl-w":
            self.mode = "workspace"
        elif key == "ctrl-f":
            self.open_picker()
        elif key == "ctrl-g":
            self.mode = "system"
        elif key == "ctrl-shift-c":
            self.enter_copy()
        elif key == "ctrl-shift-h":
            self.overlay = "clipboard"
            self.clip_sel = 0
        elif key == "ctrl-shift-v":
            self.paste_system()
        elif key == "ctrl-q":
            self.emit_result("system.quit", {})
        elif key.startswith("ctrl-alt-"):
            digits = key[len("ctrl-alt-"):]
            if digits.isdigit():
                index = int(digits) - 1
                if 0 <= index < len(self.tabs):
                    self.active = index
                    self.mode = "live"
        elif key == "esc":
            self.mode = "live"

    def handle_pane_key(self, key, char):
        tab = self.active_tab()
        pane = self.focus_pane()
        if key in ("ctrl-p", "esc"):
            self.mode = "live"
        elif key == "ctrl-f":
            self.mode = "live"
            self.open_picker()
        elif key == "ctrl-t":
            self.mode = "live"
            self.new_tab()
        elif key == "ctrl-o":
            self.mode = "floating"
            self.open_float_menu()
        elif key == "ctrl-g":
            self.mode = "system"
        elif key == "x":
            self.close_pane(tab, pane)
            self.mode = "live"
        elif key == "%":
            self.split_pane("row")
        elif key == '"':
            self.split_pane("col")
        elif key == "ctrl-d":
            self.split_pane("row")
        elif key == "ctrl-e":
            self.split_pane("col")
        elif key == "q":
            self.kill_close_pane(tab, pane)
            self.mode = "live"
        elif key == "k":
            self.kill_pane(pane)
        elif key == "t":
            self.restart_pane(pane)
        elif key == "a":
            self.take_owner(pane)
        elif key == "s":
            pane.locked = not pane.locked
        elif key == "z":
            pane.collapsed = not pane.collapsed
        elif key == "h" or key in ("left", "up"):
            self.focus_pane_delta(-1)
        elif key == "l" or key in ("right", "down"):
            self.focus_pane_delta(1)
        elif len(key) == 1 and key.isdigit() and key != "0":
            index = ord(key) - ord("1")
            if 0 <= index < len(self.tabs):
                self.active = index
                self.mode = "live"
        elif key == ":":
            self.mode = "live"
            self.open_prompt()
        elif key == "?":
            self.mode = "live"
            self.overlay = "help"

    def handle_resize_key(self, key, char):
        tab = self.active_tab()
        if key in ("ctrl-p", "esc"):
            self.mode = "live"
        elif key == "ctrl-g":
            self.mode = "system"
        elif key == ":":
            self.mode = "live"
            self.open_prompt()
        elif key == "h":
            self.resize_focused(-2)
        elif key == "l":
            self.resize_focused(2)
        elif key == "k":
            self.resize_focused(-2, vertical=True)
        elif key == "j":
            self.resize_focused(2, vertical=True)
        elif key == "r":
            self.reset_tab_splits(tab)
        elif key == "s":
            pane = self.focus_pane()
            if pane is not None:
                pane.locked = not pane.locked
        elif key == "=":
            self.reset_tab_splits(tab)
        elif key == "space":
            self.mode = "pane"
        elif key == "ctrl-shift-c":
            self.enter_copy()

    def handle_tab_key(self, key, char):
        if key in ("ctrl-t", "esc"):
            self.mode = "live"
        elif key == "c":
            self.new_tab()
            self.mode = "live"
        elif key in ("n", "l", "]"):
            self.active = (self.active + 1) % len(self.tabs)
        elif key in ("p", "h", "["):
            self.active = (self.active - 1) % len(self.tabs)
        elif key == "x":
            self.close_tab(self.active)
        elif len(key) == 1 and key.isdigit() and key != "0":
            index = ord(key) - ord("1")
            if 0 <= index < len(self.tabs):
                self.active = index
                self.mode = "live"

    def handle_workspace_key(self, key, char):
        if key in ("ctrl-w", "esc"):
            self.mode = "live"
        elif key == "c":
            self.workspace = "ws-" + str(len(self.tabs) + 1)
        elif key in ("n", "l", "]"):
            self.active = (self.active + 1) % len(self.tabs)
        elif key in ("p", "h", "["):
            self.active = (self.active - 1) % len(self.tabs)
        elif key == "t" or key == "f" or key == "s":
            self.mode = "live"
            self.overlay = "help"

    def handle_system_key(self, key, char):
        if key in ("ctrl-g", "esc"):
            self.mode = "live"
        elif key == "q":
            self.emit_result("system.quit", {})
        elif key == "o":
            self.mode = "live"
            self.open_prompt()
        elif key == "?":
            self.mode = "live"
            self.overlay = "help"
        elif key in ("h", "f", "p", "e", "w", "m", "t", "l"):
            self.mode = "live"
            self.toast = "system: " + key + " (v2 host 管理)"

    def handle_floating_key(self, key, char):
        if key in ("ctrl-o", "esc"):
            self.mode = "live"
            return
        floating = self.active_floating()
        if key == "n":
            self.new_floating()
        elif key == "o":
            self.toast = "floating: %d window(s)" % len(self.floatings)
        elif key == "x":
            if floating is not None:
                self.close_floating(floating)
        elif key in ("z", "m"):
            if floating is not None:
                self.toggle_floating_collapse(floating)
        elif key == "c":
            if floating is not None:
                self.center_floating(floating)
        elif key == "f":
            self.open_picker()
        elif key in ("h", "left") and floating is not None:
            floating.x = max(0, floating.x - 2)
            self.raise_floating(floating)
        elif key in ("l", "right") and floating is not None:
            floating.x = min(max(0, self.cols - floating.w), floating.x + 2)
            self.raise_floating(floating)
        elif key in ("k", "up") and floating is not None:
            floating.y = max(1, floating.y - 1)
            self.raise_floating(floating)
        elif key in ("j", "down") and floating is not None:
            floating.y = min(max(1, self.rows - 1 - floating.h),
                             floating.y + 1)
            self.raise_floating(floating)
        elif len(key) == 1 and key.isdigit() and key != "0":
            index = ord(key) - ord("1")
            if 0 <= index < len(self.floatings):
                target = self.floatings[index]
                target.collapsed = False
                self.raise_floating(target)

    def handle_overlay_key(self, key, char):
        if self.overlay == "picker":
            entries = self.terminals()
            if key == "up":
                self.picker = clamp(self.picker - 1, 0, max(0, len(entries)))
            elif key == "down":
                self.picker = clamp(self.picker + 1, 0, max(0, len(entries)))
            elif key in ("enter", "tab"):
                self.attach(self.picker, split=key == "tab")
            elif key == "esc":
                self.overlay = ""
        elif self.overlay == "prompt":
            matches = self.prompt_matches()
            if key == "esc":
                self.overlay = ""
            elif key == "up":
                self.prompt_sel = clamp(self.prompt_sel - 1, 0, max(0, len(matches) - 1))
            elif key == "down":
                self.prompt_sel = clamp(self.prompt_sel + 1, 0, max(0, len(matches) - 1))
            elif key == "backspace":
                if self.prompt:
                    self.prompt = self.prompt[:-1]
                    self.prompt_sel = 0
            elif key == "enter":
                if 0 <= self.prompt_sel < len(matches):
                    self.run_command(matches[self.prompt_sel])
            elif len(key) == 1:
                self.prompt += char or key
                self.prompt_sel = 0
        elif self.overlay == "help":
            if key in ("esc", "?", "enter", "q"):
                self.overlay = ""
        elif self.overlay == "clipboard":
            if key == "esc":
                self.overlay = ""
            elif key == "up":
                self.clip_sel = clamp(self.clip_sel - 1, 0, max(0, len(self.clipboard) - 1))
            elif key == "down":
                self.clip_sel = clamp(self.clip_sel + 1, 0, max(0, len(self.clipboard) - 1))
            elif key == "enter" and self.clipboard:
                self.toast = "clipboard: " + self.clipboard[self.clip_sel][:32]
                self.overlay = ""

    def handle_mouse(self, event):
        action = event.get("action")
        node = event.get("node") or ""
        x = event.get("x", 0)
        y = event.get("y", 0)
        if action == "release":
            self.dragging = ""
            return
        if action == "drag":
            self.drag_move(x, y)
            return
        if action == "press" and node.startswith("divider:"):
            self.dragging = node
            self.drag_last = (x, y)
            return
        if action == "press" and node.startswith("float:") and node.endswith(":title"):
            self.dragging = node.split(":", 2)[1]
            self.drag_last = (x, y)
            self.raise_floating(self.floating_by_id(self.dragging))
            return
        # Terminal content boxes (and demo placeholder lines) carry the raw
        # pane id; clicking them focuses that pane (v3 pane focus).
        pane = self.pane_by_id(node)
        if pane is not None:
            for floating in self.floatings:
                if floating.pane is pane:
                    if floating.collapsed:
                        return
                    self.raise_floating(floating)
                    self.mode = "live"
                    self.active_float = floating.id
                    return
            self.active_float = ""
            self.focus_pane_object(pane)
            return
        if node.startswith("hdr:tabclose:"):
            self.close_tab(atoi_node(node, "hdr:tabclose:"))
            return
        if node.startswith("hdr:tab:"):
            index = atoi_node(node, "hdr:tab:")
            if 0 <= index < len(self.tabs):
                self.active = index
                self.mode = "live"
            return
        if node == "hdr:create":
            self.new_tab()
            return
        if node == "hdr:workspace":
            self.mode = "live"
            self.overlay = "help"
            return
        if node.startswith("pane:") or node.startswith("float:") or node == "toast":
            self.handle_chrome_click(node, x, y)
            return

    def handle_chrome_click(self, node, x, y):
        if node == "toast":
            self.toast = ""
            return
        parts = node.split(":")
        head = parts[0]
        if head == "pane" and len(parts) >= 3:
            pane = self.pane_by_id(parts[1])
            if pane is None:
                return
            action = ":".join(parts[2:])
            if action == "hint":
                pane.collapsed = not pane.collapsed
                return
            if action == "close":
                __, tab = self.tab_of_pane(pane)
                self.close_pane(tab, pane)
                return
            if action in ("split-v", "split-h"):
                self.split_pane("row" if action == "split-v" else "col", pane=pane)
                return
            if action == "zoom":
                self.toast = "zoom " + pane.id
                return
            if action == "lock":
                pane.locked = not pane.locked
                return
            if action == "take-owner":
                self.take_owner(pane)
                return
            self.focus_pane_object(pane)
            return
        if head == "float" and len(parts) >= 3:
            floating = self.floating_by_id(parts[1])
            if floating is None:
                return
            action = parts[2]
            self.raise_floating(floating)
            if action == "close":
                self.close_floating(floating)
            elif action == "collapse":
                self.toggle_floating_collapse(floating)
            elif action == "center":
                self.center_floating(floating)
            elif action == "zoom":
                floating.w = max(20, self.cols - 8)
                floating.h = max(6, self.rows - 4)
                floating.x = 4
                floating.y = 2
            return
        if head in ("picker", "prompt"):
            __ = head
            return

    # ------------------------------------------------------------ actions

    def focus_pane_object(self, pane):
        index, tab = self.tab_of_pane(pane)
        if tab is None:
            return
        self.active = index
        tab.focus = tab.panes.index(pane)

    def focus_pane_delta(self, delta):
        tab = self.active_tab()
        if tab.panes:
            tab.focus = (tab.focus + delta) % len(tab.panes)
        self.active_float = ""

    def split_pane(self, flow, pane=None):
        """Split only the focused (or given) leaf: replace that Leaf with a
        Split(original, new).  Sibling nodes keep their rects and ratios."""
        tab = self.active_tab()
        source = pane if pane is not None else self.focus_pane()
        if source is None:
            return
        leaf = tab.leaf_of(source)
        if leaf is None:
            return
        clone = self.new_pane("")
        split = Split(flow, 0.5, leaf, Leaf(clone),
                      tab.next_split_seq())
        self.replace_leaf(tab, leaf, split)
        tab.rebuild()
        tab.focus = tab.panes.index(clone)
        self.mode = "pane"
        self.active_float = ""
        if self.out is not None and not self.demo:
            self.toast = "creating terminal for " + clone.id
            self.bind_pending(clone.id, None, {"endpoint": "local"})

    def replace_leaf(self, tab, leaf, replacement):
        if tab.root is leaf:
            tab.root = replacement
            return True

        def walk(node):
            if isinstance(node, Leaf):
                return False
            if node.a is leaf:
                node.a = replacement
                return True
            if node.b is leaf:
                node.b = replacement
                return True
            return walk(node.a) or walk(node.b)

        return walk(tab.root)

    def close_pane(self, tab, pane):
        """Remove the leaf; its Split promotes the sibling subtree."""
        if tab is None or pane is None:
            return
        leaf = tab.leaf_of(pane)
        if leaf is None:
            return
        index = tab.panes.index(pane)

        def remove(node):
            if node is leaf:
                return None
            if isinstance(node, Leaf):
                return node
            a = remove(node.a)
            b = remove(node.b)
            if a is None and b is None:
                return None
            if a is None:
                return b
            if b is None:
                return a
            node.a, node.b = a, b
            return node

        tab.root = remove(tab.root)
        if tab.root is None:
            tab.panes = []
            tab.root = Leaf(self.new_pane("empty"))
        tab.rebuild()
        tab.focus = min(index, len(tab.panes) - 1)

    def kill_close_pane(self, tab, pane):
        self.kill_pane(pane)
        self.close_pane(tab, pane)

    def kill_pane(self, pane):
        source = self.pane_source(pane)
        if source is None or not source.get("terminal_id"):
            self.toast = "no terminal to kill"
            return
        self.emit_result("terminal.kill", {
            "endpoint": source.get("endpoint") or "local",
            "id": source["terminal_id"]})

    def restart_pane(self, pane):
        source = self.pane_source(pane)
        if source is None or not source.get("terminal_id"):
            self.toast = "no terminal to restart"
            return
        pane.pending = "restart"
        self.emit_result("terminal.restart", {
            "endpoint": source.get("endpoint") or "local",
            "id": source["terminal_id"]})

    def take_owner(self, pane):
        self.toast = "resize owner: v2 host 自动仲裁"

    def refresh_owner_from_result(self, method):
        return

    def subtree_has_leaf(self, node, leaf):
        if isinstance(node, Leaf):
            return node is leaf
        return (self.subtree_has_leaf(node.a, leaf)
                or self.subtree_has_leaf(node.b, leaf))

    def ancestor_split(self, tab, leaf, orient):
        """The deepest Split containing ``leaf`` on the wanted axis, plus
        whether the leaf sits in its ``a`` child and the split's rect."""
        self.split_entries(tab)
        best = [None]

        def walk(node, depth):
            if isinstance(node, Leaf):
                return
            in_a = self.subtree_has_leaf(node.a, leaf)
            in_b = self.subtree_has_leaf(node.b, leaf)
            if node.orient == orient and (in_a or in_b):
                best[0] = (depth, node, in_a, node.rect)
            if in_a:
                walk(node.a, depth + 1)
            elif in_b:
                walk(node.b, depth + 1)

        if tab.root is not None:
            walk(tab.root, 0)
        return best[0]

    def reset_tab_splits(self, tab):
        def walk(node):
            if isinstance(node, Leaf):
                return
            node.ratio = 0.5
            walk(node.a)
            walk(node.b)
        if tab.root is not None:
            walk(tab.root)

    def resize_focused(self, delta, vertical=False):
        """Grow the focused leaf by ``delta`` along its nearest split axis
        (Ctrl-R SIZE mode); only that Split's ratio changes."""
        tab = self.active_tab()
        pane = self.focus_pane()
        if pane is None:
            return
        leaf = tab.leaf_of(pane)
        if leaf is None:
            return
        orient = "col" if vertical else "row"
        found = self.ancestor_split(tab, leaf, orient)
        if found is None:
            return
        _depth, node, in_a, (x, y, w, h) = found
        avail = max(2, (h if vertical else w) - 5)
        ext = clamp(int(avail * node.ratio + 0.5), 1, avail - 1)
        ext = clamp(ext + (delta if in_a else -delta), 1, avail - 1)
        node.ratio = ext / avail

    def new_tab(self):
        self.tab_seq += 1
        pane = self.new_pane("empty")
        self.tabs.append(Tab("tab-%d" % self.tab_seq, "tab-%d" % self.tab_seq, [pane]))
        self.active = len(self.tabs) - 1
        self.mode = "live"

    def close_tab(self, index):
        if index < 0 or index >= len(self.tabs) or len(self.tabs) == 1:
            return
        del self.tabs[index]
        if self.active >= len(self.tabs):
            self.active = len(self.tabs) - 1

    def open_picker(self):
        self.overlay = "picker"
        self.picker = 0

    def open_prompt(self):
        self.overlay = "prompt"
        self.prompt = ""
        self.prompt_sel = 0

    def focus_content_pane(self):
        if self.active_float:
            floating = self.floating_by_id(self.active_float)
            if floating is not None and not floating.collapsed:
                return floating.pane
        return self.focus_pane()

    def enter_copy(self):
        if self.scroll_source:
            self.scroll_end()
            return
        pane = self.focus_content_pane()
        if pane is not None:
            self.scroll_source = pane.id
            self.scroll_offset = 0
        self.toast = "copy: 上下滚轮/PgUp · Ctrl-P 退出"

    def paste_system(self):
        self.toast = "paste requested (\u21e7V)"

    def open_float_menu(self):
        for floating in reversed(self.floatings):
            if not floating.collapsed:
                self.active_float = floating.id
                return
        if self.floatings:
            self.active_float = self.floatings[-1].id

    def active_floating(self):
        for floating in self.floatings:
            if floating.id == self.active_float:
                return floating
        return None

    def floating_by_id(self, float_id):
        for floating in self.floatings:
            if floating.id == float_id:
                return floating
        return None

    def raise_floating(self, floating):
        if floating is None or floating not in self.floatings:
            return
        self.floatings.remove(floating)
        self.floatings.append(floating)
        if not floating.collapsed:
            self.active_float = floating.id

    def toggle_floating_collapse(self, floating):
        floating.collapsed = not floating.collapsed
        if floating.collapsed:
            self.dragging = ""
            self.toast = "collapsed " + floating.id
        else:
            self.raise_floating(floating)

    def center_floating(self, floating):
        floating.x = max(0, (self.cols - floating.w) // 2)
        floating.y = max(1, (self.rows - max(2, floating.h)) // 2)
        self.raise_floating(floating)

    def new_floating(self):
        self.float_seq += 1
        width = clamp(self.cols * 4 // 5,
                      min(64, max(16, self.cols - 8)),
                      min(112, max(16, self.cols - 4)))
        height = clamp(self.rows * 3 // 4,
                       min(18, max(4, self.rows - 4)),
                       min(32, max(4, self.rows - 2)))
        width = min(width, max(8, self.cols - 2))
        height = min(height, max(3, self.rows - 2))
        cascade = sum(1 for floating in self.floatings if not floating.collapsed)
        x = clamp(max(0, (self.cols - width) // 2) + cascade * 4,
                  0, max(0, self.cols - width))
        y = clamp(max(1, (self.rows - height) // 2) + cascade,
                  1, max(1, self.rows - 1 - height))
        pane = self.new_pane("floating-%d" % self.float_seq,
                             ["", "  floating pane",
                              "  Ctrl-O z 折叠 · 拖动标题移动"])
        floating = Floating("float-%d" % self.float_seq, pane.title,
                            x, y, width, height, pane)
        self.floatings.append(floating)
        self.active_float = floating.id
        if self.out is not None and not self.demo:
            self.bind_pending(pane.id, None, {"endpoint": "local"})

    def close_floating(self, floating):
        if floating in self.floatings:
            self.floatings.remove(floating)
        top = None
        for candidate in reversed(self.floatings):
            if not candidate.collapsed:
                top = candidate
                break
        self.active_float = top.id if top is not None else ""
        if not self.floatings:
            self.mode = "live"

    def drag_move(self, x, y):
        if self.dragging.startswith("divider:"):
            self.drag_resize(x, y)
            return
        floating = self.floating_by_id(self.dragging)
        if floating is None:
            return
        dx = x - self.drag_last[0]
        dy = y - self.drag_last[1]
        floating.x = clamp(floating.x + dx, 0, max(0, self.cols - floating.w))
        floating.y = clamp(floating.y + dy, 1, max(1, self.rows - 1 - floating.h))
        self.drag_last = (x, y)

    def drag_resize(self, x, y):
        """Write the dragged boundary back into exactly one Split's ratio.

        Event x/y are 1-based SGR cells (host converts for hit tests); the
        separator of a split sits at 0-based ``origin + first + 2``, so the
        dropped cell x maps to ``first = x - origin - 3``.
        """
        tab = self.active_tab()
        parts = self.dragging.split(":")
        if len(parts) < 3 or not parts[2].isdigit():
            return
        seq = int(parts[2])
        target = None
        for node, _dx, _dy, _dw, _dh in self.split_entries(tab):
            if node.seq == seq:
                target = node
                break
        if target is None:
            return
        node = target
        sx, sy, sw, sh = node.rect
        if node.orient == "row":
            avail = max(2, sw - 5)
            first = clamp(x - sx - 3, 1, avail - 1)
        else:
            avail = max(2, sh - 5)
            first = clamp(y - sy - 3, 1, avail - 1)
        node.ratio = first / avail

    def prompt_matches(self):
        query = self.prompt.strip().lower()
        if not query:
            return list(self.t.PROMPT_COMMANDS)
        return [command for command in self.t.PROMPT_COMMANDS if query in command.lower()]

    def run_command(self, command):
        self.overlay = ""
        if command == "split row":
            self.split_pane("row")
        elif command == "split col":
            self.split_pane("col")
        elif command == "close pane":
            tab = self.active_tab()
            self.close_pane(tab, self.focus_pane())
        elif command == "new tab":
            self.new_tab()
        elif command == "close tab":
            self.close_tab(self.active)
        elif command == "kill pane":
            self.kill_pane(self.focus_pane())
        elif command == "help":
            self.overlay = "help"
        elif command == "quit":
            self.emit_result("system.quit", {})

    def attach_target(self):
        if self.mode == "floating" and self.active_float:
            floating = self.floating_by_id(self.active_float)
            if floating is not None and not floating.collapsed:
                return floating.pane
        return self.focus_pane()

    def attach(self, index, split=False):
        entries = self.terminals()
        if index < len(entries):
            source = entries[index]
            pane = self.attach_target()
            if pane is None:
                return
            self.bind_pending(pane.id, source.get("id"),
                              {"endpoint": source.get("endpoint") or "local",
                               "id": source.get("terminal_id") or "", "fit": True})
            self.overlay = ""
            self.toast = "attach requested \u00b7 " + (source.get("terminal_id") or "")
            return
        pane = self.attach_target()
        if pane is None:
            return
        self.bind_pending(pane.id, None, {"endpoint": "local"})
        self.overlay = ""
        self.toast = "create requested"

    def bind_pending(self, pane_id, source_id, params):
        method = "terminal.attach" if source_id else "terminal.create"
        pane = self.pane_by_id(pane_id)
        if pane is not None:
            pane.pending = "bind"

        def done(response):
            if not response.get("ok"):
                self.toast = method + " failed: " + (response.get("error") or "")
                if pane is not None:
                    pane.pending = ""
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
        pane.pending = ""
        pane.scroll = 0
        for floating in self.floatings:
            if floating.pane is pane:
                floating.collapsed = False
                self.raise_floating(floating)
                self.active_float = floating.id
                self.mode = "live"
                return

    # --------------------------------------------------------------- wire

    def on_hello(self, hello):
        self.view_id = hello.get("view_id", "")
        self.epoch = hello.get("epoch", 0)
        self.cols = hello.get("cols") or self.cols
        self.rows = hello.get("rows") or self.rows
        self.commit()

    def clamp_floatings(self):
        for floating in self.floatings:
            floating.w = min(floating.w, max(8, self.cols - 2))
            floating.h = min(floating.h, max(3, self.rows - 2))
            floating.x = clamp(floating.x, 0,
                               max(0, self.cols - floating.w))
            floating.y = clamp(floating.y, 1,
                               max(1, self.rows - 1 - floating.h))

    def on_event(self, event):
        kind = event.get("kind")
        if kind == "resize":
            self.cols = event.get("cols") or self.cols
            self.rows = event.get("rows") or self.rows
            self.clamp_floatings()
        elif kind == "key":
            self.handle_key(event.get("key") or event.get("char") or "",
                            event.get("char") or "")
        elif kind == "mouse":
            self.handle_mouse(event)
        elif kind == "wheel":
            self.handle_wheel(event)
        elif kind == "paste":
            pane = self.focus_pane()
            if pane is not None and not pane.source_id and event.get("text"):
                pane.lines.append("> " + event["text"])
        elif kind == "sources":
            self.on_sources(event.get("items") or [])
        elif kind == "notice":
            self.toast = (event.get("level") or "notice") + ": " + (event.get("message") or "")
        self.commit()

    def on_response(self, response):
        # The core Client has already fired the per-request callback.
        self.commit()

    def on_sources(self, items):
        if self.demo:
            return
        self.sources = [source for source in items if source.get("id")]
        self.terminal_count = len(self.terminals())
        present = {source.get("id") for source in self.terminals()}
        for tab in self.tabs:
            for pane in tab.panes:
                if pane.source_id and pane.source_id not in present:
                    pane.source_id = ""
                    pane.lines = []
        if not self.sources_ready:
            self.sources_ready = True
            if not self.terminals() and not self.demo:
                self.open_picker()

    def emit_result(self, method, params=None, on_response=None):
        if self.client is None:
            self.request_id += 1
            return self.request_id
        return self.client.emit(method, params, on_response)

    def commit(self):
        if self.client is None or self.client.hello is None:
            return
        claim, all_keys = self.claim()
        self.client.commit(self.view(), claim, all_keys)

    def handle_wheel(self, event):
        node = event.get("node") or ""
        delta = event.get("delta", 0)
        pane = self.pane_by_id(node)
        if pane is None:
            return
        source = self.pane_source(pane)
        if source is not None and source.get("terminal_id") and not source.get("exited"):
            self.emit_result("terminal.scroll", {
                "endpoint": source.get("endpoint") or "local",
                "id": source["terminal_id"], "delta": delta})
            if self.scroll_source != pane.id:
                self.scroll_source = pane.id
                self.scroll_offset = 0
            self.scroll_offset += delta
        else:
            pane.scroll = clamp(pane.scroll - delta, 0, len(pane.lines))
        if self.scroll_offset <= 0:
            self.scroll_end()

    def scroll_end(self):
        pane = self.pane_by_id(self.scroll_source)
        if pane is not None:
            source = self.pane_source(pane)
            if source is not None and source.get("terminal_id"):
                self.emit_result("terminal.scrollEnd", {
                    "endpoint": source.get("endpoint") or "local",
                    "id": source["terminal_id"]})
        self.scroll_source = ""
        self.scroll_offset = 0

    PROMPT_COMMANDS = [
        "split row", "split col", "close pane", "kill pane", "new tab",
        "close tab", "help", "quit",
    ]

    HELP_LINES = [
        "全局 (global)",
        "  Ctrl-P PANE   Ctrl-R RESIZE   Ctrl-O FLOAT",
        "  Ctrl-T TAB    Ctrl-W WORKSPACE Ctrl-F PICK",
        "  Ctrl-G SYSTEM \u21e7C copy \u21e7H clipboard \u21e7V paste",
        "PANE",
        "  x close  % / Ctrl-D vsplit  \" / Ctrl-E hsplit",
        "  h/l focus  t restart  k kill  q kill+close  z collapse",
        "TAB / WORKSPACE",
        "  c create  n/p next/prev  1-9 jump  x close",
        "SYSTEM",
        "  q quit  o command  ? help",
        "esc back \u00b7 Ctrl-Q quit",
    ]


# ------------------------------------------------------------- box helpers

def text_box(node_id, text, text_style, width=0, height=0, input_kinds=None, cursor=None):
    box = {"id": node_id, "content": {"text": text}}
    if text_style:
        box["style"] = text_style
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


def clear_rect(grid, styles, rect, cols, rows):
    x, y, width, height = rect
    for row in range(max(0, y), min(rows, y + height)):
        for col in range(max(0, x), min(cols, x + width)):
            grid[row][col] = " "
            styles[row][col] = ""


def put_text(grid, styles, x, y, width, text, text_style, program):
    if y < 0 or y >= program.rows or width <= 0:
        return
    cursor = x
    remaining = width
    for ch in text:
        cells = cell_width(ch)
        if cells == 0:
            continue
        if cells > remaining or cursor >= program.cols:
            break
        if cursor >= 0:
            grid[y][cursor] = ch
            styles[y][cursor] = text_style
            for offset in range(1, cells):
                if cursor + offset < program.cols:
                    grid[y][cursor + offset] = ""
                    styles[y][cursor + offset] = text_style
        cursor += cells
        remaining -= cells


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
