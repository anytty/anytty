/**
 * TUI v2 wire codec (Node stdlib only): protobuf payloads + binary frames.
 * The TypeScript-facing types live in ../../index.d.ts.
 *
 * Wire basics: `varint(tag) | payload`, tag = (field << 3) | wire_type;
 * 0 = varint, 1 = 64-bit, 2 = length-delimited, 5 = 32-bit. A frame is
 * `u32 length (big-endian) | u8 type | payload` (PROTOCOL §0).
 */

'use strict';

const HELLO = 1;
const VIEW = 2;
const EVENT = 3;
const RESULT = 4;
const RESPONSE = 5;

const WIRE_VARINT = 0;
const WIRE_64BIT = 1;
const WIRE_LEN = 2;
const WIRE_32BIT = 5;

const MAX_MESSAGE_BYTES = 1 << 20;

class WireError extends Error {}

// ----------------------------------------------------------------- encoding

function uvarint(value) {
  let big = BigInt(value);
  if (big < 0n) big += 1n << 64n;
  const out = [];
  for (;;) {
    const byte = Number(big & 0x7fn);
    big >>= 7n;
    if (big) out.push(byte | 0x80);
    else {
      out.push(byte);
      return Buffer.from(out);
    }
  }
}

function fieldVarint(field, value) {
  if (value === undefined || value === null) return Buffer.alloc(0);
  return Buffer.concat([uvarint(field << 3), uvarint(value)]);
}

function fieldBool(field, value) {
  if (value === undefined || value === null) return Buffer.alloc(0);
  return fieldVarint(field, value ? 1 : 0);
}

function fieldBytes(field, data) {
  if (!data || data.length === 0) return Buffer.alloc(0);
  return Buffer.concat([uvarint((field << 3) | WIRE_LEN), uvarint(data.length), Buffer.from(data)]);
}

function fieldString(field, text) {
  if (!text) return Buffer.alloc(0);
  return fieldBytes(field, Buffer.from(text, 'utf8'));
}

function fieldMsg(field, message) {
  if (message === null || message === undefined) return Buffer.alloc(0);
  return fieldBytes(field, message);
}

function fieldMapStringString(field, mapping) {
  const parts = [];
  for (const key of Object.keys(mapping).sort()) {
    parts.push(Buffer.concat([fieldString(1, key), fieldString(2, mapping[key])]));
  }
  return parts.map((entry) => fieldMsg(field, entry)).reduce((a, b) => Buffer.concat([a, b]), Buffer.alloc(0));
}

function fieldMapStringBool(field, mapping) {
  const parts = [];
  for (const key of Object.keys(mapping).sort()) {
    parts.push(Buffer.concat([fieldString(1, key), fieldBool(2, mapping[key])]));
  }
  return parts.map((entry) => fieldMsg(field, entry)).reduce((a, b) => Buffer.concat([a, b]), Buffer.alloc(0));
}

// ----------------------------------------------------------------- decoding

function readUvarint(buf, index) {
  let result = 0n;
  let shift = 0n;
  for (;;) {
    const byte = buf[index];
    index += 1;
    result |= BigInt(byte & 0x7f) << shift;
    if (!(byte & 0x80)) return [result, index];
    shift += 7n;
  }
}

function parseFields(data) {
  const out = [];
  let index = 0;
  while (index < data.length) {
    let tag;
    [tag, index] = readUvarint(data, index);
    const field = Number(tag >> 3n);
    const wire = Number(tag & 7n);
    let value;
    if (wire === WIRE_VARINT) {
      [value, index] = readUvarint(data, index);
    } else if (wire === WIRE_LEN) {
      let length;
      [length, index] = readUvarint(data, index);
      length = Number(length);
      value = data.subarray(index, index + length);
      index += length;
    } else if (wire === WIRE_64BIT) {
      value = data.subarray(index, index + 8);
      index += 8;
    } else if (wire === WIRE_32BIT) {
      value = data.subarray(index, index + 4);
      index += 4;
    } else {
      throw new WireError(`unsupported wire type ${wire}`);
    }
    out.push([field, wire, value]);
  }
  return out;
}

