'use strict';

// Pure text charts: Sparkline, BarChart, Heatmap, Meter (and its Gauge alias)
// and Legend. All geometry is cell-based and every glyph is display width 1.
// Port of tui2/sdk/widgets/chart.go.

const { box, row, text, displayWidth, truncate } = require('../builder');
const { formatFloat } = require('./format');

const SPARK_GLYPHS = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];
const CHART_DEFAULT_BAR_WIDTH = 20;
const DEFAULT_HEAT_SHADES = ' ·░▒▓█';
const DEFAULT_LEGEND_MARKER = '■';

// Progress glyphs Meter reuses (progress.go defaults).
const PROGRESS_FULL = '█';
const PROGRESS_EMPTY = '░';
const PROGRESS_WIDTH = 20;

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

function clampInt(v, lo, hi) {
  if (v < lo) return lo;
  if (v > hi) return hi;
  return v;
}

function clampFloat(v, lo, hi) {
  if (v < lo) return lo;
  if (v > hi) return hi;
  return v;
}

function finite(v) {
  return !Number.isNaN(v) && Number.isFinite(v);
}

class Sparkline {
  constructor(options = {}) {
    this.values = options.values ? options.values.slice() : [];
    this.width = options.width || 0;
    this.style = options.style || '';
    this.min = options.min !== undefined ? options.min : null;
    this.max = options.max !== undefined ? options.max : null;
  }

  // bounds returns the effective low/high values; an empty or all-NaN series
  // yields (0, 0).
  bounds() {
    let min = Infinity;
    let max = -Infinity;
    for (const v of this.values) {
      if (!finite(v)) continue;
      if (v < min) min = v;
      if (v > max) max = v;
    }
    if (min === Infinity) {
      min = 0;
      max = 0;
    }
    if (this.min != null) min = this.min;
    if (this.max != null) max = this.max;
    return [min, max];
  }

  // level returns the eighth-block index [0, 7] for v under the bounds.
  level(v) {
    const [min, max] = this.bounds();
    if (max <= min) return 3;
    if (Number.isNaN(v)) return 0;
    const t = clampFloat((v - min) / (max - min), 0, 1);
    const level = Math.trunc(t * (SPARK_GLYPHS.length - 1) + 0.5);
    return clampInt(level, 0, SPARK_GLYPHS.length - 1);
  }

  // samples returns the values to render: all of them, or Width buckets of
  // averaged samples when Width is smaller than the series.
  samples() {
    if (this.width <= 0 || this.values.length <= this.width) return this.values;
    const out = new Array(this.width).fill(0);
    const n = this.values.length;
    for (let i = 0; i < this.width; i++) {
      let start = Math.trunc(i * n / this.width);
      let end = Math.trunc((i + 1) * n / this.width);
      if (end <= start) end = start + 1;
      let sum = 0;
      for (let j = start; j < end; j++) sum += this.values[j];
      out[i] = sum / (end - start);
    }
    return out;
  }

  line() {
    const samples = this.samples();
    if (samples.length === 0) return '';
    let out = '';
    for (const v of samples) out += SPARK_GLYPHS[this.level(v)];
    return out;
  }

  build() {
    const node = text(this.line()).height(1);
    if (this.style !== '') node.style(this.style);
    return node;
  }
}

class BarChart {
  constructor(options = {}) {
    this.values = options.values ? options.values.slice() : [];
    this.labels = options.labels ? options.labels.slice() : [];
    this.width = options.width || 0;
    this.height = options.height || 0;
    this.max = options.max || 0;
    this.style = options.style || '';
    this.labelStyle = options.labelStyle || '';
    this.selectedStyle = options.selectedStyle || '';
    this.selected = options.selected !== undefined ? options.selected : 0;
    this.horizontal = Boolean(options.horizontal);
  }

  // maxValue returns Max when positive, else the largest finite value (at
  // least 1 when the series is empty or non-positive).
  maxValue() {
    if (this.max > 0) return this.max;
    let max = 0;
    for (const v of this.values) {
      if (!finite(v)) continue;
      if (v > max) max = v;
    }
    if (max <= 0) return 1;
    return max;
  }

  barHeight() {
    return this.height > 0 ? this.height : 1;
  }

