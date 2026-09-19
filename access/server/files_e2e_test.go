package server_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
	"github.com/klauspost/compress/zstd"
)

// TestFileCommandSurfaceThroughAccess 覆盖 Phase 2 文件 metadata 全命令：
// list/stat/preview/mkdir/rename/delete/copy/move 全部在 access 本地终结。
func TestFileCommandSurfaceThroughAccess(t *testing.T) {
	t.Parallel()
	accessSocket := filepath.Join(t.TempDir(), "access.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(context.Context) (terminalprovider.Provider, error) {
			return nil, errors.New("provider must not be dialed for file commands")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	application, _, closeClient := dialAccessClient(t, ctx, accessSocket)
	defer closeClient()
	root := t.TempDir()
	hello := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(hello, []byte("hello-file-surface"), 0o600); err != nil {
		t.Fatal(err)
	}

	list, err := application.FileList(ctx, &apipb.FileListCommand{Path: root})
	if err != nil {
		t.Fatalf("file list: %v", err)
	}
	if len(list.GetEntries()) != 1 || list.GetEntries()[0].GetName() != "hello.txt" {
		t.Fatalf("file list = %#v", list.GetEntries())
	}
	stat, err := application.FileStat(ctx, &apipb.FileStatCommand{Path: hello})
	if err != nil {
		t.Fatalf("file stat: %v", err)
	}
	if stat.GetEntry().GetType() != apipb.FileEntryType_FILE_ENTRY_TYPE_FILE || stat.GetEntry().GetSize() != int64(len("hello-file-surface")) {
		t.Fatalf("file stat = %#v", stat.GetEntry())
	}
	preview, err := application.FilePreview(ctx, &apipb.FilePreviewCommand{Path: hello})
	if err != nil {
		t.Fatalf("file preview: %v", err)
	}
	if string(preview.GetContent()) != "hello-file-surface" || preview.GetTruncated() {
		t.Fatalf("file preview = %#v", preview)
	}

	subdir := filepath.Join(root, "sub")
	if result, err := application.FileMkdir(ctx, &apipb.FileMkdirCommand{Path: subdir}); err != nil {
		t.Fatalf("file mkdir: %v", err)
	} else if !result.GetSuccess() {
		t.Fatalf("file mkdir result = %#v", result)
	}
	renamed := filepath.Join(root, "renamed.txt")
	if result, err := application.FileRename(ctx, &apipb.FileRenameCommand{Path: hello, NewPath: renamed}); err != nil {
		t.Fatalf("file rename: %v", err)
	} else if !result.GetSuccess() {
		t.Fatalf("file rename result = %#v", result)
	}
	if result, err := application.FileCopy(ctx, &apipb.FileCopyCommand{Paths: []string{renamed}, TargetDirectory: subdir}); err != nil {
		t.Fatalf("file copy: %v", err)
	} else if len(result.GetResults()) != 1 || !result.GetResults()[0].GetSuccess() {
		t.Fatalf("file copy result = %#v", result)
	}
	copied := filepath.Join(subdir, "renamed.txt")
	if result, err := application.FileMove(ctx, &apipb.FileMoveCommand{Paths: []string{copied}, TargetDirectory: root, Overwrite: true}); err != nil {
		t.Fatalf("file move: %v", err)
	} else if len(result.GetResults()) != 1 || !result.GetResults()[0].GetSuccess() {
		t.Fatalf("file move result = %#v", result)
	}
	if result, err := application.FileDelete(ctx, &apipb.FileDeleteCommand{Path: filepath.Join(root, "renamed.txt")}); err != nil {
		t.Fatalf("file delete: %v", err)
	} else if !result.GetSuccess() {
		t.Fatalf("file delete result = %#v", result)
	}
	if _, err := application.FileStat(ctx, &apipb.FileStatCommand{Path: hello}); err == nil {
		t.Fatal("file stat after delete unexpectedly succeeded")
	}
}

// TestFileTransferResumeUploadAndDownload 覆盖 Phase 2 transfer 全链路：
// 上传→release 保留断点→新 session 续传→下载校验内容。
func TestFileTransferResumeUploadAndDownload(t *testing.T) {
	t.Parallel()
	accessSocket := filepath.Join(t.TempDir(), "access.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(context.Context) (terminalprovider.Provider, error) {
			return nil, errors.New("provider must not be dialed for file transfers")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	target := filepath.Join(t.TempDir(), "upload.bin")
	content := make([]byte, 2<<20+12345)
	for index := range content {
		content[index] = byte(index * 31)
	}
	splitAt := int64(64 << 10)
	resumeToken := uploadFirstHalf(t, ctx, accessSocket, target, content, splitAt)
	uploadSecondHalf(t, ctx, accessSocket, target, content, splitAt, resumeToken)
	uploaded, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(uploaded, content) {
		t.Fatalf("uploaded content mismatch: %d bytes vs %d", len(uploaded), len(content))
	}
	downloadAndVerify(t, ctx, accessSocket, target, content)
}

// TestBrowserProxyThroughAccess 覆盖 Phase 2 proxy 迁移：access 本机拨号并转发字节。
func TestBrowserProxyThroughAccess(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer connection.Close()
				buffer := make([]byte, 32<<10)
				for {
					n, err := connection.Read(buffer)
					if n > 0 {
						if _, writeErr := connection.Write(buffer[:n]); writeErr != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port

	accessSocket := filepath.Join(t.TempDir(), "access.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(context.Context) (terminalprovider.Provider, error) {
			return nil, errors.New("provider must not be dialed for browser proxy")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	application, client, closeClient := dialAccessClient(t, ctx, accessSocket)
	defer closeClient()
	opened, err := application.BrowserProxyOpen(ctx, &apipb.BrowserProxyOpenCommand{
		Host: "127.0.0.1", Port: uint32(port), ReceiveWindowBytes: 64 << 10, SendWindowBytes: 64 << 10,
	})
	if err != nil {
		t.Fatalf("browser proxy open: %v", err)
	}
	channel, ok := client.ApplicationResourceChannel(opened.GetResource())
	if !ok {
		t.Fatal("browser proxy resource was not bound to a channel")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	payload := []byte("browser-proxy-e2e")
	if err := client.SendBrowserFrame(channel, wire.TypeBrowserData, payload); err != nil {
		t.Fatal(err)
	}
	var echo []byte
	acked := false
	deadline := time.After(5 * time.Second)
	for !acked || echo == nil {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("browser stream closed")
			}
			switch frame.Type {
			case wire.TypeBrowserData:
				echo = append([]byte(nil), frame.Payload...)
			case wire.TypeFileAck:
				ack, err := internalprotocol.DecodeFileTransferAck(frame.Payload)
				if err != nil {
					t.Fatal(err)
				}
				if ack.Offset == int64(len(payload)) {
					acked = true
				}
			case wire.TypeError:
				t.Fatalf("browser stream error: %s", string(frame.Payload))
			}
		case <-deadline:
			t.Fatalf("timed out waiting for browser echo/ack: echo=%q acked=%v", echo, acked)
		}
	}
	if !bytes.Equal(echo, payload) {
		t.Fatalf("browser proxy echo = %q, want %q", echo, payload)
	}
	if err := application.ReleaseResource(ctx, &apipb.ReleaseResourceCommand{Resource: opened.GetResource()}); err != nil {
		t.Fatalf("release browser proxy: %v", err)
	}
}

func uploadFirstHalf(t *testing.T, ctx context.Context, socket, target string, content []byte, splitAt int64) []byte {
	t.Helper()
	application, client, closeClient := dialAccessClient(t, ctx, socket)
	defer closeClient()
	opened, err := application.FileUploadOpen(ctx, &apipb.FileUploadOpenCommand{Path: target, Size: int64(len(content)), Overwrite: true})
	if err != nil {
		t.Fatalf("upload open: %v", err)
	}
	handle := opened.GetTransfer()
	channel, ok := client.ApplicationResourceChannel(handle.GetResource())
	if !ok {
		t.Fatal("upload resource was not bound to a channel")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	offset := int64(0)
	for offset < splitAt {
		chunk := int64(handle.GetChunkBytes())
		if remaining := splitAt - offset; chunk > remaining {
			chunk = remaining
		}
		sendUploadChunk(t, client, frames, channel, handle, offset, content[offset:offset+chunk])
		offset += chunk
	}
	resumeToken := append([]byte(nil), handle.GetResume().GetOpaqueToken()...)
	if err := application.ReleaseResource(ctx, &apipb.ReleaseResourceCommand{Resource: handle.GetResource()}); err != nil {
		t.Fatalf("release upload: %v", err)
	}
	return resumeToken
}

func uploadSecondHalf(t *testing.T, ctx context.Context, socket, target string, content []byte, splitAt int64, resumeToken []byte) {
	t.Helper()
	application, client, closeClient := dialAccessClient(t, ctx, socket)
	defer closeClient()
	opened, err := application.FileUploadOpen(ctx, &apipb.FileUploadOpenCommand{
		Path: target, Size: int64(len(content)), Overwrite: true,
		Resume: &apipb.FileUploadResumeHandle{OpaqueToken: resumeToken},
	})
	if err != nil {
		t.Fatalf("upload resume open: %v", err)
	}
	handle := opened.GetTransfer()
	if handle.GetOffset() != splitAt {
		t.Fatalf("resume offset = %d, want %d", handle.GetOffset(), splitAt)
	}
	channel, ok := client.ApplicationResourceChannel(handle.GetResource())
	if !ok {
		t.Fatal("resumed upload resource was not bound to a channel")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	offset := handle.GetOffset()
	for offset < int64(len(content)) {
		chunk := int64(handle.GetChunkBytes())
		if remaining := int64(len(content)) - offset; chunk > remaining {
			chunk = remaining
		}
		sendUploadChunk(t, client, frames, channel, handle, offset, content[offset:offset+chunk])
		offset += chunk
	}
	digest := sha256.Sum256(content)
	finish, err := internalprotocol.EncodeFileTransferFinish(internalprotocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SendFileFrame(channel, wire.TypeFileFinish, finish); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("upload stream closed before result")
			}
			if frame.Type == wire.TypeFileResult {
				return
			}
			if frame.Type == wire.TypeError {
				t.Fatalf("upload stream error: %s", string(frame.Payload))
			}
		case <-deadline:
			t.Fatal("timed out waiting for upload result")
		}
	}
}

func sendUploadChunk(t *testing.T, client *internalprotocol.Client, frames <-chan internalprotocol.StreamFrame, channel uint16, handle *apipb.FileTransferHandle, offset int64, chunk []byte) {
	t.Helper()
	payload, err := internalprotocol.EncodeFileTransferData(internalprotocol.FileTransferData{Offset: offset, Data: chunk})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SendFileFrame(channel, wire.TypeFileData, payload); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("upload stream closed before ack")
			}
			if frame.Type == wire.TypeError {
				t.Fatalf("upload stream error: %s", string(frame.Payload))
			}
			if frame.Type != wire.TypeFileAck {
				t.Fatalf("upload frame type = %d, want ack", frame.Type)
			}
			ack, err := internalprotocol.DecodeFileTransferAck(frame.Payload)
			if err != nil {
				t.Fatal(err)
			}
			if ack.Offset != offset+int64(len(chunk)) {
				t.Fatalf("upload ack offset = %d, want %d", ack.Offset, offset+int64(len(chunk)))
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for upload ack")
		}
	}
}

func downloadAndVerify(t *testing.T, ctx context.Context, socket, path string, content []byte) {
	t.Helper()
	application, client, closeClient := dialAccessClient(t, ctx, socket)
	defer closeClient()
	opened, err := application.FileDownloadOpen(ctx, &apipb.FileDownloadOpenCommand{Path: path})
	if err != nil {
		t.Fatalf("download open: %v", err)
	}
	handle := opened.GetTransfer()
	channel, ok := client.ApplicationResourceChannel(handle.GetResource())
	if !ok {
		t.Fatal("download resource was not bound to a channel")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	received := make([]byte, 0, len(content))
	offset := int64(0)
	var bytesSinceAck int64
	deadline := time.After(30 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				t.Fatal("download stream closed before finish")
			}
			switch frame.Type {
			case wire.TypeFileData:
				data, err := internalprotocol.DecodeFileTransferData(frame.Payload)
				if err != nil {
					t.Fatal(err)
				}
				if data.Offset != offset {
					t.Fatalf("download offset = %d, want %d", data.Offset, offset)
				}
				received = append(received, data.Data...)
				offset += int64(len(data.Data))
				bytesSinceAck += int64(len(data.Data))
				if bytesSinceAck == handle.GetWindowBytes() {
					ack, err := internalprotocol.EncodeFileTransferAck(internalprotocol.FileTransferAck{Offset: offset, WindowBytes: bytesSinceAck})
					if err != nil {
						t.Fatal(err)
					}
					if err := client.SendFileFrame(channel, wire.TypeFileAck, ack); err != nil {
						t.Fatal(err)
					}
					bytesSinceAck = 0
				}
			case wire.TypeFileFinish:
				digest := sha256.Sum256(content)
				finish, err := internalprotocol.DecodeFileTransferFinish(frame.Payload)
				if err != nil {
					t.Fatal(err)
				}
				if finish.Size != int64(len(content)) || !bytes.Equal(finish.SHA256, digest[:]) {
					t.Fatalf("download finish mismatch: size=%d", finish.Size)
				}
				if !bytes.Equal(received, content) {
					t.Fatalf("download content mismatch: %d bytes vs %d", len(received), len(content))
				}
				return
			case wire.TypeError:
				t.Fatalf("download stream error: %s", string(frame.Payload))
			}
		case <-deadline:
			t.Fatal("timed out waiting for download finish")
		}
	}
}

// dialAccessClient 建立一条 access 客户端连接，并返回 application 与 raw framing client。
func dialAccessClient(t *testing.T, ctx context.Context, socket string) (*clientruntime.ApplicationSession, *internalprotocol.Client, func()) {
	t.Helper()
	connection, err := unixtransport.DialContext(ctx, socket)
	if err != nil {
		t.Fatal(err)
	}
	client := internalprotocol.NewClient(connection)
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "files-e2e"}); err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"),
		RouteID:    clientendpoint.RouteID("files-e2e"),
		Generation: 1,
	}, client)
	if err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	return application, client, func() { _ = client.Close() }
}

