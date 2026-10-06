/**
 * Design-token themes and themed widget constructors (port of
 * tui2/sdk/widgets/theme.go). Every field is an opaque style string (a host
 * token name or an explicit "fg:#RRGGBB;bg:#RRGGBB;bold" style), so widgets
 * stay palette-agnostic and the host remains the single ANSI resolver.
 */

import { TextArea, TextInput } from './input';
import { List, VirtualList } from './list';
import { Menu, Modal } from './modal';
import { Badge, ProgressBar, Spinner, Tags } from './progress';
import { Table } from './table';

export interface ThemeOptions {
  /** Palette identity ("dark" or "light"); informational. */
  name?: string;
  text?: string;
  title?: string;
  accent?: string;
  muted?: string;
  success?: string;
  warning?: string;
  danger?: string;
  border?: string;
  borderFocus?: string;
  selection?: string;
  selectionText?: string;
  input?: string;
  placeholder?: string;
  cursor?: string;
  statusBar?: string;
  tabActive?: string;
  tabInactive?: string;
  header?: string;
  footer?: string;
  toast?: string;
  overlay?: string;
  backdrop?: string;
  marker?: string;
  separator?: string;
  zebra?: string;
}

/** The widget design-token set. */
export declare class Theme {
  constructor(options?: ThemeOptions);
  name: string;
  text: string;
  title: string;
  accent: string;
  muted: string;
  success: string;
  warning: string;
  danger: string;
  border: string;
  borderFocus: string;
  selection: string;
  selectionText: string;
  input: string;
  placeholder: string;
  cursor: string;
  statusBar: string;
  tabActive: string;
  tabInactive: string;
  header: string;
  footer: string;
  toast: string;
  overlay: string;
  backdrop: string;
  marker: string;
  separator: string;
  zebra: string;
}

/** The token-based default; tokens resolve against the host palette. */
export declare function darkTheme(): Theme;

/** An explicit light palette for programs that must not depend on the host
 * token table. */
export declare function lightTheme(): Theme;

/** The current package default theme. */
export declare function defaultTheme(): Theme;

/** Replace the package default theme; returns the new default. */
export declare function setDefaultTheme(theme: Theme): Theme;

/** Resolve "dark"/"light" (empty means dark); the flag is false for unknown
 * names, which yield a zero Theme. */
export declare function themeByName(name: string): [Theme, boolean];

export declare function themedList(theme: Theme): List;

export declare function themedVirtualList(theme: Theme): VirtualList;

export declare function themedTable(theme: Theme): Table;

export declare function themedTextInput(theme: Theme): TextInput;

export declare function themedTextArea(theme: Theme): TextArea;

export declare function themedModal(theme: Theme): Modal;

export declare function themedMenu(theme: Theme): Menu;

export declare function themedProgressBar(theme: Theme): ProgressBar;

export declare function themedSpinner(theme: Theme): Spinner;

export declare function themedBadge(theme: Theme): Badge;

export declare function themedTags(theme: Theme): Tags;
