package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	gproto "google.golang.org/protobuf/proto"
)

// Marshal serializes one frame: u32 length | u8 type | payload. The length
// counts the type byte plus the payload. A nil message is treated as the
// empty message of type t (frame length 1).
func Marshal(t Type, m gproto.Message, maxMessageBytes uint32) ([]byte, error) {
	if !t.Valid() {
		return nil, &Error{Kind: KindUnknownType, Type: t}
	}
	if m == nil {
		m = NewPayload(t)
	}
	payload, err := gproto.Marshal(m)
	if err != nil {
		return nil, &Error{Kind: KindPayload, Type: t, Err: err}
	}
	limit := maxMessageBytes
	if limit == 0 {
		limit = DefaultMaxMessageBytes
	}
	total := uint64(len(payload)) + 1
	if total > uint64(limit) {
		return nil, &Error{Kind: KindOversize, Type: t, Limit: limit}
	}
	frame := make([]byte, 4+total)
	binary.BigEndian.PutUint32(frame[:4], uint32(total))
	frame[4] = byte(t)
	copy(frame[5:], payload)
	return frame, nil
}

// MarshalAppend appends one frame (u32 length | u8 type | payload) to dst and
// returns the extended slice, so a caller can encode into a reusable buffer.
// A nil message is treated as the empty message of type t. The error kinds
// are the same as Marshal: KindUnknownType, KindPayload and KindOversize. On
// error the returned slice has dst's original length and contents.
func MarshalAppend(dst []byte, t Type, m gproto.Message, maxMessageBytes uint32) ([]byte, error) {
	if !t.Valid() {
		return dst, &Error{Kind: KindUnknownType, Type: t}
	}
	limit := maxMessageBytes
	if limit == 0 {
		limit = DefaultMaxMessageBytes
	}
	start := len(dst)
	dst = append(dst, 0, 0, 0, 0, byte(t))
	if m != nil {
		var err error
		dst, err = gproto.MarshalOptions{}.MarshalAppend(dst, m)
		if err != nil {
			return dst[:start], &Error{Kind: KindPayload, Type: t, Err: err}
		}
	}
	total := uint64(len(dst)-start-5) + 1
	if total > uint64(limit) {
		return dst[:start], &Error{Kind: KindOversize, Type: t, Limit: limit}
	}
	binary.BigEndian.PutUint32(dst[start:start+4], uint32(total))
	return dst, nil
}

// frameDecoderPool reuses the Decoder envelope of DecodeFrame so the
// in-memory helper allocates only the reader and the payload.
var frameDecoderPool = sync.Pool{New: func() any { return &Decoder{} }}

// DecodeFrame decodes one in-memory frame with the parsing order fixed by
// PROTOCOL §0. It returns the raw payload; use UnmarshalPayload to get the
// typed message.
func DecodeFrame(frame []byte, role Role, maxMessageBytes uint32) (Type, []byte, error) {
	if maxMessageBytes == 0 {
		maxMessageBytes = DefaultMaxMessageBytes
	}
	d := frameDecoderPool.Get().(*Decoder)
	d.r = bytes.NewReader(frame)
	d.role = role
	d.max = maxMessageBytes
	d.reuse, d.buf = false, nil
	t, payload, err := d.Decode()
	d.r = nil
	frameDecoderPool.Put(d)
	return t, payload, err
}

// encodeScratchPool reuses frame buffers between Encodes: Encode marshals
// into its own pooled scratch buffer and writes it with a single Write, so
// the steady-state hot path adds no per-frame allocation. Buffers larger
// than scratchRetainLimit are not retained, so one oversized frame cannot
// pin memory. The pool holds *[]byte so Put does not box the slice header.
var encodeScratchPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 4096)
		return &buf
	},
}

// scratchRetainLimit bounds what the encoder pool keeps alive: a full
// default-size frame plus its 5-byte header.
const scratchRetainLimit = DefaultMaxMessageBytes + 5

func putScratch(buf *[]byte) {
	if cap(*buf) > int(scratchRetainLimit) {
		return
	}
	encodeScratchPool.Put(buf)
}

// Encoder writes typed frames for one endpoint, enforcing direction and the
// frame size limit. Encode is safe for concurrent use.
type Encoder struct {
	w    io.Writer
	role Role
	max  uint32
}

// NewEncoder returns an Encoder that writes to w as the given role. A zero
// maxMessageBytes means DefaultMaxMessageBytes.
func NewEncoder(w io.Writer, role Role, maxMessageBytes uint32) *Encoder {
	if maxMessageBytes == 0 {
		maxMessageBytes = DefaultMaxMessageBytes
	}
	return &Encoder{w: w, role: role, max: maxMessageBytes}
}

