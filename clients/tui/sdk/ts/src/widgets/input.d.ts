/**
 * Single-line and multi-line text editors (port of tui2/sdk/widgets/input.go).
 * Values are arrays of runes (one code point per element) so cursor indexes
 * match Go's []rune semantics.
 */

import { Node } from '../builder';

export declare const DEFAULT_CURSOR_STYLE: string;
export declare const DEFAULT_CURSOR_SHAPE: string;

/** The key-event shape the editors consume. */
export interface KeyEventLike {
  key?: string;
  char?: string;
}

export interface TextInputOptions {
  id?: string;
  value?: string | string[];
  cursor?: number;
  placeholder?: string;
  mask?: string;
  maxLen?: number;
  width?: number;
  offset?: number;
  focused?: boolean;
  style?: string;
  placeholderStyle?: string;
  cursorStyle?: string;
  cursorShape?: string;
  input?: string[];
}

export declare class TextInput {
  constructor(options?: TextInputOptions);
  id: string;
  value: string[];
  cursor: number;
  placeholder: string;
  mask: string;
  maxLen: number;
  width: number;
  offset: number;
  focused: boolean;
  style: string;
  placeholderStyle: string;
  cursorStyle: string;
  cursorShape: string;
  input: string[];
  /** Raw value (mask only affects rendering). */
  text(): string;
  /** Rendered text: masked value, or the placeholder when empty. */
  displayText(): string;
  setValue(s: string): void;
  insertRune(r: string): boolean;
  insertString(s: string): boolean;
  backspace(): boolean;
  delete(): boolean;
  deleteWordLeft(): boolean;
  deleteToEnd(): boolean;
  left(): boolean;
  right(): boolean;
  wordLeft(): boolean;
  wordRight(): boolean;
  home(): boolean;
  end(): boolean;
  handleKey(ev: KeyEventLike | null | undefined): boolean;
  build(): Node;
}

export interface TextAreaOptions {
  id?: string;
  value?: string | string[];
  cursor?: number;
  placeholder?: string;
  mask?: string;
  maxLen?: number;
  width?: number;
  height?: number;
  rowOffset?: number;
  colOffset?: number;
  focused?: boolean;
  style?: string;
  placeholderStyle?: string;
  cursorStyle?: string;
  cursorShape?: string;
  input?: string[];
}

export declare class TextArea {
  constructor(options?: TextAreaOptions);
  id: string;
  value: string[];
  cursor: number;
  placeholder: string;
  mask: string;
  maxLen: number;
  width: number;
  height: number;
  rowOffset: number;
  colOffset: number;
  focused: boolean;
  style: string;
  placeholderStyle: string;
  cursorStyle: string;
  cursorShape: string;
  input: string[];
  text(): string;
  lines(): string[];
  line(row: number): string;
  /** Cursor line and its rune column. */
  rowCol(): [number, number];
  setValue(s: string): void;
  insertRune(r: string): boolean;
  insertString(s: string): boolean;
  backspace(): boolean;
  delete(): boolean;
  deleteWordLeft(): boolean;
  left(): boolean;
  right(): boolean;
  wordLeft(): boolean;
  wordRight(): boolean;
  up(): boolean;
  down(): boolean;
  home(): boolean;
  end(): boolean;
  pageUp(): boolean;
  pageDown(): boolean;
  handleKey(ev: KeyEventLike | null | undefined): boolean;
  build(): Node;
}
