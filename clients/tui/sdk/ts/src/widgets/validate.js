'use strict';

// Pure field validators. A validator maps a value to an error message ("" when
// acceptable), so validators compose freely and can be driven by a Form or
// called directly. Port of tui2/sdk/widgets/validate.go.

// required rejects a value that is empty after trimming whitespace.
function required(msg) {
  return (value) => (value.trim() === '' ? msg : '');
}

// minLen rejects a value shorter than n runes; empty values pass.
function minLen(n, msg) {
  return (value) => {
    if (value === '' || n <= 0) return '';
    return Array.from(value).length < n ? msg : '';
  };
}

// maxLen rejects a value longer than n runes; empty values pass.
function maxLen(n, msg) {
  return (value) => {
    if (value === '' || n <= 0) return '';
    return Array.from(value).length > n ? msg : '';
  };
}

// pattern rejects a value that does not match the regular expression re. A
// pattern that fails to compile fails closed: every non-empty value is
// rejected with msg. Empty values pass.
function pattern(re, msg) {
  let compiled;
  try {
    compiled = new RegExp(re);
  } catch (err) {
    return (value) => (value === '' ? '' : msg);
  }
  return (value) => {
    if (value === '') return '';
    return compiled.test(value) ? '' : msg;
  };
}

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

// email rejects values that are not a plausible email address.
function email(msg) {
  return (value) => {
    if (value === '') return '';
    return EMAIL_PATTERN.test(value) ? '' : msg;
  };
}

// intRange rejects values that are not base-10 integers inside [min, max].
function intRange(min, max, msg) {
  return (value) => {
    const text = value.trim();
    if (text === '') return '';
    if (!/^[+-]?\d+$/.test(text)) return msg;
    const n = Number(text);
    if (!Number.isFinite(n) || n < min || n > max) return msg;
    return '';
  };
}

// oneOf rejects values absent from values.
function oneOf(values, msg) {
  return (value) => {
    if (value === '') return '';
    return values.includes(value) ? '' : msg;
  };
}

// all runs validators in order and returns the first non-empty message.
function all(...validators) {
  return (value) => {
    for (const validate of validators) {
      if (validate == null) continue;
      const msg = validate(value);
      if (msg !== '') return msg;
    }
    return '';
  };
}

// custom adapts an arbitrary function into a validator; null is a no-op.
function custom(fn) {
  if (fn == null) return () => '';
  return (value) => fn(value);
}

module.exports = {
  required,
  minLen,
  maxLen,
  pattern,
  email,
  intRange,
  oneOf,
  all,
  custom,
};
