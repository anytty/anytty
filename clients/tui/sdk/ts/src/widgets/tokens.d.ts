/**
 * Host style tokens and idempotent raw-style helpers (port of
 * tui2/sdk/widgets/tokens.go). A style field accepts either a token name or an
 * explicit "fg:#RRGGBB;bg:#RRGGBB;bold" string.
 */

export declare const STYLE_DEFAULT: string;
export declare const STYLE_BACKGROUND: string;
export declare const STYLE_FG: string;
export declare const STYLE_FOREGROUND: string;
export declare const STYLE_STRONG_FOREGROUND: string;
export declare const STYLE_MUTED: string;
export declare const STYLE_ACCENT: string;
export declare const STYLE_ACCENT_DIM: string;
export declare const STYLE_SUCCESS: string;
export declare const STYLE_OK: string;
export declare const STYLE_WARNING: string;
export declare const STYLE_DANGER: string;
export declare const STYLE_INFO: string;
export declare const STYLE_CHROME: string;
export declare const STYLE_CHROME_FOCUS: string;
export declare const STYLE_HEADER: string;
export declare const STYLE_TAB_ACTIVE: string;
export declare const STYLE_TAB_INACTIVE: string;
export declare const STYLE_FOOTER: string;
export declare const STYLE_FOOTER_ACCENT: string;
export declare const STYLE_STATUS: string;
export declare const STYLE_OVERLAY: string;
export declare const STYLE_BORDER: string;
export declare const STYLE_BORDER_FOCUS: string;
export declare const STYLE_BORDER_DEAD: string;
export declare const STYLE_ACTIVE_BORDER: string;
export declare const STYLE_INACTIVE_BORDER: string;
export declare const STYLE_SELECTION: string;

/** Whether style already carries segment as a semicolon-separated piece. */
export declare function styleHasSegment(style: string, segment: string): boolean;

/** Append the bold attribute unless the style already carries it. */
export declare function withBold(style: string): string;

/** Append the underline attribute unless the style already carries it. */
export declare function withUnderline(style: string): string;

/** Append the reverse attribute unless the style already carries it. */
export declare function withReverse(style: string): string;

/** Append the italic attribute unless the style already carries it. */
export declare function withItalic(style: string): string;

/** Set the "fg:" color segment; an empty hex leaves the style untouched. */
export declare function withFg(style: string, hex: string): string;

/** Set the "bg:" color segment; an empty hex leaves the style untouched. */
export declare function withBg(style: string, hex: string): string;
