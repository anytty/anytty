package app

import (
	"context"
	"errors"
	"github.com/anytty/anytty/tui/testkit"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestTerminalPoolListResultSchedulesResourceRefreshAndLivePreview(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Revision:   2,
				Lines:      []string{"preview"},
			},
		},
	}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPool()}
	root.TerminalPool = root.TerminalPool.RequestList()
	seq := root.TerminalPool.RequestSeq

	next, effects := reducer(root, TerminalPoolListResultMsg{
		Seq: seq,
		Result: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "shell",
			State:      "running",
			Cols:       90,
			Rows:       20,
		}}},
	})

	if next.TerminalPool.Status != state.TerminalPoolReady || !terminalPoolRefreshLoopScheduled(effects) {
		t.Fatalf("list result should refresh inventory and arm background refresh, pool=%#v effects=%#v", next.TerminalPool, effects)
	}
	msg, ok := terminalPoolLiveSurfaceMsgFromEffects(t, effects)
	if !ok || msg.Snapshot.TerminalID != "term-1" || msg.RequestedCols != 90 || msg.RequestedRows != 20 {
		t.Fatalf("list result should refresh selected live preview, msg=%#v effects=%#v", msg, effects)
	}
	if len(terminal.Surfaces) != 1 || terminal.Surfaces[0].TerminalID != "term-1" || terminal.Surfaces[0].Cols != 90 || terminal.Surfaces[0].Rows != 20 {
		t.Fatalf("preview refresh should request selected terminal surface, got %#v", terminal.Surfaces)
	}
}

func TestTerminalPoolIgnoresInFlightListAfterEndpointIsDisabled(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{
			ID: "remote", Label: "Remote", Enabled: false,
		}),
	}
	root.TerminalPool = root.TerminalPool.RequestRefresh()
	next, effects := reduceTerminalPoolListResult(root, TerminalPoolListResultMsg{
		EndpointID: "remote",
		Seq:        root.TerminalPool.RequestSeq,
		Refresh:    true,
		Result: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "late", Title: "Late result",
		}}},
	}, LiveDeps{})
	if len(next.TerminalPool.Items) != 0 || len(effects) != 0 {
		t.Fatalf("disabled endpoint applied late observation: pool=%#v effects=%#v", next.TerminalPool, effects)
	}
}

func TestTerminalPoolDisconnectedReconnectErrorStaysInPane(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	err := errors.New("ssh transport closed: exit status 255: stdio-proxy connect core-v2 daemon socket: connection refused")
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID),
		Session: state.TerminalSessionStore{
			EndpointID:    ref.EndpointID,
			TerminalID:    ref.TerminalID,
			Channel:       9,
			Attached:      true,
			State:         state.TerminalLiveAttached,
			InputChannels: map[string]uint16{ref.Key(): 9},
			ViewID:        state.TerminalPaneViewID(state.DefaultPaneID),
		},
		Surface: state.TerminalSurfaceStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, State: state.TerminalLiveAttached, Ready: true, Lines: []string{"remote"}},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: "west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Title: "remote"}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	pending, _ := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	root.TerminalViews, _ = root.TerminalViews.BeginAttach(pending)

	reducer := newTerminalPoolReducerPrepared(LiveDeps{})
	next, effects := reducer(root, TerminalPoolReconnectResultMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: state.DefaultPaneID, Err: err, LocalError: true})

	if len(effects) != 0 || len(next.Shell.Toasts) != 0 {
		t.Fatalf("local reconnect failure should not emit effects or global toasts, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.AttachPending || binding.Channel != 0 || binding.Attached || !strings.Contains(binding.LastError, "remote-daemon") {
		t.Fatalf("pane should keep terminal intent and local error, binding=%#v ok=%v", binding, ok)
	}
	if next.Session.LastError == "" || next.Session.State != state.TerminalLiveError {
		t.Fatalf("active session should show reconnect failure, session=%#v", next.Session)
	}
	if west, _ := next.Endpoints.Endpoint("west"); west.LastErrorKind != state.EndpointErrorRemoteDaemon || west.DisplayStatus() != state.EndpointStatusOffline {
		t.Fatalf("endpoint should show remote daemon reconnect failure, got %#v", west)
	}
}