// Encode marshals and writes one frame. It returns KindDirection when the
// type is illegal for this endpoint and KindOversize when the frame is over
// the limit. The frame is assembled in a pooled scratch buffer and written
// once; w must not retain the byte slice it is given (io.Writer contract).
func (e *Encoder) Encode(t Type, m gproto.Message) error {
	if !e.role.CanSend(t) {
		if !t.Valid() {
			return &Error{Kind: KindUnknownType, Type: t}
		}
		return &Error{Kind: KindDirection, Type: t}
	}
	scratch := encodeScratchPool.Get().(*[]byte)
	frame, err := MarshalAppend((*scratch)[:0], t, m, e.max)
	*scratch = frame[:0]
	if err != nil {
		putScratch(scratch)
		return err
	}
	_, err = e.w.Write(frame)
	putScratch(scratch)
	return err
}

// Decoder reads framed messages for one endpoint. It is not safe for
// concurrent use.
type Decoder struct {
	r     io.Reader
	role  Role
	max   uint32
	reuse bool
	buf   []byte
	// prefix lives in the struct so passing it to io.ReadFull does not
	// heap-allocate a 4-byte local on every Decode.
	prefix [4]byte
}

// NewDecoder returns a Decoder reading from r as the given role. A zero
// maxMessageBytes means DefaultMaxMessageBytes.
func NewDecoder(r io.Reader, role Role, maxMessageBytes uint32) *Decoder {
	if maxMessageBytes == 0 {
		maxMessageBytes = DefaultMaxMessageBytes
	}
	return &Decoder{r: r, role: role, max: maxMessageBytes}
}

// ReuseBuffer switches the decoder to a per-decoder scratch read buffer, so
// Decode does not allocate a payload buffer per frame in steady state. The
// returned payload slice is only valid until the next Decode call; callers
// that keep it must copy it. Without this call NewDecoder keeps the
// historical behavior of one allocation per frame. Chainable and not safe
// for concurrent use.
func (d *Decoder) ReuseBuffer(enabled bool) *Decoder {
	d.reuse = enabled
	return d
}

// Decode reads one frame. It returns io.EOF only at a clean frame boundary;
// a stream that ends inside a frame yields *Error{Kind: KindTruncated}. An
// oversize frame is drained without allocating its payload and reported as
// *Error{Kind: KindOversize} with the classified Type (when readable).
// With ReuseBuffer(true) the payload aliases the decoder's scratch buffer
// and stays valid only until the next Decode.
func (d *Decoder) Decode() (Type, []byte, error) {
	if _, err := io.ReadFull(d.r, d.prefix[:]); err != nil {
		return 0, nil, boundaryError(err)
	}
	length := binary.BigEndian.Uint32(d.prefix[:])
	if length == 0 {
		return 0, nil, &Error{Kind: KindZeroLength}
	}
	if length > d.max {
		var typeByte [1]byte
		if _, err := io.ReadFull(d.r, typeByte[:]); err != nil {
			return 0, nil, frameError(err)
		}
		t := Type(typeByte[0])
		if _, err := io.CopyN(io.Discard, d.r, int64(length)-1); err != nil {
			return t, nil, frameError(err)
		}
		return t, nil, &Error{Kind: KindOversize, Type: t, Limit: d.max}
	}

	var buf []byte
	if d.reuse {
		if cap(d.buf) < int(length) {
			d.buf = make([]byte, length)
		} else {
			d.buf = d.buf[:length]
		}
		buf = d.buf
	} else {
		buf = make([]byte, length)
	}
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return 0, nil, frameError(err)
	}
	t := Type(buf[0])
	if !t.Valid() {
		return t, nil, &Error{Kind: KindUnknownType, Type: t}
	}
	if !d.role.CanReceive(t) {
		return t, nil, &Error{Kind: KindDirection, Type: t}
	}
	return t, buf[1:], nil
}

// DecodeMessage reads one frame and decodes its payload.
func (d *Decoder) DecodeMessage() (Type, gproto.Message, error) {
	t, payload, err := d.Decode()
	if err != nil {
		return t, nil, err
	}
	m, err := UnmarshalPayload(t, payload)
	if err != nil {
		return t, nil, err
	}
	return t, m, nil
}

// boundaryError classifies an error while reading the length prefix: only a
// stream that ends before the first prefix byte is a clean EOF.
func boundaryError(err error) error {
	switch {
	case errors.Is(err, io.EOF):
		return io.EOF
	case errors.Is(err, io.ErrUnexpectedEOF):
		return &Error{Kind: KindTruncated, Err: err}
	default:
		return err
	}
}

// frameError classifies errors inside a frame: any short read is truncation.
func frameError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &Error{Kind: KindTruncated, Err: err}
	}
	return err
}