function toInt32(value) {
  const big = BigInt(value);
  return Number(BigInt.asIntN(32, big));
}

function toInt64(value) {
  const big = BigInt(value);
  return Number(BigInt.asIntN(64, big));
}

function first(fields, field) {
  for (const [number, , value] of fields) if (number === field) return value;
  return undefined;
}

function strings(fields, field) {
  return fields.filter(([number]) => number === field).map(([, , value]) => value.toString('utf8'));
}

function string(fields, field) {
  const value = first(fields, field);
  return value === undefined ? '' : value.toString('utf8');
}

function uint(fields, field) {
  const value = first(fields, field);
  return value === undefined ? 0 : Number(value);
}

function bool(fields, field) {
  return uint(fields, field) !== 0;
}

function parseLimits(data) {
  const fields = parseFields(data);
  return {
    max_nodes: uint(fields, 1),
    max_message_bytes: uint(fields, 2),
    max_paste_bytes: uint(fields, 3),
    max_inflight_requests: uint(fields, 4),
    owner_lease_ttl_ms: uint(fields, 5),
  };
}

function parseMapStringBool(entry) {
  const fields = parseFields(entry);
  return { [string(fields, 1)]: bool(fields, 2) };
}

function decodeHello(data) {
  const fields = parseFields(data);
  const hello = {
    schema: uint(fields, 1), view_id: string(fields, 2), epoch: uint(fields, 3),
    cols: uint(fields, 4), rows: uint(fields, 5),
    components: strings(fields, 6), events: strings(fields, 7), methods: strings(fields, 8),
    features: {}, limits: {},
  };
  for (const [field, wire, value] of fields) {
    if (field === 9 && wire === WIRE_LEN) Object.assign(hello.features, parseMapStringBool(value));
    else if (field === 10 && wire === WIRE_LEN) hello.limits = parseLimits(value);
  }
  return hello;
}

function parseSource(data) {
  const fields = parseFields(data);
  return {
    id: string(fields, 1), kind: string(fields, 2), title: string(fields, 3),
    endpoint: string(fields, 4), terminal_id: string(fields, 5),
    attached: bool(fields, 6), exited: bool(fields, 7),
    exit_code: toInt32(uint(fields, 8)), health: string(fields, 9),
    resize_owner: string(fields, 10), owner_epoch: uint(fields, 11),
    last_seen_ms: toInt64(uint(fields, 12)),
  };
}

function decodeEvent(data) {
  for (const [field, wire, value] of parseFields(data)) {
    if (wire !== WIRE_LEN) continue;
    const inner = parseFields(value);
    if (field === 1) return { kind: 'key', id: string(inner, 1), key: string(inner, 2), char: string(inner, 3) };
    if (field === 2) return { kind: 'paste', id: string(inner, 1), text: string(inner, 2) };
    if (field === 3) return {
      kind: 'mouse', action: string(inner, 1), button: string(inner, 2),
      x: toInt32(uint(inner, 3)), y: toInt32(uint(inner, 4)), node: string(inner, 5),
    };
    if (field === 4) return {
      kind: 'wheel', delta: toInt32(uint(inner, 1)), x: toInt32(uint(inner, 2)),
      y: toInt32(uint(inner, 3)), node: string(inner, 4),
    };
    if (field === 5) return { kind: 'resize', cols: uint(inner, 1), rows: uint(inner, 2) };
    if (field === 6) {
      const items = [];
      for (const [number, itemWire, item] of inner) {
        if (number === 1 && itemWire === WIRE_LEN) items.push(parseSource(item));
      }
      return { kind: 'sources', items };
    }
    if (field === 7) return { kind: 'notice', level: string(inner, 1), message: string(inner, 2) };
    if (field === 8) return { kind: 'component', source: string(inner, 1), name: string(inner, 2), value: string(inner, 3) };
    if (field === 9) return { kind: 'view_rejected', epoch: uint(inner, 1), rev: uint(inner, 2), reason: string(inner, 3) };
  }
  return { kind: 'unknown' };
}

