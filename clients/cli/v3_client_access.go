package cli

import (
	"github.com/anytty/anytty/access/localstate"
)

// v3PairingSocketPath 返回 owner-only 本地 PairingExchange socket 路径。
func v3PairingSocketPath(socketPath string) string {
	return localstate.PairingSocketPath(socketPath)
}
