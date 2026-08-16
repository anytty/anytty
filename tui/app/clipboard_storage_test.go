package app

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
)

func TestClipboardStorageReducerLoadsEntriesFromCoreStorage(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{
		LoadResult: port.ClipboardStorageLoadResult{
			Snapshot: state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("persisted")),
			Version:  3,
			Found:    true,
		},
	}
	reducer := NewClipboardStorageReducer(ClipboardDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell().OpenClipboardHistory()}

	root, effects := reducer(root, ClipboardStorageLoadRequestMsg{Reason: "open"})
	if len(effects) != 1 {
		t.Fatalf("expected load effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, msg)

	if len(storage.Loads) != 1 || storage.Loads[0].Key != state.ClipboardStorageKeyRoot {
		t.Fatalf("unexpected storage loads %#v", storage.Loads)
	}
	if len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "persisted" || root.Clipboard.SaveVersion() != 3 {
		t.Fatalf("expected loaded clipboard history, got %#v", root.Clipboard)
	}
}

func TestCopySelectionPersistsLocalBufferWhenSystemClipboardFails(t *testing.T) {
	core := &testkit.FakeCoreClient{}
	clipboard := &testkit.FakeClipboardService{WriteErr: errors.New("clipboard unavailable")}
	storage := &testkit.FakeClipboardStorageService{}
	reducer := ComposeReducers(
		NewClipboardActionReducer(ClipboardActionDeps{}),
		NewClipboardStorageReducer(ClipboardDeps{Storage: storage}),
		NewCopyModeReducer(CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20}),
	)
	root := state.Root{
		Shell: state.DefaultShell(),
		History: state.HistoryStore{EndpointID: "endpoint-a", TerminalID: "term-a", Token: "tok-a", Cols: 80, Rows: []state.HistoryRow{
			{Text: "cross-endpoint text", LineID: 1},
		}},
		CopyMode: state.CopyModeStore{
			Active: true, EndpointID: "endpoint-a", TerminalID: "term-a", BoundToken: "tok-a",
			Selection: &state.CopySelection{
				Anchor: state.CopyPosition{Row: 0, Col: 0},
				Focus:  state.CopyPosition{Row: 0, Col: 19},
			},
		},
	}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "y"}})
	if len(effects) != 2 {
		t.Fatalf("local copy should schedule system clipboard and immediate persistence, got %#v", effects)
	}
	if len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "cross-endpoint text" {
		t.Fatalf("selection must reach the local buffer before system clipboard IO, got %#v", root.Clipboard)
	}
	persistMsg := effects[1].(FuncEffect).Run(context.Background())
	root, persistEffects := reducer(root, persistMsg)
	if len(persistEffects) != 1 {
		t.Fatalf("expected immediate JSON save effect, got %#v", persistEffects)
	}
	resultMsg := persistEffects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	copyMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, copyMsg)
	if len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "cross-endpoint text" {
		t.Fatalf("system clipboard failure must retain materialized local text, got %#v", root.Clipboard)
	}
	if len(root.Shell.Toasts) == 0 || root.Shell.Toasts[len(root.Shell.Toasts)-1].Title != "Copied to buffer" {
		t.Fatalf("expected clipboard fallback feedback, got %#v", root.Shell.Toasts)
	}

	if len(storage.Saves) != 1 || storage.Saves[0].Snapshot.Entries[0].Text != "cross-endpoint text" {
		t.Fatalf("fallback buffer should be stored as materialized JSON text, saves=%#v", storage.Saves)
	}
	if len(core.CopyRequests) != 0 {
		t.Fatalf("visible selection must not depend on its endpoint, requests=%#v", core.CopyRequests)
	}
	if len(clipboard.Writes) != 1 || clipboard.LastCopy() != "" {
		t.Fatalf("expected failed system clipboard attempt without false LastCopy, clipboard=%#v", clipboard)
	}
}

