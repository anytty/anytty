package endpoint

import (
	"testing"
	"time"
)

func TestClientEndpointShareBundleIsConfigOnlyAndProducesDiff(t *testing.T) {
	now := time.Now().UTC()
	priority := 20
	target := Endpoint{
		ID: "studio", Label: "Studio", LabelSource: SourceUser,
		DaemonIdentity: DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"},
		ConnectMode:    ConnectAuto, Enabled: true, SelectionPolicy: SelectionPolicy{HedgeDelayConfigured: true, HedgeDelay: 250 * time.Millisecond},
		Routes: map[RouteID]AccessRoute{
			"local":  {ID: "local", Kind: RouteLocalUnix, Enabled: true, ManualOnly: true, Socket: "/tmp/anytty.sock", Source: SourceLocal, PolicySource: SourceLocal},
			"direct": {ID: "direct", Kind: RouteDirectWebRTCTCP, Enabled: true, Priority: &priority, CredentialRef: "grant:source", Source: SourceManual, PolicySource: SourceUser, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41121"}},
		},
	}
	bundle, err := NewClientEndpointShareBundle(target, "share-transfer", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.GetRoutes()) != 1 || bundle.GetRoutes()[0].GetLocalUnix() != nil || bundle.GetRoutes()[0].GetCredentialRef() != "" || len(bundle.GetBoundGrant()) != 0 {
		t.Fatalf("share bundle leaked non-portable state: %#v", bundle)
	}
	if len(bundle.GetCredentialDescriptors()) != 1 || bundle.GetCredentialDescriptors()[0].GetExportable() {
		t.Fatalf("share credential descriptors=%#v", bundle.GetCredentialDescriptors())
	}
	payload, err := MarshalClientEndpointShareBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseClientEndpointShareBundle(payload)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := EndpointCandidateFromShareBundle(parsed)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := PreviewShare(Registry{}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Routes) != 1 || diff.Routes[0].Action != "add" || !diff.ConnectModeChanged || !diff.SelectionPolicyChanged {
		t.Fatalf("share diff=%#v", diff)
	}
}

func TestClientEndpointShareRejectsOnlyLocalEndpoint(t *testing.T) {
	target := DefaultRegistry().Endpoints[DefaultEndpointID]
	target.DaemonIdentity = DaemonIdentity{DeviceID: "device-local", DeviceFingerprint: "SHA256:local"}
	if _, err := NewClientEndpointShareBundle(target, "share-local", time.Now(), time.Minute); err == nil {
		t.Fatal("local-only endpoint unexpectedly produced a share bundle")
	}
}
