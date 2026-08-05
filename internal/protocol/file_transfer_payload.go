package protocol

import (
	"fmt"

	"github.com/anytty/anytty/proto/wirepb"
	"google.golang.org/protobuf/proto"
)

// FileTransferData 携带 transfer channel 上一个有界且有序的数据块。
// Offset 是该块在文件中的绝对字节位置，接收端只确认连续落地的 offset。
type FileTransferData struct {
	Offset int64
	Data   []byte
}

// FileTransferAck 返回当前连续确认位置和发送方可继续占用的窗口字节数。
// WindowBytes 为零时发送方必须停止发送，不能继续缓存整个文件。
type FileTransferAck struct {
	Offset      int64
	WindowBytes int64
}

// FileTransferFinish 声明发送方已完成指定大小与 SHA-256 的数据发送。
// 接收端只有校验 size 和 digest 后才能发布上传文件或确认下载完成。
type FileTransferFinish struct {
	Size   int64
	SHA256 []byte
}

// FileTransferResult 是 owning daemon 对 transfer 最终完成状态的确认。
// 客户端收到该结果前不得把上传标记为完成。
type FileTransferResult struct {
	Path   string
	Size   int64
	SHA256 []byte
}

// EncodeFileTransferData 编码文件数据 frame payload。
func EncodeFileTransferData(value FileTransferData) ([]byte, error) {
	return proto.Marshal(&wirepb.FileTransferData{Offset: value.Offset, Data: append([]byte(nil), value.Data...)})
}

// DecodeFileTransferData 解码文件数据 frame payload，并复制 data 避免复用 transport 缓冲区。
func DecodeFileTransferData(payload []byte) (FileTransferData, error) {
	var msg wirepb.FileTransferData
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return FileTransferData{}, err
	}
	return FileTransferData{Offset: msg.GetOffset(), Data: append([]byte(nil), msg.GetData()...)}, nil
}

// EncodeFileTransferAck 编码连续确认位置与接收窗口。
func EncodeFileTransferAck(value FileTransferAck) ([]byte, error) {
	return proto.Marshal(&wirepb.FileTransferAck{Offset: value.Offset, WindowBytes: value.WindowBytes})
}

// DecodeFileTransferAck 解码连续确认位置与接收窗口，负数值由 transfer 状态机拒绝。
func DecodeFileTransferAck(payload []byte) (FileTransferAck, error) {
	var msg wirepb.FileTransferAck
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return FileTransferAck{}, err
	}
	return FileTransferAck{Offset: msg.GetOffset(), WindowBytes: msg.GetWindowBytes()}, nil
}

// EncodeFileTransferFinish 编码最终 size 与 SHA-256；digest 长度必须是 32 字节。
func EncodeFileTransferFinish(value FileTransferFinish) ([]byte, error) {
	if len(value.SHA256) != 32 {
		return nil, fmt.Errorf("file transfer sha256 must be 32 bytes")
	}
	return proto.Marshal(&wirepb.FileTransferFinish{Size: value.Size, Sha256: append([]byte(nil), value.SHA256...)})
}

// DecodeFileTransferFinish 解码最终声明并拒绝非 SHA-256 长度的摘要。
func DecodeFileTransferFinish(payload []byte) (FileTransferFinish, error) {
	var msg wirepb.FileTransferFinish
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return FileTransferFinish{}, err
	}
	if len(msg.GetSha256()) != 32 {
		return FileTransferFinish{}, fmt.Errorf("file transfer sha256 must be 32 bytes")
	}
	return FileTransferFinish{Size: msg.GetSize(), SHA256: append([]byte(nil), msg.GetSha256()...)}, nil
}

// EncodeFileTransferResult 编码 daemon 已校验完成的 transfer 结果。
func EncodeFileTransferResult(value FileTransferResult) ([]byte, error) {
	if len(value.SHA256) != 32 {
		return nil, fmt.Errorf("file transfer sha256 must be 32 bytes")
	}
	return proto.Marshal(&wirepb.FileTransferResult{Path: value.Path, Size: value.Size, Sha256: append([]byte(nil), value.SHA256...)})
}

// DecodeFileTransferResult 解码 daemon 最终确认并复制摘要 bytes。
func DecodeFileTransferResult(payload []byte) (FileTransferResult, error) {
	var msg wirepb.FileTransferResult
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return FileTransferResult{}, err
	}
	if len(msg.GetSha256()) != 32 {
		return FileTransferResult{}, fmt.Errorf("file transfer sha256 must be 32 bytes")
	}
	return FileTransferResult{Path: msg.GetPath(), Size: msg.GetSize(), SHA256: append([]byte(nil), msg.GetSha256()...)}, nil
}
