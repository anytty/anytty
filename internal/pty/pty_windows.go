//go:build windows
// +build windows

package pty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	conptyruntime "github.com/anytty/anytty/third_party/conpty"
	"golang.org/x/sys/windows"
)

const (
	_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE = 0x20016 // nolint:revive
	conPTYDLLOverrideEnv                 = "ANYTTY_CONPTY_DLL"
)

var (
	errClosedConPty = errors.New("pseudo console is closed")
	errNotStarted   = errors.New("process not started")
)

// conPty is a Windows console pseudo-terminal.
// It uses Windows pseudo console API to create a console that can be used to
// start processes attached to it.
//
// See: https://docs.microsoft.com/en-us/windows/console/creating-a-pseudoconsole-session
type conPty struct {
	handle          windows.Handle
	inPipe, outPipe *os.File
	api             *bundledConPTYAPI
	mtx             sync.RWMutex
	consoleOnce     sync.Once
	pipesOnce       sync.Once
	consoleErr      error
	pipesErr        error
}

type bundledConPTYAPI struct {
	dll    *windows.DLL
	create *windows.Proc
	resize *windows.Proc
	close  *windows.Proc
}

var _ Pty = &conPty{}

func newPty() (ConPty, error) {
	ptyIn, inPipeOurs, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipes for pseudo console: %w", err)
	}
	closeInputPipes := func() {
		_ = ptyIn.Close()
		_ = inPipeOurs.Close()
	}

	outPipeOurs, ptyOut, err := os.Pipe()
	if err != nil {
		closeInputPipes()
		return nil, fmt.Errorf("failed to create pipes for pseudo console: %w", err)
	}
	closeAllPipes := func() {
		closeInputPipes()
		_ = outPipeOurs.Close()
		_ = ptyOut.Close()
	}

	var hpc windows.Handle
	coord := windows.Coord{X: 80, Y: 25}
	api, found, err := loadBundledConPTY()
	if err != nil {
		closeAllPipes()
		return nil, fmt.Errorf("load bundled pseudo console: %w", err)
	}
	if found {
		err = api.createPseudoConsole(coord, windows.Handle(ptyIn.Fd()), windows.Handle(ptyOut.Fd()), &hpc)
	} else {
		err = windows.CreatePseudoConsole(coord, windows.Handle(ptyIn.Fd()), windows.Handle(ptyOut.Fd()), 0, &hpc)
	}
	if err != nil {
		if api != nil {
			_ = api.dll.Release()
		}
		closeAllPipes()
		return nil, fmt.Errorf("failed to create pseudo console: %w", err)
	}

	if err := ptyOut.Close(); err != nil {
		closePseudoConsole(api, hpc)
		if api != nil {
			_ = api.dll.Release()
		}
		closeAllPipes()
		return nil, fmt.Errorf("failed to close pseudo console handle: %w", err)
	}
	if err := ptyIn.Close(); err != nil {
		closePseudoConsole(api, hpc)
		if api != nil {
			_ = api.dll.Release()
		}
		closeAllPipes()
		return nil, fmt.Errorf("failed to close pseudo console handle: %w", err)
	}

	return &conPty{
		handle:  hpc,
		inPipe:  inPipeOurs,
		outPipe: outPipeOurs,
		api:     api,
	}, nil
}

// Close implements Pty.
func (p *conPty) Close() error {
	return errors.Join(p.ClosePseudoConsole(), p.ClosePipes())
}

// Command implements Pty.
func (p *conPty) Command(name string, args ...string) *Cmd {
	c := &Cmd{
		pty:  p,
		Path: name,
		Args: append([]string{name}, args...),
	}
	return c
}

// CommandContext implements Pty.
func (p *conPty) CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	if ctx == nil {
		panic("nil context")
	}
	c := p.Command(name, args...)
	c.ctx = ctx
	c.Cancel = func() error {
		return c.Process.Kill()
	}
	return c
}

