/**
 * Proportional scroll indicator (port of tui2/sdk/widgets/scrollbar.go).
 */

import { Node } from '../builder';

export declare const DEFAULT_SCROLLBAR_TRACK_STYLE: string;
export declare const DEFAULT_SCROLLBAR_THUMB_STYLE: string;
export declare const DEFAULT_SCROLLBAR_UP: string;
export declare const DEFAULT_SCROLLBAR_DOWN: string;
export declare const SCROLLBAR_TRACK_VERTICAL: string;
export declare const SCROLLBAR_THUMB_VERTICAL: string;
export declare const SCROLLBAR_TRACK_HORIZONTAL: string;
export declare const SCROLLBAR_THUMB_HORIZONTAL: string;

export interface ScrollbarOptions {
  id?: string;
  total?: number;
  visible?: number;
  offset?: number;
  height?: number;
  width?: number;
  vertical?: boolean;
  trackStyle?: string;
  thumbStyle?: string;
  up?: string;
  down?: string;
}

export declare class Scrollbar {
  constructor(options?: ScrollbarOptions);
  id: string;
  total: number;
  visible: number;
  offset: number;
  height: number;
  width: number;
  vertical: boolean;
  trackStyle: string;
  thumbStyle: string;
  up: string;
  down: string;
  /** Resolved orientation: explicit Vertical wins, else Height means vertical. */
  isVertical(): boolean;
  /** Track cells (height when vertical, width when horizontal). */
  trackLength(): number;
  /** Thumb start cell and size along the track. */
  thumb(): [number, number];
  /** Map a click position (0-based cells) to an offset, centering the thumb. */
  offsetAt(pos: number): number;
  build(): Node;
}
