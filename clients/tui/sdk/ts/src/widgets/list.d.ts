/**
 * Windowed lists (port of tui2/sdk/widgets/list.go).
 */

import { Node } from '../builder';

export declare const DEFAULT_LIST_MARKER: string;

/** One row of a VirtualList. */
export interface ListRow {
  text: string;
  id?: string;
  style?: string;
  disabled?: boolean;
}

export interface WheelLike {
  delta?: number;
}

export interface ListOptions {
  id?: string;
  items?: string[];
  width?: number;
  height?: number;
  offset?: number;
  selected?: number;
  follow?: boolean;
  header?: string;
  footer?: string;
  empty?: string;
  marker?: string;
  style?: string;
  selectedStyle?: string;
  headerStyle?: string;
  footerStyle?: string;
  emptyStyle?: string;
  rowId?: (index: number) => string;
  rowStyle?: (index: number, selected: boolean) => string;
}

/** A windowed one-line-per-item list. */
export declare class List {
  constructor(options?: ListOptions);
  id: string;
  items: string[];
  width: number;
  height: number;
  offset: number;
  selected: number;
  follow: boolean;
  header: string;
  footer: string;
  empty: string;
  marker: string;
  style: string;
  selectedStyle: string;
  headerStyle: string;
  footerStyle: string;
  emptyStyle: string;
  rowId: ((index: number) => string) | null;
  rowStyle: ((index: number, selected: boolean) => string) | null;
  /** Half-open row window [start, end) Build renders. */
  visibleRange(): [number, number];
  /** Scroll the window so selected is visible. */
  ensureVisible(): void;
  /** Shift the selection by delta and follow it when follow is set. */
  move(delta: number): void;
  pageUp(): void;
  pageDown(): void;
  top(): void;
  bottom(): void;
  /** Absolute row at viewport y, or null. */
  rowAt(y: number): number | null;
  /** Apply one wheel event to the offset, leaving selection untouched. */
  onWheel(ev: WheelLike | null | undefined): void;
  build(): Node;
}

export interface VirtualListOptions {
  id?: string;
  rows?: ListRow[];
  width?: number;
  height?: number;
  offset?: number;
  selected?: number;
  follow?: boolean;
  header?: string;
  footer?: string;
  empty?: string;
  marker?: string;
  style?: string;
  selectedStyle?: string;
  disabledStyle?: string;
  headerStyle?: string;
  footerStyle?: string;
  emptyStyle?: string;
}

/** List with rich ListRow data. */
export declare class VirtualList {
  constructor(options?: VirtualListOptions);
  id: string;
  rows: ListRow[];
  width: number;
  height: number;
  offset: number;
  selected: number;
  follow: boolean;
  header: string;
  footer: string;
  empty: string;
  marker: string;
  style: string;
  selectedStyle: string;
  disabledStyle: string;
  headerStyle: string;
  footerStyle: string;
  emptyStyle: string;
  visibleRange(): [number, number];
  ensureVisible(): void;
  move(delta: number): void;
  pageUp(): void;
  pageDown(): void;
  top(): void;
  bottom(): void;
  rowAt(y: number): number | null;
  onWheel(ev: WheelLike | null | undefined): void;
  build(): Node;
}
