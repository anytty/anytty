/**
 * Dialogs and keyboard menus (port of tui2/sdk/widgets/modal.go). Build
 * returns a Node tree; nothing is emitted.
 */

import { Node } from '../builder';

export declare const DEFAULT_BACKDROP_STYLE: string;

/** One content row of a Frame/FloatingLayer. */
export interface FrameRow {
  text: string;
  style?: string;
  id?: string;
  input?: string[];
}

export interface KeyEventLike {
  key?: string;
  char?: string;
}

export interface ModalOptions {
  id?: string;
  title?: string;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  rows?: FrameRow[];
  style?: string;
  backdrop?: boolean;
  backdropStyle?: string;
  backdropId?: string;
  center?: boolean;
  parentWidth?: number;
  parentHeight?: number;
}

export declare class Modal {
  constructor(options?: ModalOptions);
  id: string;
  title: string;
  x: number;
  y: number;
  width: number;
  height: number;
  rows: FrameRow[];
  style: string;
  backdrop: boolean;
  backdropStyle: string;
  backdropId: string;
  center: boolean;
  parentWidth: number;
  parentHeight: number;
  /** Floating layer origin: (x, y), or the centered origin when possible. */
  position(): [number, number];
  build(): Node;
}

/** One Menu row. A separator draws a rule; a disabled item is skipped by
 * move/hotkey. */
export interface MenuItem {
  id?: string;
  label?: string;
  hotkey?: string;
  disabled?: boolean;
  separator?: boolean;
  style?: string;
}

/** The item identity emitted by Menu.value. */
export declare function menuItemValue(item: MenuItem): string;

export interface MenuOptions {
  id?: string;
  title?: string;
  items?: MenuItem[];
  selected?: number;
  width?: number;
  x?: number;
  y?: number;
  style?: string;
  selectedStyle?: string;
  disabledStyle?: string;
  separatorStyle?: string;
  frameStyle?: string;
  marker?: string;
}

export declare class Menu {
  constructor(options?: MenuOptions);
  id: string;
  title: string;
  items: MenuItem[];
  selected: number;
  width: number;
  x: number;
  y: number;
  style: string;
  selectedStyle: string;
  disabledStyle: string;
  separatorStyle: string;
  frameStyle: string;
  marker: string;
  /** Shift the selection by delta selectable items; stops at the boundaries. */
  move(delta: number): void;
  /** Move the selection to index when it is selectable. */
  select(index: number): boolean;
  /** Match one key event against the item hotkeys and select the item. */
  hotkey(ev: KeyEventLike | null | undefined): { value: string; ok: boolean };
  /** Selected item value, empty when nothing selectable is selected. */
  value(): string;
  selectedItem(): { item: MenuItem | null; ok: boolean };
  build(): Node;
}
