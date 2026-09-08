// Package browserproxy provides the shared client-side loopback HTTP/CONNECT
// proxy. Target resolution and dialing always take place on the remote daemon.
package browserproxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/proto/wirepb"
	"google.golang.org/protobuf/proto"
)

const (
	windowBytes        = 512 << 10
	chunkBytes         = 32 << 10
	maximumHeaderBytes = 64 << 10
	ioTimeout          = 30 * time.Second
)

// Open returns a dedicated, ordered resource stream and its negotiated windows.
// It must clean up resources created after cancellation, and honor ctx.
type Open func(ctx context.Context, host string, port uint32) (clientruntime.ResourceStream, uint32, uint32, error)

type Server struct {
	listener  net.Listener
	open      Open
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	closed    bool
	sockets   map[net.Conn]struct{}
	resources chan struct{}
	wg        sync.WaitGroup
	sequence  atomic.Uint64
}

// Start binds IPv4 loopback only. Close cancels all work and closes local sockets;
// Wait joins workers, including any bounded native transport sends still draining.
func Start(ctx context.Context, open Open) (*Server, error) {
	if ctx == nil || open == nil {
		return nil, errors.New("browser proxy context and opener are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Server{listener: listener, open: open, ctx: ctx, cancel: cancel, sockets: make(map[net.Conn]struct{}), resources: make(chan struct{}, 16)}
	s.wg.Add(1)
	go s.accept()
	go func() { <-ctx.Done(); _ = s.Close() }()
	return s, nil
}

func (s *Server) Port() uint32 { return uint32(s.listener.Addr().(*net.TCPAddr).Port) }

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.cancel()
	_ = s.listener.Close()
	for conn := range s.sockets {
		_ = conn.Close()
	}
	return nil
}

func (s *Server) Wait() { s.wg.Wait() }

func (s *Server) Done() <-chan struct{} { return s.ctx.Done() }

func (s *Server) accept() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed || len(s.sockets) >= 64 {
			s.mu.Unlock()
			_ = conn.Close()
			continue
		}
		s.sockets[conn] = struct{}{}
		s.wg.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.wg.Done()
			defer func() { _ = conn.Close(); s.mu.Lock(); delete(s.sockets, conn); s.mu.Unlock() }()
			s.serve(conn)
		}()
	}
}

// readRequest bounds headers independently of bodies, then uses net/http for
// framing validation, chunked uploads and absolute-URI parsing.
func readRequest(reader *bufio.Reader) (*http.Request, *bufio.Reader, error) {
	var header bytes.Buffer
	continued := false
	for {
		line, err := reader.ReadSlice('\n')
		if header.Len()+len(line) > maximumHeaderBytes {
			return nil, nil, errors.New("proxy headers exceed limit")
		}
		header.Write(line)
		if err != nil && err != bufio.ErrBufferFull {
			return nil, nil, err
		}
		if err == nil && !continued && bytes.Equal(line, []byte("\r\n")) {
			break
		}
		continued = err == bufio.ErrBufferFull
	}
	input := bufio.NewReader(io.MultiReader(bytes.NewReader(header.Bytes()), reader))
	request, err := http.ReadRequest(input)
	return request, input, err
}

func target(request *http.Request) (string, uint32, bool, error) {
	connect := request.Method == http.MethodConnect
	if !connect && (request.URL.Scheme != "http" && request.URL.Scheme != "ws" || request.URL.Host == "") {
		return "", 0, false, errors.New("proxy requires an absolute HTTP URI or CONNECT")
	}
	if request.URL.User != nil {
		return "", 0, false, errors.New("proxy target userinfo is not supported")
	}
	host, portText := request.URL.Hostname(), request.URL.Port()
	if host == "" {
		return "", 0, false, errors.New("proxy target host is missing")
	}
	port := uint64(80)
	if connect {
		port = 443
	}
	if portText != "" {
		var err error
		port, err = strconv.ParseUint(portText, 10, 16)
		if err != nil || port == 0 {
			return "", 0, false, errors.New("proxy target port is invalid")
		}
	}
	return host, uint32(port), connect, nil
}

