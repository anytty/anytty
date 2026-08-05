package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
)

func TestWasAttemptedDefaultsToNoReplayForUnknownErrors(t *testing.T) {
	if WasAttempted(nil) {
		t.Fatal("nil error must not be attempted")
	}
	if !WasAttempted(errors.New("adapter failed")) {
		t.Fatal("unknown error must conservatively prevent replay")
	}
	if WasAttempted(&Error{Code: ErrorStaleSession, Message: "stale", Attempted: false}) {
		t.Fatal("generation guard failure must remain unattempted")
	}
	if !WasAttempted(&Error{Code: ErrorUnavailable, Message: "write failed", Attempted: true}) {
		t.Fatal("adapter write failure must remain attempted")
	}
}

func TestAttemptRequestBindsOneRouteAndGeneration(t *testing.T) {
	target := endpoint.NewSSHEndpoint("studio", "Studio", "studio.example", "ssh:studio", "127.0.0.1:41120", "127.0.0.1:41121", endpoint.ConnectOnDemand)
	request, err := NewAttemptRequest(target, "ssh", 7, ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if stamp := request.Stamp(); stamp.EndpointID != "studio" || stamp.RouteID != "ssh" || stamp.Generation != 7 {
		t.Fatalf("attempt stamp = %#v", stamp)
	}
	if request.Route().Kind != endpoint.RouteSSHWebRTCTCP || request.Intent() != ConnectIntentInteractive {
		t.Fatalf("attempt contract = route %#v intent %q", request.Route(), request.Intent())
	}
}

func TestAttemptRequestDoesNotExposeMutableRegistryRoute(t *testing.T) {
	target := endpoint.NewSSHEndpoint("studio", "Studio", "studio.example", "ssh:studio", "127.0.0.1:41120", "127.0.0.1:41121", endpoint.ConnectOnDemand)
	request, err := NewAttemptRequest(target, "ssh", 3, ConnectIntentBackground)
	if err != nil {
		t.Fatal(err)
	}
	route := target.Routes["ssh"]
	route.Host = "changed.example"
	target.Routes["ssh"] = route
	exposed := request.Route()
	exposed.Host = "consumer-change.example"
	if got := request.Route().Host; got != "studio.example" {
		t.Fatalf("attempt route mutated through registry or getter: %q", got)
	}
}

func TestAttemptRequestRejectsUnknownRouteAndGeneration(t *testing.T) {
	target := endpoint.NewLocalEndpoint(endpoint.DefaultEndpointID, "Local", "auto", endpoint.ConnectAuto)
	if _, err := NewAttemptRequest(target, "missing", 1, ConnectIntentInteractive); CodeOf(err) != ErrorInvalidRequest {
		t.Fatalf("missing route error = %v code=%q", err, CodeOf(err))
	}
	if _, err := NewAttemptRequest(target, "local", 0, ConnectIntentInteractive); CodeOf(err) != ErrorInvalidRequest {
		t.Fatalf("missing generation error = %v code=%q", err, CodeOf(err))
	}
}

func TestApplicationContractsRejectIncompleteStamp(t *testing.T) {
	for name, validate := range map[string]func() error{
		"lease":      func() error { return (SessionLease{}).Validate() },
		"disconnect": func() error { return (DisconnectRequest{}).Validate() },
	} {
		t.Run(name, func(t *testing.T) {
			if err := validate(); CodeOf(err) != ErrorInvalidRequest {
				t.Fatalf("error = %v code=%q", err, CodeOf(err))
			}
		})
	}
}

func TestCodeOfPreservesCancellationAndStableRuntimeCode(t *testing.T) {
	if got := CodeOf(context.Canceled); got != ErrorCanceled {
		t.Fatalf("canceled code = %q", got)
	}
	cause := errors.New("host unavailable")
	err := runtimeError(ErrorAuthorization, "authorization failed", cause)
	if got := CodeOf(err); got != ErrorAuthorization || !errors.Is(err, cause) {
		t.Fatalf("runtime error = %v code=%q", err, got)
	}
}

func TestValidateReadyPeerSessionRejectsWrongGenerationAndMissingLifecycle(t *testing.T) {
	target := endpoint.NewLocalEndpoint(endpoint.DefaultEndpointID, "Local", "auto", endpoint.ConnectAuto)
	request, err := NewAttemptRequest(target, "local", 4, ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	var nilSession *contractReadyPeerSession
	if err := ValidateReadyPeerSession(request, nilSession); CodeOf(err) != ErrorUnavailable {
		t.Fatalf("nil session error = %v code=%q", err, CodeOf(err))
	}
	readyEvidence := ReadyPeerSessionEvidence{Identity: endpoint.DaemonIdentity{DeviceID: "device-ready", DeviceFingerprint: "SHA256:device-ready"}, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: 1}
	wrong := &contractReadyPeerSession{stamp: EndpointSessionStamp{EndpointID: endpoint.DefaultEndpointID, RouteID: "local", Generation: 5}, evidence: readyEvidence, done: make(chan struct{})}
	if err := ValidateReadyPeerSession(request, wrong); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("wrong generation error = %v code=%q", err, CodeOf(err))
	}
	missingLifecycle := &contractReadyPeerSession{stamp: request.Stamp(), evidence: readyEvidence}
	if err := ValidateReadyPeerSession(request, missingLifecycle); CodeOf(err) != ErrorUnavailable {
		t.Fatalf("missing lifecycle error = %v code=%q", err, CodeOf(err))
	}
	ready := &contractReadyPeerSession{stamp: request.Stamp(), evidence: readyEvidence, done: make(chan struct{})}
	if err := ValidateReadyPeerSession(request, ready); err != nil {
		t.Fatal(err)
	}
	closed := make(chan struct{})
	close(closed)
	if err := ValidateReadyPeerSession(request, &contractReadyPeerSession{stamp: request.Stamp(), evidence: readyEvidence, done: closed}); CodeOf(err) != ErrorUnavailable {
		t.Fatalf("ended ready session error = %v", err)
	}
}

func TestReadyPeerSessionEvidenceRequiresProofAuthorizationHelloAndPin(t *testing.T) {
	expected := endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "SHA256:device-1"}
	tests := []struct {
		name     string
		evidence ReadyPeerSessionEvidence
		code     ErrorCode
	}{
		{name: "identity", evidence: ReadyPeerSessionEvidence{AuthorizationVerified: true, ProtocolVersion: 1}, code: ErrorIdentity},
		{name: "authorization", evidence: ReadyPeerSessionEvidence{Identity: expected, IdentityVerified: true, ProtocolVersion: 1}, code: ErrorAuthorization},
		{name: "hello", evidence: ReadyPeerSessionEvidence{Identity: expected, IdentityVerified: true, AuthorizationVerified: true}, code: ErrorUnavailable},
		{name: "pin", evidence: ReadyPeerSessionEvidence{Identity: endpoint.DaemonIdentity{DeviceID: "other", DeviceFingerprint: "SHA256:other"}, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: 1}, code: ErrorIdentity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.evidence.Validate(expected); CodeOf(err) != test.code {
				t.Fatalf("evidence error = %v code=%q, want %q", err, CodeOf(err), test.code)
			}
		})
	}
}

type contractReadyPeerSession struct {
	stamp    EndpointSessionStamp
	evidence ReadyPeerSessionEvidence
	done     chan struct{}
}

func (session *contractReadyPeerSession) Stamp() EndpointSessionStamp { return session.stamp }
func (session *contractReadyPeerSession) ObservedPath() string        { return "" }
func (session *contractReadyPeerSession) Readiness() ReadyPeerSessionEvidence {
	return session.evidence
}
func (session *contractReadyPeerSession) Done() <-chan struct{} { return session.done }
func (session *contractReadyPeerSession) Err() error            { return nil }
func (session *contractReadyPeerSession) Close() error          { return nil }

var _ ReadyPeerSession = (*contractReadyPeerSession)(nil)

type contractRuntime struct{}

func (contractRuntime) EnsureSession(context.Context, ConnectRequest) (SessionLease, error) {
	return SessionLease{}, nil
}

func (contractRuntime) Disconnect(context.Context, DisconnectRequest) error {
	return nil
}

func (contractRuntime) WatchEndpoint(context.Context, endpoint.EndpointID) (<-chan EndpointEvent, error) {
	return make(chan EndpointEvent), nil
}

var _ Runtime = contractRuntime{}
