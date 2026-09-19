package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/klauspost/compress/zstd"
)

type memoryStream struct {
	frames    chan streamFrame
	done      chan struct{}
	closeOnce sync.Once
}

type streamFrame struct {
	typ     uint8
	payload []byte
}

func newMemoryStream(capacity int) *memoryStream {
	return &memoryStream{frames: make(chan streamFrame, capacity), done: make(chan struct{})}
}

func (stream *memoryStream) Send(typ uint8, payload []byte) error {
	frame := streamFrame{typ: typ, payload: append([]byte(nil), payload...)}
	select {
	case stream.frames <- frame:
		return nil
	case <-stream.done:
		return io.EOF
	}
}

func (stream *memoryStream) OutboundBuffered() uint64 { return 0 }

func (stream *memoryStream) Done() <-chan struct{} { return stream.done }

func (stream *memoryStream) Close() error {
	stream.closeOnce.Do(func() { close(stream.done) })
	return nil
}

func TestDownloadStreamsWithoutWholeFileMemory(t *testing.T) {
	service := newTestService(t)
	content := make([]byte, (1<<20)*2+333)
	for index := range content {
		content[index] = byte(index * 7)
	}
	path := filepath.Join(t.TempDir(), "source.bin")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	stream := newMemoryStream(64)
	defer stream.Close()
	metadata, handle, err := service.OpenDownload(context.Background(), DownloadRequest{Path: path}, 7, stream)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	received := make([]byte, 0, len(content))
	offset := int64(0)
	var bytesSinceAck int64
	deadline := time.After(30 * time.Second)
	for {
		select {
		case frame := <-stream.frames:
			switch frame.typ {
			case wire.TypeFileData:
				data, err := protocol.DecodeFileTransferData(frame.payload)
				if err != nil {
					t.Fatal(err)
				}
				if data.Offset != offset {
					t.Fatalf("data offset = %d, want %d", data.Offset, offset)
				}
				received = append(received, data.Data...)
				offset += int64(len(data.Data))
				bytesSinceAck += int64(len(data.Data))
				if bytesSinceAck == metadata.WindowBytes {
					ack, err := protocol.EncodeFileTransferAck(protocol.FileTransferAck{Offset: offset, WindowBytes: bytesSinceAck})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := handle.HandleFrame(context.Background(), wire.TypeFileAck, ack); err != nil {
						t.Fatal(err)
					}
					bytesSinceAck = 0
				}
			case wire.TypeFileFinish:
				finish, err := protocol.DecodeFileTransferFinish(frame.payload)
				if err != nil {
					t.Fatal(err)
				}
				digest := sha256.Sum256(content)
				if finish.Size != int64(len(content)) || !bytes.Equal(finish.SHA256, digest[:]) {
					t.Fatalf("finish mismatch size=%d", finish.Size)
				}
				if !bytes.Equal(received, content) {
					t.Fatalf("received %d bytes, want %d", len(received), len(content))
				}
				return
			case wire.TypeError:
				t.Fatalf("download stream error: %s", string(frame.payload))
			}
		case <-deadline:
			t.Fatalf("timed out waiting for download finish: %d/%d bytes", offset, len(content))
		}
	}
}

func TestUploadResumeSurvivesServiceReopen(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "transfers")
	target := filepath.Join(dir, "upload.bin")
	content := bytes.Repeat([]byte("resume-across-process-restart-"), 5000)
	service, err := NewService(Config{TransferDir: storeDir})
	if err != nil {
		t.Fatal(err)
	}
	stream := newMemoryStream(8)
	defer stream.Close()
	metadata, handle, err := service.OpenUpload(context.Background(), UploadRequest{Path: target, Size: int64(len(content)), Overwrite: true}, 9, stream, "owner-a")
	if err != nil {
		t.Fatal(err)
	}
	splitAt := int64(96 << 10)
	if err := feedUploadData(handle, content[:splitAt], 0); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewService(Config{TransferDir: storeDir})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	record, ok := reopened.store.Lookup(metadata.ResumeToken)
	if !ok || record.Offset != splitAt {
		t.Fatalf("persisted record = %#v ok=%v", record, ok)
	}
	stream2 := newMemoryStream(8)
	defer stream2.Close()
	resumed, handle2, err := reopened.OpenUpload(context.Background(), UploadRequest{
		Path: target, Size: int64(len(content)), Overwrite: true, ResumeTransferToken: metadata.ResumeToken,
	}, 10, stream2, "owner-b")
	if err != nil {
		t.Fatalf("resume open: %v", err)
	}
	if resumed.Offset != splitAt {
		t.Fatalf("resume offset = %d, want %d", resumed.Offset, splitAt)
	}
	if err := feedUploadData(handle2, content[splitAt:], splitAt); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	finish, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	done, err := handle2.HandleFrame(context.Background(), wire.TypeFileFinish, finish)
	if err != nil || !done {
		t.Fatalf("finish done=%v err=%v", done, err)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, content) {
		t.Fatalf("resumed content mismatch: %d bytes", len(written))
	}
	if reopened.store.Len() != 0 {
		t.Fatalf("completed transfer record survived: %d", reopened.store.Len())
	}
}

