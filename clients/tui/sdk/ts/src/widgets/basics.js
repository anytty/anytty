'use strict';

const { box, row, text, displayWidth, truncate } = require('../builder');

const DEFAULT_ACTIVE_STYLE = 'tab_active';
const DEFAULT_INACTIVE_STYLE = 'tab_inactive';
const DEFAULT_CHROME_STYLE = 'chrome';
const DEFAULT_STATUS_STYLE = 'status';
const DEFAULT_SEP_STYLE = 'muted';
const DEFAULT_KEY_STYLE = 'muted';
const DEFAULT_BORDER_STYLE = 'border';
const DEFAULT_BUTTON_STYLE = 'accent';
const DEFAULT_SEPARATOR = '│';

function applyStyle(node, style) {
  if (style) node.style(style);
  return node;
}

function applyInput(node, kinds) {
  if (kinds && kinds.length > 0) node.input(...kinds);
  return node;
}

function maxInt(a, b) { return a > b ? a : b; }

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

class Segment {
  constructor(options = {}) {
    this.text = options.text || '';
    this.style = options.style || '';
    this.id = options.id || '';
    this.input = options.input ? options.input.slice() : [];
  }
}

class TabItem {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.active = Boolean(options.active);
  }
}

class TabBar {
  constructor(options = {}) {
    this.left = options.left || null;
    this.items = options.items || [];
    this.plus = Boolean(options.plus);
    this.plusId = options.plusId || '';
    this.plusText = options.plusText || '';
    this.activeStyle = options.activeStyle || '';
    this.inactiveStyle = options.inactiveStyle || '';
    this.chromeStyle = options.chromeStyle || '';
  }

  activeStyleValue() { return this.activeStyle || DEFAULT_ACTIVE_STYLE; }
  inactiveStyleValue() { return this.inactiveStyle || DEFAULT_INACTIVE_STYLE; }
  chromeStyleValue() { return this.chromeStyle || DEFAULT_CHROME_STYLE; }

  build() {
    const rowNode = row();
    if (this.left) rowNode.child(segmentBox(this.left, this.chromeStyleValue()));
    for (const item of this.items) {
      let label = ' ' + item.title + ' ';
      let style = this.inactiveStyleValue();
      if (item.active) {
        label = '[' + item.title + ']';
        style = this.activeStyleValue();
      }
      const node = text(label).style(style).input('mouse');
      if (item.id) node.id(item.id);
      rowNode.child(node);
    }
    if (this.plus) {
      const id = this.plusId || 'tab:new';
      const label = this.plusText || ' + ';
      rowNode.child(text(label).id(id).style(this.chromeStyleValue()).input('mouse'));
    }
    return rowNode;
  }
}

class StatusBar {
  constructor(options = {}) {
    this.id = options.id || '';
    this.left = options.left || [];
    this.right = options.right || [];
    this.width = options.width || 0;
    this.separator = options.separator || '';
    this.sepStyle = options.sepStyle || '';
  }

  text() {
    return this.segmentTexts().join(this.separatorText());
  }

  segmentTexts() {
    return this.left.map((segment) => segment.text)
      .concat(this.right.map((segment) => segment.text));
  }

  separatorText() {
    if (this.separator) return ' ' + this.separator + ' ';
    return ' ' + DEFAULT_SEPARATOR + ' ';
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    let left = this.left;
    let right = this.right;
    const sepText = this.separatorText();
    const sepStyle = this.sepStyle || DEFAULT_SEP_STYLE;
    const sepW = displayWidth(sepText);
    if (this.width > 0) {
      [right] = fitSegments(right, this.width, sepW);
      const rightW = groupWidth(right, sepW);
      let leftBudget = 0;
      if (left.length > 0) {
        leftBudget = this.width - rightW;
        if (right.length > 0) leftBudget -= sepW;
      }
      [left] = fitSegments(left, leftBudget, sepW);
    }
    let used = groupWidth(left, sepW) + groupWidth(right, sepW);
    if (left.length > 0 && right.length > 0) used += sepW;
    appendGroup(rowNode, left, sepStyle, sepText);
    if (left.length > 0 && right.length > 0) {
      rowNode.child(applyStyle(text(sepText), sepStyle));
    }
    if (this.width > 0) {
      const pad = this.width - used;
      if (pad > 0) rowNode.child(text(' '.repeat(pad)));
    }
    appendGroup(rowNode, right, sepStyle, sepText);
    return rowNode;
  }
}

