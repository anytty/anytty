#!/usr/bin/env python3
"""Minimal TUI v2 layout program in pure Python (stdlib only).

This is the reference proof that a layout program is just a process speaking
the binary protocol on stdin/stdout: it never imports the Go framework. The
UI is intentionally small but complete for daily use:

  - tab bar + one split group per tab (row/col) with empty-slot cards
  - footer with mode hints and status
  - Ctrl-P pane mode: ``%`` split side by side, ``"`` split stacked, ``x``
    close/unbind, Tab focus next, ``1..9`` switch tab, Esc back
  - Ctrl-F terminal picker: up/down, enter binds the selected source (or
    creates a new terminal), mouse click binds, Esc closes
  - Ctrl-T new tab
  - terminal.attach / terminal.create through RESULT, parsed from RESPONSE

Requires Python 3.8+. Wire codec lives in pb.py; protocol reference:
clients/tui/docs/PROTOCOL.zh-CN.md.
"""

import sys
import unicodedata

import pb

# ---------------------------------------------------------------- appearance

GAP = 1  # pane gutter in cells

BORDER = "fg:#464a5a"
BORDER_FOCUS = "fg:#a78bfa;bold"
BORDER_DEAD = "fg:#a35d5d"
MUTED = "fg:#9a94a8;dim"
STATUS = "fg:#d5d0e0;bg:#12141c"
TAB_ACTIVE = "fg:#14121c;bg:#a78bfa;bold"
TAB_INACTIVE = "fg:#9a94a8;dim"
SELECTION = "fg:#f4f1fa;bg:#3a3357"
ACCENT = "fg:#a78bfa;bold"
WARNING = "fg:#f0c05a"
FG = "fg:#e6e2ec"

CHROME_PROPS = {
    "chrome.border": BORDER,
    "chrome.title": MUTED,
    "chrome.border_focus": BORDER_FOCUS,
    "chrome.border_dead": BORDER_DEAD,
    "chrome.badge": WARNING,
}


def width(text):
    """Display cells of text (wide CJK/emoji = 2, combining marks = 0)."""
    total = 0
    for ch in text:
        if unicodedata.combining(ch):
            continue
        total += 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1
    return total


def truncate(text, max_width):
    if max_width <= 0:
        return ""
    out = []
    used = 0
    for ch in text:
        cells = width(ch)
        if used + cells > max_width and cells:
            break
        out.append(ch)
        used += cells
    return "".join(out)


def pad(text, cells):
    text = truncate(text, cells)
    return text + " " * max(0, cells - width(text))


