/**
 * Pure text charts (port of tui2/sdk/widgets/chart.go): Sparkline, BarChart,
 * Heatmap, Meter/Gauge and Legend.
 */

import { Node } from '../builder';

export declare const DEFAULT_HEAT_SHADES: string;
export declare const DEFAULT_LEGEND_MARKER: string;

export interface SparklineOptions {
  values?: number[];
  width?: number;
  style?: string;
  min?: number | null;
  max?: number | null;
}

export declare class Sparkline {
  constructor(options?: SparklineOptions);
  values: number[];
  width: number;
  style: string;
  min: number | null;
  max: number | null;
  /** Effective low/high values; empty or all-NaN yields [0, 0]. */
  bounds(): [number, number];
  /** Eighth-block index [0, 7] for v under the effective bounds. */
  level(v: number): number;
  /** Values to render (downsampled to Width buckets when smaller). */
  samples(): number[];
  line(): string;
  build(): Node;
}

export interface BarChartOptions {
  values?: number[];
  labels?: string[];
  width?: number;
  height?: number;
  max?: number;
  style?: string;
  labelStyle?: string;
  selectedStyle?: string;
  selected?: number;
  horizontal?: boolean;
}

export declare class BarChart {
  constructor(options?: BarChartOptions);
  values: number[];
  labels: string[];
  width: number;
  height: number;
  max: number;
  style: string;
  labelStyle: string;
  selectedStyle: string;
  selected: number;
  horizontal: boolean;
  /** Max when positive, else the largest finite value (at least 1). */
  maxValue(): number;
  verticalValues(): number[];
  horizontalValues(): number[];
  verticalLines(): string[] | null;
  horizontalLines(): string[] | null;
  build(): Node;
}

export interface HeatmapOptions {
  values?: number[][];
  rowLabels?: string[];
  colLabels?: string[];
  min?: number;
  max?: number;
  style?: string;
  labelStyle?: string;
  shades?: string[] | null;
}

export declare class Heatmap {
  constructor(options?: HeatmapOptions);
  values: number[][];
  rowLabels: string[];
  colLabels: string[];
  min: number;
  max: number;
  style: string;
  labelStyle: string;
  shades: string[] | null;
  rows(): number;
  cols(): number;
  bounds(): [number, number];
  level(v: number): number;
  cell(row: number, col: number): string;
  grid(): string[] | null;
  build(): Node;
}

export interface MeterOptions {
  id?: string;
  value?: number;
  max?: number;
  width?: number;
  label?: string;
  showValue?: boolean;
  style?: string;
  trackStyle?: string;
  labelStyle?: string;
  valueStyle?: string;
}

export declare class Meter {
  constructor(options?: MeterOptions);
  id: string;
  value: number;
  max: number;
  width: number;
  label: string;
  showValue: boolean;
  style: string;
  trackStyle: string;
  labelStyle: string;
  valueStyle: string;
  /** Value/Max clamped to [0, 1]; a non-positive Max is 0. */
  fraction(): number;
  barText(): string;
  valueText(): string;
  build(): Node;
}

/** Gauge is the alias of Meter. */
export declare const Gauge: typeof Meter;

export interface LegendItem {
  label?: string;
  color?: string;
  marker?: string;
}

export interface LegendOptions {
  items?: LegendItem[];
  style?: string;
  separator?: string;
}

export declare class Legend {
  constructor(options?: LegendOptions);
  items: LegendItem[];
  style: string;
  separator: string;
  text(): string;
  build(): Node;
}
