package ticket_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDaemonBindingBindsEdgeAndRejectsTamper(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	claims := &cloudv1.DaemonBindingClaims{BindingId: "binding", DaemonId: "daemon", AccountId: "account", EdgeId: "edge-a", DeviceId: "device", DevicePublicKey: make([]byte, ed25519.PublicKeySize), Capabilities: []cloudv1.DaemonCapability{cloudv1.DaemonCapability_DAEMON_CAPABILITY_SIGNALING}, IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(time.Minute)), EdgeLocatorSha256: bytes.Repeat([]byte{0x41}, sha256.Size)}
	envelope, err := ticket.SignDaemonBinding("key", privateKey, claims)
	if err != nil {
		t.Fatal(err)
	}
	keys := ticket.KeySet{"key": publicKey}
	verified, err := ticket.VerifyDaemonBinding(envelope, keys, "edge-a", now, ticket.ClockSkewTolerance)
	if err != nil {
		t.Fatal(err)
	}
	if verified.GetBindingId() != claims.GetBindingId() {
		t.Fatal("verified daemon binding changed identity")
	}
	if _, err := ticket.VerifyDaemonBinding(envelope, keys, "edge-b", now, ticket.ClockSkewTolerance); err == nil {
		t.Fatal("binding accepted on another Edge")
	}
	withoutLocator := proto.Clone(claims).(*cloudv1.DaemonBindingClaims)
	withoutLocator.EdgeLocatorSha256 = nil
	if _, err := ticket.SignDaemonBinding("key", privateKey, withoutLocator); err == nil {
		t.Fatal("binding without an Edge locator digest was signed")
	}
	envelope.Payload[0] ^= 0xff
	if _, err := ticket.VerifyDaemonBinding(envelope, keys, "edge-a", now, ticket.ClockSkewTolerance); err == nil {
		t.Fatal("tampered binding accepted")
	}
}

