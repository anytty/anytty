package files

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/access/files/transfer"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/klauspost/compress/zstd"
)

const (
	// defaultUploadChunkLimit 是历史客户端（含已上架 Flutter）可能发送的上传分片硬上限。
	// 广告的 ChunkBytes 可以更小，但校验必须接受历史值。
	defaultUploadChunkLimit = 64 << 10
	uploadResumeTTL         = 15 * time.Minute
	uploadPersistInterval   = 500 * time.Millisecond
	outboundQueueTarget     = 256 << 10
	// compressionZstd 是当前唯一协商的 data frame 编码；空字符串始终表示 identity。
	compressionZstd = "zstd"
	// maxUploadDecodeBytes 限制单帧解压上限，避免恶意压缩帧放大内存。
	maxUploadDecodeBytes = 64 << 20
)

// Stream 是 access session 提供给本地 transfer 的帧通道。
type Stream interface {
	Send(typ uint8, payload []byte) error
	OutboundBuffered() uint64
	Done() <-chan struct{}
}

// Handle 是一个已打开本地 transfer 的 session 侧句柄。
// HandleFrame 返回 done=true 表示已经到达终态，调用方必须回收 binding。
type Handle interface {
	HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error)
	Done() <-chan struct{}
	Close() error
}

var _ Handle = (*download)(nil)
var _ Handle = (*upload)(nil)

type download struct {
	service *Service
	id      string
	channel uint16
	path    string
	file    *os.File
	size    int64
	offset  int64

	windowBytes int64
	chunkBytes  int

	encoding         string
	compressor       *zstd.Encoder
	progressInterval int64
	startedAt        time.Time

	stream Stream
	ctx    context.Context
	cancel context.CancelFunc
	ack    chan protocol.FileTransferAck
	done   chan struct{}

	closeOnce sync.Once
	sendMu    sync.Mutex
	progress  *transfer.Coalescer
}

type upload struct {
	service   *Service
	id        string
	channel   uint16
	path      string
	tempPath  string
	overwrite bool
	owner     string
	size      int64

	stream Stream
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	closeOnce        sync.Once
	sendMu           sync.Mutex
	mu               sync.Mutex
	file             *os.File
	hasher           hash.Hash
	offset           int64
	completed        bool
	lastSave         time.Time
	chunkHint        int
	progress         *transfer.Coalescer
	decoder          *zstd.Decoder
	contentEncoding  string
	progressInterval int64
	lastProgress     int64
	startedAt        time.Time
}

