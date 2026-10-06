/**
 * Validators (port of tui2/sdk/widgets/validate.go). A validator maps a value
 * to an error message; "" means acceptable.
 */

export type Validator = (value: string) => string;

/** Reject a value that is empty after trimming whitespace. */
export declare function required(msg: string): Validator;

/** Reject a value shorter than n runes; empty values pass. */
export declare function minLen(n: number, msg: string): Validator;

/** Reject a value longer than n runes; empty values pass. */
export declare function maxLen(n: number, msg: string): Validator;

/** Reject a value that does not match the regular expression re. A broken
 * pattern fails closed for non-empty values. */
export declare function pattern(re: string, msg: string): Validator;

/** Reject values that are not a plausible email address. */
export declare function email(msg: string): Validator;

/** Reject values that are not base-10 integers inside [min, max]. */
export declare function intRange(min: number, max: number, msg: string): Validator;

/** Reject values absent from values. */
export declare function oneOf(values: string[], msg: string): Validator;

/** Run validators in order and return the first non-empty message. */
export declare function all(...validators: Array<Validator | null | undefined>): Validator;

/** Adapt an arbitrary function into a validator; null is a no-op. */
export declare function custom(fn: ((value: string) => string) | null | undefined): Validator;