// Name implements Pty.
func (*conPty) Name() string {
	return "windows-pty"
}

// Read implements Pty.
func (p *conPty) Read(b []byte) (n int, err error) {
	return p.outPipe.Read(b)
}

// Resize implements Pty.
func (p *conPty) Resize(width int, height int) error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if p.handle == 0 {
		return errClosedConPty
	}
	coord := windows.Coord{X: int16(width), Y: int16(height)}
	var err error
	if p.api != nil {
		err = p.api.resizePseudoConsole(p.handle, coord)
	} else {
		err = windows.ResizePseudoConsole(p.handle, coord)
	}
	if err != nil {
		return fmt.Errorf("failed to resize pseudo console: %w", err)
	}
	return nil
}

// Write implements Pty.
func (p *conPty) Write(b []byte) (n int, err error) {
	return p.inPipe.Write(b)
}

// Fd implements Pty.
func (p *conPty) Fd() uintptr {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return uintptr(p.handle)
}

// InputPipe implements ConPty.
func (p *conPty) InputPipe() *os.File {
	return p.inPipe
}

// OutputPipe implements ConPty.
func (p *conPty) OutputPipe() *os.File {
	return p.outPipe
}

// ClosePseudoConsole implements ConPty.
func (p *conPty) ClosePseudoConsole() error {
	p.consoleOnce.Do(func() {
		p.mtx.Lock()
		handle := p.handle
		api := p.api
		p.handle = 0
		p.api = nil
		p.mtx.Unlock()
		if handle != 0 {
			closePseudoConsole(api, handle)
		}
		if api != nil {
			p.consoleErr = api.dll.Release()
		}
	})
	return p.consoleErr
}

// ClosePipes implements ConPty.
func (p *conPty) ClosePipes() error {
	p.pipesOnce.Do(func() {
		p.pipesErr = errors.Join(p.inPipe.Close(), p.outPipe.Close())
	})
	return p.pipesErr
}

// updateProcThreadAttribute updates the passed in attribute list to contain the entry necessary for use with
// CreateProcess.
func (p *conPty) updateProcThreadAttribute(attrList *windows.ProcThreadAttributeListContainer) error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.handle == 0 {
		return errClosedConPty
	}

	if err := attrList.Update(
		_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		pointerFromHandle(p.handle),
		unsafe.Sizeof(p.handle),
	); err != nil {
		return fmt.Errorf("failed to update proc thread attributes for pseudo console: %w", err)
	}

	return nil
}

// Legacy inbox ConPTY rebuilds VT from its screen buffer, which loses styled
// blanks and scroll events. Windows builds embed the newer passthrough runtime.
func loadBundledConPTY() (*bundledConPTYAPI, bool, error) {
	dllPath, found, err := bundledConPTYPath()
	if err != nil || !found {
		return nil, found, err
	}
	dll, err := windows.LoadDLL(dllPath)
	if err != nil {
		return nil, true, err
	}
	releaseOnError := func(err error) (*bundledConPTYAPI, bool, error) {
		_ = dll.Release()
		return nil, true, err
	}
	create, err := dll.FindProc("ConptyCreatePseudoConsole")
	if err != nil {
		return releaseOnError(err)
	}
	resize, err := dll.FindProc("ConptyResizePseudoConsole")
	if err != nil {
		return releaseOnError(err)
	}
	closeProc, err := dll.FindProc("ConptyClosePseudoConsole")
	if err != nil {
		return releaseOnError(err)
	}
	return &bundledConPTYAPI{dll: dll, create: create, resize: resize, close: closeProc}, true, nil
}

