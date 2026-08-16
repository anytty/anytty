package pion

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	"github.com/anytty/anytty/proto/wire"
	remotewebrtc "github.com/anytty/anytty/remote/webrtc"
	"github.com/pion/transport/v4"
	"github.com/pion/transport/v4/stdnet"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestProductionPionAPIAdvertisesEncodedFrameLimit(t *testing.T) {
	peer, err := newPeerConnectionAPI(pionwebrtc.SettingEngine{}).NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	if _, err := peer.CreateDataChannel("protocol", nil); err != nil {
		t.Fatal(err)
	}
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("a=max-message-size:%d\r\n", wire.MaxEncodedFrameSize)
	if !strings.Contains(offer.SDP, want) {
		t.Fatalf("production Pion offer does not contain %q", want)
	}
}

func TestDefaultRouteNetUsesSocketSelectedAddresses(t *testing.T) {
	dials := make([]string, 0, 2)
	network, err := newDefaultRouteNet(func(network, address string) (net.Conn, error) {
		dials = append(dials, network+" "+address)
		switch network {
		case "udp4":
			return &routeProbeConn{local: &net.UDPAddr{IP: net.ParseIP("10.0.2.16"), Port: 49152}}, nil
		case "udp6":
			return nil, errors.New("IPv6 is unavailable")
		default:
			return nil, errors.New("unexpected probe")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(dials) != 2 {
		t.Fatalf("route probes = %v", dials)
	}
	interfaces, err := network.Interfaces()
	if err != nil || len(interfaces) != 1 {
		t.Fatalf("interfaces = %v, err = %v", interfaces, err)
	}
	addresses, err := interfaces[0].Addrs()
	if err != nil || len(addresses) != 1 || addresses[0].String() != "10.0.2.16/32" {
		t.Fatalf("interface addresses = %v, err = %v", addresses, err)
	}
	if _, err := network.InterfaceByName("default-route-v4"); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultRouteNetRejectsMissingRoute(t *testing.T) {
	if _, err := newDefaultRouteNet(func(string, string) (net.Conn, error) {
		return nil, errors.New("route unavailable")
	}); err == nil {
		t.Fatal("missing default route unexpectedly created a Pion network")
	}
}

func TestFactoryBuildsFreshNetworkSnapshotForEveryPeer(t *testing.T) {
	tests := []struct {
		name string
		open func(Factory) (port.WebRTCPeer, error)
	}{
		{name: "direct", open: func(factory Factory) (port.WebRTCPeer, error) {
			return factory.OpenDirectPeer(context.Background())
		}},
		{name: "cloud", open: func(factory Factory) (port.WebRTCPeer, error) {
			return factory.OpenCloudPeer(context.Background(), port.WebRTCConfig{})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			factory := Factory{NetworkFactory: func() (transport.Net, error) {
				calls++
				return &stdnet.Net{}, nil
			}}
			for wantCalls := 1; wantCalls <= 2; wantCalls++ {
				peer, err := test.open(factory)
				if err != nil {
					t.Fatal(err)
				}
				if err := peer.Close(); err != nil {
					t.Fatal(err)
				}
				if calls != wantCalls {
					t.Fatalf("network snapshots = %d, want %d", calls, wantCalls)
				}
			}
		})
	}
}

func TestFactoryReportsNetworkSnapshotFailureWhenPeerOpens(t *testing.T) {
	offline := errors.New("network is offline")
	tests := []struct {
		name string
		open func(Factory) (port.WebRTCPeer, error)
	}{
		{name: "direct", open: func(factory Factory) (port.WebRTCPeer, error) {
			return factory.OpenDirectPeer(context.Background())
		}},
		{name: "cloud", open: func(factory Factory) (port.WebRTCPeer, error) {
			return factory.OpenCloudPeer(context.Background(), port.WebRTCConfig{})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			factory := Factory{NetworkFactory: func() (transport.Net, error) {
				calls++
				return nil, offline
			}}
			peer, err := test.open(factory)
			if peer != nil {
				_ = peer.Close()
				t.Fatal("offline peer unexpectedly opened")
			}
			if !errors.Is(err, offline) {
				t.Fatalf("peer open error = %v, want %v", err, offline)
			}
			if calls != 1 {
				t.Fatalf("network snapshots = %d, want 1", calls)
			}
		})
	}
}

func TestFactoryUsesRouteNetworkOnlyForDiscoveredLANRoute(t *testing.T) {
	calls := make([]uint64, 0, 1)
	factory := Factory{
		NetworkFactory: func() (transport.Net, error) { return &stdnet.Net{}, nil },
		RouteNetworkFactory: func(handle uint64) (transport.Net, error) {
			calls = append(calls, handle)
			return &stdnet.Net{}, nil
		},
	}
	peer, err := factory.OpenDirectPeerForRoute(context.Background(), endpoint.AccessRoute{NetworkHandle: 42})
	if err != nil {
		t.Fatal(err)
	}
	_ = peer.Close()
	peer, err = factory.OpenDirectPeerForRoute(context.Background(), endpoint.AccessRoute{})
	if err != nil {
		t.Fatal(err)
	}
	_ = peer.Close()
	if len(calls) != 1 || calls[0] != 42 {
		t.Fatalf("route network calls = %v", calls)
	}
}

func TestDiscoveredLANRoutePublishesPassiveICETCPCandidate(t *testing.T) {
	routeInterface := transport.NewInterface(net.Interface{
		Index: 1,
		Name:  "test-route",
		MTU:   1500,
		Flags: net.FlagUp | net.FlagLoopback,
	})
	routeInterface.AddAddress(&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(32, 32)})
	routeNetwork := &defaultRouteNet{Net: &stdnet.Net{}, interfaces: []*transport.Interface{routeInterface}}
	factory := Factory{RouteNetworkFactory: func(handle uint64) (transport.Net, error) {
		if handle != 42 {
			t.Fatalf("route handle = %d, want 42", handle)
		}
		return routeNetwork, nil
	}}

	opened, err := factory.OpenDirectPeerForRoute(context.Background(), endpoint.AccessRoute{NetworkHandle: 42})
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	offer, err := opened.CreateOffer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(offer, " tcptype passive") {
		t.Fatalf("discovered LAN offer has no passive ICE-TCP candidate:\n%s", offer)
	}
}

func TestCloudOfferWaitsForRelayOnlyWhenTURNIsConfigured(t *testing.T) {
	if hasTURNServer([]port.ICEServer{{URLs: []string{"stun:relay.example:3478"}}}) {
		t.Fatal("STUN-only config waits for an impossible relay candidate")
	}
	if !hasTURNServer([]port.ICEServer{{URLs: []string{" turns:relay.example:5349 "}}}) {
		t.Fatal("TURN config did not request a relay candidate")
	}
	if got := cloudOfferGatheringGrace(true); got != remotewebrtc.ICEGatheringCloudGrace {
		t.Fatalf("managed ICE offer grace = %v", got)
	}
}

func TestCloudPeerGatheringWaitsBrieflyForRelayCandidateInAutomaticMode(t *testing.T) {
	factory := Factory{PeerConnections: pionwebrtc.NewPeerConnection}
	tests := []struct {
		name              string
		config            port.WebRTCConfig
		wantPreferRelay   bool
		wantGatherTimeout time.Duration
		wantMode          peerDiagnosticMode
	}{
		{name: "direct only", config: port.WebRTCConfig{}, wantGatherTimeout: cloudGatherTimeout, wantMode: peerDiagnosticCloudDirect},
		{name: "automatic", config: port.WebRTCConfig{Servers: []port.ICEServer{{URLs: []string{"turn:127.0.0.1:3478"}, Username: "user", Credential: "credential"}}}, wantPreferRelay: true, wantGatherTimeout: autoGatherTimeout, wantMode: peerDiagnosticCloudAuto},
		{name: "relay only", config: port.WebRTCConfig{Servers: []port.ICEServer{{URLs: []string{"turn:127.0.0.1:3478"}, Username: "user", Credential: "credential"}}, Policy: port.ICETransportRelayOnly}, wantPreferRelay: true, wantGatherTimeout: cloudGatherTimeout, wantMode: peerDiagnosticCloudRelay},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opened, err := factory.OpenCloudPeer(context.Background(), test.config)
			if err != nil {
				t.Fatal(err)
			}
			defer opened.Close()
			peer := opened.(*webRTCPeer)
			if peer.waitForRelayCandidate != test.wantPreferRelay || peer.gatherTimeout != test.wantGatherTimeout || peer.diagnosticMode != test.wantMode {
				t.Fatalf("prefer relay = %t, gather timeout = %s, mode = %s", peer.waitForRelayCandidate, peer.gatherTimeout, peer.diagnosticMode)
			}
		})
	}
}

func TestWaitReadyReturnsWhenChannelClosesBeforeOpen(t *testing.T) {
	peer := &webRTCPeer{ready: make(chan struct{}), channelClosed: make(chan struct{}), readyTimeout: time.Second}
	close(peer.channelClosed)
	if err := peer.WaitReady(context.Background()); err == nil {
		t.Fatal("closed DataChannel was reported ready")
	}
}

func TestCandidatePortRejectsInvalidStatsValues(t *testing.T) {
	if got := candidatePort(41121); got != 41121 {
		t.Fatalf("candidatePort(41121) = %d", got)
	}
	for _, value := range []int32{-1, 0, 65536} {
		if got := candidatePort(value); got != 0 {
			t.Fatalf("candidatePort(%d) = %d, want 0", value, got)
		}
	}
}

func TestSelectedCandidatePairUsesTransportSelectionInsteadOfNominatedReportOrder(t *testing.T) {
	selected := pionwebrtc.ICECandidatePairStats{
		ID: "z-selected", LocalCandidateID: "selected-local", RemoteCandidateID: "selected-remote",
		State: pionwebrtc.StatsICECandidatePairStateSucceeded, Nominated: true,
	}
	report := pionwebrtc.StatsReport{
		"a-stale": pionwebrtc.ICECandidatePairStats{
			ID: "a-stale", LocalCandidateID: "stale-local", RemoteCandidateID: "stale-remote",
			State: pionwebrtc.StatsICECandidatePairStateSucceeded, Nominated: true,
		},
		"stale-local":    pionwebrtc.ICECandidateStats{ID: "stale-local", IP: "203.0.113.10", CandidateType: pionwebrtc.ICECandidateTypeRelay},
		"stale-remote":   pionwebrtc.ICECandidateStats{ID: "stale-remote", IP: "203.0.113.11", CandidateType: pionwebrtc.ICECandidateTypeRelay},
		"selected-local": pionwebrtc.ICECandidateStats{ID: "selected-local", IP: "192.0.2.10", CandidateType: pionwebrtc.ICECandidateTypeHost},
		"selected-remote": pionwebrtc.ICECandidateStats{
			ID: "selected-remote", IP: "192.0.2.11", CandidateType: pionwebrtc.ICECandidateTypeHost,
		},
	}

	pair, local, remote, ok := selectedCandidatePair(staticSelectedPairStats{pair: selected, ok: true}, report)
	if !ok {
		t.Fatal("selected candidate pair was unavailable")
	}
	if pair.ID != selected.ID || local.ID != selected.LocalCandidateID || remote.ID != selected.RemoteCandidateID {
		t.Fatalf("selected pair = %#v, local = %#v, remote = %#v", pair, local, remote)
	}
	if got := candidatePath(local, remote); got != endpoint.PathDirect {
		t.Fatalf("selected path = %q, want %q", got, endpoint.PathDirect)
	}
}

func TestSelectedCandidatePairRequiresSelectedStatsAndCandidates(t *testing.T) {
	report := pionwebrtc.StatsReport{
		"local": pionwebrtc.ICECandidateStats{ID: "local"},
	}
	selected := pionwebrtc.ICECandidatePairStats{LocalCandidateID: "local", RemoteCandidateID: "remote"}
	tests := []struct {
		name   string
		reader selectedCandidatePairStatsReader
		report pionwebrtc.StatsReport
	}{
		{name: "no reader", report: report},
		{name: "no selected pair", reader: staticSelectedPairStats{}},
		{name: "missing remote candidate", reader: staticSelectedPairStats{pair: selected, ok: true}, report: report},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, ok := selectedCandidatePair(test.reader, test.report); ok {
				t.Fatal("incomplete selected candidate pair unexpectedly succeeded")
			}
		})
	}
}

