package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type ClipboardDeps struct {
	Storage port.ClipboardStorageService
	Ref     state.ClipboardStorageRef
	Logger  *slog.Logger
}

type ClipboardStorageLoadRequestMsg struct {
	Reason string
}

func (ClipboardStorageLoadRequestMsg) isMsg() {}

type ClipboardStorageWatchRequestMsg struct{}

func (ClipboardStorageWatchRequestMsg) isMsg() {}

type ClipboardStorageChangedMsg struct {
	Event port.ClipboardStorageEvent
	Err   error
}

func (ClipboardStorageChangedMsg) isMsg() {}

type ClipboardStorageLoadResultMsg struct {
	Result port.ClipboardStorageLoadResult
	Err    error
}

func (ClipboardStorageLoadResultMsg) isMsg() {}

type ClipboardStoragePersistRequestMsg struct {
	Reason string
}

func (ClipboardStoragePersistRequestMsg) isMsg() {}

type ClipboardStoragePersistResultMsg struct {
	Result port.ClipboardStorageSaveResult
	Err    error
}

func (ClipboardStoragePersistResultMsg) isMsg() {}

func NewClipboardStorageReducer(deps ClipboardDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case ClipboardStorageWatchRequestMsg:
			return reduceClipboardStorageWatchRequest(root, deps)
		case ClipboardStorageChangedMsg:
			return reduceClipboardStorageChanged(root, msg)
		case ClipboardStorageLoadRequestMsg:
			return reduceClipboardStorageLoadRequest(root, deps)
		case ClipboardStorageLoadResultMsg:
			return reduceClipboardStorageLoadResult(root, msg)
		case ClipboardStoragePersistRequestMsg:
			return reduceClipboardStoragePersistRequest(root, msg, deps)
		case ClipboardStoragePersistResultMsg:
			return reduceClipboardStoragePersistResult(root, msg)
		default:
			return root, nil
		}
	}
}

func reduceClipboardStorageWatchRequest(root state.Root, deps ClipboardDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		return root, nil
	}
	ref := clipboardStorageRef(root, deps)
	return root, []Effect{StreamEffect{Token: CancelToken("clipboard.storage.watch"), Run: func(ctx context.Context, post func(Msg)) {
		events, err := deps.Storage.WatchClipboard(ctx, ref)
		if err != nil {
			logEffectError(deps.Logger, "clipboard.storage.watch", err, "key", ref.Key, "owner_id", ref.OwnerID)
			if isContextLifecycleError(err) {
				return
			}
			post(ClipboardStorageChangedMsg{Err: err})
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				post(ClipboardStorageChangedMsg{Event: event})
			}
		}
	}}}
}

func reduceClipboardStorageChanged(root state.Root, msg ClipboardStorageChangedMsg) (state.Root, []Effect) {
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	if root.Clipboard.ShouldIgnoreEvent(msg.Event.Version) {
		root.Clipboard = root.Clipboard.MarkEvent(msg.Event.Version)
		return root.Advance(), nil
	}
	root.Clipboard = root.Clipboard.MarkEvent(msg.Event.Version)
	return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
		return ClipboardStorageLoadRequestMsg{Reason: "storage.changed"}
	}}}
}

func reduceClipboardStorageLoadRequest(root state.Root, deps ClipboardDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		return root, nil
	}
	ref := clipboardStorageRef(root, deps)
	return root, []Effect{FuncEffect{Async: true, ForceSyncInTests: true, Run: func(ctx context.Context) Msg {
		result, err := deps.Storage.LoadClipboard(ctx, ref)
		logEffectError(deps.Logger, "clipboard.storage.load", err, "key", ref.Key, "owner_id", ref.OwnerID)
		if isContextLifecycleError(err) {
			return nil
		}
		return ClipboardStorageLoadResultMsg{Result: result, Err: err}
	}}}
}

