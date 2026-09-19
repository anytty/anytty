#!/usr/bin/env python3
"""Pixel-parity checks for v3ui.py against the legacy v3 render formulas.

Two independent computations are compared cell by cell:

  - expected: this file re-implements the old tui/render formulas from the
    spec (clients/tui/docs/V3_PARITY.zh-CN.md): recommended header templates,
    pane card chrome (``paneChromeTopSlots`` / ``paneChromeTerminalLabelSlots``
    / bracket action group), footer selection + trim (``shell_bar.go``) and
    the ``1.txt`` capture rows as literal oracles;
  - actual: the view produced by ``v3ui.py``'s demo state machine, rasterized
    with the v2 kernel semantics (pos blits, truncation).

Covered:
  - 120x32 golden chrome (header / window frame + title + actions / pane
    content split / footer) plus the full-screen golden;
  - 181x56 against the literal rows of ``1.txt`` (frame + footer, and the
    3-tab header variant);
  - every recommended footer scene (live/pane/resize/tab/system/prompt/
    picker/help);
  - split chrome (stacked Ctrl-E / side-by-side Ctrl-D card row + divider)
    and the floating window title/action row with its collapsed form;
  - click/action id coverage for the card buttons.

Usage:
  python3 v3_parity_test.py [--golden-dir DIR] [--only NAME]
"""

import argparse
import os
import sys
import unicodedata

import v3ui

VIEW_ID = "view:test:v3"

# Independent copies of the recommended-coralline glyphs (yaml).
WS_ICON = "\U000f0645"
TAB_ICON = "\u2387"
CLOSE = "\U000f0156"
CREATE = "\U000f0415"
EDGE_L = "\ue0b6"
EDGE_R = "\ue0b0"
EDGE_ROUND_R = "\ue0b4"
LOCK = "\u25a1"
ZOOM = "\U000f004c"
SPLIT_V = "\ueb56"
SPLIT_H = "\ueb57"
RUNNING = "\u25cf"
FLOAT_SUMMARY = "\U000f0e59"
TERM_SUMMARY = "\uf489"
MODE_ICON = "\U000f030c"
CENTER = "\u25ce"
COLLAPSE = "\u25be"

FOOTER_SEPARATOR = " \u00b7 "

LIVE_ACTIONS = [
    "P \uebeb PANE",
    "R \U000f0656 SIZE",
    "O \U000f0e59 FLOAT",
    "T \U000f04e9 TAB",
    "W \U000f0645 WORKSPACE",
    "F \U000f0c7c PICK",
    "\u21e7C \U000f0489 SELECT",
    "\u21e7H \U000f014c CLIPBOARD",
    "\u21e7V \U000f018f PASTE",
    "G \U000f0493 SYSTEM",
]


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
        ch_cells = 0 if unicodedata.combining(ch) else (
            2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1)
        if used + ch_cells > cells and ch_cells:
            break
        out.append(ch)
        used += ch_cells
    return "".join(out)


def center_pad(text, cells):
    text = trunc(text, cells)
    pad = cells - wlen(text)
    left = pad // 2
    return " " * left + text + " " * (pad - left)


def cell_at(row, x):
    """The character whose display cells cover column ``x`` ("" on a wide
    continuation cell, " " past the end)."""
    cursor = 0
    for ch in row:
        cells = 0 if unicodedata.combining(ch) else (
            2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1)
        if cells == 0:
            continue
        if cursor == x:
            return ch
        if cursor + cells > x:
            return ""
        cursor += cells
    return " "


def cells_slice(row, x, width):
    """The substring covering display cells [x, x+width)."""
    out = []
    cursor = 0
    end = x + width
    for ch in row:
        cells = 0 if unicodedata.combining(ch) else (
            2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1)
        if cells == 0:
            continue
        if cursor >= end:
            break
        if cursor >= x:
            out.append(ch)
        cursor += cells
    return "".join(out)


# ------------------------------------------------------- expected chrome

def expected_header(workspace, tabs, active):
    """coralline-candy workspace/tab/create templates (yaml)."""
    out = EDGE_L + " " + WS_ICON + " " + workspace + " " + EDGE_R
    for index, title in enumerate(tabs):
        if index == active:
            out += EDGE_R + " " + TAB_ICON + " " + trunc(title, 14) + " \u00b7 T" + str(index + 1) + " " + EDGE_R
            out += " " + CLOSE + " "
        else:
            out += EDGE_R + " " + TAB_ICON + " " + trunc(title, 14) + " \u00b7 T" + str(index + 1) + " " + EDGE_R
            out += " " + CLOSE + " "
    out += EDGE_R + " " + CREATE + " " + EDGE_ROUND_R
    return out