def center(text, cells):
    """Pad text to `cells` display cells, centered."""
    text = truncate(text, cells)
    left = max(0, (cells - width(text)) // 2)
    return " " * left + text + " " * max(0, cells - left - width(text))


def rule(box_width, title):
    """A horizontal rule of exactly box_width cells with a centered title."""
    inner = box_width - 2
    if inner <= 0:
        return "┌┐"
    heading = truncate(title, max(0, inner - 1))
    return "┌" + heading + "─" * max(0, inner - width(heading)) + "┐"


# --------------------------------------------------------------------- state

class Slot:
    __slots__ = ("id", "source_id")

    def __init__(self, slot_id):
        self.id = slot_id
        self.source_id = ""


class Tab:
    __slots__ = ("id", "name", "flow", "slots", "focus")

    def __init__(self, tab_id, name, flow="col"):
        self.id = tab_id
        self.name = name
        self.flow = flow
        self.slots = []
        self.focus = 0


class Program:
    def __init__(self, out):
        self.out = out
        self.epoch = 0
        self.rev = 0
        self.view_id = ""
        self.cols = 80
        self.rows = 24
        self.seq = 0
        self.tabs = [self.new_tab("1")]
        self.active = 0
        self.mode = "NORMAL"
        self.sources = []
        self.sources_ready = False
        self.picker_idx = 0
        self.status = ""
        self.request_id = 0
        self.pending = {}

    # ------------------------------------------------------------- helpers

    def next_id(self, prefix):
        self.seq += 1
        return "%s-%d" % (prefix, self.seq)

    def new_tab(self, name):
        tab = Tab(self.next_id("tab"), name or str(len(self.tabs) + 1))
        tab.slots = [Slot(self.next_id("slot"))]
        return tab

    def active_tab(self):
        if self.active < 0 or self.active >= len(self.tabs):
            self.active = 0
        return self.tabs[self.active]

    def focus_slot(self):
        tab = self.active_tab()
        if not tab.slots:
            tab.slots = [Slot(self.next_id("slot"))]
            tab.focus = 0
        if tab.focus < 0 or tab.focus >= len(tab.slots):
            tab.focus = 0
        return tab.slots[tab.focus]

    def find_slot(self, slot_id):
        for tab in self.tabs:
            for index, slot in enumerate(tab.slots):
                if slot.id == slot_id:
                    return tab, index
        return None, -1

    def source_by_id(self, source_id):
        for source in self.sources:
            if source["id"] == source_id:
                return source
        return None

    def has_binding(self):
        return any(slot.source_id for tab in self.tabs for slot in tab.slots)

    def bind(self, slot_id, source_id):
        tab, index = self.find_slot(slot_id)
        if tab is None:
            return
        tab.slots[index].source_id = source_id
        self.mode = "NORMAL"

    @staticmethod
    def short_source_id(source_id):
        parts = source_id.split(":", 2)
        return parts[2] if len(parts) == 3 else source_id

    def slot_title(self, slot):
        if not slot.source_id:
            return "空槽"
        source = self.source_by_id(slot.source_id)
        if source and source["title"]:
            return source["title"]
        return self.short_source_id(slot.source_id)

    # -------------------------------------------------------------- events

    def on_hello(self, hello):
        self.view_id = hello["view_id"]
        self.epoch = hello["epoch"]
        self.cols = hello["cols"] or self.cols
        self.rows = hello["rows"] or self.rows
        self.commit()

    def on_event(self, event):
        kind = event["kind"]
        if kind == "sources":
            self.on_sources(event["items"])
        elif kind == "key":
            self.on_key(event["key"] or event.get("char", ""))
        elif kind == "mouse":
            self.on_mouse(event["action"], event["node"])
        elif kind == "resize":
            self.cols = event["cols"] or self.cols
            self.rows = event["rows"] or self.rows
            self.commit()
        elif kind == "notice":
            self.status = event["message"]
            self.commit()
        elif kind == "view_rejected":
            self.status = "view rejected: " + event["reason"]
            self.commit()

    def on_sources(self, items):
        self.sources = sorted([s for s in items if s["kind"] == "terminal"],
                              key=lambda s: s["id"])
        if not self.sources_ready:
            self.sources_ready = True
            if not self.has_binding():
                self.open_picker()
                self.commit()
                return
        present = {s["id"] for s in self.sources}
        for tab in self.tabs:
            for slot in tab.slots:
                if slot.source_id and slot.source_id not in present:
                    slot.source_id = ""
        self.commit()

    def on_key(self, key):
        self.status = ""
        if self.mode == "PICKER":
            self.key_picker(key)
        elif self.mode == "PANE":
            self.key_pane(key)
        else:
            self.key_normal(key)
        self.commit()

    def key_normal(self, key):
        if key == "ctrl-p":
            self.mode = "PANE"
        elif key == "ctrl-f":
            self.open_picker()
        elif key == "ctrl-t":
            self.add_tab()

    def key_pane(self, key):
        if key in ("ctrl-p", "esc"):
            self.mode = "NORMAL"
        elif key == "%":
            self.split("row")
        elif key == '"':
            self.split("col")
        elif key == "x":
            self.close_slot()
        elif key == "tab":
            tab = self.active_tab()
            tab.focus = (tab.focus + 1) % len(tab.slots)
        elif key == "ctrl-f":
            self.open_picker()
        elif key == "ctrl-t":
            self.add_tab()
        elif len(key) == 1 and "1" <= key <= "9":
            index = ord(key) - ord("1")
            if 0 <= index < len(self.tabs):
                self.active = index

    def key_picker(self, key):
        items = self.picker_items()
        if key == "up":
            self.picker_idx = max(0, self.picker_idx - 1)
        elif key == "down":
            self.picker_idx = min(max(0, len(items) - 1), self.picker_idx + 1)
        elif key == "enter":
            self.pick()
        elif key in ("esc", "ctrl-f"):
            self.mode = "NORMAL"

    def on_mouse(self, action, node):
        if action != "press":
            return
        if node.startswith("pick:") and self.mode == "PICKER":
            self.picker_idx = int(node[len("pick:"):] or 0)
            self.pick()
        elif node == "tab:new":
            self.add_tab()
        elif node.startswith("tab:"):
            index = int(node[len("tab:"):] or 0)
            if 0 <= index < len(self.tabs):
                self.active = index
        elif node.startswith("slot-"):
            tab, index = self.find_slot(node)
            if tab is not None:
                self.active = self.tabs.index(tab)
                tab.focus = index
                if not tab.slots[index].source_id:
                    self.mode = "PANE"
        self.commit()

    # ----------------------------------------------------------- picker

    def open_picker(self):
        self.mode = "PICKER"
        self.picker_idx = 0

    def picker_items(self):
        items = []
        for source in self.sources:
            state = "exited" if source["exited"] else "live"
            items.append({
                "label": source["title"] or source["id"],
                "info": "%s · %s" % (source["endpoint"] or "local", state),
                "source": source,
            })
        items.append({"label": "+ New terminal", "info": "create", "source": None})
        return items

    def pick(self):
        items = self.picker_items()
        if not items:
            self.mode = "NORMAL"
            return
        item = items[min(max(0, self.picker_idx), len(items) - 1)]
        slot_id = self.focus_slot().id
        self.mode = "NORMAL"
        if item["source"] is None:
            self.status = "creating terminal"
            params = {"endpoint": "local"}

            def on_create(response):
                if not response["ok"]:
                    self.status = "create failed: " + response["error"]
                    return
                endpoint = response["data"].get("endpoint") or "local"
                terminal_id = response["data"].get("id", "")
                self.bind(slot_id, "terminal:%s:%s" % (endpoint, terminal_id))
                self.status = "bound " + terminal_id

            self.emit("terminal.create", params, on_create)
            return
        source_id = item["source"]["id"]
        self.status = "binding " + self.short_source_id(source_id)
        params = self.attach_params(source_id)

        def on_attach(response):
            if not response["ok"]:
                self.status = "attach failed: " + response["error"]
                return
            self.bind(slot_id, source_id)
            self.status = "bound " + self.short_source_id(source_id)

        self.emit("terminal.attach", params, on_attach)

    @staticmethod
    def attach_params(source_id):
        parts = source_id.split(":", 2)
        endpoint = parts[1] if len(parts) == 3 else "local"
        terminal_id = parts[2] if len(parts) == 3 else source_id
        return {"endpoint": endpoint, "id": terminal_id, "fit": True}

    # -------------------------------------------------------- layout edits

    def split(self, direction):
        tab = self.active_tab()
        tab.flow = direction
        focus = tab.focus
        tab.slots.insert(focus + 1, Slot(self.next_id("slot")))
        tab.focus = focus + 1

    def close_slot(self):
        tab = self.active_tab()
        if not tab.slots:
            return
        if len(tab.slots) > 1:
            del tab.slots[tab.focus]
            tab.focus = min(tab.focus, len(tab.slots) - 1)
        else:
            tab.slots[0].source_id = ""

    def add_tab(self):
        self.tabs.append(self.new_tab(str(len(self.tabs) + 1)))
        self.active = len(self.tabs) - 1
        self.open_picker()

    # ---------------------------------------------------------- protocol

    def emit(self, method, params, on_response=None):
        self.request_id += 1
        request_id = self.request_id
        if on_response is not None:
            self.pending[request_id] = on_response
        self.write(pb.RESULT, pb.encode_result(request_id, self.epoch, method, params))

    def on_response(self, response):
        callback = self.pending.pop(response["request_id"], None)
        if callback is not None:
            callback(response)
            self.commit()

    def claim(self):
        if self.mode != "NORMAL":
            return [], True
        return ["ctrl-p", "ctrl-f", "ctrl-t"], False

    def commit(self):
        if not self.view_id:
            return  # before HELLO there is nothing to commit
        self.rev += 1
        claim, all_keys = self.claim()
        root = self.build_view()
        self.write(pb.VIEW, pb.encode_view(self.epoch, self.rev, claim, all_keys, root))

    def write(self, frame_type, payload):
        self.out.write(pb.frame(frame_type, payload))
        self.out.flush()

    # ---------------------------------------------------------------- view

    def build_view(self):
        children = [
            self.header(),
            {"id": "body", "flow": "row", "children": [self.panes()]},
            self.footer(),
        ]
        overlay = self.overlay()
        if overlay is not None:
            children.append(overlay)
        return {"id": "root", "flow": "col", "children": children}

    def header(self):
        children = [text_box("workspace", " local ", MUTED)]
        for index, tab in enumerate(self.tabs):
            if index == self.active:
                style, label = TAB_ACTIVE, "[%d:%s]" % (index + 1, tab.name)
            else:
                style, label = TAB_INACTIVE, " %d:%s " % (index + 1, tab.name)
            children.append(text_box("tab:%d" % index, label, style, input_kinds=["mouse"]))
        children.append(text_box("tab:new", " + ", MUTED, input_kinds=["mouse"]))
        return {"id": "header", "size": (0, 1, 0), "flow": "row", "children": children}

    def panes(self):
        tab = self.active_tab()
        height = max(1, self.rows - 2)
        children = []
        if tab.flow == "row":
            sizes = distribute(self.cols - (len(tab.slots) - 1) * GAP,
                               [100] * len(tab.slots))
            for index, slot in enumerate(tab.slots):
                children.append(self.slot_box(slot, sizes[index], height))
                if index < len(tab.slots) - 1 and GAP > 0:
                    children.append({"id": "divider:%d" % index, "size": (1, height, 0),
                                     "content": {"text": "│"}, "style": MUTED,
                                     "input": ["mouse"]})
        else:
            sizes = distribute(height - (len(tab.slots) - 1) * GAP,
                               [100] * len(tab.slots))
            for index, slot in enumerate(tab.slots):
                children.append(self.slot_box(slot, self.cols, sizes[index]))
                if index < len(tab.slots) - 1 and GAP > 0:
                    children.append({"id": "divider:%d" % index, "size": (self.cols, 1, 0),
                                     "content": {"text": "─" * self.cols}, "style": MUTED,
                                     "input": ["mouse"]})
        return {"id": "panes", "size": (self.cols, height, 0), "flow": tab.flow,
                "children": children}

    def slot_box(self, slot, box_width, box_height):
        if not slot.source_id:
            return self.empty_card(slot, box_width, box_height)
        focused = self.mode == "NORMAL" and self.focus_slot() is slot and self.attached(slot)
        title = self.slot_title(slot)
        style = BORDER_FOCUS if focused else MUTED
        box = {
            "id": slot.id,
            "size": (box_width, box_height, 0),
            "content": {"self": slot.source_id, "props": dict(CHROME_PROPS)},
            "input": ["key", "paste", "wheel"],
            "focused": focused,
            "children": [{
                "id": "title:" + slot.id,
                "pos": (0, 0),
                "size": (box_width, 1, 0),
                "flow": "row",
                "children": [text_box("", "▎" + title, style, height=1)],
            }],
        }
        return box

    def attached(self, slot):
        source = self.source_by_id(slot.source_id)
        return source is not None and source["attached"] and not source["exited"]

    def empty_card(self, slot, box_width, box_height):
        inner = max(1, box_width - 2)
        lines = ["Ctrl-F 选择终端", "Ctrl-P 面板命令"]
        if box_width < 3 or box_height < 2:
            rows = [text_box("", truncate(line, max(1, box_width)), BORDER, width=max(1, box_width))
                    for line in lines[:max(1, box_height)]]
        else:
            rows = [text_box("", rule(box_width, " 空槽 "), BORDER, width=box_width, height=1)]
            body_rows = box_height - 2
            offset = max(0, (body_rows - len(lines)) // 2) if body_rows >= len(lines) else 0
            for index in range(body_rows):
                text = ""
                if body_rows >= len(lines) and index >= offset and index - offset < len(lines):
                    text = lines[index - offset]
                rows.append(text_box("", "│" + center(text, inner) + "│", FG, width=box_width, height=1))
            rows.append(text_box("", "└" + "─" * inner + "┘", BORDER, width=box_width, height=1))
        return {"id": slot.id, "size": (box_width, box_height, 0), "flow": "col",
                "input": ["mouse"], "children": rows}

    def footer(self):
        if self.mode == "PANE":
            hint = '% split-h · " split-v · x close · Tab next'
        elif self.mode == "PICKER":
            hint = "↑/↓ select · enter bind · esc close"
        else:
            hint = "Ctrl-P pane · Ctrl-F picker · Ctrl-T tab"
        status = "%s │ tab %d/%d │ slot %d/%d" % (
            self.mode, self.active + 1, len(self.tabs),
            sum(1 for s in self.active_tab().slots if s.source_id),
            len(self.active_tab().slots))
        if self.status:
            status = self.status + " │ " + status
        return {
            "id": "footer", "size": (0, 1, 0), "flow": "row",
            "children": [
                text_box("footer.keys", hint, MUTED, height=1),
                {"id": "footer.gap", "size": (0, 1, 1), "content": {"text": " "}},
                text_box("footer.status", truncate(status, self.cols), STATUS, height=1),
            ],
        }

    # ------------------------------------------------------------- overlay

    def overlay(self):
        if self.mode != "PICKER":
            return None
        items = self.picker_items()
        rows = [{"text": " select a terminal", "style": MUTED}]
        for index, item in enumerate(items):
            marker = " > " if index == self.picker_idx else "   "
            style = SELECTION if index == self.picker_idx else FG
            text = marker + pad(item["label"], 30) + "  " + truncate(item["info"], 18)
            rows.append({"text": text, "style": style, "id": "pick:%d" % index, "input": ["mouse"]})
        rows.append({"text": " enter bind · esc close", "style": MUTED})
        return self.framed("terminals", 58, rows)

    def framed(self, title, box_width, rows):
        box_width = min(box_width, max(8, self.cols - 2))
        inner = max(1, box_width - 2)
        rows = rows[:max(1, self.rows - 4)]
        children = [text_box("", rule(box_width, " " + title + " "), ACCENT, width=box_width, height=1)]
        for row in rows:
            box = text_box(row.get("id", ""), "│" + pad(row["text"], inner) + "│",
                           row.get("style", ""), width=box_width, height=1)
            if row.get("input"):
                box["input"] = list(row["input"])
            children.append(box)
        children.append(text_box("", "└" + "─" * inner + "┘", ACCENT, width=box_width, height=1))
        height = len(children)
        x = max(0, (self.cols - box_width) // 2)
        y = max(0, (self.rows - height) // 2)
        return {"id": "overlay:" + title, "flow": "col", "pos": (x, y),
                "size": (box_width, height, 0), "children": children}


def text_box(node_id, text, style, width=0, height=0, input_kinds=None):
    box = {"id": node_id, "content": {"text": text}}
    if style:
        box["style"] = style
    if width or height:
        box["size"] = (width, height, 0)
    if input_kinds:
        box["input"] = list(input_kinds)
    return box


def distribute(available, ratios):
    """Split `available` cells among ratios, exact sum, 1 cell minimum."""
    count = len(ratios)
    if count == 0:
        return []
    if available < count:
        available = count
    total = sum(r for r in ratios if r > 0) or count
    sizes = [max(1, available * r // total) for r in ratios]
    sizes[-1] += available - sum(sizes)
    if sizes[-1] < 1:
        sizes[-1] = 1
    return sizes


def main():
    out = sys.stdout.buffer
    program = Program(out)
    reader = sys.stdin.buffer
    try:
        while True:
            frame = pb.read_frame(reader)
            if frame is None:
                return 0
            frame_type, payload = frame
            if frame_type == pb.HELLO:
                program.on_hello(pb.decode_hello(payload))
            elif frame_type == pb.EVENT:
                program.on_event(pb.decode_event(payload))
            elif frame_type == pb.RESPONSE:
                program.on_response(pb.decode_response(payload))
    except (BrokenPipeError, KeyboardInterrupt):
        return 0
    except EOFError:
        return 0


if __name__ == "__main__":
    sys.exit(main())
