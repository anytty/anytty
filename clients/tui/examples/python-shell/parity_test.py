#!/usr/bin/env python3
"""Pixel-parity checks for legacy.py against the legacy source formulas.

Two independent computations are compared cell by cell:

  - expected: an *independent* renderer (this file) that follows the legacy
    layout formulas read from ``shell/main.go`` (header/sidebar/paneWidths/
    window/footerText/center) and the legacy border/title rules from the old
    surface program (``┌─…┐`` with the label embedded at offset 1, content
    inset by one) with the legacy theme tokens;
  - actual: the view produced by ``legacy.py``'s state machine, rasterized
    with the tui2 kernel semantics (flow/pos/intrinsic sizes/truncation) and
    the legacy terminal-component chrome.

Covered viewports: 120x32 and 100x30. Covered scenarios: cold-start picker,
single pane, horizontal split, vertical split, help, prompt (empty/filtered),
two tabs, exit badge (code 7 and 0), scrollback badge, picker with a source,
sidebar toggle, toast and footer right alignment.

Usage:
  python3 parity_test.py [--golden-dir DIR] [--only NAME]
"""

import argparse
import os
import sys
import unicodedata

import legacy

VIEW_ID = "view:test:1"

# Independent copy of the legacy theme token mapping (LEGACY_PARITY.md).
FG = "fg:#dedbe6"
ACCENT = "fg:#a970ff;bold"
MUTED = "fg:#b8b1c4;dim"
FOOTER = "fg:#b8b1c4"
WARN = "fg:#f0c45c"
HEADER = "fg:#d4c0f4;bg:#3c2e55;bold"

NORMAL_HINTS = "[Ctrl+P] PANE │ [Ctrl+R] RESIZE │ [Ctrl+G] GLOBAL │ [Ctrl+F] PICKER │ [?] HELP │ [:] CMD"
PANE_HINTS = "[%] SPLIT H │ [\"] SPLIT V │ [x] CLOSE │ [tab] NEXT │ [Ctrl+F] PICKER │ [:] CMD │ [esc] EXIT"
EXITED_HINTS = "[Ctrl+E] RESTART │ [Ctrl+F] PICKER │ terminal exited"
SCROLL_HINTS = "[PgUp/PgDn] SCROLL +%d │ [y] COPY │ [esc] LIVE"

HELP_LINES = legacy.HELP_LINES
INITIAL_LINES = legacy.INITIAL_LINES
PROMPT_COMMANDS = legacy.PROMPT_COMMANDS


# ---------------------------------------------------------------- geometry

def wlen(text):
    total = 0
    for ch in text:
        if unicodedata.combining(ch):
            continue
        total += 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1
    return total


def trunc(text, cells):
    if cells <= 0:
        return ""
    out = []
    used = 0
    for ch in text:
        cells_ch = 0 if unicodedata.combining(ch) else (
            2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1)
        if used + cells_ch > cells and cells_ch:
            break
        out.append(ch)
        used += cells_ch
    return "".join(out)


def extents(available, weights):
    """shell/main.go paneWidths() formula."""
    count = len(weights)
    if count == 0:
        return []
    if available < count:
        available = count
    total = sum(weight if weight > 0 else 1 for weight in weights)
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