def expected_card_row(width, title, owner="owner", attach=1, active=True):
    """paneChromeTopSlots + paneChromeTerminalLabelSlots + action group."""
    actions = [ZOOM, SPLIT_V, SPLIT_H, CLOSE]
    cluster = EDGE_L + "".join(" " + glyph + " " for glyph in actions) + EDGE_ROUND_R
    action_width = wlen(cluster)
    inner_right = width - 1
    action_x = inner_right - action_width - 1
    right_limit = action_x - 1
    right = center_pad(RUNNING, 3) + center_pad("x" + str(attach), 4)
    right += center_pad(owner, max(8, wlen(owner)))
    prefix = " " + LOCK + " "
    title_width = right_limit - 2 - wlen(prefix) - wlen(right)
    title_text = (" " + trunc(title, title_width - 2) + " ") if title_width > 2 else ""
    gap = title_width - wlen(title_text)
    row = "\u250c\u2500" + prefix + title_text + "\u2500" * max(0, gap) + right
    row += "\u2500" * max(0, action_x - wlen(row)) + cluster
    tail = width - wlen(row)
    return row + "\u2500" * max(0, tail - 1) + "\u2510"


def expected_card_row_empty(width, title, active=True):
    """pane_runs for a pane without a terminal source (split placeholder)."""
    right = center_pad(" active " if active else " idle ", 12)
    prefix = " " + LOCK + " "
    right_limit = width - 1
    left_width = max(0, right_limit - 2)
    title_width = max(0, left_width - wlen(prefix) - wlen(right))
    title_text = ""
    if title:
        if title_width > 2:
            title_text = " " + trunc(title, title_width - 2) + " "
        elif title_width > 0:
            title_text = trunc(title, title_width)
    row = "\u250c\u2500" + prefix + title_text
    gap = title_width - wlen(title_text)
    row += "\u2500" * max(0, gap) + right
    tail = width - wlen(row)
    return row + "\u2500" * max(0, tail - 1) + "\u2510"


def split_first(avail, ratio=0.5):
    """v3ui.layout_node's exact extent rounding (half-up, >=1 each)."""
    return max(1, min(avail - 1, int(avail * ratio + 0.5)))


def expected_split_row(width, left_row, right_row):
    """A recursive row split at ratio 0.5: card | 1-cell gap | card."""
    avail = width - 5
    first = split_first(avail)
    left = expected_card_row(first + 2, left_row, "owner", active=True)
    right = expected_card_row(width - first - 3, right_row, "follow",
                              active=False)
    return left + "\u2502" + right


def expected_float_row(width, title):
    """floating_runs: lock + title + bracket action group, no pane splits."""
    actions = [CENTER, COLLAPSE, ZOOM, CLOSE]
    action_width = 2 + 3 * len(actions)
    inner_right = width - 1
    action_x = inner_right - action_width - 1
    right_limit = action_x - 1
    lock = " " + LOCK + " "
    title = trunc(title, max(0, right_limit - 2 - wlen(lock)))
    title_text = (" " + title + " ") if title else ""
    row = "\u250c\u2500" + lock + title_text
    x = 2 + wlen(lock) + wlen(title_text)
    if x < right_limit:
        row += "\u2500" * (right_limit - x)
        x = right_limit
    row += "\u2500" * max(0, action_x - x)
    row += EDGE_L + "".join(" " + glyph + " " for glyph in actions) + EDGE_ROUND_R
    tail = width - wlen(row)
    return row + "\u2500" * max(0, tail - 1) + "\u2510"


def footer_token_width(label):
    return 1 + wlen(label)


def select_footer_actions(actions, limit):
    """shell_bar.selectFooterActionTokens with the yaml key template ""."""
    selected = []
    used = 0
    truncated = False
    for label in actions:
        token_width = footer_token_width(label)
        if selected:
            token_width += len(FOOTER_SEPARATOR)
        if selected and used + token_width > limit:
            truncated = True
            break
        if not selected and token_width > limit:
            truncated = True
            break
        selected.append(label)
        used += token_width
    if not truncated or len(selected) == len(actions):
        return selected
    tail = actions[-1]
    tail_width = footer_token_width(tail) + (len(FOOTER_SEPARATOR) if selected else 0)
    while selected and used + tail_width > limit:
        dropped = selected.pop()
        used -= footer_token_width(dropped)
        if selected:
            used -= len(FOOTER_SEPARATOR)
        tail_width = footer_token_width(tail) + (len(FOOTER_SEPARATOR) if selected else 0)
    if tail_width <= limit and used + tail_width <= limit:
        selected.append(tail)
    return selected


def expected_live_footer(cols, workspace, floats, terminals):
    actions = select_footer_actions(LIVE_ACTIONS, cols - (wlen(" " + WS_ICON + " " + workspace)
                                                          + wlen(" " + FLOAT_SUMMARY + " " + str(floats))
                                                          + wlen(" " + TERM_SUMMARY + " " + str(terminals))
                                                          + 1) if cols >= 120 else cols)
    left = " " + MODE_ICON + " CTRL "
    for label in actions:
        left += FOOTER_SEPARATOR + " " + label
    right = " " + WS_ICON + " " + workspace
    right += " " + FLOAT_SUMMARY + " " + str(floats)
    right += " " + TERM_SUMMARY + " " + str(terminals) + " "
    # trim right by segment priority (ws=2, float=1, terminals=4, tail=4)
    segments = [(" " + WS_ICON + " " + workspace, 2),
                (" " + FLOAT_SUMMARY + " " + str(floats), 1),
                (" " + TERM_SUMMARY + " " + str(terminals), 4),
                (" ", 4)]
    while sum(wlen(text) for text, _ in segments) > cols - wlen(left) and segments:
        index = 0
        for i, (_text, priority) in enumerate(segments):
            if priority >= segments[index][1]:
                index = i
        segments.pop(index)
    right = "".join(text for text, _ in segments)
    pad = max(0, cols - wlen(left) - wlen(right))
    return left + " " * pad + right


