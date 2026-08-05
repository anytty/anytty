package protocol

import (
	"errors"
	"fmt"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/proto/wirepb"
	"google.golang.org/protobuf/proto"
)

const applicationResultEnvelopeMarginBytes = 64 << 10
const MaxApplicationResultEnvelopeBytes = wire.MaxFrameSize - applicationResultEnvelopeMarginBytes

var ErrApplicationResultTooLarge = errors.New("application result envelope exceeds framing budget")

func EncodeHelloPayload(hello Hello) ([]byte, error) {
	return proto.Marshal(&wirepb.Hello{Version: uint32(hello.Version), Client: hello.Client, Server: hello.Server})
}

func DecodeHelloPayload(payload []byte) (Hello, error) {
	var message wirepb.Hello
	if err := proto.Unmarshal(payload, &message); err != nil {
		return Hello{}, err
	}
	return Hello{Version: int(message.GetVersion()), Client: message.GetClient(), Server: message.GetServer()}, nil
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
	return proto.Marshal(&wirepb.ResponseEnvelope{Id: response.ID, Result: append([]byte(nil), response.Result...)})
}

func DecodeResponsePayload(payload []byte) (Response, error) {
	var message wirepb.ResponseEnvelope
	if err := proto.Unmarshal(payload, &message); err != nil {
		return Response{}, err
	}
	return Response{ID: message.GetId(), Result: append([]byte(nil), message.GetResult()...)}, nil
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
