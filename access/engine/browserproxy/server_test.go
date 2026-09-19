package browserproxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/proto/access/wirepb"
	"google.golang.org/protobuf/proto"
)

type testFrame struct {
	typ     uint8
	payload []byte
}
type testStream struct {
	frames chan testFrame
	done   chan struct{}
	once   sync.Once
	send   func(context.Context, uint8, []byte) error
}

func newTestStream() *testStream {
	return &testStream{frames: make(chan testFrame, 128), done: make(chan struct{})}
}
func (s *testStream) Receive(ctx context.Context) (uint8, []byte, error) {
	select {
	case f := <-s.frames:
		return f.typ, f.payload, nil
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case <-s.done:
		return 0, nil, io.EOF
	}
}
func (s *testStream) Send(ctx context.Context, typ uint8, p []byte) error {
	if s.send != nil {
		return s.send(ctx, typ, p)
	}
	return nil
}
func (s *testStream) Close() error              { s.once.Do(func() { close(s.done) }); return nil }
func (s *testStream) frame(typ uint8, p []byte) { s.frames <- testFrame{typ, bytes.Clone(p)} }

func startTestProxy(t *testing.T, open Open) *Server {
	t.Helper()
	s, err := Start(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.Close()
		done := make(chan struct{})
		go func() { s.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("proxy workers did not exit")
		}
	})
	return s
}
func dialProxy(t *testing.T, s *Server) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", s.Port()), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	return conn
}

