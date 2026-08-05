package memory

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestPairRoundTrip(t *testing.T) {
	client, server := NewPair()
	defer client.Close()
	defer server.Close()

	payload := []byte("hello")
	if err := client.Send(payload); err != nil {
		t.Fatalf("client send failed: %v", err)
	}

	got, err := server.Recv()
	if err != nil {
		t.Fatalf("server recv failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload: %q", string(got))
	}
}

func TestSendAfterLocalCloseReturnsEOF(t *testing.T) {
	client, server := NewPair()
	defer server.Close()

	if err := client.Close(); err != nil {
		t.Fatalf("client close failed: %v", err)
	}

	if err := client.Send([]byte("hello")); err == nil || err != io.EOF {
		t.Fatalf("expected EOF after local close, got %v", err)
	}
}

func TestSendAfterPeerCloseReturnsEOF(t *testing.T) {
	client, server := NewPair()
	defer client.Close()

	if err := server.Close(); err != nil {
		t.Fatalf("server close failed: %v", err)
	}

	if err := client.Send([]byte("hello")); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after peer close, got %v", err)
	}
	if _, err := client.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF from recv after peer close, got %v", err)
	}
}

func TestRecvDrainsQueuedFramesBeforePeerEOF(t *testing.T) {
	client, server := NewPair()
	defer client.Close()

	for _, payload := range [][]byte{[]byte("response"), []byte("event")} {
		if err := server.Send(payload); err != nil {
			t.Fatalf("server send failed: %v", err)
		}
	}
	if err := server.Close(); err != nil {
		t.Fatalf("server close failed: %v", err)
	}

	for _, want := range [][]byte{[]byte("response"), []byte("event")} {
		got, err := client.Recv()
		if err != nil {
			t.Fatalf("recv queued frame failed: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("unexpected queued frame: got %q want %q", got, want)
		}
	}
	if _, err := client.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after queued frames, got %v", err)
	}
}
