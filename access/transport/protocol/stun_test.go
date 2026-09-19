package protocol

import "testing"

func TestEdgeSTUNURL(t *testing.T) {
	for _, test := range []struct{ endpoint, want string }{
		{"edge.example:443", "stun:edge.example:443"},
		{"[2001:db8::1]:3478", "stun:[2001:db8::1]:3478"},
		{"", ""},
		{"edge.example", ""},
	} {
		if got := EdgeSTUNURL(test.endpoint); got != test.want {
			t.Errorf("EdgeSTUNURL(%q) = %q, want %q", test.endpoint, got, test.want)
		}
	}
}
