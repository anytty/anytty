package runtime

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
)

func TestAttemptedRuntimeErrorExplainsRouteAndSocket(t *testing.T) {
	tests := []struct {
		name   string
		route  endpoint.AccessRoute
		cause  error
		wants  []string
		unwant string
	}{
		{
			name:  "configured local socket",
			route: endpoint.AccessRoute{ID: "local", Kind: endpoint.RouteLocalUnix, Socket: "/run/anytty-root/daemon.sock"},
			cause: errors.New("connect: no such file or directory"),
			wants: []string{
				`local route "local" failed`,
				`configured Unix socket is "/run/anytty-root/daemon.sock"`,
				`Pass --socket "/run/anytty-root/daemon.sock"`,
				"connect: no such file or directory",
			},
			unwant: "route attempt failed",
		},
		{
			name:  "automatic local socket",
			route: endpoint.AccessRoute{ID: "local", Kind: endpoint.RouteLocalUnix, Socket: "auto"},
			cause: errors.New("socket unavailable"),
			wants: []string{
				`local route "local" has no explicit Unix socket`,
				"pass --socket PATH",
				"endpoints.yaml",
			},
			unwant: "route attempt failed",
		},
		{
			name:  "network route",
			route: endpoint.AccessRoute{ID: "cloud", Kind: endpoint.RouteManagedWebRTC},
			cause: errors.New("controller unavailable"),
			wants: []string{
				`route "cloud" (managed-webrtc) failed`,
				"check this route's configuration",
				"choose another configured route",
				"controller unavailable",
			},
			unwant: "route attempt failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := attemptedRuntimeError(test.route, test.cause).Error()
			for _, want := range test.wants {
				if !strings.Contains(message, want) {
					t.Fatalf("message %q does not contain %q", message, want)
				}
			}
			if strings.Contains(message, test.unwant) {
				t.Fatalf("message %q still contains generic failure text", message)
			}
		})
	}
}

func TestSessionOwnerFullRaceReturnsFirstReadyBeforeIgnoredCancelLoserCleanup(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(false)
	localRelease := make(chan struct{})
	sshRelease := make(chan struct{})
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {release: localRelease, ignoreCancel: true},
		"ssh":   {release: sshRelease},
	})
	resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	if err != nil {
		t.Fatal(err)
	}
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	events, err := owner.WatchEndpoint(watchCtx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan plannedResult, 1)
	go func() {
		lease, err := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
		result <- plannedResult{lease: lease, err: err}
	}()
	waitPlannedStarts(t, dialer.started, "local", "ssh")
	close(sshRelease)
	var connected plannedResult
	select {
	case connected = <-result:
	case <-time.After(time.Second):
		t.Fatal("first authenticated ReadyPeerSession waited for a loser that ignored cancellation")
	}
	if connected.err != nil {
		t.Fatal(connected.err)
	}
	if connected.lease.Stamp.RouteID != "ssh" {
		t.Fatalf("winner = %#v", connected.lease.Stamp)
	}
	application, err := owner.ApplicationSession(connected.lease)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, valid := application.(ConnectionSnapshotProvider).ConnectionSnapshot(time.Now())
	if !valid || snapshot.SelectionReason != "first_ready" {
		t.Fatalf("connection snapshot = %#v valid=%v", snapshot, valid)
	}
	ssh := dialer.session("ssh", 0)
	if local := dialer.session("local", 0); local != nil {
		t.Fatalf("ignored-cancel loser completed before explicit release: %#v", local)
	}
	if ssh == nil || ssh.closeCalls.Load() != 0 {
		t.Fatalf("winner=%#v winnerClose=%d", ssh, closeCalls(ssh))
	}
	phases := collectPhasesThrough(t, events, EndpointPhaseReady)
	if len(phases) > 0 && phases[0] == EndpointPhaseIdle {
		phases = phases[1:]
	}
	if len(phases) < 4 || phases[0] != EndpointPhasePlanning || phases[len(phases)-1] != EndpointPhaseReady {
		t.Fatalf("lifecycle phases = %#v", phases)
	}
	close(localRelease)
	local := waitPlannedSessionClosed(t, dialer, "local", 0)
	if local.closeCalls.Load() != 1 {
		t.Fatalf("late loser close calls = %d", local.closeCalls.Load())
	}
	current, err := owner.ApplicationSession(connected.lease)
	if err != nil || current.Stamp().RouteID != "ssh" {
		t.Fatalf("late loser changed current winner: session=%#v err=%v", current, err)
	}
	if err := ssh.Close(); err != nil {
		t.Fatal(err)
	}
	if phase := <-events; phase.Phase != EndpointPhaseOffline || phase.Stamp != connected.lease.Stamp {
		t.Fatalf("offline event = %#v", phase)
	}
}

