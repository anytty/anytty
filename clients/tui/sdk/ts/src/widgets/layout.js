'use strict';

const { row, text, displayWidth } = require('../builder');
const {
  Frame,
  FrameRow,
  Segment,
  DEFAULT_SEPARATOR,
  DEFAULT_SEP_STYLE,
  appendGroup,
  groupWidth,
  fitSegments,
  segmentTexts,
} = require('./basics');

class Rect {
  constructor(options = {}) {
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.w = options.w || 0;
    this.h = options.h || 0;
  }
}

function distribute(avail, ratios) {
  const out = new Array(ratios.length).fill(0);
  if (ratios.length === 0) return out;
  if (avail < ratios.length) avail = ratios.length;
  let total = 0;
  for (const ratio of ratios) {
    if (ratio > 0) total += ratio;
  }
  if (total <= 0) {
    total = ratios.length;
    out.fill(1);
  } else {
    for (let i = 0; i < ratios.length; i++) {
      out[i] = Math.floor(avail * ratios[i] / total);
      if (out[i] < 1) out[i] = 1;
    }
  }
  let used = 0;
  for (const value of out) used += value;
  out[out.length - 1] += avail - used;
  if (out[out.length - 1] < 1) out[out.length - 1] = 1;
  return out;
}

class SplitLayout {
  constructor(options = {}) {
    this.orient = options.orient || '';
    this.weights = options.weights ? options.weights.slice() : [];
    this.gap = options.gap || 0;
  }

  axis() {
    return this.orient === 'col' ? 'col' : 'row';
  }

  gapWidth() {
    return this.gap > 0 ? this.gap : 1;
  }

  rects(width, height) {
    const weights = this.weights.length === 0 ? [1] : this.weights;
    const count = weights.length;
    const gap = this.gapWidth();
    const panes = [];
    const dividers = [];
    if (this.axis() === 'col') {
      const avail = height - (count - 1) * gap;
      const sizes = distribute(avail, weights);
      let y = 0;
      for (let i = 0; i < sizes.length; i++) {
        const size = sizes[i];
        panes.push(new Rect({ x: 0, y, w: width, h: size }));
        y += size;
        if (i < count - 1) {
          dividers.push(new Rect({ x: 0, y, w: width, h: gap }));
          y += gap;
        }
      }
      return [panes, dividers];
    }
    const avail = width - (count - 1) * gap;
    const sizes = distribute(avail, weights);
    let x = 0;
    for (let i = 0; i < sizes.length; i++) {
      const size = sizes[i];
      panes.push(new Rect({ x, y: 0, w: size, h: height }));
      x += size;
      if (i < count - 1) {
        dividers.push(new Rect({ x, y: 0, w: gap, h: height }));
        x += gap;
      }
    }
    return [panes, dividers];
  }
}

class TitleBar {
  constructor(options = {}) {
    this.id = options.id || '';
    this.left = options.left || [];
    this.buttons = options.buttons || [];
    this.width = options.width || 0;
    this.fill = options.fill || '';
  }

  line() {
    const left = segmentsText(this.left);
    const right = this.buttons.map((button) => button.line()).join(' ');
    if (this.width > 0) {
      const pad = this.width - displayWidth(left) - displayWidth(right);
      if (pad > 0) return left + ' '.repeat(pad) + right;
    }
    return left + right;
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    let left = this.left;
    const right = this.buttons.map((button) => button.line()).join(' ');
    if (this.width > 0 && displayWidth(right) < this.width) {
      [left] = fitSegments(left, this.width - displayWidth(right), displayWidth(DEFAULT_SEPARATOR));
    }
    appendGroup(rowNode, left, this.fill, DEFAULT_SEPARATOR);
    if (this.width > 0) {
      const pad = this.width - segmentsWidth(left) - displayWidth(right);
      if (pad > 0) rowNode.child(text(' '.repeat(pad)));
    }
    for (let i = 0; i < this.buttons.length; i++) {
      if (i > 0) rowNode.child(text(' '));
      rowNode.child(this.buttons[i].build());
    }
    return rowNode;
  }
}

function segmentsText(segments) {
  return segments.map((segment) => segment.text).join('');
}

function segmentsWidth(segments) {
  let width = 0;
  for (const segment of segments) width += displayWidth(segment.text);
  return width;
}

