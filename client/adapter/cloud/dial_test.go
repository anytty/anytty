package cloud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
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

func TestDialCloudRouteHedgesDifferentControllerLocator(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	fresh := testCloudResolution(t, "fresh")
	cachedCanceled := make(chan struct{})

	opened, source, selected, err := dialCloudRouteWithHedge(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			return fresh, nil
		},
		func(ctx context.Context, _ *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			if source == cloudRouteSourceController {
				return &openedCloudPeer{}, nil
			}
			<-ctx.Done()
			close(cachedCanceled)
			return nil, ctx.Err()
		},
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("dialCloudRouteWithHedge() error = %v", err)
	}
	if opened == nil || source != cloudRouteSourceController || selected != fresh {
		t.Fatalf("route result = opened:%p source:%q selected:%p", opened, source, selected)
	}
	select {
	case <-cachedCanceled:
	case <-time.After(time.Second):
		t.Fatal("winning Controller route did not cancel the cached route")
	}
}

func TestDialCloudRouteWinnerDoesNotWaitForLoserCleanup(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	fresh := testCloudResolution(t, "fresh")
	loserCanceled := make(chan struct{})
	releaseLoser := make(chan struct{})
	t.Cleanup(func() { close(releaseLoser) })
	type routeResult struct {
		opened   *openedCloudPeer
		source   cloudRouteSource
		selected *cloudclient.RouteResolution
		err      error
	}
	result := make(chan routeResult, 1)
	go func() {
		opened, source, selected, err := dialCloudRouteWithHedge(
			context.Background(),
			cached,
			func(context.Context) (*cloudclient.RouteResolution, error) { return fresh, nil },
			func(ctx context.Context, _ *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
				if source == cloudRouteSourceController {
					return &openedCloudPeer{}, nil
				}
				<-ctx.Done()
				close(loserCanceled)
				<-releaseLoser
				return nil, ctx.Err()
			},
			nil,
			0,
		)
		result <- routeResult{opened: opened, source: source, selected: selected, err: err}
	}()
	select {
	case got := <-result:
		if got.err != nil || got.opened == nil || got.source != cloudRouteSourceController || got.selected != fresh {
			t.Fatalf("route result = opened:%p source:%q selected:%p err:%v", got.opened, got.source, got.selected, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("winning Controller route waited for cached route cleanup")
	}
	select {
	case <-loserCanceled:
	case <-time.After(time.Second):
		t.Fatal("winning Controller route did not cancel the cached route")
	}
}

func TestDialCloudRouteDoesNotDuplicateEquivalentControllerLocator(t *testing.T) {
	cached := testCloudResolution(t, "edge-a")
	equivalent := testCloudResolution(t, "edge-a")
	resolved := make(chan struct{})
	releaseCached := make(chan struct{})
	var controllerOpens atomic.Int32
	type routeResult struct {
		opened   *openedCloudPeer
		source   cloudRouteSource
		selected *cloudclient.RouteResolution
		err      error
	}
	result := make(chan routeResult, 1)
	go func() {
		opened, source, selected, err := dialCloudRouteWithHedge(
			context.Background(),
			cached,
			func(context.Context) (*cloudclient.RouteResolution, error) {
				close(resolved)
				return equivalent, nil
			},
			func(_ context.Context, _ *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
				if source == cloudRouteSourceController {
					controllerOpens.Add(1)
					return &openedCloudPeer{}, nil
				}
				<-releaseCached
				return &openedCloudPeer{}, nil
			},
			nil,
			0,
		)
		result <- routeResult{opened: opened, source: source, selected: selected, err: err}
	}()
	select {
	case <-resolved:
	case <-time.After(time.Second):
		t.Fatal("Controller resolve did not start")
	}
	close(releaseCached)
	select {
	case got := <-result:
		if got.err != nil || got.opened == nil || got.source != cloudRouteSourceCached || got.selected != cached {
			t.Fatalf("route result = opened:%p source:%q selected:%p err:%v", got.opened, got.source, got.selected, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("cached route did not complete")
	}
	if controllerOpens.Load() != 0 {
		t.Fatalf("equivalent Controller locator opened %d duplicate routes", controllerOpens.Load())
	}
}

func TestDialCloudRouteStopsForPermanentControllerFailure(t *testing.T) {
	cached := testCloudResolution(t, "cached")
	wantErr := status.Error(codes.PermissionDenied, "credential rejected")
	var cachedCanceled atomic.Bool

	opened, source, selected, err := dialCloudRouteWithHedge(
		context.Background(),
		cached,
		func(context.Context) (*cloudclient.RouteResolution, error) {
			return nil, wantErr
		},
		func(ctx context.Context, _ *cloudclient.RouteResolution, _ cloudRouteSource) (*openedCloudPeer, error) {
			<-ctx.Done()
			cachedCanceled.Store(true)
			return nil, ctx.Err()
		},
		nil,
		0,
	)
	if opened != nil || source != cloudRouteSourceController || selected != nil || !errors.Is(err, wantErr) {
		t.Fatalf("route result = opened:%p source:%q selected:%p err:%v", opened, source, selected, err)
	}
	if !cachedCanceled.Load() {
		t.Fatal("permanent Controller failure did not cancel the cached route")
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
