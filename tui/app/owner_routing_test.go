package app

import (
	"errors"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

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
