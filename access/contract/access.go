// Package contract 定义 access owner 的 typed 服务边界与 core-native 投影。
// 这些类型原先寄居在 pool/core；Phase 4 后 identity/pair/remote 都由 access
// runtime 持有，pool 不再挂载，因此归 access 所有。
package contract

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/proto/access/remoteauthpb"
)

// ErrClientAccessServiceUnavailable 表示当前 access 未装配 DeviceIdentity/AccessStore 管理边界。
// local owner 与 ManageClientAccess session 都必须 fail closed，不能回退到旧 bearer pair 命令或直接写 credential 文件。
var ErrClientAccessServiceUnavailable = errors.New("client access service is not configured")

// ClientAccessScope 是 access capability scope 的 core-native 投影。
type ClientAccessScope struct {
	AllowDaemon, MachineEventsOnly, FileReadMetadata, FileReadContent, FileWriteContent, FileMutate, ManageClientAccess bool
	TerminalID                                                                                                          string
}

// ClientAccessIdentity 是 access DeviceIdentity 的公开 core-native 投影。
type ClientAccessIdentity struct {
	DeviceID, DeviceFingerprint string
	DevicePublicKey             []byte
	Challenge, Proof            []byte
}

// ClientAccessTicketRequest 是一次 pairing ticket 签发请求。
type ClientAccessTicketRequest struct {
	Label, AccessLabel       string
	Scope                    ClientAccessScope
	TicketTTL, GrantLifetime time.Duration
	Routes                   []*remoteauthpb.EndpointRouteConfigV1
}

// ClientAccessTicket 是 access 创建一次性配对 claim 后返回给 API Layer 的 core-native 结果。
type ClientAccessTicket struct {
	ClaimOffer []byte
	ClaimCode  string
	TicketID   string
	ExpiresAt  time.Time
}

// ClientAccessRecord 是 access 持久化 grant 的脱敏投影。
type ClientAccessRecord struct {
	GrantID, RevocationID, SubjectKeyFingerprint, AccessLabel, ClientLabel string
	Scope                                                                  ClientAccessScope
	IssuedAt, ExpiresAt, RevokedAt                                         time.Time
}

// ClientAccessService 是 access session 调用 access-owned identity/pair/access runtime 的 typed hook。
// 调用方只负责 capability 校验；ticket、grant、key binding、receipt、撤销和持久化 truth
// 全部由实现方的 remoteauth.AccessStore 持有。
type ClientAccessService interface {
	// Identity 返回 access DeviceIdentity 的公开投影和当前 challenge 的签名证明；实现不得返回或记录私钥。
	Identity(ctx context.Context, challenge []byte) (ClientAccessIdentity, error)
	// CreateTicket 由 owning access 原子登记 PairingTicket，并创建仅在当前进程内存可解析的一次性 claim。
	CreateTicket(ctx context.Context, request ClientAccessTicketRequest) (ClientAccessTicket, error)
	// List 返回不含 ticket、grant body 或 client public key bytes 的脱敏授权投影。
	List(ctx context.Context) ([]ClientAccessRecord, error)
	// Revoke 按 GrantID 持久化撤销；实现失败时不得只修改 session 内存。
	Revoke(ctx context.Context, grantID string) (ClientAccessRecord, error)
}
