package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// Version 是 control protobuf method 合同的代际边界。Version 7 直接切换到
	// latest-screen revision delta，并增加按 request ID 取消长请求的 control frame。
	Version             = 7
	MaxFrameSize        = 4 << 20
	MaxEncodedFrameSize = MaxFrameSize + 7

	TypeHello          uint8 = 0x00
	TypeRequest        uint8 = 0x01
	TypeResponse       uint8 = 0x02
	TypeEvent          uint8 = 0x03
	TypeError          uint8 = 0x04
	TypeResponseBinary uint8 = 0x05
	TypeSessionClose   uint8 = 0x06
	TypeRequestCancel  uint8 = 0x07

	TypeInput          uint8 = 0x11
	TypeResize         uint8 = 0x12
	TypeBootstrapDone  uint8 = 0x13
	TypeScreenUpdate   uint8 = 0x14
	TypeStreamReady    uint8 = 0x15
	TypeSyncLost       uint8 = 0x16
	TypeClosed         uint8 = 0x17
	TypeHistoryRequest uint8 = 0x18
	TypeHistoryReplay  uint8 = 0x19
	TypePTYOutput      uint8 = 0x1a
	TypeFileData       uint8 = 0x21
	TypeFileAck        uint8 = 0x22
	TypeFileFinish     uint8 = 0x23
	TypeFileResult     uint8 = 0x24
	TypeBrowserData    uint8 = 0x31
	TypeBrowserClosed  uint8 = 0x32
)

var (
	ErrFrameTooLarge = errors.New("wire: frame too large")
	ErrShortPayload  = errors.New("wire: short payload")
)

type Encoder struct {
	w io.Writer
}

type Decoder struct {
	r io.Reader
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (e *Encoder) WriteFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	_, err = e.w.Write(frame)
	return err
}

func (d *Decoder) ReadFrame() (uint16, uint8, []byte, error) {
	header := make([]byte, 7)
	if _, err := io.ReadFull(d.r, header); err != nil {
		return 0, 0, nil, err
	}
	channel := binary.BigEndian.Uint16(header[:2])
	typ := header[2]
	length := binary.BigEndian.Uint32(header[3:])
	if length > MaxFrameSize {
		return 0, 0, nil, ErrFrameTooLarge
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(d.r, payload); err != nil {
		return 0, 0, nil, err
	}
	return channel, typ, payload, nil
}

func EncodeFrame(channel uint16, typ uint8, payload []byte) ([]byte, error) {
	if len(payload) > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}
	frame := make([]byte, 7+len(payload))
	binary.BigEndian.PutUint16(frame[:2], channel)
	frame[2] = typ
	binary.BigEndian.PutUint32(frame[3:7], uint32(len(payload)))
	copy(frame[7:], payload)
	return frame, nil
}

func DecodeFrame(frame []byte) (uint16, uint8, []byte, error) {
	if len(frame) < 7 {
		return 0, 0, nil, ErrShortPayload
	}
	channel := binary.BigEndian.Uint16(frame[:2])
	typ := frame[2]
	length := binary.BigEndian.Uint32(frame[3:7])
	if length > MaxFrameSize {
		return 0, 0, nil, ErrFrameTooLarge
	}
	if int(length) != len(frame[7:]) {
		return 0, 0, nil, fmt.Errorf("wire: malformed frame length")
	}
	// The payload aliases frame storage so hot paths like transport dispatch
	// can decode without paying for an extra copy per frame.
	return channel, typ, frame[7:], nil
}

func EncodeResizePayload(cols, rows uint16) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[:2], cols)
	binary.BigEndian.PutUint16(payload[2:], rows)
	return payload
}

func DecodeResizePayload(payload []byte) (uint16, uint16, error) {
	if len(payload) != 4 {
		return 0, 0, ErrShortPayload
	}
	return binary.BigEndian.Uint16(payload[:2]), binary.BigEndian.Uint16(payload[2:]), nil
}

func EncodeStreamReadyPayload(screenSequence uint64) []byte {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, screenSequence)
	return payload
}

func DecodeStreamReadyPayload(payload []byte) (uint64, error) {
	if len(payload) == 0 {
		return 0, nil
	}
	if len(payload) != 8 {
		return 0, ErrShortPayload
	}
	return binary.BigEndian.Uint64(payload), nil
}

func EncodeSyncLostPayload(dropped uint64) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(dropped))
	return payload
}

func DecodeSyncLostPayload(payload []byte) (uint64, error) {
	if len(payload) != 4 {
		return 0, ErrShortPayload
	}
	return uint64(binary.BigEndian.Uint32(payload)), nil
}

func EncodeClosedPayload(code int) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(int32(code)))
	return payload
}

func DecodeClosedPayload(payload []byte) (int, error) {
	if len(payload) != 4 {
		return 0, ErrShortPayload
	}
	return int(int32(binary.BigEndian.Uint32(payload))), nil
}

func EncodeHistoryRequestPayload(beforeOffset int, limit int) []byte {
	payload := make([]byte, 8)
	if beforeOffset < 0 {
		beforeOffset = 0
	}
	if limit < 0 {
		limit = 0
	}
	binary.BigEndian.PutUint32(payload[:4], uint32(beforeOffset))
	binary.BigEndian.PutUint32(payload[4:8], uint32(limit))
	return payload
}

func DecodeHistoryRequestPayload(payload []byte) (int, int, error) {
	if len(payload) != 8 && len(payload) != 9 {
		return 0, 0, ErrShortPayload
	}
	beforeOffset := int(binary.BigEndian.Uint32(payload[:4]))
	limit := int(binary.BigEndian.Uint32(payload[4:8]))
	return beforeOffset, limit, nil
}

func EncodeHistoryReplayPayload(rows int, hasMore bool, replay []byte) ([]byte, error) {
	if rows < 0 {
		rows = 0
	}
	payload := make([]byte, 5+len(replay))
	binary.BigEndian.PutUint32(payload[:4], uint32(rows))
	if hasMore {
		payload[4] = 1
	}
	copy(payload[5:], replay)
	return payload, nil
}

func DecodeHistoryReplayPayload(payload []byte) (int, bool, []byte, error) {
	if len(payload) < 5 {
		return 0, false, nil, ErrShortPayload
	}
	rows := int(binary.BigEndian.Uint32(payload[:4]))
	hasMore := payload[4] == 1
	replay := append([]byte(nil), payload[5:]...)
	return rows, hasMore, replay, nil
}
