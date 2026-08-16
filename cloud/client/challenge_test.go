package client

import (
	"bytes"
	"testing"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCloudClientValidatesServerFirstChallengeIdentityAndWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	challenge := &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{0x31}, ticket.EdgeChallengeNonceSize), EdgeId: "edge-client", EdgeBootId: "edge-boot", StreamId: "edge-stream",
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(ticket.EdgeChallengeLifetime)), Target: cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY,
	}
	signal := &cloudv1.EdgeSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: "challenge-message", SenderId: challenge.GetEdgeId(), BootId: challenge.GetEdgeBootId(), ConnectionId: challenge.GetStreamId(), StreamSeq: 1, SentAt: proto.Clone(challenge.GetIssuedAt()).(*timestamppb.Timestamp),
		Payload: &cloudv1.EdgeSignal_Challenge{Challenge: challenge},
	}
	if _, err := validateClientGatewayChallenge(signal, challenge.GetEdgeId(), now); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*cloudv1.EdgeSignal){
		"wrong edge": func(value *cloudv1.EdgeSignal) {
			value.GetChallenge().EdgeId = "edge-other"
			value.SenderId = "edge-other"
		},
		"wrong boot":   func(value *cloudv1.EdgeSignal) { value.BootId = "boot-other" },
		"wrong stream": func(value *cloudv1.EdgeSignal) { value.ConnectionId = "stream-other" },
		"wrong target": func(value *cloudv1.EdgeSignal) {
			value.GetChallenge().Target = cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY
		},
		"wrong length": func(value *cloudv1.EdgeSignal) { value.GetChallenge().Nonce = value.GetChallenge().Nonce[:31] },
		"future": func(value *cloudv1.EdgeSignal) {
			value.GetChallenge().IssuedAt = timestamppb.New(now.Add(ticket.EdgeChallengeClockSkew + time.Nanosecond))
			value.GetChallenge().ExpiresAt = timestamppb.New(now.Add(ticket.EdgeChallengeLifetime + ticket.EdgeChallengeClockSkew + time.Nanosecond))
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := proto.Clone(signal).(*cloudv1.EdgeSignal)
			mutate(value)
			if _, err := validateClientGatewayChallenge(value, challenge.GetEdgeId(), now); err == nil {
				t.Fatal("invalid server-first challenge was accepted")
			}
		})
	}
}
