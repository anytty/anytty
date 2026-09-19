// Package e2etest 为 adapter 端到端测试提供共享的 application-session 文件传输
// 驱动逻辑。它不是生产代码：只有 access/engine/adapter 下的测试可以导入。
package e2etest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
)

// TransferTimeout 限制单个 transfer 步骤的等待时间，保持 e2e 测试确定性有界。
const TransferTimeout = 10 * time.Second

// ApplicationSession 是 FileUploadOpen/FileDownloadOpen + resource stream 的最小面。
// *direct.Session、*ssh.Session 与 *protocoladapter.ApplicationClient 都满足它。
type ApplicationSession interface {
	FileUploadOpen(context.Context, *apipb.FileUploadOpenCommand) (*apipb.FileTransferOpenResult, error)
	FileDownloadOpen(context.Context, *apipb.FileDownloadOpenCommand) (*apipb.FileTransferOpenResult, error)
	OpenResourceStream(*apipb.ResourceHandle) (clientruntime.ResourceStream, error)
}

// UploadFile 通过当前 session 上传 content，逐个 chunk 断言服务端 ack offset，
// 并在 finish 后校验服务端返回的 size/SHA-256。
func UploadFile(t *testing.T, ctx context.Context, session ApplicationSession, path string, content []byte) {
	t.Helper()
	opened, err := session.FileUploadOpen(ctx, &apipb.FileUploadOpenCommand{Path: path, Size: int64(len(content)), Overwrite: true})
	if err != nil {
		t.Fatalf("upload open %s: %v", path, err)
	}
	handle := opened.GetTransfer()
	if handle.GetResource() == nil || handle.GetChunkBytes() <= 0 || handle.GetWindowBytes() <= 0 {
		t.Fatalf("upload handle = %#v", handle)
	}
	stream, err := session.OpenResourceStream(handle.GetResource())
	if err != nil {
		t.Fatalf("open upload stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	chunkBytes := int64(handle.GetChunkBytes())
	offset := int64(0)
	for offset < int64(len(content)) {
		end := offset + chunkBytes
		if end > int64(len(content)) {
			end = int64(len(content))
		}
		payload, err := internalprotocol.EncodeFileTransferData(internalprotocol.FileTransferData{Offset: offset, Data: content[offset:end]})
		if err != nil {
			t.Fatal(err)
		}
		if err := stream.Send(ctx, wire.TypeFileData, payload); err != nil {
			t.Fatalf("send upload data at %d: %v", offset, err)
		}
		typ, payload, err := receiveFrame(ctx, stream)
		if err != nil {
			t.Fatalf("receive upload ack at %d: %v", offset, err)
		}
		if typ != wire.TypeFileAck {
			t.Fatalf("upload frame type = %d at offset %d, want ack", typ, offset)
		}
		ack, err := internalprotocol.DecodeFileTransferAck(payload)
		if err != nil {
			t.Fatal(err)
		}
		if ack.Offset != end {
			t.Fatalf("upload ack offset = %d, want %d", ack.Offset, end)
		}
		offset = ack.Offset
	}
	digest := sha256.Sum256(content)
	finish, err := internalprotocol.EncodeFileTransferFinish(internalprotocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(ctx, wire.TypeFileFinish, finish); err != nil {
		t.Fatalf("send upload finish: %v", err)
	}
	typ, payload, err := receiveFrame(ctx, stream)
	if err != nil {
		t.Fatalf("receive upload result: %v", err)
	}
	if typ != wire.TypeFileResult {
		t.Fatalf("upload completion frame type = %d, want result", typ)
	}
	result, err := internalprotocol.DecodeFileTransferResult(payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.Size != int64(len(content)) || !bytes.Equal(result.SHA256, digest[:]) {
		t.Fatalf("upload result size=%d sha256=%x, want size=%d", result.Size, result.SHA256, len(content))
	}
}

// DownloadFile 通过当前 session 下载 path，按 window 语义补发 ack，
// 在 finish 时校验 size/SHA-256 与完整内容。
func DownloadFile(t *testing.T, ctx context.Context, session ApplicationSession, path string, want []byte) {
	t.Helper()
	opened, err := session.FileDownloadOpen(ctx, &apipb.FileDownloadOpenCommand{Path: path})
	if err != nil {
		t.Fatalf("download open %s: %v", path, err)
	}
	handle := opened.GetTransfer()
	if handle.GetResource() == nil || handle.GetWindowBytes() <= 0 {
		t.Fatalf("download handle = %#v", handle)
	}
	stream, err := session.OpenResourceStream(handle.GetResource())
	if err != nil {
		t.Fatalf("open download stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	windowBytes := int64(handle.GetWindowBytes())
	received := make([]byte, 0, len(want))
	offset := int64(0)
	var bytesSinceAck int64
	for {
		typ, payload, err := receiveFrame(ctx, stream)
		if err != nil {
			t.Fatalf("receive download frame at %d: %v", offset, err)
		}
		switch typ {
		case wire.TypeFileData:
			data, err := internalprotocol.DecodeFileTransferData(payload)
			if err != nil {
				t.Fatal(err)
			}
			if data.Offset != offset {
				t.Fatalf("download offset = %d, want %d", data.Offset, offset)
			}
			received = append(received, data.Data...)
			offset += int64(len(data.Data))
			bytesSinceAck += int64(len(data.Data))
			if bytesSinceAck >= windowBytes {
				ack, err := internalprotocol.EncodeFileTransferAck(internalprotocol.FileTransferAck{Offset: offset, WindowBytes: bytesSinceAck})
				if err != nil {
					t.Fatal(err)
				}
				if err := stream.Send(ctx, wire.TypeFileAck, ack); err != nil {
					t.Fatalf("send download ack at %d: %v", offset, err)
				}
				bytesSinceAck = 0
			}
		case wire.TypeFileFinish:
			digest := sha256.Sum256(want)
			finish, err := internalprotocol.DecodeFileTransferFinish(payload)
			if err != nil {
				t.Fatal(err)
			}
			if finish.Size != int64(len(want)) || !bytes.Equal(finish.SHA256, digest[:]) {
				t.Fatalf("download finish size=%d sha256=%x, want size=%d", finish.Size, finish.SHA256, len(want))
			}
			if !bytes.Equal(received, want) {
				t.Fatalf("download content mismatch: %d bytes vs %d", len(received), len(want))
			}
			return
		case wire.TypeError:
			t.Fatalf("download stream error: %s", string(payload))
		default:
			t.Fatalf("download frame type = %d, want file data/finish", typ)
		}
	}
}

func receiveFrame(ctx context.Context, stream clientruntime.ResourceStream) (uint8, []byte, error) {
	receiveCtx, cancel := context.WithTimeout(ctx, TransferTimeout)
	defer cancel()
	typ, payload, err := stream.Receive(receiveCtx)
	if err != nil {
		return 0, nil, err
	}
	if typ == wire.TypeError {
		return 0, nil, errors.New(string(payload))
	}
	return typ, payload, nil
}