# -------------------------------------------------------------- scenarios

def demo(cols, rows):
    program = v3ui.Program(None, cols=cols, rows=rows, demo=True)
    program.view_id = VIEW_ID
    return program


def sc_demo_live(cols, rows):
    program = demo(cols, rows)
    expected_header_line = expected_header("main", ["auto-push", "local"], 0)
    expected_card = expected_split_row(
        cols, "anytty-surface@hs", "opencode@hs")
    expected_footer = expected_live_footer(cols, "main", 0, 11)
    return program, expected_header_line, expected_card, expected_footer


def sc_demo_tab2(cols, rows):
    program = demo(cols, rows)
    program.active = 1
    expected_header_line = expected_header("main", ["auto-push", "local"], 1)
    expected_card = expected_card_row(cols, "local@hs", "follow")
    expected_footer = expected_live_footer(cols, "main", 0, 11)
    return program, expected_header_line, expected_card, expected_footer


def sc_capture_frame(cols, rows):
    """3-tab variant whose header/frame rows must equal 1.txt literally.

    1.txt shows a single window frame, which under the card-per-leaf model is
    the single-leaf state (the old single-card replica was a documented
    deviation); the active tab therefore closes its right leaf first.
    """
    program = demo(cols, rows)
    tab = program.active_tab()
    program.close_pane(tab, tab.panes[1])
    program.tab_seq = 3
    pane = program.new_pane("empty")
    program.tabs.append(v3ui.Tab("tab-3", "tab 3", [pane], "row"))
    program.active = 0
    return program


def capture_oracle():
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "1.txt")
    with open(path, encoding="utf-8") as handle:
        return handle.read().split("\n")


# ------------------------------------------------------------- comparison

def compare_rows(name, expected_rows, actual_lines, cols, exact_rows):
    problems = []
    if len(actual_lines) != len(expected_rows):
        problems.append("screen has %d rows, want %d" % (len(actual_lines), len(expected_rows)))
    for y in range(min(len(actual_lines), len(expected_rows))):
        if len(actual_lines[y]) != cols:
            problems.append("row %d width %d, want %d" % (y, len(actual_lines[y]), cols))
            continue
        want = expected_rows[y]
        got = actual_lines[y]
        if exact_rows is not None and y not in exact_rows:
            # Content area: placeholder assertion (first content cell only).
            continue
        if got.rstrip() != want.rstrip():
            for x in range(cols):
                if x >= len(got) or x >= len(want):
                    break
                if got[x] != want[x]:
                    problems.append("row %d col %d: got %r want %r"
                                    % (y, x, got[x], want[x]))
                    break
            else:
                problems.append("row %d: trailing content differs" % y)
    return problems


def check_demo(name, builder, cols, rows, golden_dir, write_golden):
    program, header, card, footer = builder(cols, rows)
    actual, _styles = program.screen()
    problems = []
    if actual[0].rstrip() != header.rstrip():
        problems.append("header:\n  got  %r\n  want %r" % (actual[0].rstrip(), header))
    if actual[1].rstrip() != card.rstrip():
        problems.append("card:\n  got  %r\n  want %r" % (actual[1].rstrip(), card))
    if actual[rows - 1].rstrip() != footer.rstrip():
        problems.append("footer:\n  got  %r\n  want %r" % (actual[rows - 1].rstrip(), footer))
    # Window frame: left/right rules on every content row (wide characters
    # shrink the joined string, so check the edge codepoints instead of cells).
    for y in range(2, rows - 2):
        if actual[y] and (actual[y][0] != "\u2502" or actual[y][-1] != "\u2502"):
            problems.append("row %d frame: %r" % (y, actual[y]))
            break
    if write_golden:
        if name == "demo_live":
            filename = "v3_ui_%dx%d.txt" % (cols, rows)
        else:
            filename = "v3_ui_%s_%dx%d.txt" % (name, cols, rows)
        path = os.path.join(golden_dir, filename)
        with open(path, "w", encoding="utf-8") as handle:
            handle.write("\n".join(line.rstrip() for line in actual) + "\n")
    if problems:
        print("FAIL %s %dx%d" % (name, cols, rows))
        for problem in problems[:6]:
            print("  " + problem)
        return False
    print("PASS %s %dx%d" % (name, cols, rows))
    return True


def check_capture(cols, rows):
    """The literal 1.txt rows are an independent oracle for a 3-tab state."""
    program = sc_capture_frame(cols, rows)
    actual, _styles = program.screen()
    oracle = capture_oracle()
    if len(oracle) < rows:
        print("FAIL capture_1txt: 1.txt has %d rows, want %d" % (len(oracle), rows))
        return False
    problems = []
    for y in (0, 1, rows - 1):
        got = actual[y].rstrip()
        want = oracle[y].rstrip()
        if got != want:
            for x in range(min(len(got), len(want))):
                if got[x] != want[x]:
                    problems.append("row %d col %d: got %r want %r" % (y, x, got[x], want[x]))
                    break
            else:
                problems.append("row %d length: got %d want %d" % (y, len(got), len(want)))
    # 1.txt's active tab is T1; the 3-tab header must render exactly.
    if actual[0].rstrip() != oracle[0].rstrip():
        problems.append("3-tab header must equal 1.txt row 1")
    if problems:
        print("FAIL capture_1txt %dx%d" % (cols, rows))
        for problem in problems[:6]:
            print("  " + problem)
        return False
    print("PASS capture_1txt %dx%d" % (cols, rows))
    return True


