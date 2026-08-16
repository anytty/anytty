package app

import (
	"errors"
	"github.com/anytty/anytty/tui/testkit"
	"testing"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestDetachedViewResizeResultDoesNotOverrideCurrentSessionSize(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Cols:       118,
			Rows:       38,
			Attached:   true,
		}.Resize(118, 38),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       118,
			Rows:       38,
		}.Resize(118, 38),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			8,
			118,
			38,
			state.TerminalResizeRoleOwner,
			"surface-main",
			state.TerminalPaneViewID(state.DefaultPaneID),
			true,
		)),
	}

	next, effects := reducer(root, LiveResizeResultMsg{
		ViewID: state.TerminalPaneViewID("pane-2"),
		Seq:    1,
		Cols:   64,
		Rows:   38,
		Result: port.TerminalResizeResult{
			TerminalID:   "term-1",
			Cols:         64,
			Rows:         38,
			Resized:      true,
			CanResize:    true,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-pane-2",
			ViewID:       state.TerminalPaneViewID("pane-2"),
		},
	})
	if len(effects) != 1 {
		t.Fatalf("detached stale resize result should resync latest owner size once, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(nil).(LiveResizeMsg)
	if !ok {
		t.Fatalf("expected stale recovery LiveResizeMsg, got %#v", effects[0])
	}
	if msg.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || msg.Cols != 118 || msg.Rows != 38 {
		t.Fatalf("stale recovery must reassert latest owner size, got %#v", msg)
	}
	if next.Session.Cols != 118 || next.Session.Rows != 38 {
		t.Fatalf("detached view resize result must not override session size, got %#v", next.Session)
	}
	if next.Surface.Cols != 118 || next.Surface.Rows != 38 {
		t.Fatalf("detached view resize result must not override surface size, got %#v", next.Surface)
	}
	if _, ok := next.TerminalViews.Views[state.TerminalPaneViewID("pane-2")]; ok {
		t.Fatalf("detached view must stay absent, views=%#v", next.TerminalViews.Views)
	}
}

func TestDetachedViewResizeRequestDoesNotReuseCurrentSessionChannel(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Channel:    7,
			Attached:   true,
			SurfaceID:  "surface-main",
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		}.Resize(118, 38),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       118,
			Rows:       38,
		}.Resize(118, 38),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			7,
			118,
			38,
			state.TerminalResizeRoleOwner,
			"surface-main",
			state.TerminalPaneViewID(state.DefaultPaneID),
			true,
		)),
	}

	next, effects := reducer(root, LiveResizeMsg{
		TerminalID: "term-1",
		ViewID:     state.TerminalPaneViewID("pane-2"),
		Cols:       64,
		Rows:       38,
		Seq:        3,
	})
	if len(effects) != 0 {
		t.Fatalf("detached view resize request should be dropped, got %#v", effects)
	}
	if len(terminal.Resizes) != 0 {
		t.Fatalf("detached view resize request must not reuse current session channel, got %#v", terminal.Resizes)
	}
	if next.Session.Cols != 118 || next.Session.Rows != 38 {
		t.Fatalf("detached view resize request must not change session, got %#v", next.Session)
	}
}

func TestViewScopedResizeErrorSurfacesInSessionAndBinding(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Channel:    7,
			Attached:   true,
			SurfaceID:  "surface-main",
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		}.Resize(98, 26),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       98,
			Rows:       26,
		}.Resize(98, 26),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			7,
			118,
			36,
			state.TerminalResizeRoleOwner,
			"surface-main",
			state.TerminalPaneViewID(state.DefaultPaneID),
			true,
		)),
	}
	root.TerminalViews, _ = root.TerminalViews.RequestViewResize(state.TerminalPaneViewID(state.DefaultPaneID), 118, 36)

	next, effects := reducer(root, LiveResizeResultMsg{
		ViewID: state.TerminalPaneViewID(state.DefaultPaneID),
		Seq:    1,
		Cols:   118,
		Rows:   36,
		Err:    errors.New("pty resize failed"),
	})
	if len(effects) != 0 {
		t.Fatalf("resize error should not emit effects, got %#v", effects)
	}
	if next.Session.Attached || next.Session.LastError != "pty resize failed" || next.Surface.Err != "pty resize failed" {
		t.Fatalf("view-scoped resize error must surface in session and surface, session=%#v surface=%#v", next.Session, next.Surface)
	}
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.LastError != "pty resize failed" {
		t.Fatalf("view-scoped resize error must be recorded on binding, binding=%#v ok=%v", binding, ok)
	}
}

func TestStaleViewScopedResizeErrorStillSurfacesInSession(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	root := state.Root{
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Channel:    7,
			Attached:   true,
			SurfaceID:  "surface-main",
			ViewID:     viewID,
		}.Resize(118, 36),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       118,
			Rows:       36,
		}.Resize(118, 36),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			7,
			118,
			36,
			state.TerminalResizeRoleOwner,
			"surface-main",
			viewID,
			true,
		)),
	}
	root.TerminalViews, _ = root.TerminalViews.RequestViewResize(viewID, 118, 36)
	root.TerminalViews, _ = root.TerminalViews.RequestViewResize(viewID, 120, 38)

	next, effects := reducer(root, LiveResizeResultMsg{
		ViewID: viewID,
		Seq:    1,
		Cols:   118,
		Rows:   36,
		Err:    errors.New("pty resize failed"),
	})
	if len(effects) != 0 {
		t.Fatalf("resize error should not emit stale recovery effects, got %#v", effects)
	}
	if next.Session.Attached || next.Session.LastError != "pty resize failed" || next.Surface.Err != "pty resize failed" {
		t.Fatalf("stale resize error must still surface in session and surface, session=%#v surface=%#v", next.Session, next.Surface)
	}
}
