package endpoint

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type plannerFixture struct {
	Version int                  `json:"version"`
	Cases   []plannerFixtureCase `json:"cases"`
}

type plannerFixtureCase struct {
	Name                    string                     `json:"name"`
	Generation              SessionGeneration          `json:"generation"`
	Intent                  string                     `json:"intent"`
	RouteOverride           RouteID                    `json:"route_override,omitempty"`
	HedgeDelayMillis        int64                      `json:"hedge_delay_millis,omitempty"`
	SupportedRouteKinds     []RouteKind                `json:"supported_route_kinds"`
	AvailableCredentialRefs []string                   `json:"available_credential_refs,omitempty"`
	Routes                  []plannerFixtureRoute      `json:"routes"`
	ExpectedGroups          []plannerFixtureGroup      `json:"expected_groups,omitempty"`
	ExpectedDiagnostics     []plannerFixtureDiagnostic `json:"expected_diagnostics,omitempty"`
	ExpectedError           ErrorCode                  `json:"expected_error,omitempty"`
}

type plannerFixtureDiagnostic struct {
	RouteID RouteID                   `json:"route_id"`
	Code    ErrorCode                 `json:"code"`
	Reason  RoutePlanDiagnosticReason `json:"reason"`
}

type plannerFixtureRoute struct {
	ID               RouteID   `json:"id"`
	Kind             RouteKind `json:"kind"`
	Enabled          bool      `json:"enabled"`
	ManualOnly       bool      `json:"manual_only,omitempty"`
	Priority         *int      `json:"priority,omitempty"`
	CredentialRef    string    `json:"credential_ref,omitempty"`
	SSHCredentialRef string    `json:"ssh_credential_ref,omitempty"`
}

type plannerFixtureGroup struct {
	Priority         *int      `json:"priority,omitempty"`
	StartDelayMillis int64     `json:"start_delay_millis"`
	RouteIDs         []RouteID `json:"route_ids"`
}

func TestRouteSelectionPlannerFixture(t *testing.T) {
	payload, err := os.ReadFile("testdata/route-selection-plan-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture plannerFixture
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != 1 || len(fixture.Cases) == 0 {
		t.Fatalf("invalid planner fixture header: %#v", fixture)
	}
	for _, test := range fixture.Cases {
		t.Run(test.Name, func(t *testing.T) {
			target := endpointFromPlannerFixture(test)
			plan, err := (RouteSelectionPlanner{}).Plan(RouteSelectionRequest{
				Endpoint: target, Intent: ConnectIntent{Kind: test.Intent}, RouteOverride: test.RouteOverride, Generation: test.Generation,
				SupportedRouteKinds: test.SupportedRouteKinds, AvailableCredentialRefs: test.AvailableCredentialRefs,
			})
			if test.ExpectedError != "" {
				if !IsCode(err, test.ExpectedError) {
					t.Fatalf("error = %v, want code %q", err, test.ExpectedError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			groups := plan.Groups()
			if len(groups) != len(test.ExpectedGroups) {
				t.Fatalf("groups = %#v, want %#v", groups, test.ExpectedGroups)
			}
			for index, expected := range test.ExpectedGroups {
				group := groups[index]
				if !equalPriority(group.Priority(), expected.Priority) || group.StartDelay() != time.Duration(expected.StartDelayMillis)*time.Millisecond {
					t.Fatalf("group[%d] priority/delay = %v/%s, want %v/%dms", index, group.Priority(), group.StartDelay(), expected.Priority, expected.StartDelayMillis)
				}
				attempts := group.Attempts()
				if len(attempts) != len(expected.RouteIDs) {
					t.Fatalf("group[%d] attempts = %#v, want %v", index, attempts, expected.RouteIDs)
				}
				for attemptIndex, routeID := range expected.RouteIDs {
					if attempts[attemptIndex].Route.ID != routeID {
						t.Fatalf("group[%d] route[%d] = %q, want %q", index, attemptIndex, attempts[attemptIndex].Route.ID, routeID)
					}
				}
			}
			diagnostics := plan.Diagnostics()
			if len(diagnostics) != len(test.ExpectedDiagnostics) {
				t.Fatalf("diagnostics = %#v, want %#v", diagnostics, test.ExpectedDiagnostics)
			}
			for index, expected := range test.ExpectedDiagnostics {
				if diagnostics[index] != (RoutePlanDiagnostic{RouteID: expected.RouteID, Code: expected.Code, Reason: expected.Reason}) {
					t.Fatalf("diagnostic[%d] = %#v, want %#v", index, diagnostics[index], expected)
				}
			}
		})
	}
}

func endpointFromPlannerFixture(test plannerFixtureCase) Endpoint {
	identity := DaemonIdentity{DeviceID: "device-fixture", DeviceFingerprint: "SHA256:fixture"}
	target := Endpoint{
		ID: "fixture", Label: "Fixture", LabelSource: SourceUser, DaemonIdentity: identity, ConnectMode: ConnectOnDemand, Enabled: true,
		Routes: make(map[RouteID]AccessRoute, len(test.Routes)),
	}
	if test.HedgeDelayMillis != 0 {
		target.SelectionPolicy = SelectionPolicy{HedgeDelay: time.Duration(test.HedgeDelayMillis) * time.Millisecond, HedgeDelayConfigured: true}
	}
	for _, value := range test.Routes {
		route := AccessRoute{
			ID: value.ID, Kind: value.Kind, Enabled: value.Enabled, ManualOnly: value.ManualOnly, Priority: clonePriority(value.Priority),
			CredentialRef: value.CredentialRef, SSHCredentialRef: value.SSHCredentialRef, Source: SourceManual, PolicySource: SourceUser,
		}
		switch route.Kind {
		case RouteLocalUnix:
			route.Socket = "auto"
		case RouteSSHWebRTCTCP:
			route.Host = "fixture-host"
			route.RemoteSignalingAddress = "127.0.0.1:41120"
			route.RemoteICETCPAddress = "127.0.0.1:41121"
		case RouteDirectWebRTCTCP:
			route.SignalingAddresses = []string{"fixture.local:41120"}
			route.ICETCPAddresses = []string{"fixture.local:41121"}
		case RouteManagedWebRTC:
			route.TargetDeviceID = identity.DeviceID
			route.RelayMode = RelayAuto
		}
		target.Routes[route.ID] = route
	}
	return target
}

func equalPriority(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
