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

const CLAIM = ['ctrl-p', '?'];

class RefProgram extends App {
  constructor() {
    super();
    this.lines = [];
    this.pane = false;
  }

  commit() {
    this.client.commit(builder.text(this.lines.join('\n')).build(), CLAIM, false);
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
