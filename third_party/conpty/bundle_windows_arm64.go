//go:build windows && arm64

package conptyruntime

import _ "embed"

const (
	Supported         = true
	DLLSHA256         = "b3a9f975c51d5b9b96d290bf784e990e3f85203b063c21614457353f566266ff"
	OpenConsoleSHA256 = "7fa560be0c9b6c81db5d1b855a4a6e0f3ee5a2ea75d36eebb3ae44443b8d4866"
)

//go:embed 1.25.260303002/windows-arm64/conpty.dll
var DLL []byte

//go:embed 1.25.260303002/windows-arm64/OpenConsole.exe
var OpenConsole []byte
