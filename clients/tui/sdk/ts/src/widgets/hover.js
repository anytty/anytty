'use strict';

// Pure hover tracker over a set of HitRegions. Port of
// tui2/sdk/widgets/hover.go.

const { hit } = require('./mouse');

class Hover {
  constructor(options = {}) {
    this.node = options.node || '';
    this.regions = options.regions ? options.regions.slice() : [];
    this.style = options.style || '';
    this.currentStyle = options.currentStyle || '';
  }

  // update hit-tests (x, y) and returns the hovered region id, or "" when the
  // pointer is over no region.
  update(x, y) {
    const region = hit(this.regions, x, y);
    if (region == null) {
      this.currentStyle = '';
      return '';
    }
    this.currentStyle = region.id !== '' ? region.id : this.node;
    return this.currentStyle;
  }

  // styleFor returns the hover accent when id is the hovered target.
  styleFor(id) {
    if (id !== '' && id === this.currentStyle) return this.style;
    return '';
  }

  hovered() {
    return this.currentStyle;
  }

  clear() {
    this.currentStyle = '';
  }
}

module.exports = {
  Hover,
};
