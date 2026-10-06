/**
 * Pure mouse helpers (port of tui2/sdk/widgets/mouse.go): hit regions, wheel
 * folding, drag selection and click classification.
 */

import { List, VirtualList } from './list';
import { Table } from './table';

export declare const DEFAULT_ROW_HEIGHT: number;
export declare const DEFAULT_CLICK_THRESHOLD: number;

/** One rectangular hit-test target. A W or H <= 0 is unbounded on that axis. */
export interface HitRegionOptions {
  id?: string;
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  data?: unknown;
}

export declare class HitRegion {
  constructor(options?: HitRegionOptions);
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  data: unknown;
  contains(x: number, y: number): boolean;
  rect(): { x: number; y: number; w: number; h: number };
}

/** Topmost region containing (x, y); later regions win. */
export declare function hit(regions: HitRegion[], x: number, y: number): HitRegion | null;

/** hit's convenience form. */
export declare function hitId(regions: HitRegion[], x: number, y: number): { id: string; ok: boolean };

/** One region per visible List row; data is the absolute row index. */
export declare function listRegions(list: List, originX: number, originY: number, width: number): HitRegion[];

/** One region per visible VirtualList row. */
export declare function virtualListRegions(list: VirtualList, originX: number, originY: number, width: number): HitRegion[];

/** One region per visible Table cell; data is a [row, col] pair. */
export declare function tableRegions(table: Table, originX: number, originY: number): HitRegion[];

/** Fold one wheel delta into an offset, clamped to [0, total-visible]. */
export declare function applyWheel(offset: number, total: number, visible: number, delta: number): number;

/** Table counterpart of applyWheel. */
export declare function tableScroll(offset: number, total: number, visible: number, delta: number): number;

/** Normalize two positions into an inclusive [lo, hi] pair. */
export declare function selectRange(a: number, b: number): [number, number];

/** Inclusive absolute List row range spanning two viewport y positions. */
export declare function listSelectRange(list: List, y0: number, y1: number): [number, number] | null;

/** Inclusive absolute VirtualList row range spanning two y positions. */
export declare function virtualListSelectRange(list: VirtualList, y0: number, y1: number): [number, number] | null;

export interface DragOptions {
  active?: boolean;
  startX?: number;
  startY?: number;
  x?: number;
  y?: number;
  button?: string;
  thresholdPx?: number;
}

export declare class Drag {
  constructor(options?: DragOptions);
  active: boolean;
  startX: number;
  startY: number;
  x: number;
  y: number;
  button: string;
  thresholdPx: number;
  begin(x: number, y: number, button: string): void;
  update(x: number, y: number): void;
  /** Finish the drag and return the normalized rect, or null when inactive. */
  end(): { x: number; y: number; w: number; h: number } | null;
  /** Normalized selection: min corner plus inclusive span (at least 1). */
  rect(): { x: number; y: number; w: number; h: number };
}

export interface ClickTrackerOptions {
  lastAt?: number | null;
  lastX?: number;
  lastY?: number;
  threshold?: number;
  count?: number;
}

/** Classifies clicks into single/double/triple clicks from timing and position
 * (times in milliseconds). */
export declare class ClickTracker {
  constructor(options?: ClickTrackerOptions);
  lastAt: number | null;
  lastX: number;
  lastY: number;
  threshold: number;
  count: number;
  click(x: number, y: number, at: number): number;
}