def write_golden(name, actual, cols, rows, golden_dir):
    path = os.path.join(golden_dir, "v3_%s_%dx%d.txt" % (name, cols, rows))
    with open(path, "w", encoding="utf-8") as handle:
        handle.write("\n".join(line.rstrip() for line in actual) + "\n")


def check_split_chrome(name, flow, cols, rows, golden_dir):
    """Card-per-leaf split chrome: two full cards + one draggable gap.

    ``flow="col"`` is the stacked split (Ctrl-E), ``"row"`` the side-by-side
    one (Ctrl-D); the new empty leaf keeps focus in the ``b`` half.
    """
    program = demo(cols, rows)
    tab = program.active_tab()
    program.close_pane(tab, tab.panes[1])
    program.split_pane(flow)
    actual, _styles = program.screen()
    focus = program.focus_pane()
    body_h = rows - 2
    problems = []
    if flow == "col":
        first = split_first(body_h - 5)
        top_h = first + 2
        sep_y = 1 + top_h
        bottom_y = sep_y + 1
        want_top = expected_card_row(cols, "anytty-surface@hs", "owner",
                                     active=False)
        want_bottom = expected_card_row_empty(cols, focus.id)
        if actual[1].rstrip() != want_top.rstrip():
            problems.append("top card:\n  got  %r\n  want %r"
                            % (actual[1].rstrip(), want_top.rstrip()))
        want_divider = "\u2500" * cols
        if actual[sep_y].rstrip() != want_divider.rstrip():
            problems.append("divider row %d:\n  got  %r\n  want %r"
                            % (sep_y, actual[sep_y].rstrip(),
                               want_divider.rstrip()))
        if actual[bottom_y].rstrip() != want_bottom.rstrip():
            problems.append("bottom card:\n  got  %r\n  want %r"
                            % (actual[bottom_y].rstrip(),
                               want_bottom.rstrip()))
        want_bottom_border = "\u2514" + "\u2500" * (cols - 2) + "\u2518"
        for y, label in ((top_h, "top"), (body_h, "bottom")):
            if actual[y].rstrip() != want_bottom_border.rstrip():
                problems.append("%s card bottom row %d: %r"
                                % (label, y, actual[y]))
        for y in (2, top_h - 1, bottom_y + 1):
            if actual[y] and (actual[y][0] != "\u2502"
                              or actual[y][-1] != "\u2502"):
                problems.append("card frame row %d: %r" % (y, actual[y]))
                break
    else:
        first = split_first(cols - 5)
        left_w = first + 2
        want_left = expected_card_row(left_w, "anytty-surface@hs", "owner",
                                      active=False)
        want_right = expected_card_row_empty(cols - left_w - 1, focus.id)
        want = want_left + "\u2502" + want_right
        if actual[1].rstrip() != want.rstrip():
            problems.append("split row:\n  got  %r\n  want %r"
                            % (actual[1].rstrip(), want.rstrip()))
        want_bottom_border = ("\u2514" + "\u2500" * (left_w - 2) + "\u2518"
                              + "\u2502"
                              + "\u2514" + "\u2500" * (cols - left_w - 3)
                              + "\u2518")
        if actual[body_h].rstrip() != want_bottom_border.rstrip():
            problems.append("card bottoms row %d: %r" % (body_h, actual[body_h]))
        for y in (2, body_h - 1):
            if actual[y][0] != "\u2502" or actual[y][left_w - 1] != "\u2502":
                problems.append("left card frame row %d: %r" % (y, actual[y]))
                break
            if actual[y][left_w] != "\u2502" or actual[y][left_w + 1] != "\u2502":
                problems.append("divider/right frame row %d: %r" % (y, actual[y]))
                break
            if actual[y][cols - 1] != "\u2502":
                problems.append("right card frame row %d: %r" % (y, actual[y]))
                break
    write_golden(name, actual, cols, rows, golden_dir)
    if problems:
        print("FAIL %s %dx%d" % (name, cols, rows))
        for problem in problems[:6]:
            print("  " + problem)
        return False
    print("PASS %s %dx%d" % (name, cols, rows))
    return True


