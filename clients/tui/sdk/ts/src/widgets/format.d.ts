/**
 * Numeric/text formatters (port of tui2/sdk/widgets/format.go).
 * Pure functions: no rendering, no state.
 */

/** Format a count of bytes as a base-1024 size: 1536 -> "1.5 KB". */
export declare function formatBytes(n: number): string;

/** Format an integer with "," thousands separators: 1234567 -> "1,234,567". */
export declare function formatCount(n: number): string;

/** Format a duration in nanoseconds: 1500000 -> "1.5ms", 90000000000 -> "1m30s". */
export declare function formatDuration(d: number): string;

/** Format a fraction as a percentage: formatPercent(0.125, 1) === "12.5%". */
export declare function formatPercent(f: number, digits: number): string;

/** Format a float with `digits` decimals, right-aligned in `width` cells. */
export declare function formatFloat(f: number, digits: number, width: number): string;

/** Scale a value to a base-1000 magnitude: 1500 -> [1.5, "K"]. */
export declare function scaleValue(v: number): [number, string];

/** Right-align s in width cells; s is never truncated. */
export declare function padLeft(s: string, width: number): string;

/** Left-align s in width cells; s is never truncated. */
export declare function padRight(s: string, width: number): string;
