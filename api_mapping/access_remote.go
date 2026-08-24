package apimapping

import (
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxPairingTicketLifetimeSeconds = int64((7 * 24 * time.Hour) / time.Second)
	maxClientGrantLifetimeSeconds   = int64((365 * 24 * time.Hour) / time.Second)
	deviceIdentityChallengeBytes    = 32
)

// ValidateAccessRemoteCommand 校验 client access 与 remote daemon control 的 typed command。
func ValidateAccessRemoteCommand(command *apipb.CommandEnvelope) error {
	if err := ValidateRequestContext(RequestContextForCommand(command)); err != nil {
		return err
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_ClientAccessIdentity:
		if len(value.ClientAccessIdentity.GetChallenge()) != deviceIdentityChallengeBytes {
			return validation("client_access_identity.challenge", "fresh identity challenge must be 32 bytes")
		}
	case *apipb.CommandEnvelope_ClientAccessList,
		*apipb.CommandEnvelope_RemoteStatus, *apipb.CommandEnvelope_RemoteLocalStatus,
		*apipb.CommandEnvelope_RemoteLocalDisable, *apipb.CommandEnvelope_RemoteCloudStatus,
		*apipb.CommandEnvelope_RemoteCloudEnable, *apipb.CommandEnvelope_RemoteCloudDisable,
		*apipb.CommandEnvelope_RemoteCloudEdges,
		*apipb.CommandEnvelope_RemoteCloudReselectEdge:
		return nil
	case *apipb.CommandEnvelope_ClientAccessTicketCreate:
		request := value.ClientAccessTicketCreate.GetRequest()
		if request == nil || request.GetLabel() == "" || request.GetScope() == nil {
			return validation("client_access_ticket_create.request", "label and scope are required")
		}
		if request.GetTicketTtlSeconds() <= 0 || request.GetTicketTtlSeconds() > maxPairingTicketLifetimeSeconds || request.GetGrantLifetimeSeconds() < 0 || request.GetGrantLifetimeSeconds() > maxClientGrantLifetimeSeconds {
			return validation("client_access_ticket_create.request", "ticket lifetime must be between one second and seven days; grant lifetime must be zero or at most one year")
		}
	case *apipb.CommandEnvelope_ClientAccessRevoke:
		if value.ClientAccessRevoke.GetRequest().GetGrantId() == "" {
			return validation("client_access_revoke.request.grant_id", "grant id is required")
		}
	case *apipb.CommandEnvelope_RemotePairStart:
		if value.RemotePairStart.GetTtlSeconds() <= 0 || value.RemotePairStart.GetAuthTtlSeconds() <= 0 {
			return validation("remote_pair_start", "positive ttl values are required")
		}
	case *apipb.CommandEnvelope_RemoteLocalEnable:
		if value.RemoteLocalEnable.GetLocalWebAddress() == "" {
			return validation("remote_local_enable.local_web_address", "address is required")
		}
	case *apipb.CommandEnvelope_RemoteCloudPreferEdge:
		if value.RemoteCloudPreferEdge.GetExpectedPreferenceRevision() == 0 {
			return validation("remote_cloud_prefer_edge.expected_preference_revision", "preference revision is required")
		}
	default:
		return validation("command", "access or remote command is required")
	}
	return nil
}

// RemoteCloudEdgeSelectionToProto maps the daemon-local Edge ranking without exposing CA material.
func RemoteCloudEdgeSelectionToProto(selection corev2.RemoteCloudEdgeSelection) *apipb.RemoteCloudEdgesResult {
	projected := &cloudv1.DaemonEdgeSelection{DaemonId: selection.DaemonID, PreferredEdgeId: selection.PreferredEdgeID, PreferenceRevision: selection.PreferenceRevision, CurrentEdgeId: selection.CurrentEdgeID, SelectedEdgeId: selection.SelectedEdgeID, EvaluatedAt: timestamppb.New(selection.EvaluatedAt), Candidates: make([]*cloudv1.DaemonEdgeCandidate, 0, len(selection.Candidates))}
	for _, candidate := range selection.Candidates {
		value := &cloudv1.DaemonEdgeCandidate{Locator: &cloudv1.EdgeLocator{EdgeId: candidate.EdgeID, Name: candidate.Name, Region: candidate.Region, PublicEndpoint: candidate.PublicEndpoint}, Online: candidate.Online, Eligible: candidate.Eligible, AgentCount: candidate.AgentCount, Capacity: candidate.Capacity, Preferred: candidate.Preferred, Current: candidate.Current, Score: candidate.Score, Status: candidate.Status}
		if measurement := candidate.Measurement; measurement != nil {
			value.Measurement = &cloudv1.DaemonEdgeMeasurement{EdgeId: candidate.EdgeID, Reachable: measurement.Reachable, ConnectLatencyMs: measurement.ConnectLatencyMS, ConnectionFailureRate: measurement.ConnectionFailureRate, SampleCount: measurement.SampleCount, MeasuredAt: timestamppb.New(measurement.MeasuredAt)}
		}
		projected.Candidates = append(projected.Candidates, value)
	}
	return &apipb.RemoteCloudEdgesResult{Selection: projected}
}