def check_left1right2(viewports, golden_dir):
    """Left 1 / right 2: split row, then split the right leaf column-wise.

    Every leaf is a full card; the expected rects and every chrome row come
    from this file's own layout/chrome formulas, not from v3ui.
    """
    failed = False
    for cols, rows in viewports:
        program = demo(cols, rows)
        tab = program.active_tab()
        program.close_pane(tab, tab.panes[1])
        program.split_pane("row")
        program.split_pane("col")
        actual, _styles = program.screen()
        problems = []
        body_h = rows - 2
        first = split_first(cols - 5)
        left_w = first + 2
        right_x = first + 3
        right_w = cols - first - 3
        top_first = split_first(body_h - 5)
        top_h = top_first + 2
        expected_rects = [
            (0, 1, left_w, body_h),
            (right_x, 1, right_w, top_h),
            (right_x, 1 + top_h + 1, right_w, body_h - top_h - 1),
        ]
        entries = [entry for entry in program.pane_rects(tab)[1]
                   if not isinstance(entry[0], v3ui.Split)]
        got_rects = [entry[1:] for entry in entries]
        if got_rects != expected_rects:
            problems.append("leaf rects:\n  got  %s\n  want %s"
                            % (got_rects, expected_rects))
        want_left = expected_card_row(left_w, "anytty-surface@hs", "owner",
                                      active=False)
        if cells_slice(actual[1], 0, left_w) != want_left:
            problems.append("left card row:\n  got  %r\n  want %r"
                            % (cells_slice(actual[1], 0, left_w), want_left))
        want_top = expected_card_row_empty(right_w, entries[1][0].id,
                                           active=False)
        if cells_slice(actual[1], right_x, right_w) != want_top:
            problems.append("right-top card row:\n  got  %r\n  want %r"
                            % (cells_slice(actual[1], right_x, right_w),
                               want_top))
        bottom_y = 1 + top_h + 1
        want_bottom = expected_card_row_empty(right_w, entries[2][0].id)
        if cells_slice(actual[bottom_y], right_x, right_w) != want_bottom:
            problems.append("right-bottom card row:\n  got  %r\n  want %r"
                            % (cells_slice(actual[bottom_y], right_x, right_w),
                               want_bottom))
        sep_y = 1 + top_h
        if cells_slice(actual[sep_y], right_x, right_w) != "\u2500" * right_w:
            problems.append("right horizontal divider row %d: %r"
                            % (sep_y, actual[sep_y]))
        for y in (2, bottom_y + 1):
            if actual[y][0] != "\u2502" or actual[y][left_w - 1] != "\u2502":
                problems.append("left card frame row %d: %r" % (y, actual[y]))
                break
            if actual[y][left_w] != "\u2502":
                problems.append("row divider column %d at row %d" % (left_w, y))
                break
        write_golden("left1right2", actual, cols, rows, golden_dir)
        if problems:
            failed = True
            print("FAIL left1right2 %dx%d" % (cols, rows))
            for problem in problems[:6]:
                print("  " + problem)
        else:
            print("PASS left1right2 %dx%d" % (cols, rows))
    return not failed


def mix_color(base, overlay, ratio):
    def parse(value):
        return int(value[1:3], 16), int(value[3:5], 16), int(value[5:7], 16)
    mixed = [int(a * (1 - ratio) + b * ratio + 0.5)
             for a, b in zip(parse(base), parse(overlay))]
    return "#%02x%02x%02x" % tuple(mixed)


COLOR_ACCENT = "#f0abfc"
COLOR_SUCCESS = "#86efac"
COLOR_WARNING = "#fde68a"
COLOR_DANGER = "#fb7185"
COLOR_INFO = "#7dd3fc"
COLOR_MUTED = "#9ca3c9"

# ansiForStyleToken() with the coralline-candy tokens (old tui/render).
FOOTER_STYLES = {
    "footer-accent": "fg:#f0abfc;bold",
    "footer-key-pane": "fg:#f0abfc;bold",
    "footer-key-resize": "fg:#fde68a;bold",
    "footer-key-tab": "fg:#7dd3fc;bold",
    "footer-key-workspace": "fg:#86efac;bold",
    "footer-key-float": "fg:%s;bold" % mix_color(COLOR_ACCENT, COLOR_INFO, 0.45),
    "footer-key-copy": "fg:%s;bold" % mix_color(COLOR_SUCCESS, COLOR_INFO, 0.35),
    "footer-key-picker": "fg:%s;bold" % mix_color(COLOR_DANGER, COLOR_WARNING, 0.35),
    "footer-key-global": "fg:%s;bold" % mix_color(COLOR_ACCENT, COLOR_WARNING, 0.35),
    "info": "fg:#7dd3fc",
    "success": "fg:#86efac",
    "danger": "fg:#fb7185",
    "danger-strong": "fg:#fb7185;bold",
    "muted": "fg:#9ca3c9",
}

# yaml `shortcuts.actions` style overrides.
YAML_ACTION_STYLES = {
    "menu.panel": "footer-key-pane",
    "menu.resize": "footer-key-resize",
    "menu.floating": "footer-key-float",
    "menu.tab": "footer-key-tab",
    "menu.workspace": "footer-key-workspace",
    "menu.clipboard_history": "footer-key-copy",
    "menu.system": "footer-key-global",
    "panel.close": "danger",
    "panel.kill_and_close": "danger-strong",
    "clipboard.paste_system": "info",
}


