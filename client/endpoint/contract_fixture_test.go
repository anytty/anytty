package endpoint

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/timepolicy"
)

type endpointContractFixture struct {
	SchemaVersion            int             `json:"schema_version"`
	EmptyRegistry            json.RawMessage `json:"empty_registry"`
	ValidRegistry            json.RawMessage `json:"valid_registry"`
	DefaultedManagedRegistry json.RawMessage `json:"defaulted_managed_registry"`
	UnknownFieldRegistry     json.RawMessage `json:"unknown_field_registry"`
	MissingFieldRegistry     json.RawMessage `json:"missing_field_registry"`
	WrongTypeRegistry        json.RawMessage `json:"wrong_type_registry"`
	WhitespaceKeyRegistry    json.RawMessage `json:"whitespace_key_registry"`
	InvalidRegistryCases     []struct {
		Name          string          `json:"name"`
		Registry      json.RawMessage `json:"registry"`
		ExpectedError ErrorCode       `json:"expected_error"`
	} `json:"invalid_registry_cases"`
	LocalDiscoveryCandidate struct {
		ClaimedDeviceID          string `json:"claimed_device_id"`
		ClaimedDeviceFingerprint string `json:"claimed_device_fingerprint"`
		Address                  string `json:"address"`
		Port                     uint16 `json:"port"`
		ProtocolVersion          uint32 `json:"protocol_version"`
		TTLMillis                int64  `json:"ttl_millis"`
		SignatureBytes           int    `json:"signature_bytes"`
	} `json:"local_discovery_candidate"`
	Assembler struct {
		InitialRegistry           json.RawMessage            `json:"initial_registry"`
		ConfirmedIdentityBindings []ConfirmedIdentityBinding `json:"confirmed_identity_bindings"`
		CommutativeCandidates     []EndpointCandidate        `json:"commutative_candidates"`
		IdentityConflictCandidate EndpointCandidate          `json:"identity_conflict_candidate"`
		ExpectedEndpointID        EndpointID                 `json:"expected_endpoint_id"`
		ExpectedNewEndpointID     EndpointID                 `json:"expected_new_endpoint_id"`
		ExpectedLabel             string                     `json:"expected_label"`
		ExpectedConnectMode       ConnectMode                `json:"expected_connect_mode"`
		ExpectedRouteIDs          []string                   `json:"expected_route_ids"`
		ExpectedRoutePriorities   map[string]int             `json:"expected_route_priorities"`
		ExpectedConflictError     ErrorCode                  `json:"expected_conflict_error"`
	} `json:"assembler"`
	OversizeBytes int `json:"oversize_bytes"`
}

