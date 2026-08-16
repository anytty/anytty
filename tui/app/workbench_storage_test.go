package app

import (
	"context"
	"errors"
	"github.com/anytty/anytty/tui/testkit"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestWorkbenchStorageReducerLoadsSnapshotFromOpaqueStorage(t *testing.T) {
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-logs"})
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotWorkbenchForStorage(shell),
			Version:  4,
			Found:    true,
		},
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, WorkbenchStorageLoadRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected load effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, msg)

	if len(storage.Loads) != 1 || storage.Loads[0].AppID != state.WorkbenchStorageAppID || storage.Loads[0].Key != state.WorkbenchStorageKeyRoot {
		t.Fatalf("unexpected load ref %#v", storage.Loads)
	}
	if root.Shell.ActivePaneID != "pane-logs" || root.Shell.Workspace.Tabs[0].Panes[1].TerminalID != "term-logs" {
		t.Fatalf("expected loaded workbench shell, got %#v", root.Shell)
	}
}

func TestWorkbenchStorageLoadAppliesConfiguredPanelPresentation(t *testing.T) {
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotWorkbenchForStorage(shell),
			Version:  4,
			Found:    true,
		},
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{
		Shell: state.DefaultShell(),
		Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{
			PanelPresentation: string(state.PanelPresentationCard),
		}},
	}

	root, effects := reducer(root, WorkbenchStorageLoadRequestMsg{})
	msg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, msg)

	if got := root.Shell.EnsureDefaults().PanelPresentation; got != state.PanelPresentationCard {
		t.Fatalf("configured panel presentation should override restored chrome preference, got %q", got)
	}
}

func TestWorkbenchStorageLoadAndSaveEffectsAreAsync(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	_, effects := reducer(root, WorkbenchStorageLoadRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected load effect, got %#v", effects)
	}
	load, ok := effects[0].(FuncEffect)
	if !ok || !load.Async || !load.ForceSyncInTests || load.Token != workbenchStorageLoadToken {
		t.Fatalf("workbench load must be async in real runtime and sync-capable in harness, got %#v", effects[0])
	}

	_, effects = reducer(root, WorkbenchStoragePersistRequestMsg{Reason: "test"})
	if len(effects) != 1 {
		t.Fatalf("expected save effect, got %#v", effects)
	}
	save, ok := effects[0].(FuncEffect)
	if !ok || !save.Async || !save.ForceSyncInTests || save.Token != workbenchStorageSaveToken {
		t.Fatalf("workbench save must be async in real runtime and sync-capable in harness, got %#v", effects[0])
	}
}

func TestWorkbenchStorageReducerLoadsAndPersistsTerminalViews(t *testing.T) {
	shell := state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	views := state.TerminalViewStore{}
	views = views.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	views = views.BindPane(state.NewPaneTerminalView("pane-logs", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-logs"), false))
	storage := &testkit.FakeWorkbenchStorageService{LoadResult: port.WorkbenchStorageLoadResult{Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}), Version: 4, Found: true}}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, WorkbenchStorageLoadRequestMsg{})
	loadMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, loadMsg)
	if bindings := root.TerminalViews.BindingsForTerminal("term-1"); len(bindings) != 2 {
		t.Fatalf("expected loaded terminal view bindings, got %#v", bindings)
	}

	root, effects = reducer(root, WorkbenchStoragePersistRequestMsg{Reason: "test"})
	persistMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, persistMsg)
	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.TerminalViews) != 2 || storage.Saves[0].Snapshot.SchemaVersion != state.WorkbenchStorageSchemaV2 {
		t.Fatalf("expected persisted v2 terminal view snapshot, saves=%#v", storage.Saves)
	}
}

func TestWorkbenchCommandPersistsSnapshotThroughStorageReducer(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{}
	reducer := ComposeReducers(NewShellReducer(), NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage}))
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"}})
	if len(effects) != 1 {
		t.Fatalf("expected shell command to emit persist request effect, got %#v", effects)
	}
	persistMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("expected persist request to emit storage save effect, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 {
		t.Fatalf("expected one storage save, got %#v", storage.Saves)
	}
	save := storage.Saves[0]
	if save.Ref.AppID != state.WorkbenchStorageAppID || save.Ref.Key != state.WorkbenchStorageKeyRoot || save.Snapshot.Schema != state.WorkbenchStorageSchema {
		t.Fatalf("unexpected save request %#v", save)
	}
	if !save.CheckVersion || save.ExpectedVersion != 0 {
		t.Fatalf("initial persist must use storage CAS at version 0, got %#v", save)
	}
	if len(save.Snapshot.Workspace.Tabs) != 2 || save.Snapshot.Workspace.Tabs[1].Title != "logs" {
		t.Fatalf("storage snapshot must contain updated tab tree, got %#v", save.Snapshot.Workspace.Tabs)
	}
	if len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Title != "workbench.storage" {
		t.Fatalf("persist result should show feedback, got %#v", root.Shell.Toasts)
	}
	if root.WorkbenchSync.LastSavedVersion != 1 {
		t.Fatalf("persist result should track local saved version, got %#v", root.WorkbenchSync)
	}
}

func TestTerminalPoolAttachPersistsTerminalViewBindingThroughStorageReducer(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{}
	reducer := ComposeReducers(newTerminalPoolReducerPrepared(LiveDeps{}), NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage}))
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, TerminalPoolAttachResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      7,
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-1",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	})
	if len(effects) != 1 {
		t.Fatalf("terminal attach should emit workbench persist effect, got %#v", effects)
	}
	persistMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("persist request should emit storage save effect, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 {
		t.Fatalf("expected one workbench storage save, got %#v", storage.Saves)
	}
	save := storage.Saves[0]
	if len(save.Snapshot.TerminalViews) != 1 {
		t.Fatalf("snapshot must contain terminal view binding, got %#v", save.Snapshot.TerminalViews)
	}
	binding := save.Snapshot.TerminalViews[0]
	if binding.PaneID != state.DefaultPaneID || binding.TerminalID != "term-1" || binding.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("unexpected persisted binding %#v", binding)
	}
	if save.Snapshot.Workspace.Tabs[0].Panes[0].TerminalID != "term-1" {
		t.Fatalf("workspace pane must persist terminal identity too, got %#v", save.Snapshot.Workspace.Tabs[0].Panes[0])
	}
	if !save.CheckVersion || save.ExpectedVersion != 0 {
		t.Fatalf("initial terminal attach persist must use CAS version 0, got %#v", save)
	}
	if root.WorkbenchSync.LastSavedVersion != 1 {
		t.Fatalf("terminal attach persist should update saved version, got %#v", root.WorkbenchSync)
	}
}

