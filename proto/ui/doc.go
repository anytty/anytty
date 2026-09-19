// Package proto implements the TUI v2 binary frame envelope defined by
// tui2/docs/PROTOCOL.zh-CN.md §0.
//
// A frame is:
//
//	u32 length (big endian) | u8 type | protobuf payload
//
// length counts every byte after the length field, i.e. the type byte plus
// the payload, so a frame is never empty (length 0 is malformed). The
// protobuf payloads live in tui2/protobuf (generated from tui2.proto in this
// directory).
//
// Parsing boundary (order is fixed by the protocol):
//
//  1. read and validate the length: 0 or > max_message_bytes rejects the
//     frame before any payload buffer is allocated;
//  2. classify the type byte (oversize frames are drained and reported as
//     KindOversize so the caller can answer view_rejected/oversize without
//     reallocating the payload);
//  3. allocate and read the payload, then decode the protobuf.
//
// Direction is validated on both ends: the host endpoint only receives
// VIEW/RESULT and only sends HELLO/EVENT/RESPONSE; the program endpoint is
// the mirror image. A frame in the wrong direction is a decode error and
// disconnects the program.
//
// The decoder never panics on a half-written or truncated stream: it returns
// io.EOF at a clean frame boundary and *Error{Kind: KindTruncated} in the
// middle of a frame.
package proto
