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

from . import builder, wire
from .core import App, Client

CLAIM = ["ctrl-p", "?"]

# ProbeFeatureInvalidFirstFrame is a non-standard HELLO feature that arms the
# reference program's test-only probe: on a restart epoch's first commit it
# deliberately violates PROTOCOL §2.1 by writing a VIEW_DELTA with rev_base=0
# instead of a full VIEW, so the host can reject it (reason base_mismatch) and
# the program can prove it resyncs with a full snapshot. A compliant SDK never
# sends this frame (it drops its baseline on HELLO), so the probe writes raw
# bytes around the SDK; it exists only so the fixtures can exercise the
# host-side rejection deterministically.
PROBE_INVALID_FIRST = "conformance.invalid_first_frame_delta"


class RefProgram(App):
    """The conformance program (README.zh-CN.md)."""

    def __init__(self):
        self.lines = []
        self.pane = False
        # last_root is the tree of the last commit, handed back to
        # commit_delta as the diff baseline. It exercises the VIEW_DELTA path
        # when the host advertised features["view_delta"].
        self.last_root = None
        # probe_invalid_first is armed by PROBE_INVALID_FIRST and consumed by
        # the next commit (the test-only invalid-first-frame probe).
        self.probe_invalid_first = False

    # ----------------------------------------------------------- helpers

    def commit(self):
        # One child box per line: the runner flattens the tree to text, so the
        # rendered content is identical to a single text box, but an append is
        # now a small insert patch instead of a set carrying the whole log.
        # That lets the VIEW_DELTA path produce a delta smaller than the full
        # VIEW deterministically (PROTOCOL §2.1).
        root = builder.col(*[builder.text(line) for line in self.lines]).build()
        if self.probe_invalid_first:
            self.probe_invalid_first = False
            if self._write_invalid_first_delta(root):
                self.last_root = root
                return
        self.client.commit_delta(self.last_root, root, CLAIM, False)
        self.last_root = root

    def _write_invalid_first_delta(self, root):
        """Test-only probe (PROBE_INVALID_FIRST): write a raw VIEW_DELTA with
        rev=1, rev_base=0 as the first frame of a new epoch, deliberately
        violating PROTOCOL §2.1 so the host rejects it. It bypasses the SDK,
        which correctly refuses to send such a frame."""
        if not self.last_root:
            return False
        patches = wire.diff_view(self.last_root, root)
        if not patches:
            return False
        payload = wire.encode_view_delta(self.client.epoch, 1, 0, CLAIM, False, patches)
        self.client._write(wire.VIEW_DELTA, payload)
        return True

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
        self.probe_invalid_first = bool((hello.get("features") or {}).get(PROBE_INVALID_FIRST))
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
        # PROTOCOL §2.1: after a rejection the next frame must be a full VIEW.
        self.last_root = None
        self.commit()

    def on_stream(self, frame):
        payload = frame.get("payload") or b""
        self.lines.append("stream id=%d kind=%s wire_type=%d payload=%s"
                          % (frame.get("stream_id", 0), frame.get("kind", ""),
                             frame.get("wire_type", 0), payload.hex()))
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
