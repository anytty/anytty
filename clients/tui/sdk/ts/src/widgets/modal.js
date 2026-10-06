'use strict';

// Program-side dialogs and keyboard menus. Modal/Menu build positioned
// FloatingLayers; nothing is emitted by the widget. Port of
// tui2/sdk/widgets/modal.go.

const { box } = require('../builder');
const { FloatingLayer } = require('./layout');
const { DEFAULT_LIST_MARKER } = require('./list');

const DEFAULT_BACKDROP_STYLE = 'dim';

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

function clampIndex(i, count) {
  if (i < 0 || count <= 0) return 0;
  if (i >= count) return count - 1;
  return i;
}

function absInt(value) {
  return value < 0 ? -value : value;
}

// menuItemValue returns the item identity emitted by Menu.value.
function menuItemValue(item) {
  if (item.id) return item.id;
  return item.label || '';
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

class Modal {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.rows = options.rows ? options.rows.slice() : [];
    this.style = options.style || '';
    this.backdrop = Boolean(options.backdrop);
    this.backdropStyle = options.backdropStyle || '';
    this.backdropId = options.backdropId || '';
    this.center = Boolean(options.center);
    this.parentWidth = options.parentWidth || 0;
    this.parentHeight = options.parentHeight || 0;
  }

  // position returns the floating layer origin: (X, Y), or the centered origin
  // when Center is set and the parent size is known. Never negative.
  position() {
    let x = this.x;
    let y = this.y;
    if (this.center && this.parentWidth > 0 && this.parentHeight > 0) {
      x = Math.trunc((this.parentWidth - this.width) / 2);
      y = Math.trunc((this.parentHeight - this.height) / 2);
    }
    if (x < 0) x = 0;
    if (y < 0) y = 0;
    return [x, y];
  }

  // build returns a stack: the optional backdrop stretches to fill the parent,
  // the framed layer is positioned on top.
  build() {
    const [x, y] = this.position();
    const stack = box('stack');
    if (this.parentWidth > 0) stack.width(this.parentWidth);
    if (this.parentHeight > 0) stack.height(this.parentHeight);
    if (this.backdrop) {
      const backdrop = box().style(firstNonEmpty(this.backdropStyle, DEFAULT_BACKDROP_STYLE));
      let backdropId = this.backdropId;
      if (backdropId === '' && this.id !== '') backdropId = this.id + ':backdrop';
      if (backdropId !== '') backdrop.id(backdropId);
      if (this.parentWidth > 0) backdrop.width(this.parentWidth);
      if (this.parentHeight > 0) backdrop.height(this.parentHeight);
      stack.child(backdrop);
    }
    const layer = new FloatingLayer({
      id: this.id,
      title: this.title,
      x,
      y,
      width: this.width,
      height: this.height,
      style: this.style,
      rows: this.rows,
    });
    stack.child(layer.build());
    return stack;
  }
}

class Menu {
  constructor(options = {}) {
    this.id = options.id || '';
    this.title = options.title || '';
    this.items = options.items ? options.items.slice() : [];
    this.selected = options.selected || 0;
    this.width = options.width || 0;
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.style = options.style || '';
    this.selectedStyle = options.selectedStyle || '';
    this.disabledStyle = options.disabledStyle || '';
    this.separatorStyle = options.separatorStyle || '';
    this.frameStyle = options.frameStyle || '';
    this.marker = options.marker || '';
  }

  // move shifts selected by delta selectable items, skipping separators and
  // disabled items; it stops at the boundaries.
  move(delta) {
    if (delta === 0 || this.items.length === 0) return;
    this.selected = clampIndex(this.selected, this.items.length);
    const step = delta < 0 ? -1 : 1;
    for (let n = 0; n < absInt(delta); n++) {
      let next = this.selected + step;
      while (next >= 0 && next < this.items.length && !this.selectable(next)) next += step;
      if (next < 0 || next >= this.items.length) break;
      this.selected = next;
    }
  }

  // select moves the selection to index when it is selectable.
  select(index) {
    if (!this.selectable(index)) return false;
    this.selected = index;
    return true;
  }

  // hotkey matches one key event against the item hotkeys (case-insensitive),
  // selects the item and returns its value. A non-matching event returns
  // { value: '', ok: false }.
  hotkey(ev) {
    const r = printableRune(ev);
    if (r == null) return { value: '', ok: false };
    for (let i = 0; i < this.items.length; i++) {
      const item = this.items[i];
      if (item.separator || item.disabled || !item.hotkey) continue;
      const hot = Array.from(item.hotkey);
      if (hot.length === 1 && hot[0].toLowerCase() === r.toLowerCase()) {
        this.selected = i;
        return { value: menuItemValue(item), ok: true };
      }
    }
    return { value: '', ok: false };
  }

  // value returns the selected item value, empty when nothing selectable is
  // selected.
  value() {
    const { item, ok } = this.selectedItem();
    if (!ok) return '';
    return menuItemValue(item);
  }

  // selectedItem returns the selected item and whether it is selectable.
  selectedItem() {
    if (this.selected < 0 || this.selected >= this.items.length) {
      return { item: null, ok: false };
    }
    const item = this.items[this.selected];
    if (item.separator || item.disabled) return { item, ok: false };
    return { item, ok: true };
  }

  selectable(index) {
    if (index < 0 || index >= this.items.length) return false;
    const item = this.items[index];
    return !item.separator && !item.disabled;
  }

  // menuRows builds the FrameRow list Build places in the floating layer.
  menuRows() {
    const marker = firstNonEmpty(this.marker, DEFAULT_LIST_MARKER);
    const rows = [];
    for (let i = 0; i < this.items.length; i++) {
      const item = this.items[i];
      if (item.separator) {
        rows.push({
          text: '─'.repeat(3),
          style: firstNonEmpty(this.separatorStyle, this.disabledStyle),
          id: '',
          input: [],
        });
        continue;
      }
      let text = '  ' + (item.label || '');
      let style = firstNonEmpty(item.style, this.style);
      if (i === this.selected && this.selectable(i)) {
        text = marker + (item.label || '');
        style = firstNonEmpty(this.selectedStyle, style);
      }
      if (item.disabled) style = firstNonEmpty(this.disabledStyle, style);
      const row = { text, style, id: '', input: [] };
      if (item.id) {
        row.id = item.id;
        row.input = ['mouse'];
      }
      rows.push(row);
    }
    return rows;
  }

  // build returns the menu as a positioned FloatingLayer.
  build() {
    const layer = new FloatingLayer({
      id: this.id,
      title: this.title,
      x: this.x,
      y: this.y,
      width: this.width,
      style: this.frameStyle,
      rows: this.menuRows(),
    });
    return layer.build();
  }
}

module.exports = {
  DEFAULT_BACKDROP_STYLE,
  Modal,
  Menu,
  menuItemValue,
};
