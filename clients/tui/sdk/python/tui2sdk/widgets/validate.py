"""Pure validators (port of validate.go).

A validator is a callable ``value -> str``: it returns "" when a value is
acceptable, or the message to render under the field. Validators compose
freely and can be driven by a Form or called directly.
"""

import re

_EMAIL_PATTERN = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")
_INT_PATTERN = re.compile(r"[+-]?[0-9]+")


def required(msg):
    """Reject a value that is empty after trimming surrounding whitespace.
    Unlike the other validators it does not skip empty input: it is the one
    validator whose whole job is to inspect emptiness."""
    def validate(value):
        if value.strip() == "":
            return msg
        return ""
    return validate


def min_len(n, msg):
    """Reject a value shorter than n runes. An empty value is left to
    ``required``, so ``min_len`` alone accepts "". n <= 0 disables the
    check."""
    def validate(value):
        if value == "" or n <= 0:
            return ""
        if len(value) < n:
            return msg
        return ""
    return validate


def max_len(n, msg):
    """Reject a value longer than n runes. An empty value passes and n <= 0
    disables the check."""
    def validate(value):
        if value == "" or n <= 0:
            return ""
        if len(value) > n:
            return msg
        return ""
    return validate


def pattern(regex, msg):
    """Reject a value that does not match the regular expression ``regex``. A
    pattern that fails to compile fails closed: every non-empty value is
    rejected with msg. Empty values pass so pattern composes with
    ``required``."""
    try:
        compiled = re.compile(regex)
    except re.error:
        def broken(value):
            if value == "":
                return ""
            return msg
        return broken

    def validate(value):
        if value == "":
            return ""
        if compiled.search(value) is None:
            return msg
        return ""
    return validate


def email(msg):
    """Reject values that are not a plausible email address. Empty values pass
    so email composes with ``required``."""
    def validate(value):
        if value == "":
            return ""
        if _EMAIL_PATTERN.search(value) is None:
            return msg
        return ""
    return validate


def int_range(lo, hi, msg):
    """Reject a value that is not a base-10 integer inside [lo, hi]. Empty
    (or whitespace-only) values pass so int_range composes with
    ``required``."""
    def validate(value):
        trimmed = value.strip()
        if trimmed == "":
            return ""
        if _INT_PATTERN.fullmatch(trimmed) is None:
            return msg
        n = int(trimmed)
        if n < lo or n > hi:
            return msg
        return ""
    return validate


def one_of(values, msg):
    """Reject a value absent from ``values``. Empty values pass so one_of
    composes with ``required``."""
    allowed = list(values)

    def validate(value):
        if value == "":
            return ""
        if value not in allowed:
            return msg
        return ""
    return validate


def all_of(*validators):
    """Run validators in order and return the first non-empty message. A
    ``None`` entry is skipped. No validators means always valid."""
    def validate(value):
        for validate_one in validators:
            if validate_one is None:
                continue
            msg = validate_one(value)
            if msg != "":
                return msg
        return ""
    return validate


def custom(fn):
    """Adapt an arbitrary function into a validator. ``None`` is a no-op."""
    if fn is None:
        def validate(value):
            return ""
        return validate
    return fn
