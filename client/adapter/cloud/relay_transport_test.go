package cloud

import (
	"reflect"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
)

func TestFilterManagedICEURLsHonorsRelayTransport(t *testing.T) {
	values := []string{
		"stun:relay.example:3478",
		"turn:relay.example:3478",
		"turn:relay.example:3478?transport=udp",
		"turn:relay.example:3478?transport=tcp",
		"turns:relay.example:5349?transport=tcp",
	}
	tests := []struct {
		name       string
		preference endpoint.RelayTransport
		want       []string
	}{
		{name: "auto", preference: endpoint.RelayTransportAuto, want: values},
		{name: "udp", preference: endpoint.RelayTransportUDP, want: values[:3]},
		{name: "tcp", preference: endpoint.RelayTransportTCP, want: []string{values[0], values[3], values[4]}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := filterManagedICEURLs(values, test.preference)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filtered URLs = %v, want %v", got, test.want)
			}
		})
	}
}

func TestHasManagedTURNServerDoesNotTreatSTUNAsRelayMaterial(t *testing.T) {
	if hasManagedTURNServer([]port.ICEServer{{URLs: []string{"stun:relay.example:3478"}}}) {
		t.Fatal("STUN URL was accepted as Relay-only TURN material")
	}
	if !hasManagedTURNServer([]port.ICEServer{{URLs: []string{" TURNS:relay.example:5349 "}}}) {
		t.Fatal("TURN/TLS URL was not accepted as relay material")
	}
}

func TestFilterManagedICEURLsRejectsUnknownPolicyAndURL(t *testing.T) {
	if _, err := filterManagedICEURLs(nil, endpoint.RelayTransport("quic")); err == nil {
		t.Fatal("unknown relay transport was accepted")
	}
	if _, err := filterManagedICEURLs([]string{"https://relay.example"}, endpoint.RelayTransportUDP); err == nil {
		t.Fatal("non-ICE URL was accepted")
	}
}
