'use strict';

// Single-line and multi-line text editors: pure state plus a pure Build.
// Values are arrays of runes (one code point per element) so cursor indexes
// match Go's []rune semantics. Port of tui2/sdk/widgets/input.go.

const { box, row, text, displayWidth, truncate, cellWidth } = require('../builder');

const DEFAULT_CURSOR_STYLE = 'reverse';
const DEFAULT_CURSOR_SHAPE = 'bar';

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

function toRunes(value) {
  if (value == null) return [];
  if (Array.isArray(value)) return value.slice();
  return Array.from(value);
}

function clampCursor(cursor, length) {
  if (cursor < 0) return 0;
  if (cursor > length) return length;
  return cursor;
}

function insertRuneAt(state, r) {
  state.cursor = clampCursor(state.cursor, state.value.length);
  state.value.splice(state.cursor, 0, r);
  state.cursor++;
}

function insertRunesAt(state, runes) {
  state.cursor = clampCursor(state.cursor, state.value.length);
  state.value.splice(state.cursor, 0, ...runes);
  state.cursor += runes.length;
}

function filterRunes(runes, newlines) {
  const out = [];
  for (const r of runes) {
    if (r === '\n' && !newlines) continue;
    if (r === '\r') continue;
    out.push(r);
  }
  return out;
}

function isWordRune(ch) {
  return ch === '_' || /[\p{L}\p{Nd}]/u.test(ch);
}

function wordLeftIndex(value, cursor) {
  cursor = clampCursor(cursor, value.length);
  let i = cursor;
  while (i > 0 && !isWordRune(value[i - 1])) i--;
  while (i > 0 && isWordRune(value[i - 1])) i--;
  return i;
}

function wordRightIndex(value, cursor) {
  cursor = clampCursor(cursor, value.length);
  let i = cursor;
  while (i < value.length && isWordRune(value[i])) i++;
  while (i < value.length && !isWordRune(value[i])) i++;
  return i;
}

function cellWidthOf(runes) {
  let width = 0;
  for (const r of runes) width += cellWidth(r);
  return width;
}

function cursorCellWidth(runes, cursor) {
  if (cursor >= 0 && cursor < runes.length) {
    const width = cellWidth(runes[cursor]);
    if (width > 0) return width;
  }
  return 1;
}

function truncateRunesToCells(runes, width) {
  if (width <= 0) return [];
  let cells = 0;
  for (let i = 0; i < runes.length; i++) {
    const rw = cellWidth(runes[i]);
    if (rw > 0 && cells + rw > width) return runes.slice(0, i);
    cells += rw;
  }
  return runes.slice();
}

// printableRune extracts a single printable rune from a key event.
function printableRune(ev) {
  if (ev == null) return null;
  let value = ev.char;
  if (!value) value = ev.key;
  if (!value) return null;
  const runes = Array.from(value);
  if (runes.length !== 1 || /\p{Cc}/u.test(runes[0])) return null;
  return runes[0];
}

function cursorShape(shape) {
  if (shape !== '') return shape;
  return DEFAULT_CURSOR_SHAPE;
}

function inputRowBox(id, width) {
  const node = row().height(1);
  if (id !== '') node.id(id);
  if (width > 0) node.width(width);
  return node;
}

class TextInput {
  constructor(options = {}) {
    this.id = options.id || '';
    this.value = toRunes(options.value);
    this.cursor = options.cursor !== undefined ? options.cursor : 0;
    this.placeholder = options.placeholder || '';
    this.mask = options.mask || '';
    this.maxLen = options.maxLen || 0;
    this.width = options.width || 0;
    this.offset = options.offset || 0;
    this.focused = Boolean(options.focused);
    this.style = options.style || '';
    this.placeholderStyle = options.placeholderStyle || '';
    this.cursorStyle = options.cursorStyle || '';
    this.cursorShape = options.cursorShape || '';
    this.input = options.input ? options.input.slice() : [];
  }

  // text returns the raw value (mask only affects rendering).
  text() {
    return this.value.join('');
  }

  // displayText returns the rendered text: masked value, or the placeholder
  // when the value is empty.
  displayText() {
    if (this.value.length === 0 && this.placeholder !== '') return this.placeholder;
    return this.displayRunes().join('');
  }

