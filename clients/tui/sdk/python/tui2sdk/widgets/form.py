"""Form fields with focus navigation and aggregate validation (port of
form.go).

The caller owns the values and drives keys through ``handle_key``; the widget
never emits frames and holds no timers.
"""

import copy

from .. import builder

# Fallback error stored for a required field that is empty and has no
# ``validate`` of its own.
FORM_REQUIRED_MESSAGE = "required"


class Field:
    """One Form row: an optional label, an owned TextInput, and a help or
    error line rendered underneath. Disabled and readonly fields are skipped
    by Form navigation; readonly still shows its value and can be
    validated."""

    def __init__(self, id="", label="", help="", error="", input=None,
                 required=False, validate=None, style="", label_style="",
                 error_style="", help_style="", focused=False, disabled=False,
                 readonly=False):
        self.id = id
        self.label = label
        self.help = help
        self.error = error
        self.input = input
        self.required = required
        self.validate = validate
        self.style = style
        self.label_style = label_style
        self.error_style = error_style
        self.help_style = help_style
        self.focused = focused
        self.disabled = disabled
        self.readonly = readonly

    def selectable(self):
        """Whether Form navigation may land on the field."""
        return not self.disabled and not self.readonly

    def text(self):
        """The field value, or "" when it owns no input."""
        if self.input is None:
            return ""
        return self.input.text()

    def build(self):
        """The field as a column: label, input, then the error line when set,
        otherwise the help line."""
        return _form_field_box(self, 0)


def _form_field_box(field, width):
    col = builder.box("col")
    if width > 0:
        col.width(width)
    if field.style:
        col.style(field.style)
    if field.label:
        label = field.label
        if field.required:
            label += " *"
        col.child(builder.text(label).style(field.label_style).height(1))
    if field.input is not None:
        inp = copy.copy(field.input)
        inp.focused = field.focused
        if inp.width <= 0 and width > 0:
            inp.width = width
        col.child(inp.build())
    if field.error != "":
        col.child(builder.text(field.error).style(field.error_style).height(1))
    elif field.help != "":
        col.child(builder.text(field.help).style(field.help_style).height(1))
    return col


class Form:
    """A column of Fields with keyboard focus, navigation and aggregate
    validation."""

    def __init__(self, id="", fields=None, focus=0, width=0, style="",
                 error_style="", validate_on_change=False):
        self.id = id
        self.fields = list(fields or [])
        self.focus = focus
        self.width = width
        self.style = style
        self.error_style = error_style
        self.validate_on_change = validate_on_change

    def next(self):
        """Move focus to the next selectable field, wrapping around."""
        self._move_focus(1)

    def prev(self):
        """Move focus to the previous selectable field, wrapping around."""
        self._move_focus(-1)

    def _move_focus(self, step):
        count = len(self.fields)
        if count == 0:
            self.focus = 0
            return
        self.focus = _clamp_index(self.focus, count)
        for i in range(1, count + 1):
            nxt = (self.focus + step * i) % count
            if self.fields[nxt].selectable():
                self.focus = nxt
                return

    def focus_id(self):
        """The id of the focused field, or "" when the form has none."""
        if self.focus < 0 or self.focus >= len(self.fields):
            return ""
        return self.fields[self.focus].id

    def set_values(self, values):
        """Copy values keyed by field id into the matching inputs."""
        for field in self.fields:
            if field.input is None:
                continue
            if field.id == "":
                continue
            if field.id in values:
                field.input.set_value(values[field.id])

    def values(self):
        """The current text of every identified field."""
        out = {}
        for field in self.fields:
            if field.id == "" or field.input is None:
                continue
            out[field.id] = field.input.text()
        return out

    def input(self, id):
        """The input owned by the field with the given id so the caller can
        edit it in place, or ``None`` when no field matches."""
        for field in self.fields:
            if field.id == id:
                return field.input
        return None

    def validate(self):
        """Run every field's required check and ``validate``, store each result
        in ``Field.error`` and report whether the whole form is valid."""
        valid = True
        for i in range(len(self.fields)):
            msg = self._field_error(i)
            self.fields[i].error = msg
            if msg != "":
                valid = False
        return valid

    def _field_error(self, index):
        if index < 0 or index >= len(self.fields):
            return ""
        field = self.fields[index]
        value = field.text()
        if field.required and value.strip() == "":
            return FORM_REQUIRED_MESSAGE
        if field.validate is not None:
            return field.validate(value)
        return ""

    def handle_key(self, id, ev):
        """Route one key to the focused field. A non-empty id that is not the
        focused field is ignored. Enter/Tab advance focus and Shift-Tab
        retreats; every other key is delegated to the focused TextInput. With
        ``validate_on_change`` the focused field's error is refreshed after an
        edit."""
        if ev is None:
            return False
        if id != "" and id != self.focus_id():
            return False
        key = ev.get("key")
        if key in ("enter", "tab"):
            self.next()
            return True
        if key == "shift-tab":
            self.prev()
            return True
        if (self.focus < 0 or self.focus >= len(self.fields)
                or self.fields[self.focus].input is None):
            return False
        if not self.fields[self.focus].input.handle_key(ev):
            return False
        if self.validate_on_change:
            self.fields[self.focus].error = self._field_error(self.focus)
        return True

    def build(self):
        """The form as a column of fields, marking the focused one."""
        col = builder.box("col")
        if self.id:
            col.id(self.id)
        if self.width > 0:
            col.width(self.width)
        if self.style:
            col.style(self.style)
        for i, field in enumerate(self.fields):
            field = copy.copy(field)
            field.focused = i == self.focus
            if field.error_style == "":
                field.error_style = self.error_style
            col.child(_form_field_box(field, self.width))
        return col


def _clamp_index(i, count):
    if i < 0 or count <= 0:
        return 0
    if i >= count:
        return count - 1
    return i
