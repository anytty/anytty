//go:build windows

package systemadapter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"unsafe"
)

type memoryWindowsClipboard struct {
	mu           sync.Mutex
	openFailures int
	opened       bool
	nextHandle   uintptr
	clipboard    uintptr
	blocks       map[uintptr][]uint16
}

func newMemoryWindowsClipboard() *memoryWindowsClipboard {
	return &memoryWindowsClipboard{nextHandle: 1, blocks: map[uintptr][]uint16{}}
}

func (api *memoryWindowsClipboard) Open() error {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.openFailures > 0 {
		api.openFailures--
		return errors.New("clipboard busy")
	}
	api.opened = true
	return nil
}

func (api *memoryWindowsClipboard) Close() error {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.opened = false
	return nil
}

func (api *memoryWindowsClipboard) Empty() error {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.clipboard = 0
	return nil
}

func (api *memoryWindowsClipboard) UnicodeTextAvailable() bool {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.clipboard != 0
}

func (api *memoryWindowsClipboard) GetUnicodeText() (uintptr, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.clipboard == 0 {
		return 0, errors.New("clipboard is empty")
	}
	return api.clipboard, nil
}

func (api *memoryWindowsClipboard) SetUnicodeText(handle uintptr) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.clipboard = handle
	return nil
}

func (api *memoryWindowsClipboard) Alloc(size uintptr) (uintptr, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	handle := api.nextHandle
	api.nextHandle++
	api.blocks[handle] = make([]uint16, int(size/2))
	return handle, nil
}

func (api *memoryWindowsClipboard) Free(handle uintptr) {
	api.mu.Lock()
	defer api.mu.Unlock()
	delete(api.blocks, handle)
}

func (api *memoryWindowsClipboard) Lock(handle uintptr) (unsafe.Pointer, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	block := api.blocks[handle]
	if len(block) == 0 {
		return nil, errors.New("invalid memory handle")
	}
	return unsafe.Pointer(&block[0]), nil
}

func (*memoryWindowsClipboard) Unlock(uintptr) error { return nil }

func (api *memoryWindowsClipboard) Size(handle uintptr) (uintptr, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	block := api.blocks[handle]
	if len(block) == 0 {
		return 0, errors.New("invalid memory handle")
	}
	return uintptr(len(block) * 2), nil
}

func TestWindowsClipboardUnicodeRoundTrip(t *testing.T) {
	api := newMemoryWindowsClipboard()
	want := "AnyTTY 中文 clipboard 🙂\r\nnext"
	if err := writeWindowsClipboard(context.Background(), api, want); err != nil {
		t.Fatal(err)
	}
	got, err := readWindowsClipboard(context.Background(), api)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("clipboard text = %q, want %q", got, want)
	}
}

func TestWindowsClipboardRetriesBusyOwner(t *testing.T) {
	api := newMemoryWindowsClipboard()
	api.openFailures = 2
	if err := writeWindowsClipboard(context.Background(), api, "retry"); err != nil {
		t.Fatal(err)
	}
	got, err := readWindowsClipboard(context.Background(), api)
	if err != nil || got != "retry" {
		t.Fatalf("clipboard after retry = %q, %v", got, err)
	}
}

func TestWindowsClipboardBusyOwnerHonorsContext(t *testing.T) {
	api := newMemoryWindowsClipboard()
	api.openFailures = 1000
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := readWindowsClipboard(ctx, api); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("busy clipboard error = %v", err)
	}
}
