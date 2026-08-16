package main

import (
	"path/filepath"

	"github.com/anytty/anytty/shared/userdirs"
)

// v3RemoteCredentialDir 返回 CLI host adapter 的 owner-only credential 目录。
// 它只提供平台路径，不解析 credential、不建立连接，也不持有 client runtime 状态。
func v3RemoteCredentialDir() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "remote-v2", "credentials")
}

// v3RemoteIdentityDir 返回 daemon DeviceIdentity 的 owner-only 持久目录。
// Identity 是 Direct/SSH 端到端认证真值，不属于 Cloud 配置。
func v3RemoteIdentityDir() string {
	return filepath.Join(filepath.Dir(v3RemoteCredentialDir()), "identity")
}