func TestTerminalViewLayoutCommandPersistsWorkbenchSnapshot(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{}
	reducer := ComposeReducers(NewUIInputReducer(), NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage}))
	root := state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeResize)}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true))

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyRight, Shift: true}})
	persistMsg := firstWorkbenchPersistRequestMsg(t, effects)
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("persist request should emit storage save effect, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.TerminalViews) != 1 {
		t.Fatalf("expected one saved terminal view layout snapshot, saves=%#v", storage.Saves)
	}
	layout := storage.Saves[0].Snapshot.TerminalViews[0].Layout
	if layout.PanX != 2 {
		t.Fatalf("persisted terminal view layout should include pan delta, got %#v", layout)
	}
	if root.WorkbenchSync.LastSavedVersion != 1 {
		t.Fatalf("layout persist should update saved version, got %#v", root.WorkbenchSync)
	}
}

func firstWorkbenchPersistRequestMsg(t *testing.T, effects []Effect) WorkbenchStoragePersistRequestMsg {
	t.Helper()
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || fn.Token == stickyInteractionModeTimeoutToken || fn.Token == shellShortcutPassthroughTimeoutToken {
			continue
		}
		if msg, ok := fn.Run(context.Background()).(WorkbenchStoragePersistRequestMsg); ok {
			return msg
		}
	}
	t.Fatalf("expected workbench persist request effect, got %#v", effects)
	return WorkbenchStoragePersistRequestMsg{}
}

func TestWorkbenchRestoreAttachEffectsUseFollowerRuntimeSurface(t *testing.T) {
	bindings := []state.TerminalViewBinding{
		state.NewPaneTerminalView("pane-follower", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-follower", false),
		state.NewPaneTerminalView("pane-owner", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-owner", true),
		state.NewPaneTerminalView("pane-observer", "term-2", 9, 30, 10, state.TerminalResizeRoleObserver, "surface", "view-observer", false),
	}
	root := state.Root{RuntimeSurfaceID: "runtime-a"}

	effects := workbenchRestoredTerminalAttachEffectsForBindings(root, workbenchRestoredTerminalAttachBindings(state.TerminalViewStore{}, bindings))
	if len(effects) != 3 {
		t.Fatalf("expected one attach effect per binding, got %#v", effects)
	}
	for _, effect := range effects {
		msg, ok := effect.(FuncEffect).Run(context.Background()).(LiveAttachMsg)
		if !ok {
			t.Fatalf("expected LiveAttachMsg effect, got %#v", effect)
		}
		if msg.Config.ResizePolicy != state.TerminalResizeRoleFollower || msg.Config.SurfaceID != "runtime-a" {
			t.Fatalf("restore attach must not inherit stored owner/surface, got %#v", msg.Config)
		}
	}
	first, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveAttachMsg)
	if !ok || first.Config.ViewID != "view-owner" {
		t.Fatalf("restore keeps stable ordering even though all requests are follower, got %#v", first)
	}
}

func TestWorkbenchRestoreSkipsReattachForAlreadyLiveBinding(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	live := state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-live", viewID, true)
	live.Session = &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "unix", Generation: 9}
	live.OperationID = "attach-9"
	previous := state.TerminalViewStore{NextOperation: 12}.BindPane(live)
	stored := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 0, 80, 24, state.TerminalResizeRoleOwner, "surface-stored", viewID, false))

	restored := preserveWorkbenchRuntimeTerminalViews(previous, stored)
	binding, ok := restored.PaneBinding(state.DefaultPaneID)
	if !ok || binding.Channel != 7 || binding.SurfaceID != "surface-live" || !binding.CanResize || binding.Session.GetGeneration() != 9 || binding.OperationID != "attach-9" || restored.NextOperation != 12 {
		t.Fatalf("restore should preserve current live runtime fields, binding=%#v ok=%v", binding, ok)
	}
	if effects := workbenchRestoredTerminalAttachEffects(previous, restored.Bindings()); len(effects) != 0 {
		t.Fatalf("already live same view/terminal should not reattach, effects=%#v", effects)
	}
}

func TestWorkbenchRestoreSkipsReattachForPendingStartupAttach(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	previous, _ := (state.TerminalViewStore{}).BeginAttach(state.TerminalViewBinding{
		ViewID:      viewID,
		SurfaceID:   "runtime-a",
		TerminalID:  "term-1",
		ResizeRole:  state.TerminalResizeRoleFollower,
		DesiredCols: 80,
		DesiredRows: 24,
		PaneID:      state.DefaultPaneID,
	})
	stored := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 0, 80, 24, state.TerminalResizeRoleFollower, "", viewID, false))

	restored := preserveWorkbenchRuntimeTerminalViews(previous, stored)
	if effects := workbenchRestoredTerminalAttachEffects(previous, restored.Bindings()); len(effects) != 0 {
		t.Fatalf("pending startup attach should claim same view/terminal before channel arrives, effects=%#v", effects)
	}
}

func TestWorkbenchRestorePreservesLocalFloatingDisplayState(t *testing.T) {
	local := state.DefaultShell()
	var result state.FloatingCommandResult
	local, result = local.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Title:    "local",
		Pane:     state.PaneState{ID: "float-1-pane", Title: "local", Kind: state.PaneTerminalLive, TerminalID: "term-local"},
		Rect:     state.FloatingRect{X: 5, Y: 6, W: 70, H: 20},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create local floating: %#v", result)
	}
	local, result = local.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandToggleCollapse, TargetID: "float-1"})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("collapse local floating: %#v", result)
	}

	remote := state.DefaultShell()
	remote, result = remote.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Title:    "remote",
		Pane:     state.PaneState{ID: "float-1-pane", Title: "remote", Kind: state.PaneTerminalLive, TerminalID: "term-remote"},
		Rect:     state.FloatingRect{X: 1, Y: 2, W: 30, H: 8},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create remote floating: %#v", result)
	}

	merged := mergeLocalWorkbenchRuntimeState(local, remote, true)
	floating, ok := merged.FloatingByID("float-1")
	if !ok {
		t.Fatalf("expected floating after restore, shell=%#v", merged)
	}
	if floating.Pane.TerminalID != "term-remote" || floating.Title != "remote" {
		t.Fatalf("restore should take shared slot/binding fields, floating=%#v", floating)
	}
	if floating.Rect != (state.FloatingRect{X: 5, Y: 6, W: 70, H: 20}) || !floating.Collapsed || merged.ActiveFloatingID() != "" {
		t.Fatalf("restore should keep local floating geometry/display state, floating=%#v active=%q", floating, merged.ActiveFloatingID())
	}
}