function appendGroup(rowNode, segs, sepStyle, sepText) {
  for (let i = 0; i < segs.length; i++) {
    if (i > 0) rowNode.child(applyStyle(text(sepText), sepStyle));
    rowNode.child(segmentBox(segs[i], ''));
  }
}

function segmentBox(seg, fallbackStyle) {
  const node = text(seg.text);
  if (seg.id) node.id(seg.id);
  applyStyle(node, seg.style || fallbackStyle);
  applyInput(node, seg.input);
  return node;
}

function groupWidth(segs, sepW) {
  let width = 0;
  for (let i = 0; i < segs.length; i++) {
    if (i > 0) width += sepW;
    width += displayWidth(segs[i].text);
  }
  return width;
}

function fitSegments(segs, budget, sepW) {
  if (budget <= 0) return [[], 0];
  let out = segs.slice();
  while (out.length > 0) {
    const width = groupWidth(out, sepW);
    if (width <= budget) return [out, width];
    if (out.length === 1) {
      const trimmed = truncate(out[0].text, budget);
      out[0] = cloneWithText(out[0], trimmed);
      return [out, displayWidth(trimmed)];
    }
    out = out.slice(0, out.length - 1);
  }
  return [[], 0];
}

function cloneWithText(seg, value) {
  const copy = Object.assign(Object.create(Object.getPrototypeOf(seg)), seg);
  copy.text = value;
  return copy;
}

function segmentTexts(segs) {
  return segs.map((segment) => segment.text);
}

class FrameRow {
  constructor(options = {}) {
    this.text = options.text || '';
    this.style = options.style || '';
    this.id = options.id || '';
    this.input = options.input ? options.input.slice() : [];
  }
}

class Frame {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.style = options.style || '';
    this.rows = options.rows || [];
    this.fillStyle = options.fillStyle || '';
  }

  build() {
    let width = this.width;
    let height = this.height;
    if (width <= 0) width = frameTextWidth(this.rows) + 2;
    if (height <= 0) height = this.rows.length + 2;
    const style = this.style || DEFAULT_BORDER_STYLE;

    const col = box('col');
    if (this.id) col.id(this.id);
    col.width(width).height(height);

    if (width < 3 || height < 2) {
      for (const rowDef of this.rows) {
        col.child(frameRowBox(rowDef, maxInt(1, width), ''));
      }
      return col;
    }

    const inner = width - 2;
    col.child(text(ruleText(width, this.title)).style(style).width(width).height(1));
    for (let i = 0; i < height - 2; i++) {
      const rowDef = i < this.rows.length ? this.rows[i] : new FrameRow({ style: this.fillStyle });
      const line = row().height(1);
      line.child(text('│').style(style).width(1));
      line.child(frameRowBox(rowDef, inner, ''));
      line.child(text('│').style(style).width(1));
      col.child(line);
    }
    col.child(text(bottomRuleText(width)).style(style).width(width).height(1));
    return col;
  }
}

function frameRowBox(rowDef, width, fallbackStyle) {
  const style = rowDef.style || fallbackStyle;
  const node = text(padTo(rowDef.text, width)).width(width).height(1);
  if (rowDef.id) node.id(rowDef.id);
  applyStyle(node, style);
  applyInput(node, rowDef.input);
  return node;
}

function frameTextWidth(rows) {
  let width = 0;
  for (const rowDef of rows) {
    const rowWidth = displayWidth(rowDef.text);
    if (rowWidth > width) width = rowWidth;
  }
  return width;
}

function ruleText(width, title) {
  const inner = width - 2;
  if (inner <= 0) return '┌┐';
  if (!title) return '┌' + '─'.repeat(inner) + '┐';
  let label = ' ' + title + ' ';
  const budget = inner - 1;
  label = budget >= 0 ? truncate(label, budget) : '';
  let fill = inner - 1 - displayWidth(label);
  if (fill < 0) fill = 0;
  return '┌─' + label + '─'.repeat(fill) + '┐';
}

function bottomRuleText(width) {
  if (width < 2) return '└';
  return '└' + '─'.repeat(width - 2) + '┘';
}

function padTo(value, width) {
  let out = truncate(value, width);
  const pad = width - displayWidth(out);
  if (pad > 0) out += ' '.repeat(pad);
  return out;
}

class Divider {
  constructor(options = {}) {
    this.id = options.id || '';
    this.vertical = Boolean(options.vertical);
    this.length = options.length || 0;
    this.style = options.style || '';
    this.input = options.input ? options.input.slice() : [];
  }

