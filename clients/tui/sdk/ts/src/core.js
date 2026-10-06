/**
 * Protocol client: frame loop, typed events, Emit/Commit (Node stdlib only).
 * Mirrors the Go/Python SDK semantics: request ids are monotonic per
 * connection, rev resets on every HELLO, callbacks only fire for responses of
 * the current epoch. STREAM frames of access streams (access.stream.open)
 * dispatch to onStream and are sent back with sendStream. commitDelta sends a
 * VIEW_DELTA against the committed baseline when the host advertised
 * features["view_delta"] and the patch is smaller than the full VIEW, falling
 * back to a full snapshot otherwise (PROTOCOL §2.1).
 */

'use strict';

const wire = require('./wire');

// ViewDeltaFeature is the HELLO feature a host advertises to accept
// VIEW_DELTA frames (PROTOCOL §1, §2.1).
const VIEW_DELTA_FEATURE = 'view_delta';

class App {
  constructor() {
    this.client = null;
  }

  emit(method, params, onResponse) {
    if (!this.client) throw new wire.WireError('emit before Client.run');
    return this.client.emit(method, params, onResponse);
  }

  commit(root, claim, allKeys) {
    if (!this.client) throw new wire.WireError('commit before Client.run');
    return this.client.commit(root, claim, allKeys);
  }

  commitDelta(base, nextRoot, claim, allKeys) {
    if (!this.client) throw new wire.WireError('commitDelta before Client.run');
    return this.client.commitDelta(base, nextRoot, claim, allKeys);
  }

  supports(feature) {
    return Boolean(this.client && this.client.supports(feature));
  }

  dropBase() {
    if (this.client) this.client.dropBase();
  }

  sendStream(frame) {
    if (!this.client) throw new wire.WireError('sendStream before Client.run');
    return this.client.sendStream(frame);
  }

  log(level, message) {
    if (this.client) this.client.log(level, message);
    else process.stderr.write(`[tui2sdk] ${level} ${message}\n`);
  }

  onHello() {}
  onKey() {}
  onPaste() {}
  onMouse() {}
  onWheel() {}
  onResize() {}
  onSources() {}
  onNotice() {}
  onComponent() {}
  onViewRejected() {}
  onResponse() {}
  onStream() {}

  onEvent(event) {
    const names = {
      key: 'onKey', paste: 'onPaste', mouse: 'onMouse', wheel: 'onWheel',
      resize: 'onResize', sources: 'onSources', notice: 'onNotice',
      component: 'onComponent', view_rejected: 'onViewRejected',
    };
    const handler = names[event && event.kind];
    if (handler && typeof this[handler] === 'function') this[handler](event);
  }
}

class Client {
  constructor(streamIn, streamOut, log) {
    this.streamOut = streamOut || process.stdout;
    this.logHook = log || null;
    this.reader = new wire.FrameReader(streamIn || process.stdin);
    this.app = null;
    this.hello = null;
    this.epoch = 0;
    this.rev = 0;
    this.requestId = 0;
    this.pending = new Map();
    // features mirrors HELLO.features so a program can negotiate optional
    // capabilities (e.g. view_delta) without re-reading hello.
    this.features = {};
    // base/baseRev are the box tree and rev the host is assumed to hold after
    // the last frame this client sent in the epoch. A full VIEW or a
    // VIEW_DELTA sets them; a new HELLO (epoch reset) or a view_rejected drops
    // them, forcing the next commitDelta to be a full VIEW.
    this.base = null;
    this.baseRev = 0;
  }

  async run(app) {
    this.app = app;
    app.client = this;
    for (;;) {
      const frame = await this.reader.next();
      if (frame === null) return 0;
      if (frame.type === wire.HELLO) this.dispatchHello(wire.decodeHello(frame.payload));
      else if (frame.type === wire.EVENT) {
        const event = wire.decodeEvent(frame.payload);
        // The host rejected the last committed revision: its cache no longer
        // matches our baseline, so drop it before the program sees the event
        // (PROTOCOL §2.1).
        if (event.kind === 'view_rejected') this.dropBase();
        this.app.onEvent(event);
      }
      else if (frame.type === wire.RESPONSE) this.dispatchResponse(wire.decodeResponse(frame.payload));
      else if (frame.type === wire.STREAM) this.app.onStream(wire.decodeStream(frame.payload));
      else throw new wire.WireError(`program received illegal frame type ${frame.type}`);
    }
  }

