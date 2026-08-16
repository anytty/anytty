package app

import (
	"context"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
)

type terminalEventWatchService struct {
	*testkit.FakeTerminalService
	events     chan port.TerminalLiveEvent
	endpointID state.EndpointID
}

func (service *terminalEventWatchService) WatchTerminalEvents(_ context.Context, endpointID state.EndpointID) (<-chan port.TerminalLiveEvent, error) {
	service.endpointID = endpointID
	return service.events, nil
}

func TestTerminalEventWatcherProjectsRemoteViewCountAndOwner(t *testing.T) {
	service := &terminalEventWatchService{
		FakeTerminalService: &testkit.FakeTerminalService{},
		events:              make(chan port.TerminalLiveEvent, 1),
	}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: service})
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			EndpointID: "west", TerminalID: "term-1", AttachmentCount: 1,
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(
		"west", state.DefaultPaneID, "term-1", 7, 80, 24,
		state.TerminalResizeRoleFollower, "surface-a", "pane:one", false,
	))

	_, effects := reducer(root, TerminalEventWatchRequestMsg{EndpointID: "west"})
	if len(effects) != 1 {
		t.Fatalf("expected one terminal event stream, got %#v", effects)
	}
	stream, ok := effects[0].(StreamEffect)
	if !ok || stream.Token != CancelToken(terminalEventWatchTokenPrefix+"west") {
		t.Fatalf("unexpected terminal event stream: %#v", effects[0])
	}
	ctx, cancel := context.WithCancel(context.Background())
	posted := make(chan Msg, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Run(ctx, func(msg Msg) { posted <- msg })
	}()
	service.events <- port.TerminalLiveEvent{
		EndpointID: "west", TerminalID: "term-1", Metadata: true,
		AttachmentProjection: true, AttachmentCount: 2,
		OwnerSurfaceID: "surface-a", OwnerViewID: "pane:one", ResizeEpoch: 5,
	}

	var msg Msg
	select {
	case msg = <-posted:
	case <-time.After(time.Second):
		t.Fatal("terminal event watcher did not forward the daemon event")
	}
	cancel()
	<-done
	if service.endpointID != "west" {
		t.Fatalf("watcher subscribed to wrong endpoint: %q", service.endpointID)
	}
	next, _ := reducer(root, msg)
	if next.TerminalPool.Items[0].AttachmentCount != 2 {
		t.Fatalf("watch event did not project attached view count: %#v", next.TerminalPool.Items[0])
	}
	binding, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !binding.HasProjectedResizeOwner() || !binding.CanResize || binding.ResizeEpoch != 5 {
		t.Fatalf("watch event did not project resize owner: %#v", binding)
	}
}
