package files

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/wire"
)

// BenchmarkFileDownloadStreaming 测量 64MiB 文件经 transfer 管线的服务端吞吐。
// 与 BenchmarkFileCopyBaseline 对比可验证 ≥80% 目标；分片发送保证内存与文件大小解耦。
func BenchmarkFileDownloadStreaming(b *testing.B) {
	path := benchFile(b, 64<<20)
	service, err := NewService(Config{TransferDir: b.TempDir()})
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = service.Close() }()
	b.SetBytes(64 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream := newMemoryStream(64)
		metadata, handle, err := service.OpenDownload(context.Background(), DownloadRequest{Path: path}, uint16(7+i%1000), stream)
		if err != nil {
			b.Fatal(err)
		}
		consumed := make(chan struct{})
		go func() {
			defer close(consumed)
			var since, offset int64
			for {
				select {
				case frame := <-stream.frames:
					switch frame.typ {
					case wire.TypeFileData:
						// 只计服务端管线吞吐：按协议分片规则推导 offset，不重复解码 payload。
						length := int64(metadata.ChunkBytes)
						if remaining := metadata.Size - offset; length > remaining {
							length = remaining
						}
						offset += length
						since += length
						if since >= metadata.WindowBytes {
							ack, encodeErr := protocol.EncodeFileTransferAck(protocol.FileTransferAck{Offset: offset, WindowBytes: since})
							if encodeErr != nil {
								return
							}
							if _, ackErr := handle.HandleFrame(context.Background(), wire.TypeFileAck, ack); ackErr != nil {
								return
							}
							since = 0
						}
					case wire.TypeFileFinish, wire.TypeError:
						return
					}
				case <-handle.Done():
					return
				}
			}
		}()
		<-handle.Done()
		<-consumed
		_ = stream.Close()
		_ = handle.Close()
	}
}

// BenchmarkFileCopyBaseline 是与协议等价的基线：整文件读取并计算 SHA-256
// （finish frame 必须携带整文件摘要；裸 io.Copy 不构成公平下限）。
func BenchmarkFileCopyBaseline(b *testing.B) {
	path := benchFile(b, 64<<20)
	b.SetBytes(64 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			b.Fatal(err)
		}
		_ = hasher.Sum(nil)
		_ = file.Close()
	}
}

func benchFile(b *testing.B, size int) string {
	b.Helper()
	path := filepath.Join(b.TempDir(), "bench.bin")
	content := make([]byte, size)
	for index := range content {
		content[index] = byte(index)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		b.Fatal(err)
	}
	return path
}
