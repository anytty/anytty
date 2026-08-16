//go:build windows

package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowsClipboardUnicodeText = 13
	windowsGlobalMoveable       = 0x0002
	windowsGlobalZeroInit       = 0x0040
	windowsClipboardRetryDelay  = 10 * time.Millisecond
)

type windowsClipboardAPI interface {
	Open() error
	Close() error
	Empty() error
	UnicodeTextAvailable() bool
	GetUnicodeText() (uintptr, error)
	SetUnicodeText(uintptr) error
	Alloc(uintptr) (uintptr, error)
	Free(uintptr)
	Lock(uintptr) (unsafe.Pointer, error)
	Unlock(uintptr) error
	Size(uintptr) (uintptr, error)
}

var nativeWindowsClipboard windowsClipboardAPI = win32ClipboardAPI{}

func writeSystemClipboard(ctx context.Context, text string) error {
	return writeWindowsClipboard(ctx, nativeWindowsClipboard, text)
}

func readSystemClipboard(ctx context.Context) (string, error) {
	return readWindowsClipboard(ctx, nativeWindowsClipboard)
}

func writeWindowsClipboard(ctx context.Context, api windowsClipboardAPI, text string) (returnErr error) {
	encoded, err := windows.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("encode Unicode clipboard text: %w", err)
	}
	handle, err := api.Alloc(uintptr(len(encoded) * 2))
	if err != nil {
		return err
	}
	owned := true
	defer func() {
		if owned {
			api.Free(handle)
		}
	}()
	pointer, err := api.Lock(handle)
	if err != nil {
		return err
	}
	copy(unsafe.Slice((*uint16)(pointer), len(encoded)), encoded)
	if err := api.Unlock(handle); err != nil {
		return err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := openWindowsClipboard(ctx, api); err != nil {
		return err
	}
	defer func() { returnErr = errors.Join(returnErr, api.Close()) }()
	if err := api.Empty(); err != nil {
		return err
	}
	if err := api.SetUnicodeText(handle); err != nil {
		return err
	}
	// SetClipboardData 成功后，HGLOBAL 生命周期转交给系统剪贴板。
	owned = false
	return nil
}

func readWindowsClipboard(ctx context.Context, api windowsClipboardAPI) (text string, returnErr error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := openWindowsClipboard(ctx, api); err != nil {
		return "", err
	}
	defer func() { returnErr = errors.Join(returnErr, api.Close()) }()
	if !api.UnicodeTextAvailable() {
		return "", nil
	}
	handle, err := api.GetUnicodeText()
	if err != nil {
		return "", err
	}
	pointer, err := api.Lock(handle)
	if err != nil {
		return "", err
	}
	defer func() { returnErr = errors.Join(returnErr, api.Unlock(handle)) }()
	size, err := api.Size(handle)
	if err != nil {
		return "", err
	}
	if size < 2 {
		return "", nil
	}
	units := unsafe.Slice((*uint16)(pointer), int(size/2))
	end := 0
	for end < len(units) && units[end] != 0 {
		end++
	}
	return windows.UTF16ToString(units[:end]), nil
}

func openWindowsClipboard(ctx context.Context, api windowsClipboardAPI) error {
	var lastErr error
	for {
		if err := api.Open(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("open Windows clipboard: %w", errors.Join(lastErr, ctx.Err()))
		case <-time.After(windowsClipboardRetryDelay):
		}
	}
}

var (
	user32Clipboard        = windows.NewLazySystemDLL("user32.dll")
	kernel32Clipboard      = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard      = user32Clipboard.NewProc("OpenClipboard")
	procCloseClipboard     = user32Clipboard.NewProc("CloseClipboard")
	procEmptyClipboard     = user32Clipboard.NewProc("EmptyClipboard")
	procClipboardAvailable = user32Clipboard.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData   = user32Clipboard.NewProc("GetClipboardData")
	procSetClipboardData   = user32Clipboard.NewProc("SetClipboardData")
	procGlobalAlloc        = kernel32Clipboard.NewProc("GlobalAlloc")
	procGlobalFree         = kernel32Clipboard.NewProc("GlobalFree")
	procGlobalLock         = kernel32Clipboard.NewProc("GlobalLock")
	procGlobalUnlock       = kernel32Clipboard.NewProc("GlobalUnlock")
	procGlobalSize         = kernel32Clipboard.NewProc("GlobalSize")
	procGetConsoleWindow   = kernel32Clipboard.NewProc("GetConsoleWindow")
)

type win32ClipboardAPI struct{}

func (win32ClipboardAPI) Open() error {
	owner, _, callErr := procGetConsoleWindow.Call()
	if owner == 0 {
		return clipboardCallError("GetConsoleWindow", callErr)
	}
	return clipboardBOOL("OpenClipboard", procOpenClipboard, owner)
}

func (win32ClipboardAPI) Close() error {
	return clipboardBOOL("CloseClipboard", procCloseClipboard)
}

func (win32ClipboardAPI) Empty() error {
	return clipboardBOOL("EmptyClipboard", procEmptyClipboard)
}

func (win32ClipboardAPI) UnicodeTextAvailable() bool {
	result, _, _ := procClipboardAvailable.Call(windowsClipboardUnicodeText)
	return result != 0
}

func (win32ClipboardAPI) GetUnicodeText() (uintptr, error) {
	handle, _, callErr := procGetClipboardData.Call(windowsClipboardUnicodeText)
	if handle == 0 {
		return 0, clipboardCallError("GetClipboardData", callErr)
	}
	return handle, nil
}

func (win32ClipboardAPI) SetUnicodeText(handle uintptr) error {
	result, _, callErr := procSetClipboardData.Call(windowsClipboardUnicodeText, handle)
	if result == 0 {
		return clipboardCallError("SetClipboardData", callErr)
	}
	return nil
}

func (win32ClipboardAPI) Alloc(size uintptr) (uintptr, error) {
	handle, _, callErr := procGlobalAlloc.Call(windowsGlobalMoveable|windowsGlobalZeroInit, size)
	if handle == 0 {
		return 0, clipboardCallError("GlobalAlloc", callErr)
	}
	return handle, nil
}

func (win32ClipboardAPI) Free(handle uintptr) {
	_, _, _ = procGlobalFree.Call(handle)
}

func (win32ClipboardAPI) Lock(handle uintptr) (unsafe.Pointer, error) {
	pointer, _, callErr := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, clipboardCallError("GlobalLock", callErr)
	}
	return unsafe.Pointer(pointer), nil
}

func (win32ClipboardAPI) Unlock(handle uintptr) error {
	result, _, callErr := procGlobalUnlock.Call(handle)
	if result == 0 && !clipboardCallSucceeded(callErr) {
		return clipboardCallError("GlobalUnlock", callErr)
	}
	return nil
}

func (win32ClipboardAPI) Size(handle uintptr) (uintptr, error) {
	size, _, callErr := procGlobalSize.Call(handle)
	if size == 0 {
		return 0, clipboardCallError("GlobalSize", callErr)
	}
	return size, nil
}

func clipboardBOOL(name string, procedure *windows.LazyProc, args ...uintptr) error {
	result, _, callErr := procedure.Call(args...)
	if result == 0 {
		return clipboardCallError(name, callErr)
	}
	return nil
}

func clipboardCallSucceeded(err error) bool {
	return err == nil || errors.Is(err, syscall.Errno(0))
}

func clipboardCallError(name string, err error) error {
	if clipboardCallSucceeded(err) {
		return fmt.Errorf("%s failed", name)
	}
	return fmt.Errorf("%s: %w", name, err)
}
