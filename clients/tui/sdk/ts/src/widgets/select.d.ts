/**
 * Dropdown selection (port of tui2/sdk/widgets/select.go).
 */

import { Node } from '../builder';

export declare const DEFAULT_SELECT_INDICATOR: string;

/** One Select choice. Disabled options are skipped by move/typeahead. */
export interface Option {
  value: string;
  label?: string;
  disabled?: boolean;
  style?: string;
}

/** Option label, falling back to its value. */
export declare function optionDisplay(option: Option): string;

export interface SelectOptions {
  id?: string;
  label?: string;
  options?: Option[];
  value?: string;
  open?: boolean;
  width?: number;
  placeholder?: string;
  style?: string;
  labelStyle?: string;
  placeholderStyle?: string;
  selectedStyle?: string;
  disabledStyle?: string;
  marker?: string;
  indicator?: string;
  dropdownStyle?: string;
}

export declare class Select {
  constructor(options?: SelectOptions);
  id: string;
  label: string;
  options: Option[];
  value: string;
  open: boolean;
  width: number;
  placeholder: string;
  style: string;
  labelStyle: string;
  placeholderStyle: string;
  selectedStyle: string;
  disabledStyle: string;
  marker: string;
  indicator: string;
  dropdownStyle: string;
  /** Position of the selected option, or -1. */
  index(): number;
  /** Selected option; a disabled selection reports ok=false. */
  selected(): { option: Option | null; ok: boolean };
  /** Select the option at index when it is in range and enabled. */
  selectIndex(index: number): boolean;
  /** Shift the selection by delta enabled options. */
  move(delta: number): void;
  /** Select the next enabled option whose label/value starts with prefix. */
  typeahead(prefix: string): boolean;
  build(): Node;
  /** Open list of options as a column, ready for a FloatingLayer/Modal. */
  buildDropdown(): Node;
}
