package testkit

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

const (
	ModeCollaborator = "collaborator"
	StateExited      = string(corev2.TerminalStateExited)
)

type Daemon struct {
	socketPath string
	server     *corev2.Server
	done       chan error
	cancel     context.CancelFunc
}

func StartDaemon(t testing.TB, ctx context.Context, socketName string) *Daemon {
	t.Helper()
	if socketName == "" {
		socketName = "anytty.sock"
	}
	runCtx, cancel := context.WithCancel(ctx)
	socketPath := filepath.Join(t.TempDir(), socketName)
	srv := corev2.NewServer(corev2.WithSocketPath(socketPath))
	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe(runCtx)
	}()
	d := &Daemon{
		socketPath: socketPath,
		server:     srv,
		done:       done,
		cancel:     cancel,
	}
	t.Cleanup(func() {
		d.Shutdown(t, 3*time.Second)
	})
	if err := WaitSocket(socketPath, 5*time.Second); err != nil {
		t.Fatalf("server socket never appeared: %v", err)
	}
	return d
}

func (d *Daemon) SocketPath() string {
	if d == nil {
		return ""
	}
	return d.socketPath
}

func (d *Daemon) NewClient(t testing.TB, ctx context.Context) *protocol.Client {
	t.Helper()
	client, err := DialClient(ctx, d.SocketPath())
	if err != nil {
		t.Fatalf("dial daemon client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func (d *Daemon) TerminalState(ctx context.Context, terminalID string) (string, error) {
	if d == nil || d.server == nil {
		return "", errors.New("test daemon is nil")
	}
	info, err := d.server.GetTerminal(terminalID)
	if err != nil {
		return "", err
	}
	return string(info.State), nil
}

func (d *Daemon) Shutdown(t testing.TB, timeout time.Duration) {
	t.Helper()
	if d == nil || d.server == nil {
		return
	}
	if d.cancel != nil {
		d.cancel()
	}
	_ = d.server.Shutdown(context.Background())
	select {
	case <-d.done:
	case <-time.After(timeout):
		t.Fatal("server did not stop in time")
	}
	d.server = nil
}

func DialClient(ctx context.Context, socketPath string) (*protocol.Client, error) {
	transport, err := unixtransport.Dial(socketPath)
	if err != nil {
		return nil, err
	}
	client := protocol.NewClient(transport)
	if err := client.Hello(ctx, protocol.Hello{Version: wire.Version}); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func WaitSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := unixtransport.Dial(path)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("socket did not appear in time")
}
