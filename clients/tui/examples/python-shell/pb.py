"""Compatibility shim: the TUI v2 wire codec moved to the Python SDK.

``clients/tui/sdk/python/tui2sdk/wire.py`` is now the single Python binding of
``clients/tui/proto/tui2.proto``; this module re-exports it so existing layout
programs (``shell.py``, ``legacy.py``) keep working unchanged. New programs
should import ``tui2sdk`` directly:

    from tui2sdk import Client, App, builder
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                "..", "..", "sdk", "python"))

from tui2sdk.wire import *  # noqa: F401,F403,E402
from tui2sdk.wire import (  # noqa: F401
    EVENT, HELLO, MAX_MESSAGE_BYTES, RESPONSE, RESULT, VIEW,
    WireError, decode_event, decode_hello, decode_response,
    encode_box, encode_cursor, encode_hello, encode_limits, encode_result,
    encode_view, field_bool, field_bytes, field_map_string_bool, field_msg,
    field_string, field_varint, frame, parse_fields, read_frame, read_uvarint,
    to_int32, to_int64, uvarint,
)