class Footer {
  constructor(options = {}) {
    this.id = options.id || '';
    this.badge = options.badge || new Segment();
    this.hasBadge = Boolean(options.hasBadge);
    this.groups = options.groups || [];
    this.right = options.right || [];
    this.width = options.width || 0;
    this.separator = options.separator || '';
    this.sepStyle = options.sepStyle || '';
  }

  separatorText() {
    if (this.separator) return ' ' + this.separator + ' ';
    return ' ' + DEFAULT_SEPARATOR + ' ';
  }

  sepStyleValue() {
    return this.sepStyle || DEFAULT_SEP_STYLE;
  }

  leftSegments() {
    const left = [];
    if (this.hasBadge) left.push(this.badge);
    return left.concat(this.groups);
  }

  line() {
    const sep = this.separatorText();
    let left = this.leftSegments();
    let right = this.right;
    if (this.width > 0) {
      const sepW = displayWidth(sep);
      [right] = fitSegments(right, this.width, sepW);
      let budget = this.width - groupWidth(right, sepW);
      if (left.length > 0 && right.length > 0) budget -= sepW;
      [left] = fitSegments(left, budget, sepW);
    }
    const leftText = segmentTexts(left).join(sep);
    const rightText = segmentTexts(right).join(sep);
    if (this.width <= 0) {
      if (leftText !== '' && rightText !== '') return leftText + sep + rightText;
      return leftText + rightText;
    }
    let pad = this.width - displayWidth(leftText) - displayWidth(rightText);
    if (pad < 0) pad = 0;
    return leftText + ' '.repeat(pad) + rightText;
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    const sep = this.separatorText();
    const sepStyle = this.sepStyleValue();
    const sepW = displayWidth(sep);
    let left = this.leftSegments();
    let right = this.right;
    if (this.width > 0) {
      [right] = fitSegments(right, this.width, sepW);
      let leftBudget = this.width - groupWidth(right, sepW);
      if (left.length > 0 && right.length > 0) leftBudget -= sepW;
      [left] = fitSegments(left, leftBudget, sepW);
    }
    let used = groupWidth(left, sepW) + groupWidth(right, sepW);
    if (left.length > 0 && right.length > 0) used += sepW;
    appendGroup(rowNode, left, sepStyle, sep);
    if (left.length > 0 && right.length > 0) rowNode.child(text(sep).style(sepStyle));
    if (this.width > used) rowNode.child(text(' '.repeat(this.width - used)));
    appendGroup(rowNode, right, sepStyle, sep);
    return rowNode;
  }
}

class PickerRow {
  constructor(options = {}) {
    this.text = options.text || '';
    this.id = options.id || '';
    this.style = options.style || '';
    this.selected = Boolean(options.selected);
    this.selectable = Boolean(options.selectable);
  }
}

class Picker {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.rows = options.rows || [];
    this.style = options.style || '';
    this.marker = options.marker || '';
    this.selectedStyle = options.selectedStyle || '';
  }

  build() {
    const marker = this.marker || '▸ ';
    const rows = this.rows.map((pickerRow) => {
      let label = '  ' + pickerRow.text;
      let style = pickerRow.style;
      if (pickerRow.selected) {
        label = marker + pickerRow.text;
        if (this.selectedStyle) style = this.selectedStyle;
      }
      const frameRow = new FrameRow({ text: label, style });
      if (pickerRow.id) {
        frameRow.id = pickerRow.id;
        frameRow.input = ['mouse'];
      }
      return frameRow;
    });
    const width = this.width > 0 ? this.width : 40;
    const height = this.height > 0 ? this.height : rows.length + 2;
    return new Frame({
      id: this.id,
      title: this.title,
      width,
      height,
      style: this.style,
      rows,
    }).build();
  }
}

class FloatingLayer {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.style = options.style || '';
    this.rows = options.rows || [];
    this.collapsed = Boolean(options.collapsed);
  }

  build() {
    const height = this.collapsed ? 1 : this.height;
    const frame = new Frame({
      id: this.id,
      title: this.title,
      width: this.width,
      height,
      style: this.style,
      rows: this.rows,
    });
    return frame.build().pos(this.x, this.y);
  }
}

class Toast {
  constructor(options = {}) {
    this.id = options.id || '';
    this.text = options.text || '';
    this.style = options.style || '';
  }

  build() {
    const node = text(this.text);
    if (this.id) node.id(this.id);
    if (this.style) node.style(this.style);
    return node;
  }
}

module.exports = {
  Rect,
  distribute,
  SplitLayout,
  TitleBar,
  Footer,
  PickerRow,
  Picker,
  FloatingLayer,
  Toast,
};
