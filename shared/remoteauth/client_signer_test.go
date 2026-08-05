package remoteauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestClientProofAcceptsNonExportableSignerProjection(t *testing.T) {
	identity, err := GenerateClientAccessIdentity("web-endpoint", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewPrivateClientAccessSigner(identity)
	if err != nil {
		t.Fatal(err)
	}
	publicOnly := identity
	publicOnly.PrivateKey = nil
	binding, err := DTLSChannelBinding("sha-256:" + strings.TrimSuffix(strings.Repeat("aa:", 32), ":"))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := signClientProof(
		context.Background(), publicOnly, signer, remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY,
		[]byte("grant"), "auth-session", bytes.Repeat([]byte{1}, authNonceBytes), bytes.Repeat([]byte{2}, authNonceBytes), binding,
	)
	if err != nil || len(proof) == 0 {
		t.Fatalf("non-exportable signer proof = %x err=%v", proof, err)
	}
	other, err := GenerateClientAccessIdentity("web-endpoint", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	wrongSigner, err := NewPrivateClientAccessSigner(other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signClientProof(
		context.Background(), publicOnly, wrongSigner, remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY,
		[]byte("grant"), "auth-session", bytes.Repeat([]byte{1}, authNonceBytes), bytes.Repeat([]byte{2}, authNonceBytes), binding,
	); HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH {
		t.Fatalf("wrong signer error = %v", err)
	}
}
