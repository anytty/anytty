package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
	"google.golang.org/protobuf/proto"
)

func main() {
	mode := flag.String("mode", "", "drop-response or verify")
	claimPath := flag.String("claim", "", "pairing claim path")
	pairingSocket := flag.String("pair-socket", "", "pairing Unix socket")
	credentialDir := flag.String("credential-dir", "", "client credential directory")
	endpointID := flag.String("endpoint-id", "", "client-local endpoint id")
	clientLabel := flag.String("client-label", "conn002-harness", "daemon access-list label")
	identityDir := flag.String("daemon-identity-dir", "", "daemon DeviceIdentity directory")
	accessDir := flag.String("daemon-access-dir", "", "daemon AccessStore directory")
	credentialRef := flag.String("credential-ref", "", "credential ref override")
	expect := flag.String("expect", "", "active or revoked")
	outputPath := flag.String("output", "", "output path for fixture mutation modes")
	flag.Parse()

	var err error
	switch strings.TrimSpace(*mode) {
	case "drop-response":
		err = dropPairingResponse(*claimPath, *pairingSocket, *credentialDir, *endpointID, *clientLabel, *credentialRef)
	case "verify":
		err = verifyCredential(*identityDir, *accessDir, *credentialDir, *endpointID, *credentialRef, *expect)
	case "tamper-fingerprint":
		err = tamperPairingIdentity(*claimPath, *outputPath)
	default:
		err = fmt.Errorf("unsupported mode %q", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func tamperPairingIdentity(claimPath, outputPath string) error {
	payload, err := os.ReadFile(strings.TrimSpace(claimPath))
	if err != nil {
		return err
	}
	offer, err := remoteauth.ParsePairingClaimOfferForExchange(payload)
	if err != nil {
		return err
	}
	offer.DevicePublicKey[0] ^= 0xff
	tampered, err := (proto.MarshalOptions{Deterministic: true}).Marshal(offer)
	if err != nil {
		return err
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.WriteFile(outputPath, tampered, 0o600); err != nil {
		return err
	}
	return os.Chmod(outputPath, 0o600)
}

func dropPairingResponse(claimPath, pairingSocket, credentialDir, endpointID, clientLabel, credentialRef string) error {
	payload, err := os.ReadFile(strings.TrimSpace(claimPath))
	if err != nil {
		return err
	}
	offer, err := remoteauth.ParsePairingClaimOfferForExchange(payload)
	if err != nil {
		return err
	}
	endpointID = strings.TrimSpace(endpointID)
	if endpointID == "" {
		return fmt.Errorf("endpoint-id is required")
	}
	if credentialRef = strings.TrimSpace(credentialRef); credentialRef == "" {
		credentialRef = pairingGrantRef(endpointID, offer.GetDeviceId())
	}
	store := remoteauth.NewCredentialStore(credentialDir)
	credential, err := store.LoadOrCreateIdentity(credentialRef, endpointID, nil)
	if err != nil {
		return err
	}
	binding, err := remoteauth.LocalUnixChannelBinding(pairingSocket)
	if err != nil {
		return err
	}
	connection, err := unixtransport.Dial(pairingSocket)
	if err != nil {
		return err
	}
	dropped := &dropAfterFirstSend{Transport: connection}
	_, err = (remoteauth.ClientPairingHandshake{}).Redeem(context.Background(), dropped, remoteauth.ClientPairingRequest{
		ExpectedDeviceID: offer.GetDeviceId(), ExpectedDeviceFingerprint: remoteauth.Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey())),
		PairingClaimOffer: payload, Identity: credential.Identity,
		ClientLabel: strings.TrimSpace(clientLabel), ChannelBinding: binding,
	})
	if err == nil {
		return fmt.Errorf("pairing response was not dropped")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"credential_ref": credentialRef, "subject_key_fingerprint": credential.Identity.Fingerprint,
	})
}

func verifyCredential(identityDir, accessDir, credentialDir, endpointID, credentialRef, expect string) error {
	identity, err := remoteauth.LoadOrCreateLocalIdentity(identityDir)
	if err != nil {
		return err
	}
	accessStore, err := remoteauth.LoadAccessSnapshot(accessDir, identity)
	if err != nil {
		return err
	}
	if credentialRef = strings.TrimSpace(credentialRef); credentialRef == "" {
		endpointID = strings.TrimSpace(endpointID)
		if endpointID == "" {
			return fmt.Errorf("endpoint-id or credential-ref is required")
		}
		credentialRef = pairingGrantRef(endpointID, identity.DeviceID)
	}
	credential, err := remoteauth.NewCredentialStore(credentialDir).Resolve(credentialRef)
	if err != nil {
		return err
	}
	claims, err := remoteauth.Verify(credential.CapabilityGrant, identity.Fingerprint, time.Now().UTC(), nil)
	if err != nil {
		return err
	}
	_, verifyErr := remoteauth.Verify(credential.CapabilityGrant, identity.Fingerprint, time.Now().UTC(), accessStore)
	switch strings.TrimSpace(expect) {
	case "active":
		if verifyErr != nil {
			return fmt.Errorf("expected active grant: %w", verifyErr)
		}
	case "revoked":
		if !errors.Is(verifyErr, remoteauth.ErrGrantRevoked) {
			return fmt.Errorf("expected revoked grant, got %v", verifyErr)
		}
	default:
		return fmt.Errorf("expect must be active or revoked")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"grant_id": claims.GrantID, "subject_key_fingerprint": credential.Identity.Fingerprint, "state": strings.TrimSpace(expect),
	})
}

func pairingGrantRef(endpointID, deviceID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(endpointID) + "\x00" + strings.TrimSpace(deviceID)))
	return "managed-" + hex.EncodeToString(digest[:12])
}

type dropAfterFirstSend struct {
	transport.Transport
	sent bool
}

func (connection *dropAfterFirstSend) Send(frame []byte) error {
	err := connection.Transport.Send(frame)
	if err == nil && !connection.sent {
		connection.sent = true
		_ = connection.Transport.Close()
	}
	return err
}