// ClientAccessTicketRequestFromProto 转为 core-native ticket request。
func ClientAccessTicketRequestFromProto(command *apipb.ClientAccessTicketCreateCommand) corev2.ClientAccessTicketRequest {
	request := command.GetRequest()
	routes := make([]*remoteauthpb.EndpointRouteConfigV1, 0, len(request.GetRoutes()))
	for _, route := range request.GetRoutes() {
		if route != nil {
			routes = append(routes, proto.Clone(route).(*remoteauthpb.EndpointRouteConfigV1))
		}
	}
	return corev2.ClientAccessTicketRequest{Label: request.GetLabel(), AccessLabel: request.GetAccessLabel(), Scope: clientAccessScopeFromProto(request.GetScope()), TicketTTL: time.Duration(request.GetTicketTtlSeconds()) * time.Second, GrantLifetime: time.Duration(request.GetGrantLifetimeSeconds()) * time.Second, Routes: routes}
}

// ClientAccessIdentityToProto 转为公开 DeviceIdentity 投影，私钥不在输入模型中。
func ClientAccessIdentityToProto(identity corev2.ClientAccessIdentity) *remoteauthpb.ClientAccessIdentityResult {
	return &remoteauthpb.ClientAccessIdentityResult{DeviceId: identity.DeviceID, DeviceFingerprint: identity.DeviceFingerprint, DevicePublicKey: cloneBytes(identity.DevicePublicKey)}
}

// ClientAccessTicketToProto 转为一次性 pairing claim result；官方 consumer 只能展示 claim_offer/claim_code。
func ClientAccessTicketToProto(ticket corev2.ClientAccessTicket) *remoteauthpb.ClientAccessTicketCreateResult {
	return &remoteauthpb.ClientAccessTicketCreateResult{TicketId: ticket.TicketID, ExpiresAtUnixNano: unixNanoOrZero(ticket.ExpiresAt), ClaimOffer: cloneBytes(ticket.ClaimOffer), ClaimCode: ticket.ClaimCode}
}

// ClientAccessRecordToProto 转为不包含 grant body/public key 的脱敏记录。
func ClientAccessRecordToProto(record corev2.ClientAccessRecord) *remoteauthpb.ClientAccessRecord {
	return &remoteauthpb.ClientAccessRecord{GrantId: record.GrantID, RevocationId: record.RevocationID, SubjectKeyFingerprint: record.SubjectKeyFingerprint, ClientLabel: record.ClientLabel, Scope: clientAccessScopeToProto(record.Scope), IssuedAtUnixNano: unixNanoOrZero(record.IssuedAt), ExpiresAtUnixNano: unixNanoOrZero(record.ExpiresAt), RevokedAtUnixNano: unixNanoOrZero(record.RevokedAt), AccessLabel: record.AccessLabel}
}

// ClientAccessIdentityResultToProto 包装公开 identity projection。
func ClientAccessIdentityResultToProto(identity corev2.ClientAccessIdentity) *apipb.ClientAccessIdentityResult {
	return &apipb.ClientAccessIdentityResult{Identity: ClientAccessIdentityToProto(identity), Challenge: cloneBytes(identity.Challenge), Proof: cloneBytes(identity.Proof)}
}

// ClientAccessListToProto 映射脱敏 client access record 列表。
func ClientAccessListToProto(records []corev2.ClientAccessRecord) *apipb.ClientAccessListResult {
	result := &apipb.ClientAccessListResult{Access: &remoteauthpb.ClientAccessListResult{Records: make([]*remoteauthpb.ClientAccessRecord, 0, len(records))}}
	for _, record := range records {
		result.Access.Records = append(result.Access.Records, ClientAccessRecordToProto(record))
	}
	return result
}

// ClientAccessTicketResultToProto 包装一次性 pairing ticket projection。
func ClientAccessTicketResultToProto(ticket corev2.ClientAccessTicket) *apipb.ClientAccessTicketCreateResult {
	return &apipb.ClientAccessTicketCreateResult{Ticket: ClientAccessTicketToProto(ticket)}
}

// ClientAccessRevokeToProto 包装撤销后的脱敏 access record。
func ClientAccessRevokeToProto(record corev2.ClientAccessRecord) *apipb.ClientAccessRevokeResult {
	return &apipb.ClientAccessRevokeResult{Record: ClientAccessRecordToProto(record)}
}

// RemotePairStartRequestFromProto 转为 core-native remote pairing request。
func RemotePairStartRequestFromProto(command *apipb.RemotePairStartCommand) corev2.RemotePairStartRequest {
	return corev2.RemotePairStartRequest{LocalPairURL: command.GetLocalPairUrl(), TTLSeconds: int(command.GetTtlSeconds()), AuthorizationTTLSeconds: int(command.GetAuthTtlSeconds())}
}

