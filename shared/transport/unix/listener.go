package unix

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport"
	"github.com/klauspost/compress/zstd"
)

// Transport 是本机 unix socket 上的压缩 frame transport。
// 它只拥有 frame 边界、分片、压缩和连接生命周期，不解析 protocol payload，也不持有 terminal truth。
type Transport struct {
	conn net.Conn
	done chan struct{}
	once sync.Once

	sendMu sync.Mutex
	zstdW  *zstd.Encoder
	zstdR  *zstd.Decoder

	sendQ chan sendRequest
	wg    sync.WaitGroup
}

type sendRequest struct {
	frame []byte
	done  chan error
}

const (
	packetKindFrame byte = iota
	packetKindFragmentStart
	packetKindFragmentContinue
	packetKindFragmentEnd
)

const (
	maxPacketPayloadSize = 64 << 10
	maxLogicalFrameSize  = wire.MaxEncodedFrameSize
	sendBatchWindow      = time.Millisecond
	maxBatchedBytes      = 128 << 10
	acceptPollInterval   = 50 * time.Millisecond
)

const (
	zstdTransportEncoderWindowSize = 128 << 10
	zstdTransportDecoderMaxWindow  = 256 << 10
)

// Dial 连接本机 anytty daemon unix socket，并返回 frame transport。
// path 可以是用户可见长路径；实际 socket 路径由 resolveSocketPath 统一解析，避免调用方绕过别名规则。
func Dial(path string) (*Transport, error) {
	return DialContext(context.Background(), path)
}

// DialContext 连接本机 anytty daemon unix socket，并让建连过程响应调用方取消或 deadline。
// context 只控制本次 transport 建立；成功后连接生命周期仍由返回的 Transport.Close 负责。
func DialContext(ctx context.Context, path string) (*Transport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	actualPath, _ := resolveSocketPath(path)
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", actualPath)
	if err != nil {
		return nil, err
	}
	return newTransport(conn)
}

// Send 发送一个完整 protocol frame。
// unix transport 是 frame 边界 owner：大 frame 会在本层按 packet 上限分片，但不会解释 protocol payload。
func (t *Transport) Send(frame []byte) error {
	if len(frame) > maxLogicalFrameSize {
		return wire.ErrFrameTooLarge
	}
	if t == nil || t.zstdW == nil || t.sendQ == nil {
		return io.EOF
	}
	req := sendRequest{
		frame: append([]byte(nil), frame...),
		done:  make(chan error, 1),
	}
	select {
	case <-t.done:
		return io.EOF
	case t.sendQ <- req:
	}
	select {
	case <-t.done:
		return io.EOF
	case err := <-req.done:
		return err
	}
}

// Recv 接收一个完整 protocol frame。
// 分片重组只发生在 transport 层；异常 packet kind、读错误或关闭都会作为 transport 错误返回。
func (t *Transport) Recv() ([]byte, error) {
	if t == nil || t.zstdR == nil {
		return nil, io.EOF
	}
	for {
		kind, payload, err := t.readPacket()
		if err != nil {
			return nil, err
		}
		switch kind {
		case packetKindFrame:
			return payload, nil
		case packetKindFragmentStart:
			buf := append([]byte(nil), payload...)
			for {
				nextKind, nextPayload, err := t.readPacket()
				if err != nil {
					return nil, err
				}
				switch nextKind {
				case packetKindFragmentContinue:
					if len(nextPayload) > maxLogicalFrameSize-len(buf) {
						return nil, wire.ErrFrameTooLarge
					}
					buf = append(buf, nextPayload...)
				case packetKindFragmentEnd:
					if len(nextPayload) > maxLogicalFrameSize-len(buf) {
						return nil, wire.ErrFrameTooLarge
					}
					buf = append(buf, nextPayload...)
					return buf, nil
				default:
					return nil, fmt.Errorf("transport/unix: unexpected packet kind %d during fragmented frame", nextKind)
				}
			}
		default:
			return nil, fmt.Errorf("transport/unix: unexpected packet kind %d", kind)
		}
	}
}

// Close 关闭 unix transport，并解除正在阻塞的读写。
// 先关闭 done 和底层 conn，再等待 sender goroutine，避免 zstd Flush 卡住时 Close 永久等待。
func (t *Transport) Close() error {
	if t == nil {
		return nil
	}
	var err error
	t.once.Do(func() {
		close(t.done)
		err = t.conn.Close()
		t.wg.Wait()
		t.sendMu.Lock()
		if t.zstdW != nil {
			_ = t.zstdW.Close()
		}
		t.sendMu.Unlock()
	})
	return err
}

