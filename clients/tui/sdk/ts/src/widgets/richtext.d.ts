/**
 * Rich text: styled runs, wrapping and emphasis parsing (port of
 * tui2/sdk/widgets/richtext.go).
 */

import { Node } from '../builder';

/** One styled run of rich text. */
export interface Span {
  text: string;
  style: string;
}

export interface RichTextOptions {
  spans?: Span[];
  width?: number;
  wrap?: boolean;
}

/** Create a Span of text rendered with style. */
export declare function styled(text: string, style: string): Span;

/** Variadic RichText constructor from spans. */
export declare function line(...spans: Span[]): RichText;

/** Coalesce adjacent runs with the same style and drop empty runs. */
export declare function mergeSpans(spans: Span[]): Span[];

export declare class RichText {
  constructor(options?: RichTextOptions);
  spans: Span[];
  width: number;
  wrap: boolean;
  /** Plain text of the spans, for measurement. */
  text(): string;
  /** Render as a one-row box tree; a declared width pads the row. */
  build(): Node;
  /** Re-style the whole text as a single span. */
  styled(style: string): RichText;
  /** Wrap to width, marking every resulting row as wrapped. */
  wrapSpans(width: number): RichText[] | null;
}

/** Hard-wrap s into lines of at most width display cells; width <= 0 -> null. */
export declare function wrapText(s: string, width: number): string[] | null;

/** Wrap styled runs into lines of at most width display cells; width <= 0 -> null. */
export declare function wrapSpans(spans: Span[], width: number): RichText[] | null;

/** Parse "**bold**" and "_em_" spans from s. */
export declare function parseEmphasis(s: string): Span[];

/** parseEmphasis with a base style composed onto every span. */
export declare function parseEmphasisStyled(s: string, base: string): Span[];

/** Append the italic attribute unless the style already carries it. */
export declare function withItalic(style: string): string;
