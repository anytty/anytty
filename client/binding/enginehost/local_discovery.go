package enginehost

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/wire"
)

const localDiscoveryRouteID endpoint.RouteID = "lan-discovery"

// applyPlatformLocalDiscovery adds untrusted, short-lived LAN locators only to
// this connection plan. The platform cache is never allowed to mutate the
// persisted registry or replace the endpoint identity pin.
func applyPlatformLocalDiscovery(ctx context.Context, target endpoint.Endpoint, options Options) endpoint.Endpoint {
	if options.Broker == nil || options.DirectPeers == nil || target.DaemonIdentity.Empty() {
		return target
	}
	response, err := options.Broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_LocalDiscoveryLookup{
		LocalDiscoveryLookup: &bindingpb.LocalDiscoveryLookupRequest{
			DeviceId: target.DaemonIdentity.DeviceID, DeviceFingerprint: target.DaemonIdentity.DeviceFingerprint,
		},
	}})
	if err != nil || platformResponseError(response) != nil || response.GetLocalDiscovery() == nil {
		return target
	}
	now := time.Now().UTC()
	if options.Now != nil {
		now = options.Now().UTC()
	}
	groups := make(map[uint64]map[string]struct{})
	for _, value := range response.GetLocalDiscovery().GetCandidates() {
		if value == nil || value.GetPort() == 0 || value.GetPort() > 65535 || value.GetProtocolVersion() != uint32(wire.Version) {
			continue
		}
		candidate := endpoint.LocalDiscoveryCandidate{
			ClaimedIdentity: target.DaemonIdentity, Address: strings.TrimSpace(value.GetAddress()), Port: uint16(value.GetPort()),
			ProtocolVersion: value.GetProtocolVersion(), AnnouncementExpiry: time.Unix(0, value.GetExpiresAtUnixNano()).UTC(),
		}
		if candidate.Validate(now) != nil {
			continue
		}
		handle := value.GetNetworkHandle()
		if groups[handle] == nil {
			groups[handle] = make(map[string]struct{})
		}
		groups[handle][net.JoinHostPort(candidate.Address, strconv.Itoa(int(candidate.Port)))] = struct{}{}
	}
	if len(groups) == 0 {
		return target
	}
	routes := make(map[endpoint.RouteID]endpoint.AccessRoute, len(target.Routes)+1)
	for routeID, route := range target.Routes {
		routes[routeID] = route
	}
	target.Routes = routes
	credentialRef, priority := localDiscoveryCredential(target)
	if credentialRef == "" {
		return target
	}
	handles := make([]uint64, 0, len(groups))
	for handle := range groups {
		handles = append(handles, handle)
	}
	sort.Slice(handles, func(left, right int) bool { return handles[left] < handles[right] })
	for _, handle := range handles {
		locators := make([]string, 0, len(groups[handle]))
		for address := range groups[handle] {
			locators = append(locators, address)
		}
		sort.Strings(locators)
		routeID := availableLocalDiscoveryRouteID(target.Routes)
		target.Routes[routeID] = endpoint.AccessRoute{
			ID: routeID, DisplayName: "Local network", Kind: endpoint.RouteDirectWebRTCTCP,
			Enabled: true, Priority: priority, CredentialRef: credentialRef, Source: endpoint.SourceLAN, PolicySource: endpoint.SourceLAN,
			SignalingAddresses: append([]string(nil), locators...), ICETCPAddresses: append([]string(nil), locators...),
			NetworkHandle: handle,
		}
	}
	return target
}

func availableLocalDiscoveryRouteID(routes map[endpoint.RouteID]endpoint.AccessRoute) endpoint.RouteID {
	if _, exists := routes[localDiscoveryRouteID]; !exists {
		return localDiscoveryRouteID
	}
	for suffix := 2; ; suffix++ {
		candidate := endpoint.RouteID(string(localDiscoveryRouteID) + "-" + strconv.Itoa(suffix))
		if _, exists := routes[candidate]; !exists {
			return candidate
		}
	}
}

func localDiscoveryCredential(target endpoint.Endpoint) (string, *int) {
	var credentialRef string
	var firstPriority *int
	for _, route := range target.RouteList() {
		if route.Priority != nil && (firstPriority == nil || *route.Priority < *firstPriority) {
			value := *route.Priority
			firstPriority = &value
		}
		if credentialRef == "" && (route.Kind == endpoint.RouteDirectWebRTCTCP || route.Kind == endpoint.RouteManagedWebRTC || route.Kind == endpoint.RouteSSHWebRTCTCP) {
			credentialRef = strings.TrimSpace(route.CredentialRef)
		}
	}
	return credentialRef, firstPriority
}
