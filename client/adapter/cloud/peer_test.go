package cloud

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestCloudPeerRaceWaitsForAuthenticationAndClosesLatePeer(t *testing.T) {
	attempts, err := planCloudPeerAttempts(endpoint.RelayAuto, endpoint.RelayTransportTCP)
	if err != nil {
		t.Fatal(err)
	}
	directConnected := make(chan struct{})
	winnerConn, winnerRemote := memory.NewPair()
	defer winnerRemote.Close()
	lateConn, lateRemote := memory.NewPair()
	defer lateRemote.Close()
	winner := &openedCloudPeer{connection: winnerConn}
	defer winner.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := raceCloudPeerAttempts(ctx, attempts, func(ctx context.Context, attempt cloudPeerAttempt) (*openedCloudPeer, error) {
		if attempt.icePolicy == port.ICETransportAll {
			// This transport connected first, but has not completed authentication.
			close(directConnected)
			<-ctx.Done()
			return &openedCloudPeer{connection: lateConn}, nil
		}
		<-directConnected
		return winner, nil
	})
	if err != nil || got != winner {
		t.Fatalf("authenticated winner = %p, err=%v", got, err)
	}
	closed := make(chan error, 1)
	go func() { _, err := lateRemote.Recv(); closed <- err }()
	select {
	case err := <-closed:
		if err == nil {
			t.Fatal("late transport was not closed")
		}
	case <-ctx.Done():
		t.Fatal("losing authentication attempt was not canceled and released")
	}
}

func TestCloudPeerRaceRetainsEveryAuthenticationFailure(t *testing.T) {
	attempts := []cloudPeerAttempt{
		{relayTransport: endpoint.RelayTransportTCP},
		{relayTransport: endpoint.RelayTransportUDP},
	}
	tcpErr, udpErr := errors.New("TCP certificate mismatch"), errors.New("UDP capability rejected")
	opened, err := raceCloudPeerAttempts(context.Background(), attempts, func(_ context.Context, attempt cloudPeerAttempt) (*openedCloudPeer, error) {
		if attempt.relayTransport == endpoint.RelayTransportTCP {
			return nil, tcpErr
		}
		return nil, udpErr
	})
	if opened != nil || !errors.Is(err, tcpErr) || !errors.Is(err, udpErr) {
		t.Fatalf("authentication failures lost: opened=%v err=%v", opened, err)
	}
}

func TestCloudPeerAttemptsDefaultToDirectAndTCPRelay(t *testing.T) {
	for _, mode := range []endpoint.RelayMode{"", endpoint.RelayAuto, endpoint.RelaySmart} {
		for _, preference := range []endpoint.RelayTransport{"", endpoint.RelayTransportAuto} {
			attempts, err := planCloudPeerAttempts(mode, preference)
			if err != nil {
				t.Fatalf("planCloudPeerAttempts(%q): %v", mode, err)
			}
			if len(attempts) != 2 {
				t.Fatalf("planCloudPeerAttempts(%q) returned %d attempts", mode, len(attempts))
			}
			if attempts[0].preference != cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY || attempts[0].icePolicy != port.ICETransportAll {
				t.Fatalf("direct attempt for %q = %#v", mode, attempts[0])
			}
			if attempts[1].relayTransport != endpoint.RelayTransportTCP || attempts[1].icePolicy != port.ICETransportRelayOnly {
				t.Fatalf("relay attempts for %q = %#v", mode, attempts[1:])
			}
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
	for _, preference := range []endpoint.RelayTransport{"", endpoint.RelayTransportAuto} {
		relay, err = planCloudPeerAttempts(endpoint.RelayOnly, preference)
		if err != nil || len(relay) != 1 || relay[0].relayTransport != endpoint.RelayTransportTCP {
			t.Fatalf("automatic relay attempts=%#v err=%v", relay, err)
		}
	}
	for _, mode := range []endpoint.RelayMode{endpoint.RelayAuto, endpoint.RelaySmart, endpoint.RelayOnly} {
		attempts, err := planCloudPeerAttempts(mode, endpoint.RelayTransportUDP)
		if err != nil {
			t.Fatal(err)
		}
		wantCount := 2
		if mode == endpoint.RelayOnly {
			wantCount = 1
		}
		if len(attempts) != wantCount || attempts[len(attempts)-1].relayTransport != endpoint.RelayTransportUDP {
			t.Fatalf("explicit UDP mode=%q attempts=%#v", mode, attempts)
		}
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
