/**
 * Command template: the smallest useful anytty layout program in TS/JS: a
 * process speaking the tui2 protocol on stdin/stdout through the official
 * TS/JS SDK (Node stdlib). Draws tab bar · two terminal slots · footer ·
 * picker; handles Ctrl-T/Ctrl-F/click/Esc/Ctrl-Q.
 * 改哪里: header()/footer() 文字 · key() 键位 · pick() 的 Enter 行为 ·
 * view() 的盒子树。Dev loop: bash clients/tui/scripts/dev.sh clients/tui/templates/ts
 */
'use strict';
const { App, Client, builder } = require('../../sdk/ts');
class Template extends App {
  constructor() {
    super();
    this.tabs = ['main'];
    this.active = 0;
    this.slots = ['', '']; // terminal source ids, empty = unbound
    this.focus = 0;
    this.picker = false;
    this.pickSel = 0;
    this.sources = [];
    this.status = '';
    this.cols = 80;
    this.rows = 24;
  }
  // ------------------------------------------------------------- protocol
  commit() {
    const claim = this.picker ? [] : ['ctrl-t', 'ctrl-f', 'esc'];
    this.client.commit(this.view(), claim, this.picker);
  }
  emitCreate() {
    this.emit('terminal.create', { endpoint: 'local' }, (response) => {
      if (response.ok) {
        const data = response.data || {};
        this.slots[this.focus] = `terminal:${data.endpoint || 'local'}:${data.id || ''}`;
        this.status = `bound ${data.id || ''}`;
      } else {
        this.status = `create failed: ${response.error}`;
      }
      this.commit();
    });
  }
  emitAttach(source) {
    const parts = source.id.split(':', 3);
    const endpoint = parts.length === 3 ? parts[1] : 'local';
    const terminalId = parts.length === 3 ? parts[2] : source.id;
    this.emit('terminal.attach', { endpoint, id: terminalId, fit: true }, (response) => {
      if (response.ok) {
        this.slots[this.focus] = source.id;
        this.status = `bound ${terminalId}`;
      } else {
        this.status = `attach failed: ${response.error}`;
      }
      this.commit();
    });
  }
  pick() {
    this.picker = false;
    if (this.pickSel >= this.sources.length) {
      this.status = 'creating terminal';
      this.emitCreate();
    } else {
      this.status = 'binding';
      this.emitAttach(this.sources[this.pickSel]);
    }
  }
  // --------------------------------------------------------------- events
  onHello(hello) {
    this.cols = hello.cols || this.cols;
    this.rows = hello.rows || this.rows;
    this.commit();
  }
  onSources(event) {
    this.sources = (event.items || []).filter((source) => source.kind === 'terminal');
    if (!this.slots.some(Boolean)) {
      if (this.sources.length > 0) {
        // 改哪里: 冷启动自动绑定第一个终端；想总是弹选择器就改成
        // this.picker = true; this.pickSel = 0;
        this.slots[0] = this.sources[0].id;
      } else {
        this.picker = true;
        this.pickSel = 0;
      }
    }
    this.commit();
  }
  onKey(event) {
    const key = event.key || event.char || '';
    if (this.picker) {
      if (key === 'up') this.pickSel = Math.max(0, this.pickSel - 1);
      else if (key === 'down') this.pickSel = Math.min(this.sources.length, this.pickSel + 1);
      else if (key === 'enter') this.pick();
      else if (key === 'esc' || key === 'ctrl-f') this.picker = false;
    } else {
      // 改哪里: NORMAL 模式键位。
      if (key === 'ctrl-f') {
        this.picker = true;
        this.pickSel = 0;
      } else if (key === 'ctrl-t') {
        this.tabs.push(String(this.tabs.length + 1));
        this.active = this.tabs.length - 1;
      } else if (key === 'esc') {
        this.emit('system.quit', null, null);
      }
    }
    this.commit();
  }
  onMouse(event) {
    if (event.action !== 'press') return;
    const node = event.node || '';
    if (node.startsWith('slot:')) {
      this.focus = builder.clamp(Number(node.slice(5)) || 0, 0, 1);
    } else if (node.startsWith('pick:')) {
      this.pickSel = builder.clamp(Number(node.slice(5)) || 0, 0, this.sources.length);
      this.pick();
    }
    this.commit();
  }
  onResize(event) {
    this.cols = event.cols || this.cols;
    this.rows = event.rows || this.rows;
    this.commit();
  }
  onNotice(event) {
    this.status = event.message || '';
    this.commit();
  }
  // ----------------------------------------------------------------- view
  view() {
    const root = builder.col(this.header(), this.body(), this.footer());
    if (this.picker) root.child(this.overlay());
    return root.build();
  }
  // 改哪里: tab 条（图标、标题、+ 按钮）。
  header() {
    const row = builder.row(builder.text(' main ').style('tab_inactive')).id('header').height(1);
    this.tabs.forEach((name, index) => {
      const label = index === this.active ? `[${name}]` : ` ${name} `;
      const style = index === this.active ? 'tab_active' : 'tab_inactive';
      row.child(builder.text(label).id(`tab:${index}`).style(style).input('mouse'));
    });
    return row.child(builder.text(' + ').id('tab:new').style('tab_inactive').input('mouse'));
  }
  body() {
    const row = builder.box('row').flex(1);
    this.slots.forEach((source, index) => {
      const id = `slot:${index}`;
      if (!source) {
        row.child(builder.col(builder.text('  [空槽]'), builder.text('  Ctrl-F 选择终端'))
          .id(id).style('muted').flex(1).input('mouse'));
      } else {
        row.child(builder.terminal(source).id(id).flex(1).focused(index === this.focus)
          .input('key', 'paste', 'wheel').props({ 'chrome.title': 'muted' }));
      }
    });
    return row;
  }
  // 改哪里: footer 键位提示。
  footer() {
    let hints = 'Ctrl-F picker · Ctrl-T tab · click focus · Esc quit';
    if (this.picker) hints = '↑/↓ select · Enter bind · Esc close';
    let status = `tab ${this.active + 1}/${this.tabs.length}`;
    if (this.status) status = `${this.status} │ ${status}`;
    return builder.row(builder.text(hints).style('muted'), builder.text(' ').flex(1),
      builder.text(builder.truncate(status, Math.floor(this.cols / 2))).style('status'))
      .id('footer').height(1);
  }
  overlay() {
    const rows = builder.col(builder.text(' select a terminal').style('muted')).id('picker');
    for (let index = 0; index <= this.sources.length; index += 1) {
      let label = '+ New terminal';
      let state = 'create';
      if (index < this.sources.length) {
        const source = this.sources[index];
        label = source.title || source.id;
        state = `${source.endpoint || 'local'} · ${source.exited ? 'exited' : 'live'}`;
      }
      const selected = index === this.pickSel;
      rows.child(builder.text(`${selected ? '> ' : '  '}${label}  ${state}`)
        .id(`pick:${index}`).style(selected ? 'selection' : '').input('mouse'));
    }
    rows.child(builder.text(' enter bind · esc close').style('muted'));
    const width = Math.min(46, Math.max(10, this.cols - 2));
    const height = this.sources.length + 4;
    return rows.pos(Math.max(0, Math.floor((this.cols - width) / 2)),
      Math.max(0, Math.floor((this.rows - height) / 2))).width(width).height(height);
  }
}

if (require.main === module) {
  new Client(process.stdin, process.stdout).run(new Template())
    .then((code) => process.exit(code))
    .catch((error) => {
      process.stderr.write(`template: ${error.message}\n`);
      process.exit(1);
    });
}