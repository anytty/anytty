'use strict';

// Windowed one-line-per-item lists. Build renders only the rows inside
// VisibleRange, so a huge list costs O(height). Port of
// tui2/sdk/widgets/list.go.

const { box, text, displayWidth, truncate } = require('../builder');

const DEFAULT_LIST_MARKER = '▸ ';

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

function padTo(value, width) {
  let out = truncate(value, width);
  const pad = width - displayWidth(out);
  if (pad > 0) out += ' '.repeat(pad);
  return out;
}

// chromeRows counts the fixed rows a header and a footer occupy.
function chromeRows(header, footer) {
  let rows = 0;
  if (header !== '') rows++;
  if (footer !== '') rows++;
  return rows;
}

// windowSize returns the visible window rows for a declared height; a height
// <= 0 means "all rows minus chrome".
function windowSize(height, chrome, count) {
  let size = height;
  if (size <= 0) size = count;
  size -= chrome;
  if (size < 0) size = 0;
  if (size > count) size = count;
  return size;
}

// clampIndex folds i into [0, count-1]; an empty set selects 0.
function clampIndex(i, count) {
  if (i < 0 || count <= 0) return 0;
  if (i >= count) return count - 1;
  return i;
}

// clampOffset folds offset into [0, count-size].
function clampOffset(offset, size, count) {
  if (size <= 0) return 0;
  if (offset < 0) return 0;
  const max = count - size;
  if (offset > max) return max;
  return offset;
}

// followOffset scrolls offset the minimum amount that puts selected inside a
// size-row window.
function followOffset(offset, selected, size) {
  if (size <= 0) return 0;
  if (selected < offset) offset = selected;
  if (selected >= offset + size) offset = selected - size + 1;
  if (offset < 0) return 0;
  return offset;
}

// moveWindow applies a selection delta and follows it when requested.
function moveWindow(offset, selected, delta, size, count, follow) {
  selected = clampIndex(selected + delta, count);
  if (follow) offset = followOffset(offset, selected, size);
  return [clampOffset(offset, size, count), selected];
}

function listBox(id, width, height) {
  const node = box('col');
  if (id !== '') node.id(id);
  if (width > 0) node.width(width);
  if (height > 0) node.height(height);
  return node;
}

function listRowBox(value, style, id, width) {
  if (width > 0) value = padTo(value, width);
  const node = text(value).height(1);
  if (width > 0) node.width(width);
  if (style !== '') node.style(style);
  if (id !== '') node.id(id).input('mouse');
  return node;
}

class List {
  constructor(options = {}) {
    this.id = options.id || '';
    this.items = options.items ? options.items.slice() : [];
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.offset = options.offset || 0;
    this.selected = options.selected || 0;
    this.follow = Boolean(options.follow);
    this.header = options.header || '';
    this.footer = options.footer || '';
    this.empty = options.empty || '';
    this.marker = options.marker || '';
    this.style = options.style || '';
    this.selectedStyle = options.selectedStyle || '';
    this.headerStyle = options.headerStyle || '';
    this.footerStyle = options.footerStyle || '';
    this.emptyStyle = options.emptyStyle || '';
    this.rowId = options.rowId || null;
    this.rowStyle = options.rowStyle || null;
  }

  visibleRange() {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    if (size <= 0) return [0, 0];
    let offset = this.offset;
    if (this.follow) offset = followOffset(offset, clampIndex(this.selected, this.items.length), size);
    offset = clampOffset(offset, size, this.items.length);
    return [offset, offset + size];
  }

  ensureVisible() {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    if (size <= 0 || this.items.length === 0) return;
    this.selected = clampIndex(this.selected, this.items.length);
    this.offset = followOffset(this.offset, this.selected, size);
    this.offset = clampOffset(this.offset, size, this.items.length);
  }

  move(delta) {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    const [offset, selected] = moveWindow(this.offset, this.selected, delta, size, this.items.length, this.follow);
    this.offset = offset;
    this.selected = selected;
  }

  pageUp() {
    this.move(-windowSize(this.height, chromeRows(this.header, this.footer), this.items.length));
  }

  pageDown() {
    this.move(windowSize(this.height, chromeRows(this.header, this.footer), this.items.length));
  }

  top() {
    this.selected = 0;
    this.offset = 0;
  }

  bottom() {
    if (this.items.length === 0) {
      this.selected = 0;
      this.offset = 0;
      return;
    }
    this.selected = this.items.length - 1;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    if (size > 0) this.offset = clampOffset(this.items.length - size, size, this.items.length);
  }

  // rowAt returns the absolute List row at viewport y, accounting for the
  // header and the current window.
  rowAt(y) {
    let top = 0;
    if (this.header !== '') top++;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    if (size <= 0 || y < top || y >= top + size) return null;
    const [start] = this.visibleRange();
    const row = start + (y - top);
    if (row < 0 || row >= this.items.length) return null;
    return row;
  }

