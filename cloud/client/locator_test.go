package client

import (
	"errors"
	"testing"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestCachedCapabilityRouteRoundTripsWithoutController(t *testing.T) {
	edge := &cloudv1.EdgeLocator{EdgeId: "edge-a", PublicEndpoint: "edge.example:443", ServerName: "edge.example", CaCertificatePem: []byte("ca")}
	locator, err := EncodeEdgeLocator(edge)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := proto.Marshal(&cloudv1.CloudRouteGrantClaims{})
	if err != nil {
		t.Fatal(err)
	}
	grantPayload, err := proto.Marshal(&cloudv1.SignedEnvelope{KeyId: "daemon-key", Payload: claims, Signature: []byte("signature")})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := NewCachedCapabilityRoute(locator, grantPayload)
	if err != nil {
		t.Fatal(err)
	}
	resolved := resolution.Locator()
	if resolved.GetEdgeId() != edge.GetEdgeId() || resolved.GetPublicEndpoint() != edge.GetPublicEndpoint() || resolved.GetServerName() != edge.GetServerName() {
		t.Fatalf("cached Edge = %v", resolved)
	}
	resolved.PublicEndpoint = "mutated"
	if resolution.Locator().GetPublicEndpoint() != edge.GetPublicEndpoint() {
		t.Fatal("cached route exposed mutable Edge state")
	}
	if timeout := resolution.edgeTransportTimeout(); timeout != 1500*time.Millisecond {
		t.Fatalf("cached Edge transport timeout = %s", timeout)
	}
	if timeout := resolution.edgeProtocolTimeout(); timeout != 8*time.Second {
		t.Fatalf("cached Edge protocol timeout = %s", timeout)
	}
	fresh, err := newCapabilityRoute(edge, &cloudv1.SignedEnvelope{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if timeout := fresh.edgeProtocolTimeout(); timeout != 15*time.Second {
		t.Fatalf("fresh Edge protocol timeout = %s", timeout)
	}
}

func TestShouldRefreshEdgeLocatorOnlyForStaleOrUnreachableEdge(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "migrated", err: status.Error(codes.NotFound, "daemon moved"), want: true},
		{name: "transport unavailable", err: markEdgeLocatorUnavailable(errors.New("dial timeout")), want: true},
		{name: "generic unavailable", err: status.Error(codes.Unavailable, "relay or edge unavailable")},
		{name: "wrapped unavailable", err: errors.Join(errors.New("exchange"), status.Error(codes.Unavailable, "edge unavailable"))},
		{name: "unauthorized", err: status.Error(codes.Unauthenticated, "grant rejected")},
		{name: "daemon denied", err: status.Error(codes.PermissionDenied, "revoked")},
		{name: "authenticated client rejection", err: &SignalRejectedError{Code: "CLIENT_REVOKED", Message: "client access is not active"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ShouldRefreshEdgeLocator(test.err); got != test.want {
				t.Fatalf("ShouldRefreshEdgeLocator() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestClassifyDaemonLifecycleError(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{name: "blocked", err: status.Error(codes.PermissionDenied, cloudprotocol.DaemonBlockedCode), code: cloudprotocol.DaemonBlockedCode},
		{name: "deleted", err: status.Error(codes.NotFound, cloudprotocol.DaemonDeletedCode), code: cloudprotocol.DaemonDeletedCode},
		{name: "unrelated permission denial", err: status.Error(codes.PermissionDenied, "grant rejected")},
		{name: "unrelated not found", err: status.Error(codes.NotFound, "daemon moved")},
	} {
		t.Run(test.name, func(t *testing.T) {
			classified := classifyDaemonLifecycleError(test.err)
			if got := DaemonLifecycleCode(classified); got != test.code {
				t.Fatalf("DaemonLifecycleCode() = %q, want %q", got, test.code)
			}
			if test.code == "" && classified != test.err {
				t.Fatal("unrelated gRPC error was replaced")
			}
		})
	}
}

func TestClassifyDaemonLifecycleErrorPreservesEntitlementDetail(t *testing.T) {
	failure := &cloudv1.CloudEntitlementFailure{
		Code:    cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE,
		Message: "subscription inactive",
	}
	grpcStatus, err := status.New(codes.PermissionDenied, "Cloud entitlement unavailable").WithDetails(failure)
	if err != nil {
		t.Fatal(err)
	}
	classified := classifyDaemonLifecycleError(grpcStatus.Err())
	got := EntitlementFailure(classified)
	if got.GetCode() != failure.GetCode() || got.GetMessage() != failure.GetMessage() {
		t.Fatalf("entitlement failure = %#v", got)
	}
}