func TestCopyModeEnterPersistsClipboardHistoryBeforeSystemClipboardCompletes(t *testing.T) {
	core := &testkit.FakeCoreClient{}
	clipboard := &testkit.FakeClipboardService{}
	storage := &testkit.FakeClipboardStorageService{}
	reducer := ComposeReducers(
		NewClipboardActionReducer(ClipboardActionDeps{}),
		NewClipboardStorageReducer(ClipboardDeps{Storage: storage}),
		NewCopyModeReducer(CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20}),
	)
	root := state.Root{
		Shell: state.DefaultShell(),
		History: state.HistoryStore{TerminalID: "term-1", Token: "tok-1", Cols: 80, Rows: []state.HistoryRow{
			{Text: "alpha", LineID: 1},
		}},
		CopyMode: state.CopyModeStore{
			Active: true, TerminalID: "term-1", BoundToken: "tok-1",
			Selection: &state.CopySelection{
				Anchor: state.CopyPosition{Row: 0, Col: 0},
				Focus:  state.CopyPosition{Row: 0, Col: 5},
			},
		},
	}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 2 {
		t.Fatalf("enter should schedule system clipboard and immediate persistence, got %#v", effects)
	}
	if len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "alpha" || !root.CopyMode.CopyPending {
		t.Fatalf("enter must commit local text before effects complete, root=%#v", root)
	}
	copyEffect := effects[0].(FuncEffect)
	persistMsg := effects[1].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, persistMsg)
	if len(effects) != 1 {
		t.Fatalf("expected storage save effect, got %#v", effects)
	}
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, _ = reducer(root, resultMsg)

	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.Entries) != 1 || storage.Saves[0].Snapshot.Entries[0].Text != "alpha" {
		t.Fatalf("expected persisted clipboard entry, saves=%#v", storage.Saves)
	}
	if root.Clipboard.SaveVersion() != 1 {
		t.Fatalf("expected saved version 1, got %#v", root.Clipboard)
	}
	if len(clipboard.Writes) != 0 || len(core.CopyRequests) != 0 {
		t.Fatalf("JSON persistence must not wait for clipboard or endpoint IO, clipboard=%#v backend=%#v", clipboard.Writes, core.CopyRequests)
	}
	copyMsg := copyEffect.Run(context.Background())
	root, _ = reducer(root, copyMsg)
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "alpha" || root.CopyMode.Active {
		t.Fatalf("system clipboard completion should finish enter copy, clipboard=%#v copy=%#v", clipboard.Writes, root.CopyMode)
	}
}

func TestClipboardHistoryOpenRequestsStorageLoad(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{}
	reducer := ComposeReducers(
		NewClipboardActionReducer(ClipboardActionDeps{}),
		NewClipboardStorageReducer(ClipboardDeps{Storage: storage}),
		NewShellReducer(),
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Clipboard: &testkit.FakeClipboardService{}, Rows: 20}),
	)
	root := state.Root{CopyMode: state.CopyModeStore{Active: true}}

	root, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "H"}})
	if !root.Shell.Overlay.Open || root.Shell.Overlay.Kind != state.OverlayClipboardHistory {
		t.Fatalf("expected clipboard history overlay, got %#v", root.Shell.Overlay)
	}
	if len(effects) != 1 {
		t.Fatalf("expected storage load request effect, got %#v", effects)
	}
	loadMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, loadMsg)
	if len(effects) != 1 {
		t.Fatalf("expected storage load effect, got %#v", effects)
	}
	root, _ = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(storage.Loads) != 1 {
		t.Fatalf("expected one storage load, got %#v", storage.Loads)
	}
}

