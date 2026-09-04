package cloud

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDialCloudRouteRefreshesOnceAfterCachedRouteFailure(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	fresh := testCloudResolution(t, "fresh")
	var resolveCalls atomic.Int32
	var openedSources []cloudRouteSource

	opened, source, selected, err := dialCloudRoute(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			resolveCalls.Add(1)
			return fresh, nil
		},
		func(_ context.Context, route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			openedSources = append(openedSources, source)
			if route == cached {
				return nil, errors.New("cached ICE exchange failed")
			}
			return &openedCloudPeer{}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("dialCloudRoute() error = %v", err)
	}
	if opened == nil || source != cloudRouteSourceController || selected != fresh {
		t.Fatalf("route result = opened:%p source:%q selected:%p", opened, source, selected)
	}
	if resolveCalls.Load() != 1 {
		t.Fatalf("resolve calls = %d, want 1", resolveCalls.Load())
	}
	if got := fmt.Sprint(openedSources); got != "[cached controller]" {
		t.Fatalf("opened sources = %s, want [cached controller]", got)
	}
}

func TestDialCloudRouteDoesNotResolveWhenCachedRouteSucceeds(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	var resolveCalls atomic.Int32
	opened, source, selected, err := dialCloudRoute(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			resolveCalls.Add(1)
			return nil, errors.New("resolver must not be called")
		},
		func(_ context.Context, route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			if source != cloudRouteSourceCached || route != cached {
				t.Fatalf("cached callback received route %p source %q", route, source)
			}
			return &openedCloudPeer{}, nil
		},
		nil,
	)
	if err != nil || opened == nil || source != cloudRouteSourceCached || selected != cached {
		t.Fatalf("route result = opened:%p source:%q selected:%p err:%v", opened, source, selected, err)
	}
	if resolveCalls.Load() != 0 {
		t.Fatalf("resolve calls = %d, want 0", resolveCalls.Load())
	}
}

func TestDialCloudRouteDoesNotResolvePermanentCachedFailure(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	var resolveCalls atomic.Int32
	wantErr := status.Error(codes.PermissionDenied, "credential rejected")
	_, source, selected, err := dialCloudRoute(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			resolveCalls.Add(1)
			return nil, nil
		},
		func(context.Context, *cloudclient.RouteResolution, cloudRouteSource) (*openedCloudPeer, error) {
			return nil, wantErr
		},
		nil,
	)
	if !errors.Is(err, wantErr) || source != cloudRouteSourceCached || selected != cached {
		t.Fatalf("route result = source:%q selected:%p err:%v", source, selected, err)
	}
	if resolveCalls.Load() != 0 {
		t.Fatalf("resolve calls = %d, want 0", resolveCalls.Load())
	}
}

func TestShouldRefreshCloudRouteDoesNotRefreshAuthenticatedSignalRejection(t *testing.T) {
	if shouldRefreshCloudRoute(&cloudclient.SignalRejectedError{Code: "CLIENT_REVOKED"}) {
		t.Fatal("authenticated Cloud rejection must not trigger locator refresh")
	}
}

func TestDialCloudRouteStopsOnCancellationAfterCachedFailure(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var resolveCalls atomic.Int32
	wantErr := errors.New("cached route failed")
	_, source, selected, err := dialCloudRoute(
		ctx,
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			resolveCalls.Add(1)
			return nil, nil
		},
		func(context.Context, *cloudclient.RouteResolution, cloudRouteSource) (*openedCloudPeer, error) {
			cancel()
			return nil, wantErr
		},
		nil,
	)
	if !errors.Is(err, wantErr) || source != cloudRouteSourceCached || selected != cached {
		t.Fatalf("route result = source:%q selected:%p err:%v", source, selected, err)
	}
	if resolveCalls.Load() != 0 {
		t.Fatalf("resolve calls = %d, want 0", resolveCalls.Load())
	}
}

func TestDialCloudRouteReturnsFreshRouteFailureWithoutLooping(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	fresh := testCloudResolution(t, "fresh")
	var resolveCalls atomic.Int32
	var openCalls atomic.Int32
	wantErr := errors.New("fresh ICE exchange failed")
	_, source, selected, err := dialCloudRoute(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			resolveCalls.Add(1)
			return fresh, nil
		},
		func(_ context.Context, route *cloudclient.RouteResolution, _ cloudRouteSource) (*openedCloudPeer, error) {
			openCalls.Add(1)
			if route == cached {
				return nil, errors.New("cached route failed")
			}
			return nil, wantErr
		},
		nil,
	)
	if !errors.Is(err, wantErr) || source != cloudRouteSourceController || selected != fresh {
		t.Fatalf("route result = source:%q selected:%p err:%v", source, selected, err)
	}
	if resolveCalls.Load() != 1 || openCalls.Load() != 2 {
		t.Fatalf("resolve calls = %d, open calls = %d, want 1 and 2", resolveCalls.Load(), openCalls.Load())
	}
}

func TestShouldRefreshCloudRouteBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "ice failure", err: errors.New("ICE failed"), want: true},
		{name: "edge unavailable", err: status.Error(codes.Unavailable, "edge unavailable"), want: true},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "grant rejected"), want: false},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "grant rejected"), want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "deadline", err: context.DeadlineExceeded, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRefreshCloudRoute(test.err); got != test.want {
				t.Fatalf("shouldRefreshCloudRoute(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func testCloudResolution(t *testing.T, edgeID string) *cloudclient.RouteResolution {
	t.Helper()
	resolution, err := cloudclient.NewCachedRoute(
		&cloudv1.EdgeLocator{EdgeId: edgeID, PublicEndpoint: "edge.example:443", ServerName: "edge.example", CaCertificatePem: []byte("ca")},
		&cloudv1.SignedEnvelope{KeyId: "daemon-key", Payload: []byte("grant"), Signature: []byte("signature")},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolution
}
