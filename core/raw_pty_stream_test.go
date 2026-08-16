package core

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestRawPTYBroadcasterPreservesBytesAndReportsSlowConsumer(t *testing.T) {
	broadcaster := newRawPTYBroadcaster()
	subscription := broadcaster.subscribe(context.Background())

	for index := 0; index < rawPTYStreamQueueChunks; index++ {
		broadcaster.publish([]byte{byte(index), 0x00, 0xff, 0x1b})
	}
	dropped := []byte("overflow")
	broadcaster.publish(dropped)

	for index := 0; index < rawPTYStreamQueueChunks; index++ {
		chunk, err := subscription.receive(context.Background())
		if err != nil {
			t.Fatalf("receive chunk %d: %v", index, err)
		}
		want := []byte{byte(index), 0x00, 0xff, 0x1b}
		if !bytes.Equal(chunk, want) {
			t.Fatalf("chunk %d = %x, want %x", index, chunk, want)
		}
	}
	if _, err := subscription.receive(context.Background()); !errors.Is(err, errRawPTYStreamOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	droppedBytes, exitCode := subscription.termination()
	if droppedBytes != uint64(len(dropped)) || exitCode != nil {
		t.Fatalf("termination dropped=%d exit=%v", droppedBytes, exitCode)
	}
}

func TestRawPTYBroadcasterPublishesProcessExitAfterQueuedBytes(t *testing.T) {
	broadcaster := newRawPTYBroadcaster()
	subscription := broadcaster.subscribe(context.Background())
	broadcaster.publish([]byte("tail"))
	code := 17
	broadcaster.close(&code)

	chunk, err := subscription.receive(context.Background())
	if err != nil || string(chunk) != "tail" {
		t.Fatalf("tail chunk=%q err=%v", chunk, err)
	}
	if _, err := subscription.receive(context.Background()); err == nil {
		t.Fatal("closed raw PTY subscription did not reach EOF")
	}
	_, exitCode := subscription.termination()
	if exitCode == nil || *exitCode != code {
		t.Fatalf("exit code = %v, want %d", exitCode, code)
	}
}
