/**
 * Form fields with focus, navigation and validation (port of
 * tui2/sdk/widgets/form.go).
 */

import { Node } from '../builder';
import { TextInput, KeyEventLike } from './input';
import { Validator } from './validate';

export declare const FORM_REQUIRED_MESSAGE: string;

export interface FieldOptions {
  id?: string;
  label?: string;
  help?: string;
  error?: string;
  input?: TextInput | null;
  required?: boolean;
  validate?: ((value: string) => string) | null;
  style?: string;
  labelStyle?: string;
  errorStyle?: string;
  helpStyle?: string;
  focused?: boolean;
  disabled?: boolean;
  readonly?: boolean;
}

export declare class Field {
  constructor(options?: FieldOptions);
  id: string;
  label: string;
  help: string;
  error: string;
  input: TextInput | null;
  required: boolean;
  validate: ((value: string) => string) | null;
  style: string;
  labelStyle: string;
  errorStyle: string;
  helpStyle: string;
  focused: boolean;
  disabled: boolean;
  readonly: boolean;
  /** Whether Form navigation may land on the field. */
  selectable(): boolean;
  /** Field value, or "" when it owns no input. */
  text(): string;
  build(): Node;
}

export interface FormOptions {
  id?: string;
  fields?: Field[];
  focus?: number;
  width?: number;
  style?: string;
  errorStyle?: string;
  validateOnChange?: boolean;
}

export declare class Form {
  constructor(options?: FormOptions);
  id: string;
  fields: Field[];
  focus: number;
  width: number;
  style: string;
  errorStyle: string;
  validateOnChange: boolean;
  next(): void;
  prev(): void;
  focusId(): string;
  /** Copy values keyed by field id into the matching inputs. */
  setValues(values: Record<string, string>): void;
  /** Current text of every identified field. */
  values(): Record<string, string>;
  /** The input owned by the field with the given id, or null. */
  input(id: string): TextInput | null;
  /** Run every field's Required check and Validate; stores errors. */
  validate(): boolean;
  handleKey(id: string, ev: KeyEventLike | null | undefined): boolean;
  build(): Node;
}

export type { Validator };