  // onWheel applies one wheel event to the offset, leaving selection intact.
  onWheel(ev) {
    if (ev == null) return;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.items.length);
    this.offset = applyWheel(this.offset, this.items.length, size, ev.delta || 0);
  }

  row(index) {
    let value = '  ' + this.items[index];
    let style = this.style;
    if (index === this.selected) {
      value = firstNonEmpty(this.marker, DEFAULT_LIST_MARKER) + this.items[index];
      if (this.selectedStyle !== '') style = this.selectedStyle;
    }
    if (this.rowStyle != null) {
      const override = this.rowStyle(index, index === this.selected);
      if (override) style = override;
    }
    let id = '';
    if (this.rowId != null) id = this.rowId(index) || '';
    return listRowBox(value, style, id, this.width);
  }

  build() {
    const col = listBox(this.id, this.width, this.height);
    if (this.header !== '') col.child(listRowBox(this.header, this.headerStyle, '', this.width));
    const [start, end] = this.visibleRange();
    if (end > start) {
      for (let i = start; i < end; i++) col.child(this.row(i));
    } else if (this.items.length === 0 && this.empty !== '') {
      col.child(listRowBox(this.empty, this.emptyStyle, '', this.width));
    }
    if (this.footer !== '') col.child(listRowBox(this.footer, this.footerStyle, '', this.width));
    return col;
  }
}

class VirtualList {
  constructor(options = {}) {
    this.id = options.id || '';
    this.rows = options.rows ? options.rows.slice() : [];
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.offset = options.offset || 0;
    this.selected = options.selected || 0;
    this.follow = Boolean(options.follow);
    this.header = options.header || '';
    this.footer = options.footer || '';
    this.empty = options.empty || '';
    this.marker = options.marker || '';
    this.style = options.style || '';
    this.selectedStyle = options.selectedStyle || '';
    this.disabledStyle = options.disabledStyle || '';
    this.headerStyle = options.headerStyle || '';
    this.footerStyle = options.footerStyle || '';
    this.emptyStyle = options.emptyStyle || '';
  }

  visibleRange() {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    if (size <= 0) return [0, 0];
    let offset = this.offset;
    if (this.follow) offset = followOffset(offset, clampIndex(this.selected, this.rows.length), size);
    offset = clampOffset(offset, size, this.rows.length);
    return [offset, offset + size];
  }

  ensureVisible() {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    if (size <= 0 || this.rows.length === 0) return;
    this.selected = clampIndex(this.selected, this.rows.length);
    this.offset = followOffset(this.offset, this.selected, size);
    this.offset = clampOffset(this.offset, size, this.rows.length);
  }

  move(delta) {
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    const [offset, selected] = moveWindow(this.offset, this.selected, delta, size, this.rows.length, this.follow);
    this.offset = offset;
    this.selected = selected;
  }

  pageUp() {
    this.move(-windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length));
  }

  pageDown() {
    this.move(windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length));
  }

  top() {
    this.selected = 0;
    this.offset = 0;
  }

  bottom() {
    if (this.rows.length === 0) {
      this.selected = 0;
      this.offset = 0;
      return;
    }
    this.selected = this.rows.length - 1;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    if (size > 0) this.offset = clampOffset(this.rows.length - size, size, this.rows.length);
  }

  rowAt(y) {
    let top = 0;
    if (this.header !== '') top++;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    if (size <= 0 || y < top || y >= top + size) return null;
    const [start] = this.visibleRange();
    const row = start + (y - top);
    if (row < 0 || row >= this.rows.length) return null;
    return row;
  }

  onWheel(ev) {
    if (ev == null) return;
    const size = windowSize(this.height, chromeRows(this.header, this.footer), this.rows.length);
    this.offset = applyWheel(this.offset, this.rows.length, size, ev.delta || 0);
  }

  row(index) {
    const item = this.rows[index] || { text: '' };
    const itemText = item.text == null ? '' : item.text;
    let value = '  ' + itemText;
    let style = firstNonEmpty(item.style, this.style);
    if (item.disabled) style = firstNonEmpty(this.disabledStyle, style);
    if (index === this.selected) {
      value = firstNonEmpty(this.marker, DEFAULT_LIST_MARKER) + itemText;
      style = firstNonEmpty(this.selectedStyle, style);
    }
    return listRowBox(value, style, item.id || '', this.width);
  }

  build() {
    const col = listBox(this.id, this.width, this.height);
    if (this.header !== '') col.child(listRowBox(this.header, this.headerStyle, '', this.width));
    const [start, end] = this.visibleRange();
    if (end > start) {
      for (let i = start; i < end; i++) col.child(this.row(i));
    } else if (this.rows.length === 0 && this.empty !== '') {
      col.child(listRowBox(this.empty, this.emptyStyle, '', this.width));
    }
    if (this.footer !== '') col.child(listRowBox(this.footer, this.footerStyle, '', this.width));
    return col;
  }
}

// applyWheel folds one wheel delta into an offset and clamps it to
// [0, total-visible]; a fully visible (or empty) set pins the offset to 0.
function applyWheel(offset, total, visible, delta) {
  offset += delta;
  if (visible <= 0 || visible >= total) return 0;
  return clampOffset(offset, visible, total);
}

module.exports = {
  DEFAULT_LIST_MARKER,
  List,
  VirtualList,
};
