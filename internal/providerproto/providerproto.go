// Package providerproto 是 pool terminal provider 协议的 framing 与控制载荷边界。
//
// provider 协议只服务 access→pool 的内部连接：channel 0 承载 Hello/Request/
// Response/Error/Event protobuf 载荷，channel > 0 预留给 attachment/history/live
// 流（复用现有 [channel:2][type:1] frame 与 wirepb 载荷）。它不是客户端 wire。
package providerproto

import (
	"fmt"

	"github.com/anytty/anytty/proto/access/wire"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"google.golang.org/protobuf/proto"
)

// Version 是 provider 协议代际；Hello 双方必须一致。
const Version uint32 = 1

// provider 控制 frame 类型（channel 0）。数值与 access wire 分离，避免跨协议误读。
const (
	TypeHello    uint8 = 0x40
	TypeRequest  uint8 = 0x41
	TypeResponse uint8 = 0x42
	TypeError    uint8 = 0x43
	TypeEvent    uint8 = 0x44
)

// provider stream frame 类型（channel > 0）。数值复用 access wire 的 stream
// frame 语义，payload 也复用 wirepb/既有编码（stream ready/PTY output/closed）。
const (
	StreamBootstrapDone uint8 = wire.TypeBootstrapDone
	StreamReady         uint8 = wire.TypeStreamReady
	StreamSyncLost      uint8 = wire.TypeSyncLost
	StreamClosed        uint8 = wire.TypeClosed
	StreamPTYOutput     uint8 = wire.TypePTYOutput
)

// typed 错误码；语义与 clientruntime/access code 对齐。
const (
	ErrorBadRequest    uint32 = 400
	ErrorForbidden     uint32 = 403
	ErrorNotFound      uint32 = 404
	ErrorConflict      uint32 = 409
	ErrorExhausted     uint32 = 429
	ErrorInternal      uint32 = 500
	ErrorUnavailable   uint32 = 503
	ErrorStaleResource uint32 = 412
)

// EncodeFrame 编码一个 provider frame；布局与 access wire 完全一致。
func EncodeFrame(channel uint16, typ uint8, payload []byte) ([]byte, error) {
	return wire.EncodeFrame(channel, typ, payload)
}

// DecodeFrame 解码一个 provider frame。
func DecodeFrame(frame []byte) (uint16, uint8, []byte, error) {
	return wire.DecodeFrame(frame)
}

// EncodeHelloPayload 编码 Hello control payload。
func EncodeHelloPayload(hello *providerv1.Hello) ([]byte, error) {
	if hello == nil {
		return nil, fmt.Errorf("provider: hello is required")
	}
	return proto.Marshal(hello)
}

// DecodeHelloPayload 解码 Hello control payload。
func DecodeHelloPayload(payload []byte) (*providerv1.Hello, error) {
	hello := &providerv1.Hello{}
	if err := proto.Unmarshal(payload, hello); err != nil {
		return nil, fmt.Errorf("provider: malformed Hello: %w", err)
	}
	return hello, nil
}

// EncodeRequestPayload 编码 Request control payload。
func EncodeRequestPayload(request *providerv1.Request) ([]byte, error) {
	if request == nil {
		return nil, fmt.Errorf("provider: request is required")
	}
	return proto.Marshal(request)
}

// DecodeRequestPayload 解码 Request control payload。
func DecodeRequestPayload(payload []byte) (*providerv1.Request, error) {
	var request providerv1.Request
	if err := proto.Unmarshal(payload, &request); err != nil {
		return nil, fmt.Errorf("provider: malformed request: %w", err)
	}
	return &request, nil
}

// EncodeResponsePayload 编码 Response control payload。
func EncodeResponsePayload(response *providerv1.Response) ([]byte, error) {
	if response == nil {
		return nil, fmt.Errorf("provider: response is required")
	}
	return proto.Marshal(response)
}

// DecodeResponsePayload 解码 Response control payload。
func DecodeResponsePayload(payload []byte) (*providerv1.Response, error) {
	var response providerv1.Response
	if err := proto.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("provider: malformed response: %w", err)
	}
	return &response, nil
}

// EncodeEventPayload 编码 provider Event control payload。
func DecodeEventPayload(payload []byte) (*providerv1.Event, error) {
	event := &providerv1.Event{}
	if err := proto.Unmarshal(payload, event); err != nil {
		return nil, fmt.Errorf("provider: malformed event: %w", err)
	}
	return event, nil
}

// EncodeErrorPayload 生成只带 typed error 的 Response payload。
func EncodeErrorPayload(id uint64, code uint32, message string) ([]byte, error) {
	return EncodeResponsePayload(&providerv1.Response{
		Id:    id,
		Error: &providerv1.ProtocolError{Code: code, Message: message},
	})
}