def centered(cols, rows, width, height):
    """shell/main.go center()."""
    width = min(width, cols)
    height = min(height, rows)
    return width, height, max(0, (cols - width) // 2), max(0, (rows - height) // 2)


def top_rule(width, title):
    if width < 2:
        return "┌" if width == 1 else ""
    runes = list("┌" + "─" * (width - 2) + "┐")
    label = list(" " + title + " ")
    if len(label) + 2 <= len(runes):
        runes[1:1 + len(label)] = label
    return "".join(runes)


def bottom_rule(width):
    if width < 2:
        return "└" if width == 1 else ""
    return "└" + "─" * (width - 2) + "┘"


class Screen:
    def __init__(self, cols, rows):
        self.cols = cols
        self.rows = rows
        self.grid = [[" " for _ in range(cols)] for _ in range(rows)]
        self.styles = [["" for _ in range(cols)] for _ in range(rows)]

    def put(self, x, y, text, style=""):
        if y < 0 or y >= self.rows or x >= self.cols:
            return
        cursor = x
        for ch in text:
            cells = 0 if unicodedata.combining(ch) else (
                2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1)
            if cells == 0:
                continue
            if cursor + cells > self.cols:
                break
            if cursor >= 0:
                self.grid[y][cursor] = ch
                self.styles[y][cursor] = style
                for offset in range(1, cells):
                    if cursor + offset < self.cols:
                        self.grid[y][cursor + offset] = ""
                        self.styles[y][cursor + offset] = style
            cursor += cells

    def clear(self, x, y, width, height):
        """The legacy border frame writes spaces across the whole overlay
        (and the kernel clears pos subtrees opaquely): both erase whatever was
        underneath the overlay rectangle."""
        for row in range(max(0, y), min(self.rows, y + height)):
            for col in range(max(0, x), min(self.cols, x + width)):
                self.grid[row][col] = " "
                self.styles[row][col] = ""

    def box(self, x, y, width, height, title, style):
        """Legacy borderLines(): top rule with title at offset 1, bare left/
        right rules, bottom rule. The interior is left untouched."""
        if width < 2 or height < 2:
            return
        self.put(x, y, top_rule(width, title), style)
        for row in range(1, height - 1):
            self.put(x, y + row, "│", style)
            self.put(x + width - 1, y + row, "│", style)
        self.put(x, y + height - 1, bottom_rule(width), style)

    def lines(self):
        return ["".join(row) for row in self.grid]


# ------------------------------------------------------- legacy primitives

def draw_header(sc, tabs, active):
    x = 0
    sc.put(x, 0, " WS main ", HEADER)
    x += 9
    for index, title in enumerate(tabs):
        marker = "▎" if index == active else " "
        style = ACCENT if index == active else FG
        label = " %s%d %s " % (marker, index + 1, title)
        sc.put(x, 0, label, style)
        x += wlen(label)
        sc.put(x, 0, "× ", MUTED)
        x += 2
    sc.put(x, 0, " + ", MUTED)


def draw_status(sc, x, y, tabs_n, panes_n, mode, world):
    sc.box(x, y, 30, 6, "status", MUTED)
    lines = [
        "tabs      %d" % tabs_n,
        "panes     %d" % panes_n,
        "mode      " + mode,
        world,
    ]
    for index, line in enumerate(lines):
        sc.put(x + 1, y + 1 + index, trunc(line, 28), MUTED)


def draw_local_pane(sc, x, y, width, height, title, focused, lines):
    marker, style = ("▎ ", ACCENT) if focused else ("  ", MUTED)
    sc.box(x, y, width, height, marker + title, style)
    for index, line in enumerate(lines[:max(0, height - 2)]):
        sc.put(x + 1, y + 1 + index, trunc(line, width - 2), MUTED)


def draw_terminal_pane(sc, x, y, width, height, title, focused, exited, code, scroll):
    """Legacy component chrome: exited->warning wins, focused->accent, else
    muted; the label and the border share the style."""
    if exited:
        style = WARN
        title = title + (" [exited %d]" % code if code else " [exited]")
    elif focused:
        style = ACCENT
    else:
        style = MUTED
    if scroll > 0:
        title = title + " [↑%d]" % scroll
    if width < 3 or height < 3:
        return
    prefix = " " + ("▎" if focused and not exited else "")
    label = ""
    available = width - 2 - wlen(prefix) - 1
    if width >= 4 and available > 0:
        label = prefix + trunc(title, available) + " "
    top = "┌" + label + "─" * max(0, width - 2 - wlen(label)) + "┐"
    sc.put(x, y, top, style)
    for row in range(1, height - 1):
        sc.put(x, y + row, "│", style)
        sc.put(x + width - 1, y + row, "│", style)
    sc.put(x, y + height - 1, bottom_rule(width), style)


def draw_footer(sc, cols, rows, hints, right):
    left = " " + hints
    pad = cols - wlen(left) - wlen(right)
    if pad < 1:
        pad = 1
    sc.put(0, rows - 1, left + " " * pad + right, FOOTER)


def draw_picker(sc, cols, rows, items, selected):
    rows_data = []
    for index, item in enumerate(items):
        marker, style = ("▸ ", ACCENT) if index == selected else ("  ", MUTED)
        rows_data.append((marker + item, style))
    rows_data.append(("↑↓ select · enter/click attach · esc close", MUTED))
    width, height, x, y = centered(cols, rows, legacy.PICKER_WIDTH, legacy.PICKER_HEIGHT)
    sc.clear(x, y, width, height)
    sc.box(x, y, width, height, "Terminal Picker", ACCENT)
    for index, (text, style) in enumerate(rows_data):
        sc.put(x + 1, y + 1 + index, trunc(text, width - 2), style)


def draw_help(sc, cols, rows):
    lines = [("Help", ACCENT)] + [(line, FG) for line in HELP_LINES] + [("esc close", MUTED)]
    width, height, x, y = centered(cols, rows, legacy.HELP_WIDTH, len(HELP_LINES) + 4)
    sc.clear(x, y, width, height)
    sc.box(x, y, width, height, "Help", ACCENT)
    for index, (text, style) in enumerate(lines):
        sc.put(x + 1, y + 1 + index, trunc(text, width - 2), style)


def draw_prompt(sc, cols, rows, prompt, matches, selected):
    rows_data = [(": " + prompt, ACCENT)]
    for index, command in enumerate(matches):
        marker, style = ("▸ ", ACCENT) if index == selected else ("  ", MUTED)
        rows_data.append((marker + command, style))
    rows_data.append(("enter run · ↑↓ select · esc close", MUTED))
    width, height, x, y = centered(cols, rows, legacy.PROMPT_WIDTH, len(matches) + 5)
    sc.clear(x, y, width, height)
    sc.box(x, y, width, height, "Command", ACCENT)
    for index, (text, style) in enumerate(rows_data):
        sc.put(x + 1, y + 1 + index, trunc(text, width - 2), style)
    cursor = (x + 1 + wlen(": " + prompt), y + 1)
    return cursor


def draw_toast(sc, rows, text):
    sc.put(1, max(0, rows - 2), " " + text + "  ·  any key dismiss", WARN)


def world_line(panes):
    return "host panes %d · endpoints 1 · focus %s" % (panes, VIEW_ID)


def right_status(tabs_n, panes_n, title="main"):
    return "ws:%s tabs:%d panes:%d " % (title, tabs_n, panes_n)


# -------------------------------------------------------------- scenarios

def new_program(cols, rows):
    program = legacy.Program(None, cols=cols, rows=rows, view_id=VIEW_ID)
    program.on_hello({"view_id": VIEW_ID, "epoch": 1, "cols": cols, "rows": rows})
    return program


def press(program, key, char=None):
    program.on_event({"kind": "key", "key": key, "char": char if char is not None else key})


def last_response(program, ok=True, data=None, error=""):
    program.on_response({"request_id": program.request_id, "ok": ok,
                         "data": data or {}, "error": error})


def bind_created(program):
    """Cold start -> picker -> enter -> terminal.create -> bind term-1."""
    program.on_sources([])
    press(program, "enter")
    last_response(program, data={"endpoint": "local", "id": "term-1"})


def source(terminal_id="term-1", exited=False, code=0, attached=True, title=None):
    return {"id": "terminal:local:" + terminal_id, "kind": "terminal",
            "title": title if title is not None else terminal_id,
            "endpoint": "local", "terminal_id": terminal_id,
            "attached": attached, "exited": exited, "exit_code": code}


def bind_attached(program, src):
    """Sources snapshot -> Ctrl-F -> enter -> terminal.attach -> bind."""
    program.on_sources([src])
    press(program, "ctrl-f")
    press(program, "enter")
    last_response(program)


def sc_cold_start(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    draw_picker(sc, cols, rows, ["+ New terminal"], 0)
    return program, sc


def sc_single_pane(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    return program, sc


def sc_split_row(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, "ctrl-p")
    press(program, "%", "%")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 2, "PANE", world_line(0))
    body = rows - 2
    area = cols - 30
    widths = extents(area - 1, [1, 1])
    draw_local_pane(sc, 30, 1, widths[0], body, "unconnected", False, INITIAL_LINES)
    sc.put(30 + widths[0], 1, "│", MUTED)
    draw_local_pane(sc, 31 + widths[0], 1, widths[1], body, "unconnected copy", True, ["split row"])
    draw_footer(sc, cols, rows, PANE_HINTS, right_status(1, 2))
    return program, sc


def sc_split_col(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, "ctrl-p")
    press(program, '"', '"')
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 2, "PANE", world_line(0))
    body = rows - 2
    area = cols - 30
    heights = extents(body - 1, [1, 1])
    draw_local_pane(sc, 30, 1, area, heights[0], "unconnected", False, INITIAL_LINES)
    sc.put(30, 1 + heights[0], "─" * area, MUTED)
    draw_local_pane(sc, 30, 2 + heights[0], area, heights[1], "unconnected copy", True, ["split col"])
    draw_footer(sc, cols, rows, PANE_HINTS, right_status(1, 2))
    return program, sc


def sc_help(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, "?", "?")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    draw_help(sc, cols, rows)
    return program, sc


def sc_prompt_empty(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, ":", ":")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    draw_prompt(sc, cols, rows, "", PROMPT_COMMANDS, 0)
    return program, sc


def sc_prompt_filter(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, ":", ":")
    press(program, "h", "h")
    press(program, "e", "e")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    matches = [command for command in PROMPT_COMMANDS if "he" in command.lower()]
    draw_prompt(sc, cols, rows, "he", matches, 0)
    return program, sc


def sc_tabs_two(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, "ctrl-t")
    sc = Screen(cols, rows)
    draw_header(sc, ["main", "tab2"], 1)
    draw_status(sc, 0, 1, 2, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, ["new tab"])
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(2, 1, title="tab2"))
    return program, sc


