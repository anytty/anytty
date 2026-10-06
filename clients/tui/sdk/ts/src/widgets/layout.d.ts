/**
 * Layout geometry, chrome bars and floating overlays (port of
 * tui2/sdk/widgets/chrome.go): split-pane solving, title/footer strips, the
 * picker overlay, floating layers and toasts.
 */

import { Node } from '../builder';
import { Button, FrameRow, Segment } from './basics';

export interface RectOptions {
  x?: number;
  y?: number;
  w?: number;
  h?: number;
}

/** One solved rectangle in viewport cells. */
export declare class Rect {
  constructor(options?: RectOptions);
  x: number;
  y: number;
  w: number;
  h: number;
}

/** Split avail cells over integer ratios: proportional, at least one cell
 * each, remainder on the last pane. Ratios <= 0 fall back to equal panes. */
export declare function distribute(avail: number, ratios: number[]): number[];

export interface SplitLayoutOptions {
  /** "row" (left-to-right) or "col" (top-to-bottom). */
  orient?: string;
  /** Relative pane sizes; empty means one pane. */
  weights?: number[];
  /** Separator cells between panes (default 1). */
  gap?: number;
}

/** Lay panes along one axis with a fixed gap; each visible gap is a draggable
 * divider rect the program hit-tests like any other box. */
export declare class SplitLayout {
  constructor(options?: SplitLayoutOptions);
  orient: string;
  weights: number[];
  gap: number;
  /** Resolved orientation ("row" unless "col" is explicit). */
  axis(): 'row' | 'col';
  /** Separator cells between two panes. */
  gapWidth(): number;
  /** Solve the layout: one rect per pane plus one divider rect per gap. */
  rects(width: number, height: number): [Rect[], Rect[]];
}

export interface TitleBarOptions {
  id?: string;
  left?: Segment[];
  buttons?: Button[];
  width?: number;
  fill?: string;
}

/** A one-row title strip: left segments and right-aligned buttons. With
 * Width > 0 the buttons are pushed to the right edge and the left group is
 * truncated. */
export declare class TitleBar {
  constructor(options?: TitleBarOptions);
  id: string;
  left: Segment[];
  buttons: Button[];
  width: number;
  fill: string;
  /** The bar as plain text, for measurement. */
  line(): string;
  build(): Node;
}

export interface FooterOptions {
  id?: string;
  badge?: Segment;
  hasBadge?: boolean;
  groups?: Segment[];
  right?: Segment[];
  width?: number;
  separator?: string;
  sepStyle?: string;
}

/** The bottom bar: an optional scene badge plus key groups on the left and
 * right-aligned summary segments. With Width > 0 the summary is
 * right-aligned and the left groups are truncated. */
export declare class Footer {
  constructor(options?: FooterOptions);
  id: string;
  badge: Segment;
  hasBadge: boolean;
  groups: Segment[];
  right: Segment[];
  width: number;
  separator: string;
  sepStyle: string;
  separatorText(): string;
  sepStyleValue(): string;
  /** The badge (when present) followed by the key groups. */
  leftSegments(): Segment[];
  /** The whole bar as plain text at Width cells, for measurement. */
  line(): string;
  build(): Node;
}

export interface PickerRowOptions {
  text?: string;
  id?: string;
  style?: string;
  selected?: boolean;
  selectable?: boolean;
}

/** One selectable row of a Picker. */
export declare class PickerRow {
  constructor(options?: PickerRowOptions);
  text: string;
  id: string;
  style: string;
  selected: boolean;
  selectable: boolean;
}

export interface PickerOptions {
  id?: string;
  title?: string;
  width?: number;
  height?: number;
  rows?: PickerRow[];
  style?: string;
  marker?: string;
  selectedStyle?: string;
}

/** A framed selectable list (the program-side picker overlay). Row ids are
 * hit-test targets; the program moves the selection itself. */
export declare class Picker {
  constructor(options?: PickerOptions);
  id: string;
  title: string;
  width: number;
  height: number;
  rows: PickerRow[];
  style: string;
  marker: string;
  selectedStyle: string;
  build(): Node;
}

export interface FloatingLayerOptions {
  id?: string;
  title?: string;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  style?: string;
  rows?: FrameRow[];
  collapsed?: boolean;
}

/** A program-side floating window: a Frame placed with Pos so it composites
 * above the regular flow. Collapsed keeps only the title row. */
export declare class FloatingLayer {
  constructor(options?: FloatingLayerOptions);
  id: string;
  title: string;
  x: number;
  y: number;
  width: number;
  height: number;
  style: string;
  rows: FrameRow[];
  collapsed: boolean;
  build(): Node;
}

export interface ToastOptions {
  id?: string;
  text?: string;
  style?: string;
}

/** A transient one-line notice (program-side): a styled text box the program
 * hides by dropping it from the next view. */
export declare class Toast {
  constructor(options?: ToastOptions);
  id: string;
  text: string;
  style: string;
  build(): Node;
}
