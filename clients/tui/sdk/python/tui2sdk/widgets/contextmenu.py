"""Menu anchored at a click point (port of contextmenu.go).

It embeds :class:`Menu` for its items/selection and adds the overlay
placement: ``open`` records the click and resolves a clamped top-left
(``anchor_x``/``anchor_y``) that keeps the whole menu inside the parent rect,
``close`` hides it and ``build`` emits the positioned floating layer. It stays
a pure widget: the caller decides whether a click dismisses it.
"""

from .. import builder
from .modal import Menu


class ContextMenu(Menu):
    """A menu anchored at a click point. ``x``/``y`` are the requested (click)
    anchor; ``anchor_x``/``anchor_y`` is the resolved clamped top-left."""

    def __init__(self, menu=None, x=0, y=0, visible=False, anchor_x=0,
                 anchor_y=0, parent_width=0, parent_height=0, margin=0,
                 **menu_fields):
        super().__init__(**menu_fields)
        if menu is not None:
            for name in Menu._FIELDS:
                setattr(self, name, getattr(menu, name))
        self.x = x
        self.y = y
        self.visible = visible
        self.anchor_x = anchor_x
        self.anchor_y = anchor_y
        self.parent_width = parent_width
        self.parent_height = parent_height
        self.margin = margin

    def open(self, x, y):
        """Place the menu at the click point (x, y), clamped so it fits inside
        the parent rect, marks it visible and returns the resolved
        top-left."""
        self.x, self.y = x, y
        self.visible = True
        self.anchor_x, self.anchor_y = self._place()
        return self.anchor_x, self.anchor_y

    def close(self):
        """Hide the menu."""
        self.visible = False

    def menu_width(self):
        """The declared menu width (0 when unset)."""
        return self.width

    def menu_height(self):
        """The framed overlay height for the current items: one row per item
        plus a two-cell frame, at least two."""
        height = len(self.items) + 2
        if height < 2:
            height = 2
        return height

    def _place(self):
        """Clamp (x, y) into the parent rect, keeping at least ``margin``
        cells of breathing room and never going negative."""
        margin = self.margin
        if margin < 0:
            margin = 0
        x, y = self.x, self.y
        width, height = self.width, self.menu_height()
        if self.parent_width > 0:
            if x + width > self.parent_width - margin:
                x = self.parent_width - margin - width
        if self.parent_height > 0:
            if y + height > self.parent_height - margin:
                y = self.parent_height - margin - height
        if x < margin:
            x = margin
        if y < margin:
            y = margin
        return x, y

    def build(self):
        """The positioned overlay. A hidden menu renders an invisible box, so
        callers can drop it into the view unconditionally."""
        if not self.visible:
            return builder.box().visible(False)
        self.anchor_x, self.anchor_y = self._place()
        return Menu.build(self)

    def _layer_pos(self):
        return self.anchor_x, self.anchor_y