func TestCloudRouteGrantBindsDaemonClientAndProduct(t *testing.T) {
	_, daemonPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	daemonIdentity, err := remoteauth.NewIdentity("device-r5-ticket", daemonPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	clientPublicKey, clientPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	grant, err := ticket.SignCloudRouteGrant(daemonIdentity, &cloudv1.CloudRouteGrantClaims{
		GrantId: "grant-r5", DaemonId: "daemon-r5", ClientPublicKey: clientPublicKey, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_CLI,
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(90 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ticket.VerifyCloudRouteGrant(grant, daemonIdentity.PublicKey, "daemon-r5", now)
	if err != nil || claims.GetProduct() != cloudv1.ClientProduct_CLIENT_PRODUCT_CLI {
		t.Fatalf("verified CloudRouteGrant=%v err=%v", claims, err)
	}
	if _, err := ticket.VerifyCloudRouteGrant(grant, daemonIdentity.PublicKey, "daemon-other", now); err == nil {
		t.Fatal("CloudRouteGrant accepted another daemon")
	}
	wildcardGrant, err := ticket.SignCloudRouteGrant(daemonIdentity, &cloudv1.CloudRouteGrantClaims{
		GrantId: "grant-r5-wildcard", DaemonId: "daemon-r5", ClientPublicKey: clientPublicKey, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED,
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(90 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatal(err)
	}
	wildcardClaims, err := ticket.VerifyCloudRouteGrant(wildcardGrant, daemonIdentity.PublicKey, "daemon-r5", now)
	if err != nil || wildcardClaims.GetProduct() != cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED {
		t.Fatalf("verified wildcard CloudRouteGrant=%v err=%v", wildcardClaims, err)
	}
	canonical, err := ticket.ClientRouteProofBytes("challenge-r5", bytes.Repeat([]byte{0x52}, 32), grant, "request-r5")
	if err != nil {
		t.Fatal(err)
	}
	proof := ed25519.Sign(clientPrivateKey, canonical)
	if err := ticket.VerifyClientRouteProof(clientPublicKey, proof, canonical); err != nil {
		t.Fatal(err)
	}
	if err := ticket.VerifyClientRouteProof(daemonIdentity.PublicKey, proof, canonical); err == nil {
		t.Fatal("Cloud route proof accepted another client key")
	}
	tampered := proto.Clone(grant).(*cloudv1.SignedEnvelope)
	tampered.KeyId = "another-device-fingerprint"
	if _, err := ticket.VerifyCloudRouteGrant(tampered, daemonIdentity.PublicKey, "daemon-r5", now); err == nil {
		t.Fatal("CloudRouteGrant accepted a mismatched DeviceIdentity key ID")
	}
}

func TestCloudRouteGrantUsesSharedClockSkewTolerance(t *testing.T) {
	_, daemonPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	daemonIdentity, err := remoteauth.NewIdentity("device-route-clock-skew", daemonPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	clientPublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	grant, err := ticket.SignCloudRouteGrant(daemonIdentity, &cloudv1.CloudRouteGrantClaims{
		GrantId: "grant-clock-skew", DaemonId: "daemon-clock-skew", ClientPublicKey: clientPublicKey,
		IssuedAt: timestamppb.New(now.Add(-time.Hour)), ExpiresAt: timestamppb.New(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ticket.VerifyCloudRouteGrant(grant, daemonIdentity.PublicKey, "daemon-clock-skew", now.Add(ticket.ClockSkewTolerance-time.Nanosecond)); err != nil {
		t.Fatalf("CloudRouteGrant within clock tolerance failed: %v", err)
	}
	if _, err := ticket.VerifyCloudRouteGrant(grant, daemonIdentity.PublicKey, "daemon-clock-skew", now.Add(ticket.ClockSkewTolerance)); err == nil {
		t.Fatal("CloudRouteGrant beyond clock tolerance was accepted")
	}
}

func TestCloudRouteGrantAllowsPermanentDaemonAuthorization(t *testing.T) {
	_, daemonPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	daemonIdentity, err := remoteauth.NewIdentity("device-route-permanent", daemonPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	clientPublicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	grant, err := ticket.SignCloudRouteGrant(daemonIdentity, &cloudv1.CloudRouteGrantClaims{
		GrantId: "grant-permanent", DaemonId: "daemon-permanent", ClientPublicKey: clientPublicKey,
		IssuedAt: timestamppb.New(now),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ticket.VerifyCloudRouteGrant(grant, daemonIdentity.PublicKey, "daemon-permanent", now.Add(20*365*24*time.Hour))
	if err != nil || claims.GetExpiresAt() != nil {
		t.Fatalf("permanent CloudRouteGrant = %#v, error = %v", claims, err)
	}
}

func TestEdgeChallengeRejectsWrongGatewayTimeAndLength(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	valid := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now)
	if err := ticket.ValidateEdgeChallenge(valid, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err != nil {
		t.Fatal(err)
	}
	withinSkew := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now.Add(ticket.EdgeChallengeClockSkew))
	if err := ticket.ValidateEdgeChallenge(withinSkew, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err != nil {
		t.Fatalf("challenge within clock skew was rejected: %v", err)
	}
	recentlyExpired := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now.Add(-ticket.EdgeChallengeLifetime-ticket.EdgeChallengeClockSkew+time.Nanosecond))
	if err := ticket.ValidateEdgeChallenge(recentlyExpired, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err != nil {
		t.Fatalf("challenge within expiration clock skew was rejected: %v", err)
	}
	tests := map[string]func(*cloudv1.EdgeChallenge){
		"short nonce": func(value *cloudv1.EdgeChallenge) { value.Nonce = value.Nonce[:31] },
		"long nonce":  func(value *cloudv1.EdgeChallenge) { value.Nonce = append(value.Nonce, 0) },
		"wrong gateway": func(value *cloudv1.EdgeChallenge) {
			value.Target = cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY
		},
		"future": func(value *cloudv1.EdgeChallenge) {
			value.IssuedAt = timestamppb.New(now.Add(ticket.EdgeChallengeClockSkew + time.Nanosecond))
			value.ExpiresAt = timestamppb.New(now.Add(ticket.EdgeChallengeLifetime + ticket.EdgeChallengeClockSkew + time.Nanosecond))
		},
		"expired": func(value *cloudv1.EdgeChallenge) {
			value.IssuedAt = timestamppb.New(now.Add(-ticket.EdgeChallengeLifetime - ticket.EdgeChallengeClockSkew))
			value.ExpiresAt = timestamppb.New(now.Add(-ticket.EdgeChallengeClockSkew))
		},
		"wrong lifetime": func(value *cloudv1.EdgeChallenge) {
			value.ExpiresAt = timestamppb.New(now.Add(ticket.EdgeChallengeLifetime + time.Nanosecond))
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := proto.Clone(valid).(*cloudv1.EdgeChallenge)
			mutate(value)
			if err := ticket.ValidateEdgeChallenge(value, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err == nil {
				t.Fatal("invalid challenge was accepted")
			}
		})
	}
}

func TestAgentHelloProofCoversChallengeBindingEnvelopeAndEveryHelloField(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("device-agent-proof", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	challenge := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY, now)
	event := &cloudv1.AgentEvent{
		ProtocolVersion: 2, MessageId: "agent-message", SenderId: "daemon-agent", BootId: "daemon-boot", ConnectionId: "daemon-session", StreamSeq: 1, SentAt: timestamppb.New(now),
		Payload: &cloudv1.AgentEvent_Hello{Hello: &cloudv1.AgentHello{
			DaemonBinding:   &cloudv1.SignedEnvelope{KeyId: "binding-key", Payload: []byte("binding-payload"), Signature: bytes.Repeat([]byte{0x41}, ed25519.SignatureSize)},
			SoftwareVersion: "agent-v2", AttemptGeneration: 7,
		}},
	}
	proof, err := ticket.SignAgentHelloProof(identity, challenge, event, now)
	if err != nil {
		t.Fatal(err)
	}
	event.GetHello().DeviceProof = proof
	if err := ticket.VerifyAgentHelloProof(identity.PublicKey, proof, challenge, event, now); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*cloudv1.EdgeChallenge, *cloudv1.AgentEvent){
		"challenge nonce": func(value *cloudv1.EdgeChallenge, _ *cloudv1.AgentEvent) { value.Nonce[0] ^= 1 },
		"edge id":         func(value *cloudv1.EdgeChallenge, _ *cloudv1.AgentEvent) { value.EdgeId += "-other" },
		"edge boot":       func(value *cloudv1.EdgeChallenge, _ *cloudv1.AgentEvent) { value.EdgeBootId += "-other" },
		"edge stream":     func(value *cloudv1.EdgeChallenge, _ *cloudv1.AgentEvent) { value.StreamId += "-other" },
		"binding key": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.GetHello().GetDaemonBinding().KeyId += "-other"
		},
		"binding payload": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.GetHello().GetDaemonBinding().Payload[0] ^= 1
		},
		"binding signature": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.GetHello().GetDaemonBinding().Signature[0] ^= 1
		},
		"protocol": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.ProtocolVersion++ },
		"message":  func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.MessageId += "-other" },
		"sender":   func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.SenderId += "-other" },
		"boot":     func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.BootId += "-other" },
		"session":  func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.ConnectionId += "-other" },
		"sequence": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) { value.StreamSeq++ },
		"sent at": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.SentAt = timestamppb.New(now.Add(time.Nanosecond))
		},
		"software version": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.GetHello().SoftwareVersion += "-other"
		},
		"attempt generation": func(_ *cloudv1.EdgeChallenge, value *cloudv1.AgentEvent) {
			value.GetHello().AttemptGeneration++
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changedChallenge := proto.Clone(challenge).(*cloudv1.EdgeChallenge)
			changedEvent := proto.Clone(event).(*cloudv1.AgentEvent)
			mutate(changedChallenge, changedEvent)
			if err := ticket.VerifyAgentHelloProof(identity.PublicKey, proof, changedChallenge, changedEvent, now); err == nil {
				t.Fatal("tampered AgentHello proof was accepted")
			}
		})
	}
}