// RemoteLocalEnableRequestFromProto 转为 core-native local remote runtime request。
func RemoteLocalEnableRequestFromProto(command *apipb.RemoteLocalEnableCommand) corev2.RemoteLocalEnableRequest {
	return corev2.RemoteLocalEnableRequest{LocalWebAddress: command.GetLocalWebAddress(), ICETCPAddress: command.GetIceTcpAddress(), HubURLs: append([]string(nil), command.GetHubUrls()...), ControlURL: command.GetControlUrl(), AccessToken: command.GetAccessToken(), Region: command.GetRegion(), LocalWebPassword: append([]byte(nil), command.GetLocalWebPassword()...)}
}

// RemoteStatusToProto 转为公共 remote runtime 状态。
func RemoteStatusToProto(status corev2.RemoteStatus) *apipb.RemoteStatusResult {
	return &apipb.RemoteStatusResult{State: status.State, Detail: status.Detail, DeviceId: status.DeviceID, DeviceName: status.DeviceName, ControlUrl: status.ControlURL, HubUrl: status.HubURL, HubUrls: append([]string(nil), status.HubURLs...), DataDirectory: status.DataDirectory, Mode: status.Mode, AllowLan: status.AllowLAN, TerminalCount: int32(status.TerminalCount), UpdatedAtUnixNano: unixNanoOrZero(status.UpdatedAt)}
}

// RemotePairStartToProto 转为公共 remote pairing result。
func RemotePairStartToProto(result corev2.RemotePairStartResult) *apipb.RemotePairStartResult {
	return &apipb.RemotePairStartResult{Type: result.Type, MachineId: result.MachineID, MachineName: result.MachineName, LocalPairUrl: result.LocalPairURL, PairSessionId: result.PairSessionID, PairSecret: result.PairSecret, AnswerProofSecret: result.AnswerProofSecret, ExpiresAtUnixNano: unixNanoOrZero(result.ExpiresAt)}
}

// RemoteLocalStatusToProto 转为公共 local remote runtime 状态。
func RemoteLocalStatusToProto(status corev2.RemoteLocalStatus) *apipb.RemoteLocalStatusResult {
	return &apipb.RemoteLocalStatusResult{Enabled: status.Enabled, HttpUrl: status.HTTPURL, LocalWebAddress: status.LocalWebAddress, LocalPairUrl: status.LocalPairURL, IceTcpEnabled: status.ICETCPEnabled, IceTcpAddress: status.ICETCPAddress, IceTcpPort: int32(status.ICETCPPort), UpdatedAtUnixNano: unixNanoOrZero(status.UpdatedAt), PasswordProtected: status.PasswordProtected}
}

// RemoteCloudStatusToProto 转为公共 Cloud runtime 状态。
func RemoteCloudStatusToProto(status corev2.RemoteCloudStatus) *apipb.RemoteCloudStatusResult {
	return &apipb.RemoteCloudStatusResult{
		Enrolled: status.Enrolled, Enabled: status.Enabled, Running: status.Running, State: status.State, Detail: status.Detail,
		DaemonId: status.DaemonID, AccountId: status.AccountID,
		EdgeId: status.EdgeID, EdgeName: status.EdgeName, EdgeRegion: status.EdgeRegion,
		PublicEndpoint: status.PublicEndpoint, ServerName: status.ServerName,
		LifecycleState: status.LifecycleState, LifecycleRevision: status.LifecycleRevision,
		Ready: status.Ready, ActiveSessions: int32(status.ActiveSessions),
		EnrolledAtUnixNano: unixNanoOrZero(status.EnrolledAt), UpdatedAtUnixNano: unixNanoOrZero(status.UpdatedAt),
		RecordPath: status.RecordPath, DisabledPath: status.DisabledPath,
	}
}

func clientAccessScopeFromProto(scope *remoteauthpb.ClientAccessScope) corev2.ClientAccessScope {
	if scope == nil {
		return corev2.ClientAccessScope{}
	}
	return corev2.ClientAccessScope{AllowDaemon: scope.GetAllowDaemon(), TerminalID: scope.GetTerminalId(), MachineEventsOnly: scope.GetMachineEventsOnly(), FileReadMetadata: scope.GetFileReadMetadata(), FileReadContent: scope.GetFileReadContent(), FileWriteContent: scope.GetFileWriteContent(), FileMutate: scope.GetFileMutate(), ManageClientAccess: scope.GetManageClientAccess()}
}

func clientAccessScopeToProto(scope corev2.ClientAccessScope) *remoteauthpb.ClientAccessScope {
	return &remoteauthpb.ClientAccessScope{AllowDaemon: scope.AllowDaemon, TerminalId: scope.TerminalID, MachineEventsOnly: scope.MachineEventsOnly, FileReadMetadata: scope.FileReadMetadata, FileReadContent: scope.FileReadContent, FileWriteContent: scope.FileWriteContent, FileMutate: scope.FileMutate, ManageClientAccess: scope.ManageClientAccess}
}
