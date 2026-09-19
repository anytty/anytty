"""Reference conformance program of the Python SDK.

``python3 clients/tui/sdk/python/conformance.py`` is the candidate program for
``tui2-sdk-verify --cmd``. It implements the behaviour fixed by
``clients/tui/conformance/README.zh-CN.md`` using only this SDK, exactly like the Go
reference program, so both bindings are held to the same fixture standard.

It writes one readable line per observed protocol fact and commits a view
whose flattened text is those lines; the runner checks the frames, not the
internals.
"""

import sys

from . import builder
from .core import App, Client

CLAIM = ["ctrl-p", "?"]


class RefProgram(App):
    """The conformance program (README.zh-CN.md)."""

    def __init__(self):
        self.lines = []
        self.pane = False

    # ----------------------------------------------------------- helpers

    def commit(self):
        root = builder.text("\n".join(self.lines)).build()
        self.client.commit(root, CLAIM, False)

    def _emit_read(self):
        def callback(response):
            self._callback_line(response)

        request_id = self.client.emit("clipboard.read", None, callback)
        self.lines.append("emit clipboard.read id=%d" % request_id)

    def _callback_line(self, response):
        data = response.get("data") or {}
        self.lines.append(
            "cb id=%d ok=%d text=%s endpoint=%s data_id=%s rows=%d err=%s"
            % (response.get("request_id", 0), 1 if response.get("ok") else 0,
               data.get("text", ""), data.get("endpoint", ""), data.get("id", ""),
               len(data.get("rows") or []), response.get("error", "")))

    # ---------------------------------------------------------- handlers

    def on_hello(self, hello):
        self.pane = False
        self.lines = [
            "hello view=%s epoch=%d cols=%d rows=%d schema=%d"
            % (hello.get("view_id", ""), hello.get("epoch", 0),
               hello.get("cols", 0), hello.get("rows", 0), hello.get("schema", 0))]
        if hello.get("components"):
            self.lines.append("components=" + ",".join(hello["components"]))
        if hello.get("methods"):
            self.lines.append("methods=" + ",".join(hello["methods"]))
        self._emit_read()
        self.commit()

    def on_key(self, event):
        if event.get("key") == "ctrl-p":
            self.pane = True
        self.lines.append("key key=%s char=%s pane=%d"
                          % (event.get("key", ""), event.get("char", ""), 1 if self.pane else 0))
        if event.get("key") == "ctrl-r":
            self._emit_read()
        self.commit()

    def on_paste(self, event):
        self.lines.append("paste id=%s text=%s" % (event.get("id", ""), event.get("text", "")))
        self.commit()

    def on_mouse(self, event):
        self.lines.append("mouse action=%s button=%s x=%d y=%d node=%s"
                          % (event.get("action", ""), event.get("button", ""),
                             event.get("x", 0), event.get("y", 0), event.get("node", "")))
        self.commit()

    def on_wheel(self, event):
        self.lines.append("wheel delta=%d x=%d y=%d node=%s"
                          % (event.get("delta", 0), event.get("x", 0),
                             event.get("y", 0), event.get("node", "")))
        self.commit()

    def on_resize(self, event):
        self.lines.append("resize cols=%d rows=%d" % (event.get("cols", 0), event.get("rows", 0)))
        self.commit()

    def on_sources(self, event):
        items = event.get("items") or []
        self.lines.append("sources count=%d" % len(items))
        for item in items:
            self.lines.append(
                "source id=%s kind=%s endpoint=%s terminal=%s exited=%d"
                % (item.get("id", ""), item.get("kind", ""), item.get("endpoint", ""),
                   item.get("terminal_id", ""), 1 if item.get("exited") else 0))
        for item in items:
            if item.get("kind") == "terminal" and item.get("terminal_id"):
                def callback(response):
                    self._callback_line(response)

                request_id = self.client.emit("terminal.attach", {
                    "endpoint": item.get("endpoint", ""),
                    "id": item.get("terminal_id", ""),
                    "fit": True,
                }, callback)
                self.lines.append("emit terminal.attach id=%d" % request_id)
                break
        self.commit()

    def on_notice(self, event):
        self.lines.append("notice level=%s message=%s"
                          % (event.get("level", ""), event.get("message", "")))
        self.commit()

    def on_component(self, event):
        self.lines.append("component source=%s name=%s value=%s"
                          % (event.get("source", ""), event.get("name", ""), event.get("value", "")))
        self.commit()

    def on_view_rejected(self, event):
        self.lines.append("view-rejected epoch=%d rev=%d reason=%s"
                          % (event.get("epoch", 0), event.get("rev", 0), event.get("reason", "")))
        self.commit()

    def on_response(self, response):
        self.lines.append("resp id=%d epoch=%d ok=%d err=%s"
                          % (response.get("request_id", 0), response.get("epoch", 0),
                             1 if response.get("ok") else 0, response.get("error", "")))
        self.commit()


def main():
    return Client(sys.stdin.buffer, sys.stdout.buffer).run(RefProgram())


if __name__ == "__main__":
    sys.exit(main())