func (s *Server) serve(conn net.Conn) {
	id, started := s.sequence.Add(1), time.Now()
	stats := &timings{}
	stage := "headers"
	var failure error
	defer func() {
		// Never log URLs, headers, credentials or request/response bodies.
		var networkError net.Error
		timedOut := errors.Is(failure, context.DeadlineExceeded) || errors.As(failure, &networkError) && networkError.Timeout()
		log.Printf("anytty browser proxy request_id=%d stage=closed last_stage=%s failed=%t error_type=%T error_code=%s timeout=%t total_ms=%d upload_bytes=%d download_bytes=%d credit_us=%d send_us=%d socket_us=%d ack_us=%d", id, stage, failure != nil, failure, clientruntime.CodeOf(failure), timedOut, time.Since(started).Milliseconds(), stats.upload.Load(), stats.download.Load(), stats.credit.Load(), stats.send.Load(), stats.socket.Load(), stats.ack.Load())
	}()
	respond := func(code int) {
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, _ = fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\nConnection: close\r\nContent-Length: 0\r\n\r\n", code, http.StatusText(code))
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	request, input, err := readRequest(bufio.NewReader(conn))
	if err != nil {
		failure = err
		respond(400)
		return
	}
	host, port, connect, err := target(request)
	if err != nil {
		failure = err
		respond(400)
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	stage = "resource_queue"
	queueStart := time.Now()
	select {
	case s.resources <- struct{}{}:
	case <-ctx.Done():
		failure = ctx.Err()
		return
	}
	defer func() { <-s.resources }()
	queueWait := time.Since(queueStart)
	stage = "resource_open"
	openStart := time.Now()
	openCtx, openCancel := context.WithTimeout(ctx, ioTimeout)
	stream, receiveWindow, sendWindow, err := s.open(openCtx, host, port)
	openCancel()
	if err != nil {
		failure = err
		respond(502)
		return
	}
	defer stream.Close()
	if receiveWindow > 1<<20 || sendWindow > 1<<20 {
		failure = errors.New("invalid browser window")
		respond(502)
		return
	}
	log.Printf("anytty browser proxy request_id=%d stage=resource_open queue_us=%d open_us=%d", id, queueWait.Microseconds(), time.Since(openStart).Microseconds())
	_ = conn.SetDeadline(time.Time{})
	if connect {
		_ = conn.SetWriteDeadline(time.Now().Add(ioTimeout))
		if _, err = io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			failure = err
			return
		}
	}
	stage = "transfer"
	credit := newCredit(int64(sendWindow))
	receiver := newReceiver(ctx, stream, credit, receiveWindow)
	workers := sync.WaitGroup{}
	workers.Add(2)
	go func() { defer workers.Done(); receiver.run() }()
	writer := &uploadWriter{ctx: ctx, stream: stream, credit: credit, stats: stats}
	go func() {
		defer workers.Done()
		var uploadErr error
		if connect {
			_, uploadErr = io.CopyBuffer(writer, &idleReader{conn: conn, reader: input}, make([]byte, chunkBytes))
		} else {
			websocket := strings.EqualFold(request.Header.Get("Upgrade"), "websocket") && headerToken(request.Header.Get("Connection"), "upgrade")
			request.RequestURI = ""
			request.Header.Del("Proxy-Connection")
			request.Header.Del("Proxy-Authorization")
			request.Header.Del("Connection")
			request.Close = !websocket
			if websocket {
				request.Header.Set("Connection", "Upgrade")
			} else {
				request.Header.Del("Upgrade")
			}
			// The daemon starts reading after receiving data. Resolve Expect
			// locally so a browser cannot wait for an upstream 100 indefinitely.
			if strings.EqualFold(request.Header.Get("Expect"), "100-continue") {
				request.Header.Del("Expect")
				_, uploadErr = io.WriteString(conn, "HTTP/1.1 100 Continue\r\n\r\n")
			}
			if uploadErr == nil {
				if request.Body != nil && request.Body != http.NoBody {
					request.Body = &idleBody{body: request.Body, conn: conn}
				}
				uploadErr = request.Write(writer)
			}
			if uploadErr == nil && websocket {
				_, uploadErr = io.CopyBuffer(writer, &idleReader{conn: conn, reader: input}, make([]byte, chunkBytes))
			}
		}
		if uploadErr != nil {
			receiver.fail(uploadErr)
			cancel()
			_ = conn.Close()
		}
	}()
	failure = receiver.download(conn, stats)
	cancel()
	_ = conn.Close()
	workers.Wait()
}

func headerToken(value, token string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(item), token) {
			return true
		}
	}
	return false
}

