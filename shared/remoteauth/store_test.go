package remoteauth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/securefs"
)

func TestCredentialStoreKeepsEndpointIdentityStableAcrossLostResponseRecovery(t *testing.T) {
	dir := t.TempDir()
	store := NewCredentialStore(dir)
	first, err := store.LoadOrCreateIdentity("lab-access", "endpoint-lab", bytes.NewReader(bytes.Repeat([]byte{0x31}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.LoadOrCreateIdentity("lab-access", "endpoint-lab", bytes.NewReader(bytes.Repeat([]byte{0x42}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity.Fingerprint != second.Identity.Fingerprint || first.Ready() || second.Ready() {
		t.Fatalf("pending credential identity changed: first=%#v second=%#v", first, second)
	}
	if _, err := store.LoadOrCreateIdentity("lab-access", "endpoint-other", rand.Reader); err == nil {
		t.Fatal("credential ref was rebound to another endpoint")
	}
	path := filepath.Join(dir, credentialFileName("lab-access"))
	info, err := os.Stat(path)
	if err != nil || !securefs.IsPrivateFile(path, info) {
		t.Fatalf("credential permissions: info=%v err=%v", info, err)
	}
}

func TestCredentialStoreBindsOnlyGrantForStoredSubject(t *testing.T) {
	store := NewCredentialStore(t.TempDir())
	credential, err := store.LoadOrCreateIdentity("lab-access", "endpoint-lab", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	daemonPublic, daemonPrivate, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	issue := func(subject string) string {
		grant, issueErr := Issue(daemonPrivate, Claims{
			GrantID: "grant-1", IssuerDeviceID: "device-1", SubjectKeyFingerprint: subject,
			Scope: Scope{AllowDaemon: true}, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
		})
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return grant
	}
	other, _ := GenerateClientAccessIdentity("endpoint-lab", rand.Reader)
	if _, err := store.BindGrant("lab-access", issue(other.Fingerprint), Fingerprint(daemonPublic), now, BindGrantOptions{}); !errors.Is(err, ErrGrantSubjectMismatch) {
		t.Fatalf("wrong subject bind error = %v", err)
	}
	bound, err := store.BindGrant("lab-access", issue(credential.Identity.Fingerprint), Fingerprint(daemonPublic), now, BindGrantOptions{})
	if err != nil || !bound.Ready() {
		t.Fatalf("BindGrant: credential=%#v err=%v", bound, err)
	}
	resolved, err := store.Resolve("lab-access")
	if err != nil || resolved.Identity.Fingerprint != credential.Identity.Fingerprint || !resolved.Ready() {
		t.Fatalf("Resolve: credential=%#v err=%v", resolved, err)
	}
}

func TestCredentialStoreSerializesCrossInstanceIdentityCreation(t *testing.T) {
	dir := t.TempDir()
	stores := []*CredentialStore{NewCredentialStore(dir), NewCredentialStore(dir)}
	results := make(chan ClientAccessCredential, len(stores))
	errorsCh := make(chan error, len(stores))
	var wait sync.WaitGroup
	for index, store := range stores {
		wait.Add(1)
		go func(index int, store *CredentialStore) {
			defer wait.Done()
			credential, err := store.LoadOrCreateIdentity("shared-ref", "endpoint-shared", bytes.NewReader(bytes.Repeat([]byte{byte(index + 1)}, 64)))
			results <- credential
			errorsCh <- err
		}(index, store)
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	var fingerprint string
	for credential := range results {
		if fingerprint == "" {
			fingerprint = credential.Identity.Fingerprint
		} else if credential.Identity.Fingerprint != fingerprint {
			t.Fatalf("cross-instance credential creation split keys: %q != %q", credential.Identity.Fingerprint, fingerprint)
		}
	}
}

func TestCredentialStoreRequiresExplicitScopeExpansion(t *testing.T) {
	store := NewCredentialStore(t.TempDir())
	credential, err := store.LoadOrCreateIdentity("scope-ref", "endpoint-scope", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	issue := func(grantID string, scope Scope) string {
		grant, issueErr := Issue(privateKey, Claims{
			GrantID: grantID, IssuerDeviceID: "device-scope", SubjectKeyFingerprint: credential.Identity.Fingerprint,
			Scope: scope, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
		})
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return grant
	}
	fingerprint := Fingerprint(publicKey)
	if _, err := store.BindGrant("scope-ref", issue("grant-narrow", Scope{AllowDaemon: true}), fingerprint, now, BindGrantOptions{}); err != nil {
		t.Fatal(err)
	}
	expanded := issue("grant-expanded", Scope{AllowDaemon: true, FileReadMetadata: true})
	if _, err := store.BindGrant("scope-ref", expanded, fingerprint, now, BindGrantOptions{}); !errors.Is(err, ErrGrantScopeExpansion) {
		t.Fatalf("silent scope expansion error = %v", err)
	}
	if _, err := store.BindGrant("scope-ref", expanded, fingerprint, now, BindGrantOptions{AllowScopeExpansion: true}); err != nil {
		t.Fatalf("confirmed scope expansion failed: %v", err)
	}
}

func TestCredentialStoreChecksScopeBeforeCommitAndRecoversByClaimDigest(t *testing.T) {
	_, daemonPrivate, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-pair-client", daemonPrivate)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	accessStore, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = accessStore.Close() })
	credentialStore := NewCredentialStore(t.TempDir())
	issue := func(scope Scope, grantLifetime time.Duration) PairingClaimIssueResult {
		issued, issueErr := accessStore.IssuePairingClaim(PairingIssueOptions{
			Scope: scope, TicketTTL: time.Hour, GrantLifetime: grantLifetime, Now: now,
			Routes: []*remoteauthpb.EndpointRouteConfigV1{{SchemaVersion: 1, RouteId: "direct", Enabled: true, Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{SignalingAddresses: []string{"127.0.0.1:4040"}, IceTcpAddresses: []string{"127.0.0.1:4041"}}}}},
		})
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return issued
	}
	redeem := func(issued PairingClaimIssueResult, label string) func(ClientAccessIdentity) (PairingExchangeResult, error) {
		return func(client ClientAccessIdentity) (PairingExchangeResult, error) {
			result, bundle, redeemErr := accessStore.RedeemPairingClaim(issued.OfferPayload, client.PublicKey, label, now)
			result.Bundle = bundle
			return result, redeemErr
		}
	}
	narrow := issue(Scope{TerminalID: "term-1"}, 48*time.Hour)
	if _, err := credentialStore.PairAndBind(
		context.Background(), "pair-ref", "endpoint-1", narrow.OfferPayload, fixedNow(now), rand.Reader, BindGrantOptions{}, redeem(narrow, "narrow"),
	); err != nil {
		t.Fatal(err)
	}
	expanded := issue(Scope{AllowDaemon: true}, 48*time.Hour)
	exchangeCalls := 0
	_, err = credentialStore.PairAndBind(
		context.Background(), "pair-ref", "endpoint-1", expanded.OfferPayload, fixedNow(now), rand.Reader, BindGrantOptions{},
		func(client ClientAccessIdentity) (PairingExchangeResult, error) {
			exchangeCalls++
			return redeem(expanded, "expanded")(client)
		},
	)
	if !errors.Is(err, ErrGrantScopeExpansion) || exchangeCalls != 1 {
		t.Fatalf("scope expansion reached daemon: calls=%d err=%v", exchangeCalls, err)
	}
	if accessStore.tickets[expanded.Claims.TicketID].GrantID == "" {
		t.Fatal("claim was not atomically bound to the prepared client key")
	}
	bound, err := credentialStore.PairAndBind(
		context.Background(), "pair-ref", "endpoint-1", expanded.OfferPayload, fixedNow(now), rand.Reader, BindGrantOptions{AllowScopeExpansion: true},
		func(client ClientAccessIdentity) (PairingExchangeResult, error) {
			exchangeCalls++
			return redeem(expanded, "expanded")(client)
		},
	)
	if err != nil || exchangeCalls != 2 || bound.LastPairingClaimDigest == "" {
		t.Fatalf("confirmed scope expansion = credential %#v calls=%d err=%v", bound, exchangeCalls, err)
	}
	recovered, err := credentialStore.PairAndBind(
		context.Background(), "pair-ref", "endpoint-1", expanded.OfferPayload, fixedNow(now.Add(time.Hour)), rand.Reader, BindGrantOptions{},
		func(client ClientAccessIdentity) (PairingExchangeResult, error) {
			exchangeCalls++
			result, bundle, redeemErr := accessStore.RedeemPairingClaim(expanded.OfferPayload, client.PublicKey, "expanded", now.Add(time.Hour))
			result.Bundle = bundle
			return result, redeemErr
		},
	)
	if err != nil || recovered.CapabilityGrant != bound.CapabilityGrant || exchangeCalls != 3 {
		t.Fatalf("claim delivery recovery = credential %#v calls=%d err=%v", recovered, exchangeCalls, err)
	}
	short := issue(Scope{AllowDaemon: true}, time.Second)
	_, err = credentialStore.PairAndBind(
		context.Background(), "pair-ref", "endpoint-1", short.OfferPayload, fixedNow(now.Add(time.Second+ClockSkewTolerance)), rand.Reader, BindGrantOptions{},
		func(client ClientAccessIdentity) (PairingExchangeResult, error) {
			return redeem(short, "short")(client)
		},
	)
	if !errors.Is(err, ErrGrantExpired) {
		t.Fatalf("expired exchange result error = %v", err)
	}
	unchanged, err := credentialStore.Resolve("pair-ref")
	if err != nil || unchanged.CapabilityGrant != bound.CapabilityGrant || unchanged.LastPairingClaimDigest != bound.LastPairingClaimDigest {
		t.Fatalf("expired exchange replaced existing credential: credential=%#v err=%v", unchanged, err)
	}
}

func TestAccessStoreEnforcesSingleProcessOwner(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-owner", privateKey)
	dir := t.TempDir()
	first, err := LoadAccessStore(dir, identity, AccessStoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAccessStore(dir, identity, AccessStoreOptions{}); !errors.Is(err, filelock.ErrHeld) {
		t.Fatalf("second AccessStore owner error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := LoadAccessStore(dir, identity, AccessStoreOptions{})
	if err != nil {
		t.Fatalf("owner lock was not released: %v", err)
	}
	_ = second.Close()
}

func TestAccessStoreCompactsDeliveredTicketsOnlyOnLaterMutation(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-compaction", privateKey)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, claims, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{AllowDaemon: true}, TicketTTL: time.Minute, GrantLifetime: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePairingBundle(bundle)
	client, _ := GenerateClientAccessIdentity("endpoint-compaction", rand.Reader)
	if _, err := store.RedeemPairingBundle(payload, client.PublicKey, "compaction", now); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.tickets[claims.TicketID]; !exists {
		t.Fatal("consumed ticket disappeared before delivery grace")
	}
	now = now.Add(defaultDeliveryGrace)
	if _, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{MachineEventsOnly: true}, TicketTTL: time.Minute}); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.tickets[claims.TicketID]; exists {
		t.Fatal("expired delivery record was not compacted by later low-frequency mutation")
	}
}

func TestAccessStoreKeepsOwnerAccessLabelSeparateFromClientDeviceLabel(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-access-label", privateKey)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{
		AccessLabel: "App Store review phone", Scope: FullDaemonScope(), TicketTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	client, _ := GenerateClientAccessIdentity("endpoint-access-label", rand.Reader)
	result, err := store.RedeemPairingBundle(payload, client.PublicKey, "anytty-ios", now)
	if err != nil {
		t.Fatal(err)
	}
	records := store.ListClientAccess()
	if len(records) != 1 || records[0].GrantID != result.GrantID || records[0].AccessLabel != "App Store review phone" || records[0].ClientLabel != "anytty-ios" || !records[0].ExpiresAt.IsZero() {
		t.Fatalf("access projection = %#v", records)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now.Add(time.Minute) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })
	records = reloaded.ListClientAccess()
	if len(records) != 1 || records[0].AccessLabel != "App Store review phone" || records[0].ClientLabel != "anytty-ios" || !reloaded.GrantActive(result.GrantID, time.Time{}, now.Add(time.Minute)) {
		t.Fatalf("reloaded access projection = %#v", records)
	}
}

func TestAccessStoreKeepsPublishedStateInMemoryAfterDurabilityError(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-published-state", privateKey)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, claims, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{AllowDaemon: true}, TicketTTL: time.Hour, GrantLifetime: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := EncodePairingBundle(bundle)
	client, _ := GenerateClientAccessIdentity("endpoint-published", rand.Reader)
	originalWriter := store.writeFile
	injected := errors.New("injected post-rename durability failure")
	failNextPublishedWrite := func() {
		failed := false
		store.writeFile = func(path string, payload []byte) error {
			if !failed {
				failed = true
				return writePrivateFileWithPostPublish(path, payload, func(string) error { return injected })
			}
			return originalWriter(path, payload)
		}
	}
	failNextPublishedWrite()
	if _, err := store.RedeemPairingBundle(payload, client.PublicKey, "published", now); !errors.Is(err, injected) {
		t.Fatalf("published redemption error = %v", err)
	}
	grantID := store.tickets[claims.TicketID].GrantID
	if grantID == "" || store.Revoked(grantID) {
		t.Fatalf("published redemption rolled back memory: ticket=%#v", store.tickets[claims.TicketID])
	}
	result, err := store.RedeemPairingBundle(payload, client.PublicKey, "published", now)
	if err != nil || result.GrantID != grantID {
		t.Fatalf("published redemption was not recoverable: result=%#v err=%v", result, err)
	}
	failNextPublishedWrite()
	if _, err := store.RevokeGrant(grantID); !errors.Is(err, injected) {
		t.Fatalf("published revoke error = %v", err)
	}
	if !store.Revoked(grantID) {
		t.Fatal("published revoke rolled back in-memory revocation truth")
	}
	snapshot, err := LoadAccessSnapshot(filepath.Dir(store.path), identity)
	if err != nil || !snapshot.Revoked(grantID) {
		t.Fatalf("published revoke was not visible on disk: snapshot=%#v err=%v", snapshot, err)
	}
	if record, err := store.RevokeGrant(grantID); err != nil || record.RevokedAt.IsZero() {
		t.Fatalf("published revoke idempotent retry = record %#v err=%v", record, err)
	}
}

func TestAccessStoreRevokePrePublishFailureKeepsGrantActive(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-revoke-failure", privateKey)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: FullDaemonScope(), TicketTTL: time.Hour, GrantLifetime: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	client, _ := GenerateClientAccessIdentity("endpoint-revoke-failure", rand.Reader)
	result, err := store.RedeemPairingBundle(payload, client.PublicKey, "revoke-failure", now)
	if err != nil {
		t.Fatal(err)
	}
	if !store.GrantActive(result.GrantID, result.ExpiresAt, now) {
		t.Fatal("redeemed grant is not active before revoke")
	}
	revision := store.AccessProjectionRevision()
	originalWriter := store.writeFile
	injected := errors.New("injected pre-publish revoke failure")
	store.writeFile = func(string, []byte) error { return injected }
	if _, err := store.RevokeGrant(result.GrantID); !errors.Is(err, injected) {
		t.Fatalf("revoke error = %v", err)
	}
	store.writeFile = originalWriter
	if store.Revoked(result.GrantID) || !store.GrantActive(result.GrantID, result.ExpiresAt, now) {
		t.Fatal("pre-publish failure changed in-memory grant truth")
	}
	if store.AccessProjectionRevision() != revision {
		t.Fatalf("pre-publish failure revision = %d, want %d", store.AccessProjectionRevision(), revision)
	}
	snapshot, err := LoadAccessSnapshot(dir, identity)
	if err != nil || snapshot.Revoked(result.GrantID) {
		t.Fatalf("pre-publish failure changed durable grant truth: snapshot=%#v err=%v", snapshot, err)
	}
}

func TestAccessStoreConcurrentRedeemIdempotencyRevokeAndRestart(t *testing.T) {
	_, daemonPrivate, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-1", daemonPrivate)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: FullDaemonScope(), TicketTTL: time.Hour, GrantLifetime: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	bundlePayload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	clientA, _ := GenerateClientAccessIdentity("endpoint-1", rand.Reader)
	clientB, _ := GenerateClientAccessIdentity("endpoint-1", rand.Reader)
	type redemption struct {
		result PairingExchangeResult
		err    error
	}
	start := make(chan struct{})
	results := make(chan redemption, 2)
	var wg sync.WaitGroup
	for _, client := range []ClientAccessIdentity{clientA, clientB} {
		wg.Add(1)
		go func(client ClientAccessIdentity) {
			defer wg.Done()
			<-start
			result, redeemErr := store.RedeemPairingBundle(bundlePayload, client.PublicKey, client.Fingerprint, now)
			results <- redemption{result: result, err: redeemErr}
		}(client)
	}
	close(start)
	wg.Wait()
	close(results)
	var winner redemption
	consumed := 0
	for result := range results {
		if result.err == nil {
			winner = result
		} else if errors.Is(result.err, ErrPairingTicketConsumed) {
			consumed++
		} else {
			t.Fatalf("unexpected redemption error: %v", result.err)
		}
	}
	if winner.result.Grant == "" || consumed != 1 {
		t.Fatalf("concurrent redemption winner=%#v consumed=%d", winner, consumed)
	}
	winnerPublic := clientA.PublicKey
	if winner.result.SubjectKeyFingerprint == clientB.Fingerprint {
		winnerPublic = clientB.PublicKey
	}
	retry, err := store.RedeemPairingBundle(bundlePayload, winnerPublic, "renamed client", now)
	if err != nil || retry.Grant != winner.result.Grant || retry.DeliveryReceipt != winner.result.DeliveryReceipt {
		t.Fatalf("idempotent retry changed result: retry=%#v err=%v winner=%#v", retry, err, winner.result)
	}
	expiredRetry, err := store.RedeemPairingBundle(bundlePayload, winnerPublic, "renamed again", now.Add(2*time.Hour))
	if err != nil || expiredRetry.Grant != winner.result.Grant || expiredRetry.DeliveryReceipt != winner.result.DeliveryReceipt {
		t.Fatalf("lost-response recovery after ticket expiry changed result: retry=%#v err=%v winner=%#v", expiredRetry, err, winner.result)
	}
	loserPublic := clientB.PublicKey
	if winner.result.SubjectKeyFingerprint == clientB.Fingerprint {
		loserPublic = clientA.PublicKey
	}
	if _, err := store.RedeemPairingBundle(bundlePayload, loserPublic, "loser", now.Add(2*time.Hour)); !errors.Is(err, ErrPairingTicketConsumed) {
		t.Fatalf("expired ticket bound to another key error = %v", err)
	}
	if _, err := store.RedeemPairingBundle(bundlePayload, winnerPublic, "winner", now.Add(defaultDeliveryGrace)); !errors.Is(err, ErrPairingTicketConsumed) {
		t.Fatalf("delivery grace exact expiry error = %v", err)
	}
	beforeReadOnly, err := os.Stat(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(winner.result.Grant, identity.Fingerprint, now.Add(time.Minute), store); err != nil {
		t.Fatalf("verify active grant: %v", err)
	}
	if len(store.ListClientAccess()) != 1 {
		t.Fatal("read-only access projection lost grant")
	}
	afterReadOnly, err := os.Stat(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(beforeReadOnly, afterReadOnly) {
		t.Fatal("ordinary capability verification or access listing rewrote AccessStore")
	}
	storedPayload, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(storedPayload, []byte(winner.result.Grant)) || bytes.Contains(storedPayload, []byte(winner.result.DeliveryReceipt)) ||
		bytes.Contains(storedPayload, []byte("anytty-grant-v2")) || bytes.Contains(storedPayload, bundlePayload) {
		t.Fatal("AccessStore persisted raw pairing bundle, grant, or delivery receipt")
	}
	unknownGrant, err := Issue(identity.PrivateKey, Claims{
		GrantID: "grant-outside-store", IssuerDeviceID: identity.DeviceID, SubjectKeyFingerprint: winner.result.SubjectKeyFingerprint,
		Scope: Scope{AllowDaemon: true}, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(unknownGrant, identity.Fingerprint, now.Add(time.Minute), store); !errors.Is(err, ErrGrantRevoked) {
		t.Fatalf("grant absent from AccessStore must fail closed, got %v", err)
	}
	expiredBundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{TerminalID: "term-expired"}, TicketTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	expiredPayload, err := EncodePairingBundle(expiredBundle)
	if err != nil {
		t.Fatal(err)
	}
	unusedClient, _ := GenerateClientAccessIdentity("endpoint-expired", rand.Reader)
	if _, err := store.RedeemPairingBundle(expiredPayload, unusedClient.PublicKey, "expired", now.Add(time.Minute+ClockSkewTolerance)); !errors.Is(err, ErrPairingTicketExpired) {
		t.Fatalf("unconsumed expired ticket error = %v", err)
	}
	beforeRevokeRevision := store.AccessProjectionRevision()
	if beforeRevokeRevision == 0 {
		t.Fatal("grant creation did not advance access projection revision")
	}
	if !store.GrantActive(winner.result.GrantID, winner.result.ExpiresAt, now) {
		t.Fatal("grant is not active before durable revoke")
	}
	if _, err := store.RevokeGrant(winner.result.GrantID); err != nil {
		t.Fatal(err)
	}
	if store.AccessProjectionRevision() != beforeRevokeRevision+1 {
		t.Fatalf("revoke revision = %d, want %d", store.AccessProjectionRevision(), beforeRevokeRevision+1)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now.Add(time.Minute) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })
	if !reloaded.Revoked(winner.result.GrantID) || len(reloaded.ListClientAccess()) != 1 {
		t.Fatalf("access truth did not survive restart: %#v", reloaded.ListClientAccess())
	}
	if reloaded.GrantActive(winner.result.GrantID, winner.result.ExpiresAt, now.Add(time.Minute)) {
		t.Fatal("revoked grant became active after restart")
	}
	if reloaded.AccessProjectionRevision() != beforeRevokeRevision+1 {
		t.Fatalf("reloaded access projection revision = %d", reloaded.AccessProjectionRevision())
	}
	if _, err := Verify(winner.result.Grant, identity.Fingerprint, now.Add(time.Minute), reloaded); !errors.Is(err, ErrGrantRevoked) {
		t.Fatalf("reloaded revocation error = %v", err)
	}
}

func TestCredentialStoreRejectsUnsafeOrMissingRef(t *testing.T) {
	store := NewCredentialStore(t.TempDir())
	for _, ref := range []string{"", "../escape", "with space", "/absolute"} {
		if _, err := store.LoadOrCreateIdentity(ref, "endpoint-1", rand.Reader); err == nil {
			t.Fatalf("unsafe ref %q was accepted", ref)
		}
	}
	if _, err := store.Resolve("missing"); err == nil {
		t.Fatal("missing credential ref must fail")
	}
}

func TestAccessStoreRejectsBrokenTicketGrantLinkOnRestart(t *testing.T) {
	_, daemonPrivate, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-corrupt", daemonPrivate)
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store, err := LoadAccessStore(dir, identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, claims, err := store.IssuePairingBundle(PairingIssueOptions{Scope: FullDaemonScope(), TicketTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	client, _ := GenerateClientAccessIdentity("endpoint-corrupt", rand.Reader)
	bundlePayload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RedeemPairingBundle(bundlePayload, client.PublicKey, "original", now); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var state storedAccessState
	if err := json.Unmarshal(payload, &state); err != nil {
		t.Fatal(err)
	}
	record := state.Tickets[claims.TicketID]
	record.ClientLabel = "tampered"
	state.Tickets[claims.TicketID] = record
	payload, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAccessStore(dir, identity, AccessStoreOptions{}); err == nil {
		t.Fatal("corrupt ticket/grant linkage was accepted after restart")
	}
}
