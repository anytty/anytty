package daemon

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

func TestDaemonRuntimeValidatesServerFirstChallengeIdentityAndGateway(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	challenge := &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{0x41}, ticket.EdgeChallengeNonceSize), EdgeId: "edge-agent", EdgeBootId: "edge-boot", StreamId: "edge-stream",
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(ticket.EdgeChallengeLifetime)), Target: cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY,
	}
	command := &cloudv1.EdgeCommand{
		ProtocolVersion: cloudprotocol.AgentGatewayVersion, MessageId: "challenge-message", SenderId: challenge.GetEdgeId(), BootId: challenge.GetEdgeBootId(), ConnectionId: challenge.GetStreamId(), StreamSeq: 1, SentAt: proto.Clone(challenge.GetIssuedAt()).(*timestamppb.Timestamp),
		Payload: &cloudv1.EdgeCommand_Challenge{Challenge: challenge},
	}
	if _, err := validateAgentGatewayChallenge(command, challenge.GetEdgeId(), now); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*cloudv1.EdgeCommand){
		"wrong edge": func(value *cloudv1.EdgeCommand) {
			value.GetChallenge().EdgeId = "edge-other"
			value.SenderId = "edge-other"
		},
		"wrong boot":   func(value *cloudv1.EdgeCommand) { value.BootId = "boot-other" },
		"wrong stream": func(value *cloudv1.EdgeCommand) { value.ConnectionId = "stream-other" },
		"cross gateway": func(value *cloudv1.EdgeCommand) {
			value.GetChallenge().Target = cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY
		},
		"expired": func(value *cloudv1.EdgeCommand) {
			value.GetChallenge().IssuedAt = timestamppb.New(now.Add(-ticket.EdgeChallengeLifetime - ticket.EdgeChallengeClockSkew))
			value.GetChallenge().ExpiresAt = timestamppb.New(now.Add(-ticket.EdgeChallengeClockSkew))
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := proto.Clone(command).(*cloudv1.EdgeCommand)
			mutate(value)
			if _, err := validateAgentGatewayChallenge(value, challenge.GetEdgeId(), now); err == nil {
				t.Fatal("invalid server-first challenge was accepted")
			}
		})
	}
}
