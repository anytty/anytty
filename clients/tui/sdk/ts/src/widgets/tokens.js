'use strict';

const STYLE_DEFAULT = 'default';
const STYLE_BACKGROUND = 'bg';
const STYLE_FG = 'fg';
const STYLE_FOREGROUND = 'foreground';
const STYLE_STRONG_FOREGROUND = 'strong-foreground';
const STYLE_MUTED = 'muted';
const STYLE_ACCENT = 'accent';
const STYLE_ACCENT_DIM = 'accent_dim';
const STYLE_SUCCESS = 'success';
const STYLE_OK = 'ok';
const STYLE_WARNING = 'warning';
const STYLE_DANGER = 'danger';
const STYLE_INFO = 'info';
const STYLE_CHROME = 'chrome';
const STYLE_CHROME_FOCUS = 'chrome_focus';
const STYLE_HEADER = 'header';
const STYLE_TAB_ACTIVE = 'tab_active';
const STYLE_TAB_INACTIVE = 'tab_inactive';
const STYLE_FOOTER = 'footer';
const STYLE_FOOTER_ACCENT = 'footer-accent';
const STYLE_STATUS = 'status';
const STYLE_OVERLAY = 'overlay';
const STYLE_BORDER = 'border';
const STYLE_BORDER_FOCUS = 'border_focus';
const STYLE_BORDER_DEAD = 'border_dead';
const STYLE_ACTIVE_BORDER = 'active-border';
const STYLE_INACTIVE_BORDER = 'inactive-border';
const STYLE_SELECTION = 'selection';

function styleHasSegment(style, segment) {
  if (!style) return false;
  for (const part of style.split(';')) {
    if (part.trim() === segment) return true;
  }
  return false;
}

function withAttribute(style, attribute) {
  if (styleHasSegment(style, attribute)) return style;
  if (!style) return attribute;
  return style + ';' + attribute;
}

function withBold(style) { return withAttribute(style, 'bold'); }
function withUnderline(style) { return withAttribute(style, 'underline'); }
function withReverse(style) { return withAttribute(style, 'reverse'); }
function withItalic(style) { return withAttribute(style, 'italic'); }

function withFg(style, hex) { return setStyleColor(style, 'fg', hex); }
function withBg(style, hex) { return setStyleColor(style, 'bg', hex); }

function setStyleColor(style, key, hex) {
  if (!hex) return style;
  const segment = key + ':' + hex;
  const parts = [];
  let replaced = false;
  for (let part of style.split(';')) {
    part = part.trim();
    if (part === '') continue;
    if (part.startsWith(key + ':')) {
      if (replaced) continue;
      parts.push(segment);
      replaced = true;
      continue;
    }
    parts.push(part);
  }
  if (!replaced) parts.unshift(segment);
  return parts.join(';');
}

module.exports = {
  STYLE_DEFAULT,
  STYLE_BACKGROUND,
  STYLE_FG,
  STYLE_FOREGROUND,
  STYLE_STRONG_FOREGROUND,
  STYLE_MUTED,
  STYLE_ACCENT,
  STYLE_ACCENT_DIM,
  STYLE_SUCCESS,
  STYLE_OK,
  STYLE_WARNING,
  STYLE_DANGER,
  STYLE_INFO,
  STYLE_CHROME,
  STYLE_CHROME_FOCUS,
  STYLE_HEADER,
  STYLE_TAB_ACTIVE,
  STYLE_TAB_INACTIVE,
  STYLE_FOOTER,
  STYLE_FOOTER_ACCENT,
  STYLE_STATUS,
  STYLE_OVERLAY,
  STYLE_BORDER,
  STYLE_BORDER_FOCUS,
  STYLE_BORDER_DEAD,
  STYLE_ACTIVE_BORDER,
  STYLE_INACTIVE_BORDER,
  STYLE_SELECTION,
  styleHasSegment,
  withBold,
  withUnderline,
  withReverse,
  withItalic,
  withFg,
  withBg,
};