func TestCancelResumeRemovesRecordAndTemp(t *testing.T) {
	service := newTestService(t)
	target := filepath.Join(t.TempDir(), "cancel.bin")
	stream := newMemoryStream(4)
	defer stream.Close()
	metadata, handle, err := service.OpenUpload(context.Background(), UploadRequest{Path: target, Size: 1024, Overwrite: true}, 11, stream, "owner")
	if err != nil {
		t.Fatal(err)
	}
	record, _ := service.store.Lookup(metadata.ResumeToken)
	if _, err := os.Stat(record.TempPath); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.CancelResume(metadata.ResumeToken)
	if err != nil || !cancelled {
		t.Fatalf("cancel resume = %v, %v", cancelled, err)
	}
	if _, err := os.Stat(record.TempPath); !os.IsNotExist(err) {
		t.Fatalf("temp file survived cancel: %v", err)
	}
	if _, ok := service.store.Lookup(metadata.ResumeToken); ok {
		t.Fatal("record survived cancel")
	}
}

func TestRootsRejectSymlinkEscapeThroughService(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Config{Resolver: ResolverConfig{Roots: []string{root}}, TransferDir: filepath.Join(base, "transfers")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.Close() }()
	if _, err := service.List(ListRequest{Path: filepath.Join(root, "escape")}); err == nil {
		t.Fatal("list through escaping symlink succeeded")
	}
	stream := newMemoryStream(2)
	defer stream.Close()
	if _, _, err := service.OpenDownload(context.Background(), DownloadRequest{Path: filepath.Join(root, "escape", "secret.txt")}, 7, stream); err == nil {
		t.Fatal("download through escaping symlink succeeded")
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(Config{TransferDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	return service
}

func feedUploadData(handle Handle, content []byte, startOffset int64) error {
	offset := startOffset
	for offset < startOffset+int64(len(content)) {
		chunk := content[offset-startOffset:]
		if len(chunk) > 32<<10 {
			chunk = chunk[:32<<10]
		}
		payload, err := protocol.EncodeFileTransferData(protocol.FileTransferData{Offset: offset, Data: chunk})
		if err != nil {
			return err
		}
		if _, err := handle.HandleFrame(context.Background(), wire.TypeFileData, payload); err != nil {
			return err
		}
		offset += int64(len(chunk))
	}
	return nil
}

// TestDownloadNegotiatesZstdAndStructuredFinish 覆盖可选压缩与耗时：
// 协商后 data frame 携带 per-frame encoding，identity 默认行为不变。
func TestDownloadNegotiatesZstdAndStructuredFinish(t *testing.T) {
	service := newTestService(t)
	content := bytes.Repeat([]byte("compressible-anytty-transfer-payload-"), 20000)
	path := filepath.Join(t.TempDir(), "compress.bin")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	stream := newMemoryStream(64)
	defer stream.Close()
	metadata, handle, err := service.OpenDownload(context.Background(), DownloadRequest{
		Path: path, AcceptCompression: []string{"zstd"}, ProgressIntervalBytes: 256 << 10,
	}, 21, stream)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if metadata.ContentEncoding != "zstd" || metadata.ProgressIntervalBytes != 256<<10 {
		t.Fatalf("negotiated metadata = %#v", metadata)
	}
	time.Sleep(2 * time.Millisecond)

	var received []byte
	var offset int64
	compressedFrames := 0
	deadline := time.After(30 * time.Second)
	for {
		select {
		case frame := <-stream.frames:
			switch frame.typ {
			case wire.TypeFileData:
				data, err := protocol.DecodeFileTransferData(frame.payload)
				if err != nil {
					t.Fatal(err)
				}
				chunk := data.Data
				if data.Encoding == "zstd" {
					compressedFrames++
					decoder, err := zstd.NewReader(nil)
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := decoder.DecodeAll(data.Data, nil)
					decoder.Close()
					if err != nil {
						t.Fatalf("decode compressed chunk: %v", err)
					}
					chunk = decoded
				} else if data.Encoding != "" {
					t.Fatalf("unexpected frame encoding %q", data.Encoding)
				}
				if data.Offset != offset {
					t.Fatalf("offset = %d, want %d", data.Offset, offset)
				}
				received = append(received, chunk...)
				offset += int64(len(chunk))
			case wire.TypeFileFinish:
				if compressedFrames == 0 {
					t.Fatal("download never used negotiated zstd encoding")
				}
				if !bytes.Equal(received, content) {
					t.Fatalf("received %d bytes, want %d", len(received), len(content))
				}
				finish, err := protocol.DecodeFileTransferFinish(frame.payload)
				if err != nil {
					t.Fatal(err)
				}
				if finish.ElapsedMillis < 1 {
					t.Fatalf("structured finish elapsed = %d", finish.ElapsedMillis)
				}
				return
			case wire.TypeError:
				t.Fatalf("download stream error: %s", string(frame.payload))
			}
		case <-deadline:
			t.Fatalf("timed out waiting for compressed download: %d/%d", offset, len(content))
		}
	}
}

// TestUploadAcceptsZstdAndReportsStructuredProgress 覆盖上传方向的协商与进度。
func TestUploadAcceptsZstdAndReportsStructuredProgress(t *testing.T) {
	service := newTestService(t)
	target := filepath.Join(t.TempDir(), "upload-zstd.bin")
	content := bytes.Repeat([]byte("upload-compressible-payload-"), 4000)
	stream := newMemoryStream(64)
	defer stream.Close()
	metadata, handle, err := service.OpenUpload(context.Background(), UploadRequest{
		Path: target, Size: int64(len(content)), Overwrite: true,
		AcceptCompression: []string{"zstd"}, ProgressIntervalBytes: 64 << 10,
	}, 22, stream, "owner-zstd")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ContentEncoding != "zstd" || metadata.ProgressIntervalBytes != 64<<10 {
		t.Fatalf("upload metadata = %#v", metadata)
	}
	time.Sleep(2 * time.Millisecond)
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	var progressAcks int
	offset := int64(0)
	for offset < int64(len(content)) {
		end := offset + 32<<10
		if end > int64(len(content)) {
			end = int64(len(content))
		}
		chunk := content[offset:end]
		payload, err := protocol.EncodeFileTransferData(protocol.FileTransferData{
			Offset: offset, Data: encoder.EncodeAll(chunk, nil), Encoding: "zstd",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := handle.HandleFrame(context.Background(), wire.TypeFileData, payload); err != nil {
			t.Fatal(err)
		}
		frame := <-stream.frames
		if frame.typ != wire.TypeFileAck {
			t.Fatalf("upload frame type = %d, want ack", frame.typ)
		}
		ack, err := protocol.DecodeFileTransferAck(frame.payload)
		if err != nil {
			t.Fatal(err)
		}
		if ack.Offset != offset+int64(len(chunk)) {
			t.Fatalf("ack offset = %d, want %d", ack.Offset, offset+int64(len(chunk)))
		}
		if ack.TransferredBytes > 0 {
			if ack.TotalBytes != int64(len(content)) || ack.ElapsedMillis < 1 {
				t.Fatalf("structured progress = %#v", ack)
			}
			progressAcks++
		}
		offset += int64(len(chunk))
	}
	if progressAcks == 0 {
		t.Fatal("no structured progress ack was emitted")
	}
	digest := sha256.Sum256(content)
	finish, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	done, err := handle.HandleFrame(context.Background(), wire.TypeFileFinish, finish)
	if err != nil || !done {
		t.Fatalf("finish done=%v err=%v", done, err)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, content) {
		t.Fatalf("uploaded content mismatch: %d bytes", len(written))
	}
}

// TestTransferDefaultsStayIdentity 证明未协商时字段保持零值，旧行为不变。
func TestTransferDefaultsStayIdentity(t *testing.T) {
	service := newTestService(t)
	target := filepath.Join(t.TempDir(), "plain.bin")
	stream := newMemoryStream(4)
	defer stream.Close()
	metadata, handle, err := service.OpenUpload(context.Background(), UploadRequest{Path: target, Size: 1024, Overwrite: true}, 23, stream, "owner-plain")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ContentEncoding != "" || metadata.ProgressIntervalBytes != 0 {
		t.Fatalf("default upload metadata = %#v", metadata)
	}
	if err := feedUploadData(handle, bytes.Repeat([]byte{1}, 1024), 0); err != nil {
		t.Fatal(err)
	}
	_ = handle.Close()
}
