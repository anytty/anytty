/**
 * Pure hover tracker (port of tui2/sdk/widgets/hover.go).
 */

import { HitRegion } from './mouse';

export interface HoverOptions {
  node?: string;
  regions?: HitRegion[];
  style?: string;
  currentStyle?: string;
}

export declare class Hover {
  constructor(options?: HoverOptions);
  node: string;
  regions: HitRegion[];
  style: string;
  currentStyle: string;
  /** Hit-test (x, y) and return the hovered region id ("" when none). */
  update(x: number, y: number): string;
  /** Hover accent when id is the hovered target, else "". */
  styleFor(id: string): string;
  hovered(): string;
  clear(): void;
}
