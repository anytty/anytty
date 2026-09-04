
èÆ
bindingpb/client_binding.protoanytty.client.binding.v1apipb/application.protoapipb/common.protoremoteauthpb/remote_auth.proto"—

ConnectionSnapshot
route_id (	RrouteIdL

route_kind (2-.anytty.client.binding.v1.ConnectionRouteKindR	routeKindU
observed_path (20.anytty.client.binding.v1.ConnectionObservedPathRobservedPath)
selection_reason (	RselectionReason/
sampled_at_unix_nano (RsampledAtUnixNano(
round_trip_nanos (RroundTripNanosc
local_candidate_type (21.anytty.client.binding.v1.ConnectionCandidateTypeRlocalCandidateTypee
remote_candidate_type (21.anytty.client.binding.v1.ConnectionCandidateTypeRremoteCandidateTypeT
local_protocol	 (2-.anytty.client.binding.v1.ConnectionTransportRlocalProtocolV
remote_protocol
 (2-.anytty.client.binding.v1.ConnectionTransportRremoteProtocolV
relay_transport (2-.anytty.client.binding.v1.ConnectionTransportRrelayTransport#
network_class (	RnetworkClass

bytes_sent (R	bytesSent%
bytes_received (RbytesReceived!
packets_sent (RpacketsSent
loss_events (R
lossEvents
	connected (R	connected
local_ip (	RlocalIp
	remote_ip (	RremoteIp

local_port (R	localPort
remote_port (R
remotePort*
candidate_pair_id (	RcandidatePairId(
local_related_ip (	RlocalRelatedIp,
local_related_port (RlocalRelatedPort*
remote_related_ip (	RremoteRelatedIp.
remote_related_port (RremoteRelatedPort"£
ConnectionPolicyY
route_preference (2..anytty.remote.auth.v1.EndpointRoutePreferenceRroutePreferenceW
cloud_relay_mode (2-.anytty.remote.auth.v1.ManagedWebRTCRelayModeRcloudRelayMode[
relay_transport (22.anytty.remote.auth.v1.ManagedWebRTCRelayTransportRrelayTransport"Â
!ConnectionPolicyRouteAvailabilityL

route_kind (2-.anytty.client.binding.v1.ConnectionRouteKindR	routeKind
	available (R	availableT
reason (2<.anytty.client.binding.v1.ConnectionPolicyAvailabilityReasonRreason"∞
ConnectionPolicyStateB
policy (2*.anytty.client.binding.v1.ConnectionPolicyRpolicyS
routes (2;.anytty.client.binding.v1.ConnectionPolicyRouteAvailabilityRroutes"\
ConnectionPolicyGetRequest

request_id (	R	requestId
endpoint_id (	R
endpointId"€
ConnectionPolicyGetResult

request_id (	R	requestId)
operation_handle (RoperationHandleE
state (2/.anytty.client.binding.v1.ConnectionPolicyStateRstate-
error (2.anytty.api.v1.ApiErrorRerror"¢
ConnectionPolicyApplyRequest

request_id (	R	requestId
endpoint_id (	R
endpointIdB
policy (2*.anytty.client.binding.v1.ConnectionPolicyRpolicy"›
ConnectionPolicyApplyResult

request_id (	R	requestId)
operation_handle (RoperationHandleE
state (2/.anytty.client.binding.v1.ConnectionPolicyStateRstate-
error (2.anytty.api.v1.ApiErrorRerror"d
ConnectionSnapshotGetRequest

request_id (	R	requestId%
session_handle (RsessionHandle"ã
ConnectionSnapshotGetResult

request_id (	R	requestId)
operation_handle (RoperationHandle%
session_handle (RsessionHandleL

connection (2,.anytty.client.binding.v1.ConnectionSnapshotR
connection-
error (2.anytty.api.v1.ApiErrorRerror"`
SessionInvalidateRequest

request_id (	R	requestId%
session_handle (RsessionHandle"π
SessionInvalidateResult

request_id (	R	requestId)
operation_handle (RoperationHandle%
session_handle (RsessionHandle-
error (2.anytty.api.v1.ApiErrorRerror"[
EndpointDisconnectRequest

request_id (	R	requestId
endpoint_id (	R
endpointId"¥
EndpointDisconnectResult

request_id (	R	requestId)
operation_handle (RoperationHandle
endpoint_id (	R
endpointId-
error (2.anytty.api.v1.ApiErrorRerror"a
EndpointCloudPresenceGetRequest

request_id (	R	requestId
endpoint_id (	R
endpointId"—
EndpointCloudPresenceGetResult

request_id (	R	requestId)
operation_handle (RoperationHandle
endpoint_id (	R
endpointId
online (Ronline-
error (2.anytty.api.v1.ApiErrorRerror
	device_id (	RdeviceId-
device_fingerprint (	RdeviceFingerprint
	daemon_id (	RdaemonId
edge_id	 (	RedgeId
	edge_name
 (	RedgeName
edge_region (	R
edgeRegion0
edge_public_endpoint (	RedgePublicEndpoint(
edge_server_name (	RedgeServerName%
locator_source (	RlocatorSource:
refreshed_from_controller (RrefreshedFromController"¬
OpenSessionRequest

request_id (	R	requestId
endpoint_id (	R
endpointId%
route_override (	RrouteOverride?
intent (2'.anytty.client.binding.v1.ConnectIntentRintentJ"í
ImportPairingRequest

request_id (	R	requestId)
portable_payload (	RportablePayload0
expected_endpoint_id (	RexpectedEndpointId"’
ImportPairingResult

request_id (	R	requestId)
operation_handle (RoperationHandleC
endpoint (2'.anytty.remote.auth.v1.EndpointConfigV1Rendpoint
	ticket_id (	RticketId4
client_key_fingerprint (	RclientKeyFingerprint/
expires_at_unix_nano (RexpiresAtUnixNano5
authorization_required (RauthorizationRequired-
error (2.anytty.api.v1.ApiErrorRerrorE
registry	 (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry"_
DeleteCredentialRequest

request_id (	R	requestId%
credential_ref (	RcredentialRef"ë
DeleteCredentialResult

request_id (	R	requestId)
operation_handle (RoperationHandle-
error (2.anytty.api.v1.ApiErrorRerror";
EndpointRegistryGetRequest

request_id (	R	requestId"€
EndpointRegistryGetResult

request_id (	R	requestId)
operation_handle (RoperationHandleE
registry (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry-
error (2.anytty.api.v1.ApiErrorRerror"û
EndpointUpsertRequest

request_id (	R	requestIdC
endpoint (2'.anytty.remote.auth.v1.EndpointConfigV1Rendpoint!
make_default (RmakeDefault"õ
EndpointUpsertResult

request_id (	R	requestId)
operation_handle (RoperationHandleC
endpoint (2'.anytty.remote.auth.v1.EndpointConfigV1RendpointE
registry (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry-
error (2.anytty.api.v1.ApiErrorRerror"W
EndpointDeleteRequest

request_id (	R	requestId
endpoint_id (	R
endpointId"˜
EndpointDeleteResult

request_id (	R	requestId)
operation_handle (RoperationHandle
endpoint_id (	R
endpointIdE
registry (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry-
error (2.anytty.api.v1.ApiErrorRerror"c
EndpointShareReceiveRequest

request_id (	R	requestId%
portable_offer (	RportableOffer"j
EndpointShareRouteDiff
route_id (	RrouteId

route_kind (	R	routeKind
action (	Raction"ó
EndpointSharePreview!
import_token (	RimportToken
endpoint_id (	R
endpointId
label (	RlabelI
identity (2-.anytty.remote.auth.v1.EndpointDaemonIdentityRidentityQ
route_diffs (20.anytty.client.binding.v1.EndpointShareRouteDiffR
routeDiffs0
connect_mode_changed (RconnectModeChanged8
selection_policy_changed (RselectionPolicyChangedj
credential_descriptors (23.anytty.remote.auth.v1.EndpointCredentialDescriptorRcredentialDescriptors/
expires_at_unix_nano	 (RexpiresAtUnixNano"ﬂ
EndpointShareReceiveResult

request_id (	R	requestId)
operation_handle (RoperationHandleH
preview (2..anytty.client.binding.v1.EndpointSharePreviewRpreview-
error (2.anytty.api.v1.ApiErrorRerror"^
EndpointShareCommitRequest

request_id (	R	requestId!
import_token (	RimportToken"◊
EndpointShareCommitResult

request_id (	R	requestId)
operation_handle (RoperationHandleC
endpoint (2'.anytty.remote.auth.v1.EndpointConfigV1RendpointE
registry (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry5
authorization_required (RauthorizationRequired-
error (2.anytty.api.v1.ApiErrorRerror"z
SSHCredentialProvisionRequest

request_id (	R	requestId
endpoint_id (	R
endpointId
route_id (	RrouteId"ö
SSHCredentialProvisionResult

request_id (	R	requestId)
operation_handle (RoperationHandleC
endpoint (2'.anytty.remote.auth.v1.EndpointConfigV1RendpointE
registry (2).anytty.remote.auth.v1.EndpointRegistryV1Rregistry%
credential_ref (	RcredentialRef%
authorized_key (	RauthorizedKey'
key_fingerprint (	RkeyFingerprint-
error (2.anytty.api.v1.ApiErrorRerror"‚
EngineCommandW
import_pairing (2..anytty.client.binding.v1.ImportPairingRequestH RimportPairing`
delete_credential (21.anytty.client.binding.v1.DeleteCredentialRequestH RdeleteCredentialj
endpoint_registry_get (24.anytty.client.binding.v1.EndpointRegistryGetRequestH RendpointRegistryGetZ
endpoint_upsert (2/.anytty.client.binding.v1.EndpointUpsertRequestH RendpointUpsertZ
endpoint_delete (2/.anytty.client.binding.v1.EndpointDeleteRequestH RendpointDeletem
endpoint_share_receive (25.anytty.client.binding.v1.EndpointShareReceiveRequestH RendpointShareReceivej
endpoint_share_commit (24.anytty.client.binding.v1.EndpointShareCommitRequestH RendpointShareCommits
ssh_credential_provision (27.anytty.client.binding.v1.SSHCredentialProvisionRequestH RsshCredentialProvisionj
connection_policy_get	 (24.anytty.client.binding.v1.ConnectionPolicyGetRequestH RconnectionPolicyGetp
connection_policy_apply
 (26.anytty.client.binding.v1.ConnectionPolicyApplyRequestH RconnectionPolicyApplyp
connection_snapshot_get (26.anytty.client.binding.v1.ConnectionSnapshotGetRequestH RconnectionSnapshotGetc
session_invalidate (22.anytty.client.binding.v1.SessionInvalidateRequestH RsessionInvalidatef
endpoint_disconnect (23.anytty.client.binding.v1.EndpointDisconnectRequestH RendpointDisconnectz
endpoint_cloud_presence_get (29.anytty.client.binding.v1.EndpointCloudPresenceGetRequestH RendpointCloudPresenceGetB	
command"¿
OpenSessionResult

request_id (	R	requestId)
operation_handle (RoperationHandle%
session_handle (RsessionHandle=
session (2#.anytty.api.v1.EndpointSessionStampRsession-
error (2.anytty.api.v1.ApiErrorRerrorL

connection (2,.anytty.client.binding.v1.ConnectionSnapshotR
connection"«
ExecuteResult)
operation_handle (RoperationHandle%
session_handle (RsessionHandle5
result (2.anytty.api.v1.ResultEnvelopeRresult-
error (2.anytty.api.v1.ApiErrorRerror"m
ApplicationEvent%
session_handle (RsessionHandle2
event (2.anytty.api.v1.EventEnvelopeRevent"ä
OpenResourceStreamRequest9
resource (2.anytty.api.v1.ResourceHandleRresource2
initial_upload_offset (RinitialUploadOffset"õ
ResourceStreamFrame#
stream_handle (RstreamHandleE
type (21.anytty.client.binding.v1.ResourceStreamFrameTypeRtype
payload (Rpayload"o
ResourceStreamClosedEvent#
stream_handle (RstreamHandle-
error (2.anytty.api.v1.ApiErrorRerror"©
SessionClosedEvent%
session_handle (RsessionHandle=
session (2#.anytty.api.v1.EndpointSessionStampRsession-
error (2.anytty.api.v1.ApiErrorRerror"‘
EndpointConnectionEvent

request_id (	R	requestId)
operation_handle (RoperationHandle
endpoint_id (	R
endpointId=
session (2#.anytty.api.v1.EndpointSessionStampRsessionG
phase (21.anytty.client.binding.v1.EndpointConnectionPhaseRphaseU
observed_path (20.anytty.client.binding.v1.ConnectionObservedPathRobservedPath4
route_selection_reason (	RrouteSelectionReason-
error (2.anytty.api.v1.ApiErrorRerror_
attempted_route_kind	 (2-.anytty.client.binding.v1.ConnectionRouteKindRattemptedRouteKind)
connection_stage
 (	RconnectionStage"Ö
EventEnvelope
abi_version (R
abiVersion
sequence (RsequenceP
open_session
 (2+.anytty.client.binding.v1.OpenSessionResultH RopenSessionC
execute (2'.anytty.client.binding.v1.ExecuteResultH RexecuteN
application (2*.anytty.client.binding.v1.ApplicationEventH RapplicationU
session_closed (2,.anytty.client.binding.v1.SessionClosedEventH RsessionClosedV
import_pairing (2-.anytty.client.binding.v1.ImportPairingResultH RimportPairing_
delete_credential (20.anytty.client.binding.v1.DeleteCredentialResultH RdeleteCredentialc
resource_stream_frame (2-.anytty.client.binding.v1.ResourceStreamFrameH RresourceStreamFramek
resource_stream_closed (23.anytty.client.binding.v1.ResourceStreamClosedEventH RresourceStreamClosedi
endpoint_registry_get (23.anytty.client.binding.v1.EndpointRegistryGetResultH RendpointRegistryGetY
endpoint_upsert (2..anytty.client.binding.v1.EndpointUpsertResultH RendpointUpsertY
endpoint_delete (2..anytty.client.binding.v1.EndpointDeleteResultH RendpointDeletel
endpoint_share_receive (24.anytty.client.binding.v1.EndpointShareReceiveResultH RendpointShareReceivei
endpoint_share_commit (23.anytty.client.binding.v1.EndpointShareCommitResultH RendpointShareCommitr
ssh_credential_provision (26.anytty.client.binding.v1.SSHCredentialProvisionResultH RsshCredentialProvisioni
connection_policy_get (23.anytty.client.binding.v1.ConnectionPolicyGetResultH RconnectionPolicyGeto
connection_policy_apply (25.anytty.client.binding.v1.ConnectionPolicyApplyResultH RconnectionPolicyApplyo
connection_snapshot_get (25.anytty.client.binding.v1.ConnectionSnapshotGetResultH RconnectionSnapshotGetb
session_invalidate (21.anytty.client.binding.v1.SessionInvalidateResultH RsessionInvalidated
endpoint_connection (21.anytty.client.binding.v1.EndpointConnectionEventH RendpointConnectione
endpoint_disconnect (22.anytty.client.binding.v1.EndpointDisconnectResultH RendpointDisconnecty
endpoint_cloud_presence_get (28.anytty.client.binding.v1.EndpointCloudPresenceGetResultH RendpointCloudPresenceGetB
event"b
CredentialResolveRequest
endpoint_id (	R
endpointId%
credential_ref (	RcredentialRef"b
CredentialPrepareRequest
endpoint_id (	R
endpointId%
credential_ref (	RcredentialRef"@
CredentialDeleteRequest%
credential_ref (	RcredentialRef"‰
CredentialBindRequest
endpoint_id (	R
endpointId%
credential_ref (	RcredentialRef)
capability_grant (	RcapabilityGrant*
cloud_route_grant (RcloudRouteGrant,
cloud_edge_locator (RcloudEdgeLocator"Ã
CredentialRecord
endpoint_id (	R
endpointId%
credential_ref (	RcredentialRef

public_key (R	publicKey'
key_fingerprint (	RkeyFingerprint)
capability_grant (	RcapabilityGrant#
newly_created (RnewlyCreated*
cloud_route_grant (RcloudRouteGrant,
cloud_edge_locator (RcloudEdgeLocator"X
CredentialSignRequest%
credential_ref (	RcredentialRef
payload (Rpayload"6
CredentialSignResponse
	signature (R	signature"L
CloudProfileResolveRequest.
account_profile_ref (	RaccountProfileRef"’
CloudProfileRecord.
account_profile_ref (	RaccountProfileRef-
controller_address (	RcontrollerAddress4
controller_server_name (	RcontrollerServerName*
controller_ca_pem (RcontrollerCaPem"o
SSHCredentialLookupRequest%
credential_ref (	RcredentialRef*
create_if_missing (RcreateIfMissing"C
SSHCredentialDeleteRequest%
credential_ref (	RcredentialRef"â
SSHCredentialRecord%
credential_ref (	RcredentialRef&
public_key_pkix (RpublicKeyPkix#
newly_created (RnewlyCreated"m
SSHCredentialSignRequest%
credential_ref (	RcredentialRef
digest (Rdigest
hash (	Rhash"9
SSHCredentialSignResponse
	signature (R	signature"
EndpointRegistryLoadRequest"{
EndpointRegistryStoreRequest%
registry_proto (RregistryProto4
delete_credential_refs (	RdeleteCredentialRefs"?
EndpointRegistryLoaded%
registry_proto (RregistryProto"i
LocalDiscoveryLookupRequest
	device_id (	RdeviceId-
device_fingerprint (	RdeviceFingerprint" 
LocalDiscoveryCandidate
address (	Raddress
port (Rport)
protocol_version (RprotocolVersion/
expires_at_unix_nano (RexpiresAtUnixNano%
network_handle (RnetworkHandle"o
LocalDiscoveryLookupResultQ

candidates (21.anytty.client.binding.v1.LocalDiscoveryCandidateR
candidates"
PlatformEventJ
"•

PlatformRequest

request_id (R	requestIdc
credential_resolve
 (22.anytty.client.binding.v1.CredentialResolveRequestH RcredentialResolvec
credential_prepare (22.anytty.client.binding.v1.CredentialPrepareRequestH RcredentialPrepare`
credential_delete (21.anytty.client.binding.v1.CredentialDeleteRequestH RcredentialDeleteZ
credential_sign (2/.anytty.client.binding.v1.CredentialSignRequestH RcredentialSignZ
credential_bind (2/.anytty.client.binding.v1.CredentialBindRequestH RcredentialBindm
endpoint_registry_load (25.anytty.client.binding.v1.EndpointRegistryLoadRequestH RendpointRegistryLoadp
endpoint_registry_store (26.anytty.client.binding.v1.EndpointRegistryStoreRequestH RendpointRegistryStorej
ssh_credential_lookup (24.anytty.client.binding.v1.SSHCredentialLookupRequestH RsshCredentialLookupd
ssh_credential_sign (22.anytty.client.binding.v1.SSHCredentialSignRequestH RsshCredentialSignj
ssh_credential_delete (24.anytty.client.binding.v1.SSHCredentialDeleteRequestH RsshCredentialDeletej
cloud_profile_resolve (24.anytty.client.binding.v1.CloudProfileResolveRequestH RcloudProfileResolvem
local_discovery_lookup (25.anytty.client.binding.v1.LocalDiscoveryLookupRequestH RlocalDiscoveryLookupB	
requestJJ'"˘
PlatformResponse

request_id (R	requestId-
error (2.anytty.api.v1.ApiErrorRerrorL

credential
 (2*.anytty.client.binding.v1.CredentialRecordH R
credential[
credential_sign (20.anytty.client.binding.v1.CredentialSignResponseH RcredentialSign_
endpoint_registry (20.anytty.client.binding.v1.EndpointRegistryLoadedH RendpointRegistryV
ssh_credential (2-.anytty.client.binding.v1.SSHCredentialRecordH RsshCredentiale
ssh_credential_sign (23.anytty.client.binding.v1.SSHCredentialSignResponseH RsshCredentialSignS
cloud_profile (2,.anytty.client.binding.v1.CloudProfileRecordH RcloudProfile_
local_discovery (24.anytty.client.binding.v1.LocalDiscoveryLookupResultH RlocalDiscoveryB

responseJJ#"8
PTYStreamSyncLost#
dropped_bytes (RdroppedBytes".
PTYStreamClosed
	exit_code (RexitCode"Å
EndpointSupervisorDemand
endpoint_id (	R
endpointIdD
mode (20.anytty.client.binding.v1.EndpointSupervisorModeRmode"¬
 EndpointSupervisorDemandSnapshot#
attachment_id (	RattachmentId'
demand_revision (RdemandRevisionP
	endpoints (22.anytty.client.binding.v1.EndpointSupervisorDemandR	endpoints"ê
EndpointSupervisorHostSignal
revision (Rrevision
	connected (R	connected
reason (	Rreason

foreground (R
foreground"¬
EndpointSupervisorProjection
endpoint_id (	R
endpointIdD
mode (20.anytty.client.binding.v1.EndpointSupervisorModeRmode
phase (	Rphase)
control_revision (RcontrolRevision

attempt_id (R	attemptId=
session (2#.anytty.api.v1.EndpointSessionStampRsession

error_code (	R	errorCode
message (	Rmessage
probe_count	 (R
probeCount

dial_count
 (R	dialCount#
backoff_count (RbackoffCount"r
EndpointSupervisorSnapshotT
	endpoints (26.anytty.client.binding.v1.EndpointSupervisorProjectionR	endpoints*à
ConnectIntent
CONNECT_INTENT_UNSPECIFIED 
CONNECT_INTENT_INTERACTIVE
CONNECT_INTENT_BACKGROUND
CONNECT_INTENT_PROBE*ß
ResourceStreamFrameType*
&RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED (
$RESOURCE_STREAM_FRAME_TYPE_FILE_DATA'
#RESOURCE_STREAM_FRAME_TYPE_FILE_ACK*
&RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH*
&RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT$
 RESOURCE_STREAM_FRAME_TYPE_ERROR/
+RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO)
%RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT,
(RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST)
%RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED	+
'RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA
-
)RESOURCE_STREAM_FRAME_TYPE_BROWSER_CLOSED*Í
ConnectionRouteKind%
!CONNECTION_ROUTE_KIND_UNSPECIFIED 
CONNECTION_ROUTE_KIND_LOCAL 
CONNECTION_ROUTE_KIND_DIRECT
CONNECTION_ROUTE_KIND_SSH
CONNECTION_ROUTE_KIND_CLOUD"*#CONNECTION_ROUTE_KIND_MANAGED_CLOUD*í
ConnectionObservedPath(
$CONNECTION_OBSERVED_PATH_UNSPECIFIED #
CONNECTION_OBSERVED_PATH_DIRECT)
%CONNECTION_OBSERVED_PATH_SINGLE_RELAY*Î
ConnectionCandidateType)
%CONNECTION_CANDIDATE_TYPE_UNSPECIFIED "
CONNECTION_CANDIDATE_TYPE_HOST.
*CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE,
(CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE#
CONNECTION_CANDIDATE_TYPE_RELAY*w
ConnectionTransport$
 CONNECTION_TRANSPORT_UNSPECIFIED 
CONNECTION_TRANSPORT_UDP
CONNECTION_TRANSPORT_TCP*…
"ConnectionPolicyAvailabilityReason5
1CONNECTION_POLICY_AVAILABILITY_REASON_UNSPECIFIED 3
/CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE>
:CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED8
4CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED>
:CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED@
<CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE;
7CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE*É
EndpointConnectionPhase)
%ENDPOINT_CONNECTION_PHASE_UNSPECIFIED "
ENDPOINT_CONNECTION_PHASE_IDLE&
"ENDPOINT_CONNECTION_PHASE_PLANNING'
#ENDPOINT_CONNECTION_PHASE_RESOLVING'
#ENDPOINT_CONNECTION_PHASE_SIGNALING(
$ENDPOINT_CONNECTION_PHASE_CONNECTING)
%ENDPOINT_CONNECTION_PHASE_AUTHORIZING#
ENDPOINT_CONNECTION_PHASE_READY%
!ENDPOINT_CONNECTION_PHASE_OFFLINE*é
EndpointSupervisorMode(
$ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED #
ENDPOINT_SUPERVISOR_MODE_SHADOW%
!ENDPOINT_SUPERVISOR_MODE_TAKEOVERB*Z(github.com/anytty/anytty/proto/bindingpbbproto3