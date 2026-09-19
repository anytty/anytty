#!/usr/bin/env python3
"""Command template: the smallest useful anytty layout program in Python.

A layout program is a process speaking the tui2 protocol on stdin/stdout; it
never links the Go framework. Uses the official Python SDK (pure stdlib):
tab bar · two terminal slots · footer · picker; Ctrl-T/Ctrl-F/click/Esc/
Ctrl-Q. 改哪里: header()/footer() 文字 · key() 键位 · pick() 的 Enter 行为 ·
view() 的盒子树。Dev loop: bash clients/tui/scripts/dev.sh clients/tui/templates/python
"""

import os
import sys

sys.path.insert(0, os.environ.get(
    "TUI2_PYTHON_SDK",
    os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "sdk", "python"),
))

from tui2sdk import App, Client, builder  # noqa: E402


class Template(App):
    def __init__(self):
        self.tabs = ["main"]
        self.active = 0
        self.slots = ["", ""]  # terminal source ids, empty = unbound
        self.focus = 0
        self.picker = False
        self.pick_sel = 0
        self.sources = []
        self.status = ""
        self.cols = 80
        self.rows = 24

    # ------------------------------------------------------------ protocol

    def commit(self):
        claim, all_keys = (["ctrl-t", "ctrl-f", "esc"], False)
        if self.picker:
            claim, all_keys = ([], True)
        self.client.commit(self.view(), claim, all_keys)

    def emit_create(self):
        def done(response):
            if response["ok"]:
                data = response["data"] or {}
                self.slots[self.focus] = "terminal:%s:%s" % (data.get("endpoint") or "local", data.get("id", ""))
                self.status = "bound " + data.get("id", "")
            else:
                self.status = "create failed: " + response["error"]
            self.commit()
        self.emit("terminal.create", {"endpoint": "local"}, done)

    def emit_attach(self, source):
        parts = source["id"].split(":", 2)
        endpoint, terminal_id = (parts[1], parts[2]) if len(parts) == 3 else ("local", source["id"])

        def done(response):
            if response["ok"]:
                self.slots[self.focus] = source["id"]
                self.status = "bound " + terminal_id
            else:
                self.status = "attach failed: " + response["error"]
            self.commit()
        self.emit("terminal.attach", {"endpoint": endpoint, "id": terminal_id, "fit": True}, done)
    def pick(self):
        self.picker = False
        if self.pick_sel >= len(self.sources):
            self.status = "creating terminal"
            self.emit_create()
        else:
            self.status = "binding"
            self.emit_attach(self.sources[self.pick_sel])
    # -------------------------------------------------------------- events

    def on_hello(self, hello):
        self.cols = hello["cols"] or self.cols
        self.rows = hello["rows"] or self.rows
        self.commit()
    def on_sources(self, event):
        self.sources = [s for s in event["items"] if s["kind"] == "terminal"]
        if not any(self.slots):
            if self.sources:
                # 改哪里: 冷启动自动绑定第一个终端；想总是弹选择器就改成
                # self.picker, self.pick_sel = True, 0。
                self.slots[0] = self.sources[0]["id"]
            else:
                self.picker, self.pick_sel = True, 0
        self.commit()

    def on_key(self, event):
        key = event["key"] or event.get("char", "")
        if self.picker:
            if key == "up":
                self.pick_sel = max(0, self.pick_sel - 1)
            elif key == "down":
                self.pick_sel = min(len(self.sources), self.pick_sel + 1)
            elif key == "enter":
                self.pick()
            elif key in ("esc", "ctrl-f"):
                self.picker = False
        else:
            # 改哪里: NORMAL 模式键位。
            if key == "ctrl-f":
                self.picker, self.pick_sel = True, 0
            elif key == "ctrl-t":
                self.tabs.append(str(len(self.tabs) + 1))
                self.active = len(self.tabs) - 1
            elif key == "esc":
                self.emit("system.quit", None, None)
        self.commit()

    def on_mouse(self, event):
        if event["action"] != "press":
            return
        node = event["node"]
        if node.startswith("slot:"):
            self.focus = builder.clamp(int(node[5:] or 0), 0, 1)
        elif node.startswith("pick:"):
            self.pick_sel = builder.clamp(int(node[5:] or 0), 0, len(self.sources))
            self.pick()
        self.commit()

    def on_resize(self, event):
        self.cols = event["cols"] or self.cols
        self.rows = event["rows"] or self.rows
        self.commit()

    def on_notice(self, event):
        self.status = event["message"]
        self.commit()

    # ---------------------------------------------------------------- view

    def view(self):
        root = builder.col(self.header(), self.body(), self.footer())
        if self.picker:
            root.child(self.overlay())
        return root.build()

    # 改哪里: tab 条（图标、标题、+ 按钮）。
    def header(self):
        row = builder.row(builder.text(" main ").style("tab_inactive")).id("header").height(1)
        for index, name in enumerate(self.tabs):
            if index == self.active:
                row.child(builder.text("[%s]" % name).style("tab_active"))
            else:
                row.child(builder.text(" %s " % name).style("tab_inactive"))
        return row.child(builder.text(" + ").id("tab:new").style("tab_inactive").input("mouse"))

    def body(self):
        row = builder.box("row").flex(1)
        for index, source in enumerate(self.slots):
            box_id = "slot:%d" % index
            if not source:
                row.child(builder.col(builder.text("  [空槽]"), builder.text("  Ctrl-F 选择终端"))
                          .id(box_id).style("muted").flex(1).input("mouse"))
            else:
                row.child(builder.terminal(source).id(box_id).flex(1)
                          .focused(index == self.focus).input("key", "paste", "wheel")
                          .props({"chrome.title": "muted"}))
        return row

    # 改哪里: footer 键位提示。
    def footer(self):
        hints = "Ctrl-F picker · Ctrl-T tab · click focus · Esc quit"
        if self.picker:
            hints = "↑/↓ select · Enter bind · Esc close"
        status = "tab %d/%d" % (self.active + 1, len(self.tabs))
        if self.status:
            status = self.status + " │ " + status
        return builder.row(builder.text(hints).style("muted"), builder.text(" ").flex(1),
                           builder.text(builder.truncate(status, self.cols // 2)).style("status")
                           ).id("footer").height(1)

    def overlay(self):
        rows = builder.col(builder.text(" select a terminal").style("muted")).id("picker")
        for index in range(len(self.sources) + 1):
            if index < len(self.sources):
                source = self.sources[index]
                state = "%s · %s" % (source["endpoint"] or "local", "exited" if source["exited"] else "live")
                label = source["title"] or source["id"]
            else:
                label, state = "+ New terminal", "create"
            marker, style = ("> ", "selection") if index == self.pick_sel else ("  ", "")
            rows.child(builder.text(marker + label + "  " + state).id("pick:%d" % index)
                       .style(style).input("mouse"))
        rows.child(builder.text(" enter bind · esc close").style("muted"))
        width, height = min(46, max(10, self.cols - 2)), len(self.sources) + 4
        return rows.pos(max(0, (self.cols - width) // 2), max(0, (self.rows - height) // 2)) \
                   .width(width).height(height)


def main():
    return Client(sys.stdin.buffer, sys.stdout.buffer).run(Template())


if __name__ == "__main__":
    sys.exit(main())