func TestClientHelloProofCoversChallengeAuthorizationAndEveryHelloField(t *testing.T) {
	clientPublicKey, clientPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	devicePublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	admission := &cloudv1.PairingAdmission{
		DaemonId: "daemon-pairing", DeviceId: "device-pairing", DevicePublicKey: devicePublicKey,
		PairingClaimSha256: bytes.Repeat([]byte{0x71}, sha256.Size), ExpiresAtUnixNano: now.Add(10 * time.Minute).UnixNano(),
	}
	challenge := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now)
	event := &cloudv1.ClientSignal{
		ProtocolVersion: 2, MessageId: "client-message", SenderId: remoteauth.Fingerprint(clientPublicKey), BootId: "client-boot", ConnectionId: "client-session", StreamSeq: 1, SentAt: timestamppb.New(now),
		Payload: &cloudv1.ClientSignal_Hello{Hello: &cloudv1.ClientHello{
			ClientPublicKey: clientPublicKey, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_CLI, SoftwareVersion: "client-v2", AttemptGeneration: 7,
			RelayPreference: cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO, Authorization: &cloudv1.ClientHello_PairingAdmission{PairingAdmission: admission},
		}},
	}
	canonical, err := ticket.ClientHelloProofBytes(challenge, event, now)
	if err != nil {
		t.Fatal(err)
	}
	proof := ed25519.Sign(clientPrivateKey, canonical)
	event.GetHello().ClientProof = proof
	if err := ticket.VerifyClientHelloProof(clientPublicKey, proof, challenge, event, now); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*cloudv1.EdgeChallenge, *cloudv1.ClientSignal){
		"challenge nonce": func(value *cloudv1.EdgeChallenge, _ *cloudv1.ClientSignal) { value.Nonce[0] ^= 1 },
		"edge id":         func(value *cloudv1.EdgeChallenge, _ *cloudv1.ClientSignal) { value.EdgeId += "-other" },
		"edge boot":       func(value *cloudv1.EdgeChallenge, _ *cloudv1.ClientSignal) { value.EdgeBootId += "-other" },
		"edge stream":     func(value *cloudv1.EdgeChallenge, _ *cloudv1.ClientSignal) { value.StreamId += "-other" },
		"authorization": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) {
			value.GetHello().GetPairingAdmission().PairingClaimSha256[0] ^= 1
		},
		"protocol": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.ProtocolVersion++ },
		"message":  func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.MessageId += "-other" },
		"sender":   func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.SenderId += "-other" },
		"boot":     func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.BootId += "-other" },
		"session":  func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.ConnectionId += "-other" },
		"sequence": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.StreamSeq++ },
		"sent at": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) {
			value.SentAt = timestamppb.New(now.Add(time.Nanosecond))
		},
		"public key": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.GetHello().ClientPublicKey[0] ^= 1 },
		"product":    func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.GetHello().Product++ },
		"software version": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) {
			value.GetHello().SoftwareVersion += "-other"
		},
		"attempt generation": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) {
			value.GetHello().AttemptGeneration++
		},
		"relay preference": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) { value.GetHello().RelayPreference++ },
		"presence probe": func(_ *cloudv1.EdgeChallenge, value *cloudv1.ClientSignal) {
			value.GetHello().PresenceProbe = !value.GetHello().PresenceProbe
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changedChallenge := proto.Clone(challenge).(*cloudv1.EdgeChallenge)
			changedEvent := proto.Clone(event).(*cloudv1.ClientSignal)
			mutate(changedChallenge, changedEvent)
			if err := ticket.VerifyClientHelloProof(clientPublicKey, proof, changedChallenge, changedEvent, now); err == nil {
				t.Fatal("tampered ClientHello proof was accepted")
			}
		})
	}
}