// OpenDownload 打开可续传下载：源文件 identity 在打开时固定，之后按
// 窗口/分片流式发送，不整文件驻留内存。
func (service *Service) OpenDownload(ctx context.Context, request DownloadRequest, channel uint16, stream Stream) (*Transfer, Handle, error) {
	path, err := service.resolver.Resolve(request.Path)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("download requires a regular file")
	}
	if request.Offset < 0 || request.Offset > info.Size() {
		return nil, nil, fmt.Errorf("invalid download offset")
	}
	if request.ExpectedSize > 0 && request.ExpectedSize != info.Size() {
		return nil, nil, fmt.Errorf("stale download source")
	}
	if !request.ExpectedModifiedAt.IsZero() && request.ExpectedModifiedAt.UnixNano() != info.ModTime().UnixNano() {
		return nil, nil, fmt.Errorf("stale download source")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	if _, err := file.Seek(request.Offset, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	id, err := newTransferID()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	policy := service.estimator.policy()
	encoding := chooseContentEncoding(request.AcceptCompression)
	transferCtx, cancel := context.WithCancel(lifetimeContext(ctx))
	active := &download{
		service:          service,
		id:               id,
		channel:          channel,
		path:             path,
		file:             file,
		size:             info.Size(),
		offset:           request.Offset,
		windowBytes:      int64(policy.WindowBytes),
		chunkBytes:       policy.ChunkBytes,
		encoding:         encoding,
		progressInterval: request.ProgressIntervalBytes,
		startedAt:        time.Now(),
		stream:           stream,
		ctx:              transferCtx,
		cancel:           cancel,
		ack:              make(chan protocol.FileTransferAck, 1),
		done:             make(chan struct{}),
		progress:         transfer.NewCoalescer(250 * time.Millisecond),
	}
	if encoding == compressionZstd {
		compressor, err := zstd.NewWriter(nil)
		if err != nil {
			_ = file.Close()
			return nil, nil, err
		}
		active.compressor = compressor
	}
	service.mu.Lock()
	service.downloads[id] = active
	service.mu.Unlock()
	go active.run()
	return &Transfer{
		ID: id, Channel: channel, Path: path, Offset: request.Offset, Size: info.Size(),
		ModifiedAt: info.ModTime().UTC(), WindowBytes: int64(policy.WindowBytes), ChunkBytes: policy.ChunkBytes,
		OpaqueToken: fileTransferToken(channel, id), ContentEncoding: encoding,
		ProgressIntervalBytes: request.ProgressIntervalBytes,
	}, active, nil
}

// OpenUpload 打开或恢复上传；resume token 跨 session/进程有效。
func (service *Service) OpenUpload(ctx context.Context, request UploadRequest, channel uint16, stream Stream, owner string) (*Transfer, Handle, error) {
	target, err := service.resolver.ResolveParent(request.Path)
	if err != nil {
		return nil, nil, err
	}
	if request.Size < 0 {
		return nil, nil, fmt.Errorf("invalid upload size")
	}
	now := time.Now().UTC()
	service.pruneExpired(now)

	var record transfer.Record
	resumeID := ""
	if len(request.ResumeTransferToken) > 0 {
		id, ok := fileTransferIDFromResumeToken(request.ResumeTransferToken)
		if !ok {
			return nil, nil, ErrInvalidUploadResume
		}
		resumeID = id
	}
	if resumeID != "" {
		if active := service.lookupUpload(resumeID); active != nil {
			if active.owner == owner {
				return nil, nil, ErrTransferUnavailable
			}
			active.stopForTakeover()
		}
		stored, ok := service.store.Lookup(request.ResumeTransferToken)
		if !ok || stored.Direction != transfer.DirectionUpload || stored.Path != target || stored.Size != request.Size {
			return nil, nil, ErrTransferUnavailable
		}
		record = stored
		if err := service.validateResumeRecord(record); err != nil {
			return nil, nil, err
		}
	} else {
		if !request.Overwrite {
			if _, statErr := os.Lstat(target); statErr == nil {
				return nil, nil, os.ErrExist
			} else if !os.IsNotExist(statErr) {
				return nil, nil, statErr
			}
		}
		id, err := newTransferID()
		if err != nil {
			return nil, nil, err
		}
		temp, err := os.CreateTemp(filepath.Dir(target), ".anytty-upload-*.part")
		if err != nil {
			return nil, nil, err
		}
		tempPath := temp.Name()
		_ = temp.Close()
		record = transfer.Record{
			ID: id, Token: fileUploadResumeToken(id), Direction: transfer.DirectionUpload,
			Path: target, OwnerID: owner, Size: request.Size, TargetSize: request.Size,
			Overwrite: request.Overwrite, TempPath: tempPath, ExpiresAt: now.Add(uploadResumeTTL),
		}
	}
	file, err := os.OpenFile(record.TempPath, os.O_RDWR, 0o600)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if info.Size() < record.Offset {
		_ = file.Close()
		return nil, nil, ErrTransferUnavailable
	}
	if info.Size() > record.Offset {
		// 崩溃可能让临时文件领先于已持久化 offset；回退到已校验边界。
		if err := file.Truncate(record.Offset); err != nil {
			_ = file.Close()
			return nil, nil, err
		}
	}
	hasher := sha256.New()
	if record.Offset > 0 {
		if _, err := io.CopyN(hasher, file, record.Offset); err != nil {
			_ = file.Close()
			return nil, nil, err
		}
	}
	if _, err := file.Seek(record.Offset, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	policy := service.estimator.policy()
	encoding := chooseContentEncoding(request.AcceptCompression)
	transferCtx, cancel := context.WithCancel(lifetimeContext(ctx))
	active := &upload{
		service: service, id: record.ID, channel: channel, path: target, tempPath: record.TempPath,
		overwrite: record.Overwrite, owner: owner,
		size: request.Size, stream: stream, ctx: transferCtx, cancel: cancel, done: make(chan struct{}),
		file: file, hasher: hasher, offset: record.Offset, chunkHint: policy.ChunkBytes,
		contentEncoding: encoding, progressInterval: request.ProgressIntervalBytes, startedAt: time.Now(),
		progress: transfer.NewCoalescer(250 * time.Millisecond),
	}
	if encoding == compressionZstd {
		decoder, err := zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxUploadDecodeBytes))
		if err != nil {
			_ = file.Close()
			return nil, nil, err
		}
		active.decoder = decoder
	}
	record.OwnerID = owner
	record.Offset = active.offset
	record.TargetSize = request.Size
	record.Overwrite = active.overwrite
	record.ExpiresAt = now.Add(uploadResumeTTL)
	if err := service.store.Put(record); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	service.mu.Lock()
	service.uploads[record.ID] = active
	service.mu.Unlock()
	return &Transfer{
		ID: record.ID, Channel: channel, Path: target, Offset: active.offset, Size: request.Size,
		WindowBytes: int64(policy.WindowBytes), ChunkBytes: policy.ChunkBytes,
		OpaqueToken: fileTransferToken(channel, record.ID), ResumeToken: fileUploadResumeToken(record.ID),
		ContentEncoding: encoding, ProgressIntervalBytes: request.ProgressIntervalBytes,
	}, active, nil
}