func TestRelatedCandidateAddressesStayBoundToSelectedStats(t *testing.T) {
	local := &pionwebrtc.ICECandidate{Address: "182.138.142.220", Port: 54229, Typ: pionwebrtc.ICECandidateTypeSrflx, RelatedAddress: "192.168.123.168", RelatedPort: 37129}
	remote := &pionwebrtc.ICECandidate{Address: "192.168.123.203", Port: 53537, Typ: pionwebrtc.ICECandidateTypeHost}
	pair := pionwebrtc.NewICECandidatePair(local, remote)
	localStats := pionwebrtc.ICECandidateStats{IP: local.Address, Port: int32(local.Port), CandidateType: local.Typ}
	remoteStats := pionwebrtc.ICECandidateStats{IP: remote.Address, Port: int32(remote.Port), CandidateType: remote.Typ}

	localAddress, localPort, remoteAddress, remotePort := relatedCandidateAddresses(pair, localStats, remoteStats)
	if localAddress != "192.168.123.168" || localPort != 37129 || remoteAddress != "" || remotePort != 0 {
		t.Fatalf("related addresses = %s:%d / %s:%d", localAddress, localPort, remoteAddress, remotePort)
	}

	localStats.IP = "203.0.113.99"
	localAddress, localPort, remoteAddress, remotePort = relatedCandidateAddresses(pair, localStats, remoteStats)
	if localAddress != "" || localPort != 0 || remoteAddress != "" || remotePort != 0 {
		t.Fatalf("mixed-pair related addresses = %s:%d / %s:%d", localAddress, localPort, remoteAddress, remotePort)
	}
}