func TestWorkbenchStorageRestoreInitializesFloatingGeometryFromTerminalSize(t *testing.T) {
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-1",
		Title:    "one",
		Pane:     state.PaneState{ID: "float-1-pane", Title: "one", Kind: state.PaneTerminalLive, TerminalID: "term-one"},
		Rect:     state.FloatingRect{X: 2, Y: 3, W: 40, H: 10},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create first floating: %#v", result)
	}
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "float-2",
		Title:    "two",
		Pane:     state.PaneState{ID: "float-2-pane", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-two"},
		Rect:     state.FloatingRect{X: 8, Y: 5, W: 30, H: 8},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create second floating: %#v", result)
	}
	views := state.TerminalViewStore{}
	views = views.BindFloating(state.NewFloatingTerminalView("float-1", "float-1-pane", "term-one", 7, 90, 20, state.TerminalResizeRoleOwner, "surface-one", state.TerminalFloatingViewID("float-1"), true))
	views = views.BindFloating(state.NewFloatingTerminalView("float-2", "float-2-pane", "term-two", 8, 50, 10, state.TerminalResizeRoleFollower, "surface-two", state.TerminalFloatingViewID("float-2"), false))
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
			Version:  4,
			Found:    true,
		},
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
	}

	root, effects := reducer(root, WorkbenchStorageLoadRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected load effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, msg)

	floatings := root.Shell.ActiveFloatings()
	if len(floatings) != 2 {
		t.Fatalf("expected restored floatings, got %#v", floatings)
	}
	if floatings[0].Rect != (state.FloatingRect{X: 14, Y: 9, W: 92, H: 22}) {
		t.Fatalf("first floating should fit terminal size and center, got %#v", floatings[0].Rect)
	}
	if floatings[1].Rect != (state.FloatingRect{X: 38, Y: 15, W: 52, H: 12}) {
		t.Fatalf("second floating should fit terminal size and cascade, got %#v", floatings[1].Rect)
	}
}

func TestWorkbenchPaneCRUDPersistsClosedPaneSnapshotWithCAS(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{CurrentVersion: 3}
	reducer := ComposeReducers(NewShellReducer(), NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage}))
	root := state.Root{
		Shell: state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical),
		WorkbenchSync: (state.WorkbenchSyncStore{}).MarkApplied(3),
	}

	root, effects := reducer(root, ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{
		Action: state.WorkbenchCommandPaneClose,
		Target: state.PaneCommandTarget{PaneID: "pane-logs"},
		Source: state.PaneCommandSourceTest,
	}})
	if len(effects) != 1 {
		t.Fatalf("pane close workbench command should request persist, got %#v", effects)
	}
	persistMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("persist request should emit storage save, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 || !storage.Saves[0].CheckVersion || storage.Saves[0].ExpectedVersion != 3 {
		t.Fatalf("pane CRUD persist must use storage CAS, saves=%#v", storage.Saves)
	}
	for _, pane := range storage.Saves[0].Snapshot.Workspace.Tabs[0].Panes {
		if pane.ID == "pane-logs" {
			t.Fatalf("closed pane must not be present in persisted snapshot, snapshot=%#v", storage.Saves[0].Snapshot)
		}
	}
	if root.WorkbenchSync.SaveVersion() != 4 {
		t.Fatalf("successful pane CRUD persist should advance version, sync=%#v", root.WorkbenchSync)
	}
}

func TestWorkbenchCommandPersistsAgainstLoadedStorageVersion(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{CurrentVersion: 7}
	reducer := ComposeReducers(NewShellReducer(), NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage}))
	root := state.Root{
		Shell:         state.DefaultShell(),
		WorkbenchSync: (state.WorkbenchSyncStore{}).MarkApplied(7),
	}

	root, effects := reducer(root, ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"}})
	if len(effects) != 1 {
		t.Fatalf("expected workbench command persist request, got %#v", effects)
	}
	persistMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("expected storage save effect, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 || !storage.Saves[0].CheckVersion || storage.Saves[0].ExpectedVersion != 7 {
		t.Fatalf("persist must use loaded base version, saves=%#v", storage.Saves)
	}
	if root.WorkbenchSync.LastSavedVersion != 8 || root.WorkbenchSync.SaveVersion() != 8 {
		t.Fatalf("successful save should advance base version, got %#v", root.WorkbenchSync)
	}
}

func TestWorkbenchStorageChangedReloadsExternalSnapshot(t *testing.T) {
	externalShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-external", Title: "external", Kind: state.PaneTerminalLive, TerminalID: "term-external"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-external"})
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotWorkbenchForStorage(externalShell),
			Version:  8,
			Found:    true,
		},
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, WorkbenchStorageChangedMsg{Event: port.WorkbenchStorageEvent{
		Ref:     state.DefaultWorkbenchStorageRef("").WithVersion(8),
		Version: 8,
		Op:      "put",
	}})
	if len(effects) != 1 || root.WorkbenchSync.LastEventVersion != 8 {
		t.Fatalf("storage change should request reload, root=%#v effects=%#v", root, effects)
	}
	loadRequest := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, loadRequest)
	if len(effects) != 1 {
		t.Fatalf("load request should emit storage load effect, got %#v", effects)
	}
	loadResult := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, loadResult)

	if root.Shell.ActivePaneID != "pane-external" || root.WorkbenchSync.LastAppliedVersion != 8 {
		t.Fatalf("first external snapshot should refresh shell from remote active pane, root=%#v", root)
	}
}

func TestWorkbenchStorageLoadInvalidatesFrozenHistoryAndCopyMode(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-new"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-new", 11, 80, 24, state.TerminalResizeRoleFollower, "surface-new", state.TerminalPaneViewID(state.DefaultPaneID), false))
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: &testkit.FakeWorkbenchStorageService{}})
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true)),
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			Token:      "tok-old",
			Cols:       80,
			Rows:       []state.HistoryRow{{Text: "old-history", LineID: 1}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			BoundToken: "tok-old",
			BoundCols:  80,
			ViewRows:   20,
		},
	}

	root, effects := reducer(root, WorkbenchStorageLoadResultMsg{Result: port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  9,
		Found:    true,
	}})

	if root.CopyMode.Active || root.CopyMode.TerminalID != "" || root.CopyMode.BoundToken != "" {
		t.Fatalf("workbench load must clear stale copy mode binding, got %#v", root.CopyMode)
	}
	if root.History.TerminalID != "term-old" || root.History.Token != "" || len(root.History.Rows) != 0 || root.History.Pending != nil {
		t.Fatalf("workbench load must invalidate stale frozen history window, got %#v", root.History)
	}
	if root.Shell.Workspace.Tabs[0].Panes[0].TerminalID != "term-new" {
		t.Fatalf("workbench load should still apply external shell snapshot, got %#v", root.Shell)
	}
	if len(effects) != 2 {
		t.Fatalf("restored terminal view should still request list+attach effects, got %#v", effects)
	}
}