func TestClipboardHistoryNewEntryPersistsStorage(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{}
	reducer := ComposeReducers(NewShellReducer(), NewUIInputReducer(), NewClipboardStorageReducer(ClipboardDeps{Storage: storage}))
	root := state.Root{Shell: state.DefaultShell().OpenClipboardHistory()}

	root, effects := reduceClipboardHistoryNew(root)
	if len(effects) != 0 || root.Shell.EnsureDefaults().Overlay.Kind != state.OverlayPrompt || root.Shell.EnsureDefaults().Overlay.Prompt.Purpose != "clipboard.new" {
		t.Fatalf("expected new clipboard prompt, root=%#v effects=%#v", root, effects)
	}
	root.Shell = root.Shell.SetPromptValue("manual\nentry")
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 1 {
		t.Fatalf("expected prompt submit effect, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 || len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "manual\nentry" {
		t.Fatalf("expected new clipboard entry and persist request, root=%#v effects=%#v", root, effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected storage save effect, got %#v", effects)
	}
	root, _ = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.Entries) != 1 || storage.Saves[0].Snapshot.Entries[0].Text != "manual\nentry" {
		t.Fatalf("expected new clipboard save, saves=%#v root=%#v", storage.Saves, root)
	}
}

func TestClipboardHistoryEditAndDeletePersistStorage(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{}
	reducer := ComposeReducers(NewShellReducer(), NewUIInputReducer(), NewClipboardStorageReducer(ClipboardDeps{Storage: storage}))
	root := state.Root{
		Shell: state.DefaultShell().OpenClipboardHistory(),
		Clipboard: state.ClipboardStore{
			Entries: []state.ClipboardEntry{
				{ID: "clip:alpha", Title: "alpha", Text: "alpha", Preview: "alpha"},
				{ID: "clip:beta", Title: "beta", Text: "beta", Preview: "beta"},
			},
		},
	}

	root, effects := reduceClipboardHistoryEdit(root)
	if len(effects) != 0 || root.Shell.EnsureDefaults().Overlay.Kind != state.OverlayPrompt {
		t.Fatalf("expected edit prompt, root=%#v effects=%#v", root, effects)
	}
	root.Shell = root.Shell.SetPromptValue("edited")
	root, effects = reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}})
	if len(effects) != 1 {
		t.Fatalf("expected prompt submit effect, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected edit persist request, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected edit storage save, got %#v", effects)
	}
	root, _ = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(storage.Saves) != 1 || storage.Saves[0].Snapshot.Entries[0].Text != "edited" {
		t.Fatalf("expected edited clipboard save, saves=%#v", storage.Saves)
	}

	root.Shell = root.Shell.OpenClipboardHistory()
	root, effects = reduceClipboardHistoryDelete(root)
	if len(effects) != 1 {
		t.Fatalf("expected delete persist request, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected delete storage save, got %#v", effects)
	}
	root, _ = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(storage.Saves) != 2 || len(storage.Saves[1].Snapshot.Entries) != 1 {
		t.Fatalf("expected deleted clipboard save, saves=%#v", storage.Saves)
	}
}

func TestClipboardStorageConflictReloads(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{
		CurrentVersion: 9,
		LoadResult: port.ClipboardStorageLoadResult{
			Snapshot: state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("remote")),
			Version:  9,
			Found:    true,
		},
	}
	reducer := NewClipboardStorageReducer(ClipboardDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell(), Clipboard: state.ClipboardStore{}.WithCopiedText("local")}

	root, effects := reducer(root, ClipboardStoragePersistRequestMsg{Reason: "copy"})
	resultMsg := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, resultMsg)
	if len(effects) != 1 || !root.Clipboard.Conflict {
		t.Fatalf("expected conflict reload effect, root=%#v effects=%#v", root, effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected reload storage effect, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(root.Clipboard.Entries) != 2 || root.Clipboard.Entries[0].Text != "local" || root.Clipboard.Entries[1].Text != "remote" {
		t.Fatalf("expected local clipboard merged with remote after conflict reload, got %#v", root.Clipboard)
	}
	if len(effects) != 1 {
		t.Fatalf("expected merged persist request after conflict reload, got %#v", effects)
	}
}

func TestClipboardStorageLoadMergesPendingLocalCopy(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{
		LoadResult: port.ClipboardStorageLoadResult{
			Snapshot: state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("remote")),
			Version:  4,
			Found:    true,
		},
	}
	reducer := NewClipboardStorageReducer(ClipboardDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell(), Clipboard: state.ClipboardStore{}.WithCopiedText("local")}

	root, effects := reducer(root, ClipboardStorageLoadRequestMsg{Reason: "startup"})
	loadResult := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, loadResult)
	if len(effects) != 1 {
		t.Fatalf("expected merge persist request, got %#v", effects)
	}
	if len(root.Clipboard.Entries) != 2 || root.Clipboard.Entries[0].Text != "local" || root.Clipboard.Entries[1].Text != "remote" || !root.Clipboard.Dirty {
		t.Fatalf("expected local copy merged ahead of remote storage, got %#v", root.Clipboard)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected merged storage save, got %#v", effects)
	}
	_ = effects[0].(FuncEffect).Run(context.Background())
	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.Entries) != 2 || storage.Saves[0].ExpectedVersion != 4 {
		t.Fatalf("expected merged save against loaded version, saves=%#v", storage.Saves)
	}
}

