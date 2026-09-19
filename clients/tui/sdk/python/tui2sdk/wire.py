"""TUI v2 wire codec (stdlib only): protobuf payloads + binary frames.

This is the Python binding of ``proto/ui/tui2.proto`` (the single source of
truth) for the subset a layout program needs:

  decode: HELLO, EVENT (key/paste/mouse/wheel/resize/sources/notice/
          component/view_rejected), RESPONSE
  encode: VIEW (Box tree, Keys), RESULT (MethodParams)

Wire basics: every field is ``varint(tag) | payload`` with
``tag = (field << 3) | wire_type``; 0 = varint, 1 = 64-bit, 2 =
length-delimited, 5 = 32-bit. A frame is
``u32 length (big endian) | u8 type | payload`` where length counts the type
byte plus the payload (PROTOCOL §0).

The codec deliberately depends on nothing outside the standard library: any
language can be a layout program, this module is only the Python convenience
binding.
"""

import struct

# Frame types (PROTOCOL §0): fixed, append-only.
HELLO = 1
VIEW = 2
EVENT = 3
RESULT = 4
RESPONSE = 5

WIRE_VARINT = 0
WIRE_64BIT = 1
WIRE_LEN = 2
WIRE_32BIT = 5

MAX_MESSAGE_BYTES = 1 << 20


class WireError(Exception):
    """Raised for malformed frames, payloads or protocol misuse."""


# ----------------------------------------------------------------- encoding

def uvarint(value):
    """Encode an int as an unsigned varint. Negative int32/int64 values are
    sign-extended to 64 bits, exactly like protobuf does."""
    if value < 0:
        value += 1 << 64
    out = bytearray()
    while True:
        byte = value & 0x7F
        value >>= 7
        if value:
            out.append(byte | 0x80)
        else:
            out.append(byte)
            return bytes(out)


def field_varint(field, value):
    if value is None:
        return b""
    return uvarint(field << 3) + uvarint(int(value))


def field_bool(field, value):
    if value is None:
        return b""
    return field_varint(field, 1 if value else 0)


def field_bytes(field, data):
    if not data:
        return b""
    return uvarint((field << 3) | WIRE_LEN) + uvarint(len(data)) + data


def field_string(field, text):
    if not text:
        return b""
    return field_bytes(field, text.encode("utf-8"))


def field_msg(field, message):
    if message is None:
        return b""
    return field_bytes(field, message)


def field_map_string_string(field, mapping):
    out = b""
    for key in sorted(mapping):
        entry = field_string(1, key) + field_string(2, mapping[key])
        out += field_msg(field, entry)
    return out


# ----------------------------------------------------------------- decoding

def read_uvarint(buf, index):
    result = 0
    shift = 0
    while True:
        byte = buf[index]
        index += 1
        result |= (byte & 0x7F) << shift
        if not byte & 0x80:
            return result, index
        shift += 7


def parse_fields(data):
    """Parse one message body into [(field, wire, value), ...] preserving
    order. Varint values are ints, length-delimited values are bytes."""
    out = []
    index = 0
    size = len(data)
    while index < size:
        tag, index = read_uvarint(data, index)
        field, wire = tag >> 3, tag & 7
        if wire == WIRE_VARINT:
            value, index = read_uvarint(data, index)
        elif wire == WIRE_LEN:
            length, index = read_uvarint(data, index)
            value = data[index:index + length]
            index += length
        elif wire == WIRE_64BIT:
            value = data[index:index + 8]
            index += 8
        elif wire == WIRE_32BIT:
            value = data[index:index + 4]
            index += 4
        else:
            raise WireError("unsupported wire type %d" % wire)
        out.append((field, wire, value))
    return out


def to_int32(value):
    return value - (1 << 32) if value >= 1 << 31 else value


def to_int64(value):
    return value - (1 << 64) if value >= 1 << 63 else value


def _first(fields, field):
    for number, _wire, value in fields:
        if number == field:
            return value
    return None


def _strings(fields, field):
    return [value.decode("utf-8") for number, _wire, value in fields if number == field]


def _string(fields, field):
    value = _first(fields, field)
    return value.decode("utf-8") if value is not None else ""


def _uint(fields, field):
    value = _first(fields, field)
    return value if value is not None else 0


def _bool(fields, field):
    return _uint(fields, field) != 0


def parse_map_string_bool(entry):
    fields = parse_fields(entry)
    return {_string(fields, 1): _bool(fields, 2)}


def parse_limits(data):
    fields = parse_fields(data)
    return {
        "max_nodes": _uint(fields, 1),
        "max_message_bytes": _uint(fields, 2),
        "max_paste_bytes": _uint(fields, 3),
        "max_inflight_requests": _uint(fields, 4),
        "owner_lease_ttl_ms": _uint(fields, 5),
    }