  setValue(s) {
    this.value = filterRunes(Array.from(s), false);
    if (this.maxLen > 0 && this.value.length > this.maxLen) this.value = this.value.slice(0, this.maxLen);
    this.cursor = this.value.length;
  }

  insertRune(r) {
    if (r === '\n' || r === '\r') return false;
    if (this.maxLen > 0 && this.value.length >= this.maxLen) return false;
    insertRuneAt(this, r);
    return true;
  }

  insertString(s) {
    let runes = filterRunes(Array.from(s), false);
    if (runes.length === 0) return false;
    if (this.maxLen > 0) {
      const room = this.maxLen - this.value.length;
      if (room <= 0) return false;
      if (runes.length > room) runes = runes.slice(0, room);
    }
    insertRunesAt(this, runes);
    return true;
  }

  backspace() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor === 0) return false;
    this.value.splice(this.cursor - 1, 1);
    this.cursor--;
    return true;
  }

  delete() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor >= this.value.length) return false;
    this.value.splice(this.cursor, 1);
    return true;
  }

  deleteWordLeft() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const start = wordLeftIndex(this.value, this.cursor);
    if (start === this.cursor) return false;
    this.value.splice(start, this.cursor - start);
    this.cursor = start;
    return true;
  }

  deleteToEnd() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor >= this.value.length) return false;
    this.value = this.value.slice(0, this.cursor);
    return true;
  }

  left() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor === 0) return false;
    this.cursor--;
    return true;
  }

  right() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor >= this.value.length) return false;
    this.cursor++;
    return true;
  }

  wordLeft() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const next = wordLeftIndex(this.value, this.cursor);
    if (next === this.cursor) return false;
    this.cursor = next;
    return true;
  }

  wordRight() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const next = wordRightIndex(this.value, this.cursor);
    if (next === this.cursor) return false;
    this.cursor = next;
    return true;
  }

  home() {
    const moved = this.cursor !== 0;
    this.cursor = 0;
    return moved;
  }

  end() {
    const moved = this.cursor !== this.value.length;
    this.cursor = this.value.length;
    return moved;
  }

  // handleKey applies one key event ({key, char}) and reports consumption.
  handleKey(ev) {
    if (ev == null) return false;
    switch (ev.key) {
      case 'left': this.left(); return true;
      case 'right': this.right(); return true;
      case 'home': case 'ctrl-a': this.home(); return true;
      case 'end': case 'ctrl-e': this.end(); return true;
      case 'backspace': this.backspace(); return true;
      case 'delete': this.delete(); return true;
      case 'ctrl-left': case 'alt-left': case 'ctrl-b': this.wordLeft(); return true;
      case 'ctrl-right': case 'alt-right': case 'ctrl-f': this.wordRight(); return true;
      case 'ctrl-w': case 'alt-backspace': this.deleteWordLeft(); return true;
      case 'ctrl-k': this.deleteToEnd(); return true;
      case 'ctrl-u': this.deleteToEnd(); this.home(); return true;
      default: break;
    }
    const r = printableRune(ev);
    if (r != null) {
      this.insertRune(r);
      return true;
    }
    return false;
  }

  displayRunes() {
    if (this.mask === '') return this.value;
    return this.value.map(() => this.mask);
  }

  // visibleOffset scrolls the rune window so the cursor cell fits Width cells.
  visibleOffset(runes) {
    if (this.width <= 0) return 0;
    let offset = clampCursor(this.offset, runes.length);
    const cursor = clampCursor(this.cursor, runes.length);
    if (cursor < offset) offset = cursor;
    while (offset < cursor && cellWidthOf(runes.slice(offset, cursor)) + cursorCellWidth(runes, cursor) > this.width) {
      offset++;
    }
    return offset;
  }

  pad(node, used) {
    if (this.width <= 0 || used >= this.width) return;
    node.child(text(' '.repeat(this.width - used)));
  }

  inputs() {
    if (this.input.length > 0) return this.input;
    return ['key', 'paste'];
  }

  build() {
    const node = inputRowBox(this.id, this.width);
    node.focused(this.focused).input(...this.inputs());
    if (this.value.length === 0 && this.placeholder !== '') {
      let value = this.placeholder;
      if (this.width > 0) value = truncate(value, this.width);
      node.child(text(value).style(this.placeholderStyle).height(1));
      this.pad(node, displayWidth(value));
      if (this.focused) node.cursor(0, 0, cursorShape(this.cursorShape));
      return node;
    }
    const runes = this.displayRunes();
    const cursor = clampCursor(this.cursor, runes.length);
    const start = this.visibleOffset(runes);
    const visible = runes.slice(start);
    if (!this.focused) {
      let value = visible.join('');
      if (this.width > 0) value = padTo(value, this.width);
      if (value !== '') node.child(text(value).style(this.style).height(1));
      return node;
    }
    const rel = clampCursor(cursor - start, visible.length);
    const pre = visible.slice(0, rel).join('');
    let post = visible.slice(rel);
    let cursorText = ' ';
    if (post.length > 0) {
      cursorText = post[0];
      post = post.slice(1);
    }
    let postText = post.join('');
    if (this.width > 0) {
      postText = truncate(postText, this.width - displayWidth(pre) - displayWidth(cursorText));
    }
    if (pre !== '') node.child(text(pre).style(this.style).height(1));
    node.child(text(cursorText).style(firstNonEmpty(this.cursorStyle, DEFAULT_CURSOR_STYLE)).height(1));
    if (postText !== '') node.child(text(postText).style(this.style).height(1));
    this.pad(node, displayWidth(pre) + displayWidth(cursorText) + displayWidth(postText));
    node.cursor(0, displayWidth(pre), cursorShape(this.cursorShape));
    return node;
  }
}

