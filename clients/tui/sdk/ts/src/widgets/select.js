'use strict';

// Dropdown state machine plus a pure Build. Port of tui2/sdk/widgets/select.go.

const { box, text, displayWidth, truncate } = require('../builder');
const { DEFAULT_LIST_MARKER } = require('./list');

const DEFAULT_SELECT_INDICATOR = ' ▾';

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

// listRowBox mirrors list.go's row builder (static id, no interactions).
function listRowBox(value, style, id, width) {
  if (width > 0) value = padTo(value, width);
  const node = text(value).height(1);
  if (width > 0) node.width(width);
  if (style !== '') node.style(style);
  if (id !== '') node.id(id).input('mouse');
  return node;
}

function absInt(value) {
  return value < 0 ? -value : value;
}

// optionDisplay returns the option label, falling back to its value.
function optionDisplay(option) {
  if (option.label) return option.label;
  return option.value || '';
}

function toLowerASCIIByte(b) {
  if (b >= 0x41 && b <= 0x5a) return b + 0x20;
  return b;
}

function toLowerASCII(s) {
  let out = '';
  for (let i = 0; i < s.length; i++) out += String.fromCharCode(toLowerASCIIByte(s.charCodeAt(i)));
  return out;
}

function hasPrefixFold(value, lowerPrefix) {
  if (value.length < lowerPrefix.length) return false;
  for (let i = 0; i < lowerPrefix.length; i++) {
    if (toLowerASCIIByte(value.charCodeAt(i)) !== lowerPrefix.charCodeAt(i)) return false;
  }
  return true;
}

class Select {
  constructor(options = {}) {
    this.id = options.id || '';
    this.label = options.label || '';
    this.options = (options.options || []).map((option) => ({
      value: option.value || '',
      label: option.label || '',
      disabled: Boolean(option.disabled),
      style: option.style || '',
    }));
    this.value = options.value || '';
    this.open = Boolean(options.open);
    this.width = options.width || 0;
    this.placeholder = options.placeholder || '';
    this.style = options.style || '';
    this.labelStyle = options.labelStyle || '';
    this.placeholderStyle = options.placeholderStyle || '';
    this.selectedStyle = options.selectedStyle || '';
    this.disabledStyle = options.disabledStyle || '';
    this.marker = options.marker || '';
    this.indicator = options.indicator || '';
    this.dropdownStyle = options.dropdownStyle || '';
  }

  // index returns the position of the selected option, or -1.
  index() {
    for (let i = 0; i < this.options.length; i++) {
      if (this.options[i].value === this.value) return i;
    }
    return -1;
  }

  // selected returns the selected option; a disabled selection reports
  // ok=false.
  selected() {
    const i = this.index();
    if (i < 0 || this.options[i].disabled) return { option: null, ok: false };
    return { option: this.options[i], ok: true };
  }

  // selectIndex selects the option at index when it is in range and enabled.
  selectIndex(index) {
    if (index < 0 || index >= this.options.length || this.options[index].disabled) return false;
    this.value = this.options[index].value;
    return true;
  }

  // move shifts the selection by delta enabled options, skipping disabled
  // ones, and stops at the boundaries.
  move(delta) {
    if (delta === 0 || this.options.length === 0) return;
    let current = this.index();
    const step = delta < 0 ? -1 : 1;
    if (current < 0) {
      current = -1;
      if (step < 0) current = this.options.length;
    }
    for (let n = 0; n < absInt(delta); n++) {
      let next = current + step;
      while (next >= 0 && next < this.options.length && this.options[next].disabled) next += step;
      if (next < 0 || next >= this.options.length) break;
      current = next;
    }
    if (current >= 0 && current < this.options.length) {
      this.value = this.options[current].value;
    }
  }

  // typeahead selects the next enabled option whose Label or Value starts with
  // prefix (case-insensitive ASCII), wrapping around.
  typeahead(prefix) {
    if (prefix === '' || this.options.length === 0) return false;
    const needle = toLowerASCII(prefix);
    const count = this.options.length;
    const start = this.index();
    for (let i = 1; i <= count; i++) {
      let index = start < 0 ? i - 1 : start + i;
      index %= count;
      if (index < 0) index += count;
      const option = this.options[index];
      if (option.disabled) continue;
      if (hasPrefixFold(optionDisplay(option), needle) || hasPrefixFold(option.value, needle)) {
        this.value = option.value;
        return true;
      }
    }
    return false;
  }

  container() {
    const col = box('col');
    if (this.id !== '') col.id(this.id);
    if (this.width > 0) col.width(this.width);
    return col;
  }

  // build returns the dropdown when open, otherwise the closed value row.
  build() {
    if (this.open) return this.buildDropdown();
    const col = this.container();
    if (this.label !== '') col.child(text(this.label).style(this.labelStyle).height(1));
    const row = box('row').height(1);
    if (this.width > 0) row.width(this.width);
    const { option, ok } = this.selected();
    let value = this.placeholder;
    let style = this.placeholderStyle;
    if (ok) {
      value = optionDisplay(option);
      style = firstNonEmpty(option.style, this.selectedStyle, this.style);
    }
    if (value === '') value = ' ';
    const indicator = firstNonEmpty(this.indicator, DEFAULT_SELECT_INDICATOR);
    const total = this.width;
    if (total > 0) value = truncate(value, Math.max(0, total - displayWidth(indicator)));
    row.child(text(value).style(style).height(1));
    row.child(text(indicator).style(this.placeholderStyle).height(1));
    if (total > 0) {
      const pad = total - displayWidth(value) - displayWidth(indicator);
      if (pad > 0) row.child(text(' '.repeat(pad)));
    }
    col.child(row);
    return col;
  }

  // buildDropdown returns the open list of options as a column of one-row
  // boxes, ready for a FloatingLayer or Modal.
  buildDropdown() {
    const col = this.container();
    if (this.dropdownStyle !== '') col.style(this.dropdownStyle);
    const selected = this.index();
    const marker = firstNonEmpty(this.marker, DEFAULT_LIST_MARKER);
    for (let i = 0; i < this.options.length; i++) {
      const option = this.options[i];
      let value = '  ' + optionDisplay(option);
      let style = firstNonEmpty(option.style, this.style);
      if (option.disabled) style = firstNonEmpty(this.disabledStyle, style);
      if (i === selected && !option.disabled) {
        value = marker + optionDisplay(option);
        style = firstNonEmpty(this.selectedStyle, style);
      }
      col.child(listRowBox(value, style, '', this.width));
    }
    return col;
  }
}

module.exports = {
  DEFAULT_SELECT_INDICATOR,
  Select,
  optionDisplay,
};