// TestFileTransferNegotiatesCompressionAndProgressThroughAccess 覆盖 Phase 4
// 可选字段在 access wire 上的端到端行为：协商 zstd + 结构化进度，旧客户端
// 不设置字段时行为不变。
func TestFileTransferNegotiatesCompressionAndProgressThroughAccess(t *testing.T) {
	t.Parallel()
	accessSocket := filepath.Join(t.TempDir(), "access.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	access, err := accessserver.New(accessserver.Config{
		Socket: accessSocket,
		Files:  accessserverTestFiles(t),
		Provider: func(context.Context) (terminalprovider.Provider, error) {
			return nil, errors.New("provider must not be dialed for file transfers")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = access.Close() }()
	go func() { _ = access.Serve(ctx) }()
	waitForSocket(t, accessSocket)

	application, client, closeClient := dialAccessClient(t, ctx, accessSocket)
	defer closeClient()
	target := filepath.Join(t.TempDir(), "zstd-upload.bin")
	content := bytes.Repeat([]byte("access-wire-compressible-payload-"), 5000)
	opened, err := application.FileUploadOpen(ctx, &apipb.FileUploadOpenCommand{
		Path: target, Size: int64(len(content)), Overwrite: true,
		AcceptCompression: []string{"zstd"}, ProgressIntervalBytes: 64 << 10,
	})
	if err != nil {
		t.Fatalf("upload open: %v", err)
	}
	handle := opened.GetTransfer()
	if handle.GetContentEncoding() != "zstd" || handle.GetProgressIntervalBytes() != 64<<10 {
		t.Fatalf("negotiated handle = %#v", handle)
	}
	channel, ok := client.ApplicationResourceChannel(handle.GetResource())
	if !ok {
		t.Fatal("upload resource was not bound")
	}
	frames, stop := client.Stream(channel)
	defer stop()
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	progressAcks := 0
	offset := int64(0)
	for offset < int64(len(content)) {
		end := offset + int64(handle.GetChunkBytes())
		if end > int64(len(content)) {
			end = int64(len(content))
		}
		chunk := content[offset:end]
		payload, err := internalprotocol.EncodeFileTransferData(internalprotocol.FileTransferData{
			Offset: offset, Data: encoder.EncodeAll(chunk, nil), Encoding: "zstd",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := client.SendFileFrame(channel, wire.TypeFileData, payload); err != nil {
			t.Fatal(err)
		}
		select {
		case frame := <-frames:
			if frame.Type != wire.TypeFileAck {
				t.Fatalf("upload frame type = %d, want ack", frame.Type)
			}
			ack, err := internalprotocol.DecodeFileTransferAck(frame.Payload)
			if err != nil {
				t.Fatal(err)
			}
			if ack.Offset != end {
				t.Fatalf("ack offset = %d, want %d", ack.Offset, end)
			}
			if ack.TransferredBytes > 0 {
				if ack.TotalBytes != int64(len(content)) {
					t.Fatalf("progress ack = %#v", ack)
				}
				progressAcks++
			}
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for upload ack")
		}
		offset = end
	}
	if progressAcks == 0 {
		t.Fatal("structured progress ack was not negotiated")
	}
	digest := sha256.Sum256(content)
	finish, err := internalprotocol.EncodeFileTransferFinish(internalprotocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SendFileFrame(channel, wire.TypeFileFinish, finish); err != nil {
		t.Fatal(err)
	}
	if err := waitForFrameType(frames, wire.TypeFileResult); err != nil {
		t.Fatal(err)
	}
	// 同一文件下载：协商 zstd 后 data frame 携带 encoding，内容一致。
	downloaded, err := application.FileDownloadOpen(ctx, &apipb.FileDownloadOpenCommand{
		Path: target, AcceptCompression: []string{"zstd"}, ProgressIntervalBytes: 128 << 10,
	})
	if err != nil {
		t.Fatalf("download open: %v", err)
	}
	downloadHandle := downloaded.GetTransfer()
	if downloadHandle.GetContentEncoding() != "zstd" {
		t.Fatalf("download negotiated encoding = %q", downloadHandle.GetContentEncoding())
	}
	downloadChannel, ok := client.ApplicationResourceChannel(downloadHandle.GetResource())
	if !ok {
		t.Fatal("download resource was not bound")
	}
	downloadFrames, stopDownload := client.Stream(downloadChannel)
	defer stopDownload()
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	var received []byte
	downloadOffset := int64(0)
	compressedFrames := 0
	for {
		select {
		case frame := <-downloadFrames:
			switch frame.Type {
			case wire.TypeFileData:
				data, err := internalprotocol.DecodeFileTransferData(frame.Payload)
				if err != nil {
					t.Fatal(err)
				}
				chunk := data.Data
				if data.Encoding == "zstd" {
					compressedFrames++
					decoded, err := decoder.DecodeAll(data.Data, nil)
					if err != nil {
						t.Fatal(err)
					}
					chunk = decoded
				}
				if data.Offset != downloadOffset {
					t.Fatalf("download offset = %d, want %d", data.Offset, downloadOffset)
				}
				received = append(received, chunk...)
				downloadOffset += int64(len(chunk))
			case wire.TypeFileFinish:
				if compressedFrames == 0 {
					t.Fatal("download did not use negotiated compression")
				}
				if !bytes.Equal(received, content) {
					t.Fatalf("downloaded content mismatch: %d bytes", len(received))
				}
				return
			case wire.TypeError:
				t.Fatalf("download stream error: %s", string(frame.Payload))
			}
		case <-time.After(30 * time.Second):
			t.Fatal("timed out waiting for compressed download")
		}
	}
}

func waitForFrameType(frames <-chan internalprotocol.StreamFrame, want uint8) error {
	deadline := time.After(10 * time.Second)
	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				return errors.New("stream closed before expected frame")
			}
			if frame.Type == want {
				return nil
			}
			if frame.Type == wire.TypeError {
				return errors.New(string(frame.Payload))
			}
		case <-deadline:
			return errors.New("timed out waiting for stream frame")
		}
	}
}
