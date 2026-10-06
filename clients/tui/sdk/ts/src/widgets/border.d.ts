/**
 * Box-drawing border sets and the program-side bordered container (port of
 * tui2/sdk/widgets/border.go).
 */

import { Node } from '../builder';

export interface BorderSetOptions {
  topLeft?: string;
  top?: string;
  topRight?: string;
  left?: string;
  right?: string;
  bottomLeft?: string;
  bottom?: string;
  bottomRight?: string;
}

/** A box-drawing glyph set. Corners and side pieces are single glyphs; Top,
 * Bottom, Left and Right are the repeated edge pieces. */
export declare class BorderSet {
  constructor(options?: BorderSetOptions);
  topLeft: string;
  top: string;
  topRight: string;
  left: string;
  right: string;
  bottomLeft: string;
  bottom: string;
  bottomRight: string;
}

/** The default single-line set (┌─┐│└┘). */
export declare function borderNormal(): BorderSet;

/** The rounded-corner set (╭─╮│╰╯). */
export declare function borderRounded(): BorderSet;

/** The heavy set (┏━┓┃┗┛). */
export declare function borderThick(): BorderSet;

/** The double-line set (╔═╗║╚╝). */
export declare function borderDouble(): BorderSet;

/** Return the given set when non-empty, else borderNormal(). */
export declare function borderSetOrDefault(set: BorderSet | null | undefined): BorderSet;

export interface BorderBoxOptions {
  id?: string;
  title?: string;
  width?: number;
  height?: number;
  border?: BorderSet | null;
  style?: string;
  titleStyle?: string;
  child?: Node | null;
}

/** A bordered box around an optional child, drawn program-side as styled text
 * rows. A zero geometry falls back to the child's measured content; boxes too
 * small for chrome degrade to the bare child. */
export declare class BorderBox {
  constructor(options?: BorderBoxOptions);
  id: string;
  title: string;
  width: number;
  height: number;
  border: BorderSet | null;
  style: string;
  titleStyle: string;
  child: Node | null;
  build(): Node;
  /** Top edge row: corner, optional title segment, fill run and corner. */
  topEdge(set: BorderSet, titleStyle: string, inner: number): Node;
  /** Bottom edge text: corner + repeated bottom + corner. */
  bottomEdge(set: BorderSet, inner: number): string;
}
