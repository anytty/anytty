//go:build !linux && !darwin

package pty

// unsupportedPTY keeps the package buildable on platforms without a PTY
// implementation (Windows, WASM, ...). Every operation fails with
// ErrUnsupported.
type unsupportedPTY struct{}

func newPlatform(Config) PTY { return unsupportedPTY{} }

func (unsupportedPTY) Start() error             { return ErrUnsupported }
func (unsupportedPTY) Read([]byte) (int, error) { return 0, ErrUnsupported }
func (unsupportedPTY) Write([]byte) (int, error) {
	return 0, ErrUnsupported
}
func (unsupportedPTY) Resize(int, int) error   { return ErrUnsupported }
func (unsupportedPTY) Size() (int, int, error) { return 0, 0, ErrUnsupported }
func (unsupportedPTY) ExitCode() int           { return -1 }
func (unsupportedPTY) PID() int                { return 0 }
func (unsupportedPTY) Close() error            { return nil }
