//go:build windows

package terminalhost

import (
	"sync"
	"testing"
	"time"
)

func TestWindowsResizePollUsesConfiguredSizeDescriptor(t *testing.T) {
	oldSize := windowsTerminalSize
	t.Cleanup(func() { windowsTerminalSize = oldSize })

	var mu sync.Mutex
	cols, rows := 80, 24
	var observedFD uintptr
	firstRead := make(chan struct{})
	var firstReadOnce sync.Once
	windowsTerminalSize = func(fd uintptr) (int, int, error) {
		mu.Lock()
		defer mu.Unlock()
		observedFD = fd
		firstReadOnce.Do(func() { close(firstRead) })
		return cols, rows, nil
	}

	signals, stop := defaultResizeSignalFactory(42)
	defer stop()
	select {
	case <-firstRead:
	case <-time.After(5 * windowsResizePollInterval):
		t.Fatal("Windows resize poll did not read its initial window size")
	}
	mu.Lock()
	cols, rows = 132, 43
	mu.Unlock()

	select {
	case <-signals:
	case <-time.After(5 * windowsResizePollInterval):
		t.Fatal("Windows resize poll did not publish a changed window size")
	}
	mu.Lock()
	defer mu.Unlock()
	if observedFD != 42 {
		t.Fatalf("Windows resize poll queried fd %d, want output fd 42", observedFD)
	}
}
