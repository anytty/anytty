package binding

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

func listenBrowser(t *testing.T, engine *Engine, session uint64, stopPort uint32) *bindingpb.BrowserProxyListenResult {
	t.Helper()
	payload, _ := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_BrowserProxyListen{BrowserProxyListen: &bindingpb.BrowserProxyListenRequest{RequestId: "listen", SessionHandle: session, Stop: stopPort != 0, Port: stopPort}}})
	handle, err := engine.EngineCommand(payload)
	if err != nil {
		t.Fatal(err)
	}
	result := nextBindingEvent(t, engine).GetBrowserProxyListen()
	if result == nil || result.GetError() != nil || result.GetOperationHandle() != handle || result.GetSessionHandle() != session {
		t.Fatalf("listen result %v", result)
	}
	if err := engine.Release(handle); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBrowserProxySessionListenerLifecycle(t *testing.T) {
	for _, retire := range []string{"stop", "session", "renderer", "engine", "remote"} {
		t.Run(retire, func(t *testing.T) {
			session := newBindingSession()
			engine, err := NewEngine(&bindingHost{session: session})
			if err != nil {
				t.Fatal(err)
			}
			defer engine.Close()
			handle := openBindingSession(t, engine)
			port := listenBrowser(t, engine, handle, 0).GetPort()
			if port == 0 {
				t.Fatal("missing listener port")
			}
			if again := listenBrowser(t, engine, handle, 0).GetPort(); again != port {
				t.Fatal("duplicate listener")
			}
			address := fmt.Sprintf("127.0.0.1:%d", port)
			conn, err := net.DialTimeout("tcp4", address, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			switch retire {
			case "stop":
				listenBrowser(t, engine, handle, port)
			case "session":
				err = engine.CloseSession(handle)
			case "renderer":
				_, err = engine.AttachRenderer()
			case "engine":
				err = engine.Close()
			case "remote":
				err = session.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			if _, err := conn.Read(make([]byte, 1)); err == nil {
				t.Fatal("old socket remained open")
			} else if failure, ok := err.(net.Error); ok && failure.Timeout() {
				t.Fatal("socket cleanup timed out")
			}
			if fresh, err := net.DialTimeout("tcp4", address, 50*time.Millisecond); err == nil {
				fresh.Close()
				t.Fatal("listener remained open")
			}
		})
	}
}