func TestWorkbenchStorageConflictReloadsLatestSnapshot(t *testing.T) {
	remoteShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-remote", Title: "remote", Kind: state.PaneTerminalLive, TerminalID: "term-remote"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-remote"})
	storage := &testkit.FakeWorkbenchStorageService{
		CurrentVersion: 9,
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotWorkbenchForStorage(remoteShell),
			Version:  9,
			Found:    true,
		},
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{
		Shell:         state.DefaultShell(),
		WorkbenchSync: (state.WorkbenchSyncStore{}).MarkApplied(8),
	}

	root, effects := reducer(root, WorkbenchStoragePersistRequestMsg{Reason: "tab.create"})
	if len(effects) != 1 {
		t.Fatalf("persist request should emit save effect, got %#v", effects)
	}
	persistResult := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistResult)
	if len(effects) != 1 || !root.WorkbenchSync.Conflict || root.WorkbenchSync.ConflictVersion != 8 {
		t.Fatalf("conflict should mark state and request reload, root=%#v effects=%#v", root, effects)
	}
	if len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Body != "conflict: reloading" {
		t.Fatalf("conflict should show reload feedback, toasts=%#v", root.Shell.Toasts)
	}
	loadRequest := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, loadRequest)
	if len(effects) != 1 {
		t.Fatalf("reload request should emit load effect, got %#v", effects)
	}
	loadResult := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, loadResult)

	if root.Shell.ActivePaneID != state.DefaultPaneID || root.Shell.Workspace.Tabs[0].Panes[1].ID != "pane-remote" || root.WorkbenchSync.Conflict || root.WorkbenchSync.SaveVersion() != 9 {
		t.Fatalf("conflict reload should apply latest remote structure while preserving local active pane, root=%#v", root)
	}
}

func TestWorkbenchStorageDuplicateConflictDoesNotReloadAgain(t *testing.T) {
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: &testkit.FakeWorkbenchStorageService{}})
	root := state.Root{
		Shell:         state.DefaultShell(),
		WorkbenchSync: (state.WorkbenchSyncStore{}).MarkApplied(8),
	}

	conflict := WorkbenchStoragePersistResultMsg{
		Err:             port.ErrWorkbenchStorageConflict,
		ExpectedVersion: 8,
	}
	root, effects := reducer(root, conflict)
	if len(effects) != 1 || !root.WorkbenchSync.Conflict || root.WorkbenchSync.ConflictVersion != 8 {
		t.Fatalf("first conflict should request reload, root=%#v effects=%#v", root, effects)
	}

	root, effects = reducer(root, conflict)
	if len(effects) != 0 || len(root.Shell.Toasts) != 1 {
		t.Fatalf("duplicate conflict for same version should be ignored, root=%#v effects=%#v", root, effects)
	}

	stale := WorkbenchStoragePersistResultMsg{
		Err:             port.ErrWorkbenchStorageConflict,
		ExpectedVersion: 7,
	}
	root, effects = reducer(root, stale)
	if len(effects) != 0 || root.WorkbenchSync.ConflictVersion != 8 {
		t.Fatalf("stale conflict result should not request reload, root=%#v effects=%#v", root, effects)
	}
}

func TestWorkbenchStorageChangedIgnoresSelfPersistVersion(t *testing.T) {
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: &testkit.FakeWorkbenchStorageService{}})
	root := state.Root{
		Shell:         state.DefaultShell(),
		WorkbenchSync: (state.WorkbenchSyncStore{}).MarkSaved(state.DefaultWorkbenchStorageRef("").WithVersion(5), 5),
	}

	root, effects := reducer(root, WorkbenchStorageChangedMsg{Event: port.WorkbenchStorageEvent{
		Ref:     state.DefaultWorkbenchStorageRef("").WithVersion(5),
		Version: 5,
		Op:      "put",
	}})
	if len(effects) != 0 || root.WorkbenchSync.LastEventVersion != 5 || root.WorkbenchSync.LastAppliedVersion != 0 {
		t.Fatalf("self storage event should not reload, root=%#v effects=%#v", root, effects)
	}
}

func TestWorkbenchStorageReducerReportsMissingServiceAndSaveErrors(t *testing.T) {
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{})
	root, effects := reducer(state.Root{Shell: state.DefaultShell()}, WorkbenchStoragePersistRequestMsg{Reason: "tab.create"})
	if len(effects) != 0 || len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Body != "storage service missing" {
		t.Fatalf("missing storage should be visible feedback, root=%#v effects=%#v", root, effects)
	}

	storage := &testkit.FakeWorkbenchStorageService{SaveErr: errors.New("version conflict")}
	reducer = NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root = state.Root{Shell: state.DefaultShell()}
	root, effects = reducer(root, WorkbenchStoragePersistRequestMsg{Reason: "tab.create"})
	msg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, msg)
	if len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Body != "version conflict" {
		t.Fatalf("save error should be visible feedback, got %#v", root.Shell.Toasts)
	}
}

func TestWorkbenchStorageContextCanceledStaysSilent(t *testing.T) {
	storage := &testkit.FakeWorkbenchStorageService{
		LoadErr:  context.Canceled,
		SaveErr:  context.Canceled,
		WatchErr: context.Canceled,
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, WorkbenchStorageWatchRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected watch effect, got %#v", effects)
	}
	effects[0].(StreamEffect).Run(context.Background(), func(msg Msg) {
		root, _ = reducer(root, msg)
	})
	if len(root.Shell.Toasts) != 0 {
		t.Fatalf("context canceled watch should not toast, got %#v", root.Shell.Toasts)
	}

	root, effects = reducer(root, WorkbenchStorageLoadRequestMsg{})
	if msg := effects[0].(FuncEffect).Run(context.Background()); msg != nil {
		t.Fatalf("context canceled load should not post result, got %#v", msg)
	}
	root, effects = reducer(root, WorkbenchStoragePersistRequestMsg{Reason: "test"})
	if msg := effects[0].(FuncEffect).Run(context.Background()); msg != nil {
		t.Fatalf("context canceled save should not post result, got %#v", msg)
	}
	if len(root.Shell.Toasts) != 0 {
		t.Fatalf("context canceled storage effects should stay silent, got %#v", root.Shell.Toasts)
	}
}

func TestInteractiveRuntimeWithWorkbenchPersistsWorkbenchCommand(t *testing.T) {
	host := NewFakeTerminalHost(8)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	storage := &testkit.FakeWorkbenchStorageService{WatchCh: watchCh}
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage},
	)

	if err := runtime.Post(ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"}}); err != nil {
		t.Fatalf("post workbench command: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.Workspace.Tabs) != 2 {
		t.Fatalf("interactive runtime should persist workbench command, saves=%#v", storage.Saves)
	}
	if len(storage.Watches) != 1 || storage.Watches[0].Key != state.WorkbenchStorageKeyRoot {
		t.Fatalf("interactive runtime should subscribe to storage.changed, watches=%#v", storage.Watches)
	}
}

