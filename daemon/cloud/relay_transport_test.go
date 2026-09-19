package daemon

import (
	"reflect"
	"testing"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestRequiresDaemonRelayCandidate(t *testing.T) {
	for _, transport := range []cloudv1.RelayTransport{
		cloudv1.RelayTransport_RELAY_TRANSPORT_UNSPECIFIED,
		cloudv1.RelayTransport_RELAY_TRANSPORT_UDP,
		cloudv1.RelayTransport_RELAY_TRANSPORT_TCP,
		cloudv1.RelayTransport_RELAY_TRANSPORT_TLS,
	} {
		for _, hasRelay := range []bool{false, true} {
			offer := &cloudv1.AgentOffer{RelayTransport: transport}
			if hasRelay {
				offer.Relay = &cloudv1.RelayICEConfig{}
			}
			want := hasRelay && (transport == cloudv1.RelayTransport_RELAY_TRANSPORT_TCP || transport == cloudv1.RelayTransport_RELAY_TRANSPORT_TLS)
			if got := requiresDaemonRelayCandidate(offer); got != want {
				t.Fatalf("transport=%s relay=%t: got %t, want %t", transport, hasRelay, got, want)
			}
		}
	}
	if requiresDaemonRelayCandidate(nil) {
		t.Fatal("nil offer required relay")
	}
}

func TestFilterDaemonRelayICEURLs(t *testing.T) {
	values := []string{
		"stun:relay.example:3478",
		"turn:relay.example:3478",
		"turn:relay.example:3478?transport=tcp",
		"turns:relay.example:5349",
	}
	tests := []struct {
		name       string
		preference cloudv1.RelayTransport
		want       []string
	}{
		{name: "unspecified", preference: cloudv1.RelayTransport_RELAY_TRANSPORT_UNSPECIFIED, want: values},
		{name: "udp", preference: cloudv1.RelayTransport_RELAY_TRANSPORT_UDP, want: values[:2]},
		{name: "tcp", preference: cloudv1.RelayTransport_RELAY_TRANSPORT_TCP, want: []string{values[0], values[2]}},
		{name: "tls", preference: cloudv1.RelayTransport_RELAY_TRANSPORT_TLS, want: []string{values[0], values[3]}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := filterDaemonRelayICEURLs(values, test.preference)
			if err != nil {
				t.Fatalf("filterDaemonRelayICEURLs() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("filterDaemonRelayICEURLs() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestFilterDaemonRelayICEURLsRejectsInvalidOrMissingTransport(t *testing.T) {
	if _, err := filterDaemonRelayICEURLs([]string{"turn:relay.example:3478"}, cloudv1.RelayTransport(99)); err == nil {
		t.Fatal("invalid Relay transport was accepted")
	}
	if urls, err := filterDaemonRelayICEURLs([]string{"stun:relay.example:3478", "turn:relay.example:3478?transport=tcp"}, cloudv1.RelayTransport_RELAY_TRANSPORT_UDP); err != nil {
		t.Fatalf("filterDaemonRelayICEURLs() error = %v", err)
	} else if hasDaemonTURNICEURL(urls) {
		t.Fatalf("UDP filter retained an incompatible TURN URL: %#v", urls)
	}
}
