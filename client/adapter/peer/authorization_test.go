package peer_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/shared/remoteauth"
)

func TestCapabilityAuthorizerAcceptsPublicCredentialWithPlatformSigner(t *testing.T) {
	_, daemonPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	daemon, err := remoteauth.NewIdentity("device-web", daemonPrivate)
	if err != nil {
		t.Fatal(err)
	}
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("web", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := remoteauth.NewPrivateClientAccessSigner(clientIdentity)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	grant, err := remoteauth.Issue(daemon.PrivateKey, remoteauth.Claims{
		GrantID: "grant-web", IssuerDeviceID: daemon.DeviceID, SubjectKeyFingerprint: clientIdentity.Fingerprint,
		Scope: remoteauth.Scope{AllowDaemon: true}, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	publicIdentity := clientIdentity
	publicIdentity.PrivateKey = nil
	credential := remoteauth.ClientAccessCredential{
		Version: 3, EndpointID: "web", Identity: publicIdentity, CapabilityGrant: grant, UpdatedAt: now,
	}
	attempt := authorizationAttempt(t, endpoint.DaemonIdentity{DeviceID: daemon.DeviceID, DeviceFingerprint: daemon.Fingerprint})
	prepared, err := (peeradapter.CapabilityAuthorizer{
		Credentials: staticAuthorizationCredentialSource{credential: credential},
		Signers:     staticAuthorizationSignerSource{signer: signer},
		Now:         func() time.Time { return now },
	}).Prepare(context.Background(), attempt)
	if err != nil || prepared == nil {
		t.Fatalf("prepare public credential = %T, %v", prepared, err)
	}
}

func authorizationAttempt(t *testing.T, identity endpoint.DaemonIdentity) clientruntime.AttemptRequest {
	t.Helper()
	target := endpoint.Endpoint{
		ID: "web", DaemonIdentity: identity,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"cloud": {
				ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser,
				CredentialRef: "credential:web", TargetDeviceID: identity.DeviceID, AccountProfileRef: "default", RelayMode: endpoint.RelayDirect,
			},
		},
	}
	attempt, err := clientruntime.NewAttemptRequest(target, "cloud", 1, clientruntime.ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

type staticAuthorizationCredentialSource struct {
	credential remoteauth.ClientAccessCredential
}

func (source staticAuthorizationCredentialSource) ResolveClientCredential(context.Context, string, string) (remoteauth.ClientAccessCredential, error) {
	return source.credential, nil
}

type staticAuthorizationSignerSource struct{ signer remoteauth.ClientAccessSigner }

func (source staticAuthorizationSignerSource) ResolveClientSigner(context.Context, string, string, remoteauth.ClientAccessIdentity) (remoteauth.ClientAccessSigner, error) {
	return source.signer, nil
}