func TestTerminalPoolPickerReconnectFailureClearsConnectingAndKeepsToast(t *testing.T) {
	ref := state.NewTerminalRef("west", "remote")
	err := errors.New("ssh transport closed: exit status 255")
	root := state.Root{
		Shell:     state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID),
		Endpoints: (state.EndpointStore{}).Upsert(state.EndpointItem{ID: ref.EndpointID, Label: "US West", Transport: state.EndpointTransportSSH, Enabled: true, Status: state.EndpointStatusConnecting}),
	}
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(state.DefaultPaneID), false)
	binding.AttachPending = true
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	next, effects := newTerminalPoolReducerPrepared(LiveDeps{})(root, TerminalPoolReconnectResultMsg{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, TargetPaneID: state.DefaultPaneID, Err: err})
	if len(effects) != 0 || len(next.Shell.Toasts) != 1 {
		t.Fatalf("picker reconnect failure should keep one toast and no follow-up effects, effects=%#v toasts=%#v", effects, next.Shell.Toasts)
	}
	view, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || view.AttachPending || view.LastError == "" {
		t.Fatalf("picker reconnect failure should close connecting projection, view=%#v ok=%v", view, ok)
	}
	if endpoint, ok := next.Endpoints.Endpoint(ref.EndpointID); !ok || endpoint.DisplayStatus() != state.EndpointStatusOffline {
		t.Fatalf("picker reconnect failure should mark owning endpoint offline, endpoint=%#v ok=%v", endpoint, ok)
	}
}

func TestTerminalPoolRefreshTickRequestsSilentList(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "shell",
			State:      "running",
		}}},
	}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items:  []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running"}},
		},
	}

	next, effects := reducer(root, TerminalPoolRefreshTickMsg{})
	if next.TerminalPool.Status != state.TerminalPoolReady || next.TerminalPool.RequestSeq == root.TerminalPool.RequestSeq {
		t.Fatalf("refresh tick should keep ready state and advance request seq, before=%#v after=%#v", root.TerminalPool, next.TerminalPool)
	}
	if len(effects) != 1 {
		t.Fatalf("refresh tick should only schedule list effect, got %#v", effects)
	}
	result, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolListResultMsg)
	if !ok || !result.Refresh || result.Seq != next.TerminalPool.RequestSeq {
		t.Fatalf("refresh tick should return silent list result, got %#v", result)
	}
	if len(terminal.Lists) != 1 {
		t.Fatalf("refresh tick should call terminal list once, got %#v", terminal.Lists)
	}
}

func TestTerminalPickerListResultSchedulesBackgroundRefresh(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{Shell: state.DefaultShell().OpenTerminalPicker()}
	root.TerminalPool = root.TerminalPool.RequestList()
	seq := root.TerminalPool.RequestSeq

	next, effects := reducer(root, TerminalPoolListResultMsg{
		Seq:    seq,
		Result: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "shell"}}},
	})
	if next.TerminalPool.Status != state.TerminalPoolReady || !terminalPoolRefreshLoopScheduled(effects) {
		t.Fatalf("picker list result should arm background inventory refresh, pool=%#v effects=%#v", next.TerminalPool, effects)
	}
	if terminalPoolPreviewRefreshScheduledIgnoringLoop(t, effects) {
		t.Fatalf("terminal picker must not schedule terminal-manager preview refresh, effects=%#v", effects)
	}
}

func TestTerminalPickerRefreshTickFansOutSilentEndpointList(t *testing.T) {
	terminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "shell"}}}}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items:  []state.TerminalPoolItem{{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "shell"}},
		},
	}

	next, effects := reducer(root, TerminalPoolRefreshTickMsg{})
	if next.TerminalPool.Status != state.TerminalPoolReady || next.TerminalPool.RequestSeq == root.TerminalPool.RequestSeq || len(effects) != 2 {
		t.Fatalf("picker refresh tick should silently fan out endpoint lists, before=%#v after=%#v effects=%#v", root.TerminalPool, next.TerminalPool, effects)
	}
	seen := map[state.EndpointID]bool{}
	for _, effect := range effects {
		msg, ok := effect.(FuncEffect).Run(context.Background()).(TerminalPoolListResultMsg)
		if !ok || !msg.Refresh || msg.Seq != next.TerminalPool.RequestSeq {
			t.Fatalf("picker refresh effect should return silent endpoint list result, got %#v", msg)
		}
		seen[msg.EndpointID] = true
	}
	if !seen[state.DefaultEndpointID] || !seen["west"] {
		t.Fatalf("picker refresh should include local and west endpoints, got %#v", seen)
	}
}

