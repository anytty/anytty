package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/remoteauthpb"
	remotev2daemon "github.com/anytty/anytty/remote/daemon"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
	"google.golang.org/protobuf/proto"
)

// v3ClientAccessRuntime 是当前 daemon 进程唯一装配的 DeviceIdentity、AccessStore 与 local pairing socket。
// local protocol、Direct/SSH 都必须引用这里的同一 Identity/Store，不能各自加载第二份授权真值。
type v3ClientAccessRuntime struct {
	Identity      remoteauth.Identity
	Store         *remoteauth.AccessStore
	PairingSocket string
}

// Close 释放 daemon-local AccessStore 的唯一进程 owner lock。
// 调用方必须先停止 pairing listener、Direct ingress 和 core session，再结束该 runtime。
func (runtime v3ClientAccessRuntime) Close() error {
	if runtime.Store == nil {
		return nil
	}
	return runtime.Store.Close()
}

func loadV3ClientAccessRuntime(socketPath string) (v3ClientAccessRuntime, error) {
	identity, err := remoteauth.LoadOrCreateLocalIdentity(v3RemoteIdentityDir())
	if err != nil {
		return v3ClientAccessRuntime{}, err
	}
	store, err := remoteauth.LoadAccessStore(v3RemoteAccessDir(), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		return v3ClientAccessRuntime{}, err
	}
	return v3ClientAccessRuntime{Identity: identity, Store: store, PairingSocket: v3PairingSocketPath(socketPath)}, nil
}

func v3RemoteAccessDir() string {
	return filepath.Join(filepath.Dir(v3RemoteIdentityDir()), "access")
}

func v3PairingSocketPath(socketPath string) string {
	return strings.TrimSpace(socketPath) + ".pair"
}

// startV3PairingListener 启动 owner-only Unix PairingExchange listener。
// listener 只运行 remoteauth v2，不接受 anytty Hello 或 capability session；context 取消会关闭 socket，且不会影响 core local listener。
func startV3PairingListener(ctx context.Context, runtime v3ClientAccessRuntime, logger *slog.Logger) (func(), error) {
	binding, err := remoteauth.LocalUnixChannelBinding(runtime.PairingSocket)
	if err != nil {
		return nil, err
	}
	listener, err := unixtransport.NewListener(runtime.PairingSocket)
	if err != nil {
		return nil, fmt.Errorf("listen for local PairingExchange: %w", err)
	}
	acceptor := remotev2daemon.PairingAcceptor{Identity: runtime.Identity, AccessStore: runtime.Store}
	serveCtx, cancel := context.WithCancel(ctx)
	acceptDone := make(chan struct{})
	shutdownDone := make(chan struct{})
	var handlers sync.WaitGroup
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			cancel()
			_ = listener.Close()
			<-acceptDone
			handlers.Wait()
			close(shutdownDone)
		})
	}
	go func() {
		defer close(acceptDone)
		for {
			connection, acceptErr := listener.Accept(serveCtx)
			if acceptErr != nil {
				if serveCtx.Err() == nil && logger != nil {
					logger.Warn("local PairingExchange listener stopped", "error", acceptErr)
				}
				return
			}
			handlers.Add(1)
			go func(connection transport.Transport) {
				defer handlers.Done()
				if serveErr := acceptor.ServeBoundTransport(serveCtx, connection, binding); serveErr != nil && serveCtx.Err() == nil && logger != nil {
					logger.Debug("local PairingExchange rejected", "error", serveErr)
				}
			}(connection)
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			shutdown()
		case <-shutdownDone:
		}
	}()
	return shutdown, nil
}

type v3ClientAccessService struct {
	identity remoteauth.Identity
	store    *remoteauth.AccessStore
}

// newEphemeralV3ClientAccessService 只为进程内 smoke harness 创建不落盘的 DeviceIdentity。
// 正式 daemon 必须继续使用 newV3ClientAccessRuntime 的持久 identity/store，不能调用该入口替代持久安全真值。
func newEphemeralV3ClientAccessService(deviceID string) (v3ClientAccessService, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return v3ClientAccessService{}, err
	}
	identity, err := remoteauth.NewIdentity(deviceID, privateKey)
	if err != nil {
		return v3ClientAccessService{}, err
	}
	return v3ClientAccessService{identity: identity}, nil
}

// Identity 返回当前 daemon 进程统一 DeviceIdentity 的公开投影；私钥永远不进入 protocol response。
func (service v3ClientAccessService) Identity(_ context.Context, challenge []byte) (corev2.ClientAccessIdentity, error) {
	if err := service.identity.Validate(); err != nil {
		return corev2.ClientAccessIdentity{}, err
	}
	proof, err := remoteauth.SignDeviceIdentityProof(service.identity, challenge)
	if err != nil {
		return corev2.ClientAccessIdentity{}, err
	}
	return corev2.ClientAccessIdentity{
		DeviceID: service.identity.DeviceID, DeviceFingerprint: service.identity.Fingerprint,
		DevicePublicKey: append([]byte(nil), service.identity.PublicKey...), Challenge: append([]byte(nil), challenge...), Proof: proof,
	}, nil
}

