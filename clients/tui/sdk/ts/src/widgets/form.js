'use strict';

// Column of labeled fields with keyboard focus, navigation and aggregate
// validation. Port of tui2/sdk/widgets/form.go.

const { box, text } = require('../builder');

const FORM_REQUIRED_MESSAGE = 'required';

function clampIndex(i, count) {
  if (i < 0 || count <= 0) return 0;
  if (i >= count) return count - 1;
  return i;
}

// cloneInput copies a TextInput so Build can override Focused/Width without
// mutating the caller-owned input (Go copies the struct value).
function cloneInput(input) {
  const copy = Object.assign(Object.create(Object.getPrototypeOf(input)), input);
  if (Array.isArray(input.value)) copy.value = input.value.slice();
  if (Array.isArray(input.input)) copy.input = input.input.slice();
  return copy;
}

class Field {
  constructor(options = {}) {
    this.id = options.id || '';
    this.label = options.label || '';
    this.help = options.help || '';
    this.error = options.error || '';
    this.input = options.input || null;
    this.required = Boolean(options.required);
    this.validate = options.validate || null;
    this.style = options.style || '';
    this.labelStyle = options.labelStyle || '';
    this.errorStyle = options.errorStyle || '';
    this.helpStyle = options.helpStyle || '';
    this.focused = Boolean(options.focused);
    this.disabled = Boolean(options.disabled);
    this.readonly = Boolean(options.readonly);
  }

  // selectable reports whether Form navigation may land on the field.
  selectable() {
    return !this.disabled && !this.readonly;
  }

  // text returns the field value, or "" when it owns no input.
  text() {
    if (this.input == null) return '';
    return this.input.text();
  }

  // build returns the field as a column: label, input, then the error line
  // when set, otherwise the help line.
  build() {
    return formFieldBox(this, 0);
  }
}

function formFieldBox(field, width) {
  const col = box('col');
  if (width > 0) col.width(width);
  if (field.style !== '') col.style(field.style);
  if (field.label !== '') {
    let label = field.label;
    if (field.required) label += ' *';
    col.child(text(label).style(field.labelStyle).height(1));
  }
  if (field.input != null) {
    const input = cloneInput(field.input);
    input.focused = field.focused;
    if (input.width <= 0 && width > 0) input.width = width;
    col.child(input.build());
  }
  if (field.error !== '') {
    col.child(text(field.error).style(field.errorStyle).height(1));
  } else if (field.help !== '') {
    col.child(text(field.help).style(field.helpStyle).height(1));
  }
  return col;
}

class Form {
  constructor(options = {}) {
    this.id = options.id || '';
    this.fields = options.fields ? options.fields.slice() : [];
    this.focus = options.focus || 0;
    this.width = options.width || 0;
    this.style = options.style || '';
    this.errorStyle = options.errorStyle || '';
    this.validateOnChange = Boolean(options.validateOnChange);
  }

  // next moves focus to the next selectable field, wrapping around.
  next() {
    this.moveFocus(1);
  }

  // prev moves focus to the previous selectable field, wrapping around.
  prev() {
    this.moveFocus(-1);
  }

  moveFocus(step) {
    const count = this.fields.length;
    if (count === 0) {
      this.focus = 0;
      return;
    }
    this.focus = clampIndex(this.focus, count);
    for (let i = 1; i <= count; i++) {
      let next = (this.focus + step * i) % count;
      if (next < 0) next += count;
      if (this.fields[next].selectable()) {
        this.focus = next;
        return;
      }
    }
  }

  // focusId returns the id of the focused field, or "" when the form has none.
  focusId() {
    if (this.focus < 0 || this.focus >= this.fields.length) return '';
    return this.fields[this.focus].id;
  }

  // setValues copies values keyed by field id into the matching inputs.
  setValues(values) {
    for (const field of this.fields) {
      if (field.input == null || field.id === '') continue;
      if (Object.prototype.hasOwnProperty.call(values, field.id)) {
        field.input.setValue(values[field.id]);
      }
    }
  }

  // values returns the current text of every identified field.
  values() {
    const out = {};
    for (const field of this.fields) {
      if (field.id === '' || field.input == null) continue;
      out[field.id] = field.input.text();
    }
    return out;
  }

  // input returns the input owned by the field with the given id, or null.
  input(id) {
    for (const field of this.fields) {
      if (field.id === id) return field.input;
    }
    return null;
  }

  // validate runs every field's Required check and Validate, stores each
  // result in field.error and reports whether the whole form is valid.
  validate() {
    let valid = true;
    for (let i = 0; i < this.fields.length; i++) {
      const msg = this.fieldError(i);
      this.fields[i].error = msg;
      if (msg !== '') valid = false;
    }
    return valid;
  }

  fieldError(index) {
    if (index < 0 || index >= this.fields.length) return '';
    const field = this.fields[index];
    const value = field.text();
    if (field.required && value.trim() === '') return FORM_REQUIRED_MESSAGE;
    if (field.validate != null) return field.validate(value);
    return '';
  }

  // handleKey routes one key to the focused field. Enter/Tab advance focus and
  // Shift-Tab retreats; every other key goes to the focused TextInput.
  handleKey(id, ev) {
    if (ev == null) return false;
    if (id !== '' && id !== this.focusId()) return false;
    if (ev.key === 'enter' || ev.key === 'tab') {
      this.next();
      return true;
    }
    if (ev.key === 'shift-tab') {
      this.prev();
      return true;
    }
    if (this.focus < 0 || this.focus >= this.fields.length || this.fields[this.focus].input == null) {
      return false;
    }
    if (!this.fields[this.focus].input.handleKey(ev)) return false;
    if (this.validateOnChange) {
      this.fields[this.focus].error = this.fieldError(this.focus);
    }
    return true;
  }

  // build returns the form as a column of fields, marking the focused one.
  build() {
    const col = box('col');
    if (this.id !== '') col.id(this.id);
    if (this.width > 0) col.width(this.width);
    if (this.style !== '') col.style(this.style);
    for (let i = 0; i < this.fields.length; i++) {
      const field = new Field(this.fields[i]);
      field.focused = i === this.focus;
      if (field.errorStyle === '') field.errorStyle = this.errorStyle;
      col.child(formFieldBox(field, this.width));
    }
    return col;
  }
}

module.exports = {
  FORM_REQUIRED_MESSAGE,
  Field,
  Form,
};