func TestTerminalPoolListRequestFansOutToConnectableEndpoints(t *testing.T) {
	terminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "shell"}}}}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}).
			Upsert(state.EndpointItem{ID: "manual", Label: "Manual", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectManual, Enabled: true}).
			Upsert(state.EndpointItem{ID: "disabled", Label: "Disabled", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectAuto, Enabled: false}),
	}

	next, effects := reducer(root, TerminalPoolListRequestMsg{})
	if next.TerminalPool.Status != state.TerminalPoolLoading || len(effects) != 2 {
		t.Fatalf("expected fan-out effects for local and west only, pool=%#v effects=%#v", next.TerminalPool, effects)
	}
	seen := map[state.EndpointID]bool{}
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || !fn.Async || !fn.ForceSyncInTests {
			t.Fatalf("endpoint list effect should be async but syncable in tests, got %#v", effect)
		}
		msg, ok := fn.Run(context.Background()).(TerminalPoolListResultMsg)
		if !ok {
			t.Fatalf("endpoint list effect returned %#v", msg)
		}
		seen[msg.EndpointID] = true
	}
	if !seen[state.DefaultEndpointID] || !seen["west"] || seen["manual"] || seen["disabled"] {
		t.Fatalf("unexpected endpoint fan-out set %#v", seen)
	}
}

func TestTerminalPoolEndpointListFailureMarksOnlyThatEndpointOffline(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		Session: state.TerminalSessionStore{}.
			AttachRefWithResizeOwner(state.LocalTerminalRef("term-1"), 7, 80, 24, state.TerminalResizeRoleOwner, "surface-local", state.TerminalPaneViewID(state.DefaultPaneID)),
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			EndpointID: state.DefaultEndpointID,
			TerminalID: "term-1",
			Lines:      []string{"local"},
			State:      state.TerminalLiveAttached,
		}),
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-local", state.TerminalPaneViewID(state.DefaultPaneID), true)),
		TerminalPool: state.TerminalPoolStore{
			RequestSeq: 4,
			Items: []state.TerminalPoolItem{
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "local"},
				{EndpointID: "west", TerminalID: "term-1", Title: "west"},
			},
		},
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
	}

	next, _ := reducer(root, TerminalPoolListResultMsg{EndpointID: "west", Seq: 4, Err: context.DeadlineExceeded})
	if len(next.TerminalPool.Items) != 2 {
		t.Fatalf("endpoint failure must not clear terminal pool rows, got %#v", next.TerminalPool.Items)
	}
	west, ok := next.Endpoints.Endpoint("west")
	if !ok || west.DisplayStatus() != state.EndpointStatusOffline || west.LastError == "" {
		t.Fatalf("west endpoint should be offline with error, got %#v", west)
	}
	local, ok := next.Endpoints.Endpoint(state.DefaultEndpointID)
	if !ok || local.DisplayStatus() != state.EndpointStatusConnected {
		t.Fatalf("local endpoint should stay connected, got %#v", local)
	}
	if next.Session.State != state.TerminalLiveAttached || !next.Session.TerminalRef().Equal(state.LocalTerminalRef("term-1")) {
		t.Fatalf("endpoint list failure must not poison active session, session=%#v", next.Session)
	}
	if next.Surface.State != state.TerminalLiveAttached || !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("term-1")) || next.Surface.Err != "" || next.Surface.Lines[0] != "local" {
		t.Fatalf("endpoint list failure must not poison active surface, surface=%#v", next.Surface)
	}
}