  emit(method, params, onResponse) {
    if (!this.hello) throw new wire.WireError('Result before HELLO');
    this.requestId += 1;
    const requestId = this.requestId;
    if (onResponse) this.pending.set(requestId, onResponse);
    const payload = wire.encodeResult(requestId, this.epoch, method, params || {});
    try {
      this.streamOut.write(wire.frame(wire.RESULT, payload));
    } catch (error) {
      this.pending.delete(requestId);
      throw error;
    }
    return requestId;
  }

  commit(root, claim, allKeys) {
    if (!this.hello) throw new wire.WireError('View before HELLO');
    if (!root) throw new wire.WireError('nil root box');
    this.rev += 1;
    const payload = wire.encodeView(this.epoch, this.rev, claim || [], Boolean(allKeys), root);
    this.streamOut.write(wire.frame(wire.VIEW, payload));
    this.base = root;
    this.baseRev = this.rev;
    return this.rev;
  }

  supports(feature) {
    return Boolean(this.features[feature]);
  }

  hasBase() {
    return this.base !== null;
  }

  dropBase() {
    this.base = null;
    this.baseRev = 0;
  }

  // commitDelta commits nextRoot as a VIEW_DELTA when the host advertised
  // features["view_delta"] and a patch is expressible and smaller than the
  // full VIEW (PROTOCOL §2.1). Returns true when a delta was sent. `base` is
  // an optimisation hint; the client's own committed baseline is
  // authoritative. One call advances rev exactly once.
  commitDelta(base, nextRoot, claim, allKeys) {
    if (!this.hello) throw new wire.WireError('View before HELLO');
    if (!nextRoot) throw new wire.WireError('nil root box');
    // The committed baseline is authoritative: a delta is only valid against
    // the frame the host is known to hold. The caller's base is an
    // optimisation hint used only when it is that baseline, so a stale hint
    // (or a dropped base) always falls back to a full VIEW.
    if (base !== this.base) base = this.base;
    const prevRev = this.rev;
    const rev = prevRev + 1;

    let deltaPayload = null;
    if (this.supports(VIEW_DELTA_FEATURE) && base !== null) {
      const patches = wire.diffView(base, nextRoot);
      if (patches) deltaPayload = wire.encodeViewDelta(this.epoch, rev, prevRev, claim, allKeys, patches);
    }
    const viewPayload = wire.encodeView(this.epoch, rev, claim || [], Boolean(allKeys), nextRoot);
    // Cost guard: fall back to the full snapshot when the patch is not
    // actually smaller (PROTOCOL §2.1).
    const useDelta = deltaPayload !== null && deltaPayload.length < viewPayload.length;

    this.rev = rev;
    this.streamOut.write(useDelta
      ? wire.frame(wire.VIEW_DELTA, deltaPayload)
      : wire.frame(wire.VIEW, viewPayload));
    this.base = nextRoot;
    this.baseRev = rev;
    return useDelta;
  }

  sendStream(frame) {
    if (!frame) throw new wire.WireError('nil stream frame');
    this.streamOut.write(wire.frame(wire.STREAM, wire.encodeStream(frame)));
  }

  log(level, message) {
    if (this.logHook) {
      this.logHook(level, message);
      return;
    }
    process.stderr.write(`[tui2sdk] ${level} ${message}\n`);
  }

  dispatchHello(hello) {
    this.hello = hello;
    this.epoch = Number(hello.epoch || 0);
    this.features = Object.assign({}, hello.features || {});
    this.rev = 0;
    this.pending.clear();
    // A new epoch resets the host cache (PROTOCOL §0.5): the first frame must
    // be a full VIEW, so drop any baseline.
    this.dropBase();
    this.app.onHello(hello);
  }

  dispatchResponse(response) {
    const callback = response.epoch === this.epoch ? this.pending.get(response.request_id) : undefined;
    this.pending.delete(response.request_id);
    if (callback) callback(response);
    this.app.onResponse(response);
  }
}

module.exports = { App, Client };