  // verticalValues returns the values a vertical chart can fit in Width cells:
  // a width of w fits (w+1)/2 bars.
  verticalValues() {
    if (this.width <= 0) return this.values;
    let maxBars = Math.trunc((this.width + 1) / 2);
    if (maxBars > this.values.length) maxBars = this.values.length;
    if (maxBars < 0) maxBars = 0;
    return this.values.slice(0, maxBars);
  }

  horizontalValues() {
    if (this.height > 0 && this.values.length > this.height) {
      return this.values.slice(0, this.height);
    }
    return this.values;
  }

  columnWidth(count) {
    if (count <= 0) return 1;
    if (this.width <= 0) return 1;
    const width = Math.trunc((this.width - (count - 1)) / count);
    if (width < 1) return 1;
    return width;
  }

  labelWidth() {
    let width = 0;
    for (const label of this.labels) {
      const w = displayWidth(label);
      if (w > width) width = w;
    }
    return width;
  }

  horizontalBarWidth(labelWidth) {
    if (this.width <= 0) return CHART_DEFAULT_BAR_WIDTH;
    let width = this.width;
    if (labelWidth > 0) width -= labelWidth + 1;
    if (width < 1) return 1;
    return width;
  }

  barStyle(index) {
    if (index === this.selected && this.selectedStyle !== '') return this.selectedStyle;
    return this.style;
  }

  barUnits(v, max, height) {
    if (height <= 0 || max <= 0 || Number.isNaN(v) || v <= 0) return 0;
    const t = clampFloat(v / max, 0, 1);
    const units = Math.trunc(t * (height * 8) + 0.5);
    return clampInt(units, 0, height * 8);
  }

  // verticalCells returns the bar grid as [row][bar] cells, top row first.
  verticalCells() {
    const values = this.verticalValues();
    if (values.length === 0) return null;
    const height = this.barHeight();
    const width = this.columnWidth(values.length);
    const max = this.maxValue();
    const units = values.map((v) => this.barUnits(v, max, height));
    const cells = [];
    for (let rowIndex = 0; rowIndex < height; rowIndex++) {
      const k = height - 1 - rowIndex;
      const rowCells = [];
      for (let i = 0; i < values.length; i++) rowCells.push(barCell(units[i], k, width));
      cells.push(rowCells);
    }
    return cells;
  }

  // verticalLines returns the vertical chart as plain text rows.
  verticalLines() {
    const cells = this.verticalCells();
    if (cells == null) return null;
    const width = this.columnWidth(cells[0].length);
    const lines = cells.map((rowCells) => rowCells.join(' '));
    if (this.labels.length > 0) {
      const count = cells[0].length;
      const labels = [];
      for (let i = 0; i < count; i++) {
        labels.push(i < this.labels.length ? padTo(this.labels[i], width) : ' '.repeat(width));
      }
      lines.push(labels.join(' '));
    }
    return lines;
  }

  // horizontalLines returns the horizontal chart as plain text rows.
  horizontalLines() {
    const values = this.horizontalValues();
    if (values.length === 0) return null;
    const labelWidth = this.labelWidth();
    const barWidth = this.horizontalBarWidth(labelWidth);
    const max = this.maxValue();
    const lines = [];
    for (let i = 0; i < values.length; i++) {
      let prefix = '';
      if (labelWidth > 0) {
        const label = i < this.labels.length ? this.labels[i] : '';
        prefix = padTo(label, labelWidth) + ' ';
      }
      lines.push(prefix + '█'.repeat(this.runLength(values[i], max, barWidth)));
    }
    return lines;
  }

  runLength(v, max, width) {
    if (Number.isNaN(v) || v <= 0 || max <= 0) return 0;
    const length = Math.trunc(clampFloat(v / max, 0, 1) * width + 0.5);
    return clampInt(length, 0, width);
  }

  build() {
    if (this.horizontal) return this.buildHorizontal();
    return this.buildVertical();
  }

  buildVertical() {
    const col = box('col');
    const values = this.verticalValues();
    if (values.length === 0) return col;
    const height = this.barHeight();
    const width = this.columnWidth(values.length);
    const max = this.maxValue();
    const units = values.map((v) => this.barUnits(v, max, height));
    for (let rowIndex = 0; rowIndex < height; rowIndex++) {
      const k = height - 1 - rowIndex;
      const line = row().height(1);
      for (let i = 0; i < values.length; i++) {
        if (i > 0) line.child(text(' '));
        line.child(text(barCell(units[i], k, width)).style(this.barStyle(i)));
      }
      col.child(line);
    }
    if (this.labels.length > 0) {
      const line = row().height(1);
      for (let i = 0; i < values.length; i++) {
        if (i > 0) line.child(text(' '));
        const label = i < this.labels.length ? this.labels[i] : '';
        line.child(text(padTo(label, width)).style(this.labelStyle));
      }
      col.child(line);
    }
    return col;
  }