class TextArea {
  constructor(options = {}) {
    this.id = options.id || '';
    this.value = toRunes(options.value);
    this.cursor = options.cursor !== undefined ? options.cursor : 0;
    this.placeholder = options.placeholder || '';
    this.mask = options.mask || '';
    this.maxLen = options.maxLen || 0;
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.rowOffset = options.rowOffset || 0;
    this.colOffset = options.colOffset || 0;
    this.focused = Boolean(options.focused);
    this.style = options.style || '';
    this.placeholderStyle = options.placeholderStyle || '';
    this.cursorStyle = options.cursorStyle || '';
    this.cursorShape = options.cursorShape || '';
    this.input = options.input ? options.input.slice() : [];
  }

  text() {
    return this.value.join('');
  }

  lines() {
    return this.text().split('\n');
  }

  line(row) {
    const lines = this.lines();
    if (row < 0 || row >= lines.length) return '';
    return lines[row];
  }

  // rowCol returns the cursor line and its rune column.
  rowCol() {
    const cursor = clampCursor(this.cursor, this.value.length);
    let row = 0;
    let start = 0;
    for (let i = 0; i < cursor; i++) {
      if (this.value[i] === '\n') {
        row++;
        start = i + 1;
      }
    }
    return [row, cursor - start];
  }

  setValue(s) {
    this.value = filterRunes(Array.from(s.replace(/\r\n/g, '\n')), true);
    if (this.maxLen > 0 && this.value.length > this.maxLen) this.value = this.value.slice(0, this.maxLen);
    this.cursor = this.value.length;
  }

  insertRune(r) {
    if (r === '\r') return false;
    if (this.maxLen > 0 && this.value.length >= this.maxLen) return false;
    insertRuneAt(this, r);
    return true;
  }

  insertString(s) {
    let runes = filterRunes(Array.from(s.replace(/\r\n/g, '\n')), true);
    if (runes.length === 0) return false;
    if (this.maxLen > 0) {
      const room = this.maxLen - this.value.length;
      if (room <= 0) return false;
      if (runes.length > room) runes = runes.slice(0, room);
    }
    insertRunesAt(this, runes);
    return true;
  }

