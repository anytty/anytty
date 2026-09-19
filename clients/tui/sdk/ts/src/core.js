/**
 * Protocol client: frame loop, typed events, Emit/Commit (Node stdlib only).
 * Mirrors the Go/Python SDK semantics: request ids are monotonic per
 * connection, rev resets on every HELLO, callbacks only fire for responses of
 * the current epoch.
 */

'use strict';

const wire = require('./wire');

class App {
  constructor() {
    this.client = null;
  }

  emit(method, params, onResponse) {
    if (!this.client) throw new wire.WireError('emit before Client.run');
    return this.client.emit(method, params, onResponse);
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
  }

  async run(app) {
    this.app = app;
    app.client = this;
    for (;;) {
      const frame = await this.reader.next();
      if (frame === null) return 0;
      if (frame.type === wire.HELLO) this.dispatchHello(wire.decodeHello(frame.payload));
      else if (frame.type === wire.EVENT) this.app.onEvent(wire.decodeEvent(frame.payload));
      else if (frame.type === wire.RESPONSE) this.dispatchResponse(wire.decodeResponse(frame.payload));
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
    return this.rev;
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
    this.rev = 0;
    this.pending.clear();
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