type idleReader struct {
	conn   net.Conn
	reader io.Reader
}

type idleBody struct {
	body io.ReadCloser
	conn net.Conn
}

func (b *idleBody) Read(p []byte) (int, error) {
	_ = b.conn.SetReadDeadline(time.Now().Add(ioTimeout))
	return b.body.Read(p)
}
func (b *idleBody) Close() error { return b.body.Close() }

func (r *idleReader) Read(p []byte) (int, error) {
	_ = r.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	return r.reader.Read(p)
}

type timings struct{ upload, download, credit, send, socket, ack atomic.Int64 }

type uploadWriter struct {
	ctx    context.Context
	stream clientruntime.ResourceStream
	credit *creditWindow
	stats  *timings
}

func (w *uploadWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		start := time.Now()
		count, err := w.credit.reserve(w.ctx, min(len(p), chunkBytes))
		w.stats.credit.Add(time.Since(start).Microseconds())
		if err != nil {
			return written, err
		}
		start = time.Now()
		err = w.stream.Send(w.ctx, wire.TypeBrowserData, p[:count])
		w.stats.send.Add(time.Since(start).Microseconds())
		if err != nil {
			return written, err
		}
		written += count
		p = p[count:]
		w.stats.upload.Add(int64(count))
	}
	return written, nil
}

// OpenSession adapts the same application/runtime interfaces used by CLI/TUI.
func OpenSession(session clientruntime.ApplicationReadyPeerSession) (Open, error) {
	provider, ok := session.(clientruntime.ResourceStreamSession)
	if !ok {
		return nil, errors.New("session does not support browser streams")
	}
	application, err := clientruntime.NewApplicationSession(session.Stamp(), session)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, host string, port uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
		result, err := application.BrowserProxyOpen(ctx, &apipb.BrowserProxyOpenCommand{Host: host, Port: port, ReceiveWindowBytes: windowBytes, SendWindowBytes: windowBytes})
		if err != nil {
			return nil, 0, 0, err
		}
		resource := result.GetResource()
		if resource == nil || resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_BROWSER_PROXY || len(resource.GetOpaqueToken()) == 0 {
			return nil, 0, 0, errors.New("invalid browser resource")
		}
		stream, err := provider.OpenResourceStream(resource)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			if stream != nil {
				_ = stream.Close()
			}
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = application.ReleaseResource(cleanupCtx, &apipb.ReleaseResourceCommand{Resource: resource})
			return nil, 0, 0, err
		}
		return stream, result.GetReceiveWindowBytes(), result.GetSendWindowBytes(), nil
	}, nil
}

type creditWindow struct {
	mu                          sync.Mutex
	maximum, sent, acknowledged int64
	changed                     chan struct{}
}

