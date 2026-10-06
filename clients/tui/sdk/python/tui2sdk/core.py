"""Protocol client: frame loop, typed events, Emit/Commit (stdlib only).

``Client`` owns one connection (PROTOCOL §0.5):

  - it reads HELLO/EVENT/RESPONSE/STREAM and dispatches to a typed ``App``;
  - ``emit(method, params, on_response)`` sends a RESULT with a fresh
    request_id and stores the per-request callback; the callback fires exactly
    once, and only for a RESPONSE of the current epoch (request_id is scoped
    to ``(epoch, connection)``);
  - ``commit(root, claim, all_keys)`` sends a full VIEW snapshot; rev
    increments per epoch and resets on every HELLO (atomic epoch reset);
  - ``commit_delta(base, next, claim)`` sends a VIEW_DELTA patch against the
    client's committed baseline when the host advertised
    ``features["view_delta"]`` and the patch is smaller than the full VIEW;
    otherwise it falls back to a full snapshot (PROTOCOL §2.1). ``drop_base``
    discards the baseline so the next commit is full again;
  - ``log(level, message)`` is the one logging hook; layout programs never
    write to stdout outside frames.

Typed events dispatch through ``App.on_event``: a subclass overrides
``on_key``/``on_paste``/``on_mouse``/``on_wheel``/``on_resize``/``on_sources``/
``on_notice``/``on_component``/``on_view_rejected`` (named after the protocol)
or overrides ``on_event`` to handle everything itself. STREAM frames of the
access streams opened with ``access.stream.open`` dispatch to ``App.on_stream``
and are sent back with ``App.send_stream``/``Client.send_stream``.
"""

import sys

from . import wire
from .wire import WireError


# ViewDeltaFeature is the HELLO feature a host advertises to accept
# VIEW_DELTA frames (PROTOCOL §1, §2.1).
VIEW_DELTA_FEATURE = "view_delta"


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

    def commit(self, root, claim=None, all_keys=False):
        """Send a full view snapshot; returns the new rev."""
        if self.client is None:
            raise WireError("commit before Client.run")
        return self.client.commit(root, claim, all_keys)

    def commit_delta(self, base, next_root, claim=None, all_keys=False):
        """Commit next_root as a VIEW_DELTA or fall back to a full VIEW; see
        ``Client.commit_delta`` (PROTOCOL §2.1)."""
        if self.client is None:
            raise WireError("commit_delta before Client.run")
        return self.client.commit_delta(base, next_root, claim, all_keys)

    def supports(self, feature):
        """Whether the last HELLO advertised ``feature`` (PROTOCOL §1)."""
        return self.client is not None and self.client.supports(feature)

    def drop_base(self):
        """Discard the delta baseline so the next commit is a full VIEW."""
        if self.client is not None:
            self.client.drop_base()

    def send_stream(self, frame):
        """Send one STREAM frame (access stream data/close/cancel)."""
        if self.client is None:
            raise WireError("send_stream before Client.run")
        return self.client.send_stream(frame)

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

    def on_stream(self, frame):
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
        # features mirrors HELLO.features so a program can negotiate optional
        # capabilities (e.g. view_delta) without re-reading hello.
        self.features = {}
        # base/base_rev are the box tree and rev the host is assumed to hold
        # after the last frame this client sent in the epoch. A full VIEW or a
        # VIEW_DELTA sets them; a new HELLO (epoch reset) or a view_rejected
        # drops them, forcing the next commit to be a full VIEW.
        self.base = None
        self.base_rev = 0

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
                event = wire.decode_event(payload)
                if event.get("kind") == "view_rejected":
                    # The host rejected the last committed revision: its cache
                    # no longer matches our baseline, so drop it before the
                    # program sees the event (PROTOCOL §2.1).
                    self.drop_base()
                self.app.on_event(event)
            elif frame_type == wire.RESPONSE:
                self._response(wire.decode_response(payload))
            elif frame_type == wire.STREAM:
                self.app.on_stream(wire.decode_stream(payload))
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
        self.base = root
        self.base_rev = self.rev
        return self.rev

    def supports(self, feature):
        """Whether the last HELLO advertised ``feature`` (PROTOCOL §1). False
        before the handshake and for an absent feature."""
        return bool(self.features.get(feature))

    def has_base(self):
        """Whether a committed view baseline exists in the current epoch."""
        return self.base is not None

    def base_rev(self):
        """The rev of the current baseline, or 0 when there is none."""
        return self.base_rev

    def drop_base(self):
        """Discard the baseline so the next ``commit_delta`` sends a full VIEW
        (PROTOCOL §2.1; used on view_rejected / epoch reset)."""
        self.base = None
        self.base_rev = 0

    def commit_delta(self, base, next_root, claim=None, all_keys=False):
        """Commit ``next_root`` as a VIEW_DELTA when the host advertised
        ``features["view_delta"]`` and a patch is expressible and smaller than
        the full VIEW (PROTOCOL §2.1). Returns True when a delta was sent.

        ``base`` is an optimisation hint; the client's own committed baseline
        is authoritative. One call advances rev exactly once; the baseline
        always becomes ``next_root`` on success."""
        if self.hello is None:
            raise WireError("View before HELLO")
        if next_root is None:
            raise WireError("nil root box")
        # The committed baseline is authoritative: a delta is only valid
        # against the frame the host is known to hold. The caller's base is an
        # optimisation hint used only when it is that baseline, so a stale hint
        # (or a dropped base) always falls back to a full VIEW.
        if base is not self.base:
            base = self.base
        prev_rev = self.rev
        rev = prev_rev + 1

        delta_payload = None
        if self.supports(VIEW_DELTA_FEATURE) and base is not None:
            patches = wire.diff_view(base, next_root)
            if patches:
                delta_payload = wire.encode_view_delta(self.epoch, rev, prev_rev, claim, all_keys, patches)

        view_payload = wire.encode_view(self.epoch, rev, claim or [], bool(all_keys), next_root)
        # Cost guard: fall back to the full snapshot when the patch is not
        # actually smaller (PROTOCOL §2.1).
        use_delta = delta_payload is not None and len(delta_payload) < len(view_payload)

        self.rev = rev
        self._write(wire.VIEW_DELTA if use_delta else wire.VIEW,
                    delta_payload if use_delta else view_payload)
        self.base = next_root
        self.base_rev = rev
        return use_delta

    def send_stream(self, frame):
        """Send one program -> host STREAM frame (data/close/cancel). The
        program allocates the stream ids (PROTOCOL §4 access.stream.open)."""
        if frame is None:
            raise WireError("nil stream frame")
        self._write(wire.STREAM, wire.encode_stream(frame))

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
        self.features = dict(hello.get("features") or {})
        self.rev = 0
        self.pending = {}
        # A new epoch resets the host cache (PROTOCOL §0.5): the first frame
        # must be a full VIEW, so drop any baseline.
        self.drop_base()
        self.app.on_hello(hello)

    def _response(self, response):
        request_id = response.get("request_id")
        callback = self.pending.pop(request_id, None)
        if response.get("epoch") != self.epoch:
            callback = None
        if callback is not None:
            callback(response)
        self.app.on_response(response)
