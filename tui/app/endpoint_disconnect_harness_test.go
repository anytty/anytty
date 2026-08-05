package app

import (
	"context"
	"errors"
	"github.com/anytty/anytty/tui/testkit"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestEndpointDisconnectHarnessKeepsPaneReasonAfterEmptyInventory(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	source := newEndpointDisconnectHarnessSource()
	runner := NewAsyncEffectRunner()
	t.Cleanup(func() {
		source.close()
		runner.Cancel(endpointStatusWatchToken)
		runner.Cancel(terminalPoolRefreshToken)
	})
	root := endpointDisconnectHarnessRoot(ref)
	host := NewFakeTerminalHost(64)
	host.SetSize(120, 32)
	runtime := NewInteractiveRuntime(root, host, runner, LiveDeps{
		EndpointEvents: source,
		Terminal:       &testkit.FakeTerminalService{},
	}, CopyModeDeps{})

	ctx := context.Background()
	if err := runtime.Drain(ctx); err != nil {
		t.Fatalf("initial drain failed: %v", err)
	}
	source.waitSubscribed(t)
	errText := "ssh transport closed: exit status 255: stdio-proxy connect core-v2 daemon socket: connection refused"
	source.emit(t, port.EndpointRuntimeEvent{
		EndpointID: ref.EndpointID,
		Status:     state.EndpointStatusOffline,
		ErrorKind:  state.EndpointErrorRemoteDaemon,
		Message:    errText,
	})

	if err := drainUntilFrameContains(ctx, runtime, host, "Connection interrupted"); err != nil {
		t.Fatalf("disconnect frame was not rendered: %v\nframes=%s", err, endpointDisconnectHarnessFrames(host))
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "last remote output") ||
		!frameContains(frame, "Remote anytty daemon unavailable") ||
		!frameContains(frame, "detail") ||
		!frameContains(frame, "next step") ||
		!frameContains(frame, "Reconnect this pane") ||
		!frameContains(frame, "Disconnect pane") ||
		frameContains(frame, "Attach existing terminal") {
		t.Fatalf("disconnect frame should show local reason/actions instead of unconnected CTA, frame=%#v", frame.Lines)
	}

	if err := runtime.Post(TerminalPoolListResultMsg{EndpointID: ref.EndpointID, Seq: runtime.State().TerminalPool.RequestSeq, Refresh: true, Result: port.TerminalListResult{Items: nil}}); err != nil {
		t.Fatalf("post empty inventory failed: %v", err)
	}
	if err := runtime.Drain(ctx); err != nil {
		t.Fatalf("empty inventory drain failed: %v", err)
	}
	final := runtime.State()
	if binding, ok := final.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !binding.TerminalRef().Equal(ref) || !strings.Contains(binding.LastError, "remote-daemon") {
		t.Fatalf("empty inventory must not delete disconnected pane binding, binding=%#v ok=%v", binding, ok)
	}
	if _, ok := terminalPoolItemRef(final.TerminalPool, ref); !ok {
		t.Fatalf("empty inventory must keep last known remote row while endpoint is offline, pool=%#v", final.TerminalPool.Items)
	}
	if west, ok := final.Endpoints.Endpoint(ref.EndpointID); !ok || west.DisplayStatus() != state.EndpointStatusOffline || west.LastErrorKind != state.EndpointErrorRemoteDaemon || west.LastError == "" {
		t.Fatalf("empty inventory must not clear endpoint runtime error, west=%#v ok=%v", west, ok)
	}
	if final.Session.State != state.TerminalLiveError || !final.Session.TerminalRef().Equal(ref) || !strings.Contains(final.Session.LastError, "remote-daemon") {
		t.Fatalf("session should retain endpoint disconnect reason, session=%#v", final.Session)
	}
	if final.Surface.State != state.TerminalLiveError || !final.Surface.TerminalRef().Equal(ref) || !strings.Contains(final.Surface.Err, "remote-daemon") {
		t.Fatalf("surface should retain endpoint disconnect reason, surface=%#v", final.Surface)
	}
	frame = lastFrame(t, host.Frames())
	if !frameContains(frame, "Connection interrupted") ||
		!frameContains(frame, "last terminal frame is preserved") ||
		!frameContains(frame, "next step") ||
		!frameContains(frame, "Reconnect this pane") ||
		!frameContains(frame, "Disconnect pane") ||
		frameContains(frame, "Attach existing terminal") {
		t.Fatalf("empty inventory should keep disconnected pane content, frame=%#v", frame.Lines)
	}
}

func TestLiveScreenNextEOFHarnessShowsDisconnectedPane(t *testing.T) {
	ref := state.NewTerminalRef("cn-fast", "11")
	root := endpointDisconnectHarnessRoot(ref)
	host := NewFakeTerminalHost(64)
	host.SetSize(120, 32)
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: &testkit.FakeTerminalService{}}, CopyModeDeps{})

	if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Err:        errors.New("EOF"),
	}}); err != nil {
		t.Fatalf("post live EOF failed: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("live EOF drain failed: %v", err)
	}

	final := runtime.State()
	if binding, ok := final.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !binding.TerminalRef().Equal(ref) || binding.Attached || binding.Channel != 0 || binding.LastError != "EOF" {
		t.Fatalf("live EOF should mark pane binding as disconnected, binding=%#v ok=%v", binding, ok)
	}
	if endpoint, ok := final.Endpoints.Endpoint(ref.EndpointID); !ok || endpoint.DisplayStatus() != state.EndpointStatusOffline || endpoint.LastErrorKind != state.EndpointErrorProtocol || endpoint.LastError != "EOF" {
		t.Fatalf("live EOF should mark endpoint offline protocol, endpoint=%#v ok=%v", endpoint, ok)
	}
	if final.Session.State != state.TerminalLiveError || final.Session.LastError != "EOF" || final.Session.Attached {
		t.Fatalf("live EOF should mark active session error, session=%#v", final.Session)
	}
	if final.Surface.State != state.TerminalLiveError || final.Surface.Err != "EOF" {
		t.Fatalf("live EOF should mark active surface error, surface=%#v", final.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "Connection interrupted") ||
		!frameContains(frame, "protocol session") ||
		!frameContains(frame, "protocol") ||
		!frameContains(frame, "Reconnect this pane") ||
		!frameContains(frame, "Disconnect pane") ||
		frameContains(frame, "Attach existing terminal") {
		t.Fatalf("live EOF frame should show disconnected pane content, frame=%#v", frame.Lines)
	}
}

func endpointDisconnectHarnessRoot(ref state.TerminalRef) state.Root {
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID)
	return state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{
			EndpointID:    ref.EndpointID,
			TerminalID:    ref.TerminalID,
			Channel:       9,
			Attached:      true,
			InputChannels: map[string]uint16{ref.Key(): 9},
			State:         state.TerminalLiveAttached,
			ResizePolicy:  state.TerminalResizeRoleOwner,
			SurfaceID:     "surface-west",
			ViewID:        state.TerminalPaneViewID(state.DefaultPaneID),
			DesiredCols:   100,
			DesiredRows:   30,
		},
		Surface: state.TerminalSurfaceStore{
			EndpointID: ref.EndpointID,
			TerminalID: ref.TerminalID,
			Lines:      []string{"last remote output"},
			State:      state.TerminalLiveAttached,
			Ready:      true,
			Cols:       100,
			Rows:       30,
		},
		TerminalPool: state.TerminalPoolStore{
			RequestSeq: 4,
			Status:     state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Title: "remote", State: "running", Cols: 100, Rows: 30},
			},
		},
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: ref.EndpointID, Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		TerminalViews: (state.TerminalViewStore{}).
			BindPane(state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 9, 100, 30, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true)),
	}
}