  backspace() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor === 0) return false;
    this.value.splice(this.cursor - 1, 1);
    this.cursor--;
    return true;
  }

  delete() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor >= this.value.length) return false;
    this.value.splice(this.cursor, 1);
    return true;
  }

  deleteWordLeft() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const start = wordLeftIndex(this.value, this.cursor);
    if (start === this.cursor) return false;
    this.value.splice(start, this.cursor - start);
    this.cursor = start;
    return true;
  }

  left() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor === 0) return false;
    this.cursor--;
    return true;
  }

  right() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    if (this.cursor >= this.value.length) return false;
    this.cursor++;
    return true;
  }

  wordLeft() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const next = wordLeftIndex(this.value, this.cursor);
    if (next === this.cursor) return false;
    this.cursor = next;
    return true;
  }

  wordRight() {
    this.cursor = clampCursor(this.cursor, this.value.length);
    const next = wordRightIndex(this.value, this.cursor);
    if (next === this.cursor) return false;
    this.cursor = next;
    return true;
  }

  up() {
    const [row, col] = this.rowCol();
    if (row === 0) return false;
    const lines = this.lineBounds();
    const [start, end] = lines[row - 1];
    this.cursor = clampCursor(start + col, end);
    return true;
  }

  down() {
    const [row, col] = this.rowCol();
    const lines = this.lineBounds();
    if (row >= lines.length - 1) return false;
    const [start, end] = lines[row + 1];
    this.cursor = clampCursor(start + col, end);
    return true;
  }

  home() {
    const [row] = this.rowCol();
    const start = this.lineBounds()[row][0];
    const moved = this.cursor !== start;
    this.cursor = start;
    return moved;
  }

  end() {
    const [row] = this.rowCol();
    const end = this.lineBounds()[row][1];
    const moved = this.cursor !== end;
    this.cursor = end;
    return moved;
  }

  pageUp() {
    let moved = false;
    for (let i = 0; i < this.pageHeight(); i++) {
      if (!this.up()) break;
      moved = true;
    }
    return moved;
  }

  pageDown() {
    let moved = false;
    for (let i = 0; i < this.pageHeight(); i++) {
      if (!this.down()) break;
      moved = true;
    }
    return moved;
  }

  handleKey(ev) {
    if (ev == null) return false;
    switch (ev.key) {
      case 'left': this.left(); return true;
      case 'right': this.right(); return true;
      case 'up': this.up(); return true;
      case 'down': this.down(); return true;
      case 'home': case 'ctrl-a': this.home(); return true;
      case 'end': case 'ctrl-e': this.end(); return true;
      case 'page-up': this.pageUp(); return true;
      case 'page-down': this.pageDown(); return true;
      case 'backspace': this.backspace(); return true;
      case 'delete': this.delete(); return true;
      case 'ctrl-left': case 'alt-left': case 'ctrl-b': this.wordLeft(); return true;
      case 'ctrl-right': case 'alt-right': case 'ctrl-f': this.wordRight(); return true;
      case 'ctrl-w': case 'alt-backspace': this.deleteWordLeft(); return true;
      case 'enter': case 'ctrl-j': case 'ctrl-m': this.insertRune('\n'); return true;
      default: break;
    }
    const r = printableRune(ev);
    if (r != null) {
      this.insertRune(r);
      return true;
    }
    return false;
  }

  displayRunes() {
    if (this.mask === '') return this.value;
    return this.value.map((r) => (r === '\n' ? r : this.mask));
  }

  // lineBounds returns [start, end) rune bounds per line, end excluding "\n".
  lineBounds() {
    const bounds = [[0, 0]];
    for (let i = 0; i < this.value.length; i++) {
      if (this.value[i] === '\n') {
        bounds[bounds.length - 1][1] = i;
        bounds.push([i + 1, i + 1]);
      }
    }
    bounds[bounds.length - 1][1] = this.value.length;
    return bounds;
  }

  visibleRowOffset(lines) {
    const height = this.visibleHeight(lines);
    let offset = this.rowOffset;
    const [cursorRow] = this.rowCol();
    if (height <= 0) return 0;
    if (cursorRow < offset) offset = cursorRow;
    if (cursorRow >= offset + height) offset = cursorRow - height + 1;
    const max = lines - height;
    if (offset > max) offset = max;
    if (offset < 0) offset = 0;
    return offset;
  }

  visibleHeight(lines) {
    let height = this.height;
    if (height <= 0) height = lines;
    if (height < 0) return 0;
    if (height > lines) height = lines;
    return height;
  }

  pageHeight() {
    return this.height > 1 ? this.height : 1;
  }

  // colWindow returns the horizontal offset and the cursor display column.
  colWindow(lines, cursorRow, cursorCol) {
    const line = lineRunes(lines, cursorRow);
    if (this.width <= 0) {
      return [0, cellWidthOf(line.slice(0, clampCursor(cursorCol, line.length)))];
    }
    let offset = this.colOffset;
    if (offset < 0) offset = 0;
    if (cursorCol < offset) offset = cursorCol;
    if (offset > line.length) offset = line.length;
    while (offset < cursorCol && cellWidthOf(line.slice(offset, cursorCol)) + cursorCellWidth(line, cursorCol) > this.width) {
      offset++;
    }
    if (offset > cursorCol) offset = cursorCol;
    const col = cellWidthOf(line.slice(offset, clampCursor(cursorCol, line.length)));
    return [offset, col];
  }

  lineBox(line, cursorLine, cursorCol, offset) {
    const node = row().height(1);
    if (this.width > 0) node.width(this.width);
    const start = clampCursor(offset, line.length);
    let visible = line.slice(start);
    if (this.width > 0) visible = truncateRunesToCells(visible, this.width);
    if (!cursorLine || !this.focused) {
      let value = visible.join('');
      if (this.width > 0) value = padTo(value, this.width);
      if (value !== '') node.child(text(value).style(this.style).height(1));
      return node;
    }
    const rel = clampCursor(cursorCol - start, visible.length);
    const pre = visible.slice(0, rel).join('');
    let post = visible.slice(rel);
    let cursorText = ' ';
    if (post.length > 0) {
      cursorText = post[0];
      post = post.slice(1);
    }
    let postText = post.join('');
    if (this.width > 0) {
      postText = truncate(postText, this.width - displayWidth(pre) - displayWidth(cursorText));
    }
    if (pre !== '') node.child(text(pre).style(this.style).height(1));
    node.child(text(cursorText).style(firstNonEmpty(this.cursorStyle, DEFAULT_CURSOR_STYLE)).height(1));
    if (postText !== '') node.child(text(postText).style(this.style).height(1));
    const used = displayWidth(pre) + displayWidth(cursorText) + displayWidth(postText);
    if (this.width > used) node.child(text(' '.repeat(this.width - used)));
    return node;
  }

  inputs() {
    if (this.input.length > 0) return this.input;
    return ['key', 'paste'];
  }

  build() {
    const col = box('col');
    if (this.id !== '') col.id(this.id);
    col.focused(this.focused).input(...this.inputs());
    if (this.width > 0) col.width(this.width);
    if (this.height > 0) col.height(this.height);
    if (this.value.length === 0 && this.placeholder !== '') {
      let value = this.placeholder;
      if (this.width > 0) value = padTo(value, this.width);
      col.child(text(value).style(this.placeholderStyle).height(1));
      if (this.focused) col.cursor(0, 0, cursorShape(this.cursorShape));
      return col;
    }
    let lines = this.displayRunes().join('').split('\n');
    if (lines.length === 0) lines = [''];
    const [cursorRow, cursorCol] = this.rowCol();
    const offset = this.visibleRowOffset(lines.length);
    const [colOffset, cursorDisplayCol] = this.colWindow(lines, cursorRow, cursorCol);
    const height = this.visibleHeight(lines.length - offset);
    for (let i = 0; i < height; i++) {
      const rowIndex = offset + i;
      col.child(this.lineBox(Array.from(lines[rowIndex]), rowIndex === cursorRow, cursorCol, colOffset));
    }
    if (this.focused) {
      let rowIndex = cursorRow - offset;
      if (rowIndex < 0) rowIndex = 0;
      col.cursor(rowIndex, cursorDisplayCol, cursorShape(this.cursorShape));
    }
    return col;
  }
}

function lineRunes(lines, row) {
  if (row < 0 || row >= lines.length) return [];
  return Array.from(lines[row]);
}

module.exports = {
  DEFAULT_CURSOR_STYLE,
  DEFAULT_CURSOR_SHAPE,
  TextInput,
  TextArea,
};