def decode_hello(data):
    fields = parse_fields(data)
    hello = {
        "schema": _uint(fields, 1),
        "view_id": _string(fields, 2),
        "epoch": _uint(fields, 3),
        "cols": _uint(fields, 4),
        "rows": _uint(fields, 5),
        "components": _strings(fields, 6),
        "events": _strings(fields, 7),
        "methods": _strings(fields, 8),
        "features": {},
        "limits": {},
    }
    for field, wire, value in fields:
        if field == 9 and wire == WIRE_LEN:
            hello["features"].update(parse_map_string_bool(value))
        elif field == 10 and wire == WIRE_LEN:
            hello["limits"] = parse_limits(value)
    return hello


def encode_hello(hello):
    out = field_varint(1, hello.get("schema"))
    out += field_string(2, hello.get("view_id"))
    out += field_varint(3, hello.get("epoch"))
    out += field_varint(4, hello.get("cols"))
    out += field_varint(5, hello.get("rows"))
    for name in hello.get("components", ()):
        out += field_string(6, name)
    for name in hello.get("events", ()):
        out += field_string(7, name)
    for name in hello.get("methods", ()):
        out += field_string(8, name)
    if hello.get("features"):
        out += field_map_string_bool(9, hello["features"])
    if hello.get("limits"):
        out += field_msg(10, encode_limits(hello["limits"]))
    return out


def encode_limits(limits):
    out = field_varint(1, limits.get("max_nodes"))
    out += field_varint(2, limits.get("max_message_bytes"))
    out += field_varint(3, limits.get("max_paste_bytes"))
    out += field_varint(4, limits.get("max_inflight_requests"))
    out += field_varint(5, limits.get("owner_lease_ttl_ms"))
    return out


def field_map_string_bool(field, mapping):
    out = b""
    for key in sorted(mapping):
        entry = field_string(1, key) + field_bool(2, mapping[key])
        out += field_msg(field, entry)
    return out


def parse_source(data):
    fields = parse_fields(data)
    return {
        "id": _string(fields, 1),
        "kind": _string(fields, 2),
        "title": _string(fields, 3),
        "endpoint": _string(fields, 4),
        "terminal_id": _string(fields, 5),
        "attached": _bool(fields, 6),
        "exited": _bool(fields, 7),
        "exit_code": to_int32(_uint(fields, 8)),
        "health": _string(fields, 9),
        "resize_owner": _string(fields, 10),
        "owner_epoch": _uint(fields, 11),
        "last_seen_ms": to_int64(_uint(fields, 12)),
    }


def decode_event(data):
    """Decode one EVENT payload into {"kind": ..., plus the payload keys}."""
    for field, wire, value in parse_fields(data):
        if wire != WIRE_LEN:
            continue
        inner = parse_fields(value)
        if field == 1:  # key
            return {"kind": "key", "id": _string(inner, 1),
                    "key": _string(inner, 2), "char": _string(inner, 3)}
        if field == 2:  # paste
            return {"kind": "paste", "id": _string(inner, 1), "text": _string(inner, 2)}
        if field == 3:  # mouse
            return {"kind": "mouse", "action": _string(inner, 1), "button": _string(inner, 2),
                    "x": to_int32(_uint(inner, 3)), "y": to_int32(_uint(inner, 4)),
                    "node": _string(inner, 5)}
        if field == 4:  # wheel
            return {"kind": "wheel", "delta": to_int32(_uint(inner, 1)),
                    "x": to_int32(_uint(inner, 2)), "y": to_int32(_uint(inner, 3)),
                    "node": _string(inner, 4)}
        if field == 5:  # resize
            return {"kind": "resize", "cols": _uint(inner, 1), "rows": _uint(inner, 2)}
        if field == 6:  # sources
            items = []
            for number, item_wire, item in inner:
                if number == 1 and item_wire == WIRE_LEN:
                    items.append(parse_source(item))
            return {"kind": "sources", "items": items}
        if field == 7:  # notice
            return {"kind": "notice", "level": _string(inner, 1), "message": _string(inner, 2)}
        if field == 8:  # component
            return {"kind": "component", "source": _string(inner, 1),
                    "name": _string(inner, 2), "value": _string(inner, 3)}
        if field == 9:  # view_rejected
            return {"kind": "view_rejected", "epoch": _uint(inner, 1),
                    "rev": _uint(inner, 2), "reason": _string(inner, 3)}
    return {"kind": "unknown"}


def decode_response(data):
    fields = parse_fields(data)
    response = {
        "request_id": _uint(fields, 1),
        "epoch": _uint(fields, 2),
        "ok": _bool(fields, 3),
        "data": {},
        "error": _string(fields, 5),
    }
    for field, wire, value in fields:
        if field == 4 and wire == WIRE_LEN:
            inner = parse_fields(value)
            response["data"] = {
                "rows": _strings(inner, 1),
                "text": _string(inner, 2),
                "endpoint": _string(inner, 3),
                "id": _string(inner, 4),
            }
    return response


