package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"reflect"
	"strings"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

func TestShortcutBindingsHaveCanonicalInvocationsAndDispatcherHandlers(t *testing.T) {
	for _, binding := range input.BindingCatalog() {
		intent, ok := shortcutIntentForInvocation(binding.Invocation, input.InputEvent{})
		if !ok {
			t.Fatalf("shortcut binding %q invocation=%#v has no app dispatcher handler", binding.ID, binding.Invocation)
		}
		if intent.Kind == input.IntentNone || intent.Kind == input.IntentShortcutAction {
			t.Fatalf("shortcut binding %q did not dispatch to reducer intent: %#v", binding.ID, intent)
		}
	}
}

func TestDefaultShortcutActionsReachObservableOwnerBoundary(t *testing.T) {
	seen := map[string]bool{}
	for _, binding := range shortcut.DefaultBindings() {
		invocation, _, err := actiondomain.ParseInvocation(binding.Action)
		if err != nil {
			t.Fatalf("parse default action %q: %v", binding.Action, err)
		}
		signature := string(invocation.ID)
		for name, value := range invocation.Params {
			signature = fmt.Sprintf("%s.%s=%d", signature, name, value)
		}
		if seen[signature] {
			continue
		}
		seen[signature] = true
		t.Run(signature, func(t *testing.T) {
			root := defaultActionExecutionRoot(t, invocation.ID)
			execution := runDefaultActionToOwnerBoundary(t, root, invocation)
			if len(execution.root.Shell.EnsureDefaults().Toasts) > len(root.Shell.EnsureDefaults().Toasts) {
				toast := execution.root.Shell.EnsureDefaults().Toasts[len(execution.root.Shell.EnsureDefaults().Toasts)-1]
				if toast.Severity == state.ToastWarning || toast.Severity == state.ToastError {
					t.Fatalf("available default action reached failure toast instead of owner boundary: %#v", toast)
				}
			}
			if !execution.reachedOwnerBoundary(invocation.ID, root) {
				t.Fatalf("default action did not reach reducer state or service owner: messages=%v", execution.messageTypes)
			}
			assertDefaultActionServiceOwner(t, invocation, execution)
		})
	}
	if len(seen) != 158 {
		t.Fatalf("default shortcut execution matrix changed without KS015 classification: got=%d want=158", len(seen))
	}
}

type defaultActionEndpointConnections struct {
	store state.EndpointStore
}

func (service defaultActionEndpointConnections) LoadConnections(context.Context) (state.EndpointStore, error) {
	return service.store, nil
}

func (service defaultActionEndpointConnections) SampleConnection(context.Context, state.EndpointID) (state.EndpointConnectionSnapshot, bool, error) {
	return state.EndpointConnectionSnapshot{}, false, nil
}

func (service defaultActionEndpointConnections) ApplyConnectionSettings(context.Context, state.EndpointID, state.EndpointConnectionPolicy, map[string]*int) (state.EndpointStore, error) {
	return service.store, nil
}

func (service defaultActionEndpointConnections) SetEndpointEnabled(context.Context, state.EndpointID, bool) (state.EndpointStore, error) {
	return service.store, nil
}

type defaultActionExecution struct {
	root             state.Root
	terminal         *testkit.FakeTerminalService
	core             *testkit.FakeCoreClient
	clipboard        *testkit.FakeClipboardService
	workbenchStorage *testkit.FakeWorkbenchStorageService
	clipboardStorage *testkit.FakeClipboardStorageService
	messageTypes     []string
	quit             bool
}

