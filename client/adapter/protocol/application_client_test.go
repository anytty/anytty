package protocol

import (
	"context"
	"crypto/ed25519"
	"errors"
	"io"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport/datachannel"
	"google.golang.org/protobuf/proto"
)

func TestClassifyTransportTerminalErrorMakesRawFailuresRetryable(t *testing.T) {
	cause := errors.New("data channel send failed")
	err := classifyTransportTerminalError(cause)
	var runtimeErr *clientruntime.Error
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != clientruntime.ErrorUnavailable || !runtimeErr.Retryable || !runtimeErr.Attempted || !errors.Is(err, cause) {
		t.Fatalf("classified transport error = %#v", err)
	}
}

func TestClassifyTransportTerminalErrorPreservesNonRetryableFailures(t *testing.T) {
	existing := &clientruntime.Error{Code: clientruntime.ErrorResourceExhausted, Message: "full"}
	peer := &internalprotocol.PeerError{Code: internalprotocol.ProtocolErrorCodeResourceExhausted, Message: "full"}
	request := &internalprotocol.RequestError{Code: internalprotocol.ProtocolErrorCodeBadRequest, Message: "invalid"}
	for name, source := range map[string]error{
		"runtime":            existing,
		"peer":               peer,
		"request":            request,
		"receive queue":      datachannel.ErrReceiveQueueExhausted,
		"frame size":         wire.ErrFrameTooLarge,
		"local cancellation": context.Canceled,
		"clean eof":          io.EOF,
	} {
		t.Run(name, func(t *testing.T) {
			if got := classifyTransportTerminalError(source); got != source {
				t.Fatalf("classified error = %#v, want original %#v", got, source)
			}
			if clientruntime.IsRetryable(source) {
				t.Fatalf("preserved error unexpectedly retryable: %v", source)
			}
		})
	}
}

func TestVerifyDaemonIdentityChecksPublicKeyFingerprintAndPin(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: remoteauth.Fingerprint(publicKey)}
	executor := identityExecutor{identity: remoteauth.Identity{DeviceID: expected.DeviceID, Fingerprint: expected.DeviceFingerprint, PublicKey: publicKey, PrivateKey: privateKey}, projection: &remoteauthpb.ClientAccessIdentityResult{
		DeviceId: expected.DeviceID, DeviceFingerprint: expected.DeviceFingerprint, DevicePublicKey: append([]byte(nil), publicKey...),
	}}
	session, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: "studio", RouteID: "ssh", Generation: 4}, executor)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyDaemonIdentity(context.Background(), session, expected)
	if err != nil {
		t.Fatal(err)
	}
	if verified != expected {
		t.Fatalf("verified identity = %#v, want %#v", verified, expected)
	}

	executor.projection.DeviceFingerprint = "SHA256:wrong"
	if _, err := VerifyDaemonIdentity(context.Background(), session, endpoint.DaemonIdentity{}); err == nil {
		t.Fatal("public key fingerprint mismatch must fail")
	}
}

type identityExecutor struct {
	identity   remoteauth.Identity
	projection *remoteauthpb.ClientAccessIdentityResult
}

func (executor identityExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	challenge := command.GetClientAccessIdentity().GetChallenge()
	proof, err := remoteauth.SignDeviceIdentityProof(executor.identity, challenge)
	if err != nil {
		return nil, err
	}
	return &apipb.ResultEnvelope{
		RequestId: command.GetContext().GetRequestId(), OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result: &apipb.ResultEnvelope_ClientAccessIdentity{ClientAccessIdentity: &apipb.ClientAccessIdentityResult{Identity: proto.Clone(executor.projection).(*remoteauthpb.ClientAccessIdentityResult), Challenge: append([]byte(nil), challenge...), Proof: proof}},
	}, nil
}
