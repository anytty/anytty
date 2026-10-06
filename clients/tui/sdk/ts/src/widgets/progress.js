'use strict';

const { row, text } = require('../builder');

const DEFAULT_PROGRESS_FULL = '█';
const DEFAULT_PROGRESS_EMPTY = '░';
const DEFAULT_PROGRESS_WIDTH = 20;
const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
const DEFAULT_TABLE_SEPARATOR = ' ';

function applyStyle(node, style) {
  if (style) node.style(style);
  return node;
}

function applyInput(node, kinds) {
  if (kinds && kinds.length > 0) node.input(...kinds);
  return node;
}

class ProgressBar {
  constructor(options = {}) {
    this.id = options.id || '';
    this.value = options.value || 0;
    this.max = options.max || 0;
    this.width = options.width || 0;
    this.label = options.label || '';
    this.hidePercent = Boolean(options.hidePercent);
    this.full = options.full || '';
    this.empty = options.empty || '';
    this.style = options.style || '';
    this.trackStyle = options.trackStyle || '';
    this.labelStyle = options.labelStyle || '';
    this.percentStyle = options.percentStyle || '';
  }

  fraction() {
    if (this.max <= 0) return 0;
    let value = this.value;
    if (value < 0) value = 0;
    if (value > this.max) value = this.max;
    return value / this.max;
  }

  percent() {
    return Math.floor(this.fraction() * 100 + 0.5);
  }

  barText() {
    const width = this.barWidth();
    let filled = Math.floor(this.fraction() * width + 0.5);
    if (filled > width) filled = width;
    return this.fullGlyph().repeat(filled) + this.emptyGlyph().repeat(width - filled);
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    if (this.label) rowNode.child(applyStyle(text(this.label).height(1), this.labelStyle));
    const width = this.barWidth();
    let filled = Math.floor(this.fraction() * width + 0.5);
    if (filled > width) filled = width;
    if (filled > 0) {
      rowNode.child(applyStyle(text(this.fullGlyph().repeat(filled)).height(1), this.style));
    }
    const empty = width - filled;
    if (empty > 0) {
      rowNode.child(applyStyle(text(this.emptyGlyph().repeat(empty)).height(1), this.trackStyle));
    }
    if (!this.hidePercent) {
      rowNode.child(applyStyle(text(this.percent() + '%').height(1), this.percentStyle));
    }
    return rowNode;
  }

  barWidth() {
    return this.width > 0 ? this.width : DEFAULT_PROGRESS_WIDTH;
  }

  fullGlyph() {
    return this.full || DEFAULT_PROGRESS_FULL;
  }

  emptyGlyph() {
    return this.empty || DEFAULT_PROGRESS_EMPTY;
  }
}

class Spinner {
  constructor(options = {}) {
    this.id = options.id || '';
    this.frame = options.frame || 0;
    this.frames = options.frames ? options.frames.slice() : [];
    this.style = options.style || '';
    this.label = options.label || '';
    this.labelStyle = options.labelStyle || '';
  }

  frameText() {
    const frames = this.frames.length > 0 ? this.frames : SPINNER_FRAMES;
    if (frames.length === 0) return '';
    const index = ((this.frame % frames.length) + frames.length) % frames.length;
    return frames[index];
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    rowNode.child(applyStyle(text(this.frameText()).height(1), this.style));
    if (this.label) rowNode.child(applyStyle(text(this.label).height(1), this.labelStyle));
    return rowNode;
  }
}

class Badge {
  constructor(options = {}) {
    this.id = options.id || '';
    this.text = options.text || '';
    this.style = options.style || '';
    this.input = options.input ? options.input.slice() : [];
  }

  build() {
    const node = text(this.text).height(1);
    if (this.id) node.id(this.id);
    applyStyle(node, this.style);
    applyInput(node, this.input);
    return node;
  }
}

class Tag {
  constructor(options = {}) {
    this.text = options.text || '';
    this.id = options.id || '';
    this.style = options.style || '';
    this.input = options.input ? options.input.slice() : [];
  }
}

class Tags {
  constructor(options = {}) {
    this.id = options.id || '';
    this.items = options.items || [];
    this.separator = options.separator || '';
    this.sepStyle = options.sepStyle || '';
    this.style = options.style || '';
  }

  build() {
    const rowNode = row().height(1);
    if (this.id) rowNode.id(this.id);
    const separator = this.separator || DEFAULT_TABLE_SEPARATOR;
    for (let i = 0; i < this.items.length; i++) {
      const tag = this.items[i];
      if (i > 0) rowNode.child(applyStyle(text(separator), this.sepStyle));
      const node = text(tag.text).height(1);
      applyStyle(node, tag.style || this.style);
      if (tag.id) node.id(tag.id);
      applyInput(node, tag.input);
      rowNode.child(node);
    }
    return rowNode;
  }
}

module.exports = {
  DEFAULT_PROGRESS_FULL,
  DEFAULT_PROGRESS_EMPTY,
  DEFAULT_PROGRESS_WIDTH,
  SPINNER_FRAMES,
  ProgressBar,
  Spinner,
  Badge,
  Tag,
  Tags,
};
