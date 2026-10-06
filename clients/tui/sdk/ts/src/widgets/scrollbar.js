'use strict';

// Pure proportional scroll indicator, vertical or horizontal. Port of
// tui2/sdk/widgets/scrollbar.go.

const { box, text } = require('../builder');

const DEFAULT_SCROLLBAR_TRACK_STYLE = 'muted';
const DEFAULT_SCROLLBAR_THUMB_STYLE = 'accent';
const DEFAULT_SCROLLBAR_UP = '↑';
const DEFAULT_SCROLLBAR_DOWN = '↓';
const SCROLLBAR_TRACK_VERTICAL = '│';
const SCROLLBAR_THUMB_VERTICAL = '┃';
const SCROLLBAR_TRACK_HORIZONTAL = '─';
const SCROLLBAR_THUMB_HORIZONTAL = '━';

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

class Scrollbar {
  constructor(options = {}) {
    this.id = options.id || '';
    this.total = options.total || 0;
    this.visible = options.visible || 0;
    this.offset = options.offset || 0;
    this.height = options.height || 0;
    this.width = options.width || 0;
    this.vertical = Boolean(options.vertical);
    this.trackStyle = options.trackStyle || '';
    this.thumbStyle = options.thumbStyle || '';
    this.up = options.up || '';
    this.down = options.down || '';
  }

  // isVertical reports the resolved orientation: an explicit Vertical wins,
  // otherwise a Height means vertical and a Width means horizontal.
  isVertical() {
    if (this.vertical) return true;
    if (this.width > 0 && this.height <= 0) return false;
    return true;
  }

  // trackLength returns the number of cells of the track.
  trackLength() {
    if (this.isVertical()) return this.height > 0 ? this.height : 0;
    return this.width > 0 ? this.width : 0;
  }

  // thumbMaxOffset returns the largest Offset that still shows content.
  thumbMaxOffset() {
    const max = this.total - this.visible;
    return max < 0 ? 0 : max;
  }

  // clampOffset folds Offset into [0, Total-Visible] (never negative).
  clampOffset() {
    let offset = this.offset;
    if (offset < 0) offset = 0;
    const max = this.thumbMaxOffset();
    if (offset > max) offset = max;
    return offset;
  }

  // thumb returns the thumb start cell and size along the track.
  thumb() {
    const length = this.trackLength();
    if (length <= 0) return [0, 0];
    if (this.total <= 0 || this.total <= this.visible) return [0, length];
    let size = Math.trunc(this.visible * length / this.total);
    if (size < 1) size = 1;
    if (size > length) size = length;
    let start = 0;
    const max = this.thumbMaxOffset();
    if (max > 0) start = Math.trunc(this.clampOffset() * (length - size) / max);
    if (start < 0) start = 0;
    const maxStart = length - size;
    if (start > maxStart) start = maxStart;
    return [start, size];
  }

  // offsetAt maps a click position (cells from the start of the track,
  // 0-based) to an offset, centering the thumb on the click.
  offsetAt(pos) {
    const length = this.trackLength();
    if (length <= 0) return 0;
    const max = this.thumbMaxOffset();
    if (max <= 0) return 0;
    const [, size] = this.thumb();
    const denom = length - size;
    if (denom <= 0) return 0;
    let offset = Math.trunc((pos - Math.trunc(size / 2)) * max / denom);
    if (offset < 0) return 0;
    if (offset > max) return max;
    return offset;
  }

  capCount() {
    let count = 0;
    if (this.up !== '') count++;
    if (this.down !== '') count++;
    return count;
  }

  glyphs() {
    if (this.isVertical()) return [SCROLLBAR_TRACK_VERTICAL, SCROLLBAR_THUMB_VERTICAL];
    return [SCROLLBAR_TRACK_HORIZONTAL, SCROLLBAR_THUMB_HORIZONTAL];
  }

  // build returns the track with the proportional thumb. Up/Down are optional
  // one-cell end caps drawn outside the measured track.
  build() {
    const length = this.trackLength();
    const [start, size] = this.thumb();
    const trackStyle = firstNonEmpty(this.trackStyle, DEFAULT_SCROLLBAR_TRACK_STYLE);
    const thumbStyle = firstNonEmpty(this.thumbStyle, DEFAULT_SCROLLBAR_THUMB_STYLE);
    const [trackGlyph, thumbGlyph] = this.glyphs();
    const cell = (i) => (i >= start && i < start + size ? [thumbGlyph, thumbStyle] : [trackGlyph, trackStyle]);
    if (this.isVertical()) {
      const col = box('col').width(1);
      if (this.id !== '') col.id(this.id);
      col.input('mouse');
      col.height(length + this.capCount());
      if (this.up !== '') col.child(text(this.up).style(trackStyle).width(1).height(1));
      for (let i = 0; i < length; i++) {
        const [glyph, style] = cell(i);
        col.child(text(glyph).style(style).width(1).height(1));
      }
      if (this.down !== '') col.child(text(this.down).style(trackStyle).width(1).height(1));
      return col;
    }
    const row = box('row').height(1);
    if (this.id !== '') row.id(this.id);
    row.input('mouse');
    row.width(length + this.capCount());
    if (this.up !== '') row.child(text(this.up).style(trackStyle).width(1).height(1));
    for (let i = 0; i < length; i++) {
      const [glyph, style] = cell(i);
      row.child(text(glyph).style(style).width(1).height(1));
    }
    if (this.down !== '') row.child(text(this.down).style(trackStyle).width(1).height(1));
    return row;
  }
}

module.exports = {
  DEFAULT_SCROLLBAR_TRACK_STYLE,
  DEFAULT_SCROLLBAR_THUMB_STYLE,
  DEFAULT_SCROLLBAR_UP,
  DEFAULT_SCROLLBAR_DOWN,
  SCROLLBAR_TRACK_VERTICAL,
  SCROLLBAR_THUMB_VERTICAL,
  SCROLLBAR_TRACK_HORIZONTAL,
  SCROLLBAR_THUMB_HORIZONTAL,
  Scrollbar,
};
