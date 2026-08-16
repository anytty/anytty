package core

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

// ErrClientAccessServiceUnavailable 表示当前 daemon 未装配 DeviceIdentity/AccessStore 管理边界。
// local owner 与 ManageClientAccess session 都必须 fail closed，不能回退到旧 bearer pair 命令或直接写 credential 文件。
var ErrClientAccessServiceUnavailable = errors.New("client access service is not configured")

// ClientAccessScope 是 daemon capability scope 的 core-native 投影。
type ClientAccessScope struct {
	AllowDaemon, MachineEventsOnly, FileReadMetadata, FileReadContent, FileWriteContent, FileMutate, ManageClientAccess bool
	TerminalID                                                                                                          string
}

// ClientAccessIdentity 是 daemon DeviceIdentity 的公开 core-native 投影。
type ClientAccessIdentity struct {
	DeviceID, DeviceFingerprint string
	DevicePublicKey             []byte
	Challenge, Proof            []byte
}

// ClientAccessTicketRequest 是一次 pairing ticket 签发请求。
type ClientAccessTicketRequest struct {
	Label                    string
	Scope                    ClientAccessScope
	TicketTTL, GrantLifetime time.Duration
	Routes                   []*remoteauthpb.EndpointRouteConfigV1
}

// ClientAccessTicket 是 daemon 创建一次性配对 claim 后返回给 API Layer 的 core-native 结果。
type ClientAccessTicket struct {
	ClaimOffer []byte
	ClaimCode  string
	TicketID   string
	ExpiresAt  time.Time
}

// ClientAccessRecord 是 daemon 持久化 grant 的脱敏投影。
type ClientAccessRecord struct {
	GrantID, RevocationID, SubjectKeyFingerprint, ClientLabel string
	Scope                                                     ClientAccessScope
	IssuedAt, ExpiresAt, RevokedAt                            time.Time
}

// ClientAccessService 是 core protocol session 调用 daemon-owned identity/pair/access runtime 的 typed hook。
// core 只负责认证后 method scope；ticket、grant、key binding、receipt、撤销和持久化 truth 全部由实现方的 remoteauth.AccessStore 持有。
type ClientAccessService interface {
	// Identity 返回 daemon DeviceIdentity 的公开投影和当前 challenge 的签名证明；实现不得返回或记录私钥。
	Identity(ctx context.Context, challenge []byte) (ClientAccessIdentity, error)
	// CreateTicket 由 owning daemon 原子登记 PairingTicket，并创建仅在当前 daemon 内存可解析的一次性 claim。
	CreateTicket(ctx context.Context, request ClientAccessTicketRequest) (ClientAccessTicket, error)
	// List 返回不含 ticket、grant body 或 client public key bytes 的脱敏授权投影。
	List(ctx context.Context) ([]ClientAccessRecord, error)
	// GrantActive 在 core admission 线性化边界查询同一持久 store；未知、撤销、过期或期限不匹配都返回 false。
	GrantActive(ctx context.Context, grantID string, expiresAt, now time.Time) bool
	// Revoke 按 GrantID 持久化撤销；实现失败时不得只修改 core session 内存。
	Revoke(ctx context.Context, grantID string) (ClientAccessRecord, error)
}