func TestTerminalPoolEndpointListSuccessDisconnectsMissingRemoteBinding(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-local", Title: "local", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: "west",
		TerminalID: "term-1",
		Lines:      []string{"old remote"},
		State:      state.TerminalLiveAttached,
	})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{
			EndpointID:       "west",
			TerminalID:       "term-1",
			Channel:          9,
			Attached:         true,
			InputChannels:    map[string]uint16{"west/term-1": 9},
			State:            state.TerminalLiveAttached,
			ResizePolicy:     state.TerminalResizeRoleOwner,
			SurfaceID:        "surface-west",
			ViewID:           state.TerminalPaneViewID(state.DefaultPaneID),
			DesiredCols:      100,
			DesiredRows:      30,
			LastError:        "",
			ResizeRequestSeq: 2,
		},
		Surface: surface,
		TerminalPool: state.TerminalPoolStore{
			RequestSeq: 4,
			Status:     state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: "west", TerminalID: "term-1", Title: "west"},
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "local"},
			},
		},
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
		History:  state.HistoryStore{EndpointID: "west", TerminalID: "term-1", Token: "tok-remote"},
		CopyMode: state.CopyModeStore{Active: true, EndpointID: "west", TerminalID: "term-1", BoundToken: "tok-remote"},
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 9, 100, 30, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewEndpointPaneTerminalView(state.DefaultEndpointID, "pane-local", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-local", state.TerminalPaneViewID("pane-local"), true))

	next, effects := reducer(root, TerminalPoolListResultMsg{EndpointID: "west", Seq: 4, Refresh: true, Result: port.TerminalListResult{Items: nil}})
	if _, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		t.Fatalf("missing remote terminal should clear west pane binding, views=%#v", next.TerminalViews)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "" || pane.Kind != state.PaneEmpty {
		t.Fatalf("missing remote terminal should leave pane unconnected, pane=%#v ok=%v", pane, ok)
	}
	if binding, ok := next.TerminalViews.PaneBinding("pane-local"); !ok || !binding.TerminalRef().Equal(state.LocalTerminalRef("term-1")) {
		t.Fatalf("local same-id binding must survive remote missing list, binding=%#v ok=%v", binding, ok)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: "pane-local"}); !ok || pane.TerminalID != "term-1" || pane.Kind != state.PaneTerminalLive {
		t.Fatalf("local same-id pane must stay connected, pane=%#v ok=%v", pane, ok)
	}
	if next.Session.TerminalID != "" || next.Surface.TerminalID != "" || next.History.TerminalID != "" || next.CopyMode.TerminalID != "" {
		t.Fatalf("missing remote terminal should clear live and copy state, session=%#v surface=%#v history=%#v copy=%#v", next.Session, next.Surface, next.History, next.CopyMode)
	}
	if len(next.TerminalPool.Items) != 1 || !next.TerminalPool.Items[0].TerminalRef().Equal(state.LocalTerminalRef("term-1")) {
		t.Fatalf("endpoint list should only remove west pool rows, pool=%#v", next.TerminalPool.Items)
	}
	if msg := firstWorkbenchPersistEffect(t, effects); msg.Reason != "terminal.inventory-missing" {
		t.Fatalf("missing terminal disconnect should persist workbench, msg=%#v effects=%#v", msg, effects)
	}
}

func TestTerminalPoolEndpointListSuccessKeepsRuntimeDisconnectedPane(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	ref := state.NewTerminalRef("west", "term-1")
	errText := "remote-daemon: stdio-proxy connect core-v2 daemon socket: connection refused"
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ref.TerminalID)
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Lines:      []string{"old remote"},
		State:      state.TerminalLiveError,
		Err:        errText,
	})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{
			EndpointID:    ref.EndpointID,
			TerminalID:    ref.TerminalID,
			State:         state.TerminalLiveError,
			LastError:     errText,
			InputChannels: map[string]uint16{},
			SurfaceID:     "surface-west",
			ViewID:        state.TerminalPaneViewID(state.DefaultPaneID),
			DesiredCols:   100,
			DesiredRows:   30,
		},
		Surface: surface,
		TerminalPool: state.TerminalPoolStore{
			RequestSeq: 4,
			Status:     state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Title: "west"},
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-local", Title: "local"},
			},
		},
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: ref.EndpointID, Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusOffline, LastError: errText, LastErrorKind: state.EndpointErrorRemoteDaemon}),
		History:  state.HistoryStore{EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, Token: "tok-remote"},
		CopyMode: state.CopyModeStore{Active: true, EndpointID: ref.EndpointID, TerminalID: ref.TerminalID, BoundToken: "tok-remote"},
	}
	binding := state.NewEndpointPaneTerminalView(ref.EndpointID, state.DefaultPaneID, ref.TerminalID, 0, 100, 30, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true)
	binding.LastError = errText
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	next, effects := reducer(root, TerminalPoolListResultMsg{EndpointID: ref.EndpointID, Seq: 4, Refresh: true, Result: port.TerminalListResult{Items: nil}})
	if binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !binding.TerminalRef().Equal(ref) || !strings.Contains(binding.LastError, "remote-daemon") {
		t.Fatalf("runtime-disconnected pane should keep terminal intent and reason, binding=%#v ok=%v", binding, ok)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != ref.TerminalID || pane.Kind != state.PaneTerminalLive {
		t.Fatalf("runtime-disconnected pane must not collapse to unconnected, pane=%#v ok=%v", pane, ok)
	}
	if _, ok := terminalPoolItemRef(next.TerminalPool, ref); !ok {
		t.Fatalf("runtime-disconnected endpoint should keep last known pool row until reconnect/disconnect, pool=%#v", next.TerminalPool.Items)
	}
	if west, ok := next.Endpoints.Endpoint(ref.EndpointID); !ok || west.DisplayStatus() != state.EndpointStatusOffline || west.LastErrorKind != state.EndpointErrorRemoteDaemon || west.LastError == "" {
		t.Fatalf("empty inventory refresh must not clear endpoint runtime error, west=%#v ok=%v", west, ok)
	}
	if next.Session.State != state.TerminalLiveError || !strings.Contains(next.Session.LastError, "remote-daemon") || !next.Session.TerminalRef().Equal(ref) {
		t.Fatalf("active session should keep runtime error projection, session=%#v", next.Session)
	}
	if next.Surface.State != state.TerminalLiveError || !strings.Contains(next.Surface.Err, "remote-daemon") || !next.Surface.TerminalRef().Equal(ref) {
		t.Fatalf("active surface should keep runtime error projection, surface=%#v", next.Surface)
	}
	if len(effects) != 0 {
		t.Fatalf("held runtime-disconnected inventory refresh should not persist a missing-terminal detach, effects=%#v", effects)
	}
}

