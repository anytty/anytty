package screenmodel

import (
	"errors"
	"fmt"
	"sync"

	"github.com/anytty/anytty/proto/access/wire"
	"github.com/klauspost/compress/zstd"
)

// TypeScreen 的 payload 是首字节标记 + 原始或 zstd 压缩的
// apipb.NativeScreenResult。编解码和屏幕模型放在一起：它属于屏幕状态的
// 线格式，不属于传输层 framing。
const (
	frameEncodingRaw  byte = 0
	frameEncodingZstd byte = 1
)

var (
	frameEncoderOnce sync.Once
	frameEncoder     *zstd.Encoder
	frameDecoderOnce sync.Once
	frameDecoder     *zstd.Decoder
	frameDecoderErr  error
)

// EncodeFrame 编码一帧 native screen delta。
func EncodeFrame(payload []byte) []byte {
	frameEncoderOnce.Do(func() {
		frameEncoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	})
	if frameEncoder != nil {
		if compressed := frameEncoder.EncodeAll(payload, nil); len(compressed) < len(payload) {
			frame := make([]byte, 1+len(compressed))
			frame[0] = frameEncodingZstd
			copy(frame[1:], compressed)
			return frame
		}
	}
	frame := make([]byte, 1+len(payload))
	frame[0] = frameEncodingRaw
	copy(frame[1:], payload)
	return frame
}

// DecodeFrame 解开一帧 native screen delta，返回 protobuf 字节。
func DecodeFrame(frame []byte) ([]byte, error) {
	if len(frame) == 0 {
		return nil, errors.New("screenmodel: empty screen frame")
	}
	switch frame[0] {
	case frameEncodingRaw:
		return frame[1:], nil
	case frameEncodingZstd:
		frameDecoderOnce.Do(func() {
			frameDecoder, frameDecoderErr = zstd.NewReader(nil,
				zstd.WithDecoderConcurrency(1),
				zstd.WithDecoderMaxMemory(uint64(wire.MaxFrameSize)),
			)
		})
		if frameDecoderErr != nil {
			return nil, frameDecoderErr
		}
		out, err := frameDecoder.DecodeAll(frame[1:], nil)
		if err != nil {
			return nil, err
		}
		if len(out) > wire.MaxFrameSize {
			return nil, fmt.Errorf("screenmodel: inflated screen frame exceeds %d bytes", wire.MaxFrameSize)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("screenmodel: unknown screen frame encoding %d", frame[0])
	}
}
