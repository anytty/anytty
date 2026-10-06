/**
 * Progress indicators and chips (port of tui2/sdk/widgets/progress.go).
 * Nothing here owns a timer: the caller advances Value/Frame and commits.
 */

import { Node } from '../builder';

export declare const DEFAULT_PROGRESS_FULL: string;
export declare const DEFAULT_PROGRESS_EMPTY: string;
export declare const DEFAULT_PROGRESS_WIDTH: number;
export declare const SPINNER_FRAMES: string[];

export interface ProgressBarOptions {
  id?: string;
  value?: number;
  max?: number;
  width?: number;
  label?: string;
  hidePercent?: boolean;
  full?: string;
  empty?: string;
  style?: string;
  trackStyle?: string;
  labelStyle?: string;
  percentStyle?: string;
}

/** A fixed-width bar from Value/Max plus an optional label and percentage. */
export declare class ProgressBar {
  constructor(options?: ProgressBarOptions);
  id: string;
  value: number;
  max: number;
  width: number;
  label: string;
  hidePercent: boolean;
  full: string;
  empty: string;
  style: string;
  trackStyle: string;
  labelStyle: string;
  percentStyle: string;
  /** Value/Max clamped to [0, 1]; a non-positive Max is 0. */
  fraction(): number;
  /** Rounded percentage in [0, 100]. */
  percent(): number;
  /** The full bar string (filled + empty glyphs). */
  barText(): string;
  build(): Node;
  barWidth(): number;
  fullGlyph(): string;
  emptyGlyph(): string;
}

export interface SpinnerOptions {
  id?: string;
  frame?: number;
  frames?: string[];
  style?: string;
  label?: string;
  labelStyle?: string;
}

/** One frame of an animation; Frame is wrapped for negative values. */
export declare class Spinner {
  constructor(options?: SpinnerOptions);
  id: string;
  frame: number;
  frames: string[];
  style: string;
  label: string;
  labelStyle: string;
  /** The current frame glyph. */
  frameText(): string;
  build(): Node;
}

export interface BadgeOptions {
  id?: string;
  text?: string;
  style?: string;
  input?: string[];
}

/** A small styled label (status chip, count, key marker). */
export declare class Badge {
  constructor(options?: BadgeOptions);
  id: string;
  text: string;
  style: string;
  input: string[];
  build(): Node;
}

export interface TagOptions {
  text?: string;
  id?: string;
  style?: string;
  input?: string[];
}

/** One item of Tags. */
export declare class Tag {
  constructor(options?: TagOptions);
  text: string;
  id: string;
  style: string;
  input: string[];
}

export interface TagsOptions {
  id?: string;
  items?: Tag[];
  separator?: string;
  sepStyle?: string;
  style?: string;
}

/** A row of tagged labels separated by Separator (default " "). */
export declare class Tags {
  constructor(options?: TagsOptions);
  id: string;
  items: Tag[];
  separator: string;
  sepStyle: string;
  style: string;
  build(): Node;
}