func TestTerminalPoolRefreshFailureMarksEndpointOfflineSilently(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{
			RequestSeq: 7,
			Status:     state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Title: "local"},
				{EndpointID: "west", TerminalID: "term-1", Title: "west"},
			},
		},
		Endpoints: state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true, Status: state.EndpointStatusConnected}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true, Status: state.EndpointStatusConnected}),
	}

	next, effects := reducer(root, TerminalPoolListResultMsg{EndpointID: "west", Seq: 7, Refresh: true, Err: context.DeadlineExceeded})
	if len(next.TerminalPool.Items) != 2 || next.TerminalPool.Status != state.TerminalPoolReady {
		t.Fatalf("refresh failure must not clear rows or move pool to error, pool=%#v", next.TerminalPool)
	}
	west, ok := next.Endpoints.Endpoint("west")
	if !ok || west.DisplayStatus() != state.EndpointStatusOffline || west.LastError == "" {
		t.Fatalf("west endpoint should be offline after refresh failure, got %#v", west)
	}
	if len(next.Shell.EnsureDefaults().Toasts) != 0 {
		t.Fatalf("refresh failure should stay silent, got toasts %#v", next.Shell.EnsureDefaults().Toasts)
	}
	if !terminalPoolRefreshLoopScheduled(effects) {
		t.Fatalf("refresh failure should keep background refresh loop, effects=%#v", effects)
	}
}

func TestTerminalPoolCreateResultFallsBackToRequestedRemoteIDForAttach(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	next, effects := reduceTerminalPoolCreateResult(root, TerminalPoolCreateResultMsg{
		EndpointID:  "west",
		RequestedID: "term-requested",
		Result:      port.TerminalCreateResult{State: "running"},
	})
	if next.TerminalPool.LastCreatedRef != (state.TerminalRef{EndpointID: "west", TerminalID: "term-requested"}) {
		t.Fatalf("create result should record requested remote ref, pool=%#v", next.TerminalPool)
	}
	if len(effects) != 2 {
		t.Fatalf("create result should schedule list and attach, effects=%#v", effects)
	}
	attach, ok := effects[1].(FuncEffect).Run(context.Background()).(TerminalPoolAttachRequestMsg)
	if !ok || attach.EndpointID != "west" || attach.TerminalID != "term-requested" {
		t.Fatalf("create result should attach requested remote terminal, msg=%#v", attach)
	}
}

