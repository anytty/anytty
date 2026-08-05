package remoteauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

func TestDirectSignalingAnswerSignatureBindsPinCorrelationAndLifetime(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := NewIdentity("device-direct-signature", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	answer := &remoteauthpb.DirectSignalingAnswerV2{
		SchemaVersion: DirectSignalingSchemaVersion, RequestId: "request-signature",
		Identity: &remoteauthpb.EndpointDaemonIdentity{
			DeviceId: identity.DeviceID, DevicePublicKey: append([]byte(nil), identity.PublicKey...), DeviceFingerprint: identity.Fingerprint,
		},
		AnswerSdp: "v=0\r\n", IssuedAtUnixNano: now.UnixNano(), ExpiresAtUnixNano: now.Add(DirectSignalingMaxTTL).UnixNano(),
	}
	if err := SignDirectSignalingAnswer(identity, answer); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDirectSignalingAnswer(answer, answer.GetRequestId(), identity.DeviceID, identity.Fingerprint, now); err != nil {
		t.Fatal(err)
	}
	tampered := proto.Clone(answer).(*remoteauthpb.DirectSignalingAnswerV2)
	tampered.AnswerSdp = "v=0\r\na=tampered\r\n"
	if err := VerifyDirectSignalingAnswer(tampered, answer.GetRequestId(), identity.DeviceID, identity.Fingerprint, now); err == nil {
		t.Fatal("tampered Direct signaling answer must fail signature verification")
	}
	if err := VerifyDirectSignalingAnswer(answer, "other-request", identity.DeviceID, identity.Fingerprint, now); err == nil {
		t.Fatal("Direct signaling answer must not cross request correlation")
	}
	if err := VerifyDirectSignalingAnswer(answer, answer.GetRequestId(), identity.DeviceID, "wrong-pin", now); err == nil {
		t.Fatal("Direct signaling answer must not cross Endpoint pin")
	}
	if err := VerifyDirectSignalingAnswer(answer, answer.GetRequestId(), identity.DeviceID, identity.Fingerprint, now.Add(DirectSignalingMaxTTL+time.Nanosecond)); err == nil {
		t.Fatal("expired Direct signaling answer must fail closed")
	}
}