def footer_heuristic_style(key, label):
    """footerActionKeyStyle(key, label) token heuristic."""
    upper = (key + " " + label).upper()
    if "X" in upper or "CLOSE" in upper or "KILL" in upper:
        return "footer-key-picker"
    if "W" in upper or "WORKSPACE" in upper:
        return "footer-key-workspace"
    if "F" in upper or "PICK" in upper:
        return "footer-key-picker"
    if "O" in upper or "FLOAT" in upper:
        return "footer-key-float"
    if "V" in upper or "COPY" in upper:
        return "footer-key-copy"
    if "G" in upper or "GLOBAL" in upper:
        return "footer-key-global"
    if "R" in upper or "RESIZE" in upper or "SIZE" in upper:
        return "footer-key-resize"
    if "P" in upper or "PANE" in upper:
        return "footer-key-pane"
    if "T" in upper or "TAB" in upper or "TREE" in upper:
        return "footer-key-tab"
    return "footer-accent"


def footer_expected_style(action_id, key, label):
    token = YAML_ACTION_STYLES.get(action_id)
    if token is None:
        token = footer_heuristic_style(key, label)
    return FOOTER_STYLES[token]


LIVE_SPECS = [
    ("P \uebeb PANE", "menu.panel", "^P"),
    ("R \U000f0656 SIZE", "menu.resize", "^R"),
    ("O \U000f0e59 FLOAT", "menu.floating", "^O"),
    ("T \U000f04e9 TAB", "menu.tab", "^T"),
    ("W \U000f0645 WORKSPACE", "menu.workspace", "^W"),
    ("F \U000f0c7c PICK", "terminal_picker.open", "^F"),
    ("\u21e7C \U000f0489 SELECT", "copy.enter", "^C"),
    ("\u21e7H \U000f014c CLIPBOARD", "menu.clipboard_history", "^H"),
    ("\u21e7V \U000f018f PASTE", "clipboard.paste_system", "^V"),
    ("G \U000f0493 SYSTEM", "menu.system", "^G"),
]

PANE_SPECS = [
    ("X \U000f0156 CLOSE", "panel.close", "x"),
    ("CTRL+D \ueb56 VSPLIT", "panel.split_right", "^D"),
    ("CTRL+E \ueb57 HSPLIT", "panel.split_down", "^E"),
    ("H/L \U000f0734 FOCUS", "panel.focus_prev", "h"),
    ("Q \U000f0688 KILL+CLOSE", "panel.kill_and_close", "q"),
]

PICKER_SPECS = [
    ("\u2190/\u2192 ENDPOINT", "terminal_picker.endpoint_previous", "\u2190"),
    ("\u2191/\u2193 SELECT", "terminal_picker.select_previous", "\u2191"),
    ("ENTER \U000f02fa ATTACH", "terminal_picker.attach", "enter"),
    ("ESC BACK", "shortcut.exit", "Esc"),
]

SCENE_SPECS = {
    "live": (MODE_ICON, "CTRL", "footer-accent", LIVE_SPECS),
    "pane": ("\uebeb", "PANE", "footer-key-pane", PANE_SPECS),
    "terminal-picker": ("\U000f0c7c", "PICK", "footer-key-picker", PICKER_SPECS),
}


def select_footer_specs(specs, limit):
    selected = []
    used = 0
    truncated = False
    for spec in specs:
        token_width = 1 + wlen(spec[0])
        if selected:
            token_width += len(FOOTER_SEPARATOR)
        if selected and used + token_width > limit:
            truncated = True
            break
        if not selected and token_width > limit:
            truncated = True
            break
        selected.append(spec)
        used += token_width
    if not truncated or len(selected) == len(specs):
        return selected
    tail = specs[-1]
    tail_width = 1 + wlen(tail[0]) + (len(FOOTER_SEPARATOR) if selected else 0)
    while selected and used + tail_width > limit:
        dropped = selected.pop()
        used -= 1 + wlen(dropped[0])
        if selected:
            used -= len(FOOTER_SEPARATOR)
        tail_width = 1 + wlen(tail[0]) + (len(FOOTER_SEPARATOR) if selected else 0)
    if tail_width <= limit and used + tail_width <= limit:
        selected.append(tail)
    return selected


def expected_footer_color_runs(scene, cols, workspace="main", floats=0,
                               terminals=11):
    icon, label, badge_token, specs = SCENE_SPECS[scene]
    runs = []
    if icon or label:
        badge = (" " + icon + " " + label + " ") if label else (" " + icon + " ")
        runs.append((badge, FOOTER_STYLES[badge_token], 1))
    right = [(" " + WS_ICON + " " + workspace, FOOTER_STYLES["muted"], 2),
             (" " + FLOAT_SUMMARY + " " + str(floats),
              FOOTER_STYLES["footer-accent"], 1),
             (" " + TERM_SUMMARY + " " + str(terminals),
              FOOTER_STYLES["muted"], 4),
             (" ", FOOTER_STYLES["muted"], 4)]
    reserve = sum(wlen(text) for text, _style, _priority in right)
    limit = cols
    if cols >= 120 and scene == "live":
        limit = max(0, cols - reserve)
    for spec in select_footer_specs(specs, limit):
        if runs:
            runs.append((FOOTER_SEPARATOR, FOOTER_STYLES["muted"], 1))
        runs.append((" " + spec[0],
                     footer_expected_style(spec[1], spec[2], spec[0]), 1))
    return runs, right


