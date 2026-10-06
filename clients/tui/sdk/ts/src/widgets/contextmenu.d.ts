/**
 * A menu anchored at a click point (port of tui2/sdk/widgets/contextmenu.go).
 */

import { Node } from '../builder';
import { Menu, MenuItem } from './modal';

export interface ContextMenuOptions {
  /** The embedded menu, or inline menu fields when absent. */
  menu?: Menu;
  id?: string;
  title?: string;
  items?: MenuItem[];
  selected?: number;
  width?: number;
  style?: string;
  selectedStyle?: string;
  disabledStyle?: string;
  separatorStyle?: string;
  frameStyle?: string;
  marker?: string;
  /** Requested (click) anchor. */
  x?: number;
  y?: number;
  visible?: boolean;
  /** Resolved clamped top-left. */
  anchorX?: number;
  anchorY?: number;
  parentWidth?: number;
  parentHeight?: number;
  margin?: number;
}

export declare class ContextMenu extends Menu {
  constructor(options?: ContextMenuOptions);
  x: number;
  y: number;
  visible: boolean;
  anchorX: number;
  anchorY: number;
  parentWidth: number;
  parentHeight: number;
  margin: number;
  /** Place the menu at (x, y), clamped inside the parent; returns the anchor. */
  open(x: number, y: number): [number, number];
  close(): void;
  menuWidth(): number;
  menuHeight(): number;
  build(): Node;
}
