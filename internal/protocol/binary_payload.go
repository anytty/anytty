package protocol

import (
	"github.com/anytty/anytty/proto/wirepb"
	"google.golang.org/protobuf/proto"
)

// EncodeBinaryResponsePayload 仅保留 transport framing 的二进制 response envelope。
func EncodeBinaryResponsePayload(id uint64, result []byte) ([]byte, error) {
	return proto.Marshal(&wirepb.ResponseEnvelope{Id: id, Result: append([]byte(nil), result...)})
}

// DecodeBinaryResponsePayload 解码 transport framing 的二进制 response envelope。
func DecodeBinaryResponsePayload(payload []byte) (uint64, []byte, error) {
	var envelope wirepb.ResponseEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return 0, nil, err
	}
	return envelope.GetId(), append([]byte(nil), envelope.GetResult()...), nil
}
