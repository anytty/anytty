package remoteauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestDeviceIdentityProofBindsChallengeAndIdentity(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := NewIdentity("device-proof", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	challenge := make([]byte, DeviceIdentityChallengeBytes)
	if _, err := rand.Read(challenge); err != nil {
		t.Fatal(err)
	}
	proof, err := SignDeviceIdentityProof(identity, challenge)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDeviceIdentityProof(challenge, identity.DeviceID, identity.Fingerprint, identity.PublicKey, proof); err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), challenge...)
	changed[0] ^= 0xff
	if err := VerifyDeviceIdentityProof(changed, identity.DeviceID, identity.Fingerprint, identity.PublicKey, proof); err == nil {
		t.Fatal("proof must reject a different challenge")
	}
}
