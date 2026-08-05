package protocol

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/protobuf/proto"
)

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
