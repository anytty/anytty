"""TUI v2 wire codec (stdlib only): protobuf payloads + binary frames.

This is the Python binding of ``proto/ui/tui2.proto`` (the single source of
truth) for the subset a layout program needs:

  decode: HELLO, EVENT (key/paste/mouse/wheel/resize/sources/notice/
          component/view_rejected), RESPONSE, STREAM
  encode: VIEW (Box tree, Keys), VIEW_DELTA (ViewDelta/Patch, PROTOCOL §2.1),
          RESULT (MethodParams), STREAM

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
STREAM = 6
VIEW_DELTA = 7

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


def _string_map(fields, field):
    """Decode a proto3 map<string,string>: repeated entry messages with
    key=1/value=2. Later duplicates win."""
    out = {}
    for number, wire, value in fields:
        if number != field or wire != WIRE_LEN:
            continue
        entry = parse_fields(value)
        out[_string(entry, 1)] = _string(entry, 2)
    return out


def _uints(fields, field):
    return [value for number, _wire, value in fields if number == field]


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
        "cols": to_int32(_uint(fields, 13)),
        "rows": to_int32(_uint(fields, 14)),
        "tags": _string_map(fields, 15),
        "endpoint_label": _string(fields, 16),
        "last_output_ms": to_int64(_uint(fields, 17)),
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
                "access_result": _first(inner, 5) or b"",
                "offset": _uint(inner, 6),
                "found": bool(_uint(inner, 7)),
                "wrapped": bool(_uint(inner, 8)),
                "match_start": _uint(inner, 9),
                "match_end": _uint(inner, 10),
            }
    return response


def decode_stream(data):
    """Decode one STREAM payload (PROTOCOL §4 access streams). ``kind`` is
    "data"/"close" for both directions, "cancel" program -> host and "error"
    host -> program; ``wire_type`` stays opaque to the host."""
    fields = parse_fields(data)
    return {
        "stream_id": _uint(fields, 1),
        "kind": _string(fields, 2),
        "payload": _first(fields, 3) or b"",
        "offset": _uint(fields, 4),
        "error": _string(fields, 5),
        "wire_type": _uint(fields, 6),
    }


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


# -------------------------------------------------------------- view deltas

# Patch ops (PROTOCOL §2.1); the wire carries them as strings.
OP_SET = "set"
OP_REPLACE = "replace"
OP_INSERT = "insert"
OP_REMOVE = "remove"

# Box fields a set patch may carry; children are deliberately excluded (a set
# never touches them, PROTOCOL §2.1).
_BOX_FIELDS = ("size", "pos", "flow", "visible", "cursor", "content", "input", "focused", "style")


def encode_patch(patch):
    """Encode one Patch dict (tui2.proto Patch, PROTOCOL §2.1): ``op`` plus
    the fields that op uses: ``path`` always, ``index`` for insert, ``from``/
    ``to`` for move and ``box`` for set/replace/insert."""
    out = field_string(1, patch.get("op"))
    for step in patch.get("path", ()):
        out += field_varint(2, step)
    if patch.get("index") is not None:
        out += field_varint(3, patch["index"])
    if patch.get("from") is not None:
        out += field_varint(4, patch["from"])
    if patch.get("to") is not None:
        out += field_varint(5, patch["to"])
    if patch.get("box") is not None:
        out += field_msg(6, encode_box(patch["box"]))
    return out


def encode_view_delta(epoch, rev, rev_base, claim, all_keys, patches):
    """Encode a VIEW_DELTA payload (frame type 7, PROTOCOL §2.1). ``claim``
    and ``all_keys`` omitted (None) mean "keep the previous keys"."""
    out = field_varint(1, epoch) + field_varint(2, rev) + field_varint(3, rev_base)
    if claim is not None or all_keys is not None:
        keys = b""
        for key in claim or ():
            keys += field_string(1, key)
        keys += field_bool(2, all_keys)
        out += field_msg(4, keys)
    for patch in patches:
        out += field_msg(5, encode_patch(patch))
    return out


def decode_patch(data):
    fields = parse_fields(data)
    box = _first(fields, 6)
    return {
        "op": _string(fields, 1),
        "path": _uints(fields, 2),
        "index": _uint(fields, 3),
        "from": _uint(fields, 4),
        "to": _uint(fields, 5),
        "box": decode_box(box) if box else None,
    }


def decode_view_delta(data):
    """Decode a VIEW_DELTA payload into a dict mirroring ViewDelta."""
    fields = parse_fields(data)
    delta = {
        "epoch": _uint(fields, 1),
        "rev": _uint(fields, 2),
        "rev_base": _uint(fields, 3),
        "keys": None,
        "patches": [],
    }
    for field, wire, value in fields:
        if field == 4 and wire == WIRE_LEN:
            inner = parse_fields(value)
            delta["keys"] = {"claim": _strings(inner, 1), "all": _bool(inner, 2)}
        elif field == 5 and wire == WIRE_LEN:
            delta["patches"].append(decode_patch(value))
    return delta


def decode_box(data):
    """Decode one Box payload into a dict (keys mirror tui2.proto Box)."""
    fields = parse_fields(data)
    box = {
        "id": _string(fields, 1),
        "flow": _string(fields, 4),
        "visible": _bool(fields, 5) if _first(fields, 5) is not None else None,
        "focused": _bool(fields, 10),
        "style": _string(fields, 12),
        "input": _strings(fields, 9),
        "children": [],
    }
    size = _first(fields, 2)
    if size:
        inner = parse_fields(size)
        box["size"] = (to_int32(_uint(inner, 1)), to_int32(_uint(inner, 2)), to_int32(_uint(inner, 3)))
    pos = _first(fields, 3)
    if pos:
        inner = parse_fields(pos)
        box["pos"] = (to_int32(_uint(inner, 1)), to_int32(_uint(inner, 2)))
    cursor = _first(fields, 7)
    if cursor:
        inner = parse_fields(cursor)
        box["cursor"] = {
            "row": to_int32(_uint(inner, 1)), "col": to_int32(_uint(inner, 2)),
            "shape": _string(inner, 3),
            "visible": _bool(inner, 4) if _first(inner, 4) is not None else None,
        }
    content = _first(fields, 8)
    if content:
        inner = parse_fields(content)
        decoded = {"text": _string(inner, 1), "lines": _strings(inner, 2), "self": _string(inner, 3)}
        props = {}
        for number, wire, value in inner:
            if number == 4 and wire == WIRE_LEN:
                props.update(parse_map_string_string(value))
        if props:
            decoded["props"] = props
        box["content"] = decoded
    box["children"] = [decode_box(value) for number, wire, value in fields
                       if number == 11 and wire == WIRE_LEN]
    return box


def parse_map_string_string(entry):
    fields = parse_fields(entry)
    return {_string(fields, 1): _string(fields, 2)}


# ------------------------------------------------------------------- diff

def diff_view(base, next_root):
    """Compute the VIEW_DELTA ``patches`` list turning the ``base`` box tree
    into ``next_root`` (both decoded dicts), or None when no patch form can
    express the change (the caller then falls back to a full VIEW).

    A node whose non-children fields can be merged emits one ``set`` holding
    just the changed fields. Child-list differences are expressed with
    ``insert``/``remove`` around a deeply-equal common prefix and suffix. A
    child that cannot be patched in place is replaced as a whole subtree.
    Paths are child-index paths from the root, so an empty path targets the
    root."""
    patches = []
    changed = _diff_node(base, next_root, [], patches)
    if changed is None:
        return None
    return patches if changed else None


def _diff_node(base, next_box, path, patches):
    """Append the patches for next_box at path. Returns True when patches were
    appended, False when the two nodes are equal, and None when the change
    cannot be expressed (no set, and the caller must replace the subtree)."""
    for key in _BOX_FIELDS:
        base_value, next_value = base.get(key), next_box.get(key)
        if base_value != next_value and not _merge_representable(base_value, next_value):
            return None
    changed = False
    for key in _BOX_FIELDS:
        base_value, next_value = base.get(key), next_box.get(key)
        if base_value != next_value:
            patches.append({"op": OP_SET, "path": list(path), "box": {key: next_value}})
            changed = True
    base_children = base.get("children") or []
    next_children = next_box.get("children") or []
    if len(base_children) == len(next_children):
        for index, (base_child, next_child) in enumerate(zip(base_children, next_children)):
            child_patches = []
            child_changed = _diff_node(base_child, next_child, path + [index], child_patches)
            if child_changed is None:
                patches.append({"op": OP_REPLACE, "path": path + [index], "box": next_child})
                changed = True
            elif child_changed:
                patches.extend(child_patches)
                changed = True
        return changed
    prefix = _common_prefix(base_children, next_children)
    suffix = _common_suffix(base_children, next_children, prefix)
    for _ in range(prefix, len(base_children) - suffix):
        patches.append({"op": OP_REMOVE, "path": path + [prefix]})
        changed = True
    for index in range(prefix, len(next_children) - suffix):
        patches.append({"op": OP_INSERT, "path": list(path), "index": index, "box": next_children[index]})
        changed = True
    return changed


def _common_prefix(a, b):
    limit = min(len(a), len(b))
    index = 0
    while index < limit and a[index] == b[index]:
        index += 1
    return index


def _common_suffix(a, b, start):
    limit = min(len(a), len(b)) - start
    index = 0
    while index < limit and a[len(a) - 1 - index] == b[len(b) - 1 - index]:
        index += 1
    return index


def _merge_representable(base_value, next_value):
    """Whether a ``set`` patch can turn base_value into next_value under the
    host's proto3 merge (PROTOCOL §2.1). A non-zero scalar cannot be cleared
    because zero fields are omitted from the wire; a repeated field only
    merges from empty."""
    if base_value == next_value:
        return True
    if isinstance(base_value, str):
        return bool(next_value)
    if isinstance(base_value, bool):
        return bool(next_value)
    if isinstance(base_value, tuple):
        if not isinstance(next_value, tuple):
            return False
        return all(next_value[i] != 0 for i in range(len(base_value)) if base_value[i] != 0)
    if isinstance(base_value, list):
        return not base_value
    if isinstance(base_value, dict):
        if not isinstance(next_value, dict):
            return False
        for key in ("text", "self"):
            if base_value.get(key) and not next_value.get(key):
                return False
        if base_value.get("lines") and base_value.get("lines") != next_value.get("lines"):
            return False
        for key in base_value.get("props") or {}:
            if key not in (next_value.get("props") or {}):
                return False
        return True
    return True


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
    # Generic access forwarding (tui2.proto MethodParams fields 28..30):
    # access.call's serialized CommandEnvelope and access.stream.open's
    # program-allocated stream id plus serialized ResourceHandle. All bytes
    # stay opaque to the host.
    encoded += field_bytes(28, params.get("access_command"))
    if params.get("stream_id") is not None:
        encoded += field_varint(29, params["stream_id"])
    encoded += field_bytes(30, params.get("access_resource"))
    encoded += field_string(31, params.get("query"))
    encoded += field_string(32, params.get("search_mode"))
    encoded += field_varint(33, params.get("backward"))
    encoded += field_string(35, params.get("clipboard_id"))
    return field_varint(1, request_id) + field_varint(2, epoch) + \
        field_string(3, method) + field_msg(4, encoded)


# ------------------------------------------------------------------- stream

def encode_stream(frame):
    """Encode one STREAM payload (program -> host data/close/cancel, PROTOCOL
    §4). ``wire_type`` carries the opaque access wire frame type byte."""
    return (field_varint(1, frame.get("stream_id")) +
            field_string(2, frame.get("kind")) +
            field_bytes(3, frame.get("payload")) +
            field_varint(4, frame.get("offset")) +
            field_string(5, frame.get("error")) +
            field_varint(6, frame.get("wire_type")))


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
