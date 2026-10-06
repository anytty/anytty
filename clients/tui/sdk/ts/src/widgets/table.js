'use strict';

// Aligned, truncated text tables. Port of tui2/sdk/widgets/table.go.

const { box, text, displayWidth, truncate } = require('../builder');

const ALIGN_LEFT = 'left';
const ALIGN_RIGHT = 'right';
const ALIGN_CENTER = 'center';

const DEFAULT_TABLE_SEPARATOR = ' ';

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

// sprint approximates Go's fmt.Sprint for the values tables usually carry.
function sprint(value) {
  if (value == null) return '<nil>';
  return String(value);
}

// padCell truncates text to width display cells and pads it according to the
// column alignment.
function padCell(value, width, align) {
  let out = truncate(value, width);
  const pad = width - displayWidth(out);
  if (pad <= 0) return out;
  switch (align) {
    case ALIGN_RIGHT:
      return ' '.repeat(pad) + out;
    case ALIGN_CENTER: {
      const left = Math.floor(pad / 2);
      return ' '.repeat(left) + out + ' '.repeat(pad - left);
    }
    default:
      return out + ' '.repeat(pad);
  }
}

class Table {
  constructor(options = {}) {
    this.id = options.id || '';
    this.columns = (options.columns || []).map((col) => ({
      title: col.title || '',
      width: col.width || 0,
      minWidth: col.minWidth || 0,
      flex: col.flex || 0,
      align: col.align || '',
      format: col.format || null,
    }));
    this.rows = options.rows ? options.rows.slice() : [];
    this.records = options.records ? options.records.slice() : [];
    this.width = options.width || 0;
    this.hideHeader = Boolean(options.hideHeader);
    this.rule = Boolean(options.rule);
    this.zebra = Boolean(options.zebra);
    this.zebraStyle = options.zebraStyle || '';
    this.selected = options.selected || 0;
    this.style = options.style || '';
    this.headerStyle = options.headerStyle || '';
    this.selectedStyle = options.selectedStyle || '';
    this.footerStyle = options.footerStyle || '';
    this.separatorStyle = options.separatorStyle || '';
    this.footer = options.footer ? options.footer.slice() : [];
    this.separator = options.separator || '';
    this.rowId = options.rowId || null;
  }

  // rowCount returns the number of data rows (records win over rows).
  rowCount() {
    if (this.records.length > 0) return this.records.length;
    return this.rows.length;
  }

  // cell returns the formatted text of one cell; out-of-range cells are empty.
  cell(row, col) {
    if (this.records.length > 0) {
      if (row < 0 || row >= this.records.length) return '';
      const record = this.records[row];
      if (col < 0 || col >= record.length) return '';
      if (col < this.columns.length && this.columns[col].format != null) {
        return this.columns[col].format(record[col]);
      }
      return sprint(record[col]);
    }
    if (row < 0 || row >= this.rows.length) return '';
    const cells = this.rows[row];
    if (col < 0 || col >= cells.length) return '';
    return cells[col];
  }

  // columnWidths solves the cell widths: natural size plus MinWidth, then Flex
  // distribution over a declared Width, then a right-to-left shrink.
  columnWidths() {
    const widths = new Array(this.columns.length).fill(0);
    if (this.columns.length === 0) return widths;
    for (let i = 0; i < this.columns.length; i++) {
      const col = this.columns[i];
      if (col.width > 0) {
        widths[i] = col.width;
        continue;
      }
      let width = displayWidth(col.title || '');
      for (let row = 0; row < this.rowCount(); row++) {
        const cell = displayWidth(this.cell(row, i));
        if (cell > width) width = cell;
      }
      const min = this.minWidth(i);
      if (width < min) width = min;
      if (width < 1) width = 1;
      widths[i] = width;
    }
    if (this.width <= 0) return widths;
    const avail = this.width - this.sepWidth() * (this.columns.length - 1);
    if (avail <= 0) return widths;
    let used = 0;
    for (const width of widths) used += width;
    if (used < avail) {
      let flex = 0;
      for (const col of this.columns) {
        if (col.width <= 0 && col.flex > 0) flex += col.flex;
      }
      if (flex === 0) return widths;
      const extra = avail - used;
      let added = 0;
      let last = -1;
      for (let i = 0; i < this.columns.length; i++) {
        const col = this.columns[i];
        if (col.width > 0 || col.flex <= 0) continue;
        const share = Math.trunc(extra * col.flex / flex);
        widths[i] += share;
        added += share;
        last = i;
      }
      if (last >= 0) widths[last] += extra - added;
    } else if (used > avail) {
      for (let i = widths.length - 1; i >= 0 && used > avail; i--) {
        if (this.columns[i].width > 0) continue;
        let reduce = used - avail;
        const floor = this.minWidth(i);
        if (widths[i] - reduce < floor) reduce = widths[i] - floor;
        if (reduce > 0) {
          widths[i] -= reduce;
          used -= reduce;
        }
      }
    }
    return widths;
  }