// CancelResource 取消 current-session 持有的 transfer 凭据。
func (service *Service) CancelResource(id string) bool {
	service.mu.Lock()
	activeUpload := service.uploads[id]
	activeDownload := service.downloads[id]
	service.mu.Unlock()
	if activeUpload != nil {
		activeUpload.abortAndRemove()
		return true
	}
	if activeDownload != nil {
		_ = activeDownload.Close()
		return true
	}
	return false
}

// CancelResume 按 principal-bound resume token 取消未完成上传，可跨 session。
func (service *Service) CancelResume(token []byte) (bool, error) {
	id, ok := fileTransferIDFromResumeToken(token)
	if !ok {
		return false, ErrTransferNotFound
	}
	service.mu.Lock()
	active := service.uploads[id]
	service.mu.Unlock()
	if active != nil {
		active.abortAndRemove()
		return true, nil
	}
	record, ok := service.store.Lookup(token)
	if !ok {
		return false, nil
	}
	if record.TempPath != "" {
		_ = os.Remove(record.TempPath)
	}
	_ = service.store.Delete(record.ID)
	return true, nil
}

func (service *Service) validateResumeRecord(record transfer.Record) error {
	if record.TempPath == "" {
		return ErrTransferUnavailable
	}
	info, err := os.Stat(record.TempPath)
	if err != nil {
		_ = service.store.Delete(record.ID)
		return ErrTransferUnavailable
	}
	if !info.Mode().IsRegular() || info.Size() < record.Offset {
		return ErrTransferUnavailable
	}
	return nil
}

func (service *Service) lookupUpload(id string) *upload {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.uploads[id]
}

func (service *Service) removeUpload(id string, active *upload) {
	service.mu.Lock()
	if service.uploads[id] == active {
		delete(service.uploads, id)
	}
	service.mu.Unlock()
}

func (service *Service) removeDownload(id string, active *download) {
	service.mu.Lock()
	if service.downloads[id] == active {
		delete(service.downloads, id)
	}
	service.mu.Unlock()
}

func (service *Service) snapshotUploads() []*upload {
	service.mu.Lock()
	defer service.mu.Unlock()
	out := make([]*upload, 0, len(service.uploads))
	for _, active := range service.uploads {
		out = append(out, active)
	}
	return out
}

func (service *Service) snapshotDownloads() []*download {
	service.mu.Lock()
	defer service.mu.Unlock()
	out := make([]*download, 0, len(service.downloads))
	for _, active := range service.downloads {
		out = append(out, active)
	}
	return out
}