func TestTerminalPoolCreateRequestDefaultsRemoteCommandByEndpoint(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	terminal := &testkit.FakeTerminalService{CreateResult: port.TerminalCreateResult{State: "running"}}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell: state.DefaultShell(),
		Endpoints: (state.EndpointStore{}.
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true})).
			ApplyDefaults("west", []string{"/bin/bash", "-l"}, "/srv/west", ""),
	}

	_, effects := reducer(root, TerminalPoolCreateRequestMsg{EndpointID: "west", Title: "remote", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("create request should schedule one create effect, effects=%#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolCreateResultMsg)
	if !ok || msg.EndpointID != "west" {
		t.Fatalf("create effect should return west create result, msg=%#v", msg)
	}
	if len(terminal.Creates) != 1 || terminal.Creates[0].EndpointID != "west" || strings.Join(terminal.Creates[0].Command, " ") != "/bin/bash -l" {
		t.Fatalf("remote create request should default command for target endpoint, creates=%#v", terminal.Creates)
	}
	if terminal.Creates[0].TerminalID != "remote" || terminal.Creates[0].Title != "remote" {
		t.Fatalf("create request should use terminal name as daemon-local key, creates=%#v", terminal.Creates)
	}
}

func TestTerminalPoolCreateRequestRejectsDuplicateNameOnSameEndpoint(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{
			{EndpointID: "west", TerminalID: "existing-id", Title: "build"},
			{EndpointID: state.DefaultEndpointID, TerminalID: "build", Title: "build"},
		}},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "west", Label: "West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}).
			ApplyDefaults("west", []string{"/bin/sh"}, "/srv/west", ""),
	}

	next, effects := reducer(root, TerminalPoolCreateRequestMsg{EndpointID: "west", Title: "build", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 0 || len(terminal.Creates) != 0 {
		t.Fatalf("duplicate name on same endpoint should not create, effects=%#v creates=%#v", effects, terminal.Creates)
	}
	if len(next.Shell.Toasts) != 1 || !strings.Contains(next.Shell.Toasts[0].Body, "already exists") {
		t.Fatalf("duplicate create should explain name conflict, shell=%#v", next.Shell)
	}

	_, effects = reducer(root, TerminalPoolCreateRequestMsg{EndpointID: "west", Title: "build-west-2", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("different name on same endpoint should still create, effects=%#v", effects)
	}
}

func TestTerminalPoolCreateRequestReusesNameAfterRenameWithFreshStableID(t *testing.T) {
	terminal := &testkit.FakeTerminalService{CreateResult: port.TerminalCreateResult{State: "running"}}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{
			{EndpointID: state.DefaultEndpointID, TerminalID: "A", Title: "B"},
		}},
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "Local", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			ApplyDefaults(state.DefaultEndpointID, []string{"/bin/sh"}, "/tmp", ""),
	}

	_, effects := reducer(root, TerminalPoolCreateRequestMsg{Title: "A", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("renamed terminal should release its old display name, effects=%#v", effects)
	}
	result, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolCreateResultMsg)
	if !ok || result.Err != nil {
		t.Fatalf("expected successful create result, got %#v", result)
	}
	if len(terminal.Creates) != 1 {
		t.Fatalf("expected one terminal create, got %#v", terminal.Creates)
	}
	created := terminal.Creates[0]
	if created.Title != "A" {
		t.Fatalf("new terminal should keep the requested display name, got %#v", created)
	}
	if created.TerminalID == "A" || created.TerminalID == "" {
		t.Fatalf("new terminal must avoid the renamed terminal's stable ID, got %#v", created)
	}
	if result.RequestedID != created.TerminalID {
		t.Fatalf("create result should carry the fresh stable ID, result=%#v create=%#v", result, created)
	}
}

func TestTerminalPickerEndpointCreateRowOpensPromptWithDraftEndpoint(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPicker(),
		Endpoints: (state.EndpointStore{}).
			Upsert(state.EndpointItem{ID: state.DefaultEndpointID, Label: "This Mac", Transport: state.EndpointTransportLocal, ConnectMode: state.EndpointConnectAuto, Enabled: true}).
			Upsert(state.EndpointItem{ID: "us-west", Label: "US West", Transport: state.EndpointTransportSSH, ConnectMode: state.EndpointConnectOnDemand, Enabled: true}),
	}
	root.Shell.TerminalCreateDraft = state.TerminalCreateDraft{EndpointID: "us-west", Command: "/bin/bash", Workdir: "/srv/app"}
	items := state.TerminalPickerItems(root)
	if len(items) != 1 || !items[0].CreateNew || items[0].EndpointID != "us-west" {
		t.Fatalf("expected one create row owned by the selected draft endpoint, picker=%#v", items)
	}

	_, effects := NewShellReducer()(root, shortcutTestMessage("terminal_picker.new", "", false, 0))
	if len(effects) != 2 {
		t.Fatalf("expected prompt effect, got %#v", effects)
	}
	msg, ok := effects[1].(FuncEffect).Run(context.Background()).(ShellOpenPromptMsg)
	if !ok {
		t.Fatalf("expected prompt message, got %#v", effects[1])
	}
	if msg.Prompt.TargetEndpointID != "us-west" || msg.Prompt.FieldRawValue("server") != "US West (us-west)" || !msg.Prompt.PublicTagList {
		t.Fatalf("endpoint create row should use the remembered endpoint and pure tag input, prompt=%#v", msg.Prompt)
	}
	if msg.Prompt.FieldRawValue("workdir") != "/srv/app" || msg.Prompt.FieldRawValue("command") != "/bin/bash" {
		t.Fatalf("endpoint create row should reuse last create draft fields, prompt=%#v", msg.Prompt)
	}
}