def sc_picker_source(cols, rows):
    program = new_program(cols, rows)
    src = source()
    program.on_sources([src])
    press(program, "ctrl-f")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(1))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    draw_picker(sc, cols, rows, ["● local  term-1", "+ New terminal"], 0)
    return program, sc


def exited_screen(cols, rows, code):
    program = new_program(cols, rows)
    src = source()
    bind_attached(program, src)
    program.on_sources([source(exited=True, code=code)])
    press(program, "esc")  # dismiss the bind toast (any key does)
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(1))
    draw_terminal_pane(sc, 30, 1, cols - 30, rows - 2, "term-1", True, True, code, 0)
    draw_footer(sc, cols, rows, EXITED_HINTS, right_status(1, 1))
    return program, sc


def sc_exited_badge(cols, rows):
    return exited_screen(cols, rows, 7)


def sc_exited_zero(cols, rows):
    return exited_screen(cols, rows, 0)


def sc_scrollback(cols, rows):
    program = new_program(cols, rows)
    src = source()
    bind_attached(program, src)
    press(program, "ctrl-p")
    press(program, "page-up")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "PANE", world_line(1))
    draw_terminal_pane(sc, 30, 1, cols - 30, rows - 2, "term-1", False, False, 0, 10)
    draw_footer(sc, cols, rows, SCROLL_HINTS % 10, right_status(1, 1))
    return program, sc