func (d *download) run() {
	defer close(d.done)
	defer d.service.removeDownload(d.id, d)
	credit := d.windowBytes
	buffer := make([]byte, d.chunkBytes)
	for d.offset < d.size {
		for credit <= 0 {
			select {
			case <-d.ctx.Done():
				return
			case ack := <-d.ack:
				if ack.Offset != d.offset {
					d.fail(fmt.Errorf("invalid file ack at offset %d", ack.Offset))
					return
				}
				credit += ack.WindowBytes
			}
		}
		readSize := min(int64(len(buffer)), min(credit, d.size-d.offset))
		n, err := d.file.Read(buffer[:readSize])
		if err != nil && err != io.EOF {
			d.fail(err)
			return
		}
		if n == 0 {
			break
		}
		chunk := buffer[:n]
		frameEncoding := ""
		if d.encoding == compressionZstd && d.compressor != nil {
			if compressed := d.compressor.EncodeAll(chunk, nil); len(compressed) < len(chunk) {
				chunk = compressed
				frameEncoding = compressionZstd
			}
		}
		payload, err := protocol.EncodeFileTransferData(protocol.FileTransferData{Offset: d.offset, Data: chunk, Encoding: frameEncoding})
		if err != nil {
			d.fail(err)
			return
		}
		if err := d.sendBulk(wire.TypeFileData, payload); err != nil {
			return
		}
		d.service.estimator.observeDownloadSend(d.offset + int64(n))
		d.offset += int64(n)
		credit -= int64(n)
	}
	digest, err := hashFile(d.path)
	if err != nil {
		d.fail(err)
		return
	}
	payload, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{
		Size: d.size, SHA256: digest, ElapsedMillis: d.elapsedMillis(),
	})
	if err != nil {
		return
	}
	if err := d.sendBulk(wire.TypeFileFinish, payload); err != nil {
		return
	}
	progress := d.progress.Finish(d.offset, d.size, time.Now())
	d.service.logger.Debug("file download completed", "transfer_id", d.id, "path", d.path, "size", progress.Total, "rate_bytes_per_sec", progress.RateBytesPerSec)
}

func (d *download) HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error) {
	_ = ctx
	if typ != wire.TypeFileAck {
		return false, fmt.Errorf("download channel requires ack frame")
	}
	ack, err := protocol.DecodeFileTransferAck(payload)
	if err != nil {
		return false, err
	}
	if ack.Offset < 0 || ack.Offset > d.size || ack.WindowBytes < 0 || ack.WindowBytes > d.windowBytes {
		return false, fmt.Errorf("invalid download ack")
	}
	d.service.estimator.observeAck(ack.Offset)
	if progress, emit := d.progress.Observe(ack.Offset, d.size, time.Now()); emit {
		d.service.logger.Debug("file download progress", "transfer_id", d.id, "transferred", progress.Transferred, "total", progress.Total, "rate_bytes_per_sec", progress.RateBytesPerSec)
	}
	select {
	case d.ack <- ack:
		return false, nil
	default:
		return false, fmt.Errorf("download ack backpressure exceeded")
	}
}

func (d *download) Done() <-chan struct{} { return d.done }

func (d *download) Close() error {
	d.closeOnce.Do(func() {
		d.cancel()
		_ = d.file.Close()
		d.service.removeDownload(d.id, d)
	})
	return nil
}

func (d *download) fail(err error) {
	if d.ctx.Err() == nil && err != nil {
		_ = d.sendStreamError(err)
	}
}

func (d *download) sendBulk(typ uint8, payload []byte) error {
	return sendWithBackpressure(d.ctx, d.stream, typ, payload)
}

func (d *download) sendStreamError(err error) error {
	d.sendMu.Lock()
	defer d.sendMu.Unlock()
	return sendStreamError(d.stream, err)
}

