/**
 * Reference conformance program of the TS/JS SDK.
 *
 * `node tui2/sdk/ts/conformance.js` is the candidate for
 * `tui2-sdk-verify --cmd`; it implements the behaviour fixed by
 * tui2/conformance/README.zh-CN.md using only this SDK.
 */

'use strict';

const { App, Client } = require('./src/core');
const builder = require('./src/builder');
const wire = require('./src/wire');

const CLAIM = ['ctrl-p', '?'];

// ProbeFeatureInvalidFirstFrame is a non-standard HELLO feature that arms the
// reference program's test-only probe: on a restart epoch's first commit it
// deliberately violates PROTOCOL §2.1 by writing a VIEW_DELTA with rev_base=0
// instead of a full VIEW, so the host can reject it (reason base_mismatch) and
// the program can prove it resyncs with a full snapshot. A compliant SDK never
// sends this frame (it drops its baseline on HELLO), so the probe writes raw
// bytes around the SDK; it exists only so the fixtures can exercise the
// host-side rejection deterministically.
const PROBE_INVALID_FIRST = 'conformance.invalid_first_frame_delta';

class RefProgram extends App {
  constructor() {
    super();
    this.lines = [];
    this.pane = false;
    // lastRoot is the tree of the last commit, handed back to commitDelta as
    // the diff baseline. It exercises the VIEW_DELTA path when the host
    // advertised features["view_delta"].
    this.lastRoot = null;
    // probeInvalidFirst is armed by PROBE_INVALID_FIRST and consumed by the
    // next commit (the test-only invalid-first-frame probe).
    this.probeInvalidFirst = false;
  }

  commit() {
    // One child box per line: the runner flattens the tree to text, so the
    // rendered content is identical to a single text box, but an append is
    // now a small insert patch instead of a set carrying the whole log. That
    // lets the VIEW_DELTA path produce a delta smaller than the full VIEW
    // deterministically (PROTOCOL §2.1).
    const root = builder.col(...this.lines.map((line) => builder.text(line))).build();
    if (this.probeInvalidFirst) {
      this.probeInvalidFirst = false;
      if (this.writeInvalidFirstDelta(root)) {
        this.lastRoot = root;
        return;
      }
    }
    this.client.commitDelta(this.lastRoot, root, CLAIM, false);
    this.lastRoot = root;
  }

  // writeInvalidFirstDelta is the test-only probe (PROBE_INVALID_FIRST): it
  // writes a raw VIEW_DELTA with rev=1, rev_base=0 as the first frame of a
  // new epoch, deliberately violating PROTOCOL §2.1 so the host rejects it.
  // It bypasses the SDK, which correctly refuses to send such a frame.
  writeInvalidFirstDelta(root) {
    if (!this.lastRoot) return false;
    const patches = wire.diffView(this.lastRoot, root);
    if (!patches) return false;
    const payload = wire.encodeViewDelta(this.client.epoch, 1, 0, CLAIM, false, patches);
    this.client.streamOut.write(wire.frame(wire.VIEW_DELTA, payload));
    return true;
  }

  callbackLine(response) {
    const data = response.data || {};
    this.lines.push(
      `cb id=${response.request_id || 0} ok=${response.ok ? 1 : 0} text=${data.text || ''} ` +
      `endpoint=${data.endpoint || ''} data_id=${data.id || ''} rows=${(data.rows || []).length} ` +
      `err=${response.error || ''}`);
  }

  emitRead() {
    const requestId = this.client.emit('clipboard.read', null, (response) => this.callbackLine(response));
    this.lines.push(`emit clipboard.read id=${requestId}`);
  }

  onHello(hello) {
    this.pane = false;
    this.probeInvalidFirst = Boolean((hello.features || {})[PROBE_INVALID_FIRST]);
    this.lines = [`hello view=${hello.view_id || ''} epoch=${hello.epoch || 0} cols=${hello.cols || 0} ` +
      `rows=${hello.rows || 0} schema=${hello.schema || 0}`];
    if ((hello.components || []).length) this.lines.push(`components=${hello.components.join(',')}`);
    if ((hello.methods || []).length) this.lines.push(`methods=${hello.methods.join(',')}`);
    this.emitRead();
    this.commit();
  }

  onKey(event) {
    if (event.key === 'ctrl-p') this.pane = true;
    this.lines.push(`key key=${event.key || ''} char=${event.char || ''} pane=${this.pane ? 1 : 0}`);
    if (event.key === 'ctrl-r') this.emitRead();
    this.commit();
  }

  onPaste(event) {
    this.lines.push(`paste id=${event.id || ''} text=${event.text || ''}`);
    this.commit();
  }

  onMouse(event) {
    this.lines.push(`mouse action=${event.action || ''} button=${event.button || ''} ` +
      `x=${event.x || 0} y=${event.y || 0} node=${event.node || ''}`);
    this.commit();
  }

  onWheel(event) {
    this.lines.push(`wheel delta=${event.delta || 0} x=${event.x || 0} y=${event.y || 0} node=${event.node || ''}`);
    this.commit();
  }

  onResize(event) {
    this.lines.push(`resize cols=${event.cols || 0} rows=${event.rows || 0}`);
    this.commit();
  }

  onSources(event) {
    const items = event.items || [];
    this.lines.push(`sources count=${items.length}`);
    for (const item of items) {
      this.lines.push(`source id=${item.id || ''} kind=${item.kind || ''} endpoint=${item.endpoint || ''} ` +
        `terminal=${item.terminal_id || ''} exited=${item.exited ? 1 : 0}`);
    }
    for (const item of items) {
      if (item.kind === 'terminal' && item.terminal_id) {
        const requestId = this.client.emit('terminal.attach',
          { endpoint: item.endpoint || '', id: item.terminal_id, fit: true },
          (response) => this.callbackLine(response));
        this.lines.push(`emit terminal.attach id=${requestId}`);
        break;
      }
    }
    this.commit();
  }

  onNotice(event) {
    this.lines.push(`notice level=${event.level || ''} message=${event.message || ''}`);
    this.commit();
  }

  onComponent(event) {
    this.lines.push(`component source=${event.source || ''} name=${event.name || ''} value=${event.value || ''}`);
    this.commit();
  }

  onViewRejected(event) {
    this.lines.push(`view-rejected epoch=${event.epoch || 0} rev=${event.rev || 0} reason=${event.reason || ''}`);
    // PROTOCOL §2.1: after a rejection the next frame must be a full VIEW.
    this.lastRoot = null;
    this.commit();
  }

  onStream(frame) {
    const payload = frame.payload || Buffer.alloc(0);
    this.lines.push(`stream id=${frame.stream_id || 0} kind=${frame.kind || ''} ` +
      `wire_type=${frame.wire_type || 0} payload=${payload.toString('hex')}`);
    this.commit();
  }

  onResponse(response) {
    this.lines.push(`resp id=${response.request_id || 0} epoch=${response.epoch || 0} ` +
      `ok=${response.ok ? 1 : 0} err=${response.error || ''}`);
    this.commit();
  }
}

async function main() {
  return new Client(process.stdin, process.stdout).run(new RefProgram());
}

if (require.main === module) {
  main().then((code) => process.exit(code)).catch((error) => {
    process.stderr.write(`tui2sdk-ts: ${error.message}\n`);
    process.exit(1);
  });
}

module.exports = { RefProgram, main };