func TestClientHelloProofCoversCompleteRouteGrantEnvelope(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	challenge := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now)
	event := &cloudv1.ClientSignal{
		ProtocolVersion: 2, MessageId: "route-message", SenderId: remoteauth.Fingerprint(publicKey), BootId: "route-boot", ConnectionId: "route-session", StreamSeq: 1, SentAt: timestamppb.New(now),
		Payload: &cloudv1.ClientSignal_Hello{Hello: &cloudv1.ClientHello{
			ClientPublicKey: publicKey, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_CLI, SoftwareVersion: "client-v2", AttemptGeneration: 9, RelayPreference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY,
			Authorization: &cloudv1.ClientHello_CloudRouteGrant{CloudRouteGrant: &cloudv1.SignedEnvelope{KeyId: "daemon-key", Payload: []byte("route-grant"), Signature: bytes.Repeat([]byte{0x51}, ed25519.SignatureSize)}},
		}},
	}
	canonical, err := ticket.ClientHelloProofBytes(challenge, event, now)
	if err != nil {
		t.Fatal(err)
	}
	proof := ed25519.Sign(privateKey, canonical)
	for name, mutate := range map[string]func(*cloudv1.SignedEnvelope){
		"key id":    func(value *cloudv1.SignedEnvelope) { value.KeyId += "-other" },
		"payload":   func(value *cloudv1.SignedEnvelope) { value.Payload[0] ^= 1 },
		"signature": func(value *cloudv1.SignedEnvelope) { value.Signature[0] ^= 1 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.Clone(event).(*cloudv1.ClientSignal)
			mutate(changed.GetHello().GetCloudRouteGrant())
			if err := ticket.VerifyClientHelloProof(publicKey, proof, challenge, changed, now); err == nil {
				t.Fatal("tampered route grant envelope was accepted")
			}
		})
	}
}

