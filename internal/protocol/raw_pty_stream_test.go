package protocol

import (
	"testing"
	"time"

	"github.com/anytty/anytty/proto/wire"
)

func TestClientRawPTYQueueOverflowEmitsSyncLostAndCloses(t *testing.T) {
	stream := newClientStreamWithLimits(2, 0, maxClientStreamPayloadBytes, newStreamPayloadBudget(maxClientPayloadBytes))
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("one")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("two")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("lost")})

	frames := stream.channel()
	for _, want := range []string{"one", "two"} {
		select {
		case frame := <-frames:
			if frame.Type != wire.TypePTYOutput || string(frame.Payload) != want {
				t.Fatalf("raw frame = %#v, want %q", frame, want)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for raw PTY prefix")
		}
	}
	select {
	case frame := <-frames:
		if frame.Type != wire.TypeSyncLost {
			t.Fatalf("overflow frame type=%d want=%d", frame.Type, wire.TypeSyncLost)
		}
		dropped, err := wire.DecodeSyncLostPayload(frame.Payload)
		if err != nil || dropped != uint64(len("lost")) {
			t.Fatalf("sync lost dropped=%d err=%v", dropped, err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for explicit raw PTY sync loss")
	}
	select {
	case _, ok := <-frames:
		if ok {
			t.Fatal("raw PTY stream remained open after sync loss")
		}
	case <-time.After(time.Second):
		t.Fatal("raw PTY stream did not close after sync loss")
	}
}

func TestClientRawPTYSyncLostSnapshotRejectsLaterFrames(t *testing.T) {
	stream := newClientStreamWithLimits(1, 0, maxClientStreamPayloadBytes, newStreamPayloadBudget(maxClientPayloadBytes))
	frames := stream.channel()
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("prefix")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("overflow")})
	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("before-snapshot")})

	select {
	case frame := <-frames:
		if frame.Type != wire.TypePTYOutput || string(frame.Payload) != "prefix" {
			t.Fatalf("raw prefix frame = %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for raw PTY prefix")
	}

	wantDropped := uint64(len("overflow") + len("before-snapshot"))
	deadline := time.Now().Add(time.Second)
	for {
		stream.mu.Lock()
		snapshotted := stream.rawSyncLostSent
		stream.mu.Unlock()
		if snapshotted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("SyncLost snapshot was not generated")
		}
		time.Sleep(time.Millisecond)
	}

	stream.send(StreamFrame{Type: wire.TypePTYOutput, Payload: []byte("after-snapshot")})
	stream.mu.Lock()
	gotAfterLateSend := stream.rawDroppedBytes
	stream.mu.Unlock()
	if gotAfterLateSend != wantDropped {
		t.Fatalf("late frame changed snapshotted dropped bytes: got=%d want=%d", gotAfterLateSend, wantDropped)
	}

	var encodedDropped uint64
	select {
	case frame := <-frames:
		if frame.Type != wire.TypeSyncLost {
			t.Fatalf("overflow frame type=%d want=%d", frame.Type, wire.TypeSyncLost)
		}
		var err error
		encodedDropped, err = wire.DecodeSyncLostPayload(frame.Payload)
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for snapshotted SyncLost")
	}
	select {
	case _, ok := <-frames:
		if ok {
			t.Fatal("raw PTY stream remained open after snapshotted SyncLost")
		}
	case <-time.After(time.Second):
		t.Fatal("raw PTY stream did not close after snapshotted SyncLost")
	}
	stream.mu.Lock()
	finalDropped := stream.rawDroppedBytes
	closed := stream.closed
	stream.mu.Unlock()
	if encodedDropped != wantDropped || finalDropped != encodedDropped || !closed {
		t.Fatalf("SyncLost snapshot encoded=%d final=%d want=%d closed=%t", encodedDropped, finalDropped, wantDropped, closed)
	}
}
