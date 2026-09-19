#!/usr/bin/env python3
"""Pixel replica of the legacy v3 TUI (surface framework) as a TUI v2 program.

Target: ``clients/tui/examples/python-shell/1.txt``, the real capture of the old
``tui/app`` + ``tui/render`` surface framework with the ``coralline-candy``
recommended profile (``tui/docs/tui-v3.recommended.yaml``).

The chrome itself (card/tab bar/footer/picker/floating/split tree) is the
generic program-side toolkit in ``clients/tui/sdk/python/tui2sdk/widgets.py``; this
file is only the v3 policy: the coralline-candy theme, the shortcut scenes,
the demo state and the CLI modes. Modes:

  - default: wire program (stdin/stdout frames, via the Python SDK client);
  - ``--demo``: preload the 1.txt state (workspace ``main``, tabs
    ``auto-push``/``local``, two panes, 11-terminal pool) with deterministic
    placeholder content; ``--selftest`` prints the rasterized screen;
  - ``--selftest [--cols N] [--rows N]``: print the demo screen and exit
    (used by ``v3_parity_test.py`` and acceptance.sh);
  - ``--footer-lines``: print ``<SCENE>|<line>|`` rows for the recommended
    v3 scene footer set (debug/verification helper).
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                "..", "..", "sdk", "python"))

from tui2sdk.core import Client                  # noqa: E402
from tui2sdk.wire import WireError                 # noqa: E402
from tui2sdk.widgets import (ChromeApp, Floating, Leaf, Pane, Split, Tab,
                             Theme)

COLS_DEFAULT = 120
ROWS_DEFAULT = 32

# ------------------------------------------------------------------- colors
# coralline-candy tokens (tui-v3.recommended.yaml) as explicit style strings.
FG = "#f8f4ff"
PRIMARY = "#f0abfc"
SECONDARY = "#3b2f63"
BG = "#070611"
STATUS_BG = "#17132a"
OVERLAY_BG = "#261b44"
MUTED = "#9ca3c9"
SUCCESS = "#86efac"
WARNING = "#fde68a"
DANGER = "#fb7185"
INFO = "#7dd3fc"
TAB_INACTIVE_BG = "#261b44"
TAB_INACTIVE_FG = "#c4b5fd"
CREATE_BG = "#fde68a"
CREATE_FG = "#1b1230"
WS_FG = "#1b1230"


def style(fg=None, bg=None, bold=False, dim=False, italic=False,
          underline=False, reverse=False):
    parts = []
    if fg:
        parts.append("fg:" + fg)
    if bg:
        parts.append("bg:" + bg)
    if bold:
        parts.append("bold")
    if dim:
        parts.append("dim")
    if italic:
        parts.append("italic")
    if underline:
        parts.append("underline")
    if reverse:
        parts.append("reverse")
    return ";".join(parts)


ST_WS_L = style(fg=PRIMARY, bg=BG)
ST_WS_BODY = style(fg=WS_FG, bg=PRIMARY, bold=True)
ST_WS_R = style(fg=PRIMARY, bg=SECONDARY)
ST_TAB_L = style(fg=SECONDARY, bg=PRIMARY)
ST_TAB_BODY = style(fg=WS_FG, bg=PRIMARY, bold=True)
ST_TAB_R = style(fg=PRIMARY, bg=SECONDARY)
ST_TAB_SPACE = style(fg=FG, bg=SECONDARY)
ST_TAB_CLOSE = style(fg=DANGER, bg=SECONDARY)
ST_ITAB_L = style(fg=SECONDARY, bg=TAB_INACTIVE_BG)
ST_ITAB_BODY = style(fg=TAB_INACTIVE_FG, bg=TAB_INACTIVE_BG, bold=True)
ST_ITAB_R = style(fg=TAB_INACTIVE_BG, bg=SECONDARY)
ST_ITAB_SPACE = style(fg=TAB_INACTIVE_FG, bg=SECONDARY)
ST_ITAB_CLOSE = style(fg=MUTED, bg=SECONDARY)
ST_CREATE_L = style(fg=SECONDARY, bg=CREATE_BG)
ST_CREATE_BODY = style(fg=CREATE_FG, bg=CREATE_BG, bold=True)
ST_CREATE_R = style(fg=CREATE_BG, bg=BG)
ST_HEADER_FILL = style(fg=FG, bg=BG)

ST_ACCENT = style(fg=PRIMARY)
ST_ACCENT_GROUP = style(fg=WS_FG, bg=PRIMARY, bold=True)
ST_PANEL_BORDER = style(fg=SECONDARY)
ST_INACTIVE_GROUP = style(fg=TAB_INACTIVE_FG, bg=SECONDARY, bold=True)
ST_MUTED = style(fg=MUTED)
ST_SUCCESS = style(fg=SUCCESS)
ST_WARNING = style(fg=WARNING)
ST_DANGER = style(fg=DANGER)
ST_DANGER_STRONG = style(fg=DANGER, bold=True)
ST_FOOTER = style(fg=MUTED)
ST_FOOTER_ACCENT = style(fg=PRIMARY, bold=True)
# Footer key group colors: old tui/render ansiForStyleToken() with the
# coralline-candy tokens (mixHostColor copies the Go rounding exactly).
ST_FOOTER_KEY_PANE = style(fg=PRIMARY, bold=True)
ST_FOOTER_KEY_RESIZE = style(fg=WARNING, bold=True)
ST_FOOTER_KEY_TAB = style(fg=INFO, bold=True)
ST_FOOTER_KEY_WORKSPACE = style(fg=SUCCESS, bold=True)
ST_FOOTER_KEY_FLOAT = style(fg="#bcbdfc", bold=True)
ST_FOOTER_KEY_COPY = style(fg="#83e5c8", bold=True)
ST_FOOTER_KEY_PICKER = style(fg="#fc9a87", bold=True)
ST_FOOTER_KEY_GLOBAL = style(fg="#f5c0d4", bold=True)
ST_FOOTER_INFO = style(fg=INFO)
ST_FOOTER_SUCCESS = style(fg=SUCCESS)
ST_FOOTER_DANGER = style(fg=DANGER)
ST_FOOTER_DANGER_STRONG = style(fg=DANGER, bold=True)
ST_FOOTER_FILL = style(fg=FG)
ST_OVERLAY = style(fg=FG, bg=OVERLAY_BG)
ST_CONTENT = style(fg=FG)
ST_HINT = style(fg=MUTED)
ST_TOAST = style(fg=WARNING)

# yaml `shortcuts.actions` style overrides, verbatim (tui-v3.recommended.yaml).
SHORTCUT_ACTION_STYLES = {
    "menu.panel": "footer-key-pane",
    "menu.resize": "footer-key-resize",
    "menu.system": "footer-key-global",
    "menu.floating": "footer-key-float",
    "menu.tab": "footer-key-tab",
    "menu.workspace": "footer-key-workspace",
    "menu.copy": "footer-key-copy",
    "menu.terminal_picker": "footer-key-picker",
    "menu.terminal_pool": "footer-key-picker",
    "menu.connections": "info",
    "menu.workbench_tree": "footer-key-workspace",
    "menu.prompt": "footer-key-global",
    "menu.help": "footer-key-tab",
    "system.toggle_header": "info",
    "system.toggle_footer": "success",
    "system.quit": "danger-strong",
    "panel.close": "danger",
    "panel.kill": "danger-strong",
    "panel.kill_and_close": "danger-strong",
    "clipboard.paste_latest": "success",
    "clipboard.paste_system": "info",
    "menu.clipboard_history": "footer-key-copy",
}

FOOTER_STYLE_TOKENS = {
    "footer-accent": ST_FOOTER_ACCENT,
    "footer-key-pane": ST_FOOTER_KEY_PANE,
    "footer-key-resize": ST_FOOTER_KEY_RESIZE,
    "footer-key-tab": ST_FOOTER_KEY_TAB,
    "footer-key-workspace": ST_FOOTER_KEY_WORKSPACE,
    "footer-key-float": ST_FOOTER_KEY_FLOAT,
    "footer-key-copy": ST_FOOTER_KEY_COPY,
    "footer-key-picker": ST_FOOTER_KEY_PICKER,
    "footer-key-global": ST_FOOTER_KEY_GLOBAL,
    "info": ST_FOOTER_INFO,
    "success": ST_FOOTER_SUCCESS,
    "warning": ST_WARNING,
    "danger": ST_FOOTER_DANGER,
    "danger-strong": ST_FOOTER_DANGER_STRONG,
}

# yaml `footer.modes`: per-scene badge style (unlisted scenes default to
# footer-accent, exactly like the old renderer).
MODE_STYLES = {
    "live": "footer-accent",
    "copy": "footer-key-copy",
    "pane": "footer-key-pane",
    "resize": "footer-key-resize",
    "terminal-picker": "footer-key-picker",
    "terminal-picker-endpoints": "footer-key-picker",
    "terminal-picker-tags": "footer-key-picker",
    "workbench-tree": "footer-key-workspace",
}

# -------------------------------------------------------------------- glyphs
WS_ICON = "\U000f0645"       # nf-md-folder
TAB_ICON = "\u2387"          # ⎇
TAB_CLOSE = "\U000f0156"     # nf-md-close
TAB_CREATE = "\U000f0415"    # nf-md-plus
EDGE_L = "\ue0b6"
EDGE_R = "\ue0b0"
EDGE_ROUND_R = "\ue0b4"
LOCKED = "\u25a0"            # ■
UNLOCKED = "\u25a1"          # □
ZOOM = "\U000f004c"
UNZOOM = "\u2199"
SPLIT_V = "\ueb56"
SPLIT_H = "\ueb57"
CLOSE = "\U000f0156"
RUNNING = "\u25cf"           # ●
CENTER = "\u25ce"            # ◎
COLLAPSE = "\u25be"          # ▾
FLOAT_SUMMARY = "\U000f0e59"
TERM_SUMMARY = "\uf489"
TABS_SUMMARY = "\U000f04e9"
PANES_SUMMARY = "\uebeb"
CHECK = "\u2713"
GUTTER = "\u2503"            # ┃
COLLAPSE_HINT = "Click to collapse"

MODE_ICONS = {
    "live": "\U000f030c",
    "copy": "\U000f018f",
    "pane": "\uebeb",
    "resize": "\U000f0656",
    "terminal-picker": "\U000f0c7c",
    "prompt": "\U000f0627",
    "help": "\U000f02d6",
    "tab": TABS_SUMMARY,
    "workspace": WS_ICON,
    "system": "\U000f0493",
    "floating": FLOAT_SUMMARY,
}


DEMO_LEFT_LINES = [
    "",
    "  " + GUTTER,
    "  " + GUTTER + "  " + COLLAPSE_HINT,
    "",
    "  复刻：老 v3 的 card pane + bracket 动作组",
    "  chrome 逐字符对齐（120 列 golden）",
    "",
    "  Ctrl-P PANE 模式：",
    "    % / \" 分屏 · x 关闭 · t 重启 · q kill+close",
    "    h / l 焦点 · z 折叠提示行",
    "  Ctrl-T TAB · Ctrl-O FLOAT · Ctrl-F PICK",
    "  Ctrl-G SYSTEM · Ctrl-Q 退出",
    "",
]

DEMO_RIGHT_LINES = [
    "",
    "  OpenCode / 任意终端内容区",
    "",
    "  content.self = terminal:<endpoint>:<id>",
    "  组件边框已关闭（chrome.inset=0），",
    "  pane 框由程序自绘：",
    "",
    "  ┌─ □  anytty-surface@hs \u2500\u2500\u2500 ●  x1  owner \u2500\u2500 \ue0b6 \U000f004c  \ueb56  \ueb57  \U000f0156 \ue0b4 \u2500\u2510",
    "",
]


SCENES = {
    "live": (MODE_ICONS["live"], "CTRL", [
        ("P \uebeb PANE", "f:ctrl-p", "menu.panel", "^P"),
        ("R \U000f0656 SIZE", "f:ctrl-r", "menu.resize", "^R"),
        ("O \U000f0e59 FLOAT", "f:ctrl-o", "menu.floating", "^O"),
        ("T \U000f04e9 TAB", "f:ctrl-t", "menu.tab", "^T"),
        ("W \U000f0645 WORKSPACE", "f:ctrl-w", "menu.workspace", "^W"),
        ("F \U000f0c7c PICK", "f:ctrl-f", "terminal_picker.open", "^F"),
        ("\u21e7C \U000f0489 SELECT", "f:ctrl-shift-c", "copy.enter", "^C"),
        ("\u21e7H \U000f014c CLIPBOARD", "f:ctrl-shift-h", "menu.clipboard_history", "^H"),
        ("\u21e7V \U000f018f PASTE", "f:ctrl-shift-v", "clipboard.paste_system", "^V"),
        ("G \U000f0493 SYSTEM", "f:ctrl-g", "menu.system", "^G"),
    ]),
    "pane": (MODE_ICONS["pane"], "PANE", [
        ("X \U000f0156 CLOSE", "fs:pane:close", "panel.close", "x"),
        ("CTRL+D \ueb56 VSPLIT", "fs:pane:split-h", "panel.split_right", "^D"),
        ("CTRL+E \ueb57 HSPLIT", "fs:pane:split-v", "panel.split_down", "^E"),
        ("H/L \U000f0734 FOCUS", "fs:pane:focus", "panel.focus_prev", "h"),
        ("Q \U000f0688 KILL+CLOSE", "fs:pane:kill-close", "panel.kill_and_close", "q"),
    ]),
    "resize": (MODE_ICONS["resize"], "SIZE", [
        ("H \u2190 LEFT", "fs:resize:left", "resize.left", "h"),
        ("L \u2192 RIGHT", "fs:resize:right", "resize.right", "l"),
        ("K \u2191 UP", "fs:resize:up", "resize.up", "k"),
        ("J \u2193 DOWN", "fs:resize:down", "resize.down", "j"),
        ("S \U000f033e LOCK", "fs:resize:lock", "panel.size_lock", "s"),
        ("SPACE \U000f0636 LAYOUT", "fs:resize:layout", "resize.layout_toggle", "space"),
        ("R \U000f0410 RESET", "fs:resize:reset", "resize.layout_reset", "r"),
        ("= \U000f0555 BALANCE", "fs:resize:balance", "panel.balance", "="),
    ]),
    "tab": (MODE_ICONS["tab"], "TAB", [
        ("N \U000f04ad NEXT", "fs:tab:next", "tab.next", "n"),
        ("P \U000f04ae PREV", "fs:tab:prev", "tab.previous", "p"),
    ]),
    "workspace": (MODE_ICONS["workspace"], "WORKSPACE", [
        ("N \U000f04ad NEXT", "fs:ws:next", "workspace.next", "n"),
        ("P \U000f04ae PREV", "fs:ws:prev", "workspace.previous", "p"),
        ("T \U000f0645 TREE", "fs:ws:tree", "system.open_workbench_tree", "t"),
    ]),
    "system": (MODE_ICONS["system"], "SYSTEM", [
        ("P \uf489 TERMINALS", "fs:sys:terminals", "system.open_terminal_pool", "p"),
        ("E \U000f0337 CONNECTIONS", "fs:sys:connections", "system.open_connections", "e"),
        ("W \U000f0645 TREE", "fs:sys:tree", "system.open_workbench_tree", "w"),
        ("O \uf4b5 COMMAND", "fs:sys:prompt", "system.open_prompt", "o"),
    ]),
    "floating": (MODE_ICONS["floating"], "FLOAT", [
        ("N \U000f0415 NEW", "fs:float:new", "floating.new", "n"),
        ("O \U000f0e59 OVERVIEW", "fs:float:overview", "floating.overview", "o"),
        ("F \U000f0c7c PICK", "fs:float:pick", "system.open_terminal_picker", "f"),
        ("X \U000f0156 CLOSE", "fs:float:close", "floating.close", "x"),
        ("Z \U000f0615 HIDE", "fs:float:collapse", "floating.collapse", "z"),
    ]),
    "terminal-picker": (MODE_ICONS["terminal-picker"], "PICK", [
        ("\u2190/\u2192 ENDPOINT", "", "terminal_picker.endpoint_previous", "\u2190"),
        ("\u2191/\u2193 SELECT", "", "terminal_picker.select_previous", "\u2191"),
        ("ENTER \U000f02fa ATTACH", "", "terminal_picker.attach", "enter"),
        ("ESC BACK", "", "shortcut.exit", "Esc"),
    ]),
    "prompt": (MODE_ICONS["prompt"], "", [
        ("ENTER \U000f0627 RUN", "", "prompt.submit", "enter"),
    ]),
    "help": ("", "", []),
    "copy": (MODE_ICONS["copy"], "COPY", [
        ("PGUP \U000f005d OLDER", "fs:copy:older", "copy.request_older", "PgUp"),
        ("PGDN \U000f0045 NEWER", "fs:copy:newer", "copy.request_newer", "PgDn"),
        ("Y \U000f018f COPY", "fs:copy:copy", "copy.copy_selection", "y"),
        ("G \U000f005e OLDEST", "fs:copy:oldest", "copy.oldest", "g"),
    ]),
}


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



# The theme is every module-level token (styles + glyphs + scene tables); the
# generic widgets read it through ``self.t``.
THEME = Theme()
for _name, _value in list(globals().items()):
    if _name.isupper() and _name not in ("COLS_DEFAULT", "ROWS_DEFAULT"):
        setattr(THEME, _name, _value)


class Program(ChromeApp):
    """The v3 replica: the SDK chrome with the coralline-candy policy."""

    def __init__(self, out=None, cols=COLS_DEFAULT, rows=ROWS_DEFAULT,
                 view_id="", epoch=0, demo=False):
        super().__init__(out=out, cols=cols, rows=rows, view_id=view_id,
                         epoch=epoch, demo=demo, theme=THEME)

    def load_demo(self):
        """The 1.txt target state: one window card with two content panes.

        Workspace ``main``, tabs ``auto-push``/``local``; the active tab's
        window is split into the left terminal pane (``anytty-surface@hs``)
        and the right pane (``opencode@hs``), exactly the two visual areas of
        the 1.txt capture.  The terminal pool carries 11 entries so the footer
        summary reads ``󰙅 main 󰹙 0  11``.
        """
        self.workspace = "main"
        self.tab_seq = 2
        self.pane_seq = 0
        left = self.new_pane("anytty-surface@hs", DEMO_LEFT_LINES)
        right = self.new_pane("opencode@hs", DEMO_RIGHT_LINES)
        tab1 = Tab("tab-1", "auto-push", [left, right], "row")
        tab1.focus = 0
        third = self.new_pane("local@hs", ["", "  local \u00b7 T2"])
        tab2 = Tab("tab-2", "local", [third], "row")
        self.tabs = [tab1, tab2]
        self.active = 0
        pool = []
        for index in range(11):
            if index == 0:
                terminal_id = "anytty-surface"
            elif index == 1:
                terminal_id = "opencode"
            elif index == 2:
                terminal_id = "local"
            else:
                terminal_id = "term-%d" % index
            pool.append({
                "id": "terminal:hs:" + terminal_id, "kind": "terminal",
                "title": terminal_id, "endpoint": "hs",
                "terminal_id": terminal_id, "attached": True,
                "exited": False, "exit_code": 0,
                "resize_owner": "view:demo" if index == 0 else "",
            })
        self.sources = pool
        self.sources_ready = True
        self.terminal_count = len(pool)
        left.source_id = pool[0]["id"]
        right.source_id = pool[1]["id"]
        third.source_id = pool[2]["id"]


def run_stream(stdin, stdout, demo=False):
    program = Program(stdout, demo=demo)
    return Client(stdin, stdout).run(program)


def demo_program(cols, rows):
    program = Program(None, cols=cols, rows=rows, demo=True)
    program.view_id = "selftest"
    return program


def footer_lines(cols, rows):
    program = demo_program(cols, rows)
    out = {}
    out["LIVE"] = program.footer_line()
    program.mode = "pane"
    out["PANE"] = program.footer_line()
    program.mode = "resize"
    out["RESIZE"] = program.footer_line()
    program.mode = "tab"
    out["TAB"] = program.footer_line()
    program.mode = "system"
    out["SYSTEM"] = program.footer_line()
    program.mode = "floating"
    out["FLOATING"] = program.footer_line()
    program.overlay = "picker"
    out["PICKER"] = program.footer_line()
    program.overlay = "prompt"
    out["PROMPT"] = program.footer_line()
    program.overlay = "help"
    out["HELP"] = program.footer_line()
    return out


def main(argv):
    selftest = "--selftest" in argv
    footer = "--footer-lines" in argv
    demo = "--demo" in argv or selftest
    if selftest or footer:
        cols, rows = COLS_DEFAULT, ROWS_DEFAULT
        if "--cols" in argv:
            cols = int(argv[argv.index("--cols") + 1])
        if "--rows" in argv:
            rows = int(argv[argv.index("--rows") + 1])
        if footer:
            for name, line in sorted(footer_lines(cols, rows).items()):
                sys.stdout.write("%s|%s|\n" % (name, line))
            return 0
        program = demo_program(cols, rows)
        lines, _styles = program.screen()
        sys.stdout.write("\n".join(lines) + "\n")
        return 0
    try:
        return run_stream(sys.stdin.buffer, sys.stdout.buffer, demo=demo)
    except (BrokenPipeError, KeyboardInterrupt, EOFError):
        return 0
    except WireError as error:
        sys.stderr.write("v3ui: %s\n" % error)
        return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
