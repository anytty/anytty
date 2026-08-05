package securetransport

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestValidateServerPairRejectsMalformedTrailingCertificate(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	certificatePEM, privateKeyPEM := testServerPair(t, now, "edge.example.com")
	certificatePEM = append(certificatePEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("malformed trailing certificate")})...)
	if _, err := ValidateServerPair(certificatePEM, privateKeyPEM, "edge.example.com:443", now); err == nil {
		t.Fatal("server pair with malformed trailing certificate was accepted")
	}
}

func TestValidateServerPairAcceptsMatchingIPAddress(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	certificatePEM, privateKeyPEM := testIPServerPair(t, now, "155.94.155.192")
	if _, err := ValidateServerPair(certificatePEM, privateKeyPEM, "155.94.155.192:41102", now); err != nil {
		t.Fatalf("matching IP certificate rejected: %v", err)
	}
}

func TestEdgeIDFromCertificateRequiresCanonicalIdentityURI(t *testing.T) {
	identity, err := EdgeIdentityURI("edge-1")
	if err != nil {
		t.Fatalf("create Edge identity URI: %v", err)
	}
	edgeID, err := EdgeIDFromCertificate(&x509.Certificate{URIs: []*url.URL{identity}})
	if err != nil {
		t.Fatalf("parse canonical Edge identity URI: %v", err)
	}
	if edgeID != "edge-1" {
		t.Fatalf("Edge ID = %q, want edge-1", edgeID)
	}

	queryIdentity := *identity
	queryIdentity.RawQuery = "alternate=true"
	if _, err := EdgeIDFromCertificate(&x509.Certificate{URIs: []*url.URL{&queryIdentity}}); err == nil {
		t.Fatal("Edge identity URI with query was accepted")
	}
	userIdentity := *identity
	userIdentity.User = url.User("unexpected")
	if _, err := EdgeIDFromCertificate(&x509.Certificate{URIs: []*url.URL{&userIdentity}}); err == nil {
		t.Fatal("Edge identity URI with userinfo was accepted")
	}
}

func TestEdgeIDFromCertificateRejectsMissingAndDuplicateIdentity(t *testing.T) {
	if _, err := EdgeIDFromCertificate(&x509.Certificate{}); err == nil {
		t.Fatal("certificate without Edge identity URI was accepted")
	}
	identity, err := EdgeIdentityURI("edge-1")
	if err != nil {
		t.Fatalf("create Edge identity URI: %v", err)
	}
	if _, err := EdgeIDFromCertificate(&x509.Certificate{URIs: []*url.URL{identity, identity}}); err == nil {
		t.Fatal("certificate with duplicate Edge identity URIs was accepted")
	}
}

func TestPinnedEdgeTLSRequiresMatchingServerSentCA(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	root, leaf, rootPEM := testPinnedEdgeCertificates(t, now, "edge.example.com")
	pin, err := EdgeCACertificateDERFingerprint(rootPEM)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(root.Raw)
	if !bytes.Equal(pin, want[:]) {
		t.Fatalf("CA pin = %x, want %x", pin, want)
	}
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf, root}}
	if err := verifyPinnedEdgeConnection(state, "edge.example.com", pin, now); err != nil {
		t.Fatalf("matching pinned chain rejected: %v", err)
	}
	wrongPin := append([]byte(nil), pin...)
	wrongPin[0] ^= 0xff
	if err := verifyPinnedEdgeConnection(state, "edge.example.com", wrongPin, now); err == nil {
		t.Fatal("wrong CA pin was accepted")
	}
	if err := verifyPinnedEdgeConnection(tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}, "edge.example.com", pin, now); err == nil {
		t.Fatal("chain without the pinned CA was accepted")
	}
	if err := verifyPinnedEdgeConnection(state, "other.example.com", pin, now); err == nil {
		t.Fatal("pinned chain accepted another server name")
	}
}

func TestEdgeCAPinRejectsAmbiguousPEM(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	_, _, rootPEM := testPinnedEdgeCertificates(t, now, "edge.example.com")
	if _, err := EdgeCACertificateDERFingerprint(append(append([]byte(nil), rootPEM...), rootPEM...)); err == nil {
		t.Fatal("multiple CA certificates produced an ambiguous pin")
	}
}

func testServerPair(t *testing.T, now time.Time, dnsName string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: dnsName}, DNSNames: []string{dnsName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
}

func testIPServerPair(t *testing.T, now time.Time, address string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: address}, IPAddresses: []net.IP{net.ParseIP(address)},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
}

func testPinnedEdgeCertificates(t *testing.T, now time.Time, dnsName string) (*x509.Certificate, *x509.Certificate, []byte) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(10), Subject: pkix.Name{CommonName: "AnyTTY Edge CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(11), Subject: pkix.Name{CommonName: dnsName}, DNSNames: []string{dnsName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	return root, leaf, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root.Raw})
}