def trim_footer_runs(runs, width):
    """shell_bar.trimBarSegments: drop the highest-priority-number segment."""
    out = list(runs)
    while sum(wlen(text) for text, _style, _priority in out) > width and out:
        index = 0
        for i, (_text, _style, priority) in enumerate(out):
            if priority >= out[index][2]:
                index = i
        del out[index]
    return out


def check_footer_colors(viewports, golden_dir):
    """Every footer key block carries its own old-v3 style token color.

    The expected run table is rebuilt here from the yaml overrides + the
    footerActionKeyStyle() heuristic; the screen is then compared cell by
    cell for every non-space character, and the distinct action color counts
    are pinned per scene (NORMAL/PANE/PICKER).
    """
    failed = False
    for cols, rows in viewports:
        program = demo(cols, rows)
        scenarios = [("LIVE", "live", "live", ""),
                     ("PANE", "pane", "pane", ""),
                     ("PICKER", "terminal-picker", "live", "picker")]
        golden_rows = []
        problems = []
        for name, scene, mode, overlay in scenarios:
            program.mode = mode
            program.overlay = overlay
            left_runs, right_runs = expected_footer_color_runs(scene, cols)
            left_runs = trim_footer_runs(left_runs, cols)
            left_width = sum(wlen(text) for text, _style, _priority in left_runs)
            right_runs = trim_footer_runs(right_runs, cols - left_width)
            lines, styles = program.screen()
            y = rows - 1
            x = 0
            action_styles = set()
            for index, (text, style, _priority) in enumerate(left_runs):
                for offset, ch in enumerate(text):
                    if ch != " " and styles[y][x + offset] != style:
                        problems.append(
                            "%s col %d %r: style %r want %r"
                            % (name, x + offset, ch, styles[y][x + offset],
                               style))
                if index > 0 and style != FOOTER_STYLES["muted"]:
                    action_styles.add(style)
                x += wlen(text)
                golden_rows.append("%s|%d|%s|%s|" % (name, len(golden_rows),
                                                     style, text))
            right_width = sum(wlen(text) for text, _style, _priority
                              in right_runs)
            x = left_width + max(0, cols - left_width - right_width)
            for text, style, _priority in right_runs:
                for offset, ch in enumerate(text):
                    if ch != " " and styles[y][x + offset] != style:
                        problems.append(
                            "%s right col %d %r: style %r want %r"
                            % (name, x + offset, ch, styles[y][x + offset],
                               style))
                x += wlen(text)
            if name == "LIVE":
                want_distinct = 7 if cols < 181 else 9
                if len(action_styles) != want_distinct:
                    problems.append("LIVE distinct action colors %d, want %d"
                                    % (len(action_styles), want_distinct))
                if cols >= 181:
                    copy_style = FOOTER_STYLES["footer-key-copy"]
                    if copy_style not in action_styles:
                        problems.append("LIVE missing CLIPBOARD copy color")
            elif name == "PANE":
                if len(action_styles) != 5:
                    problems.append("PANE distinct action colors %d, want 5"
                                    % len(action_styles))
            else:
                want = {FOOTER_STYLES[token]
                        for token in ("footer-key-float", "footer-key-tab",
                                      "footer-key-resize", "footer-accent")}
                if action_styles != want:
                    problems.append("PICKER action colors %s, want %s"
                                    % (sorted(action_styles), sorted(want)))
        path = os.path.join(golden_dir,
                            "v3_footer_colors_%dx%d.txt" % (cols, rows))
        if os.path.exists(path):
            with open(path, encoding="utf-8") as handle:
                want_rows = [line.rstrip("\n") for line in handle
                             if line.strip()]
            if want_rows != golden_rows:
                problems.append("footer color golden differs (%d rows vs %d)"
                                % (len(golden_rows), len(want_rows)))
        with open(path, "w", encoding="utf-8") as handle:
            handle.write("\n".join(golden_rows) + "\n")
        if problems:
            failed = True
            print("FAIL footer_colors %dx%d" % (cols, rows))
            for problem in problems[:6]:
                print("  " + problem)
        else:
            print("PASS footer_colors %dx%d" % (cols, rows))
    return not failed


def check_float_chrome(cols, rows, golden_dir):
    """Floating window chrome: title/action row, collapse keeps the title."""
    program = demo(cols, rows)
    program.new_floating()
    actual, _styles = program.screen()
    floating = program.floatings[0]
    want = expected_float_row(floating.w, floating.pane.title)
    x, y = floating.x, floating.y
    problems = []
    got = cells_slice(actual[y], x, len(want))
    if got != want:
        problems.append("float title row %d:\n  got  %r\n  want %r" % (y, got, want))
    if y + 1 < rows and cell_at(actual[y + 1], x) != "\u2502":
        problems.append("expanded float has no body border at row %d: %r"
                        % (y + 1, actual[y + 1]))
    floating.collapsed = True
    collapsed, _styles = program.screen()
    if cells_slice(collapsed[y], x, len(want)) != want:
        problems.append("collapsed float lost its title row")
    if y + 1 < rows and cell_at(collapsed[y + 1], x) == "\u2502":
        problems.append("collapsed float still draws a body at row %d" % (y + 1))
    write_golden("float", actual, cols, rows, golden_dir)
    write_golden("float_collapsed", collapsed, cols, rows, golden_dir)
    if problems:
        print("FAIL float_chrome %dx%d" % (cols, rows))
        for problem in problems[:6]:
            print("  " + problem)
        return False
    print("PASS float_chrome %dx%d" % (cols, rows))
    return True


