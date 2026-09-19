package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

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

// DecodeFrame decodes one in-memory frame with the parsing order fixed by
// PROTOCOL §0. It returns the raw payload; use UnmarshalPayload to get the
// typed message.
func DecodeFrame(frame []byte, role Role, maxMessageBytes uint32) (Type, []byte, error) {
	return NewDecoder(bytes.NewReader(frame), role, maxMessageBytes).Decode()
}

// Encoder writes typed frames for one endpoint, enforcing direction and the
// frame size limit.
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
// the limit.
func (e *Encoder) Encode(t Type, m gproto.Message) error {
	if !e.role.CanSend(t) {
		if !t.Valid() {
			return &Error{Kind: KindUnknownType, Type: t}
		}
		return &Error{Kind: KindDirection, Type: t}
	}
	frame, err := Marshal(t, m, e.max)
	if err != nil {
		return err
	}
	_, err = e.w.Write(frame)
	return err
}

// Decoder reads framed messages for one endpoint. It is not safe for
// concurrent use.
type Decoder struct {
	r    io.Reader
	role Role
	max  uint32
}

// NewDecoder returns a Decoder reading from r as the given role. A zero
// maxMessageBytes means DefaultMaxMessageBytes.
func NewDecoder(r io.Reader, role Role, maxMessageBytes uint32) *Decoder {
	if maxMessageBytes == 0 {
		maxMessageBytes = DefaultMaxMessageBytes
	}
	return &Decoder{r: r, role: role, max: maxMessageBytes}
}

// Decode reads one frame. It returns io.EOF only at a clean frame boundary;
// a stream that ends inside a frame yields *Error{Kind: KindTruncated}. An
// oversize frame is drained without allocating its payload and reported as
// *Error{Kind: KindOversize} with the classified Type (when readable).
func (d *Decoder) Decode() (Type, []byte, error) {
	var prefix [4]byte
	if _, err := io.ReadFull(d.r, prefix[:]); err != nil {
		return 0, nil, boundaryError(err)
	}
	length := binary.BigEndian.Uint32(prefix[:])
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

	buf := make([]byte, length)
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
