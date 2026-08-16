package pion

import (
	"fmt"
	"net"
	"sort"

	pionice "github.com/pion/ice/v4"
	"github.com/pion/logging"
	"github.com/pion/transport/v4"
)

// newRouteICETCPMux publishes passive candidates on the exact platform network
// that discovered a LAN daemon. This complements Pion's active ICE-TCP path,
// whose internal dialer cannot inherit Android Network.bindSocket semantics.
func newRouteICETCPMux(network transport.Net, logger logging.LeveledLogger) (pionice.TCPMux, error) {
	interfaces, err := network.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate route interfaces: %w", err)
	}

	type routeAddress struct {
		network string
		address *net.TCPAddr
	}
	addresses := make(map[string]routeAddress)
	for _, iface := range interfaces {
		if iface == nil {
			continue
		}
		addrs, addrsErr := iface.Addrs()
		if addrsErr != nil {
			continue
		}
		for _, addr := range addrs {
			ip := addressIP(addr)
			if ip == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
				continue
			}
			candidate := routeAddress{network: "tcp6", address: &net.TCPAddr{IP: ip, Port: 0}}
			if ipv4 := ip.To4(); ipv4 != nil {
				candidate.network = "tcp4"
				candidate.address.IP = ipv4
			}
			addresses[candidate.address.IP.String()] = candidate
		}
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("route network has no listenable IP address")
	}

	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	muxes := make([]pionice.TCPMux, 0, len(keys))
	var listenErr error
	for _, key := range keys {
		candidate := addresses[key]
		listener, err := network.ListenTCP(candidate.network, candidate.address)
		if err != nil {
			listenErr = err
			continue
		}
		muxes = append(muxes, pionice.NewTCPMuxDefault(pionice.TCPMuxParams{
			Listener: listener,
			Logger:   logger,
		}))
	}
	if len(muxes) == 0 {
		return nil, fmt.Errorf("listen on route network: %w", listenErr)
	}
	if len(muxes) == 1 {
		return muxes[0], nil
	}
	return pionice.NewMultiTCPMuxDefault(muxes...), nil
}

func addressIP(address net.Addr) net.IP {
	switch value := address.(type) {
	case *net.IPNet:
		return append(net.IP(nil), value.IP...)
	case *net.IPAddr:
		return append(net.IP(nil), value.IP...)
	default:
		return nil
	}
}
