package state

import "testing"

func TestHostCapabilityStoreAppliesKeyboardProbe(t *testing.T) {
	store := HostCapabilityStore{}.ApplyUpdate(HostCapabilityUpdate{KeyboardDisambiguation: true})
	if !store.KeyboardProbed || !store.KeyboardDisambiguation {
		t.Fatalf("unexpected host capability store %#v", store)
	}
}
