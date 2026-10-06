'use strict';

// Pure mouse helpers: hit-test regions, wheel folding, drag selection and
// click classification. No timers, threads or I/O. Port of
// tui2/sdk/widgets/mouse.go.

const DEFAULT_ROW_HEIGHT = 1;
const DEFAULT_CLICK_THRESHOLD = 400;

function clampOffset(offset, size, count) {
  if (size <= 0) return 0;
  if (offset < 0) return 0;
  const max = count - size;
  if (offset > max) return max;
  return offset;
}

// HitRegion is one rectangular hit-test target: id, viewport rect and opaque
// payload. A W or H <= 0 means the region is unbounded on that axis.
class HitRegion {
  constructor(options = {}) {
    this.id = options.id || '';
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.w = options.w || 0;
    this.h = options.h || 0;
    this.data = options.data !== undefined ? options.data : null;
  }

  contains(x, y) {
    if (this.w > 0 && (x < this.x || x >= this.x + this.w)) return false;
    if (this.h > 0 && (y < this.y || y >= this.y + this.h)) return false;
    return true;
  }

  rect() {
    return { x: this.x, y: this.y, w: this.w, h: this.h };
  }
}

// hit returns the topmost region containing (x, y), or null. Later regions
// win, so callers order regions bottom-to-top.
function hit(regions, x, y) {
  for (let i = regions.length - 1; i >= 0; i--) {
    if (regions[i].contains(x, y)) return regions[i];
  }
  return null;
}

// hitId is hit's convenience form.
function hitId(regions, x, y) {
  const region = hit(regions, x, y);
  if (region == null) return { id: '', ok: false };
  return { id: region.id, ok: true };
}

// listRegions builds one region per visible List row. Data is the absolute row
// index.
function listRegions(list, originX, originY, width) {
  const [start, end] = list.visibleRange();
  if (end <= start) return [];
  let top = originY;
  if (list.header !== '') top++;
  const regions = [];
  for (let row = start; row < end; row++) {
    let id = '';
    if (list.rowId != null) id = list.rowId(row) || '';
    regions.push(new HitRegion({ id, x: originX, y: top + (row - start), w: width, h: DEFAULT_ROW_HEIGHT, data: row }));
  }
  return regions;
}

// virtualListRegions is listRegions for a VirtualList; the row's own id wins.
function virtualListRegions(list, originX, originY, width) {
  const [start, end] = list.visibleRange();
  if (end <= start) return [];
  let top = originY;
  if (list.header !== '') top++;
  const regions = [];
  for (let row = start; row < end; row++) {
    regions.push(new HitRegion({
      id: list.rows[row].id || '',
      x: originX,
      y: top + (row - start),
      w: width,
      h: DEFAULT_ROW_HEIGHT,
      data: row,
    }));
  }
  return regions;
}

// tableRegions builds one region per visible Table cell. Data is a
// [row, col] pair; row -1 is the header and -2 the footer.
function tableRegions(table, originX, originY) {
  if (table.columns.length === 0) return [];
  const widths = table.columnWidths();
  const sep = table.sepWidth();
  let top = originY;
  if (!table.hideHeader) {
    if (table.rule) top += 2;
    else top++;
  }
  const xs = new Array(widths.length).fill(0);
  let x = originX;
  for (let col = 0; col < widths.length; col++) {
    if (col > 0) x += sep;
    xs[col] = x;
    x += widths[col];
  }
  const regions = [];
  for (let row = 0; row < table.rowCount(); row++) {
    let cellId = '';
    if (table.rowId != null) cellId = table.rowId(row) || '';
    for (let col = 0; col < widths.length; col++) {
      regions.push(new HitRegion({
        x: xs[col],
        y: top + row,
        w: widths[col],
        h: DEFAULT_ROW_HEIGHT,
        id: cellId,
        data: [row, col],
      }));
    }
  }
  return regions;
}

// applyWheel folds one wheel delta into an offset and clamps it to
// [0, total-visible]. Positive delta scrolls down/right; a fully visible (or
// empty) content set pins the offset to 0.
function applyWheel(offset, total, visible, delta) {
  offset += delta;
  if (visible <= 0 || visible >= total) return 0;
  return clampOffset(offset, visible, total);
}