function decodeResponse(data) {
  const fields = parseFields(data);
  const response = {
    request_id: uint(fields, 1), epoch: uint(fields, 2), ok: bool(fields, 3),
    data: {}, error: string(fields, 5),
  };
  for (const [field, wire, value] of fields) {
    if (field === 4 && wire === WIRE_LEN) {
      const inner = parseFields(value);
      response.data = {
        rows: strings(inner, 1), text: string(inner, 2),
        endpoint: string(inner, 3), id: string(inner, 4),
      };
    }
  }
  return response;
}

// ---------------------------------------------------------------- box trees

function encodeCursor(cursor) {
  let out = Buffer.concat([fieldVarint(1, cursor.row || 0), fieldVarint(2, cursor.col || 0)]);
  out = Buffer.concat([out, fieldString(3, cursor.shape)]);
  if (cursor.visible !== undefined && cursor.visible !== null) out = Buffer.concat([out, fieldBool(4, cursor.visible)]);
  return out;
}

function encodeBox(box) {
  let out = fieldString(1, box.id);
  if (box.size) {
    const size = Buffer.concat([
      fieldVarint(1, box.size[0]),
      fieldVarint(2, box.size[1]),
      fieldVarint(3, box.size[2] || 0),
    ]);
    out = Buffer.concat([out, fieldMsg(2, size)]);
  }
  if (box.pos) {
    const pos = Buffer.concat([fieldVarint(1, box.pos[0]), fieldVarint(2, box.pos[1])]);
    out = Buffer.concat([out, fieldMsg(3, pos)]);
  }
  out = Buffer.concat([out, fieldString(4, box.flow)]);
  if (box.visible !== undefined && box.visible !== null) out = Buffer.concat([out, fieldBool(5, box.visible)]);
  if (box.cursor) out = Buffer.concat([out, fieldMsg(7, encodeCursor(box.cursor))]);
  if (box.content) {
    const content = box.content;
    let encoded = fieldString(1, content.text);
    for (const line of content.lines || []) encoded = Buffer.concat([encoded, fieldString(2, line)]);
    encoded = Buffer.concat([encoded, fieldString(3, content.self)]);
    if (content.props && Object.keys(content.props).length) encoded = Buffer.concat([encoded, fieldMapStringString(4, content.props)]);
    out = Buffer.concat([out, fieldMsg(8, encoded)]);
  }
  for (const kind of box.input || []) out = Buffer.concat([out, fieldString(9, kind)]);
  out = Buffer.concat([out, fieldBool(10, box.focused)]);
  for (const child of box.children || []) out = Buffer.concat([out, fieldMsg(11, encodeBox(child))]);
  return Buffer.concat([out, fieldString(12, box.style)]);
}

function encodeView(epoch, rev, claim, allKeys, root) {
  let keys = Buffer.alloc(0);
  for (const key of claim) keys = Buffer.concat([keys, fieldString(1, key)]);
  keys = Buffer.concat([keys, fieldBool(2, allKeys)]);
  return Buffer.concat([
    fieldVarint(1, epoch), fieldVarint(2, rev), fieldMsg(3, keys), fieldMsg(4, encodeBox(root))]);
}

// ------------------------------------------------------------------- result

