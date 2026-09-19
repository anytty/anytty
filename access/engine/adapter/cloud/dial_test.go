package cloud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	cloudclient "github.com/anytty/anytty/access/transport/client"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCloudPeerCredentialRejectionDoesNotRediscoverEdge(t *testing.T) {
	err := &remoteauth.HandshakeError{Code: remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_REVOKED, Detail: "grant revoked", Cause: remoteauth.ErrGrantRevoked}
	if shouldRefreshCloudRoute(fmt.Errorf("Cloud Relay-TCP attempt: %w", err)) {
		t.Fatal("peer credential rejection triggered locator refresh")
	}
}

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

func TestDialCloudRouteMissingCacheQueriesController(t *testing.T) {
	fresh := testCloudResolution(t, "fresh")
	var calls int
	opened, source, selected, err := dialCloudRoute(context.Background(), nil,
		func(context.Context) (*cloudclient.RouteResolution, error) { calls++; return fresh, nil },
		func(_ context.Context, route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			if calls != 1 || route != fresh || source != cloudRouteSourceController {
				t.Fatal("unexpected fallback order")
			}
			return &openedCloudPeer{}, nil
		}, nil)
	if err != nil || opened == nil || selected != fresh || source != cloudRouteSourceController || calls != 1 {
		t.Fatalf("missing cache result: source=%s calls=%d error=%v", source, calls, err)
	}
}

func TestDialCloudRouteRetriesEquivalentLocatorOnlyAfterFailure(t *testing.T) {
	cached := testCloudResolution(t, "edge-a")
	equivalent := testCloudResolution(t, "edge-a")
	var opens, resolves int
	opened, source, _, err := dialCloudRoute(context.Background(), cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			if opens != 1 {
				t.Fatal("resolved before cached attempt finished")
			}
			resolves++
			return equivalent, nil
		},
		func(_ context.Context, route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			opens++
			if source == cloudRouteSourceCached {
				return nil, errors.New("cached exchange timed out")
			}
			if resolves != 1 || route != equivalent {
				t.Fatal("invalid fallback")
			}
			return &openedCloudPeer{}, nil
		}, nil)
	if err != nil || opened == nil || source != cloudRouteSourceController || opens != 2 || resolves != 1 {
		t.Fatalf("equivalent fallback: source=%s opens=%d resolves=%d error=%v", source, opens, resolves, err)
	}
}

func TestDialCloudRouteStopsForPermanentControllerFailure(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	wantErr := status.Error(codes.PermissionDenied, "credential rejected")
	opened, source, selected, err := dialCloudRoute(context.Background(), cached,
		func(context.Context) (*cloudclient.RouteResolution, error) { return nil, wantErr },
		func(context.Context, *cloudclient.RouteResolution, cloudRouteSource) (*openedCloudPeer, error) {
			return nil, errors.New("cached route failed")
		}, nil)
	if opened != nil || source != cloudRouteSourceController || selected != nil || !errors.Is(err, wantErr) {
		t.Fatalf("fallback rejection: source=%s error=%v", source, err)
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

func TestDialCloudRouteSlowCachedSuccessNeverQueriesController(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	var calls atomic.Int32
	opened, source, _, err := dialCloudRoute(context.Background(), cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			calls.Add(1)
			return nil, status.Error(codes.Unavailable, "controller offline")
		},
		func(context.Context, *cloudclient.RouteResolution, cloudRouteSource) (*openedCloudPeer, error) {
			time.Sleep(time.Second)
			return &openedCloudPeer{}, nil
		}, nil)
	if err != nil || opened == nil || source != cloudRouteSourceCached {
		t.Fatalf("cached connection failed: source=%s error=%v", source, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("healthy slow cache triggered %d Controller calls", calls.Load())
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
	var runtimeErr *clientruntime.Error
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != clientruntime.ErrorUnavailable || !runtimeErr.Retryable ||
		!strings.Contains(runtimeErr.Message, "cached Cloud route failed") || !strings.Contains(runtimeErr.Message, "Controller Cloud route failed") {
		t.Fatalf("combined route failure = %#v", err)
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
