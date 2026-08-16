package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	golangssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AgentCredentialSource 让 desktop CLI/TUI 通过 SSH_AUTH_SOCK 使用不可导出的 agent signer。
// credential ref 只需使用 `ssh:` namespace；具体 key 由 agent 按 server challenge 选择，不进入 Endpoint registry。
type AgentCredentialSource struct {
	Socket string
	Dialer *net.Dialer
}

// Available 只报告当前进程是否具备解析该 SSH credential ref 的 agent primitive。
// 该检查不打开 agent、不读取 signer body；真实连接仍必须在 ResolveSSHCredential 时 fail closed。
func (source AgentCredentialSource) Available(reference string) bool {
	if !strings.HasPrefix(strings.TrimSpace(reference), "ssh:") {
		return false
	}
	socket := strings.TrimSpace(source.Socket)
	if socket == "" {
		socket = strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	}
	return socket != ""
}

// ResolveSSHCredential 打开当前 agent socket 并返回 signer auth method。
// password/private-key secure store 由其它平台实现同一 port；本实现不会读取 key file 或环境中的 password。
func (source AgentCredentialSource) ResolveSSHCredential(ctx context.Context, reference string, _ *endpoint.CredentialDescriptor) (port.SSHCredential, error) {
	if !strings.HasPrefix(strings.TrimSpace(reference), "ssh:") {
		return port.SSHCredential{}, fmt.Errorf("SSH agent credential ref must use ssh: namespace")
	}
	socket := strings.TrimSpace(source.Socket)
	if socket == "" {
		socket = strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	}
	if socket == "" {
		return port.SSHCredential{}, fmt.Errorf("SSH_AUTH_SOCK is unavailable")
	}
	dialer := source.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	connection, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return port.SSHCredential{}, fmt.Errorf("connect SSH agent: %w", err)
	}
	signers, err := agent.NewClient(connection).Signers()
	if err != nil {
		_ = connection.Close()
		return port.SSHCredential{}, fmt.Errorf("list SSH agent signers: %w", err)
	}
	if len(signers) == 0 {
		_ = connection.Close()
		return port.SSHCredential{}, fmt.Errorf("SSH agent has no signers")
	}
	var closeOnce sync.Once
	return port.SSHCredential{
		AuthMethods: []golangssh.AuthMethod{golangssh.PublicKeys(signers...)},
		Close: func() (closeErr error) {
			closeOnce.Do(func() { closeErr = connection.Close() })
			return closeErr
		},
	}, nil
}

var _ port.SSHCredentialSource = AgentCredentialSource{}
