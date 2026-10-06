/**
 * Calendar dates, month grid and date text field (port of
 * tui2/sdk/widgets/datepicker.go). Dates are year/month/day triples using the
 * proleptic Gregorian calendar, with no time zone or clock dependency.
 */

import { Node } from '../builder';
import { Field, FieldOptions } from './form';
import { TextInput } from './input';
import { Validator } from './validate';

export declare const DATE_LAYOUT: string;
export declare const DATE_FIELD_MESSAGE: string;
export declare const DEFAULT_CALENDAR_SELECTED_STYLE: string;

export interface DateOptions {
  year?: number;
  month?: number;
  day?: number;
}

export declare class Date {
  constructor(options?: DateOptions | Date);
  year: number;
  month: number;
  day: number;
  isLeap(): boolean;
  /** Days in the date's month, or 0 when the month is out of range. */
  daysInMonth(): number;
  /** Day of week, where Sunday is 0. */
  weekday(): number;
  /** Day-of-week name ("Monday"). */
  weekdayName(): string;
  /** -1, 0 or 1 as this date sorts before, equal to, or after other. */
  compare(other: Date): number;
  equals(other: Date): boolean;
  before(other: Date): boolean;
  after(other: Date): boolean;
  addDays(n: number): Date;
  addMonths(n: number): Date;
  /** "YYYY-MM-DD"; the zero Date renders as "". */
  toString(): string;
}

/** Parse "YYYY-MM-DD"; malformed or normalized dates throw. */
export declare function parseDate(s: string): Date;

/** Validator accepting "YYYY-MM-DD" dates inside the inclusive [min, max]. */
export declare function dayValidator(min: Date, max: Date, msg: string): Validator;

export interface CalendarOptions {
  id?: string;
  /** First day of the displayed month (only year/month matter). */
  month?: Date | DateOptions;
  selected?: Date | DateOptions;
  width?: number;
  /** Marked dates keyed by "YYYY-MM-DD" (or a Map/array of pairs). */
  marked?: Map<string, string> | Array<[string | Date, string]> | Record<string, string>;
  style?: string;
  selectedStyle?: string;
  todayStyle?: string;
  headerStyle?: string;
  weekdayStyle?: string;
  markedStyle?: string;
  today?: Date | DateOptions;
}

export declare class Calendar {
  constructor(options?: CalendarOptions);
  id: string;
  month: Date;
  selected: Date;
  width: number;
  marked: Map<string, string>;
  style: string;
  selectedStyle: string;
  todayStyle: string;
  headerStyle: string;
  weekdayStyle: string;
  markedStyle: string;
  today: Date;
  /** Header text ("January 2024") for the displayed month. */
  monthLabel(): string;
  /** Displayed month as 6 rows of 7 Dates, Monday first; null when unset. */
  grid(): Date[][] | null;
  /** Move the cursor by days, following it into the adjacent month. */
  moveCursor(days: number): boolean;
  build(): Node;
}

export interface DateFieldOptions extends FieldOptions {
  min?: Date | DateOptions;
  max?: Date | DateOptions;
}

export declare class DateField extends Field {
  constructor(options?: DateFieldOptions);
  min: Date;
  max: Date;
  /** Parsed current input text, or null when empty or malformed. */
  date(): Date | null;
}

/** Build a DateField with the standard date validator attached. */
export declare function newDateField(id: string, label: string, input: TextInput | null): DateField;