  buildHorizontal() {
    const col = box('col');
    const values = this.horizontalValues();
    if (values.length === 0) return col;
    const labelWidth = this.labelWidth();
    const barWidth = this.horizontalBarWidth(labelWidth);
    const max = this.maxValue();
    for (let i = 0; i < values.length; i++) {
      const line = row().height(1);
      if (labelWidth > 0) {
        const label = i < this.labels.length ? this.labels[i] : '';
        line.child(text(padTo(label, labelWidth) + ' ').style(this.labelStyle));
      }
      const length = this.runLength(values[i], max, barWidth);
      if (length > 0) line.child(text('█'.repeat(length)).style(this.barStyle(i)));
      col.child(line);
    }
    return col;
  }
}

// barCell renders one vertical bar cell for the k-th row from the bottom.
function barCell(units, k, width) {
  if (width < 1) width = 1;
  if (units >= (k + 1) * 8) return '█'.repeat(width);
  if (units > k * 8) {
    const level = units - k * 8;
    return SPARK_GLYPHS[level - 1] + ' '.repeat(width - 1);
  }
  return ' '.repeat(width);
}

class Heatmap {
  constructor(options = {}) {
    this.values = options.values ? options.values.slice() : [];
    this.rowLabels = options.rowLabels ? options.rowLabels.slice() : [];
    this.colLabels = options.colLabels ? options.colLabels.slice() : [];
    this.min = options.min || 0;
    this.max = options.max || 0;
    this.style = options.style || '';
    this.labelStyle = options.labelStyle || '';
    this.shades = options.shades ? options.shades.slice() : null;
  }

  shadeList() {
    if (this.shades != null && this.shades.length > 0) return this.shades;
    return Array.from(DEFAULT_HEAT_SHADES);
  }

  rows() {
    return this.values.length;
  }

  cols() {
    let cols = this.colLabels.length;
    for (const rowValues of this.values) {
      if (rowValues.length > cols) cols = rowValues.length;
    }
    return cols;
  }

  // bounds returns the effective value range. An explicit Min < Max wins;
  // otherwise the range is derived from the finite samples.
  bounds() {
    if (this.min < this.max) return [this.min, this.max];
    let min = Infinity;
    let max = -Infinity;
    for (const rowValues of this.values) {
      for (const v of rowValues) {
        if (!finite(v)) continue;
        if (v < min) min = v;
        if (v > max) max = v;
      }
    }
    if (min === Infinity) return [0, 0];
    return [min, max];
  }

  // level returns the shade index [0, len(Shades)-1] for v.
  level(v) {
    const shades = this.shadeList();
    const count = shades.length;
    if (count <= 0) return 0;
    const [min, max] = this.bounds();
    if (max <= min) return Math.floor(count / 2);
    if (Number.isNaN(v)) return 0;
    const t = clampFloat((v - min) / (max - min), 0, 1);
    return clampInt(Math.trunc(t * (count - 1) + 0.5), 0, count - 1);
  }

  // cell returns the plain shade text of one matrix cell.
  cell(rowIndex, col) {
    const shades = this.shadeList();
    if (shades.length === 0) return '';
    if (rowIndex < 0 || rowIndex >= this.values.length) return shades[0];
    const rowValues = this.values[rowIndex];
    if (col < 0 || col >= rowValues.length) return shades[0];
    return shades[this.level(rowValues[col])];
  }

  rowLabelWidth() {
    let width = 0;
    for (const label of this.rowLabels) {
      const w = displayWidth(label);
      if (w > width) width = w;
    }
    return width;
  }

  colWidth() {
    let width = 1;
    for (const label of this.colLabels) {
      const w = displayWidth(label);
      if (w > width) width = w;
    }
    return width;
  }