func TestDirectSSHAndCloudConnectorsReturnOneReadyPeerSessionContract(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	identity := endpoint.DaemonIdentity{DeviceID: "device-peer", DeviceFingerprint: "SHA256:peer"}
	target := endpoint.Endpoint{
		ID: "peer", Label: "Peer", LabelSource: endpoint.SourceUser, DaemonIdentity: identity,
		ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {
				ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, CredentialRef: "grant:peer",
				Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceUser,
				SignalingAddresses: []string{"peer.local:41120"}, ICETCPAddresses: []string{"peer.local:41121"},
			},
			"ssh": {
				ID: "ssh", Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, CredentialRef: "ssh:peer",
				Source: endpoint.SourceManual, PolicySource: endpoint.SourceUser, Host: "peer-ssh",
				RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121",
			},
			"cloud": {
				ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, CredentialRef: "grant:peer",
				Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser, TargetDeviceID: identity.DeviceID, RelayMode: endpoint.RelayAuto,
			},
		},
	}
	directRelease := make(chan struct{})
	sshRelease := make(chan struct{})
	cloudRelease := make(chan struct{})
	connector := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"direct": {release: directRelease, succeedAfterCancel: true},
		"ssh":    {release: sshRelease, succeedAfterCancel: true},
		"cloud":  {release: cloudRelease},
	})
	connectors, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteDirectWebRTCTCP: routeBoundTestConnector{kind: endpoint.RouteDirectWebRTCTCP, inner: connector},
		endpoint.RouteSSHWebRTCTCP:    routeBoundTestConnector{kind: endpoint.RouteSSHWebRTCTCP, inner: connector},
		endpoint.RouteManagedWebRTC:   routeBoundTestConnector{kind: endpoint.RouteManagedWebRTC, inner: connector},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan plannedResult, 1)
	go func() {
		lease, connectErr := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, RoutePlanEnvironment{
			SupportedRouteKinds:     []endpoint.RouteKind{endpoint.RouteDirectWebRTCTCP, endpoint.RouteSSHWebRTCTCP, endpoint.RouteManagedWebRTC},
			AvailableCredentialRefs: []string{"grant:peer", "ssh:peer"},
		}, realTestClock{}, connectors)
		result <- plannedResult{lease: lease, err: connectErr}
	}()
	waitPlannedStarts(t, connector.started, "direct", "ssh", "cloud")
	close(cloudRelease)
	connected := <-result
	if connected.err != nil || connected.lease.Stamp.RouteID != "cloud" {
		t.Fatalf("connected=%#v err=%v", connected.lease, connected.err)
	}
	direct := waitPlannedSessionClosed(t, connector, "direct", 0)
	ssh := waitPlannedSessionClosed(t, connector, "ssh", 0)
	cloud := connector.session("cloud", 0)
	if closeCalls(direct) != 1 || closeCalls(ssh) != 1 || closeCalls(cloud) != 0 {
		t.Fatalf("close calls direct=%d ssh=%d cloud=%d", closeCalls(direct), closeCalls(ssh), closeCalls(cloud))
	}
}

type routeBoundTestConnector struct {
	kind  endpoint.RouteKind
	inner PeerConnector
}

func (connector routeBoundTestConnector) Connect(ctx context.Context, request AttemptRequest) (ReadyPeerSession, error) {
	if request.Route().Kind != connector.kind {
		return nil, errors.New("connector received a different route kind")
	}
	return connector.inner.Connect(ctx, request)
}

func TestSessionOwnerPriorityHedgeUsesClockAndStopsTimer(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(true)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {release: make(chan struct{})},
		"ssh":   {},
	})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	clock := &manualTestClock{}
	result := make(chan plannedResult, 1)
	go func() {
		lease, err := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, plannedEnvironment(), clock, resolver)
		result <- plannedResult{lease: lease, err: err}
	}()
	if route := waitPlannedStart(t, dialer.started); route != "local" {
		t.Fatalf("first route = %q", route)
	}
	select {
	case route := <-dialer.started:
		t.Fatalf("hedged route started before clock fired: %q", route)
	case <-time.After(20 * time.Millisecond):
	}
	clock.fireAll()
	if route := waitPlannedStart(t, dialer.started); route != "ssh" {
		t.Fatalf("hedged route = %q", route)
	}
	connected := <-result
	if connected.err != nil || connected.lease.Stamp.RouteID != "ssh" {
		t.Fatalf("connected=%#v err=%v", connected.lease, connected.err)
	}
	if !clock.allStopped() {
		t.Fatal("hedge timer was not stopped after firing")
	}
}