function encodeResult(requestId, epoch, method, params) {
  const p = params || {};
  let encoded = fieldString(1, p.endpoint);
  encoded = Buffer.concat([encoded, fieldString(2, p.id)]);
  if (p.fit !== undefined && p.fit !== null) encoded = Buffer.concat([encoded, fieldBool(3, p.fit)]);
  if (p.expected_owner_epoch !== undefined && p.expected_owner_epoch !== null) {
    encoded = Buffer.concat([encoded, fieldVarint(4, p.expected_owner_epoch)]);
  }
  for (const arg of p.argv || []) encoded = Buffer.concat([encoded, fieldString(5, arg)]);
  encoded = Buffer.concat([encoded, fieldString(6, p.cwd)]);
  if (p.env && Object.keys(p.env).length) encoded = Buffer.concat([encoded, fieldMapStringString(7, p.env)]);
  encoded = Buffer.concat([encoded, fieldString(8, p.title)]);
  if (p.ephemeral !== undefined && p.ephemeral !== null) encoded = Buffer.concat([encoded, fieldBool(9, p.ephemeral)]);
  if (p.delta !== undefined && p.delta !== null) encoded = Buffer.concat([encoded, fieldVarint(10, p.delta)]);
  if (p.sel) encoded = Buffer.concat([encoded, fieldMsg(11, Buffer.concat([
    fieldString(1, p.sel.mode), fieldVarint(2, p.sel.start), fieldVarint(3, p.sel.end)]))]);
  if (p.offset !== undefined && p.offset !== null) encoded = Buffer.concat([encoded, fieldVarint(12, p.offset)]);
  if (p.rows !== undefined && p.rows !== null) encoded = Buffer.concat([encoded, fieldVarint(13, p.rows)]);
  encoded = Buffer.concat([encoded, fieldString(14, p.event_id)]);
  encoded = Buffer.concat([encoded, fieldString(15, p.source)]);
  if (p.cleanup_owned !== undefined && p.cleanup_owned !== null) encoded = Buffer.concat([encoded, fieldBool(16, p.cleanup_owned)]);
  encoded = Buffer.concat([encoded, fieldString(17, p.kind)]);
  encoded = Buffer.concat([encoded, fieldString(18, p.socket)]);
  encoded = Buffer.concat([encoded, fieldString(19, p.connect_mode)]);
  encoded = Buffer.concat([encoded, fieldString(20, p.address)]);
  return Buffer.concat([
    fieldVarint(1, requestId), fieldVarint(2, epoch), fieldString(3, method), fieldMsg(4, encoded)]);
}

// -------------------------------------------------------------------- frame

function frame(frameType, payload) {
  const body = Buffer.concat([Buffer.from([frameType]), payload]);
  const prefix = Buffer.alloc(4);
  prefix.writeUInt32BE(body.length, 0);
  return Buffer.concat([prefix, body]);
}

/** Async reader: `next()` resolves `{type, payload}` or null on clean EOF. */
class FrameReader {
  constructor(stream) {
    this.buffer = Buffer.alloc(0);
    this.waiters = [];
    this.done = false;
    this.error = null;
    stream.on('data', (chunk) => {
      this.buffer = Buffer.concat([this.buffer, chunk]);
      this.pump();
    });
    stream.on('end', () => {
      this.done = true;
      this.pump();
    });
    stream.on('error', (error) => {
      this.error = error;
      this.pump();
    });
  }

  next() {
    return new Promise((resolve, reject) => {
      this.waiters.push({ resolve, reject });
      this.pump();
    });
  }

  pump() {
    while (this.waiters.length) {
      let parsed;
      try {
        parsed = this.tryRead();
      } catch (error) {
        const waiter = this.waiters.shift();
        waiter.reject(error);
        continue;
      }
      if (parsed === undefined) return;
      const waiter = this.waiters.shift();
      waiter.resolve(parsed);
    }
  }

  tryRead() {
    if (this.error) throw this.error;
    if (this.buffer.length < 4) return this.done ? null : undefined;
    const length = this.buffer.readUInt32BE(0);
    if (length === 0) throw new WireError('zero-length frame');
    if (length > MAX_MESSAGE_BYTES) throw new WireError('frame exceeds max_message_bytes');
    if (this.buffer.length < 4 + length) {
      if (this.done) throw new WireError('truncated frame');
      return undefined;
    }
    const body = this.buffer.subarray(4, 4 + length);
    this.buffer = this.buffer.subarray(4 + length);
    return { type: body[0], payload: body.subarray(1) };
  }
}

module.exports = {
  HELLO, VIEW, EVENT, RESULT, RESPONSE, MAX_MESSAGE_BYTES, WireError,
  uvarint, fieldVarint, fieldBool, fieldBytes, fieldString, fieldMsg,
  fieldMapStringString, fieldMapStringBool, parseFields, readUvarint,
  toInt32, toInt64, decodeHello, decodeEvent, decodeResponse,
  encodeBox, encodeCursor, encodeView, encodeResult, frame, FrameReader,
};