# ---------------------------------------------------------------- box trees

def encode_cursor(cursor):
    """Encode a program cursor dict {row,col,shape?,visible?} (Box field 7).

    ``visible`` is an optional proto3 field: it is only written when the
    program sets it, so absence keeps the kernel default (visible).
    """
    row, col = cursor.get("row", 0), cursor.get("col", 0)
    out = field_varint(1, row) + field_varint(2, col)
    out += field_string(3, cursor.get("shape"))
    if cursor.get("visible") is not None:
        out += field_bool(4, cursor["visible"])
    return out


def encode_box(box):
    """Encode one view-tree node dict (keys mirror tui2.proto Box)."""
    out = field_string(1, box.get("id"))
    if box.get("size"):
        width, height, flex = box["size"]
        out += field_msg(2, field_varint(1, width) + field_varint(2, height) + field_varint(3, flex))
    if box.get("pos") is not None:
        x, y = box["pos"]
        out += field_msg(3, field_varint(1, x) + field_varint(2, y))
    out += field_string(4, box.get("flow"))
    if box.get("visible") is not None:
        out += field_bool(5, box["visible"])
    if box.get("cursor") is not None:
        out += field_msg(7, encode_cursor(box["cursor"]))
    content = box.get("content")
    if content:
        encoded = field_string(1, content.get("text"))
        for line in content.get("lines", ()):
            encoded += field_string(2, line)
        encoded += field_string(3, content.get("self"))
        if content.get("props"):
            encoded += field_map_string_string(4, content["props"])
        out += field_msg(8, encoded)
    for kind in box.get("input", ()):
        out += field_string(9, kind)
    out += field_bool(10, box.get("focused"))
    for child in box.get("children", ()):
        out += field_msg(11, encode_box(child))
    out += field_string(12, box.get("style"))
    return out


def encode_view(epoch, rev, claim, all_keys, root):
    keys = b""
    for key in claim:
        keys += field_string(1, key)
    keys += field_bool(2, all_keys)
    return (field_varint(1, epoch) + field_varint(2, rev) +
            field_msg(3, keys) + field_msg(4, encode_box(root)))


# ------------------------------------------------------------------- result

def encode_result(request_id, epoch, method, params=None):
    params = params or {}
    encoded = field_string(1, params.get("endpoint"))
    encoded += field_string(2, params.get("id"))
    if params.get("fit") is not None:
        encoded += field_bool(3, params["fit"])
    if params.get("expected_owner_epoch") is not None:
        encoded += field_varint(4, params["expected_owner_epoch"])
    for arg in params.get("argv", ()):
        encoded += field_string(5, arg)
    encoded += field_string(6, params.get("cwd"))
    if params.get("env"):
        encoded += field_map_string_string(7, params["env"])
    encoded += field_string(8, params.get("title"))
    if params.get("ephemeral") is not None:
        encoded += field_bool(9, params["ephemeral"])
    if params.get("delta") is not None:
        encoded += field_varint(10, params["delta"])
    if params.get("sel") is not None:
        mode, start, end = params["sel"]
        encoded += field_msg(11, field_string(1, mode) + field_varint(2, start) + field_varint(3, end))
    if params.get("offset") is not None:
        encoded += field_varint(12, params["offset"])
    if params.get("rows") is not None:
        encoded += field_varint(13, params["rows"])
    encoded += field_string(14, params.get("event_id"))
    encoded += field_string(15, params.get("source"))
    if params.get("cleanup_owned") is not None:
        encoded += field_bool(16, params["cleanup_owned"])
    # Endpoint connection metadata (tui2.proto MethodParams fields 17..20):
    # kind "command"/"daemon", socket path, connect_mode and the optional
    # tcp address (HOST:PORT).
    encoded += field_string(17, params.get("kind"))
    encoded += field_string(18, params.get("socket"))
    encoded += field_string(19, params.get("connect_mode"))
    encoded += field_string(20, params.get("address"))
    return field_varint(1, request_id) + field_varint(2, epoch) + \
        field_string(3, method) + field_msg(4, encoded)


# -------------------------------------------------------------------- frame

def frame(frame_type, payload):
    """One envelope: u32 length (type + payload, big endian) | u8 type | body."""
    body = bytes([frame_type]) + payload
    return struct.pack(">I", len(body)) + body


def read_frame(stream):
    """Read one frame, returning (type, payload) or None on clean EOF."""
    prefix = stream.read(4)
    if not prefix:
        return None
    if len(prefix) < 4:
        raise WireError("truncated frame length")
    (length,) = struct.unpack(">I", prefix)
    if length == 0:
        raise WireError("zero-length frame")
    if length > MAX_MESSAGE_BYTES:
        raise WireError("frame exceeds max_message_bytes")
    body = stream.read(length)
    if len(body) < length:
        raise WireError("truncated frame")
    return body[0], body[1:]
