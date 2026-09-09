package netpath

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pion/stun/v3"
	webrtc "github.com/pion/webrtc/v4"
)

func TestSTUNGathersEverySourceNetwork(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go func() {
		buffer := make([]byte, 2048)
		for {
			n, source, err := server.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			request := &stun.Message{Raw: append([]byte(nil), buffer[:n]...)}
			if request.Decode() != nil {
				continue
			}
			response, err := stun.Build(stun.NewTransactionIDSetter(request.TransactionID), stun.BindingSuccess, &stun.XORMappedAddress{IP: source.IP, Port: source.Port})
			if err == nil {
				_, _ = server.WriteToUDP(response.Raw, source)
			}
		}
	}()
	provider.RLock()
	previous := provider.get
	provider.RUnlock()
	defer SetProvider(previous)
	SetProvider(func() []Path {
		return []Path{
			{ID: "a", Name: "loop-a", Addresses: []net.IPNet{{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}}},
			{ID: "b", Name: "loop-b", Addresses: []net.IPNet{{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}}},
		}
	})
	network, err := NewICENetwork()
	if err != nil {
		t.Fatal(err)
	}
	settings := webrtc.SettingEngine{}
	settings.SetNet(network)
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	peer, err := webrtc.NewAPI(webrtc.WithSettingEngine(settings)).NewPeerConnection(webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:" + server.LocalAddr().String()}}}})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	var mu sync.Mutex
	sources := map[string]bool{}
	done := make(chan struct{})
	peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			close(done)
			return
		}
		if candidate.Typ == webrtc.ICECandidateTypeSrflx {
			mu.Lock()
			sources[fmt.Sprintf("%s:%d", candidate.Address, candidate.Port)] = true
			mu.Unlock()
		}
	})
	if _, err := peer.CreateDataChannel("test", nil); err != nil {
		t.Fatal(err)
	}
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("gathering timed out")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sources) != 2 {
		t.Fatalf("STUN sources = %v, want sockets from both source networks", sources)
	}
}