def check_footer_scenes(golden_dir, cols=120, rows=32):
    program = demo(cols, rows)
    expected = {
        "LIVE": expected_live_footer(cols, "main", 0, 11),
        "PANE": None,
        "RESIZE": None,
        "TAB": None,
        "SYSTEM": None,
        "PICKER": None,
        "PROMPT": None,
        "HELP": None,
    }
    modes = [("PANE", "pane"), ("RESIZE", "resize"), ("TAB", "tab"),
             ("SYSTEM", "system")]
    for name, mode in modes:
        program.mode = mode
        expected[name] = program.footer_line()
    program.mode = "live"
    program.overlay = "picker"
    expected["PICKER"] = program.footer_line()
    program.overlay = "prompt"
    expected["PROMPT"] = program.footer_line()
    program.overlay = "help"
    expected["HELP"] = program.footer_line()
    # The expected table is the program's own scene table here by design:
    # this check pins the *golden text* (scene badge + labels + alignment)
    # so a scene table edit is visible in review.
    path = os.path.join(golden_dir, "v3_footer_120x32.txt")
    got = {}
    if os.path.exists(path):
        with open(path, encoding="utf-8") as handle:
            for line in handle:
                line = line.rstrip("\n")
                if not line or line.startswith("#"):
                    continue
                name, rest = line.split("|", 1)
                got[name] = rest[:-1] if rest.endswith("|") else rest
        failures = 0
        for name in sorted(got):
            if got[name] != expected.get(name):
                print("FAIL footer golden %s:\n  golden %r\n  want   %r"
                      % (name, got[name], expected.get(name)))
                failures += 1
        if set(got) != set(expected):
            print("FAIL footer golden scenes: got %s want %s"
                  % (sorted(got), sorted(expected)))
            failures += 1
        if failures:
            return False
    with open(path, "w", encoding="utf-8") as handle:
        for name in sorted(expected):
            handle.write("%s|%s|\n" % (name, expected[name]))
    for name in ("PANE", "PICKER", "PROMPT"):
        if not expected[name]:
            print("FAIL footer scene %s is empty" % name)
            return False
    print("PASS footer scenes (8 rows)")
    return True


def check_card_actions(cols=120, rows=32):
    """Card button ids must cover zoom/split/close and hit the model."""
    program = demo(cols, rows)
    out = []
    tab = program.active_tab()
    runs = program.pane_runs(program.focus_pane(), True, cols)
    ids = [run[2] for run in runs if run[2]]
    for action in ("zoom", "split-v", "split-h", "close", "lock"):
        node = "pane:%s:%s" % (program.focus_pane().id, action)
        if node not in ids:
            print("FAIL card_actions: missing %s (ids=%s)" % (node, ids))
            return False
    before = len(tab.panes)
    program.handle_mouse({"action": "press", "node": "pane:%s:split-h" % program.focus_pane().id})
    if len(tab.panes) != before + 1:
        print("FAIL card_actions: split-h button did not add a pane")
        return False
    print("PASS card actions (zoom/split/close/lock)")
    return True


SCENARIOS = [
    ("demo_live", sc_demo_live),
    ("demo_tab2", sc_demo_tab2),
]


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument("--golden-dir", default=os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "golden"))
    parser.add_argument("--only", default="")
    parser.add_argument("--viewports", default="120x32,181x56")
    options = parser.parse_args(argv)
    viewports = []
    for spec in options.viewports.split(","):
        cols, _, rows = spec.partition("x")
        viewports.append((int(cols), int(rows)))
    os.makedirs(options.golden_dir, exist_ok=True)

    failures = 0
    total = 0
    for cols, rows in viewports:
        for name, builder in SCENARIOS:
            if options.only and options.only != name:
                continue
            total += 1
            if not check_demo(name, builder, cols, rows, options.golden_dir, True):
                failures += 1
    base_cols, base_rows = viewports[0]
    if options.only in ("", "split_col", "split_row"):
        for name, flow in (("split_col", "col"), ("split_row", "row")):
            if options.only and options.only != name:
                continue
            total += 1
            if not check_split_chrome(name, flow, base_cols, base_rows,
                                      options.golden_dir):
                failures += 1
    if options.only in ("", "float_chrome"):
        total += 1
        if not check_float_chrome(base_cols, base_rows, options.golden_dir):
            failures += 1
    if options.only in ("", "capture_1txt"):
        total += 1
        if not check_capture(181, 56):
            failures += 1
    if options.only in ("", "footer_scenes"):
        total += 1
        if not check_footer_scenes(options.golden_dir, base_cols, base_rows):
            failures += 1
    if options.only in ("", "card_actions"):
        total += 1
        if not check_card_actions(base_cols, base_rows):
            failures += 1
    if options.only in ("", "left1right2"):
        total += 1
        if not check_left1right2(viewports, options.golden_dir):
            failures += 1
    if options.only in ("", "footer_colors"):
        total += 1
        if not check_footer_colors(viewports, options.golden_dir):
            failures += 1
    print("v3 parity: %d checks, %d failed" % (total, failures))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
