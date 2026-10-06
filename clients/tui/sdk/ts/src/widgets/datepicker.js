'use strict';

// Calendar dates, a month grid and a validated date text field. Dates are
// plain year/month/day triples independent of time zones and the clock; the
// civil arithmetic below is the proleptic Gregorian calendar Go's time
// package uses in UTC. Port of tui2/sdk/widgets/datepicker.go.

const { box, row, text, displayWidth, truncate } = require('../builder');
const { ALIGN_CENTER } = require('./table');
const { Field } = require('./form');

const DATE_LAYOUT = '2006-01-02';
const DATE_FIELD_MESSAGE = 'invalid date';
const DEFAULT_CALENDAR_SELECTED_STYLE = 'reverse';

const MONTH_NAMES = [
  '', 'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const WEEKDAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const CALENDAR_WEEKDAYS = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value) return value;
  }
  return '';
}

function padCell(value, width, align) {
  let out = truncate(value, width);
  const pad = width - displayWidth(out);
  if (pad <= 0) return out;
  if (align === 'right') return ' '.repeat(pad) + out;
  if (align === 'center') {
    const left = Math.floor(pad / 2);
    return ' '.repeat(left) + out + ' '.repeat(pad - left);
  }
  return out + ' '.repeat(pad);
}

// daysFromCivil is Howard Hinnant's days_from_civil (days since 1970-01-01).
function daysFromCivil(y, m, d) {
  y -= m <= 2 ? 1 : 0;
  const era = Math.floor(y / 400);
  const yoe = y - era * 400;
  const doy = Math.floor((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1;
  const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
  return era * 146097 + doe - 719468;
}

// civilFromDays is Howard Hinnant's civil_from_days.
function civilFromDays(z) {
  z += 719468;
  const era = Math.floor(z / 146097);
  const doe = z - era * 146097;
  const yoe = Math.floor((doe - Math.floor(doe / 1460) + Math.floor(doe / 36524) - Math.floor(doe / 146096)) / 365);
  const y = yoe + era * 400;
  const doy = doe - (365 * yoe + Math.floor(yoe / 4) - Math.floor(yoe / 100));
  const mp = Math.floor((5 * doy + 2) / 153);
  const d = doy - Math.floor((153 * mp + 2) / 5) + 1;
  const m = mp + (mp < 10 ? 3 : -9);
  return { year: y + (m <= 2 ? 1 : 0), month: m, day: d };
}

function isLeapYear(year) {
  return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}

function daysInMonth(year, month) {
  if (month < 1 || month > 12) return 0;
  switch (month) {
    case 4: case 6: case 9: case 11:
      return 30;
    case 2:
      return isLeapYear(year) ? 29 : 28;
    default:
      return 31;
  }
}

function addMonths(year, month, n) {
  if (month < 1 || month > 12) return [year, month];
  const index = year * 12 + (month - 1) + n;
  let targetYear = Math.trunc(index / 12);
  let targetMonth = index % 12;
  if (targetMonth < 0) {
    targetMonth += 12;
    targetYear--;
  }
  return [targetYear, targetMonth + 1];
}

function sign(n) {
  if (n < 0) return -1;
  if (n > 0) return 1;
  return 0;
}

function pad2(n) {
  return String(n).padStart(2, '0');
}

function pad4(n) {
  const digits = String(Math.abs(n));
  const signChar = n < 0 ? '-' : '';
  return signChar + digits.padStart(4 - signChar.length, '0');
}

class Date {
  constructor(options = {}) {
    this.year = options.year || 0;
    this.month = options.month || 0;
    this.day = options.day || 0;
  }

  isLeap() {
    return isLeapYear(this.year);
  }

  daysInMonth() {
    return daysInMonth(this.year, this.month);
  }

  // weekday returns the day of week, where Sunday is 0.
  weekday() {
    const civil = this.civil();
    const days = daysFromCivil(civil.year, civil.month, civil.day);
    return ((days + 4) % 7 + 7) % 7;
  }

  weekdayName() {
    return WEEKDAY_NAMES[this.weekday()];
  }

  compare(other) {
    if (this.year !== other.year) return sign(this.year - other.year);
    if (this.month !== other.month) return sign(this.month - other.month);
    if (this.day !== other.day) return sign(this.day - other.day);
    return 0;
  }

  equals(other) {
    return this.compare(other) === 0;
  }

  before(other) {
    return this.compare(other) < 0;
  }

  after(other) {
    return this.compare(other) > 0;
  }

  // addDays returns the date n days after d (n may be negative).
  addDays(n) {
    if (!this.valid()) return new Date(this);
    const civil = this.civil();
    const shifted = civilFromDays(daysFromCivil(civil.year, civil.month, civil.day) + n);
    return new Date(shifted);
  }

  // addMonths returns the date n months after d, clamping the day to the
  // target month's length.
  addMonths(n) {
    if (!this.valid()) return new Date(this);
    const [year, month] = addMonths(this.year, this.month, n);
    let day = this.day;
    const maxDay = daysInMonth(year, month);
    if (day > maxDay) day = maxDay;
    return new Date({ year, month, day });
  }

  // toString renders the date as YYYY-MM-DD; the zero Date renders as "".
  toString() {
    if (!this.valid()) return '';
    return pad4(this.year) + '-' + pad2(this.month) + '-' + pad2(this.day);
  }

  valid() {
    return this.month >= 1 && this.month <= 12 && this.day >= 1 && this.day <= 31;
  }

  // civil normalizes out-of-range month/day fields the way time.Date does.
  civil() {
    const total = this.year * 12 + (this.month - 1);
    let year = Math.trunc(total / 12);
    let month = total % 12;
    if (month < 0) {
      month += 12;
      year--;
    }
    return civilFromDays(daysFromCivil(year, month + 1, 1) + (this.day - 1));
  }
}

// parseDate parses "YYYY-MM-DD" and rejects any value Go's time would silently
// normalize (2024-02-30, 2023-13-01, trailing characters).
function parseDate(s) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (match == null) throw new Error('invalid date "' + s + '"');
  const date = new Date({ year: Number(match[1]), month: Number(match[2]), day: Number(match[3]) });
  if (!date.valid() || date.day > daysInMonth(date.year, date.month) || date.toString() !== s) {
    throw new Error('invalid date "' + s + '"');
  }
  return date;
}

function dateKey(date) {
  return date.toString();
}

function normalizeMarked(marked) {
  const out = new Map();
  if (marked instanceof Map) {
    for (const [key, value] of marked) out.set(typeof key === 'string' ? key : key.toString(), value);
    return out;
  }
  if (Array.isArray(marked)) {
    for (const [key, value] of marked) out.set(typeof key === 'string' ? key : key.toString(), value);
    return out;
  }
  if (marked != null && typeof marked === 'object') {
    for (const [key, value] of Object.entries(marked)) out.set(key, value);
    return out;
  }
  return out;
}

function calendarCellText(date, width) {
  return padCell(String(date.day).padStart(2, ' '), width, ALIGN_CENTER);
}

function calendarWeekdayRow(style, cellWidth) {
  const line = row().height(1);
  for (const name of CALENDAR_WEEKDAYS) {
    line.child(text(padCell(name, cellWidth, ALIGN_CENTER)).style(style).width(cellWidth).height(1));
  }
  return line;
}

class Calendar {
  constructor(options = {}) {
    this.id = options.id || '';
    this.month = options.month instanceof Date ? options.month : new Date(options.month || {});
    this.selected = options.selected instanceof Date ? options.selected : new Date(options.selected || {});
    this.width = options.width || 0;
    this.marked = normalizeMarked(options.marked);
    this.style = options.style || '';
    this.selectedStyle = options.selectedStyle || '';
    this.todayStyle = options.todayStyle || '';
    this.headerStyle = options.headerStyle || '';
    this.weekdayStyle = options.weekdayStyle || '';
    this.markedStyle = options.markedStyle || '';
    this.today = options.today instanceof Date ? options.today : new Date(options.today || {});
  }

  // monthLabel returns the header text ("January 2024") for the month.
  monthLabel() {
    if (this.month.month < 1 || this.month.month > 12) return '';
    return MONTH_NAMES[this.month.month] + ' ' + this.month.year;
  }

  // grid returns the displayed month as 6 rows of 7 Dates, Monday first.
  grid() {
    const first = new Date({ year: this.month.year, month: this.month.month, day: 1 });
    if (first.month < 1 || first.month > 12) return null;
    const offset = (first.weekday() + 6) % 7;
    const start = first.addDays(-offset);
    const grid = [];
    for (let rowIndex = 0; rowIndex < 6; rowIndex++) {
      const cells = [];
      for (let col = 0; col < 7; col++) cells.push(start.addDays(rowIndex * 7 + col));
      grid.push(cells);
    }
    return grid;
  }

  // moveCursor moves the cursor by days, following it into the adjacent month
  // when it leaves the grid. It reports whether the cursor moved.
  moveCursor(days) {
    if (days === 0) return false;
    const cursor = this.normalizedSelected();
    const next = cursor.addDays(days);
    this.selected = next;
    if (next.month !== this.month.month || next.year !== this.month.year) {
      this.month = new Date({ year: next.year, month: next.month, day: 1 });
    }
    return true;
  }

  normalizedSelected() {
    if (this.selected.month >= 1 && this.selected.month <= 12 && this.selected.day >= 1) {
      return this.selected;
    }
    if (this.month.month >= 1 && this.month.month <= 12) {
      return new Date({ year: this.month.year, month: this.month.month, day: 1 });
    }
    return this.month;
  }

  cellWidth(grid) {
    let width = 3;
    for (const rowDates of grid) {
      for (const date of rowDates) {
        width = Math.max(width, displayWidth(calendarCellText(date, 0)));
      }
    }
    return width;
  }

  cellStyle(date) {
    if (date.equals(this.normalizedSelected())) {
      return firstNonEmpty(this.selectedStyle, DEFAULT_CALENDAR_SELECTED_STYLE);
    }
    const key = dateKey(date);
    if (this.marked.has(key) && this.marked.get(key) !== '') {
      return firstNonEmpty(this.markedStyle, this.style);
    }
    if (this.today.month >= 1 && date.equals(this.today)) {
      return firstNonEmpty(this.todayStyle, this.style);
    }
    return this.style;
  }

  // build returns the calendar as a column: header, weekday row, then 6 grid
  // rows. Every cell is padded to a fixed display width.
  build() {
    const col = box('col');
    if (this.id !== '') col.id(this.id);
    if (this.width > 0) col.width(this.width);
    if (this.style !== '') col.style(this.style);
    const grid = this.grid();
    const width = this.cellWidth(grid || []);
    const label = this.monthLabel();
    if (label !== '') col.child(text(label).style(this.headerStyle).height(1));
    col.child(calendarWeekdayRow(this.weekdayStyle, width));
    if (grid == null) return col;
    for (const rowDates of grid) {
      const line = row().height(1);
      if (this.width > 0) line.width(this.width);
      for (const date of rowDates) {
        line.child(text(calendarCellText(date, width)).style(this.cellStyle(date)).width(width).height(1));
      }
      col.child(line);
    }
    return col;
  }
}

// dayValidator returns a Validator accepting "YYYY-MM-DD" dates inside the
// inclusive [min, max] range; the zero Date leaves a side open.
function dayValidator(min, max, msg) {
  const minimum = min instanceof Date ? min : new Date(min || {});
  const maximum = max instanceof Date ? max : new Date(max || {});
  return (value) => {
    if (value === '') return '';
    let date;
    try {
      date = parseDate(value);
    } catch (err) {
      return msg;
    }
    if (minimum.valid() && date.before(minimum)) return msg;
    if (maximum.valid() && date.after(maximum)) return msg;
    return '';
  };
}

// DateField wraps a TextInput with date validation and a display helper.
class DateField extends Field {
  constructor(options = {}) {
    super(options);
    this.min = options.min instanceof Date ? options.min : new Date(options.min || {});
    this.max = options.max instanceof Date ? options.max : new Date(options.max || {});
    this.validate = (value) => dayValidator(this.min, this.max, DATE_FIELD_MESSAGE)(value);
  }

  // date parses the current input text, returning null when empty or
  // malformed.
  date() {
    if (this.input == null) return null;
    try {
      return parseDate(this.input.text());
    } catch (err) {
      return null;
    }
  }
}

function newDateField(id, label, input) {
  return new DateField({ id, label, input });
}

module.exports = {
  DATE_LAYOUT,
  DATE_FIELD_MESSAGE,
  DEFAULT_CALENDAR_SELECTED_STYLE,
  Date,
  parseDate,
  dayValidator,
  Calendar,
  DateField,
  newDateField,
};