func newCredit(maximum int64) *creditWindow {
	return &creditWindow{maximum: maximum, changed: make(chan struct{})}
}
func (c *creditWindow) reserve(ctx context.Context, requested int) (int, error) {
	timer := time.NewTimer(ioTimeout)
	defer timer.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		c.mu.Lock()
		available := c.maximum - (c.sent - c.acknowledged)
		if c.maximum == 0 {
			available = int64(requested)
		}
		if available > 0 {
			count := min(requested, int(available))
			c.sent += int64(count)
			c.mu.Unlock()
			return count, nil
		}
		changed := c.changed
		c.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
			return 0, errors.New("browser upload credit timeout")
		}
	}
}
func (c *creditWindow) acknowledge(payload []byte) error {
	ack := &wirepb.FileTransferAck{}
	if err := proto.Unmarshal(payload, ack); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.maximum == 0 || ack.Offset < c.acknowledged || ack.Offset > c.sent || ack.WindowBytes != ack.Offset-c.acknowledged {
		return errors.New("invalid browser upload credit")
	}
	c.acknowledged = ack.Offset
	close(c.changed)
	c.changed = make(chan struct{})
	return nil
}

type receiver struct {
	ctx     context.Context
	stream  clientruntime.ResourceStream
	credit  *creditWindow
	window  uint32
	mu      sync.Mutex
	queue   [][]byte
	queued  int
	ended   bool
	err     error
	changed chan struct{}
}

func newReceiver(ctx context.Context, stream clientruntime.ResourceStream, credit *creditWindow, window uint32) *receiver {
	return &receiver{ctx: ctx, stream: stream, credit: credit, window: window, changed: make(chan struct{})}
}
func (r *receiver) signal() { close(r.changed); r.changed = make(chan struct{}) }
func (r *receiver) fail(err error) {
	r.mu.Lock()
	r.ended = true
	r.err = err
	r.signal()
	r.mu.Unlock()
}
func (r *receiver) run() {
	for {
		typ, payload, err := r.stream.Receive(r.ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			r.fail(err)
			return
		}
		switch typ {
		case wire.TypeBrowserData:
			r.mu.Lock()
			limit := int(r.window)
			if limit == 0 {
				limit = 4 << 20
			}
			if r.queued+len(payload) > limit || len(r.queue) >= 4096 {
				r.mu.Unlock()
				r.fail(errors.New("browser download window exceeded"))
				return
			}
			if len(payload) > 0 {
				r.queue = append(r.queue, payload)
				r.queued += len(payload)
				r.signal()
			}
			r.mu.Unlock()
		case wire.TypeFileAck:
			if err := r.credit.acknowledge(payload); err != nil {
				r.fail(err)
				return
			}
		case wire.TypeBrowserClosed, wire.TypeClosed:
			r.fail(nil)
			return
		default:
			r.fail(errors.New("unexpected browser frame"))
			return
		}
	}
}
func (r *receiver) download(conn net.Conn, stats *timings) error {
	var offset int64
	for {
		r.mu.Lock()
		if len(r.queue) == 0 {
			ended, err, changed := r.ended, r.err, r.changed
			r.mu.Unlock()
			if ended {
				return err
			}
			select {
			case <-r.ctx.Done():
				return r.ctx.Err()
			case <-changed:
			}
			continue
		}
		payload := r.queue[0]
		r.queue[0] = nil
		r.queue = r.queue[1:]
		r.mu.Unlock()
		start := time.Now()
		_ = conn.SetWriteDeadline(time.Now().Add(ioTimeout))
		written, err := io.Copy(conn, bytes.NewReader(payload))
		stats.socket.Add(time.Since(start).Microseconds())
		stats.download.Add(written)
		if err != nil {
			return err
		}
		r.mu.Lock()
		r.queued -= len(payload)
		ended := r.ended
		r.mu.Unlock()
		offset += written
		if r.window != 0 && !ended {
			ack, _ := proto.Marshal(&wirepb.FileTransferAck{Offset: offset, WindowBytes: written})
			start = time.Now()
			err = r.stream.Send(r.ctx, wire.TypeFileAck, ack)
			stats.ack.Add(time.Since(start).Microseconds())
			if err != nil {
				r.mu.Lock()
				ended = r.ended
				r.mu.Unlock()
				if !ended {
					return err
				}
			}
		}
	}
}
