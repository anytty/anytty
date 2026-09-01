package cloud

import (
	"context"
	"errors"
	"io"
	"log"

	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cloudFailureStage string

type cloudSessionCloseOrigin string

const (
	cloudFailureAuthorization    cloudFailureStage = "authorization"
	cloudFailureController       cloudFailureStage = "controller"
	cloudFailureEdgeExchange     cloudFailureStage = "edge_exchange"
	cloudFailurePeerFingerprint  cloudFailureStage = "peer_fingerprint"
	cloudFailureDataChannelAuth  cloudFailureStage = "datachannel_auth"
	cloudFailureProtocolHello    cloudFailureStage = "protocol_hello"
	cloudFailureApplicationProbe cloudFailureStage = "application_probe"
	cloudFailurePairingRoute     cloudFailureStage = "pairing_route"
	cloudFailurePairingExchange  cloudFailureStage = "pairing_exchange"
	cloudFailurePairingHandshake cloudFailureStage = "pairing_handshake"

	cloudSessionCloseApplication cloudSessionCloseOrigin = "application"
	cloudSessionCloseSignaling   cloudSessionCloseOrigin = "signaling"
	cloudSessionCloseLocal       cloudSessionCloseOrigin = "local"
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

// reportCloudSessionClose deliberately emits only classifications derived from
// typed errors. Raw errors, session identifiers, endpoints, and addresses stay
// in the existing non-Android detailed diagnostics.
func reportCloudSessionClose(origin cloudSessionCloseOrigin, cause, terminalErr error) {
	runtimeCode := "none"
	if cause != nil {
		runtimeCode = string(clientruntime.CodeOf(cause))
	}
	rpcCode := "none"
	if terminalErr != nil {
		rpcCode = status.Code(terminalErr).String()
	}
	log.Printf(
		"anytty cloud session stage=closed origin=%s cause=%s code=%s rpc_code=%s retryable=%t",
		origin,
		cloudSessionCloseCause(origin, terminalErr),
		runtimeCode,
		rpcCode,
		clientruntime.IsRetryable(cause),
	)
}

func cloudSessionCloseCause(origin cloudSessionCloseOrigin, err error) string {
	if origin == cloudSessionCloseLocal {
		return "local_close"
	}
	if err == nil {
		return "ended"
	}
	if cloudclient.IsAdminDisconnect(err) {
		return "admin_disconnect"
	}
	switch {
	case errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case status.Code(err) != codes.Unknown:
		return "grpc_error"
	default:
		return "error"
	}
}