func TestInteractiveRuntimeWithWorkbenchCanSkipInitialLoad(t *testing.T) {
	host := NewFakeTerminalHost(8)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	storage := &testkit.FakeWorkbenchStorageService{WatchCh: watchCh}
	runtime := NewInteractiveRuntimeWithStorage(
		state.Root{Shell: state.DefaultShell()},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage, SkipInitialLoad: true},
		ClipboardDeps{},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(storage.Loads) != 0 {
		t.Fatalf("skip initial load should not restore stale workbench snapshot, loads=%#v", storage.Loads)
	}
	if len(storage.Watches) != 1 {
		t.Fatalf("skip initial load should keep storage watch active, watches=%#v", storage.Watches)
	}
}

func TestInteractiveRuntimeWithWorkbenchPersistsFloatingCommand(t *testing.T) {
	host := NewFakeTerminalHost(8)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	storage := &testkit.FakeWorkbenchStorageService{WatchCh: watchCh}
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "floating", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 2, Y: 2, W: 30, H: 8},
		BoundsW:  80,
		BoundsH:  24,
	})

	runtime := NewInteractiveRuntimeWithWorkbench(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage},
	)

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandCollapseAll, Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("post floating command: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(storage.Saves) != 0 {
		t.Fatalf("floating display command should stay local and not persist workbench snapshot, saves=%#v", storage.Saves)
	}
}

func TestInteractiveRuntimeFloatingAutoFitRefreshDoesNotPersist(t *testing.T) {
	host := NewFakeTerminalHost(8)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	storage := &testkit.FakeWorkbenchStorageService{WatchCh: watchCh}
	root := state.Root{
		Shell:    state.DefaultShell(),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Surface:  state.TerminalSurfaceStore{Surfaces: map[string]state.LiveSurfaceSnapshot{"term-1": {TerminalID: "term-1", Cols: 40, Rows: 12, State: state.TerminalLiveAttached}}},
	}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "floating", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 2, Y: 2, W: 20, H: 8},
		BoundsW:  100,
		BoundsH:  30,
	})
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))

	runtime := NewInteractiveRuntimeWithWorkbench(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage},
	)

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandToggleAutoFit, TargetID: "floating-1", Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("enable auto-fit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain enable auto-fit: %v", err)
	}
	storage.Saves = nil

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Cols: 60, Rows: 20, State: state.TerminalLiveAttached}}); err != nil {
		t.Fatalf("post live surface update: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain live surface update: %v", err)
	}

	floating := runtime.State().Shell.ActiveFloatings()[0]
	if floating.AutoFit.Cols != 60 || floating.AutoFit.Rows != 20 || floating.Rect.W != 62 || floating.Rect.H != 22 {
		t.Fatalf("auto-fit refresh should update floating geometry from live size, got %#v", floating)
	}
	if len(storage.Saves) != 0 {
		t.Fatalf("auto-fit refresh should not persist workbench snapshot, saves=%#v", storage.Saves)
	}
}

func TestInteractiveRuntimeWithWorkbenchLoadsSnapshotBeforeWatch(t *testing.T) {
	host := NewFakeTerminalHost(8)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-restored", Title: "restored", Kind: state.PaneTerminalLive, TerminalID: "term-restored"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-restored"})
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView("pane-restored", "term-restored", 11, 80, 24, state.TerminalResizeRoleOwner, "surface-restored", state.TerminalPaneViewID("pane-restored"), true))
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
			Version:  7,
			Found:    true,
		},
		WatchCh: watchCh,
	}
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	root := runtime.State()
	if len(storage.Loads) != 1 || len(storage.Watches) != 1 {
		t.Fatalf("interactive runtime should load once and watch once, loads=%#v watches=%#v", storage.Loads, storage.Watches)
	}
	if root.Shell.ActivePaneID != "pane-restored" || root.WorkbenchSync.LastAppliedVersion != 7 || root.WorkbenchSync.LastSavedVersion != 8 {
		t.Fatalf("expected restored workbench snapshot, root=%#v", root)
	}
	if len(storage.Saves) != 1 || storage.Saves[0].ExpectedVersion != 7 || storage.Saves[0].Snapshot.TerminalViews[0].TerminalID != "term-restored" {
		t.Fatalf("restored terminal reattach should persist refreshed snapshot, saves=%#v", storage.Saves)
	}
	if binding, ok := root.TerminalViews.PaneBinding("pane-restored"); !ok || binding.TerminalID != "term-restored" || binding.ViewID == "" {
		t.Fatalf("expected restored terminal view binding, binding=%#v ok=%v", binding, ok)
	}
}

func TestInteractiveRuntimeWorkbenchRestoreReattachesTerminalViewsFromCore(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(120, 40)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-restored", Title: "restored", Kind: state.PaneTerminalLive, TerminalID: "term-restored"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-restored"})
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView("pane-restored", "term-restored", 11, 80, 24, state.TerminalResizeRoleOwner, "surface-restored", state.TerminalPaneViewID("pane-restored"), true))
	storage := &testkit.FakeWorkbenchStorageService{LoadResult: port.WorkbenchStorageLoadResult{Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}), Version: 7, Found: true}, WatchCh: watchCh}
	terminal := &testkit.FakeTerminalService{
		AttachResult:  port.TerminalAttachResult{Channel: 42, Cols: 100, Rows: 30, ResizePolicy: state.TerminalResizeRoleOwner, CanResize: true, OwnerSurfaceID: DefaultRuntimeSurfaceID, OwnerViewID: state.TerminalPaneViewID("pane-restored")},
		SurfaceResult: port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-restored", Cols: 100, Rows: 30, Lines: []string{"changed by another tui"}, State: state.TerminalLiveAttached}},
	}
	runtime := NewInteractiveRuntimeWithWorkbench(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, WorkbenchDeps{Storage: storage})

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 1 {
		t.Fatalf("restored terminal view should reattach through core, attaches=%#v", terminal.Attaches)
	}
	attach := terminal.Attaches[0]
	if attach.TerminalID != "term-restored" || attach.ViewID != state.TerminalPaneViewID("pane-restored") || attach.ResizePolicy != state.TerminalResizeRoleFollower {
		t.Fatalf("restore attach must start as follower for this runtime, attach=%#v", attach)
	}
	root := runtime.State()
	binding, ok := root.TerminalViews.PaneBinding("pane-restored")
	if !ok || binding.Channel != 42 || !binding.HasAuthoritativeResizeOwner() || !binding.Attached {
		t.Fatalf("restored binding should reflect core attach result, binding=%#v ok=%v", binding, ok)
	}
	if channel, ok := root.Session.InputChannelFor("term-restored"); !ok || channel != 42 {
		t.Fatalf("restored attach should refresh input channel, channel=%d ok=%v", channel, ok)
	}
	if len(terminal.Surfaces) != 1 || root.Surface.TerminalID != "term-restored" || root.Surface.Lines[0] != "changed by another tui" {
		t.Fatalf("restore should load authoritative live surface, surfaces=%#v surface=%#v", terminal.Surfaces, root.Surface)
	}
}