def sc_sidebar_off(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    press(program, "ctrl-w")
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_local_pane(sc, 0, 1, cols, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    return program, sc


def sc_toast(cols, rows):
    program = new_program(cols, rows)
    program.on_sources([])
    press(program, "esc")
    program.on_event({"kind": "notice", "level": "notice", "message": "hi"})
    sc = Screen(cols, rows)
    draw_header(sc, ["main"], 0)
    draw_status(sc, 0, 1, 1, 1, "", world_line(0))
    draw_local_pane(sc, 30, 1, cols - 30, rows - 2, "unconnected", True, INITIAL_LINES)
    draw_footer(sc, cols, rows, NORMAL_HINTS, right_status(1, 1))
    draw_toast(sc, rows, "notice: hi")
    return program, sc


SCENARIOS = [
    ("cold_start_picker", sc_cold_start, True),
    ("single_pane", sc_single_pane, True),
    ("split_row", sc_split_row, True),
    ("split_col", sc_split_col, True),
    ("help", sc_help, True),
    ("prompt_empty", sc_prompt_empty, True),
    ("prompt_filter", sc_prompt_filter, True),
    ("tabs_two", sc_tabs_two, True),
    ("picker_source", sc_picker_source, True),
    ("exited_badge", sc_exited_badge, True),
    ("exited_zero", sc_exited_zero, True),
    ("scrollback", sc_scrollback, True),
    ("sidebar_off", sc_sidebar_off, True),
    ("toast", sc_toast, True),
]


# ------------------------------------------------------------- comparison

def compare(name, cols, rows, actual_lines, actual_styles, expected, program):
    problems = []
    lines = expected.lines()
    styles = expected.styles
    if len(actual_lines) != rows:
        problems.append("screen has %d rows, want %d" % (len(actual_lines), rows))
    for y in range(min(rows, len(actual_lines))):
        if len(actual_lines[y]) != cols:
            problems.append("row %d width %d, want %d" % (y, len(actual_lines[y]), cols))
            continue
        for x in range(cols):
            want_ch = lines[y][x]
            got_ch = actual_lines[y][x]
            if want_ch != got_ch:
                problems.append("row %d col %d: got %r want %r" % (y, x, got_ch, want_ch))
                continue
            if want_ch != " " and styles[y][x] != actual_styles[y][x]:
                problems.append("row %d col %d %r: style got %r want %r"
                                % (y, x, want_ch, actual_styles[y][x], styles[y][x]))
    if name in ("prompt_empty", "prompt_filter"):
        prompt = "" if name == "prompt_empty" else "he"
        matches = PROMPT_COMMANDS if not prompt else [c for c in PROMPT_COMMANDS if prompt in c.lower()]
        _, _, px, py = centered(cols, rows, legacy.PROMPT_WIDTH, len(matches) + 5)
        want_cursor = (px + 1 + wlen(": " + prompt), py + 1)
        if program.cursor != want_cursor:
            problems.append("cursor got %r want %r" % (program.cursor, want_cursor))
    if problems:
        print("FAIL %s %dx%d (%d problems)" % (name, cols, rows, len(problems)))
        for problem in problems[:12]:
            print("  " + problem)
        return False
    print("PASS %s %dx%d" % (name, cols, rows))
    return True


def check_preset():
    """The optional "-config" preset must switch the icons/colors without
    touching the default replica (which the scenarios above pin)."""
    import json as jsonlib
    import subprocess
    import tempfile

    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "legacy.py")
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as handle:
        jsonlib.dump({"preset": "recommended"}, handle)
        config_path = handle.name
    try:
        result = subprocess.run(
            [sys.executable, path, "--selftest", "--config", config_path],
            capture_output=True, text=True,
        )
    finally:
        os.unlink(config_path)
    if result.returncode != 0:
        print("FAIL preset recommended: exit %d (%s)" % (result.returncode, result.stderr.strip()))
        return False
    for glyph in ("\U000F0645", "\U000F0156", "\U000F0415"):
        if glyph not in result.stdout:
            print("FAIL preset recommended: missing glyph U+%04X" % ord(glyph))
            return False
    # The default replica must keep the legacy ASCII-ish glyphs.
    if " WS main " in result.stdout:
        print("FAIL preset recommended: legacy workspace label survived")
        return False
    print("PASS preset recommended")
    return True


def check_endpoints():
    """Configured endpoints: a v1 command endpoint stays usable and a daemon
    endpoint turns on picker grouping, while the default replica (pinned by
    the scenarios above) keeps no endpoint rows at all."""
    import json as jsonlib
    import subprocess
    import tempfile

    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "legacy.py")
    config = {"endpoints": [
        {"name": "localcmd", "kind": "command", "label": "local-cmd", "argv": ["sh"]},
        {"name": "dev", "kind": "daemon", "label": "dev-daemon", "socket": "/tmp/does-not-exist.sock"},
    ]}
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as handle:
        jsonlib.dump(config, handle)
        config_path = handle.name
    try:
        grouped = subprocess.run(
            [sys.executable, path, "--selftest", "--config", config_path],
            capture_output=True, text=True,
        )
        default = subprocess.run(
            [sys.executable, path, "--selftest"],
            capture_output=True, text=True,
        )
    finally:
        os.unlink(config_path)
    if grouped.returncode != 0 or default.returncode != 0:
        print("FAIL endpoints: exit grouped=%d default=%d (%s)"
              % (grouped.returncode, default.returncode, grouped.stderr.strip()))
        return False
    for needle in ("  dev", "dev-daemon", "local-cmd", "+ New terminal", "  local"):
        if needle not in grouped.stdout:
            print("FAIL endpoints: grouped picker missing %r" % needle)
            return False
    if "dev-daemon" in default.stdout or "local-cmd" in default.stdout:
        print("FAIL endpoints: default replica must not render configured endpoints")
        return False
    print("PASS endpoints grouped picker")
    return True


