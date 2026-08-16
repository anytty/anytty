//go:build !android

package main

import (
	"context"
	"fmt"
	"net"

	"github.com/anytty/anytty/client/binding/localprobe"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/pion/transport/v4"
)

func newAndroidRouteNetwork(uint64) (transport.Net, error) {
	return nil, fmt.Errorf("Android route networks are unavailable on this platform")
}

func dialLocalDiscoveryCandidate(ctx context.Context, candidate *bindingpb.LocalDiscoveryCandidate, address string) (net.Conn, error) {
	return localprobe.DefaultDialCandidate(ctx, candidate, address)
}
