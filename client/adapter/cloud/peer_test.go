package cloud

import (
	"context"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudPeerAttemptsProbeDirectAndBothRelayTransports(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		attempts, err := planCloudPeerAttempts(mode, endpoint.RelayTransportAuto)
		if err != nil {
			t.Fatalf("planCloudPeerAttempts(%q): %v", mode, err)
		}
		if len(attempts) != 3 {
			t.Fatalf("planCloudPeerAttempts(%q) returned %d attempts", mode, len(attempts))
		}
		if attempts[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || attempts[0].icePolicy != port.ICETransportAll {
			t.Fatalf("direct attempt for %q = %#v", mode, attempts[0])
		}
		if attempts[1].relayTransport != endpoint.RelayTransportTCP || attempts[2].relayTransport != endpoint.RelayTransportUDP {
			t.Fatalf("relay attempts for %q = %#v", mode, attempts[1:])
		}
	}
}

func TestCloudPeerAttemptsPreserveExplicitPolicies(t *testing.T) {
	direct, err := planCloudPeerAttempts(endpoint.RelayDirect, endpoint.RelayTransportTCP)
	if err != nil || len(direct) != 1 || direct[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || direct[0].icePolicy != port.ICETransportAll {
		t.Fatalf("direct attempts=%#v err=%v", direct, err)
	}
	relay, err := planCloudPeerAttempts(endpoint.RelayOnly, endpoint.RelayTransportTCP)
	if err != nil || len(relay) != 1 || relay[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_RELAY_ONLY || relay[0].icePolicy != port.ICETransportRelayOnly || relay[0].relayTransport != endpoint.RelayTransportTCP {
		t.Fatalf("relay attempts=%#v err=%v", relay, err)
	}
	relay, err = planCloudPeerAttempts(endpoint.RelayOnly, endpoint.RelayTransportAuto)
	if err != nil || len(relay) != 2 || relay[0].relayTransport != endpoint.RelayTransportTCP || relay[1].relayTransport != endpoint.RelayTransportUDP {
		t.Fatalf("automatic relay attempts=%#v err=%v", relay, err)
	}
	if _, err := planCloudPeerAttempts("invalid", endpoint.RelayTransportAuto); err == nil {
		t.Fatal("invalid relay mode was accepted")
	}
	if _, err := planCloudPeerAttempts(endpoint.RelayOnly, "invalid"); err == nil {
		t.Fatal("invalid relay transport was accepted")
	}
}

func TestReleaseCloudSessionOnlyReleasesConfirmedLiveSession(t *testing.T) {
	tests := []struct {
		name      string
		confirmed bool
		closed    bool
		want      []string
	}{
		{name: "confirmed", confirmed: true, want: []string{"release"}},
		{name: "unconfirmed", want: nil},
		{name: "signaling already closed", confirmed: true, closed: true, want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			order := make([]string, 0, 1)
			signaling := &closeTrackingCloudSignaling{confirmed: test.confirmed, done: make(chan struct{}), order: &order}
			if test.closed {
				close(signaling.done)
			}
			if err := releaseCloudSession(signaling); err != nil {
				t.Fatal(err)
			}
			if len(order) != len(test.want) || (len(test.want) == 1 && order[0] != test.want[0]) {
				t.Fatalf("release actions = %#v, want %#v", order, test.want)
			}
		})
	}
}

type closeTrackingCloudSignaling struct {
	confirmed bool
	done      chan struct{}
	order     *[]string
}

func (signaling *closeTrackingCloudSignaling) Done() <-chan struct{} {
	return signaling.done
}
func (signaling *closeTrackingCloudSignaling) PathConfirmed() bool { return signaling.confirmed }
func (signaling *closeTrackingCloudSignaling) ReleaseAndWait(context.Context) error {
	*signaling.order = append(*signaling.order, "release")
	return nil
}