func TestSessionOwnerPriorityHedgeStartsNextGroupWhenEarlierGroupFails(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(true)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {err: errors.New("local route unavailable")},
		"ssh":   {},
	})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	clock := &manualTestClock{}
	result := make(chan plannedResult, 1)
	go func() {
		lease, err := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, plannedEnvironment(), clock, resolver)
		result <- plannedResult{lease: lease, err: err}
	}()
	if route := waitPlannedStart(t, dialer.started); route != "local" {
		t.Fatalf("first route = %q", route)
	}
	if route := waitPlannedStart(t, dialer.started); route != "ssh" {
		t.Fatalf("fail-fast hedged route = %q", route)
	}
	connected := <-result
	if connected.err != nil || connected.lease.Stamp.RouteID != "ssh" {
		t.Fatalf("connected=%#v err=%v", connected.lease, connected.err)
	}
	if !clock.allStopped() {
		t.Fatal("expedited hedge timer was not stopped")
	}
}

func TestSessionOwnerPriorityHedgeWaitsForEveryAttemptInEarlierGroup(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(true)
	firstPriority, secondPriority := 10, 20
	direct := endpoint.AccessRoute{
		ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceUser,
		Priority: &firstPriority, CredentialRef: "grant:planned", SignalingAddresses: []string{"planned:41120"}, ICETCPAddresses: []string{"planned:41121"},
	}
	local := target.Routes["local"]
	ssh := target.Routes["ssh"]
	local.Priority, ssh.Priority = &firstPriority, &secondPriority
	target.Routes["local"], target.Routes["direct"], target.Routes["ssh"] = local, direct, ssh
	localRelease := make(chan struct{})
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local":  {release: localRelease},
		"direct": {err: errors.New("direct route unavailable")},
		"ssh":    {},
	})
	resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:       dialer,
		endpoint.RouteDirectWebRTCTCP: dialer,
		endpoint.RouteSSHWebRTCTCP:    dialer,
	})
	if err != nil {
		t.Fatal(err)
	}
	clock := &manualTestClock{}
	result := make(chan plannedResult, 1)
	go func() {
		lease, connectErr := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, RoutePlanEnvironment{
			SupportedRouteKinds:     []endpoint.RouteKind{endpoint.RouteLocalUnix, endpoint.RouteDirectWebRTCTCP, endpoint.RouteSSHWebRTCTCP},
			AvailableCredentialRefs: []string{"grant:planned", "ssh:planned"},
		}, clock, resolver)
		result <- plannedResult{lease: lease, err: connectErr}
	}()
	waitPlannedStarts(t, dialer.started, "local", "direct")
	select {
	case route := <-dialer.started:
		t.Fatalf("next priority group started while an earlier attempt was pending: %q", route)
	case <-time.After(20 * time.Millisecond):
	}
	close(localRelease)
	connected := <-result
	if connected.err != nil || connected.lease.Stamp.RouteID != "local" {
		t.Fatalf("connected=%#v err=%v", connected.lease, connected.err)
	}
	if calls := dialer.calls("ssh"); calls != 0 {
		t.Fatalf("next priority group calls = %d, want 0", calls)
	}
}

func TestSessionOwnerAcquirePlannedSharesLeaseAndKeepsExplicitRouteSticky(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(false)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"local": {}, "ssh": {}})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	first, err := owner.AcquirePlanned(context.Background(), target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.AcquirePlanned(context.Background(), target, "", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stamp() != second.Stamp() || dialer.calls("ssh") != 1 || dialer.calls("local") != 0 {
		t.Fatalf("first=%#v second=%#v calls local=%d ssh=%d", first.Stamp(), second.Stamp(), dialer.calls("local"), dialer.calls("ssh"))
	}
	if err := dialer.session("ssh", 0).Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-first.Done():
	case <-time.After(time.Second):
		t.Fatal("shared lease did not observe ready session close")
	}
	third, err := owner.AcquirePlanned(context.Background(), target, "", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if third.Stamp().RouteID != "ssh" || dialer.calls("ssh") != 2 || dialer.calls("local") != 0 {
		t.Fatalf("sticky reconnect=%#v calls local=%d ssh=%d", third.Stamp(), dialer.calls("local"), dialer.calls("ssh"))
	}
	_ = second.Close()
	_ = third.Close()
}

func TestSessionOwnerAcquirePlannedClearsStickyRouteWhenConfigChanges(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(true)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"local": {}, "ssh": {}})
	resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := owner.AcquirePlanned(context.Background(), target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if err := dialer.session("ssh", 0).Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-first.Done():
	case <-time.After(time.Second):
		t.Fatal("explicit route session did not close")
	}

	reconnected, err := owner.AcquirePlanned(context.Background(), target, "", ConnectIntentInteractive, "config-b", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if reconnected.Stamp().RouteID != "local" || dialer.calls("local") != 1 || dialer.calls("ssh") != 1 {
		t.Fatalf("config change reused stale sticky route: lease=%#v calls local=%d ssh=%d", reconnected.Stamp(), dialer.calls("local"), dialer.calls("ssh"))
	}
	_ = first.Close()
	_ = reconnected.Close()
}

