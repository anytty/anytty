package app

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestTerminalDefaultResizePolicyUsesFollowerForPassiveAttach(t *testing.T) {
	root := state.Root{Config: state.TUIConfigStore{Terminal: state.TUITerminalConfig{AutoTakeOwner: true}}}
	if got := terminalDefaultResizePolicy(root); got != state.TerminalResizeRoleFollower {
		t.Fatalf("passive attach policy = %q, want follower", got)
	}
}

func TestAutoTakeOwnerDoesNotReclaimExternalOwnerProjection(t *testing.T) {
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "tui-surface", "tui-view", true)
	binding.OwnerSurfaceID = binding.SurfaceID
	binding.OwnerViewID = binding.ViewID
	binding.ResizeEpoch = 4
	binding.ResizePending = true
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Config:   state.TUIConfigStore{Terminal: state.TUITerminalConfig{AutoTakeOwner: true}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	reducer := ComposeReducers(newLiveReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	next, effects := reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{
		TerminalID:           "term-1",
		Metadata:             true,
		AttachmentProjection: true,
		AttachmentCount:      2,
		OwnerSurfaceID:       "app-surface",
		OwnerViewID:          "primary",
		ResizeEpoch:          5,
	}})
	if msg, ok := liveResizeMsgFromEffects(effects); ok && msg.TakeOwnership {
		t.Fatalf("external owner projection triggered automatic reclaim: %#v", msg)
	}
	got, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if got.HasResizeOwner() || got.OwnerSurfaceID != "app-surface" || got.OwnerViewID != "primary" || got.ResizeEpoch != 5 {
		t.Fatalf("external owner projection was not retained: %#v", got)
	}
}

func TestAutoTakeOwnerRequestsOnlyVacantProjection(t *testing.T) {
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "tui-surface", "tui-view", false)
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Config:   state.TUIConfigStore{Terminal: state.TUITerminalConfig{AutoTakeOwner: true}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	effects, ok := autoTakeOwnerEffects(root)
	if !ok || len(effects) != 1 {
		t.Fatalf("vacant owner projection should request ownership, effects=%#v ok=%v", effects, ok)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveResizeMsg)
	if !ok || !msg.TakeOwnership || msg.ViewID != "tui-view" || msg.ExpectedOwnerEpoch != 0 {
		t.Fatalf("unexpected vacant-owner request: %#v", msg)
	}
}

func TestLiveAttachMsgForResizeOwnerPreservesRemoteEndpoint(t *testing.T) {
	binding := state.NewEndpointPaneTerminalView("west", "pane-remote", "term-1", 0, 80, 24, state.TerminalResizeRoleFollower, "surface", "view-remote", false)

	msg := liveAttachMsgForResizeOwner(state.Root{}, binding)
	if msg.Config.EndpointID != "west" || msg.Config.TerminalID != "term-1" || msg.Config.ViewID != "view-remote" {
		t.Fatalf("remote owner attach routed incorrectly: %#v", msg.Config)
	}
}

func TestTakeResizeOwnerFailureKeepsAttachedFollower(t *testing.T) {
	binding := state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleFollower, "surface-new", "view-new", false)
	binding.OwnerSurfaceID = "surface-old"
	binding.OwnerViewID = "view-old"
	binding.ResizeEpoch = 4
	binding.Session = testEndpointSessionStamp("west")
	root := state.Root{Shell: state.DefaultShell()}
	root.TerminalViews = root.TerminalViews.BindPane(binding)
	var decision state.TerminalViewResizeDecision
	root.TerminalViews, decision = root.TerminalViews.RequestViewResizeOwner(binding.ViewID, 90, 30)
	if !decision.Allowed {
		t.Fatalf("owner request was not armed: %#v", decision)
	}

	next, _ := newLiveReducerPrepared(LiveDeps{})(root, LiveResizeResultMsg{
		EndpointID: "west", TerminalID: "term-1", ViewID: binding.ViewID,
		Seq: decision.Seq, TakeOwnership: true, Session: binding.Session, Err: errors.New("owner changed"),
	})
	got, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !got.Attached || got.Channel != 7 || got.OwnerAcquirePending || got.ResizeRole != state.TerminalResizeRoleFollower || got.HasProjectedResizeOwner() {
		t.Fatalf("failed owner request corrupted attachment or authority: %#v", got)
	}
	if got.OwnerSurfaceID != "surface-old" || got.OwnerViewID != "view-old" || got.ResizeEpoch != 4 {
		t.Fatalf("failed owner request discarded last daemon projection: %#v", got)
	}
	toasts := next.Shell.EnsureDefaults().Toasts
	if len(toasts) == 0 || toasts[len(toasts)-1].Title != "terminal.owner" {
		t.Fatalf("failed owner request should report owner-scoped warning: %#v", toasts)
	}
}