type endpointDisconnectHarnessSource struct {
	events        chan port.EndpointRuntimeEvent
	subscribed    chan struct{}
	subscribeOnce sync.Once
	closeOnce     sync.Once
}

func newEndpointDisconnectHarnessSource() *endpointDisconnectHarnessSource {
	return &endpointDisconnectHarnessSource{
		events:     make(chan port.EndpointRuntimeEvent, 4),
		subscribed: make(chan struct{}),
	}
}

func (source *endpointDisconnectHarnessSource) WatchEndpointEvents(context.Context) (<-chan port.EndpointRuntimeEvent, error) {
	source.subscribeOnce.Do(func() {
		close(source.subscribed)
	})
	return source.events, nil
}

func (source *endpointDisconnectHarnessSource) waitSubscribed(t *testing.T) {
	t.Helper()
	select {
	case <-source.subscribed:
	case <-time.After(time.Second):
		t.Fatal("endpoint event source was not subscribed")
	}
}

func (source *endpointDisconnectHarnessSource) emit(t *testing.T, event port.EndpointRuntimeEvent) {
	t.Helper()
	select {
	case source.events <- event:
	case <-time.After(time.Second):
		t.Fatal("endpoint event source blocked while emitting")
	}
}

func (source *endpointDisconnectHarnessSource) close() {
	source.closeOnce.Do(func() {
		close(source.events)
	})
}

func endpointDisconnectHarnessFrames(host *FakeTerminalHost) string {
	frames := host.Frames()
	if len(frames) == 0 {
		return "<none>"
	}
	var builder strings.Builder
	for _, line := range frames[len(frames)-1].Lines {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return builder.String()
}
