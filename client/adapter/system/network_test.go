package system

import (
	"net"
	"testing"
)

func TestPrivateLANIPv4AddressesAreCanonicalAndSorted(t *testing.T) {
	addresses, err := PrivateLANIPv4Addresses()
	if err != nil {
		t.Fatal(err)
	}
	for index, address := range addresses {
		ip := net.ParseIP(address)
		if ip == nil || ip.To4() == nil || !ip.IsPrivate() || ip.String() != address {
			t.Fatalf("LAN address[%d]=%q is not canonical private IPv4", index, address)
		}
		if index > 0 && addresses[index-1] >= address {
			t.Fatalf("LAN addresses are not strictly sorted: %#v", addresses)
		}
	}
}
