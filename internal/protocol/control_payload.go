package protocol

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/proto/access/wirepb"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

const applicationResultEnvelopeMarginBytes = 64 << 10
const MaxApplicationResultEnvelopeBytes = wire.MaxFrameSize - applicationResultEnvelopeMarginBytes

var ErrApplicationResultTooLarge = errors.New("application result envelope exceeds framing budget")

func EncodeHelloPayload(hello Hello) ([]byte, error) {
	return proto.Marshal(&wirepb.Hello{
		Version: uint32(hello.Version), Client: hello.Client, Server: hello.Server,
		Capabilities: append([]string(nil), hello.Capabilities...),
	})
}

func DecodeHelloPayload(payload []byte) (Hello, error) {
	var message wirepb.Hello
	if err := proto.Unmarshal(payload, &message); err != nil {
		return Hello{}, err
	}
	return Hello{
		Version: int(message.GetVersion()), Client: message.GetClient(), Server: message.GetServer(),
		Capabilities: append([]string(nil), message.GetCapabilities()...),
	}, nil
}

// EncodeSessionClosePayload 编码客户端主动关闭当前 protocol session 的 versioned control message。
func EncodeSessionClosePayload() ([]byte, error) {
	return proto.Marshal(&wirepb.SessionClose{Version: uint32(wire.Version)})
}

// DecodeSessionClosePayload 严格校验关闭 frame 的 wire version；失败不能被当作正常 EOF。
func DecodeSessionClosePayload(payload []byte) error {
	var message wirepb.SessionClose
	if err := proto.Unmarshal(payload, &message); err != nil {
		return err
	}
	if message.GetVersion() != uint32(wire.Version) {
		return fmt.Errorf("unsupported session close version %d", message.GetVersion())
	}
	return nil
}

func EncodeRequestCancelPayload(id uint64) ([]byte, error) {
	if id == 0 {
		return nil, fmt.Errorf("protocol request cancel ID is required")
	}
	return proto.Marshal(&wirepb.RequestCancel{Id: id})
}

func DecodeRequestCancelPayload(payload []byte) (uint64, error) {
	var message wirepb.RequestCancel
	if err := proto.Unmarshal(payload, &message); err != nil {
		return 0, err
	}
	if message.GetId() == 0 {
		return 0, fmt.Errorf("protocol request cancel ID is required")
	}
	return message.GetId(), nil
}

func EncodeRequestPayload(request Request) ([]byte, error) {
	return proto.Marshal(&wirepb.RequestEnvelope{Id: request.ID, Method: request.Method, Params: append([]byte(nil), request.Params...)})
}

func DecodeRequestPayload(payload []byte) (Request, error) {
	var message wirepb.RequestEnvelope
	if err := proto.Unmarshal(payload, &message); err != nil {
		return Request{}, err
	}
	return Request{ID: message.GetId(), Method: message.GetMethod(), Params: append([]byte(nil), message.GetParams()...)}, nil
}

func EncodeResponsePayload(response Response) ([]byte, error) {
	encoding := wirepb.ResponseEncoding_RESPONSE_ENCODING_RAW
	switch response.Encoding {
	case ResponseEncodingDeflate:
		encoding = wirepb.ResponseEncoding_RESPONSE_ENCODING_DEFLATE
	case ResponseEncodingZstd:
		encoding = wirepb.ResponseEncoding_RESPONSE_ENCODING_ZSTD
	}
	return proto.Marshal(&wirepb.ResponseEnvelope{
		Id: response.ID, Result: append([]byte(nil), response.Result...), Encoding: encoding,
	})
}

func DecodeResponsePayload(payload []byte) (Response, error) {
	var message wirepb.ResponseEnvelope
	if err := proto.Unmarshal(payload, &message); err != nil {
		return Response{}, err
	}
	encoding := ResponseEncodingRaw
	switch message.GetEncoding() {
	case wirepb.ResponseEncoding_RESPONSE_ENCODING_DEFLATE:
		encoding = ResponseEncodingDeflate
	case wirepb.ResponseEncoding_RESPONSE_ENCODING_ZSTD:
		encoding = ResponseEncodingZstd
	}
	return Response{ID: message.GetId(), Result: append([]byte(nil), message.GetResult()...), Encoding: encoding}, nil
}

var (
	zstdEncoderOnce sync.Once
	zstdEncoder     *zstd.Encoder
	zstdDecoderOnce sync.Once
	zstdDecoder     *zstd.Decoder
	zstdDecoderErr  error
)

// CompressZstdApplicationResult 用 zstd(SpeedFastest) 压缩一个 application
// result。zstd 的 EncodeAll 并发安全；比 deflate 小约 1.5×、CPU 低一个数量级。
func CompressZstdApplicationResult(payload []byte) ([]byte, bool) {
	zstdEncoderOnce.Do(func() {
		zstdEncoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	})
	if zstdEncoder == nil {
		return nil, false
	}
	return zstdEncoder.EncodeAll(payload, nil), true
}

