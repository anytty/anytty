'use strict';

// A menu anchored at a click point, clamped inside a parent rect. Port of
// tui2/sdk/widgets/contextmenu.go.

const { box } = require('../builder');
const { FloatingLayer } = require('./layout');
const { Menu } = require('./modal');

class ContextMenu extends Menu {
  constructor(options = {}) {
    super(options.menu || options);
    this.x = options.x || 0;
    this.y = options.y || 0;
    this.visible = Boolean(options.visible);
    this.anchorX = options.anchorX || 0;
    this.anchorY = options.anchorY || 0;
    this.parentWidth = options.parentWidth || 0;
    this.parentHeight = options.parentHeight || 0;
    this.margin = options.margin || 0;
  }

  // open places the menu at the click point (x, y), clamped so it fits inside
  // the parent rect, marks it visible and returns the resolved top-left.
  open(x, y) {
    this.x = x;
    this.y = y;
    this.visible = true;
    [this.anchorX, this.anchorY] = this.place();
    return [this.anchorX, this.anchorY];
  }

  // close hides the menu.
  close() {
    this.visible = false;
  }

  // menuWidth returns the declared menu width (0 when unset).
  menuWidth() {
    return this.width;
  }

  // menuHeight returns the framed overlay height for the current items.
  menuHeight() {
    const height = this.items.length + 2;
    return height < 2 ? 2 : height;
  }

  // place clamps (X, Y) into the parent rect, keeping at least Margin cells of
  // breathing room and never going negative.
  place() {
    let margin = this.margin;
    if (margin < 0) margin = 0;
    let x = this.x;
    let y = this.y;
    const width = this.width;
    const height = this.menuHeight();
    if (this.parentWidth > 0) {
      if (x + width > this.parentWidth - margin) x = this.parentWidth - margin - width;
    }
    if (this.parentHeight > 0) {
      if (y + height > this.parentHeight - margin) y = this.parentHeight - margin - height;
    }
    if (x < margin) x = margin;
    if (y < margin) y = margin;
    return [x, y];
  }

  // build returns the positioned overlay. A hidden menu renders an invisible
  // box, so callers can drop it into the view unconditionally.
  build() {
    if (!this.visible) return box().visible(false);
    [this.anchorX, this.anchorY] = this.place();
    const layer = new FloatingLayer({
      id: this.id,
      title: this.title,
      x: this.anchorX,
      y: this.anchorY,
      width: this.width,
      style: this.frameStyle,
      rows: this.menuRows(),
    });
    return layer.build();
  }
}

module.exports = {
  ContextMenu,
};
