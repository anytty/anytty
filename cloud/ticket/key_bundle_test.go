package ticket_test

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestKeyBundleValidationAndTTLBoundaries(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	valid := testKeyBundle(now, ticket.MaxKeyBundleTTL)
	if _, err := ticket.ValidateKeyBundle(valid); err != nil {
		t.Fatal(err)
	}
	if ticket.KeyBundleUsableAt(valid, now.Add(-time.Nanosecond)) || !ticket.KeyBundleUsableAt(valid, now) || !ticket.KeyBundleUsableAt(valid, valid.GetExpiresAt().AsTime().Add(-time.Nanosecond)) || ticket.KeyBundleUsableAt(valid, valid.GetExpiresAt().AsTime()) {
		t.Fatal("KeyBundle effective/expiry boundary is incorrect")
	}
	tests := map[string]func(*cloudv1.KeyBundle){
		"zero revision":  func(bundle *cloudv1.KeyBundle) { bundle.Revision = 0 },
		"missing issued": func(bundle *cloudv1.KeyBundle) { bundle.IssuedAt = nil },
		"invalid expiry": func(bundle *cloudv1.KeyBundle) { bundle.ExpiresAt = &timestamppb.Timestamp{Seconds: 253402300800} },
		"empty window":   func(bundle *cloudv1.KeyBundle) { bundle.ExpiresAt = bundle.IssuedAt },
		"over 24 hours": func(bundle *cloudv1.KeyBundle) {
			bundle.ExpiresAt = timestamppb.New(now.Add(ticket.MaxKeyBundleTTL + time.Nanosecond))
		},
		"empty keys":        func(bundle *cloudv1.KeyBundle) { bundle.Keys = nil },
		"noncanonical ID":   func(bundle *cloudv1.KeyBundle) { bundle.Keys[0].KeyId = " key-a" },
		"invalid ID":        func(bundle *cloudv1.KeyBundle) { bundle.Keys[0].KeyId = "key/a" },
		"unknown algorithm": func(bundle *cloudv1.KeyBundle) { bundle.Keys[0].Algorithm = "ed25519" },
		"short public key":  func(bundle *cloudv1.KeyBundle) { bundle.Keys[0].PublicKey = make([]byte, ed25519.PublicKeySize-1) },
		"duplicate ID": func(bundle *cloudv1.KeyBundle) {
			bundle.Keys = append(bundle.Keys, proto.Clone(bundle.Keys[0]).(*cloudv1.VerificationKey))
		},
		"duplicate public key": func(bundle *cloudv1.KeyBundle) {
			bundle.Keys = append(bundle.Keys, &cloudv1.VerificationKey{KeyId: "key-b", Algorithm: "Ed25519", PublicKey: append([]byte(nil), bundle.Keys[0].GetPublicKey()...)})
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := proto.Clone(valid).(*cloudv1.KeyBundle)
			mutate(bundle)
			if _, err := ticket.ValidateKeyBundle(bundle); err == nil {
				t.Fatal("invalid KeyBundle was accepted")
			}
		})
	}
}

func TestCanonicalKeyBundleSortsWithoutMutatingInput(t *testing.T) {
	now := time.Now().UTC()
	bundle := testKeyBundle(now, time.Hour)
	bundle.Keys = []*cloudv1.VerificationKey{
		{KeyId: "key-z", Algorithm: "Ed25519", PublicKey: make([]byte, ed25519.PublicKeySize)},
		{KeyId: "key-a", Algorithm: "Ed25519", PublicKey: append(make([]byte, ed25519.PublicKeySize-1), 1)},
	}
	canonical, _, err := ticket.CanonicalKeyBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.GetKeys()[0].GetKeyId() != "key-a" || bundle.GetKeys()[0].GetKeyId() != "key-z" {
		t.Fatal("canonicalization did not sort a defensive copy")
	}
}

func testKeyBundle(now time.Time, ttl time.Duration) *cloudv1.KeyBundle {
	return &cloudv1.KeyBundle{
		Revision: 1, IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(ttl)),
		Keys: []*cloudv1.VerificationKey{{KeyId: "key-a", Algorithm: "Ed25519", PublicKey: make([]byte, ed25519.PublicKeySize)}},
	}
}