  // grid returns the heatmap as plain text rows.
  grid() {
    const rows = this.rows();
    const cols = this.cols();
    if (rows === 0 || cols === 0) return null;
    const shades = this.shadeList();
    if (shades.length === 0) return null;
    const rowLabelWidth = this.rowLabelWidth();
    const colWidth = this.colWidth();
    const gutter = rowLabelWidth > 0 ? ' ' : '';
    const blank = ' '.repeat(rowLabelWidth + gutter.length);
    const lines = [];
    if (this.colLabels.length > 0) {
      let header = blank;
      for (let c = 0; c < cols; c++) {
        const label = c < this.colLabels.length ? this.colLabels[c] : '';
        header += padTo(label, colWidth);
      }
      lines.push(header);
    }
    for (let r = 0; r < rows; r++) {
      let line = '';
      if (rowLabelWidth > 0) {
        const label = r < this.rowLabels.length ? this.rowLabels[r] : '';
        line += padTo(label, rowLabelWidth) + gutter;
      }
      for (let c = 0; c < cols; c++) line += padTo(this.cell(r, c), colWidth);
      lines.push(line);
    }
    return lines;
  }

  build() {
    const col = box('col');
    const lines = this.grid();
    if (lines == null || lines.length === 0) return col;
    col.child(text(lines[0]).style(firstNonEmpty(this.labelStyle, this.style)).height(1));
    for (let i = 1; i < lines.length; i++) {
      col.child(text(lines[i]).style(this.style).height(1));
    }
    return col;
  }
}

class Meter {
  constructor(options = {}) {
    this.id = options.id || '';
    this.value = options.value || 0;
    this.max = options.max || 0;
    this.width = options.width || 0;
    this.label = options.label || '';
    this.showValue = Boolean(options.showValue);
    this.style = options.style || '';
    this.trackStyle = options.trackStyle || '';
    this.labelStyle = options.labelStyle || '';
    this.valueStyle = options.valueStyle || '';
  }

  // fraction returns Value/Max clamped to [0, 1]; a non-positive Max (or NaN
  // Value) is 0.
  fraction() {
    if (this.max <= 0 || Number.isNaN(this.value)) return 0;
    return clampFloat(this.value / this.max, 0, 1);
  }

  barWidth() {
    return this.width > 0 ? this.width : PROGRESS_WIDTH;
  }

  barText() {
    const width = this.barWidth();
    const filled = clampInt(Math.trunc(this.fraction() * width + 0.5), 0, width);
    return PROGRESS_FULL.repeat(filled) + PROGRESS_EMPTY.repeat(width - filled);
  }

  valueText() {
    if (!this.showValue) return '';
    let out = formatFloat(this.value, 0, 0);
    if (this.max > 0) out += '/' + formatFloat(this.max, 0, 0);
    return out;
  }

  build() {
    const line = row().height(1);
    if (this.id !== '') line.id(this.id);
    if (this.label !== '') line.child(text(this.label).style(this.labelStyle).height(1));
    const width = this.barWidth();
    const filled = clampInt(Math.trunc(this.fraction() * width + 0.5), 0, width);
    if (filled > 0) line.child(text(PROGRESS_FULL.repeat(filled)).style(this.style).height(1));
    const empty = width - filled;
    if (empty > 0) line.child(text(PROGRESS_EMPTY.repeat(empty)).style(this.trackStyle).height(1));
    const valueText = this.valueText();
    if (valueText !== '') line.child(text(valueText).style(this.valueStyle).height(1));
    return line;
  }
}

class Legend {
  constructor(options = {}) {
    this.items = options.items ? options.items.slice() : [];
    this.style = options.style || '';
    this.separator = options.separator || '';
  }

  text() {
    const separator = firstNonEmpty(this.separator, ' ');
    let out = '';
    for (let i = 0; i < this.items.length; i++) {
      const item = this.items[i];
      if (i > 0) out += separator;
      out += firstNonEmpty(item.marker, DEFAULT_LEGEND_MARKER);
      if (item.label) out += ' ' + item.label;
    }
    return out;
  }

  build() {
    const line = row().height(1);
    const separator = firstNonEmpty(this.separator, ' ');
    for (let i = 0; i < this.items.length; i++) {
      const item = this.items[i];
      if (i > 0) line.child(text(separator).style(this.style));
      const marker = text(firstNonEmpty(item.marker, DEFAULT_LEGEND_MARKER));
      if (item.color) marker.style(item.color);
      line.child(marker);
      if (item.label) line.child(text(' ' + item.label).style(this.style));
    }
    return line;
  }
}

module.exports = {
  DEFAULT_HEAT_SHADES,
  DEFAULT_LEGEND_MARKER,
  Sparkline,
  BarChart,
  Heatmap,
  Meter,
  Gauge: Meter,
  Legend,
};
