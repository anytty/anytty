// Package directsignal 实现 Direct embedded signaling 的长度前缀 protobuf framing。
// 本包只拥有 wire 边界与 payload 上限，不解释 identity、有效期、重放或 WebRTC 语义。
package directsignal

import (
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

const maxFrameBytes = 4 << 20

// WriteMessage 以 uint32 big-endian 长度前缀写入一条 deterministic protobuf message。
// 每条 signaling TCP connection 只允许一问一答；调用方负责 deadline、取消和关闭。
func WriteMessage(writer io.Writer, message proto.Message) error {
	if writer == nil || message == nil {
		return fmt.Errorf("direct signaling writer and message are required")
	}
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal direct signaling message: %w", err)
	}
	if len(payload) == 0 || len(payload) > maxFrameBytes {
		return fmt.Errorf("direct signaling payload size %d is invalid", len(payload))
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := writer.Write(header[:]); err != nil {
		return fmt.Errorf("write direct signaling header: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("write direct signaling payload: %w", err)
	}
	return nil
}

// ReadMessage 读取一条有界 protobuf message，并拒绝 unknown field。
// schema 不匹配必须显式失败，不能丢弃字段后继续协商旧路径。
func ReadMessage(reader io.Reader, message proto.Message) error {
	if reader == nil || message == nil {
		return fmt.Errorf("direct signaling reader and message are required")
	}
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return fmt.Errorf("read direct signaling header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > maxFrameBytes {
		return fmt.Errorf("direct signaling payload size %d is invalid", size)
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return fmt.Errorf("read direct signaling payload: %w", err)
	}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, message); err != nil {
		return fmt.Errorf("unmarshal direct signaling message: %w", err)
	}
	if len(message.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("direct signaling message contains unknown fields")
	}
	return nil
}
