package main

import (
	"context"
	"path/filepath"
	"testing"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCLICloudClientUsesOfficialControllerByDefault(t *testing.T) {
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS", "")
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME", "")
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_CA", "")
	client, err := cliCloudClientFromEnvironment("00000000-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("official AnyTTY Cloud client was disabled without environment overrides")
	}
	endpoint, err := cliCloudControllerEndpointFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.address != "cloud.anytty.com:443" || endpoint.serverName != "cloud.anytty.com" {
		t.Fatalf("default Controller endpoint = %s / %s", endpoint.address, endpoint.serverName)
	}
}

func TestCLICloudClientRejectsPartialOverride(t *testing.T) {
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS", "controller.example.com:443")
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME", "")
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_CA", "")
	if _, err := cliCloudClientFromEnvironment("00000000-0000-4000-8000-000000000001"); err == nil {
		t.Fatal("partial Cloud Controller override was accepted")
	}
}

func TestEndpointRuntimeCloudProductMatchesEntryPoint(t *testing.T) {
	if got := endpointRuntimeCloudProduct(cliEndpointClientName); got != cloudv1.ClientProduct_CLIENT_PRODUCT_CLI {
		t.Fatalf("CLI Cloud product = %v", got)
	}
	if got := endpointRuntimeCloudProduct(tuiEndpointClientName); got != cloudv1.ClientProduct_CLIENT_PRODUCT_TUI {
		t.Fatalf("TUI Cloud product = %v", got)
	}
}

func TestCLIRoutePlanEnvironmentRequiresCapabilityAndSSHCredentials(t *testing.T) {
	target := clientendpoint.Endpoint{
		ID: "studio", Label: "Studio", LabelSource: clientendpoint.SourceUser, ConnectMode: clientendpoint.ConnectOnDemand, Enabled: true,
		DaemonIdentity: clientendpoint.DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"},
		Routes: map[clientendpoint.RouteID]clientendpoint.AccessRoute{
			"local": {ID: "local", Kind: clientendpoint.RouteLocalUnix, Enabled: true, Source: clientendpoint.SourceLocal, PolicySource: clientendpoint.SourceUser, Socket: "auto"},
			"ssh": {ID: "ssh", Kind: clientendpoint.RouteSSHWebRTCTCP, Enabled: true, Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceUser,
				Host: "studio", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121", CredentialRef: "credential:ssh-studio", SSHCredentialRef: "ssh:studio"},
			"direct": {ID: "direct", Kind: clientendpoint.RouteDirectWebRTCTCP, Enabled: true, Source: clientendpoint.SourceManual, PolicySource: clientendpoint.SourceUser,
				SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41121"}, CredentialRef: "credential:missing"},
		},
	}
	environment := cliRoutePlanEnvironment(context.Background(), target,
		fakeCLICapabilityAvailability{"credential:ssh-studio": true}, fakeCLISSHAvailability{"ssh:studio": true})
	if len(environment.SupportedRouteKinds) != 3 {
		t.Fatalf("supported route kinds = %#v", environment.SupportedRouteKinds)
	}
	if len(environment.AvailableCredentialRefs) != 2 || environment.AvailableCredentialRefs[0] != "credential:ssh-studio" || environment.AvailableCredentialRefs[1] != "ssh:studio" {
		t.Fatalf("available credentials = %#v", environment.AvailableCredentialRefs)
	}
}

func TestCLIEndpointPlanSourceReloadsPersistedRoutePriorities(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	registry := clientendpoint.DefaultRegistry()
	target := registry.Endpoints[clientendpoint.DefaultEndpointID]
	target.Routes["backup"] = clientendpoint.AccessRoute{
		ID: "backup", Kind: clientendpoint.RouteLocalUnix, Enabled: true,
		Source: clientendpoint.SourceLocal, PolicySource: clientendpoint.SourceUser, Socket: "/tmp/anytty-backup.sock",
	}
	registry.Endpoints[target.ID] = target
	if err := clientendpoint.Save("", registry); err != nil {
		t.Fatal(err)
	}

	source := cliEndpointPlanSource{initialTarget: target}
	before, err := source.Snapshot(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	zero, ten := 0, 10
	if _, err := clientendpoint.Update("", false, func(current clientendpoint.Registry) (clientendpoint.Registry, error) {
		return clientendpoint.SetAutomaticRoutePriorities(current, target.ID, map[clientendpoint.RouteID]*int{
			clientendpoint.DefaultLocalRouteID: &zero,
			"backup":                           &ten,
		})
	}); err != nil {
		t.Fatal(err)
	}
	after, err := source.Snapshot(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ConfigKey == before.ConfigKey {
		t.Fatal("persisted priority update did not invalidate the next connection plan")
	}
	if priority := after.Endpoint.Routes[clientendpoint.DefaultLocalRouteID].Priority; priority == nil || *priority != 0 {
		t.Fatalf("reloaded local priority = %#v", priority)
	}
	if priority := after.Endpoint.Routes["backup"].Priority; priority == nil || *priority != 10 {
		t.Fatalf("reloaded backup priority = %#v", priority)
	}
}

func TestCLIEndpointPlanSourceHonorsExplicitRegistryPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	customPath := filepath.Join(t.TempDir(), "endpoints.yaml")
	target := testLocalEndpoint("custom", "Custom", "/tmp/anytty-custom.sock", clientendpoint.ConnectOnDemand, true)
	if err := clientendpoint.Save(customPath, clientendpoint.Registry{
		Version: clientendpoint.RegistryVersion,
		Default: target.ID,
		Endpoints: map[clientendpoint.EndpointID]clientendpoint.Endpoint{
			target.ID: target,
		},
	}); err != nil {
		t.Fatal(err)
	}

	source := cliEndpointPlanSource{registryPath: customPath, initialTarget: target}
	snapshot, err := source.Snapshot(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Endpoint.ID != target.ID || !snapshot.Endpoint.Enabled {
		t.Fatalf("snapshot came from wrong registry: %#v", snapshot.Endpoint)
	}
}

type fakeCLICapabilityAvailability map[string]bool

func (values fakeCLICapabilityAvailability) Available(_ context.Context, _ string, reference string) bool {
	return values[reference]
}

type fakeCLISSHAvailability map[string]bool

func (values fakeCLISSHAvailability) Available(reference string) bool { return values[reference] }