func TestWorkbenchRestoreInputDoesNotUseStoredOldChannelBeforeAttachEffect(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-restored"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-restored", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true))
	loadResult := port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  7,
		Found:    true,
	}
	loadResult.Snapshot.TerminalViews[0].Channel = 7
	loadResult.Snapshot.TerminalViews[0].Attached = true
	loadResult.Snapshot.TerminalViews[0].CanResize = true
	loadResult.Snapshot.TerminalViews[0].OwnerViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	terminal := &refreshingInputTerminalService{
		nextChannel: 21,
		FakeTerminalService: testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-restored",
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleOwner,
			CanResize:    true,
		}},
	}
	liveDeps := LiveDeps{Terminal: terminal}
	reducer := ComposeReducers(NewUIInputReducer(), NewWorkbenchStorageReducer(WorkbenchDeps{}), NewTerminalInputRouterReducer(liveDeps), newLiveReducerPrepared(liveDeps))
	root := state.Root{Shell: state.DefaultShell(), Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30}}

	root, restoreEffects := reducer(root, WorkbenchStorageLoadResultMsg{Result: loadResult})
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 0 || binding.Attached {
		t.Fatalf("restored binding must not expose stored old channel before attach, binding=%#v ok=%v", binding, ok)
	}
	if len(restoreEffects) == 0 {
		t.Fatal("restore should still schedule background reattach effects")
	}

	root, inputEffects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}})
	if len(inputEffects) != 1 {
		t.Fatalf("input on restored un-attached view should schedule one attach effect, got %#v", inputEffects)
	}
	attachMsg, ok := inputEffects[0].(FuncEffect).Run(context.Background()).(LiveInputAttachResultMsg)
	if !ok {
		t.Fatalf("expected input attach result, got %#v", inputEffects[0])
	}
	root, replayEffects := reducer(root, attachMsg)

	if len(terminal.Attaches) != 1 {
		t.Fatalf("input on restored view should attach before replay, attaches=%#v", terminal.Attaches)
	}
	inputAttach := terminal.Attaches[0]
	if inputAttach.TerminalID != "term-restored" || inputAttach.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("input attach should target restored pane view, got %#v", inputAttach)
	}
	for _, effect := range replayEffects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		_ = fn.Run(context.Background())
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].Channel == 7 || terminal.Inputs[0].Channel != terminal.nextChannel-1 || string(terminal.Inputs[0].Bytes) != "l" {
		t.Fatalf("input must replay on fresh attach channel, inputs=%#v attaches=%#v", terminal.Inputs, terminal.Attaches)
	}
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != terminal.Inputs[0].Channel || !binding.Attached {
		t.Fatalf("fresh input channel should be stored on restored view, binding=%#v ok=%v", binding, ok)
	}
}

func TestWorkbenchRestoreKeepsMissingEndpointBindingUnresolved(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-remote"
	views := state.TerminalViewStore{}.
		BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-remote", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-west", state.TerminalPaneViewID(state.DefaultPaneID), true))
	loadResult := port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  7,
		Found:    true,
	}
	root := state.Root{
		Shell:     state.DefaultShell(),
		Endpoints: state.EndpointStore{}.Upsert(state.DefaultLocalEndpoint()),
	}
	reducer := NewWorkbenchStorageReducer(WorkbenchDeps{})

	root, effects := reducer(root, WorkbenchStorageLoadResultMsg{Result: loadResult})

	binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.EndpointID != "west" || binding.TerminalID != "term-remote" || !binding.Unresolved || binding.UnresolvedReason != string(state.EndpointStatusUnregistered) {
		t.Fatalf("missing endpoint binding must be preserved unresolved, binding=%#v ok=%v", binding, ok)
	}
	if pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "term-remote" || pane.Kind != state.PaneTerminalLive {
		t.Fatalf("layout pane must remain bound, pane=%#v ok=%v", pane, ok)
	}
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		if msg := fn.Run(context.Background()); msg != nil {
			if _, attach := msg.(LiveAttachMsg); attach {
				t.Fatalf("unresolved endpoint must not auto attach, got %#v", msg)
			}
		}
	}
}

func TestInteractiveRuntimeWorkbenchRestoreShowsExitedTerminalFromCore(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-exited"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-exited", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	storage := &testkit.FakeWorkbenchStorageService{LoadResult: port.WorkbenchStorageLoadResult{Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}), Version: 7, Found: true}, WatchCh: watchCh}
	terminal := &testkit.FakeTerminalService{
		AttachResult:  port.TerminalAttachResult{Channel: 12, Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleFollower},
		SurfaceResult: port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-exited", Cols: 80, Rows: 24, State: state.TerminalLiveExited, ExitCode: 130, ExitReason: "process exited"}},
	}
	runtime := NewInteractiveRuntimeWithWorkbench(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, WorkbenchDeps{Storage: storage})

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	root := runtime.State()
	if root.Surface.TerminalID != "term-exited" || root.Surface.State != state.TerminalLiveExited || root.Surface.ExitCode != 130 {
		t.Fatalf("restored exited terminal should remain visible, surface=%#v", root.Surface)
	}
	if pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "term-exited" {
		t.Fatalf("exited terminal pane must stay bound, pane=%#v ok=%v", pane, ok)
	}
}