func runDefaultActionToOwnerBoundary(t *testing.T, root state.Root, invocation actiondomain.Invocation) defaultActionExecution {
	t.Helper()
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{EndpointID: "west", TerminalID: "term-1", Channel: 11},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{
			{EndpointID: "west", TerminalID: "term-1", Title: "main", State: "running", Tags: map[string]string{"role": "shell"}},
			{EndpointID: "west", TerminalID: "term-2", Title: "logs", State: "exited", Tags: map[string]string{"role": "build"}},
		}},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "copy-token", 98, 0, []state.HistoryRow{{Text: "latest", LineID: 30}})}},
		OlderResponses:  []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowPrepend, "term-1", "copy-token", 98, 0, []state.HistoryRow{{Text: "older", LineID: 5}})}},
		NewerResponses:  []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowAppend, "term-1", "copy-token", 98, 0, []state.HistoryRow{{Text: "newer", LineID: 40}})}},
		OldestResponses: []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "copy-token", 98, 0, []state.HistoryRow{{Text: "oldest", LineID: 1}})}},
		CopyResponses:   []port.HistoryCopyRangeResult{{Text: "alpha\nbravo"}},
		SearchResponses: []port.HistorySearchResult{{
			Found: true,
			Start: state.CopyLogicalPosition{Valid: true, LineID: 20},
			End:   state.CopyLogicalPosition{Valid: true, LineID: 20, Col: 5},
			Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "copy-token", 98, 0,
				[]state.HistoryRow{{Text: "bravo", LineID: 20}}),
		}},
	}
	clipboard := &testkit.FakeClipboardService{ReadResult: port.ClipboardReadResult{Text: "system clip"}}
	workbenchStorage := &testkit.FakeWorkbenchStorageService{}
	clipboardStorage := &testkit.FakeClipboardStorageService{}
	copyDeps := CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 3}
	connectionProjection := root.Endpoints
	if item, ok := connectionProjection.Endpoint("west"); ok {
		item.Label = "West refreshed"
		connectionProjection = connectionProjection.Upsert(item)
	}
	liveDeps := LiveDeps{Terminal: terminal, EndpointConnections: defaultActionEndpointConnections{store: connectionProjection}}
	reducer := ComposeReducers(
		NewBackNavigationReducer(copyDeps), NewClipboardActionReducer(ClipboardActionDeps{Core: core, Clipboard: clipboard, Terminal: terminal}), NewShellReducer(), NewUIInputReducer(),
		NewEndpointConnectionsReducer(liveDeps),
		NewEndpointStatusReducer(liveDeps), NewEndpointDefaultsReducer(liveDeps), NewPromptPathCompletionReducer(liveDeps),
		newTerminalPoolReducerPrepared(liveDeps),
		NewWorkbenchStorageReducer(WorkbenchDeps{Storage: workbenchStorage, SkipInitialLoad: true}),
		NewClipboardStorageReducer(ClipboardDeps{Storage: clipboardStorage}),
		NewCopyModeReducer(copyDeps), NewCopyModeResizeRebindReducer(copyDeps),
		NewTerminalInputRouterReducer(liveDeps), newLiveReducerPrepared(liveDeps), NewTerminalLayoutResizeReducer(),
	)
	queue := []Msg{ShellShortcutActionMsg{Invocation: invocation}}
	execution := defaultActionExecution{
		root: root, terminal: terminal, core: core, clipboard: clipboard,
		workbenchStorage: workbenchStorage, clipboardStorage: clipboardStorage,
	}
	for steps := 0; len(queue) > 0; steps++ {
		if steps >= 96 {
			t.Fatalf("default action effect chain did not settle: invocation=%#v messages=%v", invocation, execution.messageTypes)
		}
		msg := queue[0]
		queue = queue[1:]
		execution.messageTypes = append(execution.messageTypes, fmt.Sprintf("%T", msg))
		if _, ok := msg.(QuitMsg); ok {
			execution.quit = true
			continue
		}
		next, effects := reducer(execution.root, msg)
		execution.root = next
		queue = append(queue, runDefaultActionEffects(effects)...)
	}
	return execution
}

func runDefaultActionEffects(effects []Effect) []Msg {
	var messages []Msg
	for _, effect := range effects {
		switch effect := effect.(type) {
		case FuncEffect:
			if effect.Run == nil || (effect.Async && !effect.ForceSyncInTests) {
				continue
			}
			if msg := effect.Run(context.Background()); msg != nil {
				messages = append(messages, msg)
			}
		case BatchEffect:
			messages = append(messages, runDefaultActionEffects(effect.Effects)...)
		}
	}
	return messages
}

func (execution defaultActionExecution) reachedOwnerBoundary(id actiondomain.ID, before state.Root) bool {
	coreCalls := len(execution.core.LatestRequests) + len(execution.core.OlderRequests) + len(execution.core.NewerRequests) +
		len(execution.core.OldestRequests) + len(execution.core.CopyRequests) + len(execution.core.SearchRequests) + len(execution.core.ReleaseRequests)
	if execution.quit || terminalServiceCallCount(execution.terminal) > 0 || len(execution.clipboard.Writes) > 0 ||
		len(execution.workbenchStorage.Saves) > 0 || len(execution.clipboardStorage.Saves) > 0 || coreCalls > 0 {
		return true
	}
	after := execution.root
	before.Generation = 0
	after.Generation = 0
	if id != "system.clear_toasts" && id != "system.close_toast" {
		before.Shell.Toasts = nil
		after.Shell.Toasts = nil
	}
	return !reflect.DeepEqual(before, after)
}

