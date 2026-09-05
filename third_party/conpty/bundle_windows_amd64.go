//go:build windows && amd64

package conptyruntime

import _ "embed"

const (
	Supported         = true
	DLLSHA256         = "3319b484b80bb53d1f4d0a9eb0ea60fd0f61da69db7280ca43b84215f19245ff"
	OpenConsoleSHA256 = "7f68c840226505004215c0b82d4e502c24b5bc3f4b93c4baaaa19bd679c0def8"
)

//go:embed 1.25.260303002/windows-amd64/conpty.dll
var DLL []byte

//go:embed 1.25.260303002/windows-amd64/OpenConsole.exe
var OpenConsole []byte