func TestSessionOwnerEnsurePlannedExplicitOverrideReplacesDifferentCurrentWinner(t *testing.T) {
	owner := NewSessionOwner()
	target := plannedEndpoint(false)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"local": {}, "ssh": {}})
	resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix: dialer, endpoint.RouteSSHWebRTCTCP: dialer,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := owner.EnsurePlanned(context.Background(), target, "local", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.EnsurePlanned(context.Background(), target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stamp.RouteID != "local" || second.Stamp.RouteID != "ssh" || second.Stamp.Generation <= first.Stamp.Generation {
		t.Fatalf("explicit override did not replace winner: first=%#v second=%#v", first, second)
	}
}

func TestSessionOwnerEnsurePlannedExplicitCurrentRouteBecomesSticky(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(true)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"local": {}, "ssh": {}})
	resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix: dialer, endpoint.RouteSSHWebRTCTCP: dialer,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := owner.EnsurePlanned(context.Background(), target, "", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	reused, err := owner.EnsurePlanned(context.Background(), target, "local", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if reused.Stamp != first.Stamp || dialer.calls("local") != 1 || dialer.calls("ssh") != 0 {
		t.Fatalf("explicit current route should reuse winner: first=%#v reused=%#v calls local=%d ssh=%d", first, reused, dialer.calls("local"), dialer.calls("ssh"))
	}
	if err := owner.Disconnect(context.Background(), DisconnectRequest{Stamp: reused.Stamp}); err != nil {
		t.Fatal(err)
	}
	reconnected, err := owner.EnsurePlanned(context.Background(), target, "", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if reconnected.Stamp.RouteID != "local" || dialer.calls("local") != 2 || dialer.calls("ssh") != 0 {
		t.Fatalf("explicit current route was not sticky: lease=%#v calls local=%d ssh=%d", reconnected, dialer.calls("local"), dialer.calls("ssh"))
	}
}

func TestSessionOwnerPlannedRaceCancellationFailsAfterAdapterAttempt(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(false)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {release: make(chan struct{})},
		"ssh":   {release: make(chan struct{})},
	})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := owner.ConnectPlanned(ctx, target, "", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
		result <- err
	}()
	waitPlannedStarts(t, dialer.started, "local", "ssh")
	cancel()
	err := <-result
	if CodeOf(err) != ErrorCanceled || !WasAttempted(err) {
		t.Fatalf("cancel error = %#v", err)
	}
	assertEndpointAcquireLocksEmpty(t, owner)
}

func TestSessionOwnerPlannedRaceReturnsStableFirstFailure(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := plannedEndpoint(false)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {err: io.ErrUnexpectedEOF},
		"ssh":   {err: errors.New("ssh unavailable")},
	})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	_, err := owner.ConnectPlanned(context.Background(), target, "", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
	if !errors.Is(err, io.ErrUnexpectedEOF) || !WasAttempted(err) {
		t.Fatalf("race error = %#v", err)
	}
	assertEndpointAcquireLocksEmpty(t, owner)
}

