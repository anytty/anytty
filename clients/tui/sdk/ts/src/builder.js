/**
 * Chainable view-tree builder + text metrics (Node stdlib only).
 * Mirrors tui2/sdk/builder: a node is a plain object with protocol keys.
 */

'use strict';

class Node {
  constructor(flow) {
    this.box = {};
    if (flow) this.box.flow = flow;
  }

  id(value) { this.box.id = value; return this; }
  size(width = 0, height = 0, flex = 0) { this.box.size = [width, height, flex]; return this; }
  width(value) { return this.sizeAt(0, value); }
  height(value) { return this.sizeAt(1, value); }
  flex(value) { return this.sizeAt(2, value); }
  sizeAt(index, value) {
    const size = this.box.size ? this.box.size.slice() : [0, 0, 0];
    size[index] = value;
    this.box.size = size;
    return this;
  }

  pos(x, y) { this.box.pos = [x, y]; return this; }
  flow(value) { this.box.flow = value; return this; }
  style(value) { this.box.style = value; return this; }
  focused(value = true) { this.box.focused = Boolean(value); return this; }
  input(...kinds) {
    this.box.input = (this.box.input || []).concat(kinds.filter(Boolean));
    return this;
  }

  cursor(row = 0, col = 0, shape = '', visible = null) {
    this.box.cursor = { row, col, shape };
    if (visible !== null) this.box.cursor.visible = Boolean(visible);
    return this;
  }

  content(value) { this.box.content = { text: value }; return this; }
  lines(...values) { this.box.content = { lines: values.slice() }; return this; }
  selfRef(sourceId) { this.box.content = { self: sourceId }; return this; }

  props(values) {
    const content = this.box.content || {};
    content.props = Object.assign({}, content.props || {}, values || {});
    this.box.content = content;
    return this;
  }

  visible(value) { this.box.visible = Boolean(value); return this; }

  child(...nodes) {
    const children = this.box.children || (this.box.children = []);
    for (const node of nodes) if (node) children.push(node instanceof Node ? node.build() : node);
    return this;
  }

  build() { return JSON.parse(JSON.stringify(this.box)); }
}

function box(flow) { return new Node(flow); }
function col(...children) { return new Node('col').child(...children); }
function row(...children) { return new Node('row').child(...children); }
function stack(...children) { return new Node('stack').child(...children); }
function text(value) { return new Node().content(value); }
function terminal(sourceId) { return new Node().selfRef(sourceId); }
function divider(vertical = false, length = 1) {
  const node = new Node().input('mouse');
  return vertical ? node.width(1).height(length) : node.width(length).height(1);
}

// ------------------------------------------------------------ text metrics

function isWide(code) {
  return (code >= 0x1100 && code <= 0x115f) || (code >= 0x2e80 && code <= 0xa4cf) ||
    (code >= 0xac00 && code <= 0xd7a3) || (code >= 0xf900 && code <= 0xfaff) ||
    (code >= 0xfe30 && code <= 0xfe6f) || (code >= 0xff00 && code <= 0xff60) ||
    (code >= 0xffe0 && code <= 0xffe6) || (code >= 0x1f300 && code <= 0x1faff) ||
    (code >= 0x20000 && code <= 0x3fffd);
}

function cellWidth(ch) { return isWide(ch.codePointAt(0)) ? 2 : 1; }

function displayWidth(value) {
  let total = 0;
  for (const ch of value) total += cellWidth(ch);
  return total;
}

function truncate(value, maxWidth) {
  if (maxWidth <= 0) return '';
  let out = '';
  let used = 0;
  for (const ch of value) {
    const cells = cellWidth(ch);
    if (used + cells > maxWidth) break;
    out += ch;
    used += cells;
  }
  return out;
}

function centerPad(value, cells) {
  value = truncate(value, cells);
  const pad = cells - displayWidth(value);
  const left = Math.floor(pad / 2);
  return ' '.repeat(left) + value + ' '.repeat(pad - left);
}

function clamp(value, low, high) { return Math.max(low, Math.min(high, value)); }

module.exports = {
  Node, box, col, row, stack, text, terminal, divider,
  cellWidth, displayWidth, truncate, centerPad, clamp,
};
