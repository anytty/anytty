"""Chainable view-tree builder + text metrics (stdlib only).

A box is the only framework primitive (PROTOCOL §2): the program declares
geometry/content/input and the host solves rect/flex/hit. ``Node`` mirrors the
protocol fields one-to-one; zero values keep the protocol defaults, so a
builder only carries what the program declared.

    view = col(
        row(text(" left "), text(" right ").id("btn").input("mouse")),
        text("body"),
    ).id("root")

The dict produced by ``build()`` is what ``wire.encode_view`` encodes.
"""

import unicodedata


class Node:
    """One view-tree node; every setter returns self for chaining."""

    def __init__(self, flow=None):
        self._box = {}
        if flow:
            self._box["flow"] = flow

    def id(self, value):
        self._box["id"] = value
        return self

    def size(self, width=0, height=0, flex=0):
        self._box["size"] = (width, height, flex)
        return self

    def width(self, value):
        return self._size_at(0, value)

    def height(self, value):
        return self._size_at(1, value)

    def flex(self, value):
        return self._size_at(2, value)

    def _size_at(self, index, value):
        current = list(self._box.get("size") or (0, 0, 0))
        current[index] = value
        self._box["size"] = tuple(current)
        return self

    def pos(self, x, y):
        self._box["pos"] = (x, y)
        return self

    def flow(self, value):
        self._box["flow"] = value
        return self

    def style(self, value):
        self._box["style"] = value
        return self

    def focused(self, value=True):
        self._box["focused"] = bool(value)
        return self

    def input(self, *kinds):
        self._box["input"] = list(self._box.get("input") or ()) + [k for k in kinds if k]
        return self

    def cursor(self, row=0, col=0, shape="", visible=None):
        cursor = {"row": row, "col": col, "shape": shape}
        if visible is not None:
            cursor["visible"] = bool(visible)
        self._box["cursor"] = cursor
        return self

    def content(self, value):
        self._box["content"] = {"text": value}
        return self

    def lines(self, *values):
        self._box["content"] = {"lines": list(values)}
        return self

    def self_ref(self, source_id):
        self._box["content"] = {"self": source_id}
        return self

    def props(self, values):
        content = self._box.get("content")
        if not content:
            content = {}
            self._box["content"] = content
        merged = dict(content.get("props") or {})
        merged.update(values or {})
        content["props"] = merged
        return self

    def visible(self, value):
        self._box["visible"] = bool(value)
        return self

    def child(self, *nodes):
        children = self._box.setdefault("children", [])
        for node in nodes:
            if node is None:
                continue
            children.append(node.build() if isinstance(node, Node) else node)
        return self

    def build(self):
        out = {}
        for key, value in self._box.items():
            if key == "children":
                out["children"] = [dict(child) for child in value]
            elif key == "content":
                copied = dict(value)
                if value.get("lines") is not None:
                    copied["lines"] = list(value["lines"])
                if value.get("props"):
                    copied["props"] = dict(value["props"])
                out["content"] = copied
            elif key == "input":
                out["input"] = list(value)
            elif key == "cursor":
                out["cursor"] = dict(value)
            else:
                out[key] = value
        return out


def box(flow=None):
    return Node(flow)


def col(*children):
    return Node("col").child(*children)


def row(*children):
    return Node("row").child(*children)


def stack(*children):
    return Node("stack").child(*children)


def text(value):
    return Node().content(value)


def terminal(source_id):
    return Node().self_ref(source_id)


def divider(vertical=False, length=1):
    node = Node().input("mouse")
    if vertical:
        return node.width(1).height(length)
    return node.width(length).height(1)


# ------------------------------------------------------------ text metrics

def cell_width(ch):
    if unicodedata.combining(ch):
        return 0
    return 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1


def display_width(value):
    return sum(cell_width(ch) for ch in value)


def truncate(value, max_width):
    if max_width <= 0:
        return ""
    out = []
    used = 0
    for ch in value:
        cells = cell_width(ch)
        if used + cells > max_width and cells:
            break
        out.append(ch)
        used += cells
    return "".join(out)


def pad_right(value, cells):
    width = display_width(value)
    if width >= cells:
        return value
    return value + " " * (cells - width)


def center_pad(value, cells):
    value = truncate(value, cells)
    pad = cells - display_width(value)
    left = pad // 2
    return " " * left + value + " " * (pad - left)


def clamp(value, low, high):
    if value < low:
        return low
    if value > high:
        return high
    return value
