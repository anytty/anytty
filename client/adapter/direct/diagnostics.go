package direct

import (
	"errors"
	"log"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

type directFailureStage string

const (
	directFailureAuthorization    directFailureStage = "authorization"
	directFailurePeerSetup        directFailureStage = "peer_setup"
	directFailureDataChannelAuth  directFailureStage = "datachannel_auth"
	directFailureProtocolHello    directFailureStage = "protocol_hello"
	directFailurePairingSetup     directFailureStage = "pairing_setup"
	directFailurePairingHandshake directFailureStage = "pairing_handshake"
)

// reportDirectFailure emits only stable classifications. The raw error,
// locators, SDP, pairing claim, and credentials must never enter this log.
func reportDirectFailure(generation clientruntime.SessionGeneration, stage directFailureStage, err error) error {
	if err == nil {
		return nil
	}
	log.Printf(
		"anytty direct failure generation=%d stage=%s code=%s signaling_code=%s auth_code=%s retryable=%t",
		generation,
		stage,
		clientruntime.CodeOf(err),
		directSignalingCode(err),
		remoteAuthCode(err),
		clientruntime.IsRetryable(err),
	)
	return err
}

func directSignalingCode(err error) string {
	var signalingErr *SignalingError
	if errors.As(err, &signalingErr) && signalingErr.Code != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED {
		return signalingErr.Code.String()
	}
	return "none"
}

func remoteAuthCode(err error) string {
	var handshakeErr *remoteauth.HandshakeError
	if errors.As(err, &handshakeErr) && handshakeErr.Code != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_UNSPECIFIED {
		return handshakeErr.Code.String()
	}
	return "none"
}