def check_recommended_footer(golden_dir):
    """The recommended preset must render the same exact footer rows as the
    Go shell (golden/recommended_footer_120x32.txt), one line per v2 scene."""
    import json as jsonlib
    import subprocess
    import tempfile

    golden_path = os.path.join(golden_dir, "recommended_footer_120x32.txt")
    if not os.path.exists(golden_path):
        print("FAIL recommended footer: missing golden %s" % golden_path)
        return False
    golden = {}
    with open(golden_path, "r", encoding="utf-8") as handle:
        for line in handle:
            line = line.rstrip("\n")
            if not line or line.startswith("#"):
                continue
            name, rest = line.split("|", 1)
            golden[name] = rest[:-1] if rest.endswith("|") else rest

    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "legacy.py")
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as handle:
        jsonlib.dump({"preset": "recommended"}, handle)
        config_path = handle.name
    try:
        result = subprocess.run(
            [sys.executable, path, "--footer-lines", "--config", config_path],
            capture_output=True, text=True,
        )
    finally:
        os.unlink(config_path)
    if result.returncode != 0:
        print("FAIL recommended footer: exit %d (%s)" % (result.returncode, result.stderr.strip()))
        return False
    got = {}
    for line in result.stdout.splitlines():
        name, rest = line.split("|", 1)
        got[name] = rest[:-1] if rest.endswith("|") else rest
    failures = 0
    for name in sorted(golden):
        if got.get(name) != golden[name]:
            print("FAIL recommended footer %s:\n  got  %r\n  want %r" % (name, got.get(name), golden[name]))
            failures += 1
    if set(got) != set(golden):
        print("FAIL recommended footer scenes: got %s want %s" % (sorted(got), sorted(golden)))
        failures += 1
    if failures:
        return False
    print("PASS recommended footer (%d scenes)" % len(golden))
    return True


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument("--golden-dir", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "golden"))
    parser.add_argument("--only", default="")
    parser.add_argument("--viewports", default="120x32,100x30")
    options = parser.parse_args(argv)

    viewports = []
    for spec in options.viewports.split(","):
        cols, _, rows = spec.partition("x")
        viewports.append((int(cols), int(rows)))

    failures = 0
    total = 0
    os.makedirs(options.golden_dir, exist_ok=True)
    for cols, rows in viewports:
        for name, builder, golden in SCENARIOS:
            if options.only and options.only != name:
                continue
            program, expected = builder(cols, rows)
            actual_lines, actual_styles = program.screen()
            total += 1
            if not compare(name, cols, rows, actual_lines, actual_styles, expected, program):
                failures += 1
            if golden:
                path = os.path.join(options.golden_dir, "%s_%dx%d.txt" % (name, cols, rows))
                with open(path, "w") as handle:
                    handle.write("\n".join(line.rstrip() for line in expected.lines()) + "\n")
    if not options.only:
        total += 1
        if not check_preset():
            failures += 1
        total += 1
        if not check_endpoints():
            failures += 1
        total += 1
        if not check_recommended_footer(options.golden_dir):
            failures += 1
    print("parity: %d scenarios, %d failed" % (total, failures))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
