// Package accessruntime 是 access 进程的本地授权 runtime：
// DeviceIdentity、AccessStore、local pairing 与 client access service。
// daemon 只通过本地 control 通道消费它，不再自行持有授权真值。
package accessruntime

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
	"strings"

	accesscontract "github.com/anytty/anytty/access/contract"
	"github.com/anytty/anytty/access/sessions"
	clouddaemon "github.com/anytty/anytty/daemon/cloud"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/protobuf/proto"
)

// Service 是 access owner 的 ClientAccessService 实现：
// 所有 ticket/grant/key binding/撤销真值都在同一 AccessStore 内。
type Service struct {
	DeviceIdentity remoteauth.Identity
	Store          *remoteauth.AccessStore
	DefaultLabel   func() string
	// Sessions 非空时，Revoke 会同时关闭该 grant 的活动会话（实时踢线）。
	Sessions *sessions.Registry
}

// Identity 返回 DeviceIdentity 的公开投影和当前 challenge 的签名证明；私钥永远不进入 response。
func (service Service) Identity(_ context.Context, challenge []byte) (accesscontract.ClientAccessIdentity, error) {
	if err := service.DeviceIdentity.Validate(); err != nil {
		return accesscontract.ClientAccessIdentity{}, err
	}
	proof, err := remoteauth.SignDeviceIdentityProof(service.DeviceIdentity, challenge)
	if err != nil {
		return accesscontract.ClientAccessIdentity{}, err
	}
	return accesscontract.ClientAccessIdentity{
		DeviceID: service.DeviceIdentity.DeviceID, DeviceFingerprint: service.DeviceIdentity.Fingerprint,
		DevicePublicKey: append([]byte(nil), service.DeviceIdentity.PublicKey...), Challenge: append([]byte(nil), challenge...), Proof: proof,
	}, nil
}

// CreateTicket 把 owner 请求交给唯一 AccessStore，原子签发 ticket 并登记内存 claim。
func (service Service) CreateTicket(_ context.Context, request accesscontract.ClientAccessTicketRequest) (accesscontract.ClientAccessTicket, error) {
	if service.Store == nil {
		return accesscontract.ClientAccessTicket{}, fmt.Errorf("client access store is unavailable")
	}
	routes := make([]*remoteauthpb.EndpointRouteConfigV1, 0, len(request.Routes))
	for _, value := range request.Routes {
		if value == nil {
			continue
		}
		route := proto.Clone(value).(*remoteauthpb.EndpointRouteConfigV1)
		if managed := route.GetManagedWebrtc(); managed != nil {
			// owning access runtime 的 DeviceIdentity 是 managed Route 唯一目标；调用者不能让客户端伪造其它 daemon ID。
			managed.TargetDeviceId = service.DeviceIdentity.DeviceID
		}
		routes = append(routes, route)
	}
	label := strings.TrimSpace(request.Label)
	if label == "" && service.DefaultLabel != nil {
		label = strings.TrimSpace(service.DefaultLabel())
	}
	if label == "" {
		label = service.DeviceIdentity.DeviceID
	}
	issued, err := service.Store.IssuePairingClaim(remoteauth.PairingIssueOptions{
		Label: label, AccessLabel: request.AccessLabel, Scope: remoteAuthScopeFromCore(request.Scope), TicketTTL: request.TicketTTL, GrantLifetime: request.GrantLifetime,
		Routes: routes, Platform: runtime.GOOS,
	})
	if err != nil {
		return accesscontract.ClientAccessTicket{}, err
	}
	return accesscontract.ClientAccessTicket{
		ClaimOffer: issued.OfferPayload, ClaimCode: issued.ClaimCode,
		TicketID: issued.Claims.TicketID, ExpiresAt: issued.Claims.ExpiresAt,
	}, nil
}

// List 返回 AccessStore 的脱敏授权记录；该调用不刷新时间戳或写入本地状态。
func (service Service) List(context.Context) ([]accesscontract.ClientAccessRecord, error) {
	if service.Store == nil {
		return nil, fmt.Errorf("client access store is unavailable")
	}
	records := service.Store.ListClientAccess()
	result := make([]accesscontract.ClientAccessRecord, 0, len(records))
	for _, record := range records {
		result = append(result, clientAccessRecordFromRemoteAuth(record))
	}
	return result, nil
}

// Revoke 由 owning access runtime 原子持久化撤销状态。
func (service Service) Revoke(_ context.Context, grantID string) (accesscontract.ClientAccessRecord, error) {
	if service.Store == nil {
		return accesscontract.ClientAccessRecord{}, fmt.Errorf("client access store is unavailable")
	}
	record, err := service.Store.RevokeGrant(grantID)
	if err != nil {
		return accesscontract.ClientAccessRecord{}, err
	}
	if service.Sessions != nil {
		service.Sessions.CloseGrant(grantID)
	}
	return clientAccessRecordFromRemoteAuth(record), nil
}

// NewEphemeralService 只为进程内 harness 创建不落盘的 DeviceIdentity。
func NewEphemeralService(deviceID string) (Service, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Service{}, err
	}
	identity, err := remoteauth.NewIdentity(deviceID, privateKey)
	if err != nil {
		return Service{}, err
	}
	return Service{DeviceIdentity: identity}, nil
}

// DefaultPairingLabelFromEnrollment 用 Cloud enrollment display name 作为默认 pairing 标签，
// 缺失或损坏时回退 hostname；它只读 Cloud 状态文件，不加载身份或 store。
func DefaultPairingLabelFromEnrollment(recordPath string) string {
	if record, err := clouddaemon.LoadRecord(recordPath); err == nil {
		return record.DisplayName
	}
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(hostname)
}

func remoteAuthScopeFromCore(scope accesscontract.ClientAccessScope) remoteauth.Scope {
	return remoteauth.Scope{AllowDaemon: scope.AllowDaemon, TerminalID: scope.TerminalID, MachineEventsOnly: scope.MachineEventsOnly, FileReadMetadata: scope.FileReadMetadata, FileReadContent: scope.FileReadContent, FileWriteContent: scope.FileWriteContent, FileMutate: scope.FileMutate, ManageClientAccess: scope.ManageClientAccess}
}

func clientAccessRecordFromRemoteAuth(record remoteauth.ClientAccessRecord) accesscontract.ClientAccessRecord {
	return accesscontract.ClientAccessRecord{GrantID: record.GrantID, RevocationID: record.RevocationID, SubjectKeyFingerprint: record.SubjectKeyFingerprint, AccessLabel: record.AccessLabel, ClientLabel: record.ClientLabel, Scope: accesscontract.ClientAccessScope{AllowDaemon: record.Scope.AllowDaemon, TerminalID: record.Scope.TerminalID, MachineEventsOnly: record.Scope.MachineEventsOnly, FileReadMetadata: record.Scope.FileReadMetadata, FileReadContent: record.Scope.FileReadContent, FileWriteContent: record.Scope.FileWriteContent, FileMutate: record.Scope.FileMutate, ManageClientAccess: record.Scope.ManageClientAccess}, IssuedAt: record.IssuedAt, ExpiresAt: record.ExpiresAt, RevokedAt: record.RevokedAt}
}
