'use strict';

const { box, row, text, displayWidth, truncate } = require('../builder');
const { STYLE_BORDER, STYLE_STRONG_FOREGROUND } = require('./tokens');

class BorderSet {
  constructor(options = {}) {
    this.topLeft = options.topLeft || '';
    this.top = options.top || '';
    this.topRight = options.topRight || '';
    this.left = options.left || '';
    this.right = options.right || '';
    this.bottomLeft = options.bottomLeft || '';
    this.bottom = options.bottom || '';
    this.bottomRight = options.bottomRight || '';
  }
}

function borderNormal() {
  return new BorderSet({
    topLeft: '┌', top: '─', topRight: '┐',
    left: '│', right: '│',
    bottomLeft: '└', bottom: '─', bottomRight: '┘',
  });
}

function borderRounded() {
  return new BorderSet({
    topLeft: '╭', top: '─', topRight: '╮',
    left: '│', right: '│',
    bottomLeft: '╰', bottom: '─', bottomRight: '╯',
  });
}

function borderThick() {
  return new BorderSet({
    topLeft: '┏', top: '━', topRight: '┓',
    left: '┃', right: '┃',
    bottomLeft: '┗', bottom: '━', bottomRight: '┛',
  });
}

function borderDouble() {
  return new BorderSet({
    topLeft: '╔', top: '═', topRight: '╗',
    left: '║', right: '║',
    bottomLeft: '╚', bottom: '═', bottomRight: '╝',
  });
}

function borderSetOrDefault(set) {
  if (!set) return borderNormal();
  if (set.topLeft || set.top || set.topRight || set.left || set.right ||
      set.bottomLeft || set.bottom || set.bottomRight) {
    return set;
  }
  return borderNormal();
}

class BorderBox {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.border = options.border || null;
    this.style = options.style || '';
    this.titleStyle = options.titleStyle || '';
    this.child = options.child || null;
  }

  build() {
    const set = borderSetOrDefault(this.border);
    const style = this.style || STYLE_BORDER;
    const titleStyle = this.titleStyle || STYLE_STRONG_FOREGROUND;

    let width = this.width;
    let height = this.height;
    if (width <= 0) {
      width = boxChildWidth(this.child) + 2;
      if (width < 2) width = 2;
    }
    if (height <= 0) {
      height = 2;
      if (this.child) height++;
    }

    const col = box('col');
    if (this.id) col.id(this.id);
    col.width(width).height(height);

    if (width < 2 || height < 2) {
      if (this.child) col.child(this.child);
      return col;
    }

    const inner = width - 2;
    col.child(this.topEdge(set, titleStyle, inner));
    const body = height - 2;
    for (let i = 0; i < body; i++) {
      const line = row().height(1);
      line.child(text(set.left).style(style).width(1));
      if (this.child && i === 0) {
        line.child(this.child);
      } else {
        line.child(text(' '.repeat(inner)).width(inner).flex(1));
      }
      line.child(text(set.right).style(style).width(1));
      col.child(line);
    }
    col.child(text(this.bottomEdge(set, inner)).style(style).width(width).height(1));
    return col;
  }

  topEdge(set, titleStyle, inner) {
    const borderStyle = this.style || STYLE_BORDER;
    const line = row().height(1);
    line.child(text(set.topLeft).style(borderStyle).width(1));
    if (!this.title) {
      line.child(text(set.top.repeat(inner)).style(borderStyle).width(inner).flex(1));
      line.child(text(set.topRight).style(borderStyle).width(1));
      return line;
    }
    let fill = inner - 1;
    if (fill < 0) fill = 0;
    const label = truncate(' ' + this.title + ' ', fill + 1);
    const labelWidth = displayWidth(label);
    const rest = Math.max(0, inner - labelWidth);
    line.child(text(label).style(titleStyle));
    if (rest > 0) {
      line.child(text(set.top.repeat(rest)).style(borderStyle).width(rest).flex(1));
    }
    line.child(text(set.topRight).style(borderStyle).width(1));
    return line;
  }

  bottomEdge(set, inner) {
    return set.bottomLeft + set.bottom.repeat(inner) + set.bottomRight;
  }
}

function boxChildWidth(child) {
  if (!child) return 0;
  const node = child.build();
  if (node.size && node.size[0] > 0) return node.size[0];
  if (node.content) {
    let width = 0;
    if (node.content.text) width = displayWidth(node.content.text);
    for (const line of node.content.lines || []) {
      const lineWidth = displayWidth(line);
      if (lineWidth > width) width = lineWidth;
    }
    return width;
  }
  return 0;
}

module.exports = {
  BorderSet,
  borderNormal,
  borderRounded,
  borderThick,
  borderDouble,
  borderSetOrDefault,
  BorderBox,
};