  build() {
    let glyph = '─';
    let width = this.length;
    let height = 1;
    if (this.vertical) {
      glyph = '│';
      width = 1;
      height = this.length;
    }
    if (width <= 0) width = 1;
    if (height <= 0) height = 1;
    const style = this.style || DEFAULT_SEP_STYLE;
    const node = text(glyph.repeat(maxInt(1, this.length))).width(width).height(height);
    if (this.id) node.id(this.id);
    applyStyle(node, style);
    applyInput(node, this.input);
    return node;
  }
}

class Card {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.lines = options.lines || [];
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.style = options.style || '';
    this.lineStyle = options.lineStyle || '';
    this.center = Boolean(options.center);
  }

  build() {
    let lines = this.lines.slice();
    if (this.center) lines = centerLines(lines, this.width, this.height);
    const rows = lines.map((line) => new FrameRow({ text: line, style: this.lineStyle }));
    return new Frame({
      id: this.id,
      title: this.title,
      width: this.width,
      height: this.height,
      style: this.style,
      rows,
    }).build();
  }
}

function centerLines(lines, width, height) {
  let out = lines;
  if (width > 2) {
    const inner = width - 2;
    out = out.map((line) => {
      const pad = Math.floor((inner - displayWidth(line)) / 2);
      return pad > 0 ? ' '.repeat(pad) + line : line;
    });
  }
  if (height > 2) {
    const inner = height - 2;
    const pad = Math.floor((inner - out.length) / 2);
    if (pad > 0) out = new Array(pad).fill('').concat(out);
  }
  return out;
}

class Button {
  constructor(options = {}) {
    this.id = options.id || '';
    this.text = options.text || '';
    this.hot = options.hot || '';
    this.style = options.style || '';
    this.input = options.input ? options.input.slice() : [];
  }

  line() { return this.text; }

  build() {
    const inputs = this.input.length > 0 ? this.input : ['mouse'];
    const style = this.style || DEFAULT_BUTTON_STYLE;
    const idx = this.hot ? this.text.indexOf(this.hot) : -1;
    if (idx < 0) return this.plainBox(style, inputs);
    const rowNode = row();
    if (this.id) rowNode.id(this.id);
    applyInput(rowNode, inputs);
    const pre = this.text.slice(0, idx);
    const hot = this.text.slice(idx, idx + this.hot.length);
    const post = this.text.slice(idx + this.hot.length);
    if (pre) rowNode.child(applyStyle(text(pre), style));
    rowNode.child(applyStyle(text(hot), DEFAULT_BUTTON_STYLE));
    if (post) rowNode.child(applyStyle(text(post), style));
    return rowNode;
  }

  plainBox(style, inputs) {
    const node = text(this.text);
    if (this.id) node.id(this.id);
    applyStyle(node, style);
    applyInput(node, inputs);
    return node;
  }
}

class KeyHint {
  constructor(options = {}) {
    this.id = options.id || '';
    this.mode = options.mode || '';
    this.keys = options.keys ? options.keys.slice() : [];
    this.modeStyle = options.modeStyle || '';
    this.keyStyle = options.keyStyle || '';
    this.sepStyle = options.sepStyle || '';
  }

  text() {
    const keys = this.keys.join(' · ');
    if (!this.mode) return keys;
    if (!keys) return this.mode;
    return this.mode + ' ' + DEFAULT_SEPARATOR + ' ' + keys;
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    const modeStyle = this.modeStyle || DEFAULT_STATUS_STYLE;
    const keyStyle = this.keyStyle || DEFAULT_KEY_STYLE;
    const sepStyle = this.sepStyle || DEFAULT_SEP_STYLE;
    if (this.mode) rowNode.child(applyStyle(text(this.mode), modeStyle));
    if (this.keys.length > 0) {
      if (this.mode) rowNode.child(applyStyle(text(' ' + DEFAULT_SEPARATOR + ' '), sepStyle));
      rowNode.child(applyStyle(text(this.keys.join(' · ')), keyStyle));
    }
    return rowNode;
  }
}

module.exports = {
  DEFAULT_ACTIVE_STYLE,
  DEFAULT_INACTIVE_STYLE,
  DEFAULT_CHROME_STYLE,
  DEFAULT_STATUS_STYLE,
  DEFAULT_SEP_STYLE,
  DEFAULT_KEY_STYLE,
  DEFAULT_BORDER_STYLE,
  DEFAULT_BUTTON_STYLE,
  DEFAULT_SEPARATOR,
  Segment,
  TabItem,
  TabBar,
  StatusBar,
  FrameRow,
  Frame,
  Divider,
  Card,
  Button,
  KeyHint,
  appendGroup,
  segmentBox,
  groupWidth,
  fitSegments,
  segmentTexts,
  firstNonEmpty,
};