func TestHTTPProxyStreamsChunkedAndFixedUploads(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		t.Run(fmt.Sprint(chunked), func(t *testing.T) {
			stream := newTestStream()
			remote, daemon := net.Pipe()
			defer remote.Close()
			defer daemon.Close()
			var sent int64
			stream.send = func(ctx context.Context, typ uint8, p []byte) error {
				if typ != wire.TypeBrowserData {
					return nil
				}
				_ = remote.SetWriteDeadline(time.Now().Add(time.Second))
				n, err := remote.Write(p)
				sent += int64(n)
				ack, _ := proto.Marshal(&wirepb.FileTransferAck{Offset: sent, WindowBytes: int64(n)})
				stream.frame(wire.TypeFileAck, ack)
				return err
			}
			s := startTestProxy(t, func(_ context.Context, host string, port uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
				if host != "remote-only.invalid" || port != 8080 {
					t.Errorf("target = %s:%d", host, port)
				}
				return stream, windowBytes, 4096, nil
			})
			body := bytes.Repeat([]byte("upload"), 50000)
			checked := make(chan error, 1)
			go func() {
				_ = daemon.SetReadDeadline(time.Now().Add(2 * time.Second))
				req, err := http.ReadRequest(bufio.NewReader(daemon))
				if err == nil {
					var got []byte
					got, err = io.ReadAll(req.Body)
					if err == nil && (!bytes.Equal(got, body) || req.RequestURI != "/upload?q=1" || !req.Close || req.Header.Get("Proxy-Authorization") != "") {
						err = errors.New("forwarded request differs")
					}
				}
				checked <- err
				stream.frame(wire.TypeBrowserData, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"))
				stream.frame(wire.TypeBrowserClosed, nil)
			}()
			conn := dialProxy(t, s)
			req, _ := http.NewRequest("POST", "http://remote-only.invalid:8080/upload?q=1", bytes.NewReader(body))
			req.Header.Set("Proxy-Authorization", "secret-test")
			if chunked {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			if err := req.WriteProxy(conn); err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(conn), req)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(response.Body)
			if err != nil || string(got) != "ok" {
				t.Fatalf("response %q: %v", got, err)
			}
			if err := <-checked; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCONNECTPreservesLeftoverAndRemoteAddress(t *testing.T) {
	for _, authority := range []string{"127.0.0.1:8080", "[::1]:8080", "remote-only.invalid:8080"} {
		t.Run(authority, func(t *testing.T) {
			stream := newTestStream()
			got := make(chan string, 1)
			stream.send = func(_ context.Context, typ uint8, p []byte) error {
				if typ == wire.TypeBrowserData {
					got <- string(p)
					stream.frame(wire.TypeBrowserData, p)
					stream.frame(wire.TypeBrowserClosed, nil)
				}
				return nil
			}
			s := startTestProxy(t, func(_ context.Context, host string, port uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
				if net.JoinHostPort(host, fmt.Sprint(port)) != authority {
					t.Errorf("wrong target %s:%d", host, port)
				}
				return stream, windowBytes, windowBytes, nil
			})
			conn := dialProxy(t, s)
			_, _ = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\nhello", authority, authority)
			reader := bufio.NewReader(conn)
			resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
			if err != nil || resp.StatusCode != 200 {
				t.Fatalf("CONNECT: %v %v", resp, err)
			}
			body, err := io.ReadAll(reader)
			if err != nil || string(body) != "hello" {
				t.Fatalf("body %q %v", body, err)
			}
			if value := <-got; value != "hello" {
				t.Fatalf("upload %q", value)
			}
		})
	}
}

func TestWebSocketUpgradeAndBytes(t *testing.T) {
	stream := newTestStream()
	var upload bytes.Buffer
	stream.send = func(_ context.Context, typ uint8, p []byte) error {
		if typ != wire.TypeBrowserData {
			return nil
		}
		upload.Write(p)
		if strings.HasSuffix(upload.String(), "frame") {
			if !strings.Contains(upload.String(), "Connection: Upgrade\r\n") || strings.Contains(upload.String(), "Connection: close") {
				return errors.New("upgrade headers changed")
			}
			stream.frame(wire.TypeBrowserData, []byte("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\nreply"))
			stream.frame(wire.TypeBrowserClosed, nil)
		}
		return nil
	}
	s := startTestProxy(t, func(context.Context, string, uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
		return stream, 0, 0, nil
	})
	conn := dialProxy(t, s)
	_, _ = io.WriteString(conn, "GET http://remote.invalid/ws HTTP/1.1\r\nHost: remote.invalid\r\nConnection: keep-alive, Upgrade\r\nUpgrade: websocket\r\n\r\nframe")
	got, err := io.ReadAll(conn)
	if err != nil || !bytes.HasSuffix(got, []byte("reply")) {
		t.Fatalf("response %q %v", got, err)
	}
}

func TestClosedDownloadDrainsAfterACKFailure(t *testing.T) {
	stream := newTestStream()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := newReceiver(ctx, stream, newCredit(4096), windowBytes)
	ackStarted := make(chan struct{})
	ackRelease := make(chan struct{})
	stream.send = func(_ context.Context, typ uint8, _ []byte) error {
		if typ == wire.TypeFileAck {
			close(ackStarted)
			<-ackRelease
			return io.ErrClosedPipe
		}
		return nil
	}
	local, browser := net.Pipe()
	defer local.Close()
	defer browser.Close()
	go r.run()
	finished := make(chan error, 1)
	go func() { finished <- r.download(local, &timings{}); _ = local.Close() }()
	stream.frame(wire.TypeBrowserData, bytes.Repeat([]byte{1}, 1024))
	first := make([]byte, 1024)
	if _, err := io.ReadFull(browser, first); err != nil {
		t.Fatal(err)
	}
	<-ackStarted
	stream.frame(wire.TypeBrowserData, bytes.Repeat([]byte{2}, chunkBytes))
	stream.frame(wire.TypeBrowserClosed, nil)
	for deadline := time.Now().Add(time.Second); ; {
		r.mu.Lock()
		ended := r.ended
		r.mu.Unlock()
		if ended {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("close not processed while ACK blocked")
		}
		time.Sleep(time.Millisecond)
	}
	close(ackRelease)
	remaining, err := io.ReadAll(browser)
	if err != nil || !bytes.Equal(remaining, bytes.Repeat([]byte{2}, chunkBytes)) {
		t.Fatalf("remaining bytes %d: %v", len(remaining), err)
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func TestUploadCreditValidatedAndCancellationUnblocks(t *testing.T) {
	c := newCredit(4)
	ctx, cancel := context.WithCancel(context.Background())
	if n, err := c.reserve(ctx, 8); err != nil || n != 4 {
		t.Fatalf("reserve %d %v", n, err)
	}
	for _, ack := range []*wirepb.FileTransferAck{{Offset: 5, WindowBytes: 5}, {Offset: 2, WindowBytes: 1}, {Offset: -1, WindowBytes: -1}} {
		p, _ := proto.Marshal(ack)
		if c.acknowledge(p) == nil {
			t.Fatal("accepted invalid credit")
		}
	}
	p, _ := proto.Marshal(&wirepb.FileTransferAck{Offset: 2, WindowBytes: 2})
	if err := c.acknowledge(p); err != nil {
		t.Fatal(err)
	}
	if n, _ := c.reserve(ctx, 8); n != 2 {
		t.Fatalf("reserve %d", n)
	}
	cancel()
	if _, err := c.reserve(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestSlowStreamDoesNotBlockAnotherConnection(t *testing.T) {
	blocked := make(chan struct{})
	s := startTestProxy(t, func(_ context.Context, host string, _ uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
		stream := newTestStream()
		stream.send = func(ctx context.Context, typ uint8, p []byte) error {
			if typ != wire.TypeBrowserData {
				return nil
			}
			if host == "slow.invalid" {
				close(blocked)
				<-ctx.Done()
				return ctx.Err()
			}
			stream.frame(wire.TypeBrowserData, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"))
			stream.frame(wire.TypeBrowserClosed, nil)
			return nil
		}
		return stream, windowBytes, windowBytes, nil
	})
	slow := dialProxy(t, s)
	_, _ = io.WriteString(slow, "GET http://slow.invalid/ HTTP/1.1\r\nHost: slow.invalid\r\n\r\n")
	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("slow stream not started")
	}
	fast := dialProxy(t, s)
	_, _ = io.WriteString(fast, "GET http://fast.invalid/ HTTP/1.1\r\nHost: fast.invalid\r\n\r\n")
	got, err := io.ReadAll(fast)
	if err != nil || !bytes.HasSuffix(got, []byte("ok")) {
		t.Fatalf("independent response %q %v", got, err)
	}
	_ = s.Close()
}

func TestRejectMalformedHeadersAndDoNotDial(t *testing.T) {
	for _, request := range []string{
		"GET /relative HTTP/1.1\r\nHost: remote.invalid\r\n\r\n",
		"CONNECT remote.invalid:0 HTTP/1.1\r\n\r\n",
		"POST http://remote.invalid/ HTTP/1.1\r\nContent-Length: 2\r\nContent-Length: 3\r\n\r\n",
		"GET http://remote.invalid/ HTTP/1.1\r\nX-Large: " + strings.Repeat("x", maximumHeaderBytes) + "\r\n\r\n",
	} {
		s := startTestProxy(t, func(context.Context, string, uint32) (clientruntime.ResourceStream, uint32, uint32, error) {
			t.Error("invalid request dialed")
			return nil, 0, 0, errors.New("unexpected")
		})
		conn := dialProxy(t, s)
		_, _ = io.WriteString(conn, request)
		resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil || resp.StatusCode != 400 {
			t.Fatalf("response %v %v", resp, err)
		}
	}
}
