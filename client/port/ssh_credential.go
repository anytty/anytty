package port

import (
	"context"

	"github.com/anytty/anytty/client/endpoint"
	golangssh "golang.org/x/crypto/ssh"
)

// SSHCredentialSource 解析当前平台 secure store 中的 SSH credential ref。
// 返回值只在单次 route attempt 内存活；Endpoint registry 永远不包含 password、private key 或 signer body。
type SSHCredentialSource interface {
	// ResolveSSHCredential 必须按 descriptor kind 返回可用于 x/crypto/ssh 的认证方法，并提供精确 cleanup。
	ResolveSSHCredential(context.Context, string, *endpoint.CredentialDescriptor) (SSHCredential, error)
}

// SSHCredential 是单次 Go SSH handshake 使用的内存凭据。
// AuthMethods 可以封装 agent、不可导出 signer 或 password；Close 必须释放 agent socket 或平台 handle。
type SSHCredential struct {
	AuthMethods []golangssh.AuthMethod
	Close       func() error
}

// Release 幂等语义由 credential source 保证；nil cleanup 表示认证材料没有独立资源。
func (credential SSHCredential) Release() error {
	if credential.Close == nil {
		return nil
	}
	return credential.Close()
}
