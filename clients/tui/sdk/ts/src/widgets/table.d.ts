/**
 * Aligned text tables (port of tui2/sdk/widgets/table.go).
 */

import { Node } from '../builder';

export declare const ALIGN_LEFT: 'left';
export declare const ALIGN_RIGHT: 'right';
export declare const ALIGN_CENTER: 'center';
export declare const DEFAULT_TABLE_SEPARATOR: string;

/** One Table column spec. Width > 0 is fixed; otherwise the width is the
 * widest of title and formatted cells, raised to MinWidth. Flex distributes
 * leftover cells on a declared Table.Width. */
export interface Column {
  title?: string;
  width?: number;
  minWidth?: number;
  flex?: number;
  align?: string;
  format?: (value: unknown) => string;
}

export interface TableOptions {
  id?: string;
  columns?: Column[];
  rows?: string[][];
  records?: unknown[][];
  width?: number;
  hideHeader?: boolean;
  rule?: boolean;
  zebra?: boolean;
  zebraStyle?: string;
  selected?: number;
  style?: string;
  headerStyle?: string;
  selectedStyle?: string;
  footerStyle?: string;
  separatorStyle?: string;
  footer?: string[];
  separator?: string;
  rowId?: (index: number) => string;
}

export declare class Table {
  constructor(options?: TableOptions);
  id: string;
  columns: Column[];
  rows: string[][];
  records: unknown[][];
  width: number;
  hideHeader: boolean;
  rule: boolean;
  zebra: boolean;
  zebraStyle: string;
  selected: number;
  style: string;
  headerStyle: string;
  selectedStyle: string;
  footerStyle: string;
  separatorStyle: string;
  footer: string[];
  separator: string;
  rowId: ((index: number) => string) | null;
  /** Number of data rows (records win over rows). */
  rowCount(): number;
  /** Formatted text of one cell; out-of-range cells are empty. */
  cell(row: number, col: number): string;
  /** Solved cell widths: natural size, MinWidth, Flex, then a shrink to fit. */
  columnWidths(): number[];
  /** Data row at viewport y, or null for the header/rule/footer/out of range. */
  rowAt(y: number): number | null;
  build(): Node;
}
