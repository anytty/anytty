//go:build windows && !amd64 && !arm64

package conptyruntime

const Supported = false

const (
	DLLSHA256         = ""
	OpenConsoleSHA256 = ""
)

var (
	DLL         []byte
	OpenConsole []byte
)