func bundledConPTYPath() (string, bool, error) {
	dllPath := os.Getenv(conPTYDLLOverrideEnv)
	if dllPath == "" {
		if !conptyruntime.Supported {
			return "", false, nil
		}
		materialized, err := materializeBundledConPTY()
		return materialized, true, err
	}
	absolute, err := filepath.Abs(dllPath)
	if err != nil {
		return "", true, err
	}
	if _, err := os.Stat(absolute); err != nil {
		return "", true, err
	}
	openConsole := filepath.Join(filepath.Dir(absolute), "OpenConsole.exe")
	if _, err := os.Stat(openConsole); err != nil {
		return "", true, fmt.Errorf("bundled OpenConsole.exe: %w", err)
	}
	return absolute, true, nil
}

func materializeBundledConPTY() (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache: %w", err)
	}
	directoryName := fmt.Sprintf(
		"conpty-%s-%s-%s-%s",
		conptyruntime.Version,
		runtime.GOARCH,
		conptyruntime.DLLSHA256[:12],
		conptyruntime.OpenConsoleSHA256[:12],
	)
	directory := filepath.Join(cacheRoot, "AnyTTY", "runtime", directoryName)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create ConPTY runtime directory: %w", err)
	}

	openConsolePath := filepath.Join(directory, "OpenConsole.exe")
	if err := ensureBundledFile(openConsolePath, conptyruntime.OpenConsole, conptyruntime.OpenConsoleSHA256); err != nil {
		return "", fmt.Errorf("materialize OpenConsole.exe: %w", err)
	}
	dllPath := filepath.Join(directory, "conpty.dll")
	if err := ensureBundledFile(dllPath, conptyruntime.DLL, conptyruntime.DLLSHA256); err != nil {
		return "", fmt.Errorf("materialize conpty.dll: %w", err)
	}
	return dllPath, nil
}

func ensureBundledFile(path string, contents []byte, expectedSHA256 string) error {
	if !bytesHaveSHA256(contents, expectedSHA256) {
		return errors.New("embedded content failed SHA-256 verification")
	}
	matches, err := fileHasSHA256(path, expectedSHA256)
	if err == nil && matches {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	from, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
		// A concurrent AnyTTY process may have installed the same verified file.
		if matches, verifyErr := fileHasSHA256(path, expectedSHA256); verifyErr == nil && matches {
			return nil
		}
		return err
	}
	matches, err = fileHasSHA256(path, expectedSHA256)
	if err != nil {
		return err
	}
	if !matches {
		return errors.New("materialized content failed SHA-256 verification")
	}
	return nil
}

func bytesHaveSHA256(contents []byte, expected string) bool {
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:]) == expected
}

func fileHasSHA256(path string, expected string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return false, err
	}
	return hex.EncodeToString(digest.Sum(nil)) == expected, nil
}

func (api *bundledConPTYAPI) createPseudoConsole(size windows.Coord, input, output windows.Handle, handle *windows.Handle) error {
	result, _, _ := api.create.Call(packCoord(size), uintptr(input), uintptr(output), 0, uintptr(unsafe.Pointer(handle)))
	return checkHRESULT("ConptyCreatePseudoConsole", result)
}

func (api *bundledConPTYAPI) resizePseudoConsole(handle windows.Handle, size windows.Coord) error {
	result, _, _ := api.resize.Call(uintptr(handle), packCoord(size))
	return checkHRESULT("ConptyResizePseudoConsole", result)
}

func closePseudoConsole(api *bundledConPTYAPI, handle windows.Handle) {
	if api != nil {
		_, _, _ = api.close.Call(uintptr(handle))
		return
	}
	windows.ClosePseudoConsole(handle)
}

func packCoord(coord windows.Coord) uintptr {
	return uintptr(uint32(uint16(coord.X)) | uint32(uint16(coord.Y))<<16)
}

func pointerFromHandle(handle windows.Handle) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&handle))
}

func checkHRESULT(operation string, result uintptr) error {
	hresult := uint32(result)
	if hresult&0x80000000 == 0 {
		return nil
	}
	return fmt.Errorf("%s failed with HRESULT 0x%08x", operation, hresult)
}