func TestSharedEndpointContractFixture(t *testing.T) {
	fixture := loadEndpointContractFixture(t)
	if fixture.SchemaVersion != 1 {
		t.Fatalf("fixture schema version = %d", fixture.SchemaVersion)
	}
	empty, err := Parse(fixture.EmptyRegistry)
	if err != nil || empty.Default != "" || len(empty.Endpoints) != 0 {
		t.Fatalf("strict parse shared empty registry: registry=%#v err=%v", empty, err)
	}

	registry, err := Parse(fixture.ValidRegistry)
	if err != nil {
		t.Fatalf("strict parse shared registry: %v", err)
	}
	encoded, err := Encode(registry)
	if err != nil {
		t.Fatalf("encode shared registry: %v", err)
	}
	roundTripped, err := Parse(encoded)
	if err != nil || !reflect.DeepEqual(roundTripped, registry) {
		t.Fatalf("shared registry round trip mismatch: err=%v\ngot=%#v\nwant=%#v", err, roundTripped, registry)
	}
	if _, err := Parse(fixture.UnknownFieldRegistry); !IsCode(err, ErrorConfig) {
		t.Fatalf("unknown field error = %v", err)
	}
	if _, err := Parse(fixture.MissingFieldRegistry); !IsCode(err, ErrorConfig) {
		t.Fatalf("missing field error = %v", err)
	}
	if _, err := Parse(fixture.WrongTypeRegistry); !IsCode(err, ErrorConfig) {
		t.Fatalf("wrong type error = %v", err)
	}
	if _, err := Parse(fixture.WhitespaceKeyRegistry); !IsCode(err, ErrorConfig) {
		t.Fatalf("whitespace key error = %v", err)
	}
	defaultedManaged, err := Parse(fixture.DefaultedManagedRegistry)
	if err != nil || defaultedManaged.Endpoints["cloud"].Routes["cloud"].RelayMode != RelayAuto {
		t.Fatalf("managed relay default = %#v err=%v", defaultedManaged, err)
	}
	for _, testCase := range fixture.InvalidRegistryCases {
		t.Run("invalid_registry_"+testCase.Name, func(t *testing.T) {
			if _, err := Parse(testCase.Registry); !IsCode(err, testCase.ExpectedError) {
				t.Fatalf("invalid registry error = %v, want %s", err, testCase.ExpectedError)
			}
		})
	}
	if _, err := Parse(bytes.Repeat([]byte{'x'}, fixture.OversizeBytes)); !IsCode(err, ErrorSizeLimit) {
		t.Fatalf("oversize error = %v", err)
	}
	now := time.Now()
	discovery := LocalDiscoveryCandidate{
		ClaimedIdentity: DaemonIdentity{
			DeviceID: fixture.LocalDiscoveryCandidate.ClaimedDeviceID, DeviceFingerprint: fixture.LocalDiscoveryCandidate.ClaimedDeviceFingerprint,
		},
		Address: fixture.LocalDiscoveryCandidate.Address, Port: fixture.LocalDiscoveryCandidate.Port,
		ProtocolVersion:    fixture.LocalDiscoveryCandidate.ProtocolVersion,
		AnnouncementExpiry: now.Add(time.Duration(fixture.LocalDiscoveryCandidate.TTLMillis) * time.Millisecond),
		Signature:          make([]byte, fixture.LocalDiscoveryCandidate.SignatureBytes),
	}
	if err := discovery.Validate(now); err != nil {
		t.Fatalf("local discovery candidate: %v", err)
	}
	discovery.AnnouncementExpiry = now.Add(-timepolicy.ClockSkewTolerance + time.Nanosecond)
	if err := discovery.Validate(now); err != nil {
		t.Fatalf("local discovery within clock tolerance: %v", err)
	}
	discovery.AnnouncementExpiry = now.Add(-timepolicy.ClockSkewTolerance)
	if err := discovery.Validate(now); !IsCode(err, ErrorConfig) {
		t.Fatalf("expired local discovery error = %v", err)
	}
	if bytes.Contains(fixture.ValidRegistry, []byte(fixture.LocalDiscoveryCandidate.Address)) {
		t.Fatal("ephemeral discovery address entered saved registry fixture")
	}

	initial, err := Parse(fixture.Assembler.InitialRegistry)
	if err != nil {
		t.Fatalf("parse assembler initial registry: %v", err)
	}
	forward := assembleFixtureCandidates(t, initial, fixture.Assembler.CommutativeCandidates, fixture.Assembler.ConfirmedIdentityBindings)
	reverseCandidates := append([]EndpointCandidate(nil), fixture.Assembler.CommutativeCandidates...)
	for left, right := 0, len(reverseCandidates)-1; left < right; left, right = left+1, right-1 {
		reverseCandidates[left], reverseCandidates[right] = reverseCandidates[right], reverseCandidates[left]
	}
	reverse := assembleFixtureCandidates(t, initial, reverseCandidates, fixture.Assembler.ConfirmedIdentityBindings)
	if !reflect.DeepEqual(forward, reverse) {
		t.Fatalf("shared fixture import is not commutative\nforward=%#v\nreverse=%#v", forward, reverse)
	}
	if len(forward.Endpoints) != 1 || forward.Default != fixture.Assembler.ExpectedEndpointID {
		t.Fatalf("confirmed identity binding did not preserve the SSH endpoint: %#v", forward)
	}
	fromEmpty := assembleFixtureCandidates(t, empty, fixture.Assembler.CommutativeCandidates, nil)
	if len(fromEmpty.Endpoints) != 1 || fromEmpty.Default != fixture.Assembler.ExpectedNewEndpointID {
		t.Fatalf("shared empty registry import invented another endpoint: %#v", fromEmpty)
	}
	endpoint := forward.Endpoints[fixture.Assembler.ExpectedEndpointID]
	routeIDs := make([]string, 0, len(endpoint.Routes))
	for routeID := range endpoint.Routes {
		routeIDs = append(routeIDs, string(routeID))
	}
	sort.Strings(routeIDs)
	priorities := make(map[string]int, len(endpoint.Routes))
	for routeID, route := range endpoint.Routes {
		if route.Priority == nil {
			t.Fatalf("shared fixture route %q has no priority", routeID)
		}
		priorities[string(routeID)] = *route.Priority
	}
	if endpoint.Label != fixture.Assembler.ExpectedLabel || endpoint.ConnectMode != fixture.Assembler.ExpectedConnectMode ||
		!reflect.DeepEqual(routeIDs, fixture.Assembler.ExpectedRouteIDs) || !reflect.DeepEqual(priorities, fixture.Assembler.ExpectedRoutePriorities) {
		t.Fatalf("shared fixture endpoint = %#v routes=%v", endpoint, routeIDs)
	}
	_, err = AssembleEndpoints(EndpointAssemblerInput{Registry: forward, Candidates: []EndpointCandidate{fixture.Assembler.IdentityConflictCandidate}})
	if !IsCode(err, fixture.Assembler.ExpectedConflictError) {
		t.Fatalf("shared fixture conflict error = %v", err)
	}
}

func loadEndpointContractFixture(t *testing.T) endpointContractFixture {
	t.Helper()
	payload, err := os.ReadFile("testdata/endpoint-contract-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture endpointContractFixture
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatalf("decode endpoint contract fixture: %v", err)
	}
	return fixture
}

func assembleFixtureCandidates(t *testing.T, initial Registry, candidates []EndpointCandidate, bindings []ConfirmedIdentityBinding) Registry {
	t.Helper()
	registry := initial
	for index, candidate := range candidates {
		input := EndpointAssemblerInput{Registry: registry, Candidates: []EndpointCandidate{candidate}}
		if index == 0 {
			input.ConfirmedIdentityBindings = bindings
		}
		result, err := AssembleEndpoints(input)
		if err != nil {
			t.Fatalf("assemble fixture candidate: %v", err)
		}
		registry = result.Registry
	}
	return registry
}
