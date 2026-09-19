"""Protocol client: frame loop, typed events, Emit/Commit (stdlib only).

``Client`` owns one connection (PROTOCOL §0.5):

  - it reads HELLO/EVENT/RESPONSE and dispatches to a typed ``App``;
  - ``emit(method, params, on_response)`` sends a RESULT with a fresh
    request_id and stores the per-request callback; the callback fires exactly
    once, and only for a RESPONSE of the current epoch (request_id is scoped
    to ``(epoch, connection)``);
  - ``commit(root, claim, all_keys)`` sends a full VIEW snapshot; rev
    increments per epoch and resets on every HELLO (atomic epoch reset);
  - ``log(level, message)`` is the one logging hook; layout programs never
    write to stdout outside frames.

Typed events dispatch through ``App.on_event``: a subclass overrides
``on_key``/``on_paste``/``on_mouse``/``on_wheel``/``on_resize``/``on_sources``/
``on_notice``/``on_component``/``on_view_rejected`` (named after the protocol)
or overrides ``on_event`` to handle everything itself.
"""

import sys

from . import wire
from .wire import WireError


class App:
    """Base class for programs driven by ``Client``. Every handler is
    optional; the defaults drop the event."""

    client = None

    # ---------------------------------------------------------- helpers

    def emit(self, method, params=None, on_response=None):
        """Send one method call; returns the request_id."""
        if self.client is None:
            raise WireError("emit before Client.run")
        return self.client.emit(method, params, on_response)

    def log(self, level, message):
        if self.client is not None:
            self.client.log(level, message)

    # ---------------------------------------------------------- handlers

    def on_hello(self, hello):
        pass

    def on_event(self, event):
        handler = getattr(self, "on_" + str(event.get("kind")), None)
        if handler is not None:
            handler(event)

    def on_key(self, event):
        pass

    def on_paste(self, event):
        pass

    def on_mouse(self, event):
        pass

    def on_wheel(self, event):
        pass

    def on_resize(self, event):
        pass

    def on_sources(self, event):
        pass

    def on_notice(self, event):
        pass

    def on_component(self, event):
        pass

    def on_view_rejected(self, event):
        pass

    def on_response(self, response):
        pass


class Client:
    """One layout-program connection over binary stdin/stdout."""

    def __init__(self, stream_in, stream_out, log=None):
        self._in = stream_in
        self._out = stream_out
        self._log = log
        self.app = None
        self.hello = None
        self.epoch = 0
        self.rev = 0
        self.request_id = 0
        self.pending = {}
        self.closed = False

    # ------------------------------------------------------------- loop

    def run(self, app):
        """Read frames until clean EOF; returns the process exit code (0)."""
        if self.app is not None:
            raise WireError("Client.run may only be called once")
        self.app = app
        app.client = self
        while True:
            frame = wire.read_frame(self._in)
            if frame is None:
                self.closed = True
                return 0
            frame_type, payload = frame
            if frame_type == wire.HELLO:
                self._hello(wire.decode_hello(payload))
            elif frame_type == wire.EVENT:
                self.app.on_event(wire.decode_event(payload))
            elif frame_type == wire.RESPONSE:
                self._response(wire.decode_response(payload))
            else:
                raise WireError("program received illegal frame type %d" % frame_type)

    # ----------------------------------------------------------- frames

    def emit(self, method, params=None, on_response=None):
        if self.hello is None:
            raise WireError("Result before HELLO")
        self.request_id += 1
        request_id = self.request_id
        if on_response is not None:
            self.pending[request_id] = on_response
        payload = wire.encode_result(request_id, self.epoch, method, params or {})
        try:
            self._write(wire.RESULT, payload)
        except Exception:
            self.pending.pop(request_id, None)
            raise
        return request_id

    def commit(self, root, claim=None, all_keys=False):
        """Send a full view snapshot; returns the new rev."""
        if self.hello is None:
            raise WireError("View before HELLO")
        if root is None:
            raise WireError("nil root box")
        self.rev += 1
        payload = wire.encode_view(self.epoch, self.rev, claim or [], bool(all_keys), root)
        self._write(wire.VIEW, payload)
        return self.rev

    def log(self, level, message):
        """One logging hook; stderr by default, never stdout."""
        if self._log is not None:
            self._log(level, message)
            return
        stream = sys.stderr
        try:
            stream.write("[tui2sdk] %s %s\n" % (level, message))
            stream.flush()
        except Exception:
            pass

    def _write(self, frame_type, payload):
        self._out.write(wire.frame(frame_type, payload))
        self._out.flush()

    # --------------------------------------------------------- dispatch

    def _hello(self, hello):
        self.hello = hello
        self.epoch = int(hello.get("epoch") or 0)
        self.rev = 0
        self.pending = {}
        self.app.on_hello(hello)

    def _response(self, response):
        request_id = response.get("request_id")
        callback = self.pending.pop(request_id, None)
        if response.get("epoch") != self.epoch:
            callback = None
        if callback is not None:
            callback(response)
        self.app.on_response(response)