func TestClipboardStorageLoadRebasesPendingDeleteWithoutRevivingRemoteEntry(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{
		LoadResult: port.ClipboardStorageLoadResult{
			Snapshot: state.SnapshotClipboardForStorage(state.ClipboardStore{}.WithCopiedText("remote").WithCopiedText("deleted")),
			Version:  4,
			Found:    true,
		},
	}
	reducer := NewClipboardStorageReducer(ClipboardDeps{Storage: storage})
	root := state.Root{
		Shell:     state.DefaultShell(),
		Clipboard: state.ClipboardStore{}.WithCopiedText("remote").WithCopiedText("deleted"),
	}
	root.Clipboard = root.Clipboard.MarkSaved(state.DefaultClipboardStorageRef(state.DefaultWorkspaceID), 3)
	root.Clipboard = root.Clipboard.DeleteEntry(root.Clipboard.Entries[0].ID, root.Clipboard.Entries[0].Text)

	root, effects := reducer(root, ClipboardStorageLoadRequestMsg{Reason: "storage.changed"})
	loadResult := effects[0].(FuncEffect).Run(context.Background())
	root, effects = reducer(root, loadResult)
	if len(root.Clipboard.Entries) != 1 || root.Clipboard.Entries[0].Text != "remote" {
		t.Fatalf("pending delete should not be revived by storage reload, got %#v", root.Clipboard)
	}
	if len(effects) != 1 {
		t.Fatalf("expected rebase persist request, got %#v", effects)
	}
	root, effects = reducer(root, effects[0].(FuncEffect).Run(context.Background()))
	if len(effects) != 1 {
		t.Fatalf("expected rebased storage save, got %#v", effects)
	}
	_ = effects[0].(FuncEffect).Run(context.Background())
	if len(storage.Saves) != 1 || len(storage.Saves[0].Snapshot.Entries) != 1 || storage.Saves[0].Snapshot.Entries[0].Text != "remote" || storage.Saves[0].ExpectedVersion != 4 {
		t.Fatalf("expected rebased delete save against loaded version, saves=%#v", storage.Saves)
	}
}

func TestClipboardStorageContextCanceledStaysSilent(t *testing.T) {
	storage := &testkit.FakeClipboardStorageService{
		LoadErr:  context.Canceled,
		SaveErr:  context.Canceled,
		WatchErr: context.Canceled,
	}
	reducer := NewClipboardStorageReducer(ClipboardDeps{Storage: storage})
	root := state.Root{Shell: state.DefaultShell()}

	root, effects := reducer(root, ClipboardStorageWatchRequestMsg{})
	if len(effects) != 1 {
		t.Fatalf("expected watch effect, got %#v", effects)
	}
	effects[0].(StreamEffect).Run(context.Background(), func(msg Msg) {
		root, _ = reducer(root, msg)
	})
	if len(root.Shell.Toasts) != 0 {
		t.Fatalf("context canceled watch should not toast, got %#v", root.Shell.Toasts)
	}

	root, effects = reducer(root, ClipboardStorageLoadRequestMsg{Reason: "test"})
	if msg := effects[0].(FuncEffect).Run(context.Background()); msg != nil {
		t.Fatalf("context canceled load should not post result, got %#v", msg)
	}
	root.Clipboard = root.Clipboard.WithCopiedText("alpha")
	root, effects = reducer(root, ClipboardStoragePersistRequestMsg{Reason: "test"})
	if msg := effects[0].(FuncEffect).Run(context.Background()); msg != nil {
		t.Fatalf("context canceled save should not post result, got %#v", msg)
	}
	if len(root.Shell.Toasts) != 0 {
		t.Fatalf("context canceled clipboard effects should stay silent, got %#v", root.Shell.Toasts)
	}
}