// tableScroll is the Table counterpart of applyWheel.
function tableScroll(offset, total, visible, delta) {
  return applyWheel(offset, total, visible, delta);
}

// Drag is a pure mouse drag selection model: begin on press, update on motion,
// end on release. Coordinates are absolute viewport cells.
class Drag {
  constructor(options = {}) {
    this.active = Boolean(options.active);
    this.startX = options.startX || 0;
    this.startY = options.startY || 0;
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.button = options.button || '';
    this.thresholdPx = options.thresholdPx || 0;
  }

  begin(x, y, button) {
    this.active = true;
    this.startX = x;
    this.startY = y;
    this.x = x;
    this.y = y;
    this.button = button;
  }

  update(x, y) {
    if (!this.active) return;
    this.x = x;
    this.y = y;
  }

  // end finishes the drag and returns the normalized selection rect, or null
  // when no drag was active.
  end() {
    if (!this.active) return null;
    this.active = false;
    return this.rect();
  }

  // rect returns the selection normalized to a non-negative origin: x/y are
  // the min corner, w/h the inclusive span (at least 1).
  rect() {
    let minX = this.startX;
    let maxX = this.x;
    if (minX > maxX) {
      const swap = minX;
      minX = maxX;
      maxX = swap;
    }
    let minY = this.startY;
    let maxY = this.y;
    if (minY > maxY) {
      const swap = minY;
      minY = maxY;
      maxY = swap;
    }
    return { x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1 };
  }
}

// selectRange normalizes two positions into an inclusive (lo, hi) pair.
function selectRange(a, b) {
  return a > b ? [b, a] : [a, b];
}

// listSelectRange maps two viewport y positions to the inclusive range of
// absolute List rows they span, clamped to the current window. It returns null
// when neither position lands on a row.
function listSelectRange(list, y0, y1) {
  const [lo, hi] = selectRange(y0, y1);
  let first = list.rowAt(lo);
  if (first == null) {
    first = list.rowAt(hi);
    if (first == null) return null;
  }
  let last = list.rowAt(hi);
  if (last == null) last = first;
  if (first > last) {
    const swap = first;
    first = last;
    last = swap;
  }
  return [first, last];
}

// virtualListSelectRange is listSelectRange for a VirtualList.
function virtualListSelectRange(list, y0, y1) {
  const [lo, hi] = selectRange(y0, y1);
  let first = list.rowAt(lo);
  if (first == null) {
    first = list.rowAt(hi);
    if (first == null) return null;
  }
  let last = list.rowAt(hi);
  if (last == null) last = first;
  if (first > last) {
    const swap = first;
    first = last;
    last = swap;
  }
  return [first, last];
}

// ClickTracker classifies clicks into single/double/triple clicks purely from
// timing (milliseconds) and position.
class ClickTracker {
  constructor(options = {}) {
    this.lastAt = options.lastAt !== undefined ? options.lastAt : null;
    this.lastX = options.lastX || 0;
    this.lastY = options.lastY || 0;
    this.threshold = options.threshold || 0;
    this.count = options.count || 0;
  }

  // click records a click at (x, y) happening at `at` and returns the click
  // count (1, 2 or 3).
  click(x, y, at) {
    let threshold = this.threshold;
    if (threshold <= 0) threshold = DEFAULT_CLICK_THRESHOLD;
    const sameCell = this.lastX === x && this.lastY === y;
    const close = this.lastAt != null && at - this.lastAt <= threshold;
    if (close && sameCell && this.count >= 1 && this.count < 3) {
      this.count++;
    } else {
      this.count = 1;
    }
    this.lastAt = at;
    this.lastX = x;
    this.lastY = y;
    return this.count;
  }
}

module.exports = {
  DEFAULT_ROW_HEIGHT,
  DEFAULT_CLICK_THRESHOLD,
  HitRegion,
  hit,
  hitId,
  listRegions,
  virtualListRegions,
  tableRegions,
  applyWheel,
  tableScroll,
  selectRange,
  listSelectRange,
  virtualListSelectRange,
  Drag,
  ClickTracker,
};
