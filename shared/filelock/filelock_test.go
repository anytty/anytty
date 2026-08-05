package filelock

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireContextStopsWaitingAtDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	owner, err := Acquire(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := AcquireContext(ctx, path, false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("lock wait ignored context deadline: %s", elapsed)
	}
}