type staticSelectedPairStats struct {
	pair pionwebrtc.ICECandidatePairStats
	ok   bool
}

func (stats staticSelectedPairStats) GetSelectedCandidatePairStats() (pionwebrtc.ICECandidatePairStats, bool) {
	return stats.pair, stats.ok
}

func TestWaitReadyReturnsPeerFailureAndTimeout(t *testing.T) {
	connectionFailed := make(chan error, 1)
	connectionFailed <- errors.New("ICE failed")
	peer := &webRTCPeer{ready: make(chan struct{}), channelClosed: make(chan struct{}), connectionFailed: connectionFailed, readyTimeout: time.Second}
	if err := peer.WaitReady(context.Background()); err == nil || err.Error() != "ICE failed" {
		t.Fatalf("peer failure = %v", err)
	}

	peer = &webRTCPeer{ready: make(chan struct{}), channelClosed: make(chan struct{}), connectionFailed: make(chan error), readyTimeout: time.Millisecond}
	if err := peer.WaitReady(context.Background()); err == nil {
		t.Fatal("WebRTC ready timeout unexpectedly succeeded")
	}
}

func TestPeerFailureAfterReadyClosesProtocolChannel(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)
	peer.handleConnectionState(pionwebrtc.PeerConnectionStateFailed)

	select {
	case <-channel.closed:
	case <-time.After(time.Second):
		t.Fatal("final peer failure left the ready protocol channel half-open")
	}
	select {
	case err := <-peer.connectionFailed:
		if err == nil || err.Error() != "WebRTC peer failed" {
			t.Fatalf("peer failure = %v", err)
		}
	default:
		t.Fatal("final peer failure did not notify the pre-ready waiter")
	}
	peer.closeProtocolChannel()
	if channel.closeCalls != 1 {
		t.Fatalf("protocol channel close calls = %d, want 1", channel.closeCalls)
	}
}