func TestInteractiveRuntimeWorkbenchRestoreLegacyExitedPaneUsesCoreRunningLifecycle(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneKind("exited")
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 9, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	snapshot := state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views})
	// 模拟 R75 之前已经落盘的旧 storage：snapshot restore 入口必须自己 scrub。
	snapshot.Workspace.Tabs[0].Panes[0].Kind = state.PaneKind("exited")
	snapshot.Workspaces[0].Tabs[0].Panes[0].Kind = state.PaneKind("exited")
	storage := &testkit.FakeWorkbenchStorageService{LoadResult: port.WorkbenchStorageLoadResult{Snapshot: snapshot, Version: 7, Found: true}, WatchCh: watchCh}
	terminal := &testkit.FakeTerminalService{
		ListResult:    port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-main", Title: "main", State: "running", Cols: 80, Rows: 24}}},
		AttachResult:  port.TerminalAttachResult{TerminalID: "term-main", Channel: 12, Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner},
		SurfaceResult: port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-main", Cols: 80, Rows: 24, Lines: []string{"terminal exited: term-main code:0 exited", "% "}, Cursor: state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"}, State: state.TerminalLiveAttached}},
	}
	runtime := NewInteractiveRuntimeWithWorkbench(state.Root{}, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, WorkbenchDeps{Storage: storage})

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	root := runtime.State()
	if pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.Kind != state.PaneTerminalLive || pane.TerminalID != "term-main" {
		t.Fatalf("legacy exited pane should restore as live binding intent, pane=%#v ok=%v", pane, ok)
	}
	if root.Surface.State != state.TerminalLiveAttached || root.Session.State == state.TerminalLiveExited {
		t.Fatalf("core running lifecycle should win, session=%#v surface=%#v", root.Session, root.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "restart") || !frameContains(frame, "% ") || !frame.Cursor.Visible {
		t.Fatalf("running restored terminal should render prompt without restart CTA, frame=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestInteractiveRuntimeWorkbenchRestoreLegacyPaneWithoutTerminalViewsUsesCoreRunningLifecycle(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	snapshot := state.WorkbenchStorageSnapshot{
		Schema:        state.WorkbenchStorageSchema,
		SchemaVersion: state.WorkbenchStorageSchemaV2,
		Workspace: state.WorkspaceState{
			ID:          state.DefaultWorkspaceID,
			ActiveTabID: state.DefaultTabID,
			Tabs: []state.TabState{{
				ID:           state.DefaultTabID,
				ActivePaneID: state.DefaultPaneID,
				RootSplit:    state.SplitNode{PaneID: state.DefaultPaneID},
				Panes: []state.PaneState{{
					ID:         state.DefaultPaneID,
					Title:      "main",
					Kind:       state.PaneKind("exited"),
					TerminalID: "term-main",
					Active:     true,
				}},
			}},
		},
		Workspaces: []state.WorkspaceState{{
			ID:          state.DefaultWorkspaceID,
			ActiveTabID: state.DefaultTabID,
			Tabs: []state.TabState{{
				ID:           state.DefaultTabID,
				ActivePaneID: state.DefaultPaneID,
				RootSplit:    state.SplitNode{PaneID: state.DefaultPaneID},
				Panes: []state.PaneState{{
					ID:         state.DefaultPaneID,
					Title:      "main",
					Kind:       state.PaneKind("exited"),
					TerminalID: "term-main",
					Active:     true,
				}},
			}},
		}},
		PanelPresentation: state.PanelPresentationCard,
		ActivePaneID:      state.DefaultPaneID,
		HeaderVisible:     true,
		FooterVisible:     true,
	}
	storage := &testkit.FakeWorkbenchStorageService{LoadResult: port.WorkbenchStorageLoadResult{Snapshot: snapshot, Version: 7, Found: true}, WatchCh: watchCh}
	terminal := &testkit.FakeTerminalService{
		ListResult:    port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-main", Title: "main", State: "running", Cols: 80, Rows: 24}}},
		AttachResult:  port.TerminalAttachResult{TerminalID: "term-main", Channel: 12, Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner},
		SurfaceResult: port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-main", Cols: 80, Rows: 24, Lines: []string{"terminal exited: term-main code:0 exited", "% "}, Cursor: state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"}, State: state.TerminalLiveAttached}},
	}
	initial := state.Root{
		Surface: state.TerminalSurfaceStore{}.ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-main",
			Revision:   9,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"terminal exited: term-main code:0 exited"},
		}).MarkExitedWithMetadata("term-main", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
		Session: state.TerminalSessionStore{}.Attach("term-main", 7, 80, 24).MarkExitedWithMetadata("term-main", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
	}
	runtime := NewInteractiveRuntimeWithWorkbench(initial, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, WorkbenchDeps{Storage: storage})

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	root := runtime.State()
	if pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.Kind != state.PaneTerminalLive || pane.TerminalID != "term-main" {
		t.Fatalf("legacy pane without terminal views should restore as live intent, pane=%#v ok=%v", pane, ok)
	}
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.TerminalID != "term-main" || !binding.Attached || binding.Channel != 12 {
		t.Fatalf("legacy pane intent should reattach through core, binding=%#v ok=%v", binding, ok)
	}
	if root.Surface.State != state.TerminalLiveAttached || root.Session.State == state.TerminalLiveExited {
		t.Fatalf("core running lifecycle should clear old exited session/surface, session=%#v surface=%#v", root.Session, root.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "restart") || !frameContains(frame, "% ") || !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("running restored terminal should render live prompt without restart CTA, frame=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestInteractiveRuntimeWorkbenchRestoreAlreadyLiveBindingQueriesCoreLifecycle(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	views := state.TerminalViewStore{}.BindPane(binding)
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
			Version:  7,
			Found:    true,
		},
		WatchCh: watchCh,
	}
	initial := state.Root{
		Shell:         shell,
		TerminalViews: views,
		Surface: state.TerminalSurfaceStore{}.ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-main",
			Revision:   9,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"terminal exited: term-main code:0 exited"},
			State:      state.TerminalLiveAttached,
		}).MarkExitedWithMetadata("term-main", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
		Session: state.TerminalSessionStore{}.Attach("term-main", 7, 80, 24).MarkExitedWithMetadata("term-main", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
	}
	terminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-main", Title: "main", State: "running", Cols: 80, Rows: 24}}},
		SurfaceResult: port.TerminalSurfaceResult{
			Ready:          true,
			LifecycleKnown: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-main",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"terminal exited: term-main code:0 exited", "% "},
				Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
				State:      state.TerminalLiveAttached,
			},
		},
	}
	runtime := NewInteractiveRuntimeWithWorkbench(initial, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, WorkbenchDeps{Storage: storage})

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 0 {
		t.Fatalf("already-live restored binding should not reattach, attaches=%#v", terminal.Attaches)
	}
	if len(terminal.Surfaces) == 0 || terminal.Surfaces[0].TerminalID != "term-main" {
		t.Fatalf("restore must query core lifecycle for bound terminal, surfaces=%#v", terminal.Surfaces)
	}
	root := runtime.State()
	if root.Surface.State != state.TerminalLiveAttached || root.Session.State == state.TerminalLiveExited {
		t.Fatalf("core running lifecycle should clear stale exited projection, session=%#v surface=%#v", root.Session, root.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "restart") || !frameContains(frame, "% ") || !frame.Cursor.Visible {
		t.Fatalf("running queried terminal should render prompt without restart CTA, frame=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestInteractiveRuntimeStartupLoadsTerminalPoolTitleAfterWorkbenchRestore(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	watchCh := make(chan port.WorkbenchStorageEvent)
	close(watchCh)
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Title = "shell"
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 11, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID), true))
	storage := &testkit.FakeWorkbenchStorageService{
		LoadResult: port.WorkbenchStorageLoadResult{
			Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
			Version:  7,
			Found:    true,
		},
		WatchCh: watchCh,
	}
	terminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-main", Title: "main", State: "running"}}}}
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{Storage: storage},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Lists) == 0 {
		t.Fatalf("startup should load terminal pool, lists=%#v", terminal.Lists)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, " shell ") || !frameContains(frame, " main ") {
		t.Fatalf("restored terminal chrome should use terminal title after startup list, frame=%#v", frame.Lines)
	}
}

