package cloud

import (
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudPeerAttemptUsesSingleICECompetitionForAutomaticModes(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		attempt, err := planCloudPeerAttempt(mode)
		if err != nil {
			t.Fatalf("planCloudPeerAttempt(%q): %v", mode, err)
		}
		if attempt.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_AUTO || attempt.icePolicy != port.ICETransportAll {
			t.Fatalf("planCloudPeerAttempt(%q) = %#v", mode, attempt)
		}
	}
}

func TestCloudPeerAttemptPreservesExplicitPolicies(t *testing.T) {
	direct, err := planCloudPeerAttempt(endpoint.RelayDirect)
	if err != nil || direct.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || direct.icePolicy != port.ICETransportAll {
		t.Fatalf("direct attempts=%#v err=%v", direct, err)
	}
	relay, err := planCloudPeerAttempt(endpoint.RelayOnly)
	if err != nil || relay.preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY || relay.icePolicy != port.ICETransportRelayOnly {
		t.Fatalf("relay attempts=%#v err=%v", relay, err)
	}
	if _, err := planCloudPeerAttempt("invalid"); err == nil {
		t.Fatal("invalid relay mode was accepted")
	}
}