func TestPeerDisconnectedKeepsProtocolChannelRecoverable(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)

	peer.handleConnectionState(pionwebrtc.PeerConnectionStateDisconnected)

	if channel.closeCalls != 0 {
		t.Fatalf("recoverable disconnect closed protocol channel %d times", channel.closeCalls)
	}
	select {
	case err := <-peer.connectionFailed:
		t.Fatalf("recoverable disconnect reported final failure: %v", err)
	default:
	}
	peer.handleConnectionState(pionwebrtc.PeerConnectionStateConnected)
}

func TestPeerDisconnectedClosesProtocolChannelAfterGrace(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
		disconnectGrace: 10 * time.Millisecond,
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)

	peer.handleConnectionState(pionwebrtc.PeerConnectionStateDisconnected)

	select {
	case <-channel.closed:
	case <-time.After(time.Second):
		t.Fatal("expired disconnected grace left the protocol channel half-open")
	}
	select {
	case err := <-peer.connectionFailed:
		if err == nil || !strings.Contains(err.Error(), "remained disconnected") {
			t.Fatalf("disconnect failure = %v", err)
		}
	default:
		t.Fatal("expired disconnected grace did not report failure")
	}
}

func TestPeerConnectingDoesNotCancelDisconnectedGrace(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
		disconnectGrace: 10 * time.Millisecond,
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)

	peer.handleConnectionState(pionwebrtc.PeerConnectionStateDisconnected)
	peer.handleConnectionState(pionwebrtc.PeerConnectionStateConnecting)

	select {
	case <-channel.closed:
	case <-time.After(time.Second):
		t.Fatal("peer stuck connecting outlived its disconnected grace")
	}
}