func terminalServiceCallCount(service *testkit.FakeTerminalService) int {
	return len(service.Attaches) + len(service.Detaches) + len(service.Lists) + len(service.Creates) +
		len(service.Restarts) + len(service.Reconnects) + len(service.Kills) + len(service.Removes) +
		len(service.Edits) + len(service.TagEdits) + len(service.Inputs) + len(service.Resizes) + len(service.Surfaces)
}

func assertDefaultActionServiceOwner(t *testing.T, invocation actiondomain.Invocation, execution defaultActionExecution) {
	t.Helper()
	id := invocation.ID
	terminal := execution.terminal
	switch id {
	case "menu.terminal_picker", "menu.terminal_pool":
		if len(terminal.Lists) != 1 {
			t.Fatalf("inventory action must list exactly once: lists=%#v messages=%v", terminal.Lists, execution.messageTypes)
		}
	case "menu.copy":
		assertSingleHistoryRef(t, "history latest", len(execution.core.LatestRequests), execution.messageTypes, func() state.TerminalRef {
			return state.NewTerminalRef(execution.core.LatestRequests[0].EndpointID, execution.core.LatestRequests[0].TerminalID)
		})
		assertHistoryReadCount(t, execution, 1)
	case "copy.request_older":
		assertSingleHistoryRef(t, "history older", len(execution.core.OlderRequests), execution.messageTypes, func() state.TerminalRef {
			return state.NewTerminalRef(execution.core.OlderRequests[0].EndpointID, execution.core.OlderRequests[0].TerminalID)
		})
		assertHistoryReadCount(t, execution, 1)
	case "copy.request_newer":
		assertSingleHistoryRef(t, "history newer", len(execution.core.NewerRequests), execution.messageTypes, func() state.TerminalRef {
			return state.NewTerminalRef(execution.core.NewerRequests[0].EndpointID, execution.core.NewerRequests[0].TerminalID)
		})
		assertHistoryReadCount(t, execution, 1)
	case "copy.oldest":
		assertSingleHistoryRef(t, "history oldest", len(execution.core.OldestRequests), execution.messageTypes, func() state.TerminalRef {
			return state.NewTerminalRef(execution.core.OldestRequests[0].EndpointID, execution.core.OldestRequests[0].TerminalID)
		})
		assertHistoryReadCount(t, execution, 1)
	case "copy.search_next", "copy.search_previous":
		assertSingleHistoryRef(t, "history search", len(execution.core.SearchRequests), execution.messageTypes, func() state.TerminalRef {
			return state.NewTerminalRef(execution.core.SearchRequests[0].EndpointID, execution.core.SearchRequests[0].TerminalID)
		})
	case "copy.copy_selection", "copy.accept":
		if len(execution.core.CopyRequests) != 1 || len(execution.clipboard.Writes) != 1 || len(execution.clipboardStorage.Saves) != 1 {
			t.Fatalf("selection must copy from frozen history and persist exactly once: backend=%d writes=%d saves=%d messages=%v", len(execution.core.CopyRequests), len(execution.clipboard.Writes), len(execution.clipboardStorage.Saves), execution.messageTypes)
		}
		expectedReleases := 0
		if id == "copy.accept" {
			expectedReleases = 1
		}
		if len(execution.core.ReleaseRequests) != expectedReleases {
			t.Fatalf("copy action released the wrong history session count: releases=%d want=%d messages=%v", len(execution.core.ReleaseRequests), expectedReleases, execution.messageTypes)
		}
	case "clipboard_history.delete":
		if len(execution.clipboardStorage.Saves) != 1 {
			t.Fatalf("clipboard delete must persist exactly once: saves=%d messages=%v", len(execution.clipboardStorage.Saves), execution.messageTypes)
		}
	case "resize.layout_toggle", "resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down",
		"resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom", "resize.center", "resize.center_x", "resize.center_y", "resize.layout_reset":
		if len(execution.workbenchStorage.Saves) != 1 {
			t.Fatalf("terminal layout action must persist workbench exactly once: saves=%d messages=%v", len(execution.workbenchStorage.Saves), execution.messageTypes)
		}
	}
	gotMutations := terminalMutationVectorFromService(terminal)
	wantMutations := expectedDefaultActionTerminalMutations(invocation)
	if !reflect.DeepEqual(gotMutations, wantMutations) {
		t.Fatalf("terminal mutation operation/TerminalRef vector mismatch: got=%#v want=%#v messages=%v", gotMutations, wantMutations, execution.messageTypes)
	}
	assertNoCoreServiceFallback(t, execution.core, execution.messageTypes)
}