  headerCells() {
    return this.columns.map((col) => col.title || '');
  }

  cells(row) {
    const cells = new Array(this.columns.length);
    for (let i = 0; i < this.columns.length; i++) cells[i] = this.cell(row, i);
    return cells;
  }

  align(col) {
    if (col >= 0 && col < this.columns.length) return this.columns[col].align || '';
    return '';
  }

  minWidth(col) {
    if (col >= 0 && col < this.columns.length && this.columns[col].minWidth > 0) {
      return this.columns[col].minWidth;
    }
    return 1;
  }

  separatorText() {
    if (this.separator === '') return DEFAULT_TABLE_SEPARATOR;
    return this.separator;
  }

  sepWidth() {
    return displayWidth(this.separatorText());
  }

  totalWidth(widths) {
    if (widths.length === 0) return 0;
    let total = this.sepWidth() * (widths.length - 1);
    for (const width of widths) total += width;
    return total;
  }

  rowBox(cells, widths, style, id, row) {
    const line = box('row').height(1);
    if (id !== '') line.id(id).input('mouse');
    for (let i = 0; i < widths.length; i++) {
      if (i > 0) line.child(text(this.separatorText()).style(this.separatorStyle));
      const value = i < cells.length ? cells[i] : '';
      line.child(text(padCell(value, widths[i], this.align(i))).style(style).width(widths[i]).height(1));
    }
    return line;
  }

  // rowAt returns the data row at viewport y, skipping the header and rule.
  rowAt(y) {
    if (this.columns.length === 0) return null;
    let top = 0;
    if (!this.hideHeader) {
      top++;
      if (this.rule) top++;
    }
    const row = y - top;
    if (row < 0 || row >= this.rowCount()) return null;
    return row;
  }

  build() {
    const col = box('col');
    if (this.id !== '') col.id(this.id);
    if (this.width > 0) col.width(this.width);
    if (this.columns.length === 0) return col;
    const widths = this.columnWidths();
    if (!this.hideHeader) {
      col.child(this.rowBox(this.headerCells(), widths, this.headerStyle, '', -1));
      if (this.rule) {
        col.child(text('─'.repeat(this.totalWidth(widths))).style(this.headerStyle).height(1));
      }
    }
    for (let row = 0; row < this.rowCount(); row++) {
      let style = this.style;
      if (this.zebra && row % 2 === 1) style = firstNonEmpty(this.zebraStyle, style);
      if (row === this.selected) style = firstNonEmpty(this.selectedStyle, style);
      let id = '';
      if (this.rowId != null) id = this.rowId(row) || '';
      col.child(this.rowBox(this.cells(row), widths, style, id, row));
    }
    if (this.footer.length > 0) {
      col.child(this.rowBox(this.footer.slice(), widths, this.footerStyle, '', -1));
    }
    return col;
  }
}

module.exports = {
  ALIGN_LEFT,
  ALIGN_RIGHT,
  ALIGN_CENTER,
  DEFAULT_TABLE_SEPARATOR,
  Table,
};