// inflateZstdApplicationResult 解开 zstd 压缩的 application result。
func inflateZstdApplicationResult(payload []byte) ([]byte, error) {
	zstdDecoderOnce.Do(func() {
		zstdDecoder, zstdDecoderErr = zstd.NewReader(nil,
			zstd.WithDecoderConcurrency(1),
			zstd.WithDecoderMaxMemory(uint64(wire.MaxFrameSize)),
		)
	})
	if zstdDecoderErr != nil {
		return nil, zstdDecoderErr
	}
	out, err := zstdDecoder.DecodeAll(payload, nil)
	if err != nil {
		return nil, err
	}
	if len(out) > wire.MaxFrameSize {
		return nil, fmt.Errorf("inflated application result exceeds %d bytes", wire.MaxFrameSize)
	}
	return out, nil
}

// screenStreamEncoding 是 TypeScreen payload 的首字节标记。
const (
	screenStreamEncodingRaw  byte = 0
	screenStreamEncodingZstd byte = 1
)

// EncodeScreenStreamPayload 把一帧 native screen delta 编码成 attachment
// stream 的 TypeScreen payload：首字节标记 + 原始或 zstd 数据。屏幕帧和
// LiveScreenNext 应答走同一套压缩，慢链路上体积一致。
func EncodeScreenStreamPayload(payload []byte) []byte {
	if compressed, ok := CompressZstdApplicationResult(payload); ok && len(compressed) < len(payload) {
		frame := make([]byte, 1+len(compressed))
		frame[0] = screenStreamEncodingZstd
		copy(frame[1:], compressed)
		return frame
	}
	frame := make([]byte, 1+len(payload))
	frame[0] = screenStreamEncodingRaw
	copy(frame[1:], payload)
	return frame
}

// DecodeScreenStreamPayload 解开 TypeScreen payload，返回 native screen delta
// 的 protobuf 字节。
func DecodeScreenStreamPayload(frame []byte) ([]byte, error) {
	if len(frame) == 0 {
		return nil, errors.New("protocol: empty screen stream frame")
	}
	switch frame[0] {
	case screenStreamEncodingRaw:
		return frame[1:], nil
	case screenStreamEncodingZstd:
		return inflateZstdApplicationResult(frame[1:])
	default:
		return nil, fmt.Errorf("protocol: unknown screen stream encoding %d", frame[0])
	}
}

// inflateApplicationResult 解开 deflate 压缩的 application result。
// 解压上限是单帧上限，防止对端构造 zip bomb。
func inflateApplicationResult(payload []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(payload))
	defer func() { _ = reader.Close() }()
	limited := io.LimitReader(reader, wire.MaxFrameSize+1)
	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(out) > wire.MaxFrameSize {
		return nil, fmt.Errorf("inflated application result exceeds %d bytes", wire.MaxFrameSize)
	}
	return out, nil
}

func EncodeErrorPayload(message ErrorMessage) ([]byte, error) {
	return proto.Marshal(&wirepb.ErrorEnvelope{Id: message.ID, Error: &wirepb.ProtocolError{Code: int32(message.Error.Code), Message: message.Error.Message}})
}

func DecodeErrorPayload(payload []byte) (ErrorMessage, error) {
	var message wirepb.ErrorEnvelope
	if err := proto.Unmarshal(payload, &message); err != nil {
		return ErrorMessage{}, err
	}
	if message.GetError() == nil || message.GetError().GetCode() == 0 || message.GetError().GetMessage() == "" {
		return ErrorMessage{}, fmt.Errorf("protocol error payload requires code and message")
	}
	return ErrorMessage{ID: message.GetId(), Error: ProtocolError{Code: int(message.GetError().GetCode()), Message: message.GetError().GetMessage()}}, nil
}

// EncodeApplicationCommand 编码唯一公共 application command payload。
func EncodeApplicationCommand(command *apipb.CommandEnvelope) ([]byte, error) {
	if command == nil {
		return nil, fmt.Errorf("protocol: application command is required")
	}
	return proto.Marshal(command)
}

// DecodeApplicationCommand 解码唯一公共 application command payload。
func DecodeApplicationCommand(payload []byte) (*apipb.CommandEnvelope, error) {
	var command apipb.CommandEnvelope
	if err := proto.Unmarshal(payload, &command); err != nil {
		return nil, err
	}
	return &command, nil
}

// EncodeApplicationResult 编码唯一公共 application result payload。
func EncodeApplicationResult(envelope *apipb.ResultEnvelope) ([]byte, error) {
	if envelope == nil {
		return nil, fmt.Errorf("protocol: application result is required")
	}
	if size := proto.Size(envelope); size > MaxApplicationResultEnvelopeBytes {
		return nil, fmt.Errorf("%w: maximum is %d bytes", ErrApplicationResultTooLarge, MaxApplicationResultEnvelopeBytes)
	}
	return proto.Marshal(envelope)
}

// DecodeApplicationResult 解码唯一公共 application result payload。
func DecodeApplicationResult(payload []byte) (*apipb.ResultEnvelope, error) {
	var envelope apipb.ResultEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
