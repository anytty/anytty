/**
 * Style-run bars and framed panels (port of tui2/sdk/widgets/widgets.go). The
 * shared primitives behind the layout chrome: segments, tab/status bars, the
 * program-side Frame, cards, buttons and key hints.
 */

import { Node } from '../builder';

export declare const DEFAULT_ACTIVE_STYLE: string;
export declare const DEFAULT_INACTIVE_STYLE: string;
export declare const DEFAULT_CHROME_STYLE: string;
export declare const DEFAULT_STATUS_STYLE: string;
export declare const DEFAULT_SEP_STYLE: string;
export declare const DEFAULT_KEY_STYLE: string;
export declare const DEFAULT_BORDER_STYLE: string;
export declare const DEFAULT_BUTTON_STYLE: string;
export declare const DEFAULT_SEPARATOR: string;

export interface SegmentOptions {
  text?: string;
  style?: string;
  id?: string;
  input?: string[];
}

/** One styled text run of a bar. */
export declare class Segment {
  constructor(options?: SegmentOptions);
  text: string;
  style: string;
  id: string;
  input: string[];
}

export interface TabItemOptions {
  id?: string;
  title?: string;
  active?: boolean;
}

/** One tab: the id the caller hit-tests, the visible title and whether it is
 * the active one. */
export declare class TabItem {
  constructor(options?: TabItemOptions);
  id: string;
  title: string;
  active: boolean;
}

export interface TabBarOptions {
  left?: Segment | null;
  items?: TabItem[];
  plus?: boolean;
  plusId?: string;
  plusText?: string;
  activeStyle?: string;
  inactiveStyle?: string;
  chromeStyle?: string;
}

/** The top tab strip: an optional left segment, one clickable box per tab and
 * an optional "+" new-tab box. */
export declare class TabBar {
  constructor(options?: TabBarOptions);
  left: Segment | null;
  items: TabItem[];
  plus: boolean;
  plusId: string;
  plusText: string;
  activeStyle: string;
  inactiveStyle: string;
  chromeStyle: string;
  activeStyleValue(): string;
  inactiveStyleValue(): string;
  chromeStyleValue(): string;
  build(): Node;
}

export interface StatusBarOptions {
  id?: string;
  left?: Segment[];
  right?: Segment[];
  width?: number;
  separator?: string;
  sepStyle?: string;
}

/** A one-row status line from styled segments. With Width > 0 the right group
 * is right-aligned and the left group is trimmed to fit. */
export declare class StatusBar {
  constructor(options?: StatusBarOptions);
  id: string;
  left: Segment[];
  right: Segment[];
  width: number;
  separator: string;
  sepStyle: string;
  /** Segments rendered as one plain string, for measurement. */
  text(): string;
  segmentTexts(): string[];
  separatorText(): string;
  build(): Node;
}

export interface FrameRowOptions {
  text?: string;
  style?: string;
  id?: string;
  input?: string[];
}

/** One content row of a Frame. */
export declare class FrameRow {
  constructor(options?: FrameRowOptions);
  text: string;
  style: string;
  id: string;
  input: string[];
}

export interface FrameOptions {
  id?: string;
  title?: string;
  width?: number;
  height?: number;
  style?: string;
  rows?: FrameRow[];
  fillStyle?: string;
}

/** A program-side bordered panel: a top rule with the title, side bars on
 * every row and a bottom rule. Width/height <= 0 fall back to the row content
 * size; boxes too small for chrome degrade to plain rows. */
export declare class Frame {
  constructor(options?: FrameOptions);
  id: string;
  title: string;
  width: number;
  height: number;
  style: string;
  rows: FrameRow[];
  fillStyle: string;
  build(): Node;
}

export interface DividerOptions {
  id?: string;
  vertical?: boolean;
  length?: number;
  style?: string;
  input?: string[];
}

/** A one-cell rule used as a pane gutter: a vertical run of "│" cells (length
 * rows) or a horizontal run of "─" cells (length cols). */
export declare class Divider {
  constructor(options?: DividerOptions);
  id: string;
  vertical: boolean;
  length: number;
  style: string;
  input: string[];
  build(): Node;
}

export interface CardOptions {
  id?: string;
  title?: string;
  lines?: string[];
  width?: number;
  height?: number;
  style?: string;
  lineStyle?: string;
  center?: boolean;
}

/** A framed text box, optionally centering its lines in the declared
 * geometry. */
export declare class Card {
  constructor(options?: CardOptions);
  id: string;
  title: string;
  lines: string[];
  width: number;
  height: number;
  style: string;
  lineStyle: string;
  center: boolean;
  build(): Node;
}

export interface ButtonOptions {
  id?: string;
  text?: string;
  hot?: string;
  style?: string;
  input?: string[];
}

/** A clickable text box ("clickable" is a protocol input flag plus the
 * caller's hit handling). A non-empty Hot character is rendered as its own
 * accent-styled run. */
export declare class Button {
  constructor(options?: ButtonOptions);
  id: string;
  text: string;
  hot: string;
  style: string;
  input: string[];
  /** The button label as plain text, for measurement. */
  line(): string;
  build(): Node;
  plainBox(style: string, inputs: string[]): Node;
}

export interface KeyHintOptions {
  id?: string;
  mode?: string;
  keys?: string[];
  modeStyle?: string;
  keyStyle?: string;
  sepStyle?: string;
}

/** The left footer group: the current mode plus the bindings that work in
 * it. */
export declare class KeyHint {
  constructor(options?: KeyHintOptions);
  id: string;
  mode: string;
  keys: string[];
  modeStyle: string;
  keyStyle: string;
  sepStyle: string;
  /** The hint as one plain string, for measurement. */
  text(): string;
  build(): Node;
}

/** Append segs to rowNode separated by styled sepText. */
export declare function appendGroup(rowNode: Node, segs: Segment[], sepStyle: string, sepText: string): void;

/** One segment as a text box, styled with seg.style or fallbackStyle. */
export declare function segmentBox(seg: Segment, fallbackStyle: string): Node;

/** Width of segs in display cells, including separators between them. */
export declare function groupWidth(segs: Segment[], sepW: number): number;

/** Trim segs to budget cells: trailing segments are dropped first, then the
 * last remaining text is truncated. */
export declare function fitSegments(segs: Segment[], budget: number, sepW: number): [Segment[], number];

/** Plain text of every segment. */
export declare function segmentTexts(segs: Segment[]): string[];

/** The first non-empty value, or "". */
export declare function firstNonEmpty(...values: string[]): string;
