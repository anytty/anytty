package ice

import (
	"testing"
	"time"
)

func TestRelayAcceptanceWaitUsesFinalOptions(t *testing.T) {
	relayOnly := WithCandidateTypes([]CandidateType{CandidateTypeRelay})
	mixed := WithCandidateTypes([]CandidateType{CandidateTypeHost, CandidateTypeRelay})
	explicit := 7 * time.Second
	for _, test := range []struct {
		name    string
		config  *AgentConfig
		options []AgentOption
		want    time.Duration
	}{
		{name: "default", want: 2 * time.Second},
		{name: "relay only", options: []AgentOption{relayOnly}, want: 0},
		{name: "mixed", options: []AgentOption{mixed}, want: 2 * time.Second},
		{name: "last candidate option wins", options: []AgentOption{relayOnly, mixed}, want: 2 * time.Second},
		{name: "explicit before candidates", options: []AgentOption{WithRelayAcceptanceMinWait(explicit), relayOnly}, want: explicit},
		{name: "explicit after candidates", options: []AgentOption{relayOnly, WithRelayAcceptanceMinWait(explicit)}, want: explicit},
		{name: "explicit zero mixed", options: []AgentOption{WithRelayAcceptanceMinWait(0), mixed}, want: 0},
		{name: "legacy explicit", config: &AgentConfig{RelayAcceptanceMinWait: &explicit}, options: []AgentOption{relayOnly}, want: explicit},
		{name: "legacy relay only", config: &AgentConfig{CandidateTypes: []CandidateType{CandidateTypeRelay}}, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			agent, err := newAgentFromConfig(test.config, test.options...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := agent.Close(); err != nil {
					t.Error(err)
				}
			}()
			if agent.relayAcceptanceMinWait != test.want {
				t.Fatalf("relay nomination wait = %v, want %v", agent.relayAcceptanceMinWait, test.want)
			}
		})
	}
}