func TestGatewayHelloProofRejectsCrossGatewayChallengeAndDomain(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientChallenge := testEdgeChallenge(cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now)
	clientEvent := &cloudv1.ClientSignal{
		ProtocolVersion: 2, MessageId: "message", SenderId: remoteauth.Fingerprint(publicKey), BootId: "boot", ConnectionId: "session", StreamSeq: 1, SentAt: timestamppb.New(now),
		Payload: &cloudv1.ClientSignal_Hello{Hello: &cloudv1.ClientHello{ClientPublicKey: publicKey, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_CLI, SoftwareVersion: "v2", AttemptGeneration: 1, RelayPreference: cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO, Authorization: &cloudv1.ClientHello_PairingAdmission{PairingAdmission: &cloudv1.PairingAdmission{DaemonId: "daemon", DeviceId: "device", DevicePublicKey: bytes.Repeat([]byte{1}, ed25519.PublicKeySize), PairingClaimSha256: bytes.Repeat([]byte{2}, sha256.Size), ExpiresAtUnixNano: now.Add(time.Minute).UnixNano()}}}},
	}
	canonical, err := ticket.ClientHelloProofBytes(clientChallenge, clientEvent, now)
	if err != nil {
		t.Fatal(err)
	}
	proof := ed25519.Sign(privateKey, canonical)
	agentChallenge := proto.Clone(clientChallenge).(*cloudv1.EdgeChallenge)
	agentChallenge.Target = cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY
	if _, err := ticket.ClientHelloProofBytes(agentChallenge, clientEvent, now); err == nil {
		t.Fatal("ClientHello accepted an AgentGateway challenge")
	}
	changed := proto.Clone(clientChallenge).(*cloudv1.EdgeChallenge)
	changed.StreamId = "another-gateway-stream"
	if err := ticket.VerifyClientHelloProof(publicKey, proof, changed, clientEvent, now); err == nil {
		t.Fatal("ClientHello proof crossed Gateway stream identity")
	}
}

func testEdgeChallenge(target cloudv1.EdgeChallengeTarget, now time.Time) *cloudv1.EdgeChallenge {
	return &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{0x42}, ticket.EdgeChallengeNonceSize), EdgeId: "edge-v2", EdgeBootId: "edge-boot-v2", StreamId: "edge-stream-v2",
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(ticket.EdgeChallengeLifetime)), Target: target,
	}
}
