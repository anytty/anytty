package netpath

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRaceStartsThreePathsAndDoesNotWaitForCleanup(t *testing.T) {
	started := make(chan string, 4)
	release := make(chan struct{})
	defer close(release)
	cleanup := make(chan struct{})
	defer close(cleanup)
	var active, peak atomic.Int32
	dialer := &Dialer{Paths: func() []Path { return []Path{{ID: "wifi"}, {ID: "vpn"}, {ID: "cellular"}, {ID: "ethernet"}} }}
	dialer.dial = func(ctx context.Context, path Path, _, _ string) (net.Conn, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		started <- path.ID
		if path.ID == "vpn" {
			<-release
			conn, peer := net.Pipe()
			_ = peer.Close()
			return conn, nil
		}
		<-ctx.Done()
		<-cleanup
		return nil, ctx.Err()
	}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		conn, err := dialer.DialContext(ctx, "tcp", "edge.example:443")
		if conn != nil {
			_ = conn.Close()
		}
		done <- err
	}()
	for range 3 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("network attempts did not start concurrently")
		}
	}
	if peak.Load() != 3 {
		t.Fatalf("peak = %d", peak.Load())
	}
	// Allow the VPN to complete without waiting for either other path to fail.
	release <- struct{}{}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("winner waited for canceled paths to finish cleanup")
	}
}

func TestRacePreservesAllNetworkFailures(t *testing.T) {
	dialer := &Dialer{Paths: func() []Path { return []Path{{ID: "wifi", Name: "wifi"}, {ID: "vpn", Name: "vpn"}} }}
	dialer.dial = func(_ context.Context, path Path, _, _ string) (net.Conn, error) {
		return nil, errors.New(path.ID + " DNS failed")
	}
	_, err := dialer.DialContext(context.Background(), "tcp", "controller.example:443")
	if err == nil || !strings.Contains(err.Error(), "wifi DNS failed") || !strings.Contains(err.Error(), "vpn DNS failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestRaceWaitsForSuccessfulHandshake(t *testing.T) {
	dialer := &Dialer{Paths: func() []Path { return []Path{{ID: "bad"}, {ID: "good"}} }}
	dialer.dial = func(_ context.Context, path Path, _, _ string) (net.Conn, error) {
		conn, peer := net.Pipe()
		_ = peer.Close()
		return &namedConn{Conn: conn, name: path.ID}, nil
	}
	conn, err := dialer.Connect(context.Background(), "tcp", "edge.example:443", func(_ context.Context, conn net.Conn) (net.Conn, error) {
		if conn.(*namedConn).name == "bad" {
			return nil, errors.New("x509: wrong certificate")
		}
		return conn, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if conn.(*namedConn).name != "good" {
		t.Fatal("TCP success bypassed TLS failure")
	}
}

type namedConn struct {
	net.Conn
	name string
}
