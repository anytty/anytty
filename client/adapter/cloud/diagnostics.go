package cloud

import (
	"errors"
	"log"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc/status"
)

type cloudFailureStage string

const (
	cloudFailureAuthorization    cloudFailureStage = "authorization"
	cloudFailureController       cloudFailureStage = "controller"
	cloudFailureEdgeExchange     cloudFailureStage = "edge_exchange"
	cloudFailurePeerFingerprint  cloudFailureStage = "peer_fingerprint"
	cloudFailureDataChannelAuth  cloudFailureStage = "datachannel_auth"
	cloudFailureProtocolHello    cloudFailureStage = "protocol_hello"
	cloudFailurePairingRoute     cloudFailureStage = "pairing_route"
	cloudFailurePairingExchange  cloudFailureStage = "pairing_exchange"
	cloudFailurePairingHandshake cloudFailureStage = "pairing_handshake"
)

// reportCloudFailure emits only stable classifications. The raw error, endpoint,
// signaling payload, addresses, SDP, and credentials must never enter this log.
func reportCloudFailure(generation clientruntime.SessionGeneration, stage cloudFailureStage, err error) error {
	if err == nil {
		return nil
	}
	log.Printf(
		"anytty cloud failure generation=%d stage=%s code=%s rpc_code=%s auth_code=%s retryable=%t",
		generation,
		stage,
		clientruntime.CodeOf(err),
		status.Code(err).String(),
		cloudRemoteAuthCode(err),
		clientruntime.IsRetryable(err),
	)
	return err
}

func cloudRemoteAuthCode(err error) string {
	var handshakeErr *remoteauth.HandshakeError
	if errors.As(err, &handshakeErr) && handshakeErr.Code != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_UNSPECIFIED {
		return handshakeErr.Code.String()
	}
	return "none"
}