// CreateTicket 把已通过 transport scope 校验的 owner 请求交给唯一 AccessStore，原子签发 ticket 并登记内存 claim。
func (service v3ClientAccessService) CreateTicket(_ context.Context, request corev2.ClientAccessTicketRequest) (corev2.ClientAccessTicket, error) {
	if service.store == nil {
		return corev2.ClientAccessTicket{}, fmt.Errorf("client access store is unavailable")
	}
	routes := make([]*remoteauthpb.EndpointRouteConfigV1, 0, len(request.Routes))
	for _, value := range request.Routes {
		if value == nil {
			continue
		}
		route := proto.Clone(value).(*remoteauthpb.EndpointRouteConfigV1)
		if managed := route.GetManagedWebrtc(); managed != nil {
			// owning daemon 的 DeviceIdentity 是 managed Route 唯一目标；CLI 不能让调用者伪造其它 daemon ID。
			managed.TargetDeviceId = service.identity.DeviceID
		}
		routes = append(routes, route)
	}
	issued, err := service.store.IssuePairingClaim(remoteauth.PairingIssueOptions{
		Label: request.Label, Scope: remoteAuthScopeFromCore(request.Scope), TicketTTL: request.TicketTTL, GrantLifetime: request.GrantLifetime,
		Routes: routes,
	})
	if err != nil {
		return corev2.ClientAccessTicket{}, err
	}
	return corev2.ClientAccessTicket{
		ClaimOffer: issued.OfferPayload, ClaimCode: issued.ClaimCode,
		TicketID: issued.Claims.TicketID, ExpiresAt: issued.Claims.ExpiresAt,
	}, nil
}

// List 返回 AccessStore 的脱敏授权记录；该调用不刷新时间戳或写入本地状态。
func (service v3ClientAccessService) List(context.Context) ([]corev2.ClientAccessRecord, error) {
	if service.store == nil {
		return nil, fmt.Errorf("client access store is unavailable")
	}
	records := service.store.ListClientAccess()
	result := make([]corev2.ClientAccessRecord, 0, len(records))
	for _, record := range records {
		result = append(result, clientAccessRecordFromRemoteAuth(record))
	}
	return result, nil
}

func (service v3ClientAccessService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	return service.store != nil && service.store.GrantActive(grantID, expiresAt, now)
}

// Revoke 由 owning daemon 原子持久化撤销状态；删除客户端本地 credential 不能替代该操作。
func (service v3ClientAccessService) Revoke(_ context.Context, grantID string) (corev2.ClientAccessRecord, error) {
	if service.store == nil {
		return corev2.ClientAccessRecord{}, fmt.Errorf("client access store is unavailable")
	}
	record, err := service.store.RevokeGrant(grantID)
	if err != nil {
		return corev2.ClientAccessRecord{}, err
	}
	return clientAccessRecordFromRemoteAuth(record), nil
}

func remoteAuthScopeFromCore(scope corev2.ClientAccessScope) remoteauth.Scope {
	return remoteauth.Scope{AllowDaemon: scope.AllowDaemon, TerminalID: scope.TerminalID, MachineEventsOnly: scope.MachineEventsOnly, FileReadMetadata: scope.FileReadMetadata, FileReadContent: scope.FileReadContent, FileWriteContent: scope.FileWriteContent, FileMutate: scope.FileMutate, ManageClientAccess: scope.ManageClientAccess}
}

func clientAccessRecordFromRemoteAuth(record remoteauth.ClientAccessRecord) corev2.ClientAccessRecord {
	return corev2.ClientAccessRecord{GrantID: record.GrantID, RevocationID: record.RevocationID, SubjectKeyFingerprint: record.SubjectKeyFingerprint, ClientLabel: record.ClientLabel, Scope: corev2.ClientAccessScope{AllowDaemon: record.Scope.AllowDaemon, TerminalID: record.Scope.TerminalID, MachineEventsOnly: record.Scope.MachineEventsOnly, FileReadMetadata: record.Scope.FileReadMetadata, FileReadContent: record.Scope.FileReadContent, FileWriteContent: record.Scope.FileWriteContent, FileMutate: record.Scope.FileMutate, ManageClientAccess: record.Scope.ManageClientAccess}, IssuedAt: record.IssuedAt, ExpiresAt: record.ExpiresAt, RevokedAt: record.RevokedAt}
}
