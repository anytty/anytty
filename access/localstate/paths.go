// Package localstate 提供 access 组件的 owner-only 本地路径约定。
// 它只解析平台路径，不加载 credential、不建立连接，也不持有 runtime 状态。
package localstate

import (
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/shared/userdirs"
)

// RemoteRootDir 返回 remote-v2 凭证/身份根目录。
func RemoteRootDir() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "remote-v2")
}

// RemoteCredentialDir 返回客户端 credential 的 owner-only 目录。
func RemoteCredentialDir() string {
	return filepath.Join(RemoteRootDir(), "credentials")
}

// RemoteIdentityDir 返回 DeviceIdentity 的 owner-only 持久目录。
func RemoteIdentityDir() string {
	return filepath.Join(RemoteRootDir(), "identity")
}

// RemoteAccessDir 返回 AccessStore 的 owner-only 持久目录。
func RemoteAccessDir() string {
	return filepath.Join(RemoteRootDir(), "access")
}

// PairingSocketPath 返回 owner-only 本地 PairingExchange socket 路径。
func PairingSocketPath(socketPath string) string {
	return strings.TrimSpace(socketPath) + ".pair"
}

// ControlSocketPath 返回 pool 到 access owner 的 owner-only control socket 路径。