func assertHistoryReadCount(t *testing.T, execution defaultActionExecution, want int) {
	t.Helper()
	got := len(execution.core.LatestRequests) + len(execution.core.OlderRequests) + len(execution.core.NewerRequests) + len(execution.core.OldestRequests) + len(execution.core.CopyRequests)
	if got != want {
		t.Fatalf("history action reached extra or wrong core operations: got=%d want=%d messages=%v", got, want, execution.messageTypes)
	}
}

func assertSingleHistoryRef(t *testing.T, operation string, count int, messages []string, ref func() state.TerminalRef) {
	t.Helper()
	if count != 1 {
		t.Fatalf("%s must call its core owner exactly once: count=%d messages=%v", operation, count, messages)
	}
	if got := ref().Normalize(); got.EndpointID != "west" || got.TerminalID != "term-1" {
		t.Fatalf("%s reached the wrong TerminalRef: got=%#v messages=%v", operation, got, messages)
	}
}

type terminalMutationVector struct {
	Attaches   []state.TerminalRef
	Detaches   []state.TerminalRef
	Creates    []state.TerminalRef
	Restarts   []state.TerminalRef
	Reconnects []state.TerminalRef
	Kills      []state.TerminalRef
	Removes    []state.TerminalRef
	Edits      []state.TerminalRef
	TagEdits   []state.TerminalRef
	Inputs     []state.TerminalRef
	Resizes    []state.TerminalRef
}