func (u *upload) HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error) {
	_ = ctx
	u.mu.Lock()
	defer u.mu.Unlock()
	switch typ {
	case wire.TypeFileData:
		data, err := protocol.DecodeFileTransferData(payload)
		if err != nil {
			return false, err
		}
		raw, err := u.decodeChunk(data)
		if err != nil {
			return false, err
		}
		if data.Offset != u.offset || len(raw) == 0 || len(raw) > defaultUploadChunkLimit || u.offset+int64(len(raw)) > u.size {
			return false, fmt.Errorf("invalid upload data offset or size")
		}
		if _, err := u.file.Write(raw); err != nil {
			return false, err
		}
		if _, err := u.hasher.Write(raw); err != nil {
			return false, err
		}
		u.offset += int64(len(raw))
		u.persist(u.offset, time.Now(), false)
		if progress, emit := u.progress.Observe(u.offset, u.size, time.Now()); emit {
			u.service.logger.Debug("file upload progress", "transfer_id", u.id, "path", u.path, "transferred", progress.Transferred, "total", progress.Total, "rate_bytes_per_sec", progress.RateBytesPerSec)
		}
		ackPayload, err := protocol.EncodeFileTransferAck(u.progressAck(int64(len(raw))))
		if err != nil {
			return false, err
		}
		u.sendMu.Lock()
		err = u.stream.Send(wire.TypeFileAck, ackPayload)
		u.sendMu.Unlock()
		return false, err
	case wire.TypeFileFinish:
		finish, err := protocol.DecodeFileTransferFinish(payload)
		if err != nil {
			return false, err
		}
		if finish.Size != u.size || u.offset != u.size || !bytes.Equal(finish.SHA256, u.hasher.Sum(nil)) {
			return false, fmt.Errorf("upload checksum mismatch")
		}
		if err := u.file.Sync(); err != nil {
			return false, err
		}
		if err := u.file.Close(); err != nil {
			return false, err
		}
		u.file = nil
		if err := u.publish(); err != nil {
			return false, err
		}
		resultPayload, err := protocol.EncodeFileTransferResult(protocol.FileTransferResult{Path: u.path, Size: u.size, SHA256: finish.SHA256})
		if err != nil {
			return false, err
		}
		u.sendMu.Lock()
		err = u.stream.Send(wire.TypeFileResult, resultPayload)
		u.sendMu.Unlock()
		if err != nil {
			return false, err
		}
		u.completed = true
		progress := u.progress.Finish(u.offset, u.size, time.Now())
		u.service.logger.Debug("file upload completed", "transfer_id", u.id, "path", u.path, "size", progress.Total)
		u.service.removeUpload(u.id, u)
		return true, nil
	default:
		return false, fmt.Errorf("unsupported upload frame type %d", typ)
	}
}

func (u *upload) Done() <-chan struct{} { return u.done }

// Close 停止上传但保留断点记录与临时文件，供稍后续传。
func (u *upload) Close() error {
	u.closeOnce.Do(func() {
		u.cancel()
		u.mu.Lock()
		if u.file != nil {
			_ = u.file.Close()
			u.file = nil
		}
		if !u.completed {
			u.persist(u.offset, time.Now(), true)
		}
		u.mu.Unlock()
		u.service.removeUpload(u.id, u)
		close(u.done)
	})
	return nil
}

// stopForTakeover 关闭当前 owner 的 stream，但保留记录供新 session 接管。
func (u *upload) stopForTakeover() {
	_ = u.Close()
}

// abortAndRemove 取消上传并删除临时文件与断点记录。
func (u *upload) abortAndRemove() {
	u.closeOnce.Do(func() {
		u.cancel()
		u.mu.Lock()
		if u.file != nil {
			_ = u.file.Close()
			u.file = nil
		}
		tempPath := u.tempPath
		u.mu.Unlock()
		u.service.removeUpload(u.id, u)
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
		_ = u.service.store.Delete(u.id)
		close(u.done)
	})
}

func (u *upload) persist(offset int64, now time.Time, force bool) {
	if !force && !u.lastSave.IsZero() && now.Sub(u.lastSave) < uploadPersistInterval {
		return
	}
	u.lastSave = now
	record, ok := u.service.store.Get(u.id)
	if !ok {
		return
	}
	record.Offset = offset
	record.ExpiresAt = now.UTC().Add(uploadResumeTTL)
	_ = u.service.store.Put(record)
}

