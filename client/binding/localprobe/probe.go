// Package localprobe verifies that a discovered local endpoint is currently
// reachable. DNS-SD browsing remains a platform primitive; validation,
// ordering, deadlines, and connection attempts are shared here.
package localprobe

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/anytty/anytty/proto/bindingpb"
)

const CandidateTimeout = 400 * time.Millisecond

type DialCandidate func(context.Context, *bindingpb.LocalDiscoveryCandidate, string) (net.Conn, error)

func DefaultDialCandidate(ctx context.Context, _ *bindingpb.LocalDiscoveryCandidate, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp", address)
}

// Reachable probes valid numeric candidates, preferring IPv4 to match the
// mobile route policy. It stops on the first successful TCP connection.
func Reachable(ctx context.Context, result *bindingpb.LocalDiscoveryLookupResult, dial DialCandidate) bool {
	if result == nil {
		return false
	}
	if dial == nil {
		dial = DefaultDialCandidate
	}
	candidates := append([]*bindingpb.LocalDiscoveryCandidate(nil), result.GetCandidates()...)
	sort.SliceStable(candidates, func(left, right int) bool {
		return net.ParseIP(candidates[left].GetAddress()).To4() != nil && net.ParseIP(candidates[right].GetAddress()).To4() == nil
	})
	for _, candidate := range candidates {
		ip := net.ParseIP(candidate.GetAddress())
		if ip == nil || candidate.GetPort() == 0 || candidate.GetPort() > 65535 {
			continue
		}
		candidateCtx, cancel := context.WithTimeout(ctx, CandidateTimeout)
		connection, err := dial(candidateCtx, candidate, net.JoinHostPort(ip.String(), fmt.Sprint(candidate.GetPort())))
		cancel()
		if err == nil && connection != nil {
			_ = connection.Close()
			return true
		}
		if connection != nil {
			_ = connection.Close()
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return false
}