func TestPeerConnectedCancelsDisconnectedGrace(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
		disconnectGrace: 10 * time.Millisecond,
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)

	peer.handleConnectionState(pionwebrtc.PeerConnectionStateDisconnected)
	peer.handleConnectionState(pionwebrtc.PeerConnectionStateConnected)
	time.Sleep(30 * time.Millisecond)

	if channel.closeCalls != 0 {
		t.Fatalf("recovered peer closed protocol channel %d times", channel.closeCalls)
	}
	select {
	case err := <-peer.connectionFailed:
		t.Fatalf("recovered peer reported failure: %v", err)
	default:
	}
}

func TestPeerStaleConnectedCallbackDoesNotCancelCurrentDisconnectGrace(t *testing.T) {
	channel := &lifecycleChannel{closed: make(chan struct{})}
	peer := &webRTCPeer{
		channel: channel, ready: make(chan struct{}), channelClosed: channel.closed,
		connectionFailed: make(chan error, 1), readyTimeout: time.Second,
		disconnectGrace: 10 * time.Millisecond,
		connectionState: func() pionwebrtc.PeerConnectionState {
			return pionwebrtc.PeerConnectionStateDisconnected
		},
	}
	channel.SetCloseHandler(func() {
		peer.channelClosedOnce.Do(func() { close(peer.channelClosed) })
	})
	close(peer.ready)

	peer.handleConnectionState(pionwebrtc.PeerConnectionStateDisconnected)
	peer.handleConnectionState(pionwebrtc.PeerConnectionStateConnected)

	select {
	case <-channel.closed:
	case <-time.After(time.Second):
		t.Fatal("stale connected callback cancelled the current disconnected grace")
	}
}

type lifecycleChannel struct {
	mu         sync.Mutex
	closed     chan struct{}
	closeOnce  sync.Once
	closeCalls int
	onClose    func()
}

type routeProbeConn struct{ local net.Addr }

func (*routeProbeConn) Read([]byte) (int, error)          { return 0, errors.New("unused") }
func (*routeProbeConn) Write(payload []byte) (int, error) { return len(payload), nil }
func (*routeProbeConn) Close() error                      { return nil }
func (connection *routeProbeConn) LocalAddr() net.Addr    { return connection.local }
func (*routeProbeConn) RemoteAddr() net.Addr              { return &net.UDPAddr{} }
func (*routeProbeConn) SetDeadline(time.Time) error       { return nil }
func (*routeProbeConn) SetReadDeadline(time.Time) error   { return nil }
func (*routeProbeConn) SetWriteDeadline(time.Time) error  { return nil }

func (*lifecycleChannel) SetMessageHandler(func([]byte)) {}
func (channel *lifecycleChannel) SetCloseHandler(handler func()) {
	channel.onClose = handler
}
func (*lifecycleChannel) BufferedAmount() uint64               { return 0 }
func (*lifecycleChannel) SetBufferedAmountLowThreshold(uint64) {}
func (*lifecycleChannel) SetBufferedAmountLowHandler(func())   {}
func (*lifecycleChannel) Send([]byte) error                    { return nil }
func (channel *lifecycleChannel) Close() error {
	channel.closeOnce.Do(func() {
		channel.mu.Lock()
		channel.closeCalls++
		channel.mu.Unlock()
		if channel.onClose != nil {
			channel.onClose()
		} else {
			close(channel.closed)
		}
	})
	return nil
}