func (u *upload) publish() error {
	if !u.overwrite {
		if _, err := os.Lstat(u.path); err == nil {
			return os.ErrExist
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(u.tempPath, u.path); err != nil {
		return err
	}
	_ = u.service.store.Delete(u.id)
	return nil
}

// sendWithBackpressure 在 reporter 报告发送队列超过目标时等待，避免大文件发送
// 把整个文件内容堆进 transport 队列。
func sendWithBackpressure(ctx context.Context, stream Stream, typ uint8, payload []byte) error {
	if stream == nil {
		return ErrTransferUnavailable
	}
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for stream.OutboundBuffered() > outboundQueueTarget {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stream.Done():
			return io.EOF
		case <-ticker.C:
		}
	}
	return stream.Send(typ, payload)
}

func sendStreamError(stream Stream, err error) error {
	payload, encodeErr := protocol.EncodeErrorPayload(protocol.ErrorMessage{Error: protocol.ProtocolError{Code: 500, Message: err.Error()}})
	if encodeErr != nil {
		return encodeErr
	}
	return stream.Send(wire.TypeError, payload)
}

func lifetimeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func hashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return nil, err
	}
	return digest.Sum(nil), nil
}

func fileTransferToken(channel uint16, transferID string) []byte {
	token := make([]byte, 4+len(transferID))
	binary.BigEndian.PutUint16(token[:2], channel)
	copy(token[2:4], "ft")
	copy(token[4:], transferID)
	return token
}

func fileTransferIDFromResourceToken(token []byte) (string, bool) {
	if len(token) != 36 || binary.BigEndian.Uint16(token[:2]) == 0 || string(token[2:4]) != "ft" {
		return "", false
	}
	return validFileTransferID(string(token[4:]))
}

func fileUploadResumeToken(transferID string) []byte {
	return append([]byte("fr"), transferID...)
}

func fileTransferIDFromResumeToken(token []byte) (string, bool) {
	if len(token) != 34 || string(token[:2]) != "fr" {
		return "", false
	}
	return validFileTransferID(string(token[2:]))
}

// chooseContentEncoding 选择双方都支持的 data frame 编码；未知值忽略，空=identity。
func chooseContentEncoding(accept []string) string {
	for _, candidate := range accept {
		if strings.EqualFold(strings.TrimSpace(candidate), compressionZstd) {
			return compressionZstd
		}
	}
	return ""
}

// elapsedMillis 只在启用进度时报告耗时；未启用时保持 0（旧字节行为）。
func (d *download) elapsedMillis() int64 {
	if d.progressInterval <= 0 {
		return 0
	}
	return time.Since(d.startedAt).Milliseconds()
}

// progressAck 在启用进度窗口时携带结构化进度；未启用时字段全为 0。
func (u *upload) progressAck(windowBytes int64) protocol.FileTransferAck {
	ack := protocol.FileTransferAck{Offset: u.offset, WindowBytes: windowBytes}
	if u.progressInterval > 0 && (u.offset-u.lastProgress >= u.progressInterval || u.offset == u.size) {
		ack.TransferredBytes = u.offset
		ack.TotalBytes = u.size
		ack.ElapsedMillis = time.Since(u.startedAt).Milliseconds()
		u.lastProgress = u.offset
	}
	return ack
}

// decodeChunk 按帧内 encoding 解压上传数据；未知编码必须拒绝。
func (u *upload) decodeChunk(data protocol.FileTransferData) ([]byte, error) {
	switch data.Encoding {
	case "", "identity":
		return data.Data, nil
	case compressionZstd:
		if u.decoder == nil {
			return nil, fmt.Errorf("zstd upload chunk was not negotiated")
		}
		decoded, err := u.decoder.DecodeAll(data.Data, nil)
		if err != nil {
			return nil, fmt.Errorf("decode zstd upload chunk: %w", err)
		}
		if len(decoded) > maxUploadDecodeBytes {
			return nil, fmt.Errorf("zstd upload chunk exceeds decode limit")
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported upload data encoding %q", data.Encoding)
	}
}

func newTransferID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func validFileTransferID(id string) (string, bool) {
	if _, err := hex.DecodeString(id); err != nil {
		return "", false
	}
	return id, true
}