func reduceClipboardStorageLoadResult(root state.Root, msg ClipboardStorageLoadResultMsg) (state.Root, []Effect) {
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	if !msg.Result.Found {
		root.Clipboard = root.Clipboard.MarkApplied(0)
		return root.Advance(), nil
	}
	loaded, err := msg.Result.Snapshot.ToClipboardStore()
	if err != nil {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard.storage", Body: errorString(err)})
		return root.Advance(), nil
	}
	if root.Clipboard.PendingChangesAreMergeable() {
		root.Clipboard = root.Clipboard.MergeLoadedEntries(loaded).MarkMerged(msg.Result.Version)
		root.Shell = root.Shell.SetClipboardHistorySelectedIndex(root.Shell.EnsureDefaults().Overlay.SelectedIndex, len(state.ClipboardHistoryItems(root)))
		return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
			return ClipboardStoragePersistRequestMsg{Reason: "merge"}
		}}}
	}
	if root.Clipboard.HasPendingLocalChanges() {
		root.Clipboard = root.Clipboard.MarkMerged(msg.Result.Version)
		root.Shell = root.Shell.SetClipboardHistorySelectedIndex(root.Shell.EnsureDefaults().Overlay.SelectedIndex, len(state.ClipboardHistoryItems(root)))
		return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
			return ClipboardStoragePersistRequestMsg{Reason: "local-rebase"}
		}}}
	}
	loaded = loaded.MarkApplied(msg.Result.Version)
	root.Clipboard = loaded
	root.Shell = root.Shell.SetClipboardHistorySelectedIndex(root.Shell.EnsureDefaults().Overlay.SelectedIndex, len(state.ClipboardHistoryItems(root)))
	return root.Advance(), nil
}

func reduceClipboardStoragePersistRequest(root state.Root, _ ClipboardStoragePersistRequestMsg, deps ClipboardDeps) (state.Root, []Effect) {
	if deps.Storage == nil {
		return root, nil
	}
	ref := clipboardStorageRef(root, deps)
	snapshot := state.SnapshotClipboardForStorage(root.Clipboard)
	expectedVersion := root.Clipboard.SaveVersion()
	return root, []Effect{FuncEffect{Async: true, ForceSyncInTests: true, Run: func(ctx context.Context) Msg {
		result, err := deps.Storage.SaveClipboard(ctx, port.ClipboardStorageSaveRequest{
			Ref:             ref.WithVersion(expectedVersion),
			Snapshot:        snapshot,
			CheckVersion:    true,
			ExpectedVersion: expectedVersion,
		})
		logEffectError(deps.Logger, "clipboard.storage.save", err, "key", ref.Key, "owner_id", ref.OwnerID, "expected_version", expectedVersion)
		if isContextLifecycleError(err) {
			return nil
		}
		return ClipboardStoragePersistResultMsg{Result: result, Err: err}
	}}}
}

func reduceClipboardStoragePersistResult(root state.Root, msg ClipboardStoragePersistResultMsg) (state.Root, []Effect) {
	if msg.Err != nil {
		if isContextLifecycleError(msg.Err) {
			return root, nil
		}
		if errors.Is(msg.Err, port.ErrClipboardStorageConflict) {
			root.Clipboard = root.Clipboard.MarkConflict(root.Clipboard.SaveVersion())
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard.storage", Body: "conflict: reloading"})
			return root.Advance(), []Effect{FuncEffect{Run: func(context.Context) Msg {
				return ClipboardStorageLoadRequestMsg{Reason: "conflict"}
			}}}
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard.storage", Body: errorString(msg.Err)})
		return root.Advance(), nil
	}
	root.Clipboard = root.Clipboard.MarkSaved(msg.Result.Ref, msg.Result.Version)
	return root.Advance(), nil
}

func clipboardStorageRef(root state.Root, deps ClipboardDeps) state.ClipboardStorageRef {
	if deps.Ref.AppID != "" || deps.Ref.Key != "" {
		return deps.Ref
	}
	return state.DefaultClipboardStorageRef(root.Shell.EnsureDefaults().Workspace.ID)
}
