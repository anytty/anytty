"""Pure hover tracking over hit regions (port of hover.go)."""

from .list import first_non_empty
from .mouse import hit


class Hover:
    """A pure hover tracker over a set of :class:`HitRegion` values.
    ``update`` resolves the region under (x, y) and returns its id (empty when
    none); ``style_for`` returns the style the hovered id should render with.
    A region id that equals ``node`` means "the whole hover surface is one
    target"."""

    def __init__(self, node="", regions=None, style="", current_style=""):
        self.node = node
        self.regions = list(regions or [])
        self.style = style
        self.current_style = current_style

    def update(self, x, y):
        """Hit-test (x, y) and return the hovered region id, or "" when the
        pointer is over no region."""
        region, ok = hit(self.regions, x, y)
        if not ok:
            self.current_style = ""
            return ""
        self.current_style = first_non_empty(region.id, self.node)
        return self.current_style

    def style_for(self, id):
        """The hover accent when id is the hovered target and "" otherwise.
        An empty id means nothing is hovered."""
        if id and id == self.current_style:
            return self.style
        return ""

    def hovered(self):
        """The currently hovered id."""
        return self.current_style

    def clear(self):
        """Forget the hovered target."""
        self.current_style = ""
