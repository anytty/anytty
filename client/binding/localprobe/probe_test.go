package localprobe

import (
	"context"
	"net"
	"testing"

	"github.com/anytty/anytty/proto/bindingpb"
)

func TestReachableValidatesOrdersAndStopsAtSuccess(t *testing.T) {
	result := &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "not-an-ip", Port: 22},
		{Address: "::1", Port: 23},
		{Address: "127.0.0.1", Port: 24, NetworkHandle: 7},
	}}
	var addresses []string
	reachable := Reachable(context.Background(), result, func(_ context.Context, candidate *bindingpb.LocalDiscoveryCandidate, address string) (net.Conn, error) {
		addresses = append(addresses, address)
		if candidate.GetNetworkHandle() == 7 {
			left, right := net.Pipe()
			_ = right.Close()
			return left, nil
		}
		return nil, context.DeadlineExceeded
	})
	if !reachable {
		t.Fatal("Reachable returned false")
	}
	if len(addresses) != 1 || addresses[0] != "127.0.0.1:24" {
		t.Fatalf("probe order = %v", addresses)
	}
}

func TestReachableRejectsInvalidCandidates(t *testing.T) {
	result := &bindingpb.LocalDiscoveryLookupResult{Candidates: []*bindingpb.LocalDiscoveryCandidate{
		{Address: "example.com", Port: 22},
		{Address: "127.0.0.1", Port: 0},
	}}
	called := false
	if Reachable(context.Background(), result, func(context.Context, *bindingpb.LocalDiscoveryCandidate, string) (net.Conn, error) {
		called = true
		return nil, nil
	}) {
		t.Fatal("invalid candidates were reachable")
	}
	if called {
		t.Fatal("dialer was called for invalid candidates")
	}
}
