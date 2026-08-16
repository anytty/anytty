// Package daemon owns the authorized remote DataChannel-to-core-v2 boundary.
package daemon

import (
	"context"
	"fmt"
	"io"
	"time"

	core "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
)

// ScopedTransportServer 是 remote-v2 允许调用的 core-v2 transport 边界。
// 实现方只能按已经验证的 capability scope 建立 protocol session，不能把远程 DataChannel 当作本地无限权限 listener。
type ScopedTransportServer interface {
	// ServeScopedTransport 按 daemon 已验证的 scope 服务同一条 DataChannel 上的 anytty protocol。
	ServeScopedTransport(context.Context, transport.Transport, core.TransportScope) error
}

type outboundDrainer interface {
	Drain(context.Context) error
}

// SessionAcceptor 是远程 DataChannel 进入 core-v2 的唯一授权入口。
// Identity、grant expiry/revoke 和 scope mapping 都属于 daemon；信令结果不能预授权 session。
type SessionAcceptor struct {
	Core        ScopedTransportServer
	Identity    remoteauth.Identity
	AccessStore *remoteauth.AccessStore
	Random      io.Reader
	Now         func() time.Time
}

// ServeDataChannel 在同一 transport 上完成 DeviceHello/CapabilityOpen，成功后才调用 core-v2。
// daemonDTLSFingerprint 必须来自当前 Pion 本端 DTLSTransport；失败会关闭当前 channel，且不创建 protocol session 或 fallback。
func (acceptor SessionAcceptor) ServeDataChannel(ctx context.Context, connection transport.Transport, daemonDTLSFingerprint string) error {
	binding, err := remoteauth.DTLSChannelBinding(daemonDTLSFingerprint)
	if err != nil {
		if connection != nil {
			_ = connection.Close()
		}
		return fmt.Errorf("bind remote data channel: %w", err)
	}
	return acceptor.ServeBoundTransport(ctx, connection, binding)
}

// ServeBoundTransport 在任意已提供可信 TLS/DTLS channel binding 的 transport 上执行同一 v2 auth 状态机。
// PairingExchange 成功后只关闭 transport；只有 client-bound capability 成功才把 scope 交给 core-v2，二者不能复用同一 channel。
func (acceptor SessionAcceptor) ServeBoundTransport(ctx context.Context, connection transport.Transport, binding remoteauth.ChannelBinding) error {
	if acceptor.Core == nil {
		return fmt.Errorf("remote daemon core transport server is not configured")
	}
	if connection == nil {
		return fmt.Errorf("remote daemon data channel transport is not configured")
	}
	handedToCore := false
	defer func() {
		if !handedToCore {
			_ = connection.Close()
		}
	}()
	if acceptor.AccessStore == nil || !acceptor.AccessStore.Available() {
		return fmt.Errorf("remote daemon client access store is not configured")
	}
	if binding.Kind == remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_LOCAL_UNIX {
		return fmt.Errorf("remote daemon session ingress rejects local Unix binding")
	}
	result, err := (remoteauth.ServerHandshake{
		Identity: acceptor.Identity, AccessStore: acceptor.AccessStore,
		Random: acceptor.Random, Now: acceptor.Now,
	}).Accept(ctx, connection, binding)
	if err != nil {
		return fmt.Errorf("authorize remote transport: %w", err)
	}
	if result.Mode == remoteauth.ServerHandshakeModePairing {
		// PairingAccepted 是该 DataChannel 的最后一帧。Send 成功只表示进入底层队列，
		// 必须在关闭 pairing-only transport 前 drain，避免 peer close 丢弃已签发 grant。
		if drainer, ok := connection.(outboundDrainer); ok {
			if err := drainer.Drain(ctx); err != nil {
				return fmt.Errorf("drain PairingExchange response: %w", err)
			}
		}
		return nil
	}
	if result.Mode != remoteauth.ServerHandshakeModeCapability {
		return fmt.Errorf("authorize remote transport: unsupported handshake mode %d", result.Mode)
	}
	claims := result.Claims
	scope := core.TransportScope{
		GrantID:            claims.GrantID,
		GrantExpiresAt:     claims.ExpiresAt,
		PrincipalID:        claims.SubjectKeyFingerprint,
		AllowDaemon:        claims.Scope.AllowDaemon,
		TerminalID:         claims.Scope.TerminalID,
		MachineEventsOnly:  claims.Scope.MachineEventsOnly,
		FileReadMetadata:   claims.Scope.FileReadMetadata,
		FileReadContent:    claims.Scope.FileReadContent,
		FileWriteContent:   claims.Scope.FileWriteContent,
		FileMutate:         claims.Scope.FileMutate,
		ManageClientAccess: claims.Scope.ManageClientAccess,
	}
	handedToCore = true
	return acceptor.Core.ServeScopedTransport(ctx, connection, scope)
}

// GrantActive 让 Direct signaling 在分配 PeerConnection 前复用 daemon-local 持久 grant 真值。
// DataChannel 仍必须完成完整 capability proof，preauth 结果不能进入 core。
func (acceptor SessionAcceptor) GrantActive(grantID string, expiresAt time.Time) bool {
	now := time.Now().UTC()
	if acceptor.Now != nil {
		now = acceptor.Now().UTC()
	}
	return acceptor.AccessStore != nil && acceptor.AccessStore.GrantActive(grantID, expiresAt, now)
}

func (acceptor SessionAcceptor) PairingClaimActive(digest, clientPublicKey []byte, expiresAt time.Time) bool {
	now := time.Now().UTC()
	if acceptor.Now != nil {
		now = acceptor.Now().UTC()
	}
	return acceptor.AccessStore != nil && acceptor.AccessStore.PairingClaimActive(digest, clientPublicKey, expiresAt, now)
}

// PairingAcceptor 是 owner-only local Unix pairing socket 的受限 daemon 入口。
// 它复用同一 DeviceIdentity/AccessStore 与 channel-bound PairingExchange，但永远不持有 core server，因此 capability open 不能借此进入 terminal protocol。
type PairingAcceptor struct {
	Identity    remoteauth.Identity
	AccessStore *remoteauth.AccessStore
	Random      io.Reader
	Now         func() time.Time
}

// ServeBoundTransport 完成一次 local pairing exchange 并关闭当前 transport。
// capability open、缺失 AccessStore 或非 pairing 成功模式都返回错误，不提供 local-owner terminal fallback。
func (acceptor PairingAcceptor) ServeBoundTransport(ctx context.Context, connection transport.Transport, binding remoteauth.ChannelBinding) error {
	if connection == nil {
		return fmt.Errorf("remote daemon pairing transport is not configured")
	}
	if acceptor.AccessStore == nil || !acceptor.AccessStore.Available() {
		return fmt.Errorf("remote daemon pairing store is not configured")
	}
	if binding.Kind != remoteauthpb.ChannelBindingKind_CHANNEL_BINDING_KIND_LOCAL_UNIX {
		return fmt.Errorf("owner-only pairing acceptor requires local Unix binding")
	}
	defer connection.Close()
	result, err := (remoteauth.ServerHandshake{
		Identity: acceptor.Identity, AccessStore: acceptor.AccessStore, PairingOnly: true,
		Random: acceptor.Random, Now: acceptor.Now,
	}).Accept(ctx, connection, binding)
	if err != nil {
		return fmt.Errorf("serve PairingExchange: %w", err)
	}
	if result.Mode != remoteauth.ServerHandshakeModePairing {
		return fmt.Errorf("pairing transport rejected non-pairing auth mode %d", result.Mode)
	}
	return nil
}
