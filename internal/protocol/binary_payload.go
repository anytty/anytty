package protocol

import (
	"github.com/anytty/anytty/proto/access/wirepb"
	"google.golang.org/protobuf/proto"
)

// EncodeBinaryResponsePayload 仅保留 transport framing 的二进制 response envelope。
func DecodeBinaryResponsePayload(payload []byte) (uint64, []byte, error) {
	var envelope wirepb.ResponseEnvelope
	if err := proto.Unmarshal(payload, &envelope); err != nil {
		return 0, nil, err
	}
	return envelope.GetId(), append([]byte(nil), envelope.GetResult()...), nil
}