func TestSessionOwnerPlannedRacePrioritizesDaemonLifecycleFailure(t *testing.T) {
	for _, test := range []struct {
		name      string
		localErr  error
		sshErr    error
		wantCode  ErrorCode
		retryable bool
	}{
		{
			name:      "blocked over generic route failure",
			localErr:  io.ErrUnexpectedEOF,
			sshErr:    &Error{Code: ErrorDaemonBlocked, Message: "Cloud disabled", Retryable: true},
			wantCode:  ErrorDaemonBlocked,
			retryable: true,
		},
		{
			name:     "deleted over blocked",
			localErr: &Error{Code: ErrorDaemonBlocked, Message: "Cloud disabled", Retryable: true},
			sshErr:   &Error{Code: ErrorDaemonDeleted, Message: "Cloud enrollment deleted"},
			wantCode: ErrorDaemonDeleted,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			owner := NewSessionOwner()
			defer owner.Close()
			dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
				"local": {err: test.localErr},
				"ssh":   {err: test.sshErr},
			})
			resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
				endpoint.RouteLocalUnix:    dialer,
				endpoint.RouteSSHWebRTCTCP: dialer,
			})
			_, err := owner.ConnectPlanned(context.Background(), plannedEndpoint(false), "", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
			if CodeOf(err) != test.wantCode || !WasAttempted(err) {
				t.Fatalf("race error = %#v, want code %q", err, test.wantCode)
			}
			var runtimeErr *Error
			if !errors.As(err, &runtimeErr) || runtimeErr.Retryable != test.retryable {
				t.Fatalf("retryable = %v, want %v", runtimeErr.Retryable, test.retryable)
			}
			assertEndpointAcquireLocksEmpty(t, owner)
		})
	}
}

func TestSessionOwnerPlannedRaceKeepsSuccessfulRouteOverDaemonLifecycleFailure(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{
		"local": {err: &Error{Code: ErrorDaemonDeleted, Message: "Cloud enrollment deleted"}},
		"ssh":   {},
	})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	lease, err := owner.ConnectPlanned(context.Background(), plannedEndpoint(false), "", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Stamp.RouteID != "ssh" {
		t.Fatalf("winner = %q, want ssh", lease.Stamp.RouteID)
	}
}

func TestSessionOwnerLifecycleMailboxKeepsLatestStateUnderBackpressure(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := owner.WatchEndpoint(ctx, "planned")
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 32; index++ {
		owner.publishEndpointEvent(EndpointEvent{EndpointID: "planned", Phase: EndpointPhaseConnecting})
	}
	owner.publishEndpointEvent(EndpointEvent{EndpointID: "planned", Phase: EndpointPhaseReady})
	var last EndpointEvent
	for len(events) > 0 {
		last = <-events
	}
	if last.Phase != EndpointPhaseReady {
		t.Fatalf("latest mailbox event = %#v", last)
	}
}