// Done 返回 transport 生命周期结束信号。
// protocol client 用它区分阻塞中的 Recv 和已经关闭的底层连接。
func (t *Transport) Done() <-chan struct{} {
	return t.done
}

// Listener 是本机 unix socket listener。
// 它负责 path/alias 生命周期和 Accept 边界，不拥有 daemon session 或 protocol client 状态。
type Listener struct {
	path       string
	actualPath string
	aliasPath  string
	ln         net.Listener
}

// NewListener 创建本机 unix socket listener。
// 当用户可见 path 超过系统 socket path 限制时，会创建短实际路径并在原路径放置 symlink。
func NewListener(path string) (*Listener, error) {
	actualPath, aliasPath := resolveSocketPath(path)
	if err := preserveActiveSocket(actualPath); err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	if actualPath != path {
		_ = os.Remove(actualPath)
	}
	ln, err := net.Listen("unix", actualPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(actualPath, 0600); err != nil {
		_ = ln.Close()
		_ = os.Remove(actualPath)
		return nil, err
	}
	if aliasPath != "" {
		if err := os.MkdirAll(filepath.Dir(aliasPath), 0o755); err != nil {
			_ = ln.Close()
			_ = os.Remove(actualPath)
			return nil, err
		}
		if err := os.Symlink(actualPath, aliasPath); err != nil {
			_ = ln.Close()
			_ = os.Remove(actualPath)
			return nil, err
		}
	}
	return &Listener{path: path, actualPath: actualPath, aliasPath: aliasPath, ln: ln}, nil
}

func preserveActiveSocket(path string) error {
	conn, err := net.DialTimeout("unix", path, 100*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("transport/unix: socket %q already has an active listener", path)
	}
	if os.IsNotExist(err) || errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
		return nil
	}
	return fmt.Errorf("transport/unix: inspect existing socket %q: %w", path, err)
}

// Accept 接收一个 unix socket 连接。
// ctx 取消必须只影响本次等待，不能留下阻塞 goroutine 抢走后续 Dial；因此这里用 listener deadline 轮询。
func (l *Listener) Accept(ctx context.Context) (transport.Transport, error) {
	if l == nil || l.ln == nil {
		return nil, transport.ErrListenerClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	deadlineLn, _ := l.ln.(interface {
		SetDeadline(time.Time) error
	})
	if deadlineLn != nil {
		defer deadlineLn.SetDeadline(time.Time{})
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if deadlineLn != nil {
			_ = deadlineLn.SetDeadline(time.Now().Add(acceptPollInterval))
		}
		conn, err := l.ln.Accept()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return nil, transport.ErrListenerClosed
			}
			return nil, err
		}
		if conn == nil {
			return nil, transport.ErrListenerClosed
		}
		return newTransport(conn)
	}
}

// Close 关闭 listener 并清理短路径/别名路径。
func (l *Listener) Close() error {
	err := l.ln.Close()
	if l.aliasPath != "" {
		_ = os.Remove(l.aliasPath)
	}
	_ = os.Remove(l.actualPath)
	if l.actualPath != l.path {
		_ = os.Remove(l.path)
	}
	return err
}

// Addr 返回用户配置的 listener path。
// 如果实际 socket 因路径过长被映射到短路径，这里仍返回用户可见 alias。
func (l *Listener) Addr() string {
	return l.path
}

func resolveSocketPath(path string) (string, string) {
	if len(path) <= maxSocketPathBytes() {
		return path, ""
	}
	sum := sha256.Sum256([]byte(path))
	actual := filepath.Join(shortSocketBaseDir(), fmt.Sprintf("anytty-%x.sock", sum[:8]))
	if runtime.GOOS == "windows" {
		// 中文说明：Windows AF_UNIX 支持短路径，但普通用户创建 symlink 需要额外特权；Dial/Listen 共享哈希路径即可保持同一 transport truth。
		return actual, ""
	}
	return actual, path
}

func shortSocketBaseDir() string {
	if runtime.GOOS == "darwin" {
		return "/tmp"
	}
	return os.TempDir()
}

func maxSocketPathBytes() int {
	return len(syscall.RawSockaddrUnix{}.Path) - 1
}