func terminalMutationVectorFromService(terminal *testkit.FakeTerminalService) terminalMutationVector {
	var vector terminalMutationVector
	for _, request := range terminal.Attaches {
		vector.Attaches = append(vector.Attaches, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Detaches {
		vector.Detaches = append(vector.Detaches, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Creates {
		vector.Creates = append(vector.Creates, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Restarts {
		vector.Restarts = append(vector.Restarts, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Reconnects {
		vector.Reconnects = append(vector.Reconnects, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Kills {
		vector.Kills = append(vector.Kills, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Removes {
		vector.Removes = append(vector.Removes, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Edits {
		vector.Edits = append(vector.Edits, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.TagEdits {
		vector.TagEdits = append(vector.TagEdits, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Inputs {
		vector.Inputs = append(vector.Inputs, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	for _, request := range terminal.Resizes {
		vector.Resizes = append(vector.Resizes, state.NewTerminalRef(request.EndpointID, request.TerminalID))
	}
	return vector
}

func expectedDefaultActionTerminalMutations(invocation actiondomain.Invocation) terminalMutationVector {
	term1 := state.NewTerminalRef("west", "term-1")
	term2 := state.NewTerminalRef("west", "term-2")
	resizeTerm1 := []state.TerminalRef{term1}
	switch invocation.ID {
	case "panel.close", "panel.detach", "floating.close":
		vector := terminalMutationVector{Detaches: []state.TerminalRef{term1}}
		if invocation.ID == "panel.detach" {
			vector.Resizes = resizeTerm1
		}
		return vector
	case "panel.reconnect":
		return terminalMutationVector{Attaches: []state.TerminalRef{term1}, Reconnects: []state.TerminalRef{term1}}
	case "panel.restart":
		return terminalMutationVector{Attaches: []state.TerminalRef{term1}, Restarts: []state.TerminalRef{term1}}
	case "panel.kill":
		return terminalMutationVector{Kills: []state.TerminalRef{term1}, Resizes: resizeTerm1}
	case "terminal_picker.kill":
		return terminalMutationVector{Kills: []state.TerminalRef{term1}}
	case "tab.kill":
		return terminalMutationVector{Kills: []state.TerminalRef{term2}}
	case "tab.close":
		return terminalMutationVector{Detaches: []state.TerminalRef{term1, term2}}
	case "terminal_pool.restart":
		return terminalMutationVector{Attaches: []state.TerminalRef{term2}, Restarts: []state.TerminalRef{term2}, Resizes: resizeTerm1}
	case "terminal_pool.kill":
		return terminalMutationVector{Kills: []state.TerminalRef{term2}}
	case "terminal_picker.delete":
		return terminalMutationVector{Removes: []state.TerminalRef{term1}}
	case "terminal_pool.delete":
		return terminalMutationVector{Removes: []state.TerminalRef{term2}}
	case "terminal_picker.attach", "terminal_picker.split":
		vector := terminalMutationVector{Attaches: []state.TerminalRef{term1}}
		if invocation.ID == "terminal_picker.split" {
			vector.Resizes = resizeTerm1
		}
		return vector
	case "terminal_pool.attach", "terminal_pool.attach_tab":
		return terminalMutationVector{Attaches: []state.TerminalRef{term2}}
	case "terminal_pool.attach_float":
		return terminalMutationVector{Attaches: []state.TerminalRef{term2}, Resizes: resizeTerm1}
	case "panel.take_owner", "floating.take_owner", "resize.left", "resize.right", "resize.up", "resize.down", "resize.left_large", "resize.right_large", "resize.up_large", "resize.down_large":
		return terminalMutationVector{Resizes: resizeTerm1}
	case "panel.split_right", "panel.split_down", "panel.balance", "panel.presentation_card", "panel.presentation_split_line",
		"system.toggle_header", "system.toggle_footer", "floating.center", "floating.fit", "floating.auto_fit",
		"floating.move_left", "floating.move_right", "floating.move_up", "floating.move_down", "floating.narrow", "floating.wide", "floating.short", "floating.tall":
		return terminalMutationVector{Resizes: resizeTerm1}
	case "tab.jump":
		if index, ok := invocation.Param("index"); ok && index == 1 {
			return terminalMutationVector{Resizes: resizeTerm1}
		}
		return terminalMutationVector{}
	case "panel.size_lock":
		return terminalMutationVector{TagEdits: []state.TerminalRef{term1}}
	case "clipboard.paste_latest", "clipboard.paste_system", "clipboard_history.paste":
		return terminalMutationVector{Inputs: []state.TerminalRef{term1}}
	case "workbench_tree.delete", "workbench_tree.detach":
		return terminalMutationVector{Detaches: []state.TerminalRef{term2}}
	default:
		return terminalMutationVector{}
	}
}

func assertNoCoreServiceFallback(t *testing.T, core *testkit.FakeCoreClient, messages []string) {
	t.Helper()
	assertRef := func(operation string, endpoint state.EndpointID, terminalID string) {
		if ref := state.NewTerminalRef(endpoint, terminalID); ref.EndpointID != "west" || ref.TerminalID != "term-1" {
			t.Fatalf("%s issued history fallback request %#v: messages=%v", operation, ref, messages)
		}
	}
	for _, request := range core.LatestRequests {
		assertRef("history latest", request.EndpointID, request.TerminalID)
	}
	for _, request := range core.OlderRequests {
		assertRef("history older", request.EndpointID, request.TerminalID)
	}
	for _, request := range core.NewerRequests {
		assertRef("history newer", request.EndpointID, request.TerminalID)
	}
	for _, request := range core.OldestRequests {
		assertRef("history oldest", request.EndpointID, request.TerminalID)
	}
	for _, request := range core.CopyRequests {
		assertRef("history copy", request.EndpointID, request.TerminalID)
	}
	for _, request := range core.ReleaseRequests {
		assertRef("history release", request.EndpointID, request.TerminalID)
	}
}

func TestTerminalPickerSplitAttachFailureKeepsDeclaredPartialResult(t *testing.T) {
	invocation, _, err := actiondomain.ParseInvocation("terminal_picker.split")
	if err != nil {
		t.Fatal(err)
	}
	root := defaultActionExecutionRoot(t, invocation.ID)
	intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
	if !ok {
		t.Fatal("split action has no dispatcher")
	}
	next, effects := reduceShortcutIntentWithContext(root, intent, -1)
	newPaneID := next.Shell.EnsureDefaults().ActivePaneID
	var request TerminalPoolAttachRequestMsg
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok {
			if msg, ok := fn.Run(context.Background()).(TerminalPoolAttachRequestMsg); ok {
				request = msg
			}
		}
	}
	if request.EndpointID != "west" || request.TerminalID != "term-1" || request.TargetPaneID != newPaneID || newPaneID == state.DefaultPaneID {
		t.Fatalf("split transaction must declare its endpoint-aware target before attach: pane=%q request=%#v", newPaneID, request)
	}

	terminal := &testkit.FakeTerminalService{AttachErr: errors.New("west attach denied")}
	poolReducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	next, effects = poolReducer(next, request)
	if len(effects) != 1 {
		t.Fatalf("attach request must reach terminal service, effects=%#v", effects)
	}
	result := effects[0].(FuncEffect).Run(context.Background())
	next, effects = poolReducer(next, result)
	if len(effects) != 0 || len(terminal.Attaches) != 1 || terminal.Attaches[0].EndpointID != "west" {
		t.Fatalf("failed attach must not retry or fallback endpoint, attaches=%#v effects=%#v", terminal.Attaches, effects)
	}
	if original, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || original.EndpointID != "west" || original.TerminalID != "term-1" {
		t.Fatalf("failed split attach must not disturb original terminal view, binding=%#v ok=%v", original, ok)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: newPaneID}); !ok || pane.Kind != state.PaneEmpty || pane.TerminalID != "" {
		t.Fatalf("declared partial result must keep the new slot empty and visible, pane=%#v ok=%v", pane, ok)
	}
	if _, ok := next.TerminalViews.PaneBinding(newPaneID); ok {
		t.Fatalf("failed attach must not fabricate a target binding: %#v", next.TerminalViews)
	}
}

func TestTerminalPoolCombinationAttachFailuresKeepDeclaredSlots(t *testing.T) {
	tests := []struct {
		name   string
		create func(state.Root) (state.Root, []Effect)
		check  func(*testing.T, state.Root, TerminalPoolAttachRequestMsg)
	}{
		{
			name: "tab",
			create: func(root state.Root) (state.Root, []Effect) {
				return reduceTerminalPoolAttachTab(root, "west", "term-1")
			},
			check: func(t *testing.T, root state.Root, request TerminalPoolAttachRequestMsg) {
				pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: request.TargetPaneID})
				if !ok || pane.Kind != state.PaneEmpty || pane.TerminalID != "" {
					t.Fatalf("failed tab attach must leave its declared pane empty: pane=%#v ok=%v", pane, ok)
				}
				if _, ok := root.TerminalViews.PaneBinding(request.TargetPaneID); ok {
					t.Fatalf("failed tab attach fabricated a terminal binding: %#v", root.TerminalViews)
				}
			},
		},
		{
			name: "floating",
			create: func(root state.Root) (state.Root, []Effect) {
				return reduceTerminalPoolAttachFloating(root, "west", "term-1")
			},
			check: func(t *testing.T, root state.Root, request TerminalPoolAttachRequestMsg) {
				floating, ok := root.Shell.FloatingByID(request.TargetFloatingID)
				if !ok || floating.Pane.Kind != state.PaneEmpty || floating.Pane.TerminalID != "" {
					t.Fatalf("failed floating attach must leave its declared slot empty: floating=%#v ok=%v", floating, ok)
				}
				if _, ok := root.TerminalViews.FloatingBinding(request.TargetFloatingID); ok {
					t.Fatalf("failed floating attach fabricated a terminal binding: %#v", root.TerminalViews)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := defaultActionExecutionRoot(t, "terminal_pool.attach_"+actiondomain.ID(tc.name))
			next, effects := tc.create(root)
			request, ok := terminalPoolAttachRequestFromEffects(effects)
			if !ok || request.EndpointID != "west" || request.TerminalID != "term-1" {
				t.Fatalf("combination must declare endpoint-aware attach target: request=%#v ok=%v", request, ok)
			}
			terminal := &testkit.FakeTerminalService{AttachErr: errors.New("west attach denied")}
			poolReducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
			next, effects = poolReducer(next, request)
			if len(effects) != 1 {
				t.Fatalf("attach request must reach terminal service: effects=%#v", effects)
			}
			result := effects[0].(FuncEffect).Run(context.Background())
			next, effects = poolReducer(next, result)
			if len(effects) != 0 || len(terminal.Attaches) != 1 || terminal.Attaches[0].EndpointID != "west" {
				t.Fatalf("failed attach must not retry or fallback: attaches=%#v effects=%#v", terminal.Attaches, effects)
			}
			if original, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !original.TerminalRef().Equal(state.NewTerminalRef("west", "term-1")) {
				t.Fatalf("failed combination attach disturbed original terminal: binding=%#v ok=%v", original, ok)
			}
			tc.check(t, next, request)
		})
	}
}

func terminalPoolAttachRequestFromEffects(effects []Effect) (TerminalPoolAttachRequestMsg, bool) {
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok && fn.Run != nil {
			if request, ok := fn.Run(context.Background()).(TerminalPoolAttachRequestMsg); ok {
				return request, true
			}
		}
	}
	return TerminalPoolAttachRequestMsg{}, false
}

func defaultActionExecutionRoot(t *testing.T, id actiondomain.ID) state.Root {
	t.Helper()
	shell := state.DefaultShell()
	for index := 2; index <= 9; index++ {
		var result state.WorkbenchCommandResult
		shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: fmt.Sprintf("tab %d", index)})
		if result.Status != state.WorkbenchCommandOK {
			t.Fatalf("create tab fixture %d: %#v", index, result)
		}
	}
	var result state.WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch default tab fixture: %#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create workspace fixture: %#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceSwitch, TargetID: state.DefaultWorkspaceID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch default workspace fixture: %#v", result)
	}
	splitDirection := state.SplitDirectionVertical
	if id == "resize.up" || id == "resize.down" || id == "resize.up_large" || id == "resize.down_large" {
		splitDirection = state.SplitDirectionHorizontal
	}
	shell, _ = shell.ApplyPaneCommand(state.PaneCommand{
		Action: state.PaneCommandSplit, Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID}, SplitDirection: splitDirection,
		NewPane: state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"},
	})
	shell = shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	shell = shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "fixture", Body: "dismiss"})

	root := state.Root{
		Shell: shell, Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Session: state.TerminalSessionStore{
			EndpointID: "west", TerminalID: "term-1", Channel: 7, Attached: true, Cols: 98, Rows: 26,
			DesiredCols: 98, DesiredRows: 26, ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID: "surface-1", ViewID: state.TerminalPaneViewID(state.DefaultPaneID), State: state.TerminalLiveAttached,
		},
	}
	root.Endpoints = root.Endpoints.Upsert(state.EndpointItem{
		ID: "west", Label: "West", Enabled: true, Status: state.EndpointStatusConnected,
		Routes: []state.EndpointRouteItem{{ID: "local", Kind: state.EndpointTransportLocal, Enabled: true}},
	})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 98, 26, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewEndpointPaneTerminalView("west", "pane-2", "term-2", 8, 98, 26, state.TerminalResizeRoleFollower, "surface-2", state.TerminalPaneViewID("pane-2"), true))
	root.TerminalPool, _ = root.TerminalPool.ApplyList(0, []state.TerminalPoolItem{
		{EndpointID: "west", TerminalID: "term-1", Title: "main", State: "running", Tags: map[string]string{"role": "shell"}},
		{EndpointID: "west", TerminalID: "term-2", Title: "logs", State: "exited", Tags: map[string]string{"role": "build"}},
	}, "")
	root.Clipboard = root.Clipboard.WithCopiedText("first clip").WithCopiedText("second clip")

	if strings.HasPrefix(string(id), "floating.") || strings.HasPrefix(string(id), "floating_overview.") {
		for index := 1; index <= 9; index++ {
			root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
				Action: state.FloatingCommandCreate, TargetID: fmt.Sprintf("floating-%d", index),
				Pane: state.PaneState{ID: fmt.Sprintf("floating-pane-%d", index), Title: "float", Kind: state.PaneEmpty},
			})
		}
		root.Shell = root.Shell.BindFloatingTerminal("floating-9", "term-1")
		root.TerminalViews = root.TerminalViews.BindFloating(state.NewEndpointFloatingTerminalView(
			"west", "floating-9", "floating-pane-9", "term-1", 9, 50, 12, state.TerminalResizeRoleFollower,
			"surface-floating-9", state.TerminalFloatingViewID("floating-9"), false,
		))
	}
	switch {
	case strings.HasPrefix(string(id), "terminal_picker."):
		root.Shell = root.Shell.OpenTerminalPicker()
		items := state.TerminalPickerItems(root)
		for index, item := range items {
			if item.EndpointID == "west" && item.TerminalID == "term-1" {
				root.Shell = root.Shell.SetTerminalPickerSelectedIndex(index, len(items))
				break
			}
		}
	case strings.HasPrefix(string(id), "terminal_pool."):
		root.Shell = root.Shell.OpenTerminalPool()
	case strings.HasPrefix(string(id), "connections."):
		root.Shell = root.Shell.OpenConnections()
	case strings.HasPrefix(string(id), "workbench_tree."):
		root.Shell = root.Shell.OpenWorkbenchTree()
		items := state.WorkbenchTreeItems(root)
		for index, item := range items {
			if item.Kind == state.WorkbenchTreeKindPane && item.PaneID == "pane-2" {
				root.Shell = root.Shell.SetWorkbenchTreeSelectedIndex(index, len(items))
				break
			}
		}
	case strings.HasPrefix(string(id), "clipboard_history."):
		root.Shell = root.Shell.OpenClipboardHistory()
	case strings.HasPrefix(string(id), "floating_overview."):
		root.Shell = root.Shell.OpenFloatingOverview()
	case id == "prompt.submit":
		root.Shell = root.Shell.OpenPrompt(state.PromptState{Purpose: "action.command", Value: "system.clear_toasts"})
	case strings.HasPrefix(string(id), "help."):
		root.Shell = root.Shell.OpenHelp("most-used").SetHelpSelection(10, len(input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation)))
	}

	if strings.HasPrefix(string(id), "copy.") || id == "menu.copy" {
		root.CopyMode = state.CopyModeStore{
			Active: true, Phase: state.CopyModeFrozenHistory, PaneID: state.DefaultPaneID, ViewID: state.TerminalPaneViewID(state.DefaultPaneID),
			EndpointID: "west", TerminalID: "term-1", ViewportTop: 0, ViewRows: 3, Cursor: state.CopyPosition{Row: 1, Col: 2},
			Mark:       &state.CopyPosition{Row: 0, Col: 0},
			Selection:  &state.CopySelection{Anchor: state.CopyPosition{Row: 0, Col: 0}, Focus: state.CopyPosition{Row: 1, Col: 2}},
			BoundToken: "copy-token", BoundCols: 98,
		}
		root.History = state.HistoryStore{
			PaneID: state.DefaultPaneID, ViewID: state.TerminalPaneViewID(state.DefaultPaneID), EndpointID: "west", TerminalID: "term-1", Token: "copy-token", Cols: 98,
			Rows:     []state.HistoryRow{{Text: "alpha", LineID: 10}, {Text: "bravo", LineID: 20}, {Text: "charlie", LineID: 30}},
			Cursor:   state.HistoryCursor{Valid: true, BeforeLineID: 10},
			Boundary: state.HistoryBoundary{FirstLineID: 10, LastLineID: 40},
			HasMore:  true,
		}
		switch id {
		case "copy.request_older", "copy.oldest":
			root.CopyMode.Cursor = state.CopyPosition{Row: 0, Col: 2}
		case "copy.request_newer":
			root.CopyMode.Cursor = state.CopyPosition{Row: 2, Col: 2}
		case "copy.search_next", "copy.search_previous":
			root.CopyMode.Query = "bravo"
		}
		if id == "menu.copy" {
			root.CopyMode = state.CopyModeStore{}
		}
	}
	return root
}

func TestTabWorkspaceFooterHintsMatchInputBindings(t *testing.T) {
	shell := state.DefaultShell()
	var result state.WorkbenchCommandResult
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create tab fixture: %#v", result)
	}
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create workspace fixture: %#v", result)
	}

	cases := []struct {
		name   string
		mode   state.InteractionMode
		input  input.InteractionMode
		expect map[string]string
	}{
		{
			name:  "tab",
			mode:  state.InteractionModeTab,
			input: input.InteractionModeTab,
			expect: map[string]string{
				render.ActionTabCreate.String(): "tab create",
				"tab.previous":                  "tab previous",
				"tab.next":                      "tab next",
				"tab.rename":                    "tab rename",
				render.ActionTabClose.String():  "tab close",
			},
		},
		{
			name:  "workspace",
			mode:  state.InteractionModeWorkspace,
			input: input.InteractionModeWorkspace,
			expect: map[string]string{
				"workspace.create":   "workspace create",
				"workspace.previous": "workspace previous",
				"workspace.next":     "workspace next",
				"workspace.rename":   "workspace rename",
				"workspace.delete":   "workspace delete confirm=accepted",
			},
		},
	}

	for _, tc := range cases {
		caseShell := shell.SetInteractionMode(tc.mode)
		if tc.mode == state.InteractionModeTab {
			var switchResult state.WorkbenchCommandResult
			caseShell, switchResult = caseShell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceSwitch, TargetID: state.DefaultWorkspaceID})
			if switchResult.Status != state.WorkbenchCommandOK {
				t.Fatalf("switch tab footer fixture: %#v", switchResult)
			}
			caseShell = caseShell.SetInteractionMode(tc.mode)
		}
		vm := render.NewRenderVMBuilder().Build(state.Root{Shell: caseShell})
		for _, token := range vm.Shell.Footer.ActionTokens {
			command, ok := tc.expect[token.ActionID]
			if !ok {
				continue
			}
			key := firstFooterShortcutKey(token.Key)
			intent := input.RouteWithMode(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: key}, false, tc.input)
			if intent.Kind == input.IntentShortcutAction {
				intent, ok = shortcutIntentForInvocation(intent.Invocation, intent.Event)
				if !ok {
					t.Fatalf("%s footer key %q action %q has no dispatcher", tc.name, token.Key, token.ActionID)
				}
			}
			if intent.Kind != input.IntentWorkbenchCommand || intent.Command != command {
				t.Fatalf("%s footer key %q action %q should route to %q, got %#v", tc.name, token.Key, token.ActionID, command, intent)
			}
			delete(tc.expect, token.ActionID)
		}
		if len(tc.expect) != 0 {
			t.Fatalf("%s footer missed expected actions %#v in %#v", tc.name, tc.expect, vm.Shell.Footer.ActionTokens)
		}
	}
}

func firstFooterShortcutKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return key
	}
	return strings.TrimSpace(parts[0])
}