func TestClientRuntimeEnsureSessionReusesOwnerWinner(t *testing.T) {
	owner := NewSessionOwner()
	target := plannedEndpoint(false)
	dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"local": {}, "ssh": {}})
	resolver, _ := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{
		endpoint.RouteLocalUnix:    dialer,
		endpoint.RouteSSHWebRTCTCP: dialer,
	})
	source := &staticPlanSource{snapshots: map[endpoint.EndpointID]EndpointPlanSnapshot{
		target.ID: {Endpoint: target, Environment: plannedEnvironment(), ConfigKey: "planned-config"},
	}}
	runtime, err := NewClientRuntime(owner, source, realTestClock{}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	request := ConnectRequest{EndpointID: target.ID, RouteOverride: "ssh", Intent: ConnectIntentInteractive}
	first, err := runtime.EnsureSession(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.EnsureSession(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || dialer.calls("ssh") != 1 || source.calls != 2 {
		t.Fatalf("first=%#v second=%#v dialCalls=%d sourceCalls=%d", first, second, dialer.calls("ssh"), source.calls)
	}
}

func TestSessionOwnerPlannedEntryPointsReclaimEndpointLocks(t *testing.T) {
	type entryPointResult struct {
		stamp   EndpointSessionStamp
		session ApplicationReadyPeerSession
		err     error
	}
	tests := []struct {
		name      string
		wantCalls int
		wantReuse bool
		call      func(*SessionOwner, endpoint.Endpoint, PeerConnectorMap) entryPointResult
	}{
		{
			name: "connect", wantCalls: 2,
			call: func(owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) entryPointResult {
				lease, err := owner.ConnectPlanned(context.Background(), target, "ssh", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
				return entryPointResult{stamp: lease.Stamp, err: err}
			},
		},
		{
			name: "acquire", wantCalls: 1, wantReuse: true,
			call: func(owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) entryPointResult {
				session, err := owner.AcquirePlanned(context.Background(), target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
				result := entryPointResult{session: session, err: err}
				if session != nil {
					result.stamp = session.Stamp()
				}
				return result
			},
		},
		{
			name: "ensure", wantCalls: 1, wantReuse: true,
			call: func(owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) entryPointResult {
				lease, err := owner.EnsurePlanned(context.Background(), target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
				return entryPointResult{stamp: lease.Stamp, err: err}
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			owner := NewSessionOwner()
			defer owner.Close()
			target := plannedEndpoint(false)
			release := make(chan struct{})
			dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"ssh": {release: release}})
			resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{endpoint.RouteSSHWebRTCTCP: dialer})
			if err != nil {
				t.Fatal(err)
			}

			results := make(chan entryPointResult, 2)
			go func() { results <- testCase.call(owner, target, resolver) }()
			if route := waitPlannedStart(t, dialer.started); route != "ssh" {
				t.Fatalf("first route = %q", route)
			}
			go func() { results <- testCase.call(owner, target, resolver) }()
			entry := waitForEndpointAcquireRefs(t, owner, target.ID, 2)
			if calls := dialer.calls("ssh"); calls != 1 {
				t.Fatalf("dial calls with registered waiter = %d, want 1", calls)
			}

			close(release)
			receive := func() entryPointResult {
				select {
				case result := <-results:
					return result
				case <-time.After(time.Second):
					t.Fatal("planned entry point deadlocked after release")
					return entryPointResult{}
				}
			}
			first, second := receive(), receive()
			for _, result := range []entryPointResult{first, second} {
				if result.err != nil {
					t.Fatal(result.err)
				}
				if result.session != nil {
					defer result.session.Close()
				}
			}
			if testCase.wantReuse && first.stamp != second.stamp {
				t.Fatalf("%s stamps = %#v/%#v, want reuse", testCase.name, first.stamp, second.stamp)
			}
			if !testCase.wantReuse && first.stamp.Generation == second.stamp.Generation {
				t.Fatalf("%s generations = %d/%d, want distinct", testCase.name, first.stamp.Generation, second.stamp.Generation)
			}
			if calls := dialer.calls("ssh"); calls != testCase.wantCalls {
				t.Fatalf("final dial calls = %d, want %d", calls, testCase.wantCalls)
			}
			owner.mu.Lock()
			refs := entry.refs
			owner.mu.Unlock()
			if refs != 0 {
				t.Fatalf("released entry refs = %d, want 0", refs)
			}
			assertEndpointAcquireLocksEmpty(t, owner)
		})
	}
}

func TestSessionOwnerPlannedEntryPointsCancelWhileWaitingForEndpoint(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *SessionOwner, endpoint.Endpoint, PeerConnectorMap) error
	}{
		{
			name: "connect",
			call: func(ctx context.Context, owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) error {
				_, err := owner.ConnectPlanned(ctx, target, "ssh", ConnectIntentInteractive, plannedEnvironment(), realTestClock{}, resolver)
				return err
			},
		},
		{
			name: "acquire",
			call: func(ctx context.Context, owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) error {
				_, err := owner.AcquirePlanned(ctx, target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
				return err
			},
		},
		{
			name: "ensure",
			call: func(ctx context.Context, owner *SessionOwner, target endpoint.Endpoint, resolver PeerConnectorMap) error {
				_, err := owner.EnsurePlanned(ctx, target, "ssh", ConnectIntentInteractive, "config-a", plannedEnvironment(), realTestClock{}, resolver)
				return err
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			owner := NewSessionOwner()
			target := plannedEndpoint(false)
			dialer := newPlannedDialer(map[endpoint.RouteID]*plannedBehavior{"ssh": {}})
			resolver, err := NewPeerConnectorMap(map[endpoint.RouteKind]PeerConnector{endpoint.RouteSSHWebRTCTCP: dialer})
			if err != nil {
				t.Fatal(err)
			}
			holderUnlock, err := owner.acquireEndpoint(context.Background(), target.ID)
			if err != nil {
				t.Fatal(err)
			}
			holderReleased := false
			t.Cleanup(func() {
				if !holderReleased {
					holderUnlock()
				}
				_ = owner.Close()
			})

			ctx := newAcquireTestContext(false)
			result := make(chan error, 1)
			go func() { result <- testCase.call(ctx, owner, target, resolver) }()
			<-ctx.observed
			entry := endpointAcquireEntryForTest(t, owner, target.ID, 2)
			ctx.finish(context.Canceled)
			select {
			case err := <-result:
				if CodeOf(err) != ErrorCanceled || !errors.Is(err, context.Canceled) || WasAttempted(err) {
					t.Fatalf("gate cancellation error = %v code=%q attempted=%v", err, CodeOf(err), WasAttempted(err))
				}
			case <-time.After(time.Second):
				t.Fatal("gate cancellation did not wake planned entry point")
			}

			owner.authority.mu.Lock()
			generation, generated := owner.authority.generations[target.ID]
			owner.authority.mu.Unlock()
			owner.mu.Lock()
			current := owner.acquireLocks[target.ID]
			refs := entry.refs
			owner.mu.Unlock()
			if generated || generation != 0 {
				t.Fatalf("generation after gate cancellation = %d present=%v, want absent", generation, generated)
			}
			if current != entry || refs != 1 {
				t.Fatalf("after gate cancellation: entry=%p refs=%d, want entry=%p refs=1", current, refs, entry)
			}
			if calls := dialer.calls("ssh"); calls != 0 {
				t.Fatalf("connector calls after gate cancellation = %d, want 0", calls)
			}

			holderUnlock()
			holderReleased = true
			assertEndpointAcquireLocksEmpty(t, owner)
		})
	}
}

type plannedResult struct {
	lease SessionLease
	err   error
}

type staticPlanSource struct {
	mu        sync.Mutex
	snapshots map[endpoint.EndpointID]EndpointPlanSnapshot
	calls     int
}

func (source *staticPlanSource) Snapshot(_ context.Context, endpointID endpoint.EndpointID) (EndpointPlanSnapshot, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.calls++
	snapshot, ok := source.snapshots[endpointID]
	if !ok {
		return EndpointPlanSnapshot{}, runtimeError(ErrorNotFound, "endpoint not found", nil)
	}
	return snapshot, nil
}

type plannedBehavior struct {
	release            chan struct{}
	err                error
	succeedAfterCancel bool
	ignoreCancel       bool
}

type plannedDialer struct {
	mu        sync.Mutex
	behaviors map[endpoint.RouteID]*plannedBehavior
	started   chan endpoint.RouteID
	sessions  map[endpoint.RouteID][]*ownerSession
	counts    map[endpoint.RouteID]int
}

func newPlannedDialer(behaviors map[endpoint.RouteID]*plannedBehavior) *plannedDialer {
	return &plannedDialer{behaviors: behaviors, started: make(chan endpoint.RouteID, 16), sessions: make(map[endpoint.RouteID][]*ownerSession), counts: make(map[endpoint.RouteID]int)}
}

func (dialer *plannedDialer) Connect(ctx context.Context, request AttemptRequest) (ReadyPeerSession, error) {
	routeID := request.Route().ID
	dialer.mu.Lock()
	behavior := dialer.behaviors[routeID]
	dialer.counts[routeID]++
	dialer.mu.Unlock()
	dialer.started <- routeID
	if behavior == nil {
		return nil, errors.New("missing planned behavior")
	}
	if behavior.release != nil {
		if behavior.ignoreCancel {
			<-behavior.release
		} else {
			select {
			case <-behavior.release:
			case <-ctx.Done():
				if !behavior.succeedAfterCancel {
					return nil, ctx.Err()
				}
			}
		}
	}
	if behavior.err != nil {
		return nil, behavior.err
	}
	session := newOwnerSession(request.Stamp())
	session.evidence = ReadyPeerSessionEvidence{Identity: request.DaemonIdentity(), IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: 1}
	dialer.mu.Lock()
	dialer.sessions[routeID] = append(dialer.sessions[routeID], session)
	dialer.mu.Unlock()
	return session, nil
}

func (dialer *plannedDialer) session(routeID endpoint.RouteID, index int) *ownerSession {
	dialer.mu.Lock()
	defer dialer.mu.Unlock()
	if index >= len(dialer.sessions[routeID]) {
		return nil
	}
	return dialer.sessions[routeID][index]
}

func (dialer *plannedDialer) calls(routeID endpoint.RouteID) int {
	dialer.mu.Lock()
	defer dialer.mu.Unlock()
	return dialer.counts[routeID]
}

func waitPlannedSessionClosed(t *testing.T, dialer *plannedDialer, routeID endpoint.RouteID, index int) *ownerSession {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if session := dialer.session(routeID, index); session != nil && session.closeCalls.Load() > 0 {
			return session
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("route %q late session was not closed", routeID)
	return nil
}

func plannedEndpoint(priority bool) endpoint.Endpoint {
	identity := endpoint.DaemonIdentity{DeviceID: "device-planned", DeviceFingerprint: "SHA256:device-planned"}
	target := endpoint.Endpoint{
		ID: "planned", Label: "Planned", LabelSource: endpoint.SourceUser, DaemonIdentity: identity,
		ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"local": {ID: "local", Kind: endpoint.RouteLocalUnix, Enabled: true, Source: endpoint.SourceLocal, PolicySource: endpoint.SourceUser, Socket: "auto"},
			"ssh":   {ID: "ssh", Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, Source: endpoint.SourceManual, PolicySource: endpoint.SourceUser, Host: "planned", RemoteSignalingAddress: "127.0.0.1:41120", RemoteICETCPAddress: "127.0.0.1:41121", CredentialRef: "ssh:planned"},
		},
	}
	if priority {
		first, second := 10, 20
		local, ssh := target.Routes["local"], target.Routes["ssh"]
		local.Priority, ssh.Priority = &first, &second
		target.Routes["local"], target.Routes["ssh"] = local, ssh
		target.SelectionPolicy = endpoint.SelectionPolicy{HedgeDelay: 250 * time.Millisecond, HedgeDelayConfigured: true}
	}
	return target
}

func plannedEnvironment() RoutePlanEnvironment {
	return RoutePlanEnvironment{SupportedRouteKinds: []endpoint.RouteKind{endpoint.RouteLocalUnix, endpoint.RouteSSHWebRTCTCP}, AvailableCredentialRefs: []string{"ssh:planned"}}
}

func waitPlannedStarts(t *testing.T, started <-chan endpoint.RouteID, expected ...endpoint.RouteID) {
	t.Helper()
	seen := make(map[endpoint.RouteID]bool, len(expected))
	for len(seen) < len(expected) {
		seen[waitPlannedStart(t, started)] = true
	}
	for _, routeID := range expected {
		if !seen[routeID] {
			t.Fatalf("route %q did not start: %#v", routeID, seen)
		}
	}
}

func waitPlannedStart(t *testing.T, started <-chan endpoint.RouteID) endpoint.RouteID {
	t.Helper()
	select {
	case routeID := <-started:
		return routeID
	case <-time.After(time.Second):
		t.Fatal("route attempt did not start")
		return ""
	}
}

func collectPhasesThrough(t *testing.T, events <-chan EndpointEvent, final EndpointPhase) []EndpointPhase {
	t.Helper()
	phases := make([]EndpointPhase, 0, 4)
	for {
		select {
		case event := <-events:
			phases = append(phases, event.Phase)
			if event.Phase == final {
				return phases
			}
		case <-time.After(time.Second):
			t.Fatalf("lifecycle phases timed out: %#v", phases)
		}
	}
}

func closeCalls(session *ownerSession) int32 {
	if session == nil {
		return -1
	}
	return session.closeCalls.Load()
}

type realTestClock struct{}

func (realTestClock) Now() time.Time { return time.Now() }
func (realTestClock) NewTimer(delay time.Duration) port.Timer {
	return &realTestTimer{timer: time.NewTimer(delay)}
}

type realTestTimer struct{ timer *time.Timer }

func (timer *realTestTimer) C() <-chan time.Time { return timer.timer.C }
func (timer *realTestTimer) Stop() bool          { return timer.timer.Stop() }

type manualTestClock struct {
	mu     sync.Mutex
	timers []*manualTestTimer
}

func (clock *manualTestClock) Now() time.Time { return time.Unix(1, 0) }
func (clock *manualTestClock) NewTimer(time.Duration) port.Timer {
	timer := &manualTestTimer{channel: make(chan time.Time, 1)}
	clock.mu.Lock()
	clock.timers = append(clock.timers, timer)
	clock.mu.Unlock()
	return timer
}
func (clock *manualTestClock) fireAll() {
	clock.mu.Lock()
	timers := append([]*manualTestTimer(nil), clock.timers...)
	clock.mu.Unlock()
	for _, timer := range timers {
		timer.channel <- clock.Now()
	}
}
func (clock *manualTestClock) allStopped() bool {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	for _, timer := range clock.timers {
		if !timer.isStopped() {
			return false
		}
	}
	return len(clock.timers) > 0
}

type manualTestTimer struct {
	mu      sync.Mutex
	channel chan time.Time
	stopped bool
}

func (timer *manualTestTimer) C() <-chan time.Time { return timer.channel }
func (timer *manualTestTimer) Stop() bool {
	timer.mu.Lock()
	defer timer.mu.Unlock()
	wasActive := !timer.stopped
	timer.stopped = true
	return wasActive
}
func (timer *manualTestTimer) isStopped() bool {
	timer.mu.Lock()
	defer timer.mu.Unlock()
	return timer.stopped
}