func TestLiveAttachTerminalNotFoundDisconnectsPane(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachErr: errors.New("protocol error 404: terminal not found")}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-missing")}

	pending, effects := reducer(root, LiveAttachMsg{Config: LiveConfig{
		EndpointID: "west",
		TerminalID: "term-missing",
		Cols:       80,
		Rows:       24,
		SurfaceID:  "surface-west",
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
	}})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	next, _ := reducer(pending, msg)
	if _, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		t.Fatalf("terminal-not-found attach should clear pending binding, views=%#v", next.TerminalViews)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "" || pane.Kind != state.PaneEmpty {
		t.Fatalf("terminal-not-found attach should leave pane unconnected, pane=%#v ok=%v", pane, ok)
	}
	if next.Session.TerminalID != "" || next.Surface.TerminalID != "" {
		t.Fatalf("terminal-not-found attach should clear live state, session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestTerminalPoolSelectionInputRefreshesPreview(t *testing.T) {
	reducer := NewUIInputReducer()
	root := state.Root{
		Shell: state.DefaultShell().OpenTerminalPool(),
		TerminalPool: state.TerminalPoolStore{
			Status: state.TerminalPoolReady,
			Items: []state.TerminalPoolItem{
				{TerminalID: "term-1", Title: "one"},
				{TerminalID: "term-2", Title: "two"},
			},
		},
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDown}})
	if next.Shell.EnsureDefaults().Overlay.SelectedIndex != 1 || !terminalPoolPreviewRefreshScheduled(t, effects) {
		t.Fatalf("keyboard selection should refresh preview, shell=%#v effects=%#v", next.Shell, effects)
	}
	next, effects = reducer(next, ShellOverlayMouseSelectMsg{Delta: -1})
	if next.Shell.EnsureDefaults().Overlay.SelectedIndex != 0 || !terminalPoolPreviewRefreshScheduled(t, effects) {
		t.Fatalf("mouse selection should refresh preview, shell=%#v effects=%#v", next.Shell, effects)
	}

	shellNext, shellEffects := NewShellReducer()(next, shortcutTestMessage("terminal_pool.select", "", false, 1))
	if shellNext.Shell.EnsureDefaults().Overlay.SelectedIndex != 1 || !terminalPoolPreviewRefreshScheduled(t, shellEffects) {
		t.Fatalf("row select action should refresh preview, shell=%#v effects=%#v", shellNext.Shell, shellEffects)
	}
}

func terminalPoolRefreshLoopScheduled(effects []Effect) bool {
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if ok && fn.Token == terminalPoolRefreshToken && fn.Async {
			return true
		}
	}
	return false
}

func terminalPoolLiveSurfaceMsgFromEffects(t *testing.T, effects []Effect) (LiveSurfaceMsg, bool) {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || fn.Token == terminalPoolRefreshToken {
			continue
		}
		if msg, ok := fn.Run(context.Background()).(LiveSurfaceMsg); ok {
			return msg, true
		}
	}
	return LiveSurfaceMsg{}, false
}

func terminalPoolPreviewRefreshScheduled(t *testing.T, effects []Effect) bool {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil {
			continue
		}
		if _, ok := fn.Run(context.Background()).(TerminalPoolPreviewRefreshMsg); ok {
			return true
		}
	}
	return false
}

func terminalPoolPreviewRefreshScheduledIgnoringLoop(t *testing.T, effects []Effect) bool {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || fn.Token == terminalPoolRefreshToken {
			continue
		}
		if _, ok := fn.Run(context.Background()).(TerminalPoolPreviewRefreshMsg); ok {
			return true
		}
	}
	return false
}