func newTransport(conn net.Conn) (*Transport, error) {
	if conn == nil {
		return nil, io.EOF
	}
	// 本地 RPC 已按 64KB 分片、128KB 批量发送；默认 8MB zstd window 会让每个
	// attach 的多条连接常驻几十 MB 历史表，真实收益很低。
	writer, err := zstd.NewWriter(
		conn,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithWindowSize(zstdTransportEncoderWindowSize),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader, err := zstd.NewReader(
		conn,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
		// decoder 上限高于本端 encoder window，用来兼容同版本不同发送方向的批量，
		// 但仍明显低于库默认的超大 window。
		zstd.WithDecoderMaxWindow(zstdTransportDecoderMaxWindow),
	)
	if err != nil {
		_ = writer.Close()
		_ = conn.Close()
		return nil, err
	}
	t := &Transport{
		conn:  conn,
		done:  make(chan struct{}),
		zstdW: writer,
		zstdR: reader,
		sendQ: make(chan sendRequest, 256),
	}
	t.wg.Add(1)
	go t.runSender()
	return t, nil
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func (t *Transport) readPacket() (byte, []byte, error) {
	var header [5]byte
	if _, err := io.ReadFull(t.zstdR, header[:]); err != nil {
		return 0, nil, err
	}
	kind := header[0]
	if !validPacketKind(kind) {
		return 0, nil, fmt.Errorf("transport/unix: unexpected packet kind %d", kind)
	}
	n := binary.BigEndian.Uint32(header[1:])
	if n > maxPacketPayloadSize {
		return 0, nil, fmt.Errorf("transport/unix: packet too large: %d > %d", n, maxPacketPayloadSize)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(t.zstdR, buf); err != nil {
		return 0, nil, err
	}
	return kind, buf, nil
}

func (t *Transport) runSender() {
	defer t.wg.Done()
	var (
		batch            []sendRequest
		batchedByteCount int
		timer            *time.Timer
		timerCh          <-chan time.Time
	)
	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		err := t.writeBatch(batch)
		for _, req := range batch {
			req.done <- err
		}
		batch = batch[:0]
		batchedByteCount = 0
		if timer != nil {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
		timerCh = nil
		return err == nil
	}
	for {
		select {
		case req, ok := <-t.sendQ:
			if !ok {
				_ = flush()
				return
			}
			batch = append(batch, req)
			batchedByteCount += len(req.frame)
			if timer == nil {
				timer = time.NewTimer(sendBatchWindow)
			} else {
				timer.Reset(sendBatchWindow)
			}
			timerCh = timer.C
			if batchedByteCount >= maxBatchedBytes {
				if !flush() {
					return
				}
			}
		case <-timerCh:
			if !flush() {
				return
			}
		case <-t.done:
			_ = flush()
			return
		}
	}
}

func (t *Transport) writeBatch(batch []sendRequest) error {
	if t == nil || t.zstdW == nil {
		return io.EOF
	}
	t.sendMu.Lock()
	defer t.sendMu.Unlock()
	for _, req := range batch {
		if err := t.writeLogicalFrame(req.frame); err != nil {
			return err
		}
	}
	return t.zstdW.Flush()
}

func (t *Transport) writeLogicalFrame(frame []byte) error {
	if len(frame) <= maxPacketPayloadSize {
		return t.writePacket(packetKindFrame, frame)
	}
	offset := 0
	for offset < len(frame) {
		end := offset + maxPacketPayloadSize
		if end > len(frame) {
			end = len(frame)
		}
		kind := packetKindFragmentContinue
		switch {
		case offset == 0:
			kind = packetKindFragmentStart
		case end == len(frame):
			kind = packetKindFragmentEnd
		}
		if err := t.writePacket(kind, frame[offset:end]); err != nil {
			return err
		}
		offset = end
	}
	return nil
}

func (t *Transport) writePacket(kind byte, payload []byte) error {
	if !validPacketKind(kind) {
		return errors.New("transport/unix: invalid packet kind")
	}
	if len(payload) > maxPacketPayloadSize {
		return fmt.Errorf("transport/unix: packet too large: %d > %d", len(payload), maxPacketPayloadSize)
	}
	var header [5]byte
	header[0] = kind
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
	if err := writeAll(t.zstdW, header[:]); err != nil {
		return err
	}
	return writeAll(t.zstdW, payload)
}

func validPacketKind(kind byte) bool {
	return kind == packetKindFrame ||
		kind == packetKindFragmentStart ||
		kind == packetKindFragmentContinue ||
		kind == packetKindFragmentEnd
}