func TestInteractiveRuntimeWorkbenchReloadDoesNotKeepOldFrozenHistory(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
			TerminalViews: state.TerminalViewStore{}.
				BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true)),
			History: state.HistoryStore{
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				Token:      "tok-old",
				Cols:       80,
				Rows:       []state.HistoryRow{{Text: "old-history", LineID: 1}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  80,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	if !frameContains(lastFrame(t, host.Frames()), "old-history") {
		t.Fatalf("expected initial frame to render old frozen history, frames=%#v", host.Frames())
	}

	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-new"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-new", 11, 80, 24, state.TerminalResizeRoleFollower, "surface-new", state.TerminalPaneViewID(state.DefaultPaneID), false))
	if err := runtime.Post(WorkbenchStorageLoadResultMsg{Result: port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  9,
		Found:    true,
	}}); err != nil {
		t.Fatalf("post workbench load result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workbench load result: %v", err)
	}

	if runtime.State().CopyMode.Active {
		t.Fatalf("workbench load must exit stale copy mode, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.TerminalID != "term-old" || runtime.State().History.Token != "" {
		t.Fatalf("workbench load must invalidate stale frozen history window, got %#v", runtime.State().History)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("workbench load must not keep rendering old frozen history, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeWorkbenchReloadIgnoresDelayedOldHistoryWindow(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	pendingHistory, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID:         7,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-old",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest pending: %v", err)
	}
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
			TerminalViews: state.TerminalViewStore{}.
				BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true)),
			History: pendingHistory,
			CopyMode: state.CopyModeStore{}.BindLatest(
				state.DefaultPaneID,
				state.TerminalPaneViewID(state.DefaultPaneID),
				"term-old",
				7,
				80,
				20,
			),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{},
	)

	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-new"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-new", 11, 80, 24, state.TerminalResizeRoleFollower, "surface-new", state.TerminalPaneViewID(state.DefaultPaneID), false))
	if err := runtime.Post(WorkbenchStorageLoadResultMsg{Result: port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  9,
		Found:    true,
	}}); err != nil {
		t.Fatalf("post workbench load result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workbench load result: %v", err)
	}

	delayed := state.HistoryWindow{
		TerminalID: "term-old",
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		Token:      "tok-delayed",
		Cols:       80,
		Op:         state.HistoryWindowReplace,
		Rows:       []state.HistoryRow{{Text: "delayed-old-history", LineID: 8}},
	}
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 7, Window: delayed}}); err != nil {
		t.Fatalf("post delayed old history window: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed old history window: %v", err)
	}

	if runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 0 {
		t.Fatalf("workbench reload must ignore delayed old history window, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("workbench reload must keep copy mode inactive after delayed old history window, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "delayed-old-history") {
		t.Fatalf("workbench reload must not render delayed old history window, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeWorkbenchReloadIgnoresDelayedOldHistoryError(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	pendingHistory, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID:         7,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-old",
		Cols:       80,
	})
	if err != nil {
		t.Fatalf("begin latest pending: %v", err)
	}
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
			TerminalViews: state.TerminalViewStore{}.
				BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true)),
			History: pendingHistory,
			CopyMode: state.CopyModeStore{}.BindLatest(
				state.DefaultPaneID,
				state.TerminalPaneViewID(state.DefaultPaneID),
				"term-old",
				7,
				80,
				20,
			),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{},
	)

	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-new"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-new", 11, 80, 24, state.TerminalResizeRoleFollower, "surface-new", state.TerminalPaneViewID(state.DefaultPaneID), false))
	if err := runtime.Post(WorkbenchStorageLoadResultMsg{Result: port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  9,
		Found:    true,
	}}); err != nil {
		t.Fatalf("post workbench load result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workbench load result: %v", err)
	}
	beforeSurfaceErr := runtime.State().Surface.Err
	beforeSessionErr := runtime.State().Session.LastError

	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 7},
		Err:    errors.New("delayed old history failed"),
	}); err != nil {
		t.Fatalf("post delayed old history error: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed old history error: %v", err)
	}

	if runtime.State().Surface.Err != beforeSurfaceErr || runtime.State().Session.LastError != beforeSessionErr {
		t.Fatalf("workbench reload must ignore delayed old history error without replacing current ui error state, state=%#v", runtime.State())
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "delayed old history failed") {
		t.Fatalf("workbench reload must not render delayed old history error, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeWorkbenchReloadIgnoresDelayedOldAttachResult(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntimeWithWorkbench(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
			TerminalViews: state.TerminalViewStore{}.
				BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true)),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
		WorkbenchDeps{},
	)

	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].Kind = state.PaneTerminalLive
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-new"
	views := state.TerminalViewStore{}.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-new", 11, 80, 24, state.TerminalResizeRoleFollower, "surface-new", state.TerminalPaneViewID(state.DefaultPaneID), false))
	if err := runtime.Post(WorkbenchStorageLoadResultMsg{Result: port.WorkbenchStorageLoadResult{
		Snapshot: state.SnapshotRootWorkbenchForStorage(state.Root{Shell: shell, TerminalViews: views}),
		Version:  9,
		Found:    true,
	}}); err != nil {
		t.Fatalf("post workbench load result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workbench load result: %v", err)
	}

	if err := runtime.Post(LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-old",
		Channel:      99,
		Cols:         100,
		Rows:         30,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "surface-old",
		ViewID:       "pane:stale-old",
		CanResize:    true,
	}}); err != nil {
		t.Fatalf("post delayed old attach result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed old attach result: %v", err)
	}

	binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.TerminalID != "term-new" || binding.Channel != 0 || binding.Attached || binding.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("workbench reload must ignore delayed old attach result without rebinding active pane, binding=%#v ok=%v", binding, ok)
	}
	if runtime.State().Shell.Workspace.Tabs[0].Panes[0].TerminalID != "term-new" {
		t.Fatalf("workbench reload must keep reloaded pane terminal, shell=%#v", runtime.State().Shell)
	}
	if channel, ok := runtime.State().Session.InputChannelFor("term-new"); ok && channel == 99 {
		t.Fatalf("workbench reload must not move delayed old attach channel onto reloaded session, session=%#v", runtime.State().Session)
	}
}
