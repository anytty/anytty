package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestCopyModePageUpLatestAndOlderE2E(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "new", LineID: 20}},
		)}},
		OlderResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowPrepend,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "old", LineID: 10}},
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if len(core.LatestRequests) != 1 || core.LatestRequests[0].TerminalID != "term-1" || core.LatestRequests[0].Cols != 78 {
		t.Fatalf("expected latest request, got %#v", core.LatestRequests)
	}
	if runtime.State().History.Token != "tok-1" || runtime.State().CopyMode.BoundToken != "tok-1" {
		t.Fatalf("state did not accept latest response %#v", runtime.State())
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("expected older request, got %#v", core.OlderRequests)
	}
	olderReq := core.OlderRequests[0]
	if olderReq.Token != "tok-1" || olderReq.Generation != 7 || olderReq.Boundary.LastLineID != 20 {
		t.Fatalf("unexpected older request %#v", olderReq)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"old", "new"}) {
		t.Fatalf("unexpected history rows %v", got)
	}
	if runtime.State().CopyMode.ViewportTop != 0 {
		t.Fatalf("viewport should clamp to top when visible rows cover loaded history, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if len(last.Lines) == 0 || !frameContains(last, "old") || !frameContains(last, "new") {
		t.Fatalf("expected latest rendered copy frame to start with older row, got %#v", last.Lines)
	}
	if frameContains(last, "● old") || frameContains(last, "● new") {
		t.Fatalf("copy-history content should not inject engineering markers into history text, got %#v", last.Lines)
	}
}

func TestCopyModeContinuousOlderPrependsAndKeepsTailBoundary(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "line-953", LineID: 953}, {Text: "line-954", LineID: 954}},
		)}},
		OlderResponses: []port.HistoryResult{
			{Window: historyWindowForApp(
				state.HistoryWindowPrepend,
				"term-1",
				"tok-1",
				78,
				7,
				[]state.HistoryRow{{Text: "line-951", LineID: 951}, {Text: "line-952", LineID: 952}},
			)},
			{Window: historyWindowForApp(
				state.HistoryWindowPrepend,
				"term-1",
				"tok-1",
				78,
				7,
				[]state.HistoryRow{{Text: "line-949", LineID: 949}, {Text: "line-950", LineID: 950}},
			)},
		},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 953}
	core.LatestResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 953, LastLineID: 954}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 951}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 951, LastLineID: 954}
	core.OlderResponses[1].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 949}
	core.OlderResponses[1].Window.Boundary = state.HistoryBoundary{FirstLineID: 949, LastLineID: 954}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 10)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("first older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first older: %v", err)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"line-951", "line-952", "line-953", "line-954"}) {
		t.Fatalf("first older should prepend rows, got %v", got)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("second older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain second older: %v", err)
	}
	if len(core.OlderRequests) != 2 {
		t.Fatalf("expected two older requests, got %#v", core.OlderRequests)
	}
	secondReq := core.OlderRequests[1]
	if secondReq.Cursor.BeforeLineID != 951 || secondReq.Boundary.FirstLineID != 951 || secondReq.Boundary.LastLineID != 954 {
		t.Fatalf("second older request should keep latest tail boundary and move cursor, got %#v", secondReq)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"line-949", "line-950", "line-951", "line-952", "line-953", "line-954"}) {
		t.Fatalf("second older should prepend older rows, got %v", got)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "line-949") || !frameContains(last, "line-950") {
		t.Fatalf("second older should visibly refresh to older rows, got %#v", last.Lines)
	}
}

func TestCopyModeGoAtLoadedTopRequestsOldest(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "line-964", LineID: 964}, {Text: "line-965", LineID: 965}},
		)}},
		OldestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "line-000", LineID: 1}, {Text: "line-001", LineID: 2}},
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 964}
	core.LatestResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 964, LastLineID: 965}
	core.OldestResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 1, LastLineID: 2}
	core.OldestResponses[0].Window.HasMore = true
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 10)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "g"}); err != nil {
		t.Fatalf("send g: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain g oldest: %v", err)
	}

	if len(core.OldestRequests) != 1 {
		t.Fatalf("g at loaded top should request authoritative oldest, got %#v", core.OldestRequests)
	}
	oldestReq := core.OldestRequests[0]
	if oldestReq.Token != "tok-1" || oldestReq.Generation != 7 || oldestReq.Boundary.LastLineID != 965 {
		t.Fatalf("unexpected oldest request %#v", oldestReq)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"line-000", "line-001"}) {
		t.Fatalf("g should replace local window with oldest page, got %v", got)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "line-000") || !frameContains(last, "line-001") {
		t.Fatalf("g oldest request should visibly refresh to oldest page, got %#v", last.Lines)
	}
}

func TestInteractiveRuntimeCtrlVEntersCopyModeWithoutBlockingOnSlowHistoryLatest(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	core := &blockingHistoryClient{}
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, &testkit.FakeTerminalService{}, NewAsyncEffectRunner())
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"live stays visible while copy history loads"},
		State:      state.TerminalLiveAttached,
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain live surface: %v", err)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}

	if runtime.State().CopyMode.Active || !runtime.State().CopyMode.Entering {
		t.Fatalf("expected copy mode entering before history activation, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.Kind != state.HistoryRequestLatest {
		t.Fatalf("expected pending latest request immediately, got %#v", runtime.State().History)
	}
	if len(core.latestRequests()) != 1 {
		t.Fatalf("expected async latest request to start, got %#v", core.latestRequests())
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "live stays visible while copy history loads") || frameContains(frame, "window pending") {
		t.Fatalf("entering copy mode must keep live frame until authoritative history arrives, got %#v", frame.Lines)
	}

	core.finishLatest(port.HistoryResult{
		Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			1,
			[]state.HistoryRow{{Text: "loaded", LineID: 10}},
		),
	}, nil)
	deadline := time.Now().Add(200 * time.Millisecond)
	for len(runtime.State().History.Rows) == 0 && time.Now().Before(deadline) {
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain async latest result: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"loaded"}) {
		t.Fatalf("expected latest history after async completion, got %v", got)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.Entering {
		t.Fatalf("latest result should activate copy mode, got %#v", runtime.State().CopyMode)
	}
	if frame := lastFrame(t, host.Frames()); !frameContains(frame, "loaded") {
		t.Fatalf("latest result should render authoritative copy history, got %#v", frame.Lines)
	}
}

func TestCopyModeEnteringSwallowsInputAndEscCancelsPendingLatest(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	core := &blockingHistoryClient{}
	terminal := &testkit.FakeTerminalService{}
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, terminal, NewAsyncEffectRunner())

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}
	if !runtime.State().CopyMode.Entering || runtime.State().History.Pending == nil {
		t.Fatalf("expected entering pending copy mode, got copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"}); err != nil {
		t.Fatalf("send x: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain x: %v", err)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("entering copy mode must swallow terminal input, got %#v", terminal.Inputs)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if runtime.State().CopyMode.Active || runtime.State().CopyMode.Entering || runtime.State().History.Pending != nil {
		t.Fatalf("esc should cancel entering copy mode and pending latest, copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
}

func TestCopyModeExitReleasesFrozenHistoryToken(t *testing.T) {
	host := NewFakeTerminalHost(16)
	core := &testkit.FakeCoreClient{}
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		Rows:       []state.HistoryRow{{Text: "alpha", LineID: 10}},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}

	if runtime.State().CopyMode.Active || runtime.State().History.Token != "" {
		t.Fatalf("copy exit should clear local history state, copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if len(core.ReleaseRequests) != 1 || core.ReleaseRequests[0].TerminalID != "term-1" || core.ReleaseRequests[0].Token != "tok-1" {
		t.Fatalf("copy exit should release frozen history token, got %#v", core.ReleaseRequests)
	}
}

func TestCopyModeEnteringDoesNotBindLiveSurfaceWhileLatestIsPending(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	core := &blockingHistoryClient{}
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, &testkit.FakeTerminalService{}, NewAsyncEffectRunner())
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"live-before-copy"},
		State:      state.TerminalLiveAttached,
	}}); err != nil {
		t.Fatalf("post initial live: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial live: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}
	if !runtime.State().CopyMode.Entering || runtime.State().CopyMode.Active {
		t.Fatalf("expected copy entering pending without active history, got %#v", runtime.State().CopyMode)
	}
	requests := core.latestRequests()
	if len(requests) != 1 || requests[0].GenerationBoundary != 0 {
		t.Fatalf("copy latest should not cap frozen history by live display revision, got %#v", requests)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"live-after-copy-should-not-move"},
		State:      state.TerminalLiveAttached,
	}}); err != nil {
		t.Fatalf("post later live: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain later live: %v", err)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "live-after-copy-should-not-move") || frameContains(last, "window pending") {
		t.Fatalf("entering copy should leave live render independent from history pending state, got %#v", last.Lines)
	}
}

func TestCopyModeEnteringWheelScrollAppliesWhenLatestArrives(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	core := &blockingHistoryClient{}
	terminal := &testkit.FakeTerminalService{}
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, terminal, NewAsyncEffectRunner())

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}
	content := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, state.DefaultPaneID)
	wheelUp := mouseEventAt(content.Rect)
	wheelUp.Mouse = input.MouseWheelUp
	wheelUp.RawSeq = "\x1b[<64;10;5M"
	wheelDown := mouseEventAt(content.Rect)
	wheelDown.Mouse = input.MouseWheelDown
	wheelDown.RawSeq = "\x1b[<65;10;5M"
	for _, event := range []input.InputEvent{wheelUp, wheelUp, wheelDown} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send entering wheel: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain entering wheel: %v", err)
		}
	}
	if got := runtime.State().CopyMode.EnteringScrollDelta; got != -1 {
		t.Fatalf("entering should keep net wheel intent, got %d in %#v", got, runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("entering wheel input must not leak to terminal, got %#v", terminal.Inputs)
	}

	core.finishLatest(port.HistoryResult{
		Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			historyRowsForEnteringScrollTest("latest", 10, 100),
		),
	}, nil)
	deadline := time.Now().Add(200 * time.Millisecond)
	for !runtime.State().CopyMode.Active && time.Now().Before(deadline) {
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain async latest result: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if runtime.State().CopyMode.ViewportTop != 0 || runtime.State().CopyMode.Cursor.Row != 8 {
		t.Fatalf("latest should apply entering wheel as cursor movement before viewport scroll, got copy=%#v", runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.EnteringScrollDelta != 0 || runtime.State().CopyMode.Entering {
		t.Fatalf("latest should clear entering scroll state, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeEnteringPageUpBeyondLatestContinuesOlder(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	core := &blockingHistoryClient{allowOlder: true}
	terminal := &testkit.FakeTerminalService{}
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, terminal, NewAsyncEffectRunner())

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("send ctrl-v: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ctrl-v: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send entering page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain entering page up: %v", err)
	}
	if got := runtime.State().CopyMode.EnteringScrollDelta; got != -18 {
		t.Fatalf("entering page up should keep pending page delta, got %d", got)
	}

	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		historyRowsForEnteringScrollTest("latest", 3, 100),
	)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 100}
	core.finishLatest(port.HistoryResult{Window: latest}, nil)
	deadline := time.Now().Add(200 * time.Millisecond)
	for len(core.olderRequests()) == 0 && time.Now().Before(deadline) {
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain async latest result: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if len(core.olderRequests()) != 1 {
		t.Fatalf("entering page up beyond latest should request older, got %#v", core.olderRequests())
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.Kind != state.HistoryRequestOlder || runtime.State().History.Pending.DeferredScrollRows != 16 {
		t.Fatalf("older request should carry unconsumed entering scroll, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().CopyMode.ViewportTop != 0 || runtime.State().CopyMode.Cursor.Row < runtime.State().CopyMode.ViewportTop || runtime.State().CopyMode.Cursor.Row >= runtime.State().CopyMode.ViewportTop+runtime.State().CopyMode.ViewRows || runtime.State().CopyMode.Entering {
		t.Fatalf("latest should consume local rows before older, got copy=%#v", runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("entering page up must not leak to terminal, got %#v", terminal.Inputs)
	}

	older := historyWindowForApp(
		state.HistoryWindowPrepend,
		"term-1",
		"tok-1",
		78,
		7,
		historyRowsForEnteringScrollTest("older", 20, 80),
	)
	older.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 80}
	older.Boundary = state.HistoryBoundary{FirstLineID: 80, LastLineID: 102}
	core.finishOlder(port.HistoryResult{Window: older}, nil)
	deadline = time.Now().Add(200 * time.Millisecond)
	for runtime.State().History.Pending != nil && time.Now().Before(deadline) {
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain async older result: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("older result should clear pending, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().CopyMode.ViewportTop != 3 || runtime.State().CopyMode.Cursor.Row < runtime.State().CopyMode.ViewportTop || runtime.State().CopyMode.Cursor.Row >= runtime.State().CopyMode.ViewportTop+runtime.State().CopyMode.ViewRows {
		t.Fatalf("older result should continue pending entering scroll, got copy=%#v", runtime.State().CopyMode)
	}
	visibleEnd := runtime.State().CopyMode.ViewportTop + runtime.State().CopyMode.ViewRows
	if visibleEnd > len(runtime.State().History.Rows) {
		visibleEnd = len(runtime.State().History.Rows)
	}
	visible := historyRowTexts(runtime.State().History.Rows[runtime.State().CopyMode.ViewportTop:visibleEnd])
	hasOlder := false
	for _, text := range visible {
		if strings.HasPrefix(text, "older-") {
			hasOlder = true
			break
		}
	}
	if !hasOlder {
		t.Fatalf("older result should move viewport into prepended history, top=%d visible=%v", runtime.State().CopyMode.ViewportTop, visible)
	}
}

func TestCopyModeDuplicateLatestWhilePendingDoesNotSurfaceError(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send first page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first latest: %v", err)
	}
	if runtime.State().History.Pending == nil {
		t.Fatalf("first latest should leave pending request in state, got %#v", runtime.State().History)
	}
	if historyRequestEffectCount(runner.Effects) != 1 {
		t.Fatalf("first copy mode entry should schedule one frozen latest effect, got %#v", runner.Effects)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send second page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain duplicate latest: %v", err)
	}
	if historyRequestEffectCount(runner.Effects) != 1 {
		t.Fatalf("duplicate latest while pending must not schedule another effect, got %#v", runner.Effects)
	}
	if runtime.State().Session.LastError != "" || runtime.State().Surface.Err != "" {
		t.Fatalf("duplicate latest while pending must not surface error, got %#v", runtime.State())
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "live") || frameContains(last, "window pending") {
		t.Fatalf("duplicate pending latest should keep live frame without fake error, got %#v", last.Lines)
	}
}

func TestCopyModeMouseWheelRequestsOlderAfterLatest(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "new-1", LineID: 19}, {Text: "new-2", LineID: 20}},
		)}},
		OlderResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowPrepend,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "old", LineID: 10}},
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("send wheel latest: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("send wheel older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}
	if len(core.LatestRequests) != 1 || len(core.OlderRequests) != 1 {
		t.Fatalf("unexpected history requests latest=%#v older=%#v", core.LatestRequests, core.OlderRequests)
	}
}

func TestCopyModeMouseWheelRawSeqEntersCopyMode(t *testing.T) {
	latestRows := make([]state.HistoryRow, 0, 30)
	for i := 1; i <= 30; i++ {
		latestRows = append(latestRows, state.HistoryRow{
			Text:     fmt.Sprintf("live-tail-%02d", i),
			LineID:   uint64(i),
			LiveTail: true,
		})
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			latestRows,
		)}},
	}
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core},
	)
	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfigForPane("term-1", 80, 24, "pane-2")}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	content := frameHitRegion(t, frame, render.HitRegionPaneContent, state.DefaultPaneID)
	wheel := mouseEventAt(content.Rect)
	wheel.Mouse = input.MouseWheelUp
	wheel.RawSeq = "\x1b[<64;10;5M"

	if err := host.SendInput(wheel); err != nil {
		t.Fatalf("send raw wheel latest: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain raw wheel latest: %v", err)
	}
	if len(core.LatestRequests) != 1 || !runtime.State().CopyMode.Active {
		t.Fatalf("raw wheel up should enter copy mode, latest=%#v copy=%#v", core.LatestRequests, runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Cursor.Row != len(latestRows)-2 {
		t.Fatalf("initial wheel up should move from newest tail by one row, got copy=%#v", runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.ViewportTop == 0 {
		t.Fatalf("initial live-tail wheel should not anchor cursor at current frame top, got copy=%#v", runtime.State().CopyMode)
	}
}

func TestCopyModeMouseWheelTargetsHitPaneWithoutWaitingForFocus(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			30,
			7,
			[]state.HistoryRow{{Text: "hit-pane-copy", LineID: 20}},
		)}},
	}
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Channel:    4,
			InputChannels: map[string]uint16{
				"term-1": 4,
			},
			Attached: true,
			Cols:     80,
			Rows:     24,
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"live"},
			State:      state.TerminalLiveAttached,
		},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 30, 20, state.TerminalResizeRoleOwner, "surface-left", state.TerminalPaneViewID(state.DefaultPaneID), true)).
			BindPane(state.NewPaneTerminalView("pane-2", "term-1", 4, 30, 20, state.TerminalResizeRoleFollower, "surface-right", state.TerminalPaneViewID("pane-2"), false)),
	}
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: core})
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	content := frameHitRegion(t, frame, render.HitRegionPaneContent, "pane-2")
	wheel := mouseEventAt(content.Rect)
	wheel.Mouse = input.MouseWheelUp
	wheel.RawSeq = "\x1b[<64;10;5M"

	if err := host.SendInput(wheel); err != nil {
		t.Fatalf("send pane-2 wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-2 wheel: %v", err)
	}
	if len(core.LatestRequests) != 1 || core.LatestRequests[0].ViewID != state.TerminalPaneViewID("pane-2") {
		t.Fatalf("wheel should request latest for hit pane view, requests=%#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.PaneID != "pane-2" || !runtime.State().CopyMode.Active {
		t.Fatalf("wheel should enter copy mode for hit pane, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeFloatingScrollDoesNotMutateTiledCopySession(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := state.TerminalFloatingViewID("floating-1")
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true)),
	}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	paneHistory := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		Token:       "tok-pane",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "pane-old", LineID: 1}, {Text: "pane-current", LineID: 2}}),
		Rows:        []state.HistoryRow{{Text: "pane-old", LineID: 1}, {Text: "pane-current", LineID: 2}},
	}
	paneCopy := state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		BoundToken:  "tok-pane",
		BoundCols:   78,
		ViewRows:    10,
		Cursor:      state.CopyPosition{Row: 1},
		ViewportTop: 1,
	}
	floatHistory := state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      floatingView,
		TerminalID:  "term-float",
		Token:       "tok-float",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-old", LineID: 11}, {Text: "float-current", LineID: 12}}),
		Rows:        []state.HistoryRow{{Text: "float-old", LineID: 11}, {Text: "float-current", LineID: 12}},
	}
	floatCopy := state.CopyModeStore{
		Active:      true,
		PaneID:      "float-pane",
		ViewID:      floatingView,
		TerminalID:  "term-float",
		BoundToken:  "tok-float",
		BoundCols:   40,
		ViewRows:    8,
		Cursor:      state.CopyPosition{Row: 1},
		ViewportTop: 1,
	}
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)
	root = root.WithCopyHistorySession(floatingView, floatHistory, floatCopy)
	root.History, root.CopyMode = root.CopyHistorySessionForView(floatingView)

	reducer := NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 10})
	next, _ := reducer(root, CopyModeScrollMsg{Delta: -1})

	_, nextPaneCopy := next.CopyHistorySessionForView(paneView)
	_, nextFloatCopy := next.CopyHistorySessionForView(floatingView)
	if nextPaneCopy.Cursor.Row != 1 || nextPaneCopy.ViewportTop != 1 {
		t.Fatalf("floating scroll must not mutate tiled copy session, got %#v", nextPaneCopy)
	}
	if nextFloatCopy.Cursor.Row != 0 {
		t.Fatalf("floating scroll should mutate only floating copy session, got %#v", nextFloatCopy)
	}
}

func TestCopyModeRuntimeFloatingWheelScrollUsesHitView(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := state.TerminalFloatingViewID("floating-1")
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 6, Y: 3, W: 44, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{
		Shell: shell,
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-pane",
			State:      state.TerminalLiveAttached,
			Lines:      []string{"tiled live tail"},
		}),
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleFollower, "surface-float", floatingView, true)),
	}
	paneHistory := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		Token:       "tok-pane",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "pane-old", LineID: 1}, {Text: "pane-current", LineID: 2}}),
		Rows:        []state.HistoryRow{{Text: "pane-old", LineID: 1}, {Text: "pane-current", LineID: 2}},
	}
	paneCopy := state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		BoundToken:  "tok-pane",
		BoundCols:   78,
		ViewRows:    8,
		Cursor:      state.CopyPosition{Row: 1},
		ViewportTop: 1,
	}
	floatHistory := state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      floatingView,
		TerminalID:  "term-float",
		Token:       "tok-float",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-old", LineID: 11}, {Text: "float-current", LineID: 12}}),
		Rows:        []state.HistoryRow{{Text: "float-old", LineID: 11}, {Text: "float-current", LineID: 12}},
	}
	floatCopy := state.CopyModeStore{
		Active:      true,
		PaneID:      "float-pane",
		ViewID:      floatingView,
		TerminalID:  "term-float",
		BoundToken:  "tok-float",
		BoundCols:   40,
		ViewRows:    8,
		Cursor:      state.CopyPosition{Row: 1},
		ViewportTop: 1,
	}
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)
	root = root.WithCopyHistorySession(floatingView, floatHistory, floatCopy)
	root.History, root.CopyMode = root.CopyHistorySessionForView(paneView)

	host := NewFakeTerminalHost(16)
	host.SetSize(120, 32)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 10},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	region := copyHistoryRowHitRegionForPane(t, lastFrame(t, host.Frames()), "float-pane", 1)
	if err := runtime.Post(InputMsg{Event: input.InputEvent{
		Kind:  input.EventKindMouse,
		Mouse: input.MouseWheelUp,
		Row:   region.Rect.Y + 1,
		Col:   region.Rect.X + 1,
	}}); err != nil {
		t.Fatalf("wheel floating history: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain wheel: %v", err)
	}

	_, nextPaneCopy := runtime.State().CopyHistorySessionForView(paneView)
	_, nextFloatCopy := runtime.State().CopyHistorySessionForView(floatingView)
	if nextPaneCopy.Cursor.Row != 1 || nextPaneCopy.ViewportTop != 1 {
		t.Fatalf("floating wheel must not mutate tiled copy session, got %#v", nextPaneCopy)
	}
	if nextFloatCopy.Cursor.Row != 0 || nextFloatCopy.ViewportTop != 0 {
		t.Fatalf("floating wheel should mutate only floating copy session, got %#v", nextFloatCopy)
	}
}

func TestCopyModeRuntimeWheelHitRegionsKeepFloatingAndTiledViewsIsolated(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := state.TerminalFloatingViewID("floating-1")
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 48, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{
		Shell: shell,
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 46, 10, state.TerminalResizeRoleFollower, "surface-float", floatingView, true)),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 28},
	}
	paneHistory, paneCopy := copyHistorySessionForWheelIsolation(state.DefaultPaneID, paneView, "term-pane", "tok-pane", 78, "pane", 20, 6)
	floatHistory, floatCopy := copyHistorySessionForWheelIsolation("float-pane", floatingView, "term-float", "tok-float", 46, "float", 20, 6)
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)
	root = root.WithCopyHistorySession(floatingView, floatHistory, floatCopy)
	root.History, root.CopyMode = root.CopyHistorySessionForView(floatingView)

	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(100, 28)
	runtime := NewAppRuntime(
		root,
		ComposeReducers(
			NewBackNavigationReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 10}),
			NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 10}),
		),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	floatingRegion := copyHistoryRowHitRegionForPane(t, copyHistoryPerfFrame, "float-pane", floatCopy.Cursor.Row)
	if !floatingRegion.Floating {
		t.Fatalf("initial floating history hit region must carry floating flag, got %#v", floatingRegion)
	}
	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp, Row: floatingRegion.Rect.Y + 1, Col: floatingRegion.Rect.X + 1}}); err != nil {
		t.Fatalf("first floating wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first floating wheel: %v", err)
	}
	_, paneAfterFirst := runtime.State().CopyHistorySessionForView(paneView)
	_, floatAfterFirst := runtime.State().CopyHistorySessionForView(floatingView)
	if paneAfterFirst.Cursor.Row != paneCopy.Cursor.Row || floatAfterFirst.Cursor.Row != floatCopy.Cursor.Row-1 {
		t.Fatalf("first floating wheel should only move floating copy, pane=%#v float=%#v", paneAfterFirst, floatAfterFirst)
	}

	floatRaise := frameActionHitRegion(t, copyHistoryPerfFrame, render.ActionFloatingRaise.String(), "float-pane")
	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp, Row: floatRaise.Rect.Y + floatRaise.Rect.H, Col: floatRaise.Rect.X + floatRaise.Rect.W}}); err != nil {
		t.Fatalf("floating blank wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating blank wheel: %v", err)
	}
	_, paneAfterBlank := runtime.State().CopyHistorySessionForView(paneView)
	_, floatAfterBlank := runtime.State().CopyHistorySessionForView(floatingView)
	if paneAfterBlank.Cursor.Row != paneCopy.Cursor.Row || floatAfterBlank.Cursor.Row != floatCopy.Cursor.Row-2 {
		t.Fatalf("floating blank wheel should still target floating view only, pane=%#v float=%#v", paneAfterBlank, floatAfterBlank)
	}

	paneCopy = paneAfterBlank
	floatCopy = floatAfterBlank
	tiledRegion := copyHistoryRowHitRegionForPane(t, render.Frame{HitRegions: runtime.lastHitRegions}, state.DefaultPaneID, paneCopy.Cursor.Row)
	if tiledRegion.Floating {
		t.Fatalf("tiled history hit region must not be marked floating, got %#v", tiledRegion)
	}
	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp, Row: tiledRegion.Rect.Y + 1, Col: tiledRegion.Rect.X + 1}}); err != nil {
		t.Fatalf("tiled wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tiled wheel: %v", err)
	}
	_, paneAfterTiled := runtime.State().CopyHistorySessionForView(paneView)
	_, floatAfterTiled := runtime.State().CopyHistorySessionForView(floatingView)
	if paneAfterTiled.Cursor.Row != paneCopy.Cursor.Row-1 || floatAfterTiled.Cursor.Row != floatCopy.Cursor.Row {
		t.Fatalf("tiled wheel should not leak to floating copy, pane=%#v float=%#v", paneAfterTiled, floatAfterTiled)
	}
}

func TestCopyModeEnteringSecondViewKeepsFirstUntilEsc(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := state.TerminalFloatingViewID("floating-1")
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{Surfaces: map[string]state.LiveSurfaceSnapshot{
			"term-float": {TerminalID: "term-float", Cols: 40, Rows: 10, Revision: 7},
		}},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true)),
	}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	paneHistory := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		Token:       "tok-pane",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "pane-history", LineID: 1}}),
		Rows:        []state.HistoryRow{{Text: "pane-history", LineID: 1}},
	}
	paneCopy := state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     paneView,
		TerminalID: "term-pane",
		BoundToken: "tok-pane",
		BoundCols:  78,
		ViewRows:   10,
	}
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)

	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
		state.HistoryWindowReplace,
		"term-float",
		"tok-float",
		40,
		9,
		[]state.HistoryRow{{Text: "float-history", LineID: 10}},
	)}}}
	deps := CopyModeDeps{Core: core, Rows: 10}
	reducer := ComposeReducers(NewBackNavigationReducer(deps), NewCopyModeReducer(deps))
	next, effects := reducer(root, CopyModeEnterViewMsg{
		Binding: state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true),
		Cols:    40,
		Rows:    10,
	})
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		msg := fn.Run(context.Background())
		next, _ = reducer(next, msg)
	}

	_, keptPaneCopy := next.CopyHistorySessionForView(paneView)
	_, enteredFloatCopy := next.CopyHistorySessionForView(floatingView)
	if !keptPaneCopy.Active || keptPaneCopy.BoundToken != "tok-pane" {
		t.Fatalf("entering floating copy must not exit existing pane copy session, got %#v", keptPaneCopy)
	}
	if !enteredFloatCopy.Active || enteredFloatCopy.BoundToken != "tok-float" {
		t.Fatalf("floating copy should become active after latest result, got %#v", enteredFloatCopy)
	}

	next.Shell, _ = next.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "floating-1"})
	next, _ = reducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	_, afterEscPaneCopy := next.CopyHistorySessionForView(paneView)
	_, afterEscFloatCopy := next.CopyHistorySessionForView(floatingView)
	if !afterEscPaneCopy.Active {
		t.Fatalf("ESC in floating must not exit pane copy session, got %#v", afterEscPaneCopy)
	}
	if afterEscFloatCopy.Active || afterEscFloatCopy.BoundToken != "" {
		t.Fatalf("ESC in floating should exit only floating copy session, got %#v", afterEscFloatCopy)
	}
}

func TestCopyModeEnteringFloatingWithBoundViewIDKeepsPaneUntilEsc(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := "anytty-floating-view-main"
	root := state.Root{
		Shell: state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{Surfaces: map[string]state.LiveSurfaceSnapshot{
			"term-float": {TerminalID: "term-float", Cols: 40, Rows: 10, Revision: 7},
		}},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true)),
	}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	paneHistory := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		Token:       "tok-pane",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "pane-history", LineID: 1}}),
		Rows:        []state.HistoryRow{{Text: "pane-history", LineID: 1}},
	}
	paneCopy := state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     paneView,
		TerminalID: "term-pane",
		BoundToken: "tok-pane",
		BoundCols:  78,
		ViewRows:   10,
	}
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)

	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
		state.HistoryWindowReplace,
		"term-float",
		"tok-float",
		40,
		9,
		[]state.HistoryRow{{Text: "float-history", LineID: 10}},
	)}}}
	deps := CopyModeDeps{Core: core, Rows: 10}
	reducer := ComposeReducers(NewBackNavigationReducer(deps), NewCopyModeReducer(deps))
	next, effects := reducer(root, CopyModeEnterViewMsg{
		Binding: state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true),
		Cols:    40,
		Rows:    10,
	})
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		msg := fn.Run(context.Background())
		next, _ = reducer(next, msg)
	}

	_, keptPaneCopy := next.CopyHistorySessionForView(paneView)
	_, enteredFloatCopy := next.CopyHistorySessionForView(floatingView)
	_, derivedFloatCopy := next.CopyHistorySessionForView(state.TerminalFloatingViewID("floating-1"))
	if !keptPaneCopy.Active || keptPaneCopy.BoundToken != "tok-pane" {
		t.Fatalf("entering bound floating copy must not exit existing pane copy session, got %#v", keptPaneCopy)
	}
	if !enteredFloatCopy.Active || enteredFloatCopy.BoundToken != "tok-float" {
		t.Fatalf("bound floating copy should become active after latest result, got %#v", enteredFloatCopy)
	}
	if derivedFloatCopy.Active {
		t.Fatalf("floating copy session must be keyed by real TerminalView ID, got derived session %#v", derivedFloatCopy)
	}

	next.Shell, _ = next.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "floating-1"})
	next, _ = reducer(next, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}})
	_, afterEscPaneCopy := next.CopyHistorySessionForView(paneView)
	_, afterEscFloatCopy := next.CopyHistorySessionForView(floatingView)
	if !afterEscPaneCopy.Active {
		t.Fatalf("ESC in bound floating must not exit pane copy session, got %#v", afterEscPaneCopy)
	}
	if afterEscFloatCopy.Active || afterEscFloatCopy.BoundToken != "" {
		t.Fatalf("ESC in bound floating should exit only floating copy session, got %#v", afterEscFloatCopy)
	}
}

func TestCopyModeCtrlVEnteringSecondViewDoesNotOverwriteFirstSession(t *testing.T) {
	paneView := state.TerminalPaneViewID(state.DefaultPaneID)
	floatingView := "anytty-floating-view-main"
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	paneHistory := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      paneView,
		TerminalID:  "term-pane",
		Token:       "tok-pane",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "pane-history", LineID: 1}}),
		Rows:        []state.HistoryRow{{Text: "pane-history", LineID: 1}},
	}
	paneCopy := state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     paneView,
		TerminalID: "term-pane",
		BoundToken: "tok-pane",
		BoundCols:  78,
		ViewRows:   10,
	}
	root := state.Root{
		Shell: shell,
		Surface: state.TerminalSurfaceStore{Surfaces: map[string]state.LiveSurfaceSnapshot{
			"term-float": {TerminalID: "term-float", Cols: 40, Rows: 10, Revision: 7},
		}},
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-pane", paneView, true)).
			BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 5, 40, 10, state.TerminalResizeRoleOwner, "surface-float", floatingView, true)),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 24},
	}
	root = root.WithCopyHistorySession(paneView, paneHistory, paneCopy)
	root.History, root.CopyMode = root.CopyHistorySessionForView(paneView)

	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
		state.HistoryWindowReplace,
		"term-float",
		"tok-float",
		40,
		9,
		[]state.HistoryRow{{Text: "float-history", LineID: 10}},
	)}}}
	reducer := NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 10})
	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}})
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		msg := fn.Run(context.Background())
		next, _ = reducer(next, msg)
	}

	_, keptPaneCopy := next.CopyHistorySessionForView(paneView)
	_, enteredFloatCopy := next.CopyHistorySessionForView(floatingView)
	if !keptPaneCopy.Active || keptPaneCopy.BoundToken != "tok-pane" || keptPaneCopy.ViewID != paneView {
		t.Fatalf("Ctrl-V in floating must keep existing pane copy session, got %#v", keptPaneCopy)
	}
	if !enteredFloatCopy.Active || enteredFloatCopy.BoundToken != "tok-float" || enteredFloatCopy.ViewID != floatingView {
		t.Fatalf("Ctrl-V in floating should enter floating copy session, got %#v", enteredFloatCopy)
	}
}

func TestCopyModeMouseWheelRawSeqScrollsDown(t *testing.T) {
	latestRows := make([]state.HistoryRow, 0, 40)
	for i := 1; i <= 40; i++ {
		latestRows = append(latestRows, state.HistoryRow{Text: fmt.Sprintf("raw-wheel-%02d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			latestRows,
		)}},
	}
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 12)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 12}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	content := frameHitRegion(t, frame, render.HitRegionPaneContent, state.DefaultPaneID)
	wheelUp := mouseEventAt(content.Rect)
	wheelUp.Mouse = input.MouseWheelUp
	wheelUp.RawSeq = "\x1b[<64;10;5M"
	if err := host.SendInput(wheelUp); err != nil {
		t.Fatalf("send raw wheel up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain raw wheel up: %v", err)
	}

	runtime.state.CopyMode.ViewportTop = 10
	runtime.state.CopyMode.Cursor = state.CopyPosition{Row: 10, Col: 2}
	frame = lastFrame(t, host.Frames())
	historyRow := frameHitRegion(t, frame, render.HitRegionHistoryRow, state.DefaultPaneID)
	wheelDown := mouseEventAt(historyRow.Rect)
	wheelDown.Mouse = input.MouseWheelDown
	wheelDown.RawSeq = "\x1b[<65;10;5M"
	if err := host.SendInput(wheelDown); err != nil {
		t.Fatalf("send raw wheel down: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain raw wheel down: %v", err)
	}
	if runtime.State().CopyMode.Cursor.Row != 11 || runtime.State().CopyMode.ViewportTop != 10 {
		t.Fatalf("raw wheel down should stay in copy/history reducer, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeLatestStartsAtNewestVisibleTail(t *testing.T) {
	latestRows := make([]state.HistoryRow, 0, 10)
	for i := 1; i <= 10; i++ {
		latestRows = append(latestRows, state.HistoryRow{Text: fmt.Sprintf("latest-%02d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			latestRows,
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 1}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 7)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if runtime.State().CopyMode.ViewportTop == 0 {
		t.Fatalf("latest copy mode should start at newest tail, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "latest-10") || frameContains(last, "latest-01") {
		t.Fatalf("latest copy mode should render newest tail, got %#v", last.Lines)
	}
}

func TestCopyModeEntryUsesTerminalSizeInsideLargerPane(t *testing.T) {
	binding := state.NewPaneTerminalView(state.DefaultPaneID, "term-small", 4, 40, 8, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(state.DefaultPaneID), false)
	window := historyWindowForApp(state.HistoryWindowReplace, "term-small", "tok-small", 40, 7, []state.HistoryRow{{Text: "old", LineID: 1}})
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: window}}}
	root := state.Root{
		TerminalViews: state.TerminalViewStore{}.BindPane(binding),
		Surface:       state.TerminalSurfaceStore{TerminalID: "term-small", Cols: 40, Rows: 8},
	}
	reducer := NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 24})

	next, effects := reducer(root, CopyModeEnterViewMsg{Binding: binding, Cols: 100, Rows: 30})
	for _, effect := range effects {
		if fn, ok := effect.(FuncEffect); ok {
			next, _ = reducer(next, fn.Run(context.Background()))
		}
	}

	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 40 {
		t.Fatalf("history latest must use terminal columns inside a larger pane, got %#v", core.LatestRequests)
	}
	if next.CopyMode.BoundCols != 40 || next.CopyMode.ViewRows != 8 {
		t.Fatalf("history scrolling must use terminal rows inside a larger pane, got %#v", next.CopyMode)
	}
}

func TestCopyModeWheelUpScrollsByLineAndPrefetchesOlderWithoutJump(t *testing.T) {
	olderRows := make([]state.HistoryRow, 0, 24)
	for i := 1; i <= 24; i++ {
		olderRows = append(olderRows, state.HistoryRow{Text: fmt.Sprintf("old-%02d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{
				{Text: "new-01", LineID: 101},
				{Text: "new-02", LineID: 102},
				{Text: "new-03", LineID: 103},
				{Text: "new-04", LineID: 104},
				{Text: "new-05", LineID: 105},
				{Text: "new-06", LineID: 106},
				{Text: "new-07", LineID: 107},
				{Text: "new-08", LineID: 108},
				{Text: "new-09", LineID: 109},
				{Text: "new-10", LineID: 110},
			},
		)}},
		OlderResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowPrepend,
			"term-1",
			"tok-1",
			78,
			7,
			olderRows,
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 101}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 1}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 1, LastLineID: 110}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 10)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	runtime.state.CopyMode.ViewportTop = 3
	runtime.state.CopyMode.Cursor = state.CopyPosition{Row: 3}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("wheel loaded rows: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain prefetch wheel: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("wheel near loaded top should prefetch older once, got %#v", core.OlderRequests)
	}
	if runtime.State().CopyMode.ViewportTop != len(olderRows)+2 {
		t.Fatalf("older prefetch should keep the one-line scrolled content anchored, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Cursor.Row != len(olderRows)+2 {
		t.Fatalf("older prefetch should keep cursor as scroll truth, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Rows[runtime.State().CopyMode.ViewportTop].Text != "new-03" {
		t.Fatalf("older prefetch must fill cache without jumping to older page, rows=%v top=%d", historyRowTexts(runtime.State().History.Rows), runtime.State().CopyMode.ViewportTop)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("wheel loaded prefetched rows: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain local wheel after prefetch: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("wheel over thick prefetched data should not request another older page, got %#v", core.OlderRequests)
	}
	if runtime.State().CopyMode.ViewportTop != len(olderRows)+1 || runtime.State().History.Rows[runtime.State().CopyMode.ViewportTop].Text != "new-02" {
		t.Fatalf("wheel should continue moving one row through local cache, rows=%v copy=%#v", historyRowTexts(runtime.State().History.Rows), runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Cursor.Row != len(olderRows)+1 {
		t.Fatalf("wheel should move cursor through local cache, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-01") && !frameContains(last, "new-02") {
		t.Fatalf("older prefetch should not steal viewport to prepended page, got %#v", last.Lines)
	}
}

func TestCopyModeMouseWheelMovesCursorAndKeepsItVisible(t *testing.T) {
	latestRows := make([]state.HistoryRow, 0, 40)
	for i := 1; i <= 40; i++ {
		latestRows = append(latestRows, state.HistoryRow{Text: fmt.Sprintf("row-%02d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 1},
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 1, LastLineID: 40},
		SourceLines: historyLogicalLinesForApp(latestRows),
		Rows:        latestRows,
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   78,
		ViewRows:    5,
		ViewportTop: 10,
		Cursor:      state.CopyPosition{Row: 14, Col: 2},
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("wheel inside viewport: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain wheel inside viewport: %v", err)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 13, Col: 2}) || runtime.State().CopyMode.ViewportTop != 10 {
		t.Fatalf("wheel should move cursor inside viewport before moving viewport, got %#v", runtime.State().CopyMode)
	}

	runtime.state.CopyMode.Cursor = state.CopyPosition{Row: 10, Col: 2}
	runtime.state.CopyMode.ViewportTop = 10
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("wheel across top edge: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain wheel across top edge: %v", err)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 9, Col: 2}) || runtime.State().CopyMode.ViewportTop != 9 {
		t.Fatalf("wheel should keep cursor visible when viewport crosses top edge, got %#v", runtime.State().CopyMode)
	}

	visibleRows := runtime.State().CopyMode.ViewRows
	if visibleRows <= 0 {
		t.Fatalf("runtime should keep a positive copy view height, got %#v", runtime.State().CopyMode)
	}
	bottomTop := 10
	bottomRow := bottomTop + visibleRows - 1
	runtime.state.CopyMode.Cursor = state.CopyPosition{Row: bottomRow, Col: 2}
	runtime.state.CopyMode.ViewportTop = 10
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelDown}); err != nil {
		t.Fatalf("wheel across bottom edge: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain wheel across bottom edge: %v", err)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: bottomRow + 1, Col: 2}) || runtime.State().CopyMode.ViewportTop != bottomTop+1 {
		t.Fatalf("wheel should move viewport only after cursor crosses bottom edge, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModePageDownMovesCursorBeforeRequestingNewer(t *testing.T) {
	rows := make([]state.HistoryRow, 0, 40)
	for i := 1; i <= 40; i++ {
		rows = append(rows, state.HistoryRow{Text: fmt.Sprintf("page-row-%02d", i), LineID: uint64(i)})
	}
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			Token:       "tok-1",
			Cols:        78,
			Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 1},
			Generation:  7,
			Boundary:    state.HistoryBoundary{FirstLineID: 1, LastLineID: 40},
			SourceLines: historyLogicalLinesForApp(rows),
			Rows:        rows,
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   78,
			ViewRows:    10,
			ViewportTop: 10,
			Cursor:      state.CopyPosition{Row: 12, Col: 2},
		},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 78, 10, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))

	next, effects := NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageDn}})

	if len(effects) == 0 {
		t.Fatalf("page down should be handled")
	}
	if next.CopyMode.Cursor.Row != 20 || next.CopyMode.ViewportTop != 11 {
		t.Fatalf("page down should move cursor first and only scroll viewport enough to keep it visible, got %#v", next.CopyMode)
	}
}

func TestCopyModeWheelAtTopRevealsOneOlderRow(t *testing.T) {
	olderRows := make([]state.HistoryRow, 0, 10)
	for i := 1; i <= 10; i++ {
		olderRows = append(olderRows, state.HistoryRow{Text: fmt.Sprintf("old-%02d", i), LineID: uint64(10 + i)})
	}
	latestRows := make([]state.HistoryRow, 0, 20)
	for i := 1; i <= 20; i++ {
		latestRows = append(latestRows, state.HistoryRow{Text: fmt.Sprintf("new-%02d", i), LineID: uint64(20 + i)})
	}
	core := &testkit.FakeCoreClient{
		OlderResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowPrepend,
			"term-1",
			"tok-1",
			78,
			7,
			olderRows,
		)}},
	}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 11}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 11, LastLineID: 40}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 21},
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 21, LastLineID: 40},
		SourceLines: historyLogicalLinesForApp(latestRows),
		Rows:        latestRows,
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   78,
		ViewRows:    3,
		ViewportTop: 0,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
		t.Fatalf("wheel older at top: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("wheel at top should request one older page, got %#v", core.OlderRequests)
	}
	got := historyRowTexts(runtime.State().History.Rows)
	if len(got) != 30 || got[0] != "old-01" || got[10] != "new-01" {
		t.Fatalf("older should fill local cache before latest rows, got %v", got)
	}
	if runtime.State().CopyMode.ViewportTop != 9 || runtime.State().History.Rows[runtime.State().CopyMode.ViewportTop].Text != "old-10" {
		t.Fatalf("wheel at top should reveal exactly one older row, rows=%v copy=%#v", historyRowTexts(runtime.State().History.Rows), runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Cursor.Row != 9 {
		t.Fatalf("wheel at top should move cursor to revealed older row, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeHistoryRequestsScaleWithPanelHeight(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			118,
			7,
			[]state.HistoryRow{{Text: "new", LineID: 20}},
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	host := NewFakeTerminalHost(8)
	host.SetSize(120, 40)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if len(core.LatestRequests) != 1 {
		t.Fatalf("expected latest request, got %#v", core.LatestRequests)
	}
	if core.LatestRequests[0].Rows <= 20 {
		t.Fatalf("history latest request should scale with current panel height, got %#v", core.LatestRequests[0])
	}
}

func TestCopyModeEntryUsesSingleFrozenLatestRequest(t *testing.T) {
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		8,
		[]state.HistoryRow{{Text: "frozen-latest", LineID: 20}},
	)
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 12)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if len(core.LatestRequests) != 1 {
		t.Fatalf("copy enter should request only frozen latest, latest=%#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.PhaseKind() != state.CopyModeFrozenHistory || runtime.State().CopyMode.BoundToken != "tok-1" {
		t.Fatalf("sync fake should end on frozen latest, got %#v", runtime.State().CopyMode)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"frozen-latest"}) {
		t.Fatalf("frozen latest should populate history rows, got %v", got)
	}
}

func TestCopyModeDuplicateOlderWhilePendingDoesNotSurfaceError(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)
	runtime.state.History = state.HistoryStore{
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 20},
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 20},
		Rows:       []state.HistoryRow{{Text: "new", LineID: 20}},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   20,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send first older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first older: %v", err)
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.Kind != state.HistoryRequestOlder {
		t.Fatalf("first older should leave older request pending, got %#v", runtime.State().History)
	}
	if len(runner.Effects) != 1 {
		t.Fatalf("first older should schedule exactly one effect, got %#v", runner.Effects)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send second older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain duplicate older: %v", err)
	}
	if len(runner.Effects) != 1 {
		t.Fatalf("duplicate older while pending must not schedule another effect, got %#v", runner.Effects)
	}
	if runtime.State().Session.LastError != "" || runtime.State().Surface.Err != "" {
		t.Fatalf("duplicate older while pending must not surface error, got %#v", runtime.State())
	}
}

func TestCopyModeOlderBoundaryTokensAndExhaustedGuard(t *testing.T) {
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "new", LineID: 20}},
	)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	latest.HasMore = true
	exhausted := historyWindowForApp(state.HistoryWindowPrepend, "term-1", "tok-1", 78, 7, nil)
	exhausted.Cursor = latest.Cursor
	exhausted.HasMore = false
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
		OlderResponses:  []port.HistoryResult{{Window: exhausted}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send latest page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if status := activeCopyContentStatus(runtime); !strings.Contains(status, "older:more") {
		t.Fatalf("latest window with cursor should expose older-more status, got %q", status)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send exhausted older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exhausted older: %v", err)
	}
	if status := activeCopyContentStatus(runtime); !strings.Contains(status, "older:top") {
		t.Fatalf("exhausted older window should expose top status, got %q", status)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send redundant older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain redundant older: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("exhausted boundary must not request older again, got %#v", core.OlderRequests)
	}
}

func TestCopyModeExhaustedGuardSurvivesLocalReflowResize(t *testing.T) {
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "old-window", LineID: 20}},
	)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	latest.HasMore = true
	exhausted := historyWindowForApp(state.HistoryWindowPrepend, "term-1", "tok-1", 78, 7, nil)
	exhausted.Cursor = latest.Cursor
	exhausted.HasMore = false
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
		OlderResponses:  []port.HistoryResult{{Window: exhausted}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send latest page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send exhausted older page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exhausted older: %v", err)
	}

	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain local reflow resize: %v", err)
	}
	if status := activeCopyContentStatus(runtime); !strings.Contains(status, "older:top") {
		t.Fatalf("local reflow must preserve exhausted top status, got %q", status)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send redundant older after local reflow: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain redundant older after local reflow: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("local reflow must not clear exhausted older guard, got %#v", core.OlderRequests)
	}
}

func TestCopyModeOlderResponseKeepsLocalReflowColsBinding(t *testing.T) {
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "old-window", LineID: 20}},
	)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	latest.HasMore = true
	older := historyWindowForApp(
		state.HistoryWindowPrepend,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "older-window", LineID: 10}},
	)
	older.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	older.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	older.HasMore = true
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
		OlderResponses:  []port.HistoryResult{{Window: older}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send latest page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send older page up: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize before older response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain local reflow with older response: %v", err)
	}

	if runtime.State().History.Cols != 78 || runtime.State().CopyMode.BoundCols != 78 {
		t.Fatalf("pane resize must keep history bound to terminal cols, got history=%#v copy=%#v", runtime.State().History, runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "history cols changed") {
		t.Fatalf("older response after local reflow must keep copy-history render bound, got %#v", last.Lines)
	}
	if !frameContains(last, "older-window") || !frameContains(last, "old-window") {
		t.Fatalf("older response should still render frozen prepended rows after local reflow, got %#v", last.Lines)
	}
}

func TestCopyModeAttachRebindInvalidatesFrozenHistoryForNewTerminal(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{
		Shell: state.DefaultShell(),
		Session: state.TerminalSessionStore{
			TerminalID: "term-old",
			Attached:   true,
			Cols:       80,
			Rows:       24,
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-old",
			Cols:       80,
			Rows:       24,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-old",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-new",
		Channel:      9,
		Cols:         78,
		Rows:         20,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "surface",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
		CanResize:    true,
	}})
	if len(effects) == 0 {
		t.Fatalf("attach rebind should keep live follow-up effects, got %#v", effects)
	}
	if next.History.Token != "" || next.History.TerminalID != "term-new" || len(next.History.Rows) != 0 {
		t.Fatalf("attach rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("attach rebind must clear old frozen copy binding for new terminal, got %#v", next.CopyMode)
	}
}

func TestCopyModeReattachSameTerminalKeepsFrozenHistory(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{
		Shell: state.DefaultShell(),
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Attached:   true,
			Cols:       80,
			Rows:       24,
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       80,
			Rows:       24,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      9,
		Cols:         78,
		Rows:         20,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
		CanResize:    false,
	}})
	if len(effects) == 0 {
		t.Fatalf("same-terminal reattach should keep live follow-up effects, got %#v", effects)
	}
	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal reattach must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 78 || next.CopyMode.Empty {
		t.Fatalf("same-terminal reattach must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal reattach must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeAttachRebindInvalidatesFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.Session = state.TerminalSessionStore{
		TerminalID: "term-old",
		Attached:   true,
		Cols:       120,
		Rows:       24,
	}
	root.Surface = state.TerminalSurfaceStore{
		TerminalID: "term-old",
		Cols:       120,
		Rows:       24,
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView(
		"floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true,
	))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-old",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-old",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-new",
		Channel:      8,
		Cols:         40,
		Rows:         12,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalFloatingViewID("floating-1"),
		CanResize:    true,
	}})
	if len(effects) == 0 {
		t.Fatalf("floating attach rebind should keep live follow-up effects, got %#v", effects)
	}
	if next.History.Token != "" || next.History.TerminalID != "term-new" || len(next.History.Rows) != 0 {
		t.Fatalf("floating attach rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("floating attach rebind must clear old frozen copy binding for new terminal, got %#v", next.CopyMode)
	}
}

func TestCopyModeReattachSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.Session = state.TerminalSessionStore{
		TerminalID: "term-1",
		Attached:   true,
		Cols:       120,
		Rows:       24,
	}
	root.Surface = state.TerminalSurfaceStore{
		TerminalID: "term-1",
		Cols:       120,
		Rows:       24,
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView(
		"floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true,
	))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      8,
		Cols:         40,
		Rows:         12,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalFloatingViewID("floating-1"),
		CanResize:    false,
	}})
	if len(effects) == 0 {
		t.Fatalf("same-terminal floating reattach should keep live follow-up effects, got %#v", effects)
	}
	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal floating reattach must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 40 || next.CopyMode.Empty {
		t.Fatalf("same-terminal floating reattach must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal floating reattach must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeRuntimeAttachRebindDoesNotRenderOldFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-new",
		Channel:      9,
		Cols:         78,
		Rows:         20,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "surface",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
		CanResize:    true,
	}}); err != nil {
		t.Fatalf("post attach rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("attach rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("attach rebind should fall back to pending history state for new terminal, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeReattachSameTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	if !frameContains(lastFrame(t, host.Frames()), "old-history") {
		t.Fatalf("expected initial frame to render old frozen history, frames=%#v", host.Frames())
	}

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      9,
		Cols:         78,
		Rows:         20,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
		CanResize:    false,
	}}); err != nil {
		t.Fatalf("post same-terminal reattach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal reattach: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal reattach must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.TerminalID != "term-1" {
		t.Fatalf("same-terminal reattach must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-history") {
		t.Fatalf("same-terminal reattach must keep rendering frozen history, got %#v", last.Lines)
	}
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal reattach must not fall back to pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeAttachRebindDoesNotRenderOldFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-new",
		Channel:      8,
		Cols:         40,
		Rows:         12,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalFloatingViewID("floating-1"),
		CanResize:    true,
	}}); err != nil {
		t.Fatalf("post floating attach rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating attach rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("floating attach rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.TerminalID != "term-new" || runtime.State().CopyMode.BoundToken != "" || runtime.State().History.TerminalID != "term-new" {
		t.Fatalf("floating attach rebind should leave copy mode in pending state for new terminal, state=%#v", runtime.State())
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("floating attach rebind should render pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeReattachSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	initialBoundCols := runtime.State().CopyMode.BoundCols

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      8,
		Cols:         40,
		Rows:         12,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "surface-new",
		ViewID:       state.TerminalFloatingViewID("floating-1"),
		CanResize:    false,
	}}); err != nil {
		t.Fatalf("post same-terminal floating reattach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal floating reattach: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal floating reattach must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.TerminalID != "term-1" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.BoundCols != initialBoundCols || runtime.State().CopyMode.Empty {
		t.Fatalf("same-terminal floating reattach must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("same-terminal floating reattach must not force pending latest, got %#v", runtime.State().History.Pending)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal floating reattach must not fall back to pending history, got %#v", last.Lines)
	}
}

func TestCopyModeRemoveClearsFrozenHistoryForDeletedTerminal(t *testing.T) {
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: &testkit.FakeTerminalService{}})
	root := state.Root{
		Shell: state.DefaultShell(),
		Session: state.TerminalSessionStore{
			TerminalID: "term-old",
			Attached:   true,
			Cols:       80,
			Rows:       24,
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-old",
			Cols:       80,
			Rows:       24,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-old",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, effects := reducer(root, TerminalPoolRemoveResultMsg{TerminalID: "term-old"})
	if len(effects) == 0 {
		t.Fatalf("remove result should keep list refresh effect, got %#v", effects)
	}
	if next.History.Token != "" || next.History.TerminalID != "" || len(next.History.Rows) != 0 {
		t.Fatalf("remove result must invalidate deleted terminal history window, got %#v", next.History)
	}
	if next.CopyMode.Active || next.CopyMode.TerminalID != "" || next.CopyMode.BoundToken != "" {
		t.Fatalf("remove result must clear copy mode bound to deleted terminal, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimeRemoveDoesNotRenderDeletedFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-old"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := runtime.Post(TerminalPoolRemoveResultMsg{TerminalID: "term-old"}); err != nil {
		t.Fatalf("post remove result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain remove result: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("remove result must not keep rendering deleted frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("remove result must exit copy mode bound to deleted terminal, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModePaneCloseInvalidatesFrozenHistoryForClosedPane(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell().SplitActivePane(
			state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-2"},
			state.SplitDirectionVertical,
		).FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		TerminalViews: state.TerminalViewStore{}.
			BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
			BindPane(state.NewPaneTerminalView("pane-2", "term-2", 5, 38, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID("pane-2"), true)),
		History: state.HistoryStore{
			PaneID:      "pane-2",
			ViewID:      state.TerminalPaneViewID("pane-2"),
			TerminalID:  "term-2",
			Token:       "tok-old",
			Cols:        38,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     "pane-2",
			ViewID:     state.TerminalPaneViewID("pane-2"),
			TerminalID: "term-2",
			BoundToken: "tok-old",
			BoundCols:  38,
			ViewRows:   20,
		},
	}

	next, effects := reducePaneCommand(root, state.PaneCommand{
		Action: state.PaneCommandClose,
		Target: state.PaneCommandTarget{PaneID: "pane-2"},
		Source: state.PaneCommandSourceTest,
	})
	if len(effects) == 0 {
		t.Fatalf("pane close should keep normal pane command effects, got %#v", effects)
	}
	if next.History.Token != "" || next.History.TerminalID != "" || len(next.History.Rows) != 0 || len(next.History.SourceLines) != 0 {
		t.Fatalf("pane close must release frozen history for closed pane, got %#v", next.History)
	}
	if next.CopyMode.Active || next.CopyMode.PaneID != "" || next.CopyMode.BoundToken != "" {
		t.Fatalf("pane close must clear copy mode bound to closed pane, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimePaneCloseDoesNotRenderClosedFrozenHistory(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell().SplitActivePane(
				state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-2"},
				state.SplitDirectionVertical,
			).FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
			TerminalViews: state.TerminalViewStore{}.
				BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
				BindPane(state.NewPaneTerminalView("pane-2", "term-2", 5, 38, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID("pane-2"), true)),
			History: state.HistoryStore{
				PaneID:      "pane-2",
				ViewID:      state.TerminalPaneViewID("pane-2"),
				TerminalID:  "term-2",
				Token:       "tok-old",
				Cols:        38,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "pane-2",
				ViewID:     state.TerminalPaneViewID("pane-2"),
				TerminalID: "term-2",
				BoundToken: "tok-old",
				BoundCols:  38,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action: state.PaneCommandClose,
		Target: state.PaneCommandTarget{PaneID: "pane-2"},
		Source: state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post pane close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane close: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("pane close must not keep rendering closed pane frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("pane close must exit copy mode bound to closed pane, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeTabSwitchKeepsFrozenHistoryUntilExplicitExit(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell, _ = root.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, TargetID: "tab-2", Name: "logs"})
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{TabID: "tab-2"}, "term-2")
	root.Shell, _ = root.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})
	root.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        80,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 10}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 10}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  80,
	}

	next, _ := reduceWorkbenchCommand(root, state.WorkbenchCommand{
		Action:   state.WorkbenchCommandTabSwitch,
		TargetID: "tab-2",
		Source:   state.PaneCommandSourceTest,
	})

	if next.History.Token != "tok-old" || len(next.History.Rows) != 1 {
		t.Fatalf("tab switch should keep inactive pane frozen history until ESC, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.BoundToken != "tok-old" {
		t.Fatalf("tab switch should keep copy mode bound to previous pane, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimeTabSwitchKeepsButDoesNotRenderPreviousPaneFrozenHistory(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        80,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 10}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 10}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  80,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, TargetID: "tab-2", Name: "logs"})
	runtime.state.Shell = runtime.state.Shell.BindPaneTerminal(state.PaneCommandTarget{TabID: "tab-2"}, "term-2")
	runtime.state.Shell, _ = runtime.state.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})

	if err := runtime.Post(ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{
		Action:   state.WorkbenchCommandTabSwitch,
		TargetID: "tab-2",
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post tab switch: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tab switch: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("tab switch must not render inactive pane frozen history in active tab, got %#v", last.Lines)
	}
	if !runtime.State().CopyMode.Active {
		t.Fatalf("tab switch should keep copy mode until explicit ESC, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeWorkspaceSwitchKeepsFrozenHistoryUntilExplicitExit(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	root.Shell, _ = root.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "ws-2"})
	secondWorkspaceID := root.Shell.Workspace.ID
	root.Shell = root.Shell.BindPaneTerminal(state.PaneCommandTarget{WorkspaceID: secondWorkspaceID}, "term-2")
	root.Shell, _ = root.Shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceSwitch, TargetID: state.DefaultWorkspaceID})
	root.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        80,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 10}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 10}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  80,
	}

	next, _ := reduceWorkbenchCommand(root, state.WorkbenchCommand{
		Action:   state.WorkbenchCommandWorkspaceSwitch,
		TargetID: secondWorkspaceID,
		Source:   state.PaneCommandSourceTest,
	})

	if next.History.Token != "tok-old" || len(next.History.Rows) != 1 {
		t.Fatalf("workspace switch should keep inactive pane frozen history until ESC, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.BoundToken != "tok-old" {
		t.Fatalf("workspace switch should keep copy mode bound to previous pane, got %#v", next.CopyMode)
	}
}

func TestCopyModeFloatingCloseInvalidatesFrozenHistoryForClosedFloating(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 7, 40, 12, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-float",
		Token:       "tok-float",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "float-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-float",
		BoundToken: "tok-float",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceFloatingCommand(root, state.FloatingCommand{
		Action:   state.FloatingCommandClose,
		TargetID: "floating-1",
		Source:   state.PaneCommandSourceTest,
	})

	if next.History.Token != "" || len(next.History.Rows) != 0 {
		t.Fatalf("floating close must invalidate frozen history for closed floating, got %#v", next.History)
	}
	if next.CopyMode.Active || next.CopyMode.ViewID != "" || next.CopyMode.BoundToken != "" {
		t.Fatalf("floating close must clear copy mode bound to closed floating, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimeFloatingCloseDoesNotRenderClosedFrozenHistory(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-float",
				Token:       "tok-float",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "float-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-float",
				BoundToken: "tok-float",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 7, 40, 12, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action:   state.FloatingCommandClose,
		TargetID: "floating-1",
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post floating close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating close: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "float-history") {
		t.Fatalf("floating close must not keep rendering closed floating frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("floating close must exit copy mode bound to closed floating, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeFloatingDeactivateKeepsFrozenHistoryUntilExplicitExit(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 7, 40, 12, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-float",
		Token:       "tok-float",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "float-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-float",
		BoundToken: "tok-float",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceFloatingCommand(root, state.FloatingCommand{
		Action: state.FloatingCommandDeactivate,
		Source: state.PaneCommandSourceTest,
	})

	if next.History.Token != "tok-float" || len(next.History.Rows) != 1 {
		t.Fatalf("floating deactivate should keep frozen history until ESC, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.BoundToken != "tok-float" {
		t.Fatalf("floating deactivate should keep copy mode bound to inactive floating, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimeFloatingDeactivateKeepsFrozenHistoryInFloating(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-float",
				Token:       "tok-float",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "float-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "float-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-float",
				BoundToken: "tok-float",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-float", 7, 40, 12, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action: state.FloatingCommandDeactivate,
		Source: state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post floating deactivate: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating deactivate: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if !frameContains(last, "float-history") {
		t.Fatalf("floating deactivate should keep floating frozen history visible in its window, got %#v", last.Lines)
	}
	if !runtime.State().CopyMode.Active {
		t.Fatalf("floating deactivate should keep copy mode until explicit ESC, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeTerminalPoolAttachRebindInvalidatesFrozenHistoryForNewPaneTerminal(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(
			state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true),
		),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-old",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, _ := reduceTerminalPoolAttachResultPrepared(root, TerminalPoolAttachResultMsg{
		TerminalID: "term-new",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.TerminalID != "term-new" || next.History.Token != "" || len(next.History.Rows) != 0 {
		t.Fatalf("terminal pool pane attach rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("terminal pool pane attach rebind must clear old frozen copy binding, got %#v", next.CopyMode)
	}
}

func TestCopyModeTerminalPoolReattachSamePaneTerminalKeepsFrozenHistory(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(
			state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true),
		),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, _ := reduceTerminalPoolAttachResultPrepared(root, TerminalPoolAttachResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    false,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool pane attach must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 78 || next.CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool pane attach must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal terminal pool pane attach must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeTerminalPoolAttachRebindInvalidatesFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-old",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-old",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceTerminalPoolAttachResultPrepared(root, TerminalPoolAttachResultMsg{
		TerminalID:       "term-new",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    true,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.TerminalID != "term-new" || next.History.Token != "" || len(next.History.Rows) != 0 {
		t.Fatalf("terminal pool floating attach rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("terminal pool floating attach rebind must clear old frozen copy binding, got %#v", next.CopyMode)
	}
}

func TestCopyModeTerminalPoolReattachSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceTerminalPoolAttachResultPrepared(root, TerminalPoolAttachResultMsg{
		TerminalID:       "term-1",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    false,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool floating attach must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 40 || next.CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool floating attach must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal terminal pool floating attach must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeRuntimeTerminalPoolAttachRebindDoesNotRenderOldFrozenHistoryForNewPaneTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID: "term-new",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	}); err != nil {
		t.Fatalf("post terminal pool attach rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal pool attach rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("terminal pool pane attach rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("terminal pool pane attach rebind should fall back to pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeTerminalPoolReattachSamePaneTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	if !frameContains(lastFrame(t, host.Frames()), "old-history") {
		t.Fatalf("expected initial frame to render old frozen history, frames=%#v", host.Frames())
	}

	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    false,
		},
	}); err != nil {
		t.Fatalf("post same-terminal terminal pool pane attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal terminal pool pane attach: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool pane attach must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.TerminalID != "term-1" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.BoundCols != 78 || runtime.State().CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool pane attach must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}

	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-history") {
		t.Fatalf("same-terminal terminal pool pane attach must keep rendering frozen history, got %#v", last.Lines)
	}
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal terminal pool pane attach must not fall back to pending history, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeTerminalPoolAttachRebindDoesNotRenderOldFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID:       "term-new",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    true,
		},
	}); err != nil {
		t.Fatalf("post terminal pool floating attach rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal pool floating attach rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("terminal pool floating attach rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.TerminalID != "term-new" || runtime.State().CopyMode.BoundToken != "" || runtime.State().History.TerminalID != "term-new" {
		t.Fatalf("terminal pool floating attach rebind should leave copy mode in pending state for new terminal, state=%#v", runtime.State())
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("terminal pool floating attach rebind should render pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeTerminalPoolReattachSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	initialBoundCols := runtime.State().CopyMode.BoundCols

	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID:       "term-1",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    false,
		},
	}); err != nil {
		t.Fatalf("post same-terminal terminal pool floating attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal terminal pool floating attach: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool floating attach must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.TerminalID != "term-1" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.BoundCols != initialBoundCols || runtime.State().CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool floating attach must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("same-terminal terminal pool floating attach must not force pending latest, got %#v", runtime.State().History.Pending)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal terminal pool floating attach must not fall back to pending history, got %#v", last.Lines)
	}
}

func TestCopyModeTerminalPoolReconnectRebindInvalidatesFrozenHistoryForNewPaneTerminal(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(
			state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true),
		),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-old",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-old",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, _ := reduceTerminalPoolReconnectResultPrepared(root, TerminalPoolReconnectResultMsg{
		TerminalID: "term-new",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.TerminalID != "term-new" || next.History.Token != "" || len(next.History.Rows) != 0 {
		t.Fatalf("terminal pool pane reconnect rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("terminal pool pane reconnect rebind must clear old frozen copy binding, got %#v", next.CopyMode)
	}
}

func TestCopyModeTerminalPoolReconnectSamePaneTerminalKeepsFrozenHistory(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(
			state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true),
		),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, _ := reduceTerminalPoolReconnectResultPrepared(root, TerminalPoolReconnectResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    false,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool pane reconnect must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 78 || next.CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool pane reconnect must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal terminal pool pane reconnect must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeRuntimeTerminalPoolReconnectRebindDoesNotRenderOldFrozenHistoryForNewPaneTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-old", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := postPreparedTerminalPoolReconnectResult(runtime, TerminalPoolReconnectResultMsg{
		TerminalID: "term-new",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	}); err != nil {
		t.Fatalf("post terminal pool reconnect rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal pool reconnect rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("terminal pool pane reconnect rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("terminal pool pane reconnect rebind should fall back to pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeTerminalPoolReconnectSamePaneTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface-old", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
			History: state.HistoryStore{
				PaneID:      state.DefaultPaneID,
				ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        78,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     state.DefaultPaneID,
				ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  78,
				ViewRows:   20,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	if !frameContains(lastFrame(t, host.Frames()), "old-history") {
		t.Fatalf("expected initial frame to render old frozen history, frames=%#v", host.Frames())
	}

	if err := postPreparedTerminalPoolReconnectResult(runtime, TerminalPoolReconnectResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      9,
			Cols:         78,
			Rows:         20,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    false,
		},
	}); err != nil {
		t.Fatalf("post same-terminal terminal pool pane reconnect: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal terminal pool pane reconnect: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool pane reconnect must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.TerminalID != "term-1" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.BoundCols != 78 || runtime.State().CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool pane reconnect must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}

	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-history") {
		t.Fatalf("same-terminal terminal pool pane reconnect must keep rendering frozen history, got %#v", last.Lines)
	}
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal terminal pool pane reconnect must not fall back to pending history, got %#v", last.Lines)
	}
}

func TestCopyModeTerminalPoolReconnectRebindInvalidatesFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-old",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-old",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceTerminalPoolReconnectResultPrepared(root, TerminalPoolReconnectResultMsg{
		TerminalID:       "term-new",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    true,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.TerminalID != "term-new" || next.History.Token != "" || len(next.History.Rows) != 0 {
		t.Fatalf("terminal pool floating reconnect rebind must invalidate old frozen history window, got %#v", next.History)
	}
	if next.CopyMode.TerminalID != "term-new" || next.CopyMode.BoundToken != "" || next.CopyMode.BoundCols != 0 || !next.CopyMode.Empty {
		t.Fatalf("terminal pool floating reconnect rebind must clear old frozen copy binding, got %#v", next.CopyMode)
	}
}

func TestCopyModeTerminalPoolReconnectSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.TerminalViews = root.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))
	root.History = state.HistoryStore{
		PaneID:      "float-pane",
		ViewID:      state.TerminalFloatingViewID("floating-1"),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        40,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     "float-pane",
		ViewID:     state.TerminalFloatingViewID("floating-1"),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  40,
		ViewRows:   10,
	}

	next, _ := reduceTerminalPoolReconnectResultPrepared(root, TerminalPoolReconnectResultMsg{
		TerminalID:       "term-1",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    false,
		},
	}, LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if next.History.Token != "tok-old" || next.History.TerminalID != "term-1" || len(next.History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool floating reconnect must keep frozen history window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.TerminalID != "term-1" || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 40 || next.CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool floating reconnect must keep frozen copy binding, got %#v", next.CopyMode)
	}
	if next.History.Pending != nil {
		t.Fatalf("same-terminal terminal pool floating reconnect must not force pending latest, got %#v", next.History.Pending)
	}
}

func TestCopyModeRuntimeTerminalPoolReconnectRebindDoesNotRenderOldFrozenHistoryForNewFloatingTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-old",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-old",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-old",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-old"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-old", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := postPreparedTerminalPoolReconnectResult(runtime, TerminalPoolReconnectResultMsg{
		TerminalID:       "term-new",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-new",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    true,
		},
	}); err != nil {
		t.Fatalf("post terminal pool floating reconnect rebind: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal pool floating reconnect rebind: %v", err)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-history") {
		t.Fatalf("terminal pool floating reconnect rebind must not keep rendering old frozen history, got %#v", last.Lines)
	}
	if runtime.State().CopyMode.TerminalID != "term-new" || runtime.State().CopyMode.BoundToken != "" || runtime.State().History.TerminalID != "term-new" {
		t.Fatalf("terminal pool floating reconnect rebind should leave copy mode in pending state for new terminal, state=%#v", runtime.State())
	}
	if !frameContains(last, "window pending") {
		t.Fatalf("terminal pool floating reconnect rebind should render pending history state, got %#v", last.Lines)
	}
}

func TestCopyModeRuntimeTerminalPoolReconnectSameFloatingTerminalKeepsFrozenHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       120,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       120,
				Rows:       24,
				Lines:      []string{"live-new"},
			},
			History: state.HistoryStore{
				PaneID:      "float-pane",
				ViewID:      state.TerminalFloatingViewID("floating-1"),
				TerminalID:  "term-1",
				Token:       "tok-old",
				Cols:        40,
				SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-history", LineID: 20}}),
				Rows:        []state.HistoryRow{{Text: "old-history", LineID: 20}},
			},
			CopyMode: state.CopyModeStore{
				Active:     true,
				PaneID:     "float-pane",
				ViewID:     state.TerminalFloatingViewID("floating-1"),
				TerminalID: "term-1",
				BoundToken: "tok-old",
				BoundCols:  40,
				ViewRows:   10,
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20},
	)
	runtime.state.Shell, _ = runtime.state.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Title:    "float",
		Pane:     state.PaneState{ID: "float-pane", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 4, Y: 2, W: 40, H: 12},
	})
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindFloating(state.NewFloatingTerminalView("floating-1", "float-pane", "term-1", 7, 40, 12, state.TerminalResizeRoleOwner, "surface-old", state.TerminalFloatingViewID("floating-1"), true))

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial frozen history: %v", err)
	}
	initialBoundCols := runtime.State().CopyMode.BoundCols

	if err := postPreparedTerminalPoolReconnectResult(runtime, TerminalPoolReconnectResultMsg{
		TerminalID:       "term-1",
		TargetFloatingID: "floating-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         40,
			Rows:         12,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-new",
			ViewID:       state.TerminalFloatingViewID("floating-1"),
			CanResize:    false,
		},
	}); err != nil {
		t.Fatalf("post same-terminal terminal pool floating reconnect: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain same-terminal terminal pool floating reconnect: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || runtime.State().History.TerminalID != "term-1" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("same-terminal terminal pool floating reconnect must keep frozen history window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.TerminalID != "term-1" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.BoundCols != initialBoundCols || runtime.State().CopyMode.Empty {
		t.Fatalf("same-terminal terminal pool floating reconnect must keep frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("same-terminal terminal pool floating reconnect must not force pending latest, got %#v", runtime.State().History.Pending)
	}

	last := lastFrame(t, host.Frames())
	if frameContains(last, "window pending") {
		t.Fatalf("same-terminal terminal pool floating reconnect must not fall back to pending history, got %#v", last.Lines)
	}
}

func TestCopyModeOlderGuardSilentlyBlocksAnyPendingHistoryRequest(t *testing.T) {
	reducer := NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		Viewport: state.ViewportStore{
			Valid: true,
			Cols:  80,
			Rows:  24,
		},
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       78,
			Generation: 7,
			Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 20},
			Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 20},
			Rows:       []state.HistoryRow{{Text: "new", LineID: 20}},
			Pending: &state.HistoryPendingRequest{
				ID:         9,
				Kind:       state.HistoryRequestLatest,
				TerminalID: "term-1",
				Cols:       78,
			},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  78,
		},
	}

	next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}})
	if len(effects) != 1 {
		t.Fatalf("pending guard should only return handled effect, got %#v", effects)
	}
	if _, ok := effects[0].(handledEffect); !ok {
		t.Fatalf("pending guard should keep input handled, got %#v", effects[0])
	}
	if next.Session.LastError != "" || next.Surface.Err != "" {
		t.Fatalf("pending guard should stay silent, session=%q surface=%q", next.Session.LastError, next.Surface.Err)
	}
	if next.History.Pending == nil || next.History.Pending.Kind != state.HistoryRequestLatest {
		t.Fatalf("pending latest request should remain unchanged, got %#v", next.History.Pending)
	}
}

func TestCopyModeFooterOlderActionUsesAuthoritativeHistoryPath(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "new", LineID: 20}},
		)}},
		OlderResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowPrepend,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "old", LineID: 10}},
		)}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 20}
	host := NewFakeTerminalHost(8)
	terminal := &testkit.FakeTerminalService{}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell(),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Attached:   true,
				Cols:       80,
				Rows:       24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	host.SetSize(80, 24)

	if err := runtime.Post(shortcutTestMessage("menu.copy", "", false, 0)); err != nil {
		t.Fatalf("post copy footer latest action: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy footer latest action: %v", err)
	}
	if len(core.LatestRequests) != 1 || core.LatestRequests[0].TerminalID != "term-1" || core.LatestRequests[0].Cols != 78 {
		t.Fatalf("copy footer action should enter latest through copy reducer, got %#v", core.LatestRequests)
	}

	action := frameActionHitRegion(t, lastFrame(t, host.Frames()), "copy.request_older", "")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send copy footer older click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy footer older click: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("expected copy footer older request, got %#v", core.OlderRequests)
	}
	olderReq := core.OlderRequests[0]
	if olderReq.Token != "tok-1" || olderReq.Generation != 7 || olderReq.Boundary.LastLineID != 20 {
		t.Fatalf("unexpected copy footer older request %#v", olderReq)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"old", "new"}) {
		t.Fatalf("copy footer older should prepend authoritative rows, got %v", got)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("copy footer older action must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestCopyModeLatestFallsBackToCopyContentRectColsWithoutNativeSurface(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "new", LineID: 20}},
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.Surface = state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"live-without-native-size"}}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 78 {
		t.Fatalf("copy latest must use content rect cols, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundCols != 78 || runtime.State().History.Cols != 78 {
		t.Fatalf("expected copy/history bound to content cols, got %#v", runtime.State())
	}
}

func TestCopyModeLatestIgnoresLiveSurfaceColsForHistoryRequest(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: strings.Repeat("x", 76), LineID: 20}},
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)
	binding, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.ResizeRole = state.TerminalResizeRoleFollower
	binding.DesiredCols = 78
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)
	runtime.state.Surface = runtime.state.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       120,
		Rows:       24,
		Lines:      []string{"native-wide-live"},
	})

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 78 {
		t.Fatalf("copy latest must use binding cols without live surface input, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundCols != 78 || runtime.State().History.Cols != 78 {
		t.Fatalf("expected copy/history bound to requested history cols, got %#v", runtime.State())
	}
}

func TestCopyModeHostResizeKeepsTerminalGeometryAndFrozenWindow(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{
			{Window: historyWindowForApp(
				state.HistoryWindowReplace,
				"term-1",
				"tok-1",
				78,
				7,
				[]state.HistoryRow{{Text: "old-window", LineID: 20}},
			)},
			{Window: historyWindowForApp(
				state.HistoryWindowReplace,
				"term-1",
				"tok-2",
				98,
				8,
				[]state.HistoryRow{{Text: "new-window", LineID: 30}},
			)},
		},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := runtime.Post(CopyModeSetMarkMsg{Position: state.CopyPosition{Row: 0, Col: 1}}); err != nil {
		t.Fatalf("post mark: %v", err)
	}
	if err := runtime.Post(CopyModeMoveCursorMsg{Position: state.CopyPosition{Row: 0, Col: 4}}); err != nil {
		t.Fatalf("post cursor: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain selection: %v", err)
	}

	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain rebind: %v", err)
	}

	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 78 {
		t.Fatalf("host resize should not request a second latest snapshot, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundToken != "tok-1" || runtime.State().CopyMode.BoundCols != 78 || runtime.State().CopyMode.ViewRows != 20 {
		t.Fatalf("host resize must not expand frozen history beyond terminal geometry, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Mark == nil || runtime.State().CopyMode.Selection == nil || runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 0, Col: 4}) {
		t.Fatalf("local reflow should preserve selection and cursor on same logical line, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-window") || frameContains(last, "new-window") {
		t.Fatalf("resized copy mode must keep frozen history content, got %#v", last.Lines)
	}
}

func TestCopyModeResizeRebindInvalidatesOldWindowBeforeLatestResponse(t *testing.T) {
	reducer := NewCopyModeResizeRebindReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		Viewport: state.ViewportStore{
			Valid: true,
			Cols:  100,
			Rows:  40,
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-old",
			Cols:       78,
			Rows:       []state.HistoryRow{{Text: "old-window", LineID: 20}},
			Lines:      []state.HistoryLineSpan{{LineID: 20, StartRow: 0, EndRow: 0}},
			Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 20},
			Generation: 7,
			Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 20},
			HasMore:    true,
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			BoundToken:  "tok-old",
			BoundCols:   78,
			ViewRows:    20,
			ViewportTop: 4,
			Cursor:      state.CopyPosition{Row: 0, Col: 3},
		},
	}
	mark := state.CopyPosition{Row: 0, Col: 1}
	root.CopyMode.Mark = &mark
	root.CopyMode.Selection = &state.CopySelection{Anchor: mark, Focus: root.CopyMode.Cursor}
	root.History.SourceLines = historyLogicalLinesForApp(root.History.Rows)

	next, effects := reducer(root, HostResizeMsg{Cols: 100, Rows: 40})
	if len(effects) != 0 {
		t.Fatalf("frozen copy resize should not schedule latest effect, got %#v", effects)
	}
	if next.History.Token != "tok-old" || next.History.Cols != 78 || len(next.History.Rows) == 0 {
		t.Fatalf("pane resize should keep frozen authoritative source at terminal cols, got %#v", next.History)
	}
	if next.History.Pending != nil {
		t.Fatalf("resize should not wait for latest at new cols, got %#v", next.History.Pending)
	}
	if next.CopyMode.BoundToken != "tok-old" || next.CopyMode.BoundCols != 78 || next.CopyMode.ViewRows != 20 {
		t.Fatalf("copy mode should keep frozen terminal geometry when only the pane changes, got %#v", next.CopyMode)
	}
	if next.CopyMode.Cursor != (state.CopyPosition{Row: 0, Col: 3}) || next.CopyMode.Mark == nil || next.CopyMode.Selection == nil {
		t.Fatalf("local reflow should preserve cursor and selection, got %#v", next.CopyMode)
	}
}

func TestCopyModeResizeRebindUpdatesPendingLatestLocalWidth(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	requestID := state.RequestID(9)
	pending := state.HistoryPendingRequest{
		ID: requestID, PaneID: state.DefaultPaneID, ViewID: viewID,
		EndpointID: state.EndpointID("local"), TerminalID: "term-1", Cols: 78,
		Kind: state.HistoryRequestLatest,
	}
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", viewID, true,
		)),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 40},
		History:  state.HistoryStore{Pending: &pending},
		CopyMode: state.CopyModeStore{
			Entering: true, Phase: state.CopyModeEnteringPending,
			PaneID: state.DefaultPaneID, ViewID: viewID, EndpointID: state.EndpointID("local"),
			TerminalID: "term-1", RequestID: requestID, BoundCols: 78, ViewRows: 20,
		},
	}
	reducer := NewCopyModeResizeRebindReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})

	next, effects := reducer(root, HostResizeMsg{Cols: 100, Rows: 40})
	if len(effects) != 0 {
		t.Fatalf("pending latest resize should only change local reflow dimensions, got %#v", effects)
	}
	if next.History.Pending == nil || next.History.Pending.Cols != 78 {
		t.Fatalf("pending latest must remain bound to terminal width, got %#v", next.History.Pending)
	}
	if next.CopyMode.BoundCols != 78 || next.CopyMode.ViewRows != 20 || !next.CopyMode.Entering {
		t.Fatalf("entering copy mode must retain terminal dimensions: %#v", next.CopyMode)
	}
	if root.History.Pending.Cols != 78 {
		t.Fatalf("reducer mutated the previous root pending request: %#v", root.History.Pending)
	}
}

func TestCopyModeResizeRebindPendingFrameDoesNotShowOldRowsOrLiveFallback(t *testing.T) {
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Lines:      []string{"live-fallback"},
		},
		Viewport: state.ViewportStore{
			Valid: true,
			Cols:  100,
			Rows:  40,
		},
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-old",
			Cols:       78,
			Rows:       []state.HistoryRow{{Text: "old-window", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}
	reducer := NewCopyModeResizeRebindReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})
	root.History.SourceLines = historyLogicalLinesForApp(root.History.Rows)
	root, _ = reducer(root, HostResizeMsg{Cols: 100, Rows: 40})
	frame := render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))

	if !frameContains(frame, "old-window") || frameContains(frame, "live-fallback") {
		t.Fatalf("resize should keep frozen history and must not fallback to live surface, got %#v", frame.Lines)
	}
}

func TestCopyModeResizeRebindRecoversClippedSourceFromRowsFallback(t *testing.T) {
	reducer := NewCopyModeResizeRebindReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		Viewport: state.ViewportStore{
			Valid: true,
			Cols:  100,
			Rows:  40,
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-old",
			Cols:       78,
			Rows: []state.HistoryRow{
				{Text: "abc", LineID: 20, RowInLine: 0, ClippedStart: true},
				{Text: "def", LineID: 20, RowInLine: 1, ClippedEnd: true},
			},
			Lines: []state.HistoryLineSpan{{
				LineID:        20,
				StartRow:      0,
				EndRow:        1,
				ClippedBefore: true,
				ClippedAfter:  true,
			}},
			Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 20},
			Generation: 7,
			Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 20},
			HasMore:    true,
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}
	binding, _ := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 98
	binding.DesiredRows = 36
	root.TerminalViews = root.TerminalViews.BindPane(binding)

	next, effects := reducer(root, HostResizeMsg{Cols: 100, Rows: 40})
	if len(effects) != 0 {
		t.Fatalf("frozen copy resize should stay local, got %#v", effects)
	}
	if len(next.History.SourceLines) != 1 {
		t.Fatalf("rows fallback should recover one logical-line source, got %#v", next.History.SourceLines)
	}
	if got := next.History.SourceLines[0]; got.Text != "abcdef" || !got.ClippedBefore || !got.ClippedAfter {
		t.Fatalf("rows fallback must preserve clipped source flags during rebind, got %#v", got)
	}
	if len(next.History.Lines) != 1 || !next.History.Lines[0].ClippedBefore || !next.History.Lines[0].ClippedAfter {
		t.Fatalf("local reflow after rows fallback must keep clipped span semantics, got %#v", next.History.Lines)
	}
}

func TestCopyModeResizeRebindRuntimeRendersPendingBeforeLatestResponse(t *testing.T) {
	runner := &recordingEffectRunner{}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)
	runtime.state.Surface.Lines = []string{"live-fallback"}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-window", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-window", LineID: 20}},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}

	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize: %v", err)
	}

	if len(runner.Effects) != 0 {
		t.Fatalf("frozen copy resize should not schedule latest request, got %#v", runner.Effects)
	}
	pendingFrame := lastFrame(t, host.Frames())
	if !frameContains(pendingFrame, "old-window") || frameContains(pendingFrame, "live-fallback") {
		t.Fatalf("runtime resize must keep frozen history and avoid live fallback, got %#v", pendingFrame.Lines)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("runtime should not keep pending latest request after local reflow, got %#v", runtime.State().History.Pending)
	}
}

func TestCopyModeResizeRowsOnlyKeepsWindowAndDoesNotRequestLatest(t *testing.T) {
	reducer := NewCopyModeResizeRebindReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}, Rows: 20})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Shell:   state.DefaultShell(),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
		Viewport: state.ViewportStore{
			Valid: true,
			Cols:  80,
			Rows:  18,
		},
		History: state.HistoryStore{
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       78,
			Rows: []state.HistoryRow{
				{Text: "one", LineID: 1},
				{Text: "two", LineID: 2},
				{Text: "three", LineID: 3},
				{Text: "four", LineID: 4},
			},
		},
		CopyMode: state.CopyModeStore{
			Active:      true,
			TerminalID:  "term-1",
			BoundToken:  "tok-1",
			BoundCols:   78,
			ViewRows:    20,
			ViewportTop: 3,
			Cursor:      state.CopyPosition{Row: 2, Col: 1},
		},
	}

	next, effects := reducer(root, HostResizeMsg{Cols: 80, Rows: 18})
	if len(effects) != 0 {
		t.Fatalf("rows-only resize must not request latest, got %#v", effects)
	}
	if next.History.Token != "tok-1" || next.History.Cols != 78 || len(next.History.Rows) != 4 {
		t.Fatalf("rows-only resize should preserve authoritative window, got %#v", next.History)
	}
	if next.CopyMode.BoundToken != "tok-1" || next.CopyMode.BoundCols != 78 || next.CopyMode.ViewRows != 14 {
		t.Fatalf("rows-only resize should only update view rows, got %#v", next.CopyMode)
	}
	if next.CopyMode.Cursor != (state.CopyPosition{Row: 2, Col: 1}) {
		t.Fatalf("rows-only resize should preserve cursor, got %#v", next.CopyMode.Cursor)
	}
	if next.CopyMode.ViewportTop != 0 {
		t.Fatalf("rows-only resize should clamp viewport to the loaded top when all rows fit, got %#v", next.CopyMode)
	}
}

func TestCopyModePaneSizeCommandReflowsFrozenWindowAtContentCols(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 38, 7, []state.HistoryRow{{Text: "old-window", LineID: 20}})},
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-2", 22, 8, []state.HistoryRow{{Text: "sized-window", LineID: 30}})},
		},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	initialShell := state.DefaultShell().
		SetPanelPresentation(state.PanelPresentationSplitLine).
		SplitActivePane(state.PaneState{ID: "pane-2", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: initialShell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4}}},
		CopyModeDeps{Core: core, Rows: 20},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfigForPane("term-1", 80, 24, "pane-2")}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:   state.PaneCommandSetSize,
		Target:   state.PaneCommandTarget{PaneID: "pane-2"},
		SizeMode: state.PaneSizeCells,
		Cols:     24,
	}}); err != nil {
		t.Fatalf("post pane size command: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane size rebind: %v", err)
	}

	if len(core.LatestRequests) != 1 || core.LatestRequests[0].Cols != 38 {
		t.Fatalf("pane size command must not request a second latest snapshot, got %#v", core.LatestRequests)
	}
	if runtime.State().CopyMode.BoundToken != "tok-1" || runtime.State().CopyMode.BoundCols != 22 {
		t.Fatalf("expected copy mode to keep frozen token and update cols locally, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Token != "tok-1" || runtime.State().History.Cols != 22 || len(runtime.State().History.Rows) != 1 || runtime.State().History.Rows[0].Text != "old-window" {
		t.Fatalf("pane size reflow must keep authoritative frozen source, got %#v", runtime.State().History)
	}
}

func TestInteractiveRuntimeHostResizeKeepsReboundCopyWindowAfterTerminalResizeResult(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 78, Rows: 20}}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 78, 7, []state.HistoryRow{{Text: "old-window", LineID: 20}})},
			{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-2", 98, 8, []state.HistoryRow{{Text: "new-window", LineID: 30}})},
		},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize/rebind: %v", err)
	}

	if len(terminal.Resizes) != 1 || terminal.Resizes[0].Cols != 98 || terminal.Resizes[0].Rows != 36 {
		t.Fatalf("expected terminal resize to content rect, got %#v", terminal.Resizes)
	}
	if runtime.State().CopyMode.BoundToken != "tok-1" || runtime.State().History.Token != "tok-1" || runtime.State().History.Cols != 98 {
		t.Fatalf("terminal resize result must preserve frozen copy window and local cols, got %#v", runtime.State())
	}
}

func TestCopyModeResizeRejectsOldColsResponseAsStale(t *testing.T) {
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-2", 98, 8, []state.HistoryRow{{Text: "rebound", LineID: 30}})}}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.CopyMode = state.CopyModeStore{Active: true, PaneID: state.DefaultPaneID, ViewID: state.TerminalPaneViewID(state.DefaultPaneID), TerminalID: "term-1", BoundToken: "tok-1", BoundCols: 78}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-window", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-window", LineID: 20}},
	}

	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize: %v", err)
	}
	if len(core.LatestRequests) != 0 {
		t.Fatalf("resize should not rebind latest snapshot, got %#v", core.LatestRequests)
	}
	if runtime.State().History.Token != "tok-1" || len(runtime.State().History.Rows) != 1 || runtime.State().History.Rows[0].Text != "old-window" {
		t.Fatalf("expected frozen window before stale response, got %#v", runtime.State().History)
	}
	staleWindow := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-old", 98, 7, []state.HistoryRow{{Text: "stale", LineID: 1}})
	staleWindow.ViewID = "stale-view"
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 99, Window: staleWindow}}); err != nil {
		t.Fatalf("post stale response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale response: %v", err)
	}

	if runtime.State().History.Token != "tok-1" || len(runtime.State().History.Rows) != 1 || runtime.State().History.Rows[0].Text != "old-window" {
		t.Fatalf("stale response must not disturb frozen window, got %#v", runtime.State().History)
	}
	if runtime.State().Session.LastError != "" || runtime.State().Surface.Err != "" {
		t.Fatalf("stale cols response must not pollute live errors, got %#v", runtime.State())
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-window") || frameContains(last, "│stale") {
		t.Fatalf("stale resize response must keep frozen rows and reject stale payload, got %#v", last.Lines)
	}
}

func TestCopyModeRejectsSiblingViewHistoryResponseForSameTerminal(t *testing.T) {
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "active-window", LineID: 20}},
	)}}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if runtime.State().History.Token != "tok-1" || runtime.State().CopyMode.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("expected active copy binding, got %#v", runtime.State())
	}

	sibling := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-sibling", 78, 8, []state.HistoryRow{{Text: "sibling-window", LineID: 30}})
	sibling.PaneID = "pane-2"
	sibling.ViewID = "pane:pane-2"
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: core.LatestRequests[0].RequestID, Window: sibling}}); err != nil {
		t.Fatalf("post sibling response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain sibling response: %v", err)
	}

	if runtime.State().History.Token != "tok-1" || len(runtime.State().History.Rows) != 1 || runtime.State().History.Rows[0].Text != "active-window" {
		t.Fatalf("sibling view response must not replace active authoritative window, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || runtime.State().CopyMode.BoundToken != "tok-1" {
		t.Fatalf("sibling view response must not disturb active copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "sibling-window") || !frameContains(last, "active-window") {
		t.Fatalf("sibling view response must not render into active copy frame, got %#v", last.Lines)
	}
}

func TestCopyModeRebindIgnoresStaleResponseFromPreviousViewBinding(t *testing.T) {
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-2",
		98,
		8,
		[]state.HistoryRow{{Text: "rebound-window", LineID: 30}},
	)}}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-window", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-window", LineID: 20}},
	}

	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize rebind: %v", err)
	}
	if runtime.State().History.Token != "tok-old" || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().History.Cols != 78 {
		t.Fatalf("expected frozen active copy binding to retain terminal cols, got history=%#v copy=%#v", runtime.State().History, runtime.State().CopyMode)
	}

	stale := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-stale", 98, 9, []state.HistoryRow{{Text: "previous-view", LineID: 40}})
	stale.PaneID = "pane-old"
	stale.ViewID = "pane:pane-old"
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 99, Window: stale}}); err != nil {
		t.Fatalf("post stale view response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale view response: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || len(runtime.State().History.Rows) != 1 || runtime.State().History.Rows[0].Text != "old-window" {
		t.Fatalf("stale previous-view response must not replace frozen window, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || runtime.State().CopyMode.BoundToken != "tok-old" {
		t.Fatalf("stale previous-view response must not disturb frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "previous-view") || !frameContains(last, "old-window") {
		t.Fatalf("stale previous-view response must not render into frozen frame, got %#v", last.Lines)
	}
}

func TestCopyModeIgnoresLaterDestructiveLatestAndKeepsFrozenRows(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, &testkit.FakeCoreClient{}, nil)
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}},
	}

	destructive := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-after-clear", 78, 9, []state.HistoryRow{{Text: "new-live-tail", LineID: 30}})
	destructive.Boundary = state.HistoryBoundary{FirstLineID: 30, LastLineID: 30}
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 999, Window: destructive}}); err != nil {
		t.Fatalf("post destructive latest response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain destructive latest response: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || len(runtime.State().History.Rows) != 2 || runtime.State().History.Rows[0].Text != "old-one" || runtime.State().History.Rows[1].Text != "old-two" {
		t.Fatalf("later destructive latest must not replace frozen copy window, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("later destructive latest must not disturb frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-one") || !frameContains(last, "old-two") || frameContains(last, "new-live-tail") {
		t.Fatalf("later destructive latest must not render into frozen copy frame, got %#v", last.Lines)
	}
}

func TestCopyModeIgnoresLaterRestartLatestAndKeepsFrozenRows(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, &testkit.FakeCoreClient{}, nil)
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}},
	}

	restarted := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-after-restart", 78, 10, nil)
	restarted.Boundary = state.HistoryBoundary{}
	restarted.TotalLines = 0
	restarted.HasMore = false
	restarted.Cursor = state.HistoryCursor{}
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 1001, Window: restarted}}); err != nil {
		t.Fatalf("post restart latest response: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain restart latest response: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || len(runtime.State().History.Rows) != 2 || runtime.State().History.Rows[0].Text != "old-one" || runtime.State().History.Rows[1].Text != "old-two" {
		t.Fatalf("later restart latest must not replace frozen copy window, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("later restart latest must not disturb frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-one") || !frameContains(last, "old-two") {
		t.Fatalf("later restart latest must keep frozen copy frame content, got %#v", last.Lines)
	}
	if frameContains(last, "history copy empty") {
		t.Fatalf("later restart latest must not replace frozen copy frame with empty restart view, got %#v", last.Lines)
	}
}

func TestCopyModeRestartResultKeepsFrozenHistory(t *testing.T) {
	root := state.Root{
		Shell: state.DefaultShell(),
		History: state.HistoryStore{
			PaneID:      state.DefaultPaneID,
			ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID:  "term-1",
			Token:       "tok-old",
			Cols:        78,
			SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}}),
			Rows:        []state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-old",
			BoundCols:  78,
			ViewRows:   20,
		},
	}

	next, effects := reduceTerminalPoolRestartResult(root, TerminalPoolRestartResultMsg{TerminalID: "term-1"}, LiveDeps{})
	if len(effects) != 1 {
		t.Fatalf("restart result should still trigger pool refresh, got %#v", effects)
	}
	if next.History.Token != "tok-old" || len(next.History.Rows) != 2 || next.History.Rows[0].Text != "old-one" || next.History.Rows[1].Text != "old-two" {
		t.Fatalf("restart result must not replace frozen copy window, got %#v", next.History)
	}
	if !next.CopyMode.Active || next.CopyMode.BoundToken != "tok-old" || next.CopyMode.TerminalID != "term-1" {
		t.Fatalf("restart result must not disturb frozen copy binding, got %#v", next.CopyMode)
	}
}

func TestCopyModeRuntimeRestartResultKeepsFrozenRows(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newCopyModeRuntime(host, &testkit.FakeCoreClient{}, nil)
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-old",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}}),
		Rows:        []state.HistoryRow{{Text: "old-one", LineID: 10}, {Text: "old-two", LineID: 20}},
	}

	if err := runtime.Post(TerminalPoolRestartResultMsg{TerminalID: "term-1"}); err != nil {
		t.Fatalf("post restart result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain restart result: %v", err)
	}

	if runtime.State().History.Token != "tok-old" || len(runtime.State().History.Rows) != 2 || runtime.State().History.Rows[0].Text != "old-one" || runtime.State().History.Rows[1].Text != "old-two" {
		t.Fatalf("restart result must not replace frozen copy window, got %#v", runtime.State().History)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.BoundToken != "tok-old" || runtime.State().CopyMode.TerminalID != "term-1" {
		t.Fatalf("restart result must not disturb frozen copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if !frameContains(last, "old-one") || !frameContains(last, "old-two") {
		t.Fatalf("restart result must keep frozen copy frame content, got %#v", last.Lines)
	}
	if frameContains(last, "history copy empty") || frameContains(last, "window pending") {
		t.Fatalf("restart result must not replace frozen copy frame with cleared history state, got %#v", last.Lines)
	}
}

func TestCopyModeExitThenReenterPendingDoesNotReuseStaleFrozenRows(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)
	pendingHistory, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID:         1,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
	})
	if err != nil {
		t.Fatalf("begin latest seed: %v", err)
	}
	initialWindow := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-old",
		78,
		7,
		[]state.HistoryRow{{Text: "old-frozen", LineID: 20}},
	)
	initialWindow.PaneID = state.DefaultPaneID
	initialWindow.ViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	runtime.state.History, _, err = pendingHistory.ApplyWindow(1, initialWindow)
	if err != nil {
		t.Fatalf("seed frozen history: %v", err)
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-old",
		BoundCols:  78,
		ViewRows:   20,
	}
	host.sink.frames = nil

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("expected copy mode to exit, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 0 || len(runtime.State().History.SourceLines) != 0 || len(runtime.State().History.Lines) != 0 {
		t.Fatalf("copy exit must release local frozen history window, got %#v", runtime.State().History)
	}

	stale := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-stale",
		78,
		8,
		[]state.HistoryRow{{Text: "stale-after-exit", LineID: 30}},
	)
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 999, Window: stale}}); err != nil {
		t.Fatalf("post stale after exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale after exit: %v", err)
	}
	if runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 0 {
		t.Fatalf("stale response after exit must not revive released history, got %#v", runtime.State().History)
	}

	runner.Effects = nil
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain page up: %v", err)
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.Kind != state.HistoryRequestLatest {
		t.Fatalf("expected new pending latest request after re-enter, got %#v", runtime.State().History.Pending)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "old-frozen") || frameContains(last, "stale-after-exit") || frameContains(last, "window pending") {
		t.Fatalf("re-enter pending frame must not reuse stale frozen rows or show pending text, got %#v", last.Lines)
	}
	if !frameContains(last, "live") {
		t.Fatalf("re-enter pending should keep live frame while latest loads, got %#v", last.Lines)
	}
}

func TestCopyModeExitWhileLatestPendingIgnoresDelayedMatchingLatest(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain page up: %v", err)
	}
	pending := runtime.State().History.Pending
	if pending == nil || pending.Kind != state.HistoryRequestLatest {
		t.Fatalf("expected latest pending request, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().CopyMode.Active || !runtime.State().CopyMode.Entering {
		t.Fatalf("expected copy mode entering while latest pending, got %#v", runtime.State().CopyMode)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if runtime.State().CopyMode.Active || runtime.State().CopyMode.Entering {
		t.Fatalf("exit while pending must clear copy mode enter state, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("exit while pending must clear pending history request, got %#v", runtime.State().History.Pending)
	}
	if len(runner.CancelTokens) != 1 || runner.CancelTokens[0] != copyModeHistoryRequestToken(state.TerminalPaneViewID(state.DefaultPaneID)) {
		t.Fatalf("exit while latest is pending must cancel its request, got %#v", runner.CancelTokens)
	}

	delayed := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-delayed",
		78,
		9,
		[]state.HistoryRow{{Text: "delayed-after-exit", LineID: 40}},
	)
	delayed.PaneID = state.DefaultPaneID
	delayed.ViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: port.RequestID(pending.ID), Window: delayed}}); err != nil {
		t.Fatalf("post delayed latest after exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed latest after exit: %v", err)
	}
	if runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 0 {
		t.Fatalf("delayed matching latest after exit must not mutate history store, got %#v", runtime.State().History)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "delayed-after-exit") {
		t.Fatalf("delayed matching latest after exit must not render into frame, got %#v", last.Lines)
	}
}

func TestCopyModeExitWhileLatestPendingIgnoresDelayedMatchingError(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain page up: %v", err)
	}
	pending := runtime.State().History.Pending
	if pending == nil || pending.Kind != state.HistoryRequestLatest {
		t.Fatalf("expected latest pending request, got %#v", runtime.State().History.Pending)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("exit while pending must clear pending history request, got %#v", runtime.State().History.Pending)
	}

	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: port.RequestID(pending.ID)},
		Err:    errors.New("late history failed"),
	}); err != nil {
		t.Fatalf("post delayed latest error after exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed latest error after exit: %v", err)
	}
	if runtime.State().Surface.Err != "" || runtime.State().Session.LastError != "" {
		t.Fatalf("delayed matching latest error after exit must not surface ui error, state=%#v", runtime.State())
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "late history failed") {
		t.Fatalf("delayed matching latest error after exit must not render error text, got %#v", last.Lines)
	}
}

func TestCopyModeIgnoresDelayedHistoryErrorForSupersededPendingRequest(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	pendingHistory, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID:         2,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
	})
	if err != nil {
		t.Fatalf("begin latest seed: %v", err)
	}
	runtime.state.History = pendingHistory
	runtime.state.CopyMode = state.CopyModeStore{}.BindLatest(
		state.DefaultPaneID,
		state.TerminalPaneViewID(state.DefaultPaneID),
		"term-1",
		2,
		78,
		20,
	)

	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 1},
		Err:    errors.New("superseded history failed"),
	}); err != nil {
		t.Fatalf("post superseded history error: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain superseded history error: %v", err)
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.ID != 2 {
		t.Fatalf("superseded history error must not disturb current pending request, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().Surface.Err != "" || runtime.State().Session.LastError != "" {
		t.Fatalf("superseded history error must not surface ui error, state=%#v", runtime.State())
	}
	if runtime.State().CopyMode.Active || !runtime.State().CopyMode.Entering || runtime.State().CopyMode.RequestID != 2 {
		t.Fatalf("superseded history error must not disturb entering copy binding, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeIgnoresDelayedHistoryWindowForSupersededPendingRequest(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	pendingHistory, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID:         2,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
	})
	if err != nil {
		t.Fatalf("begin latest seed: %v", err)
	}
	runtime.state.History = pendingHistory
	runtime.state.CopyMode = state.CopyModeStore{}.BindLatest(
		state.DefaultPaneID,
		state.TerminalPaneViewID(state.DefaultPaneID),
		"term-1",
		2,
		78,
		20,
	)

	superseded := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-superseded",
		78,
		9,
		[]state.HistoryRow{{Text: "superseded-window", LineID: 41}},
	)
	superseded.PaneID = state.DefaultPaneID
	superseded.ViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 1, Window: superseded},
	}); err != nil {
		t.Fatalf("post superseded history window: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain superseded history window: %v", err)
	}
	if runtime.State().History.Pending == nil || runtime.State().History.Pending.ID != 2 {
		t.Fatalf("superseded history window must not disturb current pending request, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 0 {
		t.Fatalf("superseded history window must not backfill current history store, got %#v", runtime.State().History)
	}
	if runtime.State().CopyMode.Active || !runtime.State().CopyMode.Entering || runtime.State().CopyMode.RequestID != 2 {
		t.Fatalf("superseded history window must not disturb entering copy binding, got %#v", runtime.State().CopyMode)
	}
	last := lastFrame(t, host.Frames())
	if frameContains(last, "superseded-window") {
		t.Fatalf("superseded history window must not render stale rows, got %#v", last.Lines)
	}
}

func TestCopyModeClearsOlderPendingForMatchingStaleHistoryWindow(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	store := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
		SourceLines: []state.HistoryLogicalLine{{Text: "new", LineID: 30}},
		Rows:        []state.HistoryRow{{Text: "new", LineID: 30}},
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 30},
	}
	pending, err := store.BeginOlder(state.HistoryPendingRequest{
		ID:         4,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 30},
		Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
	})
	if err != nil {
		t.Fatalf("begin older seed: %v", err)
	}
	runtime.state.History = pending
	runtime.state.CopyMode = state.CopyModeStore{}.BindLatest(
		state.DefaultPaneID,
		state.TerminalPaneViewID(state.DefaultPaneID),
		"term-1",
		4,
		78,
		20,
	)
	runtime.state.CopyMode.BoundToken = "tok-1"

	stale := historyWindowForApp(
		state.HistoryWindowPrepend,
		"term-1",
		"tok-stale",
		78,
		7,
		[]state.HistoryRow{{Text: "old", LineID: 10}},
	)
	stale.PaneID = state.DefaultPaneID
	stale.ViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 4, Window: stale},
	}); err != nil {
		t.Fatalf("post matching stale older window: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain matching stale older window: %v", err)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("matching stale response must clear pending older request, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().History.OlderRequestState() == state.OlderRequestPending {
		t.Fatalf("older request state must not stay loading after matching stale response")
	}
	if runtime.State().Surface.Err != "" || runtime.State().Session.LastError != "" || !runtime.State().Session.Attached {
		t.Fatalf("matching stale response must not break the live session, state=%#v", runtime.State())
	}
}

func TestCopyModeLatestNetworkErrorReturnsToAttachedLiveSession(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	history, err := (state.HistoryStore{}).BeginLatest(state.HistoryPendingRequest{
		ID: 4, PaneID: state.DefaultPaneID, ViewID: viewID, TerminalID: "term-1", Cols: 80,
	})
	if err != nil {
		t.Fatalf("begin latest: %v", err)
	}
	root := state.Root{
		Shell: state.DefaultShell(), History: history,
		CopyMode: state.CopyModeStore{}.BindLatest(state.DefaultPaneID, viewID, "term-1", 4, 80, 20),
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Channel: 7},
		Surface:  state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"live"}},
	}
	next, _ := reduceCopyModeHistoryResult(root, CopyModeHistoryResultMsg{
		Err: errors.New("network unavailable"), RequestID: 4, ViewID: viewID, TerminalID: "term-1",
	}, CopyModeDeps{})

	if next.CopyMode.Active || next.CopyMode.Entering || next.History.Pending != nil {
		t.Fatalf("latest failure should leave history mode: copy=%#v history=%#v", next.CopyMode, next.History)
	}
	if !next.Session.Attached || next.Session.Channel != 7 || next.Session.LastError != "" || next.Surface.Err != "" {
		t.Fatalf("latest failure broke the live terminal: session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestCopyModeOlderNetworkErrorKeepsLoadedRowsAndAllowsRetry(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	history := state.HistoryStore{
		PaneID: state.DefaultPaneID, ViewID: viewID, TerminalID: "term-1", Token: "tok-1", Cols: 80,
		Boundary: state.HistoryBoundary{FirstLineID: 10, LastLineID: 20},
		Rows:     []state.HistoryRow{{Text: "loaded", LineID: 20}},
	}
	var err error
	history, err = history.BeginOlder(state.HistoryPendingRequest{
		ID: 5, Kind: state.HistoryRequestOlder, PaneID: state.DefaultPaneID, ViewID: viewID,
		TerminalID: "term-1", Token: "tok-1", Cols: 80, Boundary: history.Boundary,
	})
	if err != nil {
		t.Fatalf("begin older: %v", err)
	}
	root := state.Root{
		Shell: state.DefaultShell(), History: history,
		CopyMode: state.CopyModeStore{Active: true, PaneID: state.DefaultPaneID, ViewID: viewID, TerminalID: "term-1", BoundToken: "tok-1"},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Channel: 7},
		Surface:  state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"live"}},
	}
	next, _ := reduceCopyModeHistoryResult(root, CopyModeHistoryResultMsg{
		Err: errors.New("network unavailable"), RequestID: 5, ViewID: viewID, TerminalID: "term-1",
	}, CopyModeDeps{})

	if next.History.Pending != nil || next.History.OlderRequestState() == state.OlderRequestPending {
		t.Fatalf("older failure should clear pending for retry: %#v", next.History.Pending)
	}
	if !next.CopyMode.Active || len(next.History.Rows) != 1 || next.History.Rows[0].Text != "loaded" {
		t.Fatalf("older failure should keep the loaded history window: copy=%#v history=%#v", next.CopyMode, next.History)
	}
	if !next.Session.Attached || next.Session.Channel != 7 || next.Session.LastError != "" || next.Surface.Err != "" {
		t.Fatalf("older failure broke the live terminal: session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestCopyModeInvalidatesTypedStaleHistoryWithoutDiscardingRows(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	store := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
		SourceLines: []state.HistoryLogicalLine{{Text: "new", LineID: 30}},
		Rows:        []state.HistoryRow{{Text: "new", LineID: 30}},
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 30},
	}
	pending, err := store.BeginOlder(state.HistoryPendingRequest{
		ID:         4,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 30},
		Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
	})
	if err != nil {
		t.Fatalf("begin older seed: %v", err)
	}
	runtime.state.History = pending
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   78,
		ViewRows:    8,
		Cursor:      state.CopyPosition{Row: 0},
		ViewportTop: 0,
	}
	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 4},
		Err:    fmt.Errorf("%w: protocol stale resource", port.ErrStaleHistoryWindow),
	}); err != nil {
		t.Fatalf("post protocol stale older error: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain protocol stale older error: %v", err)
	}
	if runtime.State().History.Pending != nil {
		t.Fatalf("protocol stale error must clear pending older request, got %#v", runtime.State().History.Pending)
	}
	if runtime.State().Surface.Err != "" || runtime.State().Session.LastError != "" {
		t.Fatalf("protocol stale error must not surface raw protocol error, state=%#v", runtime.State())
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.BoundToken != "" || runtime.State().History.Token != "" || len(runtime.State().History.Rows) != 1 {
		t.Fatalf("protocol stale error should clear stale state but keep rendered history, state=%#v", runtime.State())
	}
	toasts := runtime.State().Shell.Toasts
	if len(toasts) != 1 || toasts[0].Title != "History window expired" {
		t.Fatalf("protocol stale error should provide one visible recovery notice, got %#v", toasts)
	}
}

func TestCopyModeShowsWindowTooLargeWithoutDiscardingRows(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runner := &recordingEffectRunner{}
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, &testkit.FakeTerminalService{}, runner)

	store := state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
		SourceLines: []state.HistoryLogicalLine{{Text: "visible", LineID: 30}},
		Rows:        []state.HistoryRow{{Text: "visible", LineID: 30}},
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: 30},
	}
	pending, err := store.BeginOlder(state.HistoryPendingRequest{
		ID:         5,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Cols:       78,
		Token:      "tok-1",
		Generation: 7,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 30},
		Boundary:   state.HistoryBoundary{FirstLineID: 20, LastLineID: 30},
	})
	if err != nil {
		t.Fatalf("begin older seed: %v", err)
	}
	runtime.state.History = pending
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   8,
	}
	if err := runtime.Post(CopyModeHistoryResultMsg{
		Result: port.HistoryResult{RequestID: 5},
		Err:    fmt.Errorf("%w: bounded response exceeded", port.ErrHistoryWindowTooLarge),
	}); err != nil {
		t.Fatalf("post history window too large: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain history window too large: %v", err)
	}

	result := runtime.State()
	if result.History.Pending != nil || result.History.Token != "" || result.CopyMode.BoundToken != "" {
		t.Fatalf("window-too-large must clear pending token state: %#v", result)
	}
	if len(result.History.Rows) != 1 || result.History.Rows[0].Text != "visible" {
		t.Fatalf("window-too-large discarded rendered history: %#v", result.History.Rows)
	}
	if result.Surface.Err != "" || result.Session.LastError != "" {
		t.Fatalf("window-too-large surfaced raw protocol error: %#v", result)
	}
	toasts := result.Shell.Toasts
	if len(toasts) != 1 || toasts[0].Title != "History line is too large" {
		t.Fatalf("window-too-large visible notice = %#v", toasts)
	}
}

func TestCopyModeCopyTooLargeKeepsSelectionAndShowsWarning(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntime(host, &testkit.FakeCoreClient{}, nil)
	selection := &state.CopySelection{
		Anchor:        state.CopyPosition{Row: 0, Col: 0},
		Focus:         state.CopyPosition{Row: 0, Col: 7},
		LogicalAnchor: state.CopyLogicalPosition{Valid: true, LineID: 30, Col: 0},
		LogicalFocus:  state.CopyLogicalPosition{Valid: true, LineID: 30, Col: 7},
	}
	runtime.state.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Rows:       []state.HistoryRow{{Text: "visible", LineID: 30}},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:        true,
		TerminalID:    "term-1",
		BoundToken:    "tok-1",
		Selection:     selection,
		CopyPending:   true,
		CopyRequestID: 9,
	}

	if err := runtime.Post(CopyModeCopyResultMsg{
		Err:       fmt.Errorf("%w: bounded copy exceeded", port.ErrHistoryCopyTooLarge),
		Commit:    true,
		RequestID: 9,
	}); err != nil {
		t.Fatalf("post copy-too-large: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy-too-large: %v", err)
	}

	result := runtime.State()
	if result.CopyMode.Selection == nil || result.CopyMode.BoundToken != "tok-1" || result.History.Token != "tok-1" {
		t.Fatalf("copy-too-large discarded retryable selection state: %#v", result)
	}
	if len(result.Clipboard.Entries) != 0 {
		t.Fatalf("copy-too-large wrote partial clipboard data: %#v", result.Clipboard)
	}
	toasts := result.Shell.Toasts
	if len(toasts) != 1 || toasts[0].Title != "Selection is too large" {
		t.Fatalf("copy-too-large visible warning = %#v", toasts)
	}
}

func TestInteractiveRuntimeRoutesTerminalInputAndCopyModeInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "copy-row", LineID: 20}},
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send terminal char: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain terminal input: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "x" {
		t.Fatalf("expected terminal input, got %#v", terminal.Inputs)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send page up: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy mode: %v", err)
	}
	if len(core.LatestRequests) != 1 {
		t.Fatalf("expected history latest request, got %#v", core.LatestRequests)
	}
	if len(terminal.Inputs) != 1 {
		t.Fatalf("copy mode page up should not be terminal input %#v", terminal.Inputs)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("expected copy mode to exit %#v", runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 1 {
		t.Fatalf("copy mode esc should not be terminal input %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeCopyModeSwallowsUnboundRawKeys(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "copy-row", LineID: 20}},
		)}},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy mode: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyF5, RawSeq: "\x1b[15~"}); err != nil {
		t.Fatalf("send copy unbound raw key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy unbound raw key: %v", err)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("copy mode unbound raw key must not leak to terminal, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimePassesRawSpecialKeysAndSwallowsUIModeUnboundKeys(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyLeft, RawSeq: "\x1b[D"},
		{Kind: input.EventKindKey, Key: input.KeyDelete, RawSeq: "\x1b[3~"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x03", Ctrl: true, RawSeq: "\x03"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", Alt: true, RawSeq: "\x1bx"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send raw key %#v: %v", event, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain raw keys: %v", err)
	}
	got := make([]string, 0, len(terminal.Inputs))
	for _, req := range terminal.Inputs {
		got = append(got, string(req.Bytes))
	}
	want := []string{"\x1b[D", "\x1b[3~", "\x03", "\x1bx"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected raw key passthrough got=%q want=%q", got, want)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x10", Ctrl: true}); err != nil {
		t.Fatalf("enter pane mode: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyDelete, RawSeq: "\x1b[3~"}); err != nil {
		t.Fatalf("send ui unbound key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain ui unbound: %v", err)
	}
	if len(terminal.Inputs) != len(want) {
		t.Fatalf("ui mode unbound key must not leak to terminal, got %#v", terminal.Inputs)
	}
}

func TestCopyModeSelectionCopiesAuthoritativeRows(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	core := &testkit.FakeCoreClient{}
	host := NewFakeTerminalHost(4)
	runtime := newCopyModeRuntime(host, core, clipboard)
	runtime.state.History = historyStoreForCopySelection()
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
	}

	if err := runtime.Post(CopyModeSetMarkMsg{Position: state.CopyPosition{Row: 0, Col: 1}}); err != nil {
		t.Fatalf("post mark: %v", err)
	}
	if err := runtime.Post(CopyModeMoveCursorMsg{Position: state.CopyPosition{Row: 1, Col: 2}}); err != nil {
		t.Fatalf("post move: %v", err)
	}
	if err := runtime.Post(CopyModeCopySelectionMsg{}); err != nil {
		t.Fatalf("post copy: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "lpha\nbe" {
		t.Fatalf("unexpected clipboard writes %#v", clipboard.Writes)
	}
	if len(core.CopyRequests) != 0 {
		t.Fatalf("visible selection should be materialized locally, got backend requests %#v", core.CopyRequests)
	}
	last := lastFrame(t, host.Frames())
	assertPaneANSIState(t, last, "lpha", render.ANSICellStyle{FG: "ansi:8", BG: "ansi:3"})
	assertPaneANSIState(t, last, "be", render.ANSICellStyle{FG: "ansi:8", BG: "ansi:3"})
	if len(runtime.State().Shell.Toasts) == 0 || runtime.State().Shell.Toasts[len(runtime.State().Shell.Toasts)-1].Title != "Copied to clipboard" || frameContains(last, "selection yanked") {
		t.Fatalf("copy feedback toast should stay in state without legacy visible text, state=%#v frame=%#v", runtime.State().Shell.Toasts, last.Lines)
	}
}

func TestCopyModeSelectionFetchesTrimmedRangeFromBackend(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	core := &testkit.FakeCoreClient{
		CopyResponses: []port.HistoryCopyRangeResult{{Text: "older\ntrimmed\ncurrent"}},
	}
	host := NewFakeTerminalHost(4)
	runtime := newCopyModeRuntime(host, core, clipboard)
	runtime.state.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 30},
		SourceLines: []state.HistoryLogicalLine{
			{Text: "current", LineID: 30},
		},
		Rows: []state.HistoryRow{
			{Text: "current", LineID: 30},
		},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		Selection: &state.CopySelection{
			Anchor:        state.CopyPosition{Row: 0, Col: 0},
			Focus:         state.CopyPosition{Row: 0, Col: 7},
			LogicalAnchor: state.CopyLogicalPosition{Valid: true, LineID: 10, Col: 0},
			LogicalFocus:  state.CopyLogicalPosition{Valid: true, LineID: 30, Col: 7},
		},
	}

	if err := runtime.Post(CopyModeCopySelectionMsg{}); err != nil {
		t.Fatalf("post copy: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(core.CopyRequests) != 1 {
		t.Fatalf("expected backend copy range request, got %#v", core.CopyRequests)
	}
	req := core.CopyRequests[0]
	if req.Token != "tok-1" || req.Generation != 7 || req.Boundary.FirstLineID != 10 || req.Boundary.LastLineID != 30 || req.Start.LineID != 10 || req.End.LineID != 30 {
		t.Fatalf("unexpected backend copy request %#v", req)
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "older\ntrimmed\ncurrent" {
		t.Fatalf("unexpected clipboard writes %#v", clipboard.Writes)
	}
	if len(runtime.State().Clipboard.Entries) == 0 || runtime.State().Clipboard.Entries[0].Text != "older\ntrimmed\ncurrent" {
		t.Fatalf("copy should update reducer clipboard from backend text, got %#v", runtime.State().Clipboard)
	}
}

func TestCopyModeCanonicalKeysMoveSelectAndCopy(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	core := &testkit.FakeCoreClient{}
	host := NewFakeTerminalHost(16)
	runtime := newCopyModeRuntime(host, core, clipboard)
	runtime.state.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		Rows: []state.HistoryRow{
			{Text: "alpha", LineID: 10},
			{Text: "beta", LineID: 11},
			{Text: "gamma", LineID: 12},
		},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   3,
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "j"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "j"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "u"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "g"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "j"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "j"},
		{Kind: input.EventKindKey, Key: input.KeyHome},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: " "},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"},
		{Kind: input.EventKindKey, Key: input.KeyEnd},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "y"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send copy key %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain copy key %#v: %v", event, err)
		}
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "gamma" {
		t.Fatalf("expected y to copy selected authoritative row, got %#v", clipboard.Writes)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 2, Col: 5}) {
		t.Fatalf("expected copy cursor at end of gamma, got %#v", runtime.State().CopyMode.Cursor)
	}
	if runtime.State().CopyMode.Query != "" {
		t.Fatalf("u/d should scroll copy viewport instead of editing search query, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeEnterCopiesAndExits(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	core := &testkit.FakeCoreClient{}
	host := NewFakeTerminalHost(16)
	runtime := newCopyModeRuntime(host, core, clipboard)
	runtime.state.History = historyStoreForCopySelection()
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   4,
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: " "},
		{Kind: input.EventKindKey, Key: input.KeyEnd},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send copy key %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain copy key %#v: %v", event, err)
		}
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "alpha" {
		t.Fatalf("expected enter to copy selected authoritative row, got %#v", clipboard.Writes)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("enter copy should exit copy mode, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeEnterReleasesHistoryOnlyAfterCopySucceeds(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	core := &testkit.FakeCoreClient{}
	runner := &recordingEffectRunner{}
	host := NewFakeTerminalHost(16)
	runtime := newCopyModeRuntimeWithRunner(host, core, clipboard, &testkit.FakeTerminalService{}, runner)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner.Effects = nil
	runtime.state.History = historyStoreForCopySelection()
	mark := state.CopyPosition{Row: 0, Col: 0}
	runtime.state.CopyMode = state.CopyModeStore{
		Active: true, ViewID: state.TerminalPaneViewID(state.DefaultPaneID), TerminalID: "term-1",
		BoundToken: "tok-1", BoundCols: 78, ViewRows: 4, Mark: &mark,
		Selection: &state.CopySelection{
			Anchor: mark, Focus: state.CopyPosition{Row: 0, Col: 5},
			LogicalAnchor: state.CopyLogicalPosition{Valid: true, LineID: 10},
			LogicalFocus:  state.CopyLogicalPosition{Valid: true, LineID: 10, Col: 5},
		},
	}

	for index := 0; index < 2; index++ {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
			t.Fatal(err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if !runtime.State().CopyMode.Active || !runtime.State().CopyMode.CopyPending || runtime.State().History.Token != "tok-1" {
		t.Fatalf("enter released frozen history before copy completed: %#v", runtime.State())
	}
	toasts := runtime.State().Shell.Toasts
	if len(toasts) == 0 || toasts[len(toasts)-1].Title != "Copy in progress" {
		t.Fatalf("repeated enter must provide pending feedback, got %#v", toasts)
	}
	if len(runner.Effects) != 2 || len(core.ReleaseRequests) != 0 {
		t.Fatalf("pending enter should start clipboard sync and local persistence without release, effects=%#v releases=%#v", runner.Effects, core.ReleaseRequests)
	}
	copyEffect, ok := runner.Effects[0].(FuncEffect)
	if !ok || copyEffect.Token != copyModeCopyRequestToken(state.TerminalPaneViewID(state.DefaultPaneID)) {
		t.Fatalf("copy effect missing per-view cancellation token: %#v", runner.Effects[0])
	}
	result := copyEffect.Run(context.Background())
	if err := runtime.Post(result); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Active || runtime.State().History.Token != "" {
		t.Fatalf("successful enter copy did not exit: %#v", runtime.State())
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "alpha" {
		t.Fatalf("copy did not reach clipboard: %#v", clipboard.Writes)
	}
	if len(runner.Effects) != 3 {
		t.Fatalf("copy should schedule persistence immediately and release after clipboard completion, effects=%#v", runner.Effects)
	}
}

func TestCopyModePasteLastCopyExitsAndTargetsActiveTerminal(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{LastCopied: "hello\nworld"}
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, clipboard, terminal, NewSyncEffectRunner())
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   4,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"}); err != nil {
		t.Fatalf("send p: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain p: %v", err)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("paste last copy should exit copy mode, got %#v", runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "hello\nworld" {
		t.Fatalf("paste last copy should target active terminal, got %#v", terminal.Inputs)
	}
}

func TestClipboardPasteOutsideCopyModeKeepsFrozenHistoryState(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, clipboard, terminal, NewSyncEffectRunner())
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	runtime.state.History = state.HistoryStore{
		TerminalID: "term-1", Token: "tok-1", Cols: 78,
		Rows: []state.HistoryRow{{Text: "frozen", LineID: 10}},
	}
	before := runtime.state.History
	if err := runtime.Post(ClipboardPasteTextMsg{Text: "paste me"}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runtime.State().History, before) {
		t.Fatalf("normal paste changed frozen history: before=%#v after=%#v", before, runtime.State().History)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "paste me" {
		t.Fatalf("normal paste did not reach terminal: %#v", terminal.Inputs)
	}
}

func TestCopyModePasteLastCopyFallsBackToClipboardHistory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, nil, terminal, NewSyncEffectRunner())
	runtime.state.Clipboard = state.ClipboardStore{}.WithCopiedText("from-history")
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   4,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"}); err != nil {
		t.Fatalf("send p: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain p: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "from-history" {
		t.Fatalf("paste last copy should fallback to reducer clipboard history, got %#v", terminal.Inputs)
	}
}

func TestCopyModePasteClipboardUsesSystemClipboardAndBracketedPaste(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{ReadResult: port.ClipboardReadResult{Text: "clip-text"}}
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, clipboard, terminal, NewSyncEffectRunner())
	runtime.state.Surface = runtime.state.Surface.Attach("term-1", 80, 24)
	runtime.state.Surface.Modes = state.LiveTerminalModes{BracketedPaste: true}
	runtime.state.Surface.Surfaces = map[string]state.LiveSurfaceSnapshot{
		"term-1": {TerminalID: "term-1", Modes: state.LiveTerminalModes{BracketedPaste: true}, State: state.TerminalLiveAttached},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   4,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "P"}); err != nil {
		t.Fatalf("send P: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain P: %v", err)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("paste clipboard should exit copy mode, got %#v", runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b[200~clip-text\x1b[201~" {
		t.Fatalf("paste clipboard should send bracketed paste bytes, got %#v", terminal.Inputs)
	}
}

func TestCopyModeEmptySystemClipboardKeepsHistoryAndLiveAttachment(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(8)
	runtime := newCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, clipboard, terminal, NewSyncEffectRunner())
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	runtime.state.History = state.HistoryStore{TerminalID: "term-1", Token: "tok-1", Cols: 78, Rows: []state.HistoryRow{{Text: "frozen", LineID: 1}}}
	runtime.state.CopyMode = state.CopyModeStore{
		Active: true, ViewID: state.TerminalPaneViewID(state.DefaultPaneID), TerminalID: "term-1", BoundToken: "tok-1", BoundCols: 78,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "P"}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	result := runtime.State()
	if !result.CopyMode.Active || result.History.Token != "tok-1" {
		t.Fatalf("empty clipboard discarded copy history: %#v", result)
	}
	if !result.Session.Attached || result.Session.State == state.TerminalLiveError || result.Session.LastError != "" {
		t.Fatalf("empty clipboard poisoned live session: %#v", result.Session)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("empty clipboard sent terminal input: %#v", terminal.Inputs)
	}
}

func TestCopyModeClipboardHistoryOverlayFiltersAndPastesSelectedEntry(t *testing.T) {
	clipboard := &testkit.FakeClipboardService{}
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, &testkit.FakeCoreClient{}, clipboard, terminal, NewSyncEffectRunner())
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   4,
	}
	runtime.state.Clipboard = state.ClipboardStore{
		Entries: []state.ClipboardEntry{
			{ID: "clip:1", Title: "alpha", Text: "alpha", Preview: "alpha"},
			{ID: "clip:2", Title: "build log", Text: "build\nlog", Preview: "build …"},
		},
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "H"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "b"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "u"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "i"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "d"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send clipboard history key %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain clipboard history key %#v: %v", event, err)
		}
	}
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("clipboard history paste should close overlay, got %#v", runtime.State().Shell.Overlay)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("clipboard history paste should exit copy mode, got %#v", runtime.State().CopyMode)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "build\nlog" {
		t.Fatalf("clipboard history paste should target selected entry, got %#v", terminal.Inputs)
	}
}

func TestCopyModeSearchReturnsToLiveAtFrozenBottom(t *testing.T) {
	rows := []state.HistoryRow{
		{Text: "alpha", LineID: 10},
		{Text: "beta", LineID: 11},
		{Text: "gamma beta", LineID: 12},
		{Text: "delta", LineID: 13},
		{Text: "epsilon", LineID: 14},
		{Text: "zeta", LineID: 15},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			rows,
		)}},
		SearchResponses: []port.HistorySearchResult{
			{Found: true, Start: state.CopyLogicalPosition{Valid: true, LineID: 11}, End: state.CopyLogicalPosition{Valid: true, LineID: 11, Col: 4}, Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 78, 7, rows)},
			{Found: true, Start: state.CopyLogicalPosition{Valid: true, LineID: 12, Col: 6}, End: state.CopyLogicalPosition{Valid: true, LineID: 12, Col: 10}, Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 78, 7, rows)},
		},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "/"}); err != nil {
		t.Fatalf("start search: %v", err)
	}
	for _, ch := range []string{"b", "e", "t", "a"} {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: ch}); err != nil {
			t.Fatalf("send query %q: %v", ch, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain query: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("submit search: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain search: %v", err)
	}
	if runtime.State().CopyMode.Query != "beta" || len(runtime.State().CopyMode.Matches) != 1 || runtime.State().CopyMode.Cursor.Row != 1 {
		t.Fatalf("expected search matches and cursor on first beta, got %#v", runtime.State().CopyMode)
	}
	queryFrame := lastFrame(t, host.Frames())
	if status := activeCopyContentStatus(runtime); !strings.Contains(status, `search:"beta"`) || !strings.Contains(status, "older:top") {
		t.Fatalf("expected search status outside history body, got %q", status)
	}
	for _, line := range activeCopyContentLines(runtime) {
		if strings.Contains(line, "⌕ search beta") || strings.Contains(line, "SCROLL") {
			t.Fatalf("copy history body should not render search/status rows, got %#v", activeCopyContentLines(runtime))
		}
	}
	assertPaneVisualState(t, queryFrame, "beta", render.StyleWarning)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "n"}); err != nil {
		t.Fatalf("send next match: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain next match: %v", err)
	}
	if runtime.State().CopyMode.ActiveMatch != 0 || runtime.State().CopyMode.Cursor.Row != 2 {
		t.Fatalf("expected next search match, got %#v", runtime.State().CopyMode)
	}
	runtime.state.CopyMode.Cursor = state.CopyPosition{Row: 5}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageDn}); err != nil {
		t.Fatalf("send page down: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain page down: %v", err)
	}
	if runtime.State().CopyMode.Active || runtime.State().History.Token != "" {
		t.Fatalf("page down at the frozen bottom should resume live mode, copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if len(core.ReleaseRequests) != 1 || core.ReleaseRequests[0].Token != "tok-1" {
		t.Fatalf("resuming live should release the frozen token once, got %#v", core.ReleaseRequests)
	}
}

func TestCopyModeSelectionStopsAtFrozenBottomWithoutExiting(t *testing.T) {
	mark := state.CopyPosition{Row: 0}
	root := state.Root{
		History: state.HistoryStore{
			TerminalID: "term-1", Token: "tok-1", Cols: 80,
			Rows:     []state.HistoryRow{{Text: "first", LineID: 1}, {Text: "last", LineID: 2}},
			Boundary: state.HistoryBoundary{FirstLineID: 1, LastLineID: 2},
		},
		CopyMode: state.CopyModeStore{
			Active: true, TerminalID: "term-1", BoundToken: "tok-1", ViewRows: 2,
			Cursor: state.CopyPosition{Row: 1}, Mark: &mark,
			Selection: &state.CopySelection{Anchor: mark, Focus: state.CopyPosition{Row: 1}},
		},
	}
	next, effects := reduceCopyModeScrollNewer(root, CopyModeDeps{Core: &testkit.FakeCoreClient{}}, 1)
	if !next.CopyMode.Active || next.CopyMode.Selection == nil || next.History.Token != "tok-1" {
		t.Fatalf("selection was discarded at frozen bottom: %#v", next)
	}
	if len(effects) != 0 {
		t.Fatalf("selection at frozen bottom unexpectedly released history: %#v", effects)
	}
}

func TestCopyModeSearchResultReturnsToRequestingView(t *testing.T) {
	viewA := state.TerminalPaneViewID("pane-a")
	viewB := state.TerminalPaneViewID("pane-b")
	historyA := state.HistoryStore{
		PaneID: "pane-a", ViewID: viewA, TerminalID: "term-a", Token: "tok-a", Cols: 80, Generation: 3,
		Rows: []state.HistoryRow{{Text: "old-a", LineID: 1}},
	}
	copyA := state.CopyModeStore{
		Active: true, PaneID: "pane-a", ViewID: viewA, TerminalID: "term-a", BoundToken: "tok-a",
		Query: "needle", SearchPending: true, SearchRequestID: 7,
	}
	historyB := state.HistoryStore{
		PaneID: "pane-b", ViewID: viewB, TerminalID: "term-b", Token: "tok-b", Cols: 80, Generation: 4,
		Rows: []state.HistoryRow{{Text: "visible-b", LineID: 1}},
	}
	copyB := state.CopyModeStore{Active: true, PaneID: "pane-b", ViewID: viewB, TerminalID: "term-b", BoundToken: "tok-b"}
	root := state.Root{}
	root = root.WithCopyHistorySession(viewA, historyA, copyA)
	root = root.WithCopyHistorySession(viewB, historyB, copyB)
	window := historyWindowForApp(state.HistoryWindowReplace, "term-a", "tok-a", 80, 3, []state.HistoryRow{{Text: "needle", LineID: 1}})
	window.ViewID = viewA
	window.PaneID = "pane-a"
	next, _ := NewCopyModeReducer(CopyModeDeps{})(root, CopyModeSearchResultMsg{
		RequestID: 7,
		Result: port.HistorySearchResult{
			RequestID: 7, Found: true,
			Start:  state.CopyLogicalPosition{Valid: true, LineID: 1},
			End:    state.CopyLogicalPosition{Valid: true, LineID: 1, Col: 6},
			Window: window,
		},
		Query: "needle", ViewID: viewA, PaneID: "pane-a", TerminalID: "term-a", Token: "tok-a",
	})

	updatedA := next.CopyModeByView[viewA]
	if updatedA.SearchPending || len(updatedA.Matches) != 1 || updatedA.Cursor.Row != 0 {
		t.Fatalf("search result did not update requesting view: %#v", updatedA)
	}
	unchangedB := next.CopyModeByView[viewB]
	if unchangedB.TerminalID != "term-b" || unchangedB.BoundToken != "tok-b" || len(unchangedB.Matches) != 0 {
		t.Fatalf("search result polluted active view: %#v", unchangedB)
	}
}

func TestCopyModeSearchErrorClearsPendingWithoutBreakingLiveSession(t *testing.T) {
	viewID := state.TerminalPaneViewID("pane-a")
	root := state.Root{
		Shell:   state.DefaultShell(),
		Session: state.TerminalSessionStore{TerminalID: "term-a", Attached: true, Channel: 9},
		Surface: state.TerminalSurfaceStore{TerminalID: "term-a", Lines: []string{"live"}},
	}
	root = root.WithCopyHistorySession(viewID,
		state.HistoryStore{PaneID: "pane-a", ViewID: viewID, TerminalID: "term-a", Token: "tok-a"},
		state.CopyModeStore{
			Active: true, PaneID: "pane-a", ViewID: viewID, TerminalID: "term-a", BoundToken: "tok-a",
			Query: "needle", SearchPending: true, SearchRequestID: 8,
		},
	)
	next, _ := NewCopyModeReducer(CopyModeDeps{})(root, CopyModeSearchResultMsg{
		Err: errors.New("remote unavailable"), RequestID: 8, Query: "needle", ViewID: viewID,
		PaneID: "pane-a", TerminalID: "term-a", Token: "tok-a",
	})

	updated := next.CopyModeByView[viewID]
	if updated.SearchPending || !updated.Active {
		t.Fatalf("search error should clear only the pending operation: %#v", updated)
	}
	if !next.Session.Attached || next.Session.Channel != 9 || next.Session.LastError != "" || next.Surface.Err != "" {
		t.Fatalf("search error broke the live terminal session: session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestClosingPaneCancelsCopyWorkAndReleasesFrozenHistory(t *testing.T) {
	core := &testkit.FakeCoreClient{}
	host := NewFakeTerminalHost(16)
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, &testkit.FakeClipboardService{}, &testkit.FakeTerminalService{}, NewSyncEffectRunner())
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initialize runtime: %v", err)
	}
	viewID := state.TerminalPaneViewID("pane-2")
	runtime.state.Shell = runtime.state.Shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(state.NewPaneTerminalView(
		"pane-2", "term-1", 8, 40, 20, state.TerminalResizeRoleFollower, "surface", viewID, false,
	))
	runtime.state = runtime.state.WithCopyHistorySession(viewID,
		state.HistoryStore{
			PaneID: "pane-2", ViewID: viewID, TerminalID: "term-1", Token: "tok-close", Cols: 40,
			Pending: &state.HistoryPendingRequest{ID: 11, ViewID: viewID, TerminalID: "term-1"},
		},
		state.CopyModeStore{
			Active: true, PaneID: "pane-2", ViewID: viewID, TerminalID: "term-1", BoundToken: "tok-close",
			SearchPending: true, CopyPending: true,
		},
	)
	cleanup := copyHistoryCleanupEffectsForView(runtime.state, viewID)
	cancelled := map[CancelToken]bool{}
	for _, effect := range cleanup {
		if cancel, ok := effect.(CancelEffect); ok {
			cancelled[cancel.Token] = true
		}
	}
	for _, token := range []CancelToken{copyModeHistoryRequestToken(viewID), copyModeSearchRequestToken(viewID), copyModeCopyRequestToken(viewID)} {
		if !cancelled[token] {
			t.Fatalf("missing cleanup cancellation %q in %#v", token, cleanup)
		}
	}
	command := state.WorkbenchCommand{Action: state.WorkbenchCommandPaneClose, Target: state.PaneCommandTarget{PaneID: "pane-2"}}
	_, closeEffects := reduceWorkbenchCommand(runtime.state, command)
	foundRelease := false
	for _, effect := range closeEffects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil {
			continue
		}
		if release, ok := fn.Run(context.Background()).(CopyModeReleaseHistoryMsg); ok && release.Token == "tok-close" {
			foundRelease = true
		}
	}
	if !foundRelease {
		t.Fatalf("pane close did not schedule frozen history release: %#v", closeEffects)
	}

	if err := runtime.Post(ShellWorkbenchCommandMsg{Command: command}); err != nil {
		t.Fatalf("post pane close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane close: %v", err)
	}
	if _, copyMode := runtime.State().CopyHistorySessionForView(viewID); copyMode.Active || copyMode.BoundToken != "" {
		t.Fatalf("closed pane retained copy history session: %#v", copyMode)
	}
	if len(core.ReleaseRequests) != 1 || core.ReleaseRequests[0].TerminalID != "term-1" || core.ReleaseRequests[0].Token != "tok-close" {
		t.Fatalf("closed pane did not release its frozen history: %#v", core.ReleaseRequests)
	}
}

func TestCopyModeLatestShowsCodexLiveTailFrameHead(t *testing.T) {
	rows := []state.HistoryRow{
		{Text: "lozzow@RedmiBook: ~/Documents/workdir/anytty", LineID: 1},
		{Text: "ζ", LineID: 2},
		{Text: "lozzow@RedmiBook: ~/Documents/workdir/anytty", LineID: 3},
		{Text: "ζ codex --yolo", LineID: 4},
		{Text: "Update available! 0.141.0 -> 0.142.0", LineID: 10, LiveTail: true},
		{Text: "Run brew upgrade --cask codex to update.", LineID: 11, LiveTail: true},
		{Text: "See full release notes:", LineID: 12, LiveTail: true},
		{Text: "https://github.com/openai/codex/releases/latest", LineID: 13, LiveTail: true},
		{Text: "OpenAI Codex (v0.141.0)", LineID: 14, LiveTail: true},
		{Text: "model: gpt-5.5 xhigh", LineID: 15, LiveTail: true},
		{Text: "directory: ~/Documents/workdir/anytty", LineID: 16, LiveTail: true},
		{Text: "permissions: YOLO mode", LineID: 17, LiveTail: true},
		{Text: "Tip: Visit the Codex community forum", LineID: 18, LiveTail: true},
		{Text: "> Explain this codebase", LineID: 19, LiveTail: true},
		{Text: "gpt-5.5 xhigh · ~/Documents/workdir/anytty", LineID: 20, LiveTail: true},
	}
	latest := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: latest}}}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if runtime.State().CopyMode.ViewportTop != 11 || runtime.State().CopyMode.Cursor.Row != 14 {
		t.Fatalf("copy mode should start at Codex live-tail newest rows, got %#v", runtime.State().CopyMode)
	}
	lines := activeCopyContentLines(runtime)
	if !copyHistoryLinesContain(lines, "> Explain this codebase") || !copyHistoryLinesContain(lines, "gpt-5.5 xhigh") {
		t.Fatalf("copy history should render Codex newest tail after entry, got %#v", lines)
	}
}

func TestCopyModeLatestKeepsFrozenViewportLogicalAnchor(t *testing.T) {
	rows := []state.HistoryRow{
		{Text: "old shell prompt", LineID: 1, Segment: state.HistoryCursorSegmentCommitted},
		{Text: "old shell marker", LineID: 2, Segment: state.HistoryCursorSegmentCommitted},
		{Text: "visible shell prompt", LineID: 3, Segment: state.HistoryCursorSegmentCommitted},
		{Text: "visible shell marker", LineID: 4, Segment: state.HistoryCursorSegmentCommitted},
		{Text: "OpenAI Codex", LineID: 20, Segment: state.HistoryCursorSegmentCurrentPrimaryFrame},
		{Text: "> Explain this codebase", LineID: 21, Segment: state.HistoryCursorSegmentCurrentPrimaryFrame},
	}
	latest := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 80, 7, rows)
	latest.ViewportAnchor = state.HistoryViewportAnchor{TopLineID: 3, ScreenCols: 80, ScreenRows: 16, Valid: true}
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: latest}}}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 16)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if runtime.State().CopyMode.ViewportTop != 2 || runtime.State().CopyMode.Cursor.Row != 5 {
		t.Fatalf("copy latest should keep the frozen logical viewport anchor, got %#v", runtime.State().CopyMode)
	}
	lines := activeCopyContentLines(runtime)
	if copyHistoryLinesContain(lines, "old shell prompt") || !copyHistoryLinesContain(lines, "visible shell prompt") || !copyHistoryLinesContain(lines, "OpenAI Codex") {
		t.Fatalf("copy history should start at the frozen viewport top, got %#v", lines)
	}
}

func TestCopyModeSearchMatchesAcrossReflowRows(t *testing.T) {
	searchWindow := state.HistoryWindow{
		TerminalID: "term-1",
		Token:      "tok-1",
		Op:         state.HistoryWindowReplace,
		Cols:       8,
		SourceLines: []state.HistoryLogicalLine{{
			Text:   "alphabetagamma",
			Cells:  []state.HistoryCell{{Text: "alphabetagamma", Width: 14}},
			LineID: 10,
		}},
		Rows: []state.HistoryRow{
			{Text: "alphabe", LineID: 10, RowInLine: 0},
			{Text: "tagamma", LineID: 10, RowInLine: 1},
		},
		Lines:      []state.HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 1}},
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 10},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: searchWindow}},
		SearchResponses: []port.HistorySearchResult{{
			Found:  true,
			Start:  state.CopyLogicalPosition{Valid: true, LineID: 10, Col: 5},
			End:    state.CopyLogicalPosition{Valid: true, LineID: 10, Col: 9},
			Window: searchWindow,
		}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(10, 12)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "/"}); err != nil {
		t.Fatalf("start search: %v", err)
	}
	for _, ch := range []string{"b", "e", "t", "a"} {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: ch}); err != nil {
			t.Fatalf("send query %q: %v", ch, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain query: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("submit search: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain search: %v", err)
	}
	if runtime.State().CopyMode.Query != "beta" || len(runtime.State().CopyMode.Matches) != 1 {
		t.Fatalf("expected one cross-row search match, got %#v", runtime.State().CopyMode)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 0, Col: 5}) {
		t.Fatalf("expected cursor on cross-row match start, got %#v", runtime.State().CopyMode.Cursor)
	}
}

func TestCopyModeLocalReflowResizeKeepsSelectionOnOriginalContent(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: state.HistoryWindow{
			TerminalID: "term-1",
			Token:      "tok-1",
			Op:         state.HistoryWindowReplace,
			Cols:       3,
			SourceLines: []state.HistoryLogicalLine{{
				Text:   "abcdef",
				Cells:  []state.HistoryCell{{Text: "abcdef", Width: 6}},
				LineID: 10,
			}},
			Rows: []state.HistoryRow{
				{Text: "abc", LineID: 10, RowInLine: 0},
				{Text: "def", LineID: 10, RowInLine: 1},
			},
			Lines:      []state.HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 1}},
			Generation: 7,
			Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 10},
		}}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(5, 12)
	runtime := newCopyModeRuntime(host, core, nil)
	binding, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 3
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)
	runtime.state.Surface = state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"live"}}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	runtime.state.CopyMode = runtime.state.CopyMode.SetMark(state.CopyPosition{Row: 0, Col: 1})
	runtime.state.CopyMode = runtime.state.CopyMode.MoveCursor(state.CopyPosition{Row: 1, Col: 2})

	if got := SelectedText(runtime.State().History, runtime.State().CopyMode); got != "bcde" {
		t.Fatalf("expected initial selected text before local reflow, got %q", got)
	}
	binding, _ = runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 6
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)

	if err := host.SendResize(8, 12); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain local reflow resize: %v", err)
	}

	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"abcdef"}) {
		t.Fatalf("expected local reflow to widen into one row, got %v", got)
	}
	if runtime.State().CopyMode.Selection == nil {
		t.Fatalf("expected selection after local reflow, got %#v", runtime.State().CopyMode)
	}
	if got := SelectedText(runtime.State().History, runtime.State().CopyMode); got != "bcde" {
		t.Fatalf("expected selected text to stay on original content after local reflow, got %q", got)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 0, Col: 5}) {
		t.Fatalf("expected cursor to rebind to widened row, got %#v", runtime.State().CopyMode.Cursor)
	}
}

func TestCopyModeOlderPrependKeepsCurrentSearchMatch(t *testing.T) {
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{
			{Text: "beta-one", LineID: 20},
			{Text: "beta-two", LineID: 21},
		},
	)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 20}
	latest.HasMore = true
	older := historyWindowForApp(
		state.HistoryWindowPrepend,
		"term-1",
		"tok-1",
		78,
		7,
		[]state.HistoryRow{{Text: "older-prefix", LineID: 10}},
	)
	older.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	older.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 21}
	older.HasMore = true
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
		OlderResponses:  []port.HistoryResult{{Window: older}},
		SearchResponses: []port.HistorySearchResult{{
			Found:  true,
			Start:  state.CopyLogicalPosition{Valid: true, LineID: 21},
			End:    state.CopyLogicalPosition{Valid: true, LineID: 21, Col: 4},
			Window: latest,
		}},
	}
	host := NewFakeTerminalHost(16)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "/"}); err != nil {
		t.Fatalf("start search: %v", err)
	}
	for _, ch := range []string{"b", "e", "t", "a"} {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: ch}); err != nil {
			t.Fatalf("send query %q: %v", ch, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain query: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("submit search: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain search: %v", err)
	}
	if runtime.State().CopyMode.ActiveMatch != 0 || runtime.State().CopyMode.Cursor.Row != 1 {
		t.Fatalf("expected daemon search match before prepend, got %#v", runtime.State().CopyMode)
	}

	beginPendingOlderForTest(&runtime.state, 2, 0)
	core.OlderRequests = append(core.OlderRequests, port.HistoryOlderRequest{RequestID: 2})
	if err := postHistoryResultForTest(runtime, 2, older); err != nil {
		t.Fatalf("post older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}

	if runtime.State().CopyMode.ActiveMatch != 0 || runtime.State().CopyMode.Cursor.Row != 2 {
		t.Fatalf("older prepend should keep active search match on original content, got %#v", runtime.State().CopyMode)
	}
	if status := activeCopyContentStatus(runtime); !strings.Contains(status, `search:"beta"`) {
		t.Fatalf("expected search status to stay on second match after prepend, got %q", status)
	}
}

func TestCopyModeMouseSelectionUsesHistoryDisplayColumns(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{
				Text:   "a好bc",
				LineID: 10,
				Cells: []state.HistoryCell{
					{Text: "a", Width: 1},
					{Text: "好", Width: 2},
					{Text: "bc", Width: 2},
				},
			}},
		)}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 8)
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	var target render.HitRegion
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionHistoryRow && region.Row == 0 {
			target = region
			break
		}
	}
	if target.Kind == "" || target.Rect.W != 5 {
		t.Fatalf("expected text-only history row region with display width, got %#v", frame.HitRegions)
	}

	if err := host.SendInput(input.InputEvent{
		Kind:  input.EventKindMouse,
		Mouse: input.MouseLeft,
		Row:   target.Rect.Y + 1,
		Col:   target.Rect.X + 1 + 2,
	}); err != nil {
		t.Fatalf("send row click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain row click: %v", err)
	}
	if got := runtime.State().CopyMode.Cursor; got != (state.CopyPosition{Row: 0, Col: 2}) {
		t.Fatalf("mouse selection should use display columns, got %#v", got)
	}
	if runtime.State().CopyMode.Mark == nil || *runtime.State().CopyMode.Mark != (state.CopyPosition{Row: 0, Col: 2}) {
		t.Fatalf("mouse selection should set mark at display column, got %#v", runtime.State().CopyMode.Mark)
	}
}

func TestCopyModeHistoryTextClickFocusesInactivePaneBeforeSelecting(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			36,
			7,
			[]state.HistoryRow{
				{Text: "inactive history one", LineID: 10},
				{Text: "inactive history two", LineID: 11},
			},
		)}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(42, 10)
	runtime := newInteractiveCopyModeRuntimeWithRunner(host, core, nil, &testkit.FakeTerminalService{}, NewSyncEffectRunner())

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy enter: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneEmpty},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != "pane-2" {
		t.Fatalf("split should activate pane-2, got %#v", runtime.State().Shell)
	}
	before := runtime.State().CopyMode.Cursor
	frame := lastFrame(t, host.Frames())
	target := copyHistoryRowHitRegionForPane(t, frame, state.DefaultPaneID, 1)

	if err := host.SendInput(mouseEventAt(target.Rect)); err != nil {
		t.Fatalf("send inactive history click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain inactive history click: %v", err)
	}
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID {
		t.Fatalf("history text click should focus bound pane first, got %#v", runtime.State().Shell)
	}
	if got := runtime.State().CopyMode.Cursor; got != before {
		t.Fatalf("inactive history click must not select before focus got=%#v want=%#v", got, before)
	}
	if runtime.State().CopyMode.Mark != nil {
		t.Fatalf("inactive history click must not set mark before focus, got %#v", runtime.State().CopyMode.Mark)
	}

	frame = lastFrame(t, host.Frames())
	target = copyHistoryRowHitRegionForPane(t, frame, state.DefaultPaneID, 1)
	if err := host.SendInput(mouseEventAt(target.Rect)); err != nil {
		t.Fatalf("send active history click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain active history click: %v", err)
	}
	if runtime.State().CopyMode.Cursor.Row != 1 || runtime.State().CopyMode.Mark == nil || runtime.State().CopyMode.Mark.Row != 1 {
		t.Fatalf("focused history click should select row, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeOlderPrependKeepsCursorAndSelectionOnOriginalContent(t *testing.T) {
	host := NewFakeTerminalHost(8)
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{
			Window: historyWindowForApp(
				state.HistoryWindowReplace,
				"term-1",
				"tok-latest",
				78,
				7,
				[]state.HistoryRow{
					{Text: "line-1", LineID: 10},
					{Text: "line-2", LineID: 11},
					{Text: "line-3", LineID: 12},
					{Text: "line-4", LineID: 13},
				},
			),
		}},
		OlderResponses: []port.HistoryResult{{
			Window: historyWindowForApp(
				state.HistoryWindowPrepend,
				"term-1",
				"tok-latest",
				78,
				7,
				[]state.HistoryRow{
					{Text: "older-1", LineID: 8},
					{Text: "older-2", LineID: 9},
				},
			),
		}},
	}
	core.LatestResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 11}
	core.OlderResponses[0].Window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 9}
	core.OlderResponses[0].Window.Boundary = state.HistoryBoundary{FirstLineID: 8, LastLineID: 13}
	runtime := newCopyModeRuntime(host, core, nil)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}

	if err := runtime.Post(CopyModeSetMarkMsg{Position: state.CopyPosition{Row: 1, Col: 1}}); err != nil {
		t.Fatalf("set mark: %v", err)
	}
	if err := runtime.Post(CopyModeMoveCursorMsg{Position: state.CopyPosition{Row: 3, Col: 2}}); err != nil {
		t.Fatalf("move cursor: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain selection setup: %v", err)
	}

	beginPendingOlderForTest(&runtime.state, 2, 0)
	core.OlderRequests = append(core.OlderRequests, port.HistoryOlderRequest{RequestID: 2})
	if err := postHistoryResultForTest(runtime, 2, core.OlderResponses[0].Window); err != nil {
		t.Fatalf("post older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("expected one older request, got %#v", core.OlderRequests)
	}
	if got := historyRowTexts(runtime.State().History.Rows); !reflect.DeepEqual(got, []string{"older-1", "older-2", "line-1", "line-2", "line-3", "line-4"}) {
		t.Fatalf("expected older rows to prepend before original content, got %v", got)
	}

	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 5, Col: 2}) {
		t.Fatalf("older prepend must keep cursor on original content, got %#v", runtime.State().CopyMode.Cursor)
	}
	if runtime.State().CopyMode.Mark == nil || *runtime.State().CopyMode.Mark != (state.CopyPosition{Row: 3, Col: 1}) {
		t.Fatalf("older prepend must shift mark to original content row, got %#v", runtime.State().CopyMode.Mark)
	}
	if runtime.State().CopyMode.Selection == nil {
		t.Fatal("older prepend must preserve selection")
	}
	if runtime.State().CopyMode.Selection.Anchor != (state.CopyPosition{Row: 3, Col: 1}) || runtime.State().CopyMode.Selection.Focus != (state.CopyPosition{Row: 5, Col: 2}) {
		t.Fatalf("older prepend must shift selection rows with inserted history, got %#v", runtime.State().CopyMode.Selection)
	}
}

func TestCopyModeContinuousOlderKeepsBoundedLocalHistoryWindow(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 12)
	latestRows := make([]state.HistoryRow, 0, 128)
	for i := 2000; i < 2128; i++ {
		latestRows = append(latestRows, state.HistoryRow{Text: fmt.Sprintf("new-%04d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{}
	runtime := newCopyModeRuntime(host, core, nil)
	latest := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 98, 7, latestRows)
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: latestRows[0].LineID}
	latest.HasMore = true
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       latest.Token,
		Cols:        latest.Cols,
		SourceLines: latest.SourceLines,
		Rows:        latest.Rows,
		Lines:       latest.Lines,
		Cursor:      latest.Cursor,
		Generation:  latest.Generation,
		Boundary:    state.HistoryBoundary{FirstLineID: latestRows[0].LineID, LastLineID: latestRows[len(latestRows)-1].LineID},
		HasMore:     true,
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  latest.Token,
		BoundCols:   latest.Cols,
		ViewRows:    8,
		ViewportTop: 0,
		Cursor:      state.CopyPosition{},
	}

	for page := 0; page < 10; page++ {
		runtime.state.CopyMode.ViewportTop = 0
		runtime.state.CopyMode.Cursor = state.CopyPosition{}
		first := 2000 - (page+1)*64
		olderRows := make([]state.HistoryRow, 0, 64)
		for i := 0; i < 64; i++ {
			lineID := uint64(first + i)
			olderRows = append(olderRows, state.HistoryRow{Text: fmt.Sprintf("old-%04d", lineID), LineID: lineID})
		}
		beginPendingOlderForTest(&runtime.state, state.RequestID(page+2), 0)
		if got := runtime.state.History.Pending.Boundary.LastLineID; got != latestRows[len(latestRows)-1].LineID {
			t.Fatalf("pending older must keep frozen tail boundary after local trim, got %d", got)
		}
		window := historyWindowForApp(state.HistoryWindowPrepend, "term-1", "tok-1", 98, 7, olderRows)
		window.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: olderRows[0].LineID}
		window.Boundary = state.HistoryBoundary{FirstLineID: olderRows[0].LineID, LastLineID: latestRows[len(latestRows)-1].LineID}
		window.HasMore = true
		if err := postHistoryResultForTest(runtime, port.RequestID(page+2), window); err != nil {
			t.Fatalf("post older page %d: %v", page, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain older page %d: %v", page, err)
		}
		if got := len(runtime.State().History.Rows); got > copyModeHistoryMaxRequestRows {
			t.Fatalf("continuous older should keep bounded TUI rows, got %d", got)
		}
	}
	if len(runtime.State().History.SourceLines) != len(runtime.State().History.Rows) {
		t.Fatalf("source lines should be trimmed with rows, rows=%d source=%d", len(runtime.State().History.Rows), len(runtime.State().History.SourceLines))
	}
	if runtime.State().History.Boundary.LastLineID != latestRows[len(latestRows)-1].LineID {
		t.Fatalf("trim must not rewrite frozen tail stale guard, got %#v", runtime.State().History.Boundary)
	}
}

func TestCopyModeScrollsBackToTrimmedNewerWindowFromBackend(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 12)
	loadedRows := make([]state.HistoryRow, 0, 32)
	for i := 1000; i < 1032; i++ {
		loadedRows = append(loadedRows, state.HistoryRow{Text: fmt.Sprintf("old-%04d", i), LineID: uint64(i)})
	}
	loadedRows[len(loadedRows)-1].Segment = state.HistoryCursorSegmentArchivedPrimaryFrame
	newerRows := make([]state.HistoryRow, 0, 8)
	for i := 1032; i < 1040; i++ {
		newerRows = append(newerRows, state.HistoryRow{Text: fmt.Sprintf("new-%04d", i), LineID: uint64(i)})
	}
	newer := historyWindowForApp(state.HistoryWindowAppend, "term-1", "tok-1", 98, 7, newerRows)
	newer.Boundary = state.HistoryBoundary{FirstLineID: newerRows[0].LineID, LastLineID: 1039}
	core := &testkit.FakeCoreClient{
		NewerResponses: []port.HistoryResult{{Window: newer}},
	}
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        98,
		SourceLines: historyLogicalLinesForApp(loadedRows),
		Rows:        loadedRows,
		Lines:       []state.HistoryLineSpan{{LineID: loadedRows[0].LineID, StartRow: 0, EndRow: len(loadedRows) - 1}},
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: loadedRows[0].LineID},
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: loadedRows[0].LineID, LastLineID: 1039},
		ViewportAnchor: state.HistoryViewportAnchor{
			Valid: true, TopLineID: 1036,
		},
		HasMore: true,
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   98,
		ViewRows:    8,
		ViewportTop: len(loadedRows) - 8,
		Cursor:      state.CopyPosition{Row: len(loadedRows) - 1},
	}
	binding, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 98
	binding.DesiredRows = 8
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)

	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageDn}}); err != nil {
		t.Fatalf("post page down: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain page down: %v", err)
	}
	if len(core.NewerRequests) != 1 {
		t.Fatalf("expected one newer request, got %#v", core.NewerRequests)
	}
	req := core.NewerRequests[0]
	if req.Token != "tok-1" || req.Generation != 7 || req.Boundary.LastLineID != 1039 {
		t.Fatalf("newer request must keep frozen token/generation/tail boundary, got %#v", req)
	}
	if req.Cursor.BeforeLineID != loadedRows[len(loadedRows)-1].LineID || req.Cursor.BeforeRowInLine != loadedRows[len(loadedRows)-1].RowInLine {
		t.Fatalf("newer request should start after local tail, got %#v", req.Cursor)
	}
	if req.Cursor.Segment != state.HistoryCursorSegmentArchivedPrimaryFrame {
		t.Fatalf("newer request must use authoritative tail row segment, got %#v", req.Cursor)
	}
	if got := len(runtime.State().History.Rows); got > copyModeHistoryMaxRequestRows {
		t.Fatalf("newer append should keep a bounded local window, got %d", got)
	}
	if cursorLine := runtime.State().History.Rows[runtime.State().CopyMode.Cursor.Row].LineID; cursorLine <= loadedRows[len(loadedRows)-1].LineID {
		t.Fatalf("page down should continue into appended newer rows, cursor row=%d line=%d", runtime.State().CopyMode.Cursor.Row, cursorLine)
	}
	if runtime.State().History.Boundary.LastLineID != 1039 {
		t.Fatalf("append must keep frozen tail boundary, got %#v", runtime.State().History.Boundary)
	}
	if runtime.State().CopyMode.ViewportTailRows != 4 {
		t.Fatalf("newer append should restore four frozen viewport tail rows, got %#v", runtime.State().CopyMode)
	}
}

func TestCopyModeQueuesNewerScrollWhilePageIsPending(t *testing.T) {
	root := state.Root{}
	root.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Rows:       []state.HistoryRow{{Text: "local-tail", LineID: 10}},
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 20},
		Pending: &state.HistoryPendingRequest{
			ID:                 1,
			Kind:               state.HistoryRequestNewer,
			TerminalID:         "term-1",
			Token:              "tok-1",
			DeferredScrollRows: 2,
		},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		ViewRows:   8,
	}

	next, effects := reduceCopyModeScrollNewer(root, CopyModeDeps{}, 5)

	if len(effects) != 0 {
		t.Fatalf("pending newer scroll must not start another request, got %#v", effects)
	}
	if next.History.Pending == nil || next.History.Pending.DeferredScrollRows != 7 {
		t.Fatalf("pending newer scroll should accumulate unconsumed rows, got %#v", next.History.Pending)
	}
}

func TestCopyModeQueuesOlderScrollOnceWhilePageIsPending(t *testing.T) {
	root := state.Root{}
	root.History = state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       80,
		Rows:       []state.HistoryRow{{Text: "local-head", LineID: 10}},
		Boundary:   state.HistoryBoundary{FirstLineID: 1, LastLineID: 10},
		Pending: &state.HistoryPendingRequest{
			ID:                 1,
			Kind:               state.HistoryRequestOlder,
			TerminalID:         "term-1",
			Token:              "tok-1",
			DeferredScrollRows: 2,
		},
	}
	root.CopyMode = state.CopyModeStore{
		Active:     true,
		Phase:      state.CopyModeFrozenHistory,
		TerminalID: "term-1",
		BoundToken: "tok-1",
		ViewRows:   8,
	}

	next, effects := reduceCopyModeScrollOlderRows(root, CopyModeDeps{}, 5)

	if len(effects) != 0 {
		t.Fatalf("pending older scroll must not start another request, got %#v", effects)
	}
	if next.History.Pending == nil || next.History.Pending.DeferredScrollRows != 7 {
		t.Fatalf("pending older scroll should add the unconsumed rows once, got %#v", next.History.Pending)
	}
}

func TestCopyModeContinuesNewerPagesForDeferredScroll(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 12)
	firstRows := []state.HistoryRow{{Text: "line-2", LineID: 2}, {Text: "line-3", LineID: 3}}
	secondRows := make([]state.HistoryRow, 0, 8)
	for lineID := uint64(4); lineID <= 11; lineID++ {
		secondRows = append(secondRows, state.HistoryRow{Text: fmt.Sprintf("line-%d", lineID), LineID: lineID})
	}
	first := historyWindowForApp(state.HistoryWindowAppend, "term-1", "tok-1", 78, 7, firstRows)
	first.Boundary = state.HistoryBoundary{FirstLineID: 2, LastLineID: 11}
	first.HasMore = true
	second := historyWindowForApp(state.HistoryWindowAppend, "term-1", "tok-1", 78, 7, secondRows)
	second.Boundary = state.HistoryBoundary{FirstLineID: 4, LastLineID: 11}
	core := &testkit.FakeCoreClient{NewerResponses: []port.HistoryResult{{Window: first}, {Window: second}}}
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{{Text: "line-1", LineID: 1}}),
		Rows:        []state.HistoryRow{{Text: "line-1", LineID: 1}},
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: 1, LastLineID: 11},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   78,
		ViewRows:    8,
		ViewportTop: 0,
		Cursor:      state.CopyPosition{Row: 0},
	}

	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageDn}}); err != nil {
		t.Fatalf("post page down: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain chained newer pages: %v", err)
	}
	if len(core.NewerRequests) != 2 {
		t.Fatalf("deferred scroll should request newer pages serially, got %#v", core.NewerRequests)
	}
	if runtime.State().CopyMode.Cursor.Row != 6 || runtime.State().History.Rows[6].LineID != 7 {
		t.Fatalf("deferred scroll should continue across both pages, copy=%#v rows=%#v", runtime.State().CopyMode, runtime.State().History.Rows)
	}
}

func TestCopyModeNewestReturnsToLiveAndIgnoresDelayedNewer(t *testing.T) {
	host := NewFakeTerminalHost(8)
	core := &testkit.FakeCoreClient{}
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		Rows:       []state.HistoryRow{{Text: "line-1", LineID: 1}},
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 1, LastLineID: 20},
		Pending: &state.HistoryPendingRequest{
			ID:         41,
			Kind:       state.HistoryRequestNewer,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-1",
			Generation: 7,
		},
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:     true,
		Phase:      state.CopyModeFrozenHistory,
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		BoundToken: "tok-1",
		BoundCols:  78,
		ViewRows:   8,
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "G"}); err != nil {
		t.Fatalf("send newest: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain newest: %v", err)
	}
	if runtime.State().CopyMode.Active || runtime.State().History.Token != "" {
		t.Fatalf("newest should return directly to live, copy=%#v history=%#v", runtime.State().CopyMode, runtime.State().History)
	}
	if len(core.ReleaseRequests) != 1 || core.ReleaseRequests[0].Token != "tok-1" {
		t.Fatalf("newest should release the frozen token once, got %#v", core.ReleaseRequests)
	}

	delayed := historyWindowForApp(state.HistoryWindowAppend, "term-1", "tok-1", 78, 7, []state.HistoryRow{{Text: "late", LineID: 2}})
	delayed.PaneID = state.DefaultPaneID
	delayed.ViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	delayed.Boundary = state.HistoryBoundary{FirstLineID: 2, LastLineID: 20}
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 41, Window: delayed}}); err != nil {
		t.Fatalf("post delayed newer: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain delayed newer: %v", err)
	}
	if runtime.State().CopyMode.Active || runtime.State().History.Token != "" || runtime.State().Surface.Err != "" {
		t.Fatalf("delayed newer must not revive history or surface an error, state=%#v", runtime.State())
	}
	if len(core.ReleaseRequests) != 1 {
		t.Fatalf("delayed newer must not release the token twice, got %#v", core.ReleaseRequests)
	}
}

func TestCopyModeContinuousNewerKeepsBoundedLocalHistoryWindow(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 12)
	loadedRows := make([]state.HistoryRow, 0, 384)
	for i := 1000; i < 1384; i++ {
		loadedRows = append(loadedRows, state.HistoryRow{Text: fmt.Sprintf("old-%04d", i), LineID: uint64(i)})
	}
	core := &testkit.FakeCoreClient{}
	runtime := newCopyModeRuntime(host, core, nil)
	runtime.state.History = state.HistoryStore{
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        98,
		SourceLines: historyLogicalLinesForApp(loadedRows),
		Rows:        loadedRows,
		Lines:       []state.HistoryLineSpan{{LineID: loadedRows[0].LineID, StartRow: 0, EndRow: len(loadedRows) - 1}},
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: loadedRows[0].LineID},
		Generation:  7,
		Boundary:    state.HistoryBoundary{FirstLineID: loadedRows[0].LineID, LastLineID: 3000},
		HasMore:     true,
	}
	runtime.state.CopyMode = state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		BoundToken:  "tok-1",
		BoundCols:   98,
		ViewRows:    8,
		ViewportTop: len(loadedRows) - 8,
		Cursor:      state.CopyPosition{Row: len(loadedRows) - 1},
	}
	binding, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 98
	binding.DesiredRows = 8
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)

	nextLineID := 1384
	for page := 0; page < 10; page++ {
		first := nextLineID
		newerRows := make([]state.HistoryRow, 0, 64)
		for i := 0; i < 64; i++ {
			lineID := uint64(first + i)
			newerRows = append(newerRows, state.HistoryRow{Text: fmt.Sprintf("new-%04d", lineID), LineID: lineID})
		}
		nextLineID += len(newerRows)
		window := historyWindowForApp(state.HistoryWindowAppend, "term-1", "tok-1", 98, 7, newerRows)
		window.Boundary = state.HistoryBoundary{FirstLineID: newerRows[0].LineID, LastLineID: 3000}
		window.HasMore = true
		core.NewerResponses = append(core.NewerResponses, port.HistoryResult{Window: window})

		runtime.state.CopyMode.ViewportTop = len(runtime.state.History.Rows) - runtime.state.CopyMode.ViewRows
		runtime.state.CopyMode.Cursor = state.CopyPosition{Row: len(runtime.state.History.Rows) - 1}
		if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageDn}}); err != nil {
			t.Fatalf("post page down %d: %v", page, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain newer page %d: %v", page, err)
		}
		if got := len(runtime.State().History.Rows); got > copyModeHistoryMaxRequestRows {
			t.Fatalf("continuous newer should keep bounded TUI rows, got %d", got)
		}
		if runtime.State().History.Boundary.LastLineID != 3000 {
			t.Fatalf("trim must keep frozen tail boundary, got %#v", runtime.State().History.Boundary)
		}
		if runtime.State().History.Rows[runtime.State().CopyMode.Cursor.Row].LineID < uint64(first) {
			t.Fatalf("page down should continue into appended newer rows, state=%#v", runtime.State().CopyMode)
		}
	}
	if len(core.NewerRequests) != 10 {
		t.Fatalf("expected ten newer requests, got %#v", core.NewerRequests)
	}
	if len(runtime.State().History.SourceLines) != len(runtime.State().History.Rows) {
		t.Fatalf("source lines should be trimmed with rows, rows=%d source=%d", len(runtime.State().History.Rows), len(runtime.State().History.SourceLines))
	}
}

func TestCopyModeOlderBoundaryOverlapKeepsSelectionOnOriginalContent(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(6, 12)
	latest := historyWindowForApp(
		state.HistoryWindowReplace,
		"term-1",
		"tok-latest",
		4,
		7,
		[]state.HistoryRow{{Text: "cdef", LineID: 10}},
	)
	latest.SourceLines = []state.HistoryLogicalLine{{
		Text:          "cdef",
		LineID:        10,
		ClippedBefore: true,
	}}
	latest.Lines = []state.HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0, ClippedBefore: true}}
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	latest.HasMore = true
	older := historyWindowForApp(
		state.HistoryWindowPrepend,
		"term-1",
		"tok-latest",
		4,
		7,
		[]state.HistoryRow{{Text: "ab", LineID: 10}},
	)
	older.SourceLines = []state.HistoryLogicalLine{{
		Text:         "ab",
		LineID:       10,
		ClippedAfter: true,
	}}
	older.Lines = []state.HistoryLineSpan{{LineID: 10, StartRow: 0, EndRow: 0, ClippedAfter: true}}
	older.Boundary = state.HistoryBoundary{FirstLineID: 10, LastLineID: 10}
	older.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 10}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: latest}},
		OlderResponses:  []port.HistoryResult{{Window: older}},
	}
	runtime := newCopyModeRuntime(host, core, nil)
	binding, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding.DesiredCols = 4
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding)
	runtime.state.Surface = state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"live"}}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("enter copy mode: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain latest: %v", err)
	}
	if err := runtime.Post(CopyModeSetMarkMsg{Position: state.CopyPosition{Row: 0, Col: 1}}); err != nil {
		t.Fatalf("set mark: %v", err)
	}
	if err := runtime.Post(CopyModeMoveCursorMsg{Position: state.CopyPosition{Row: 0, Col: 3}}); err != nil {
		t.Fatalf("move cursor: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain selection setup: %v", err)
	}
	if got := SelectedText(runtime.State().History, runtime.State().CopyMode); got != "de" {
		t.Fatalf("expected initial selected suffix before overlap prepend, got %q", got)
	}

	beginPendingOlderForTest(&runtime.state, 2, 0)
	core.OlderRequests = append(core.OlderRequests, port.HistoryOlderRequest{RequestID: 2})
	if err := postHistoryResultForTest(runtime, 2, older); err != nil {
		t.Fatalf("post older: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older: %v", err)
	}
	if got := SelectedText(runtime.State().History, runtime.State().CopyMode); got != "de" {
		t.Fatalf("overlap prepend must keep selection on original suffix, got %q", got)
	}
	if runtime.State().CopyMode.Cursor != (state.CopyPosition{Row: 1, Col: 1}) {
		t.Fatalf("overlap prepend must rebind cursor to original suffix, got %#v", runtime.State().CopyMode.Cursor)
	}
	if runtime.State().CopyMode.Mark == nil || *runtime.State().CopyMode.Mark != (state.CopyPosition{Row: 0, Col: 3}) {
		t.Fatalf("overlap prepend must rebind mark to original suffix, got %#v", runtime.State().CopyMode.Mark)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "⇡") || frameContains(frame, "⇣") {
		t.Fatalf("overlap prepend should not keep clipped markers after line is complete, got %#v", frame.Lines)
	}
}

func TestSelectedTextSupportsReversedMultiRowSelection(t *testing.T) {
	history := historyStoreForCopySelection()
	copyMode := state.CopyModeStore{
		Active: true,
		Selection: &state.CopySelection{
			Anchor: state.CopyPosition{Row: 1, Col: 2},
			Focus:  state.CopyPosition{Row: 0, Col: 1},
		},
	}
	if got := SelectedText(history, copyMode); got != "lpha\nbe" {
		t.Fatalf("unexpected selected text %q", got)
	}
}

func TestSelectedTextUsesDisplayColumns(t *testing.T) {
	history := state.HistoryStore{
		Rows: []state.HistoryRow{{Text: "你好abc", LineID: 1}},
	}
	copyMode := state.CopyModeStore{
		Active: true,
		Selection: &state.CopySelection{
			Anchor: state.CopyPosition{Row: 0, Col: 2},
			Focus:  state.CopyPosition{Row: 0, Col: 5},
		},
	}
	if got := SelectedText(history, copyMode); got != "好a" {
		t.Fatalf("unexpected selected text %q", got)
	}
}

func TestSelectedTextJoinsRowsFromSameLogicalLineWithoutNewline(t *testing.T) {
	history := state.HistoryStore{
		Rows: []state.HistoryRow{
			{Text: "alpha", LineID: 10, RowInLine: 0},
			{Text: "beta", LineID: 10, RowInLine: 1},
			{Text: "gamma", LineID: 20, RowInLine: 0},
		},
		Lines: []state.HistoryLineSpan{
			{LineID: 10, StartRow: 0, EndRow: 1},
			{LineID: 20, StartRow: 2, EndRow: 2},
		},
	}
	copyMode := state.CopyModeStore{
		Active: true,
		Selection: &state.CopySelection{
			Anchor: state.CopyPosition{Row: 0, Col: 2},
			Focus:  state.CopyPosition{Row: 1, Col: 2},
		},
	}
	if got := SelectedText(history, copyMode); got != "phabe" {
		t.Fatalf("selection across reflow rows of same logical line must not inject newline, got %q", got)
	}
}

func TestSelectedTextKeepsNewlineAcrossLogicalLines(t *testing.T) {
	history := state.HistoryStore{
		Rows: []state.HistoryRow{
			{Text: "alpha", LineID: 10, RowInLine: 0},
			{Text: "beta", LineID: 10, RowInLine: 1},
			{Text: "gamma", LineID: 20, RowInLine: 0},
		},
		Lines: []state.HistoryLineSpan{
			{LineID: 10, StartRow: 0, EndRow: 1},
			{LineID: 20, StartRow: 2, EndRow: 2},
		},
	}
	copyMode := state.CopyModeStore{
		Active: true,
		Selection: &state.CopySelection{
			Anchor: state.CopyPosition{Row: 1, Col: 2},
			Focus:  state.CopyPosition{Row: 2, Col: 2},
		},
	}
	if got := SelectedText(history, copyMode); got != "ta\nga" {
		t.Fatalf("selection that crosses into next logical line must keep newline, got %q", got)
	}
}

func TestCopyModeLineEndAndClampUseDisplayColumns(t *testing.T) {
	history := state.HistoryStore{
		Rows: []state.HistoryRow{{Text: "a好", LineID: 1}},
	}
	if got := copyModeLineEndPosition(history, 0); got != (state.CopyPosition{Row: 0, Col: 3}) {
		t.Fatalf("line end should use display width, got %#v", got)
	}
	copyMode := state.CopyModeStore{Cursor: state.CopyPosition{Row: 0, Col: 99}}
	copyMode = clampCopyCursor(copyMode, history)
	if copyMode.Cursor != (state.CopyPosition{Row: 0, Col: 3}) {
		t.Fatalf("clamp should use display width, got %#v", copyMode.Cursor)
	}
}

func TestCopyModeRejectsStaleHistoryResult(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := newCopyModeRuntime(host, &testkit.FakeCoreClient{}, nil)
	runtime.state.History = state.HistoryStore{
		Pending: &state.HistoryPendingRequest{ID: 2, Kind: state.HistoryRequestOlder, TerminalID: "term-1", Cols: 80, Token: "tok-1", Generation: 7},
	}

	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{
		RequestID: 2,
		Window:    historyWindowForApp(state.HistoryWindowPrepend, "term-1", "stale", 80, 7, []state.HistoryRow{{Text: "old", LineID: 1}}),
	}}); err != nil {
		t.Fatalf("post stale result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if runtime.State().History.Token != "" {
		t.Fatalf("stale response should not mutate history %#v", runtime.State().History)
	}
	if runtime.State().Session.LastError != "" || runtime.State().Surface.Err != "" {
		t.Fatalf("stale response should be ignored without live error %#v", runtime.State())
	}
}

func newCopyModeRuntime(host *FakeTerminalHost, core port.CoreClient, clipboard port.ClipboardService) *AppRuntime {
	return newCopyModeRuntimeWithRunner(host, core, clipboard, &testkit.FakeTerminalService{}, NewSyncEffectRunner())
}

func newCopyModeRuntimeWithRunner(host *FakeTerminalHost, core port.CoreClient, clipboard port.ClipboardService, terminal port.TerminalService, runner EffectRunner) *AppRuntime {
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	if cols, rows, _ := host.Size(); cols <= 0 || rows <= 0 {
		host.SetSize(80, 24)
	}
	return NewAppRuntime(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Channel:    4,
				InputChannels: map[string]uint16{
					"term-1": 4,
				},
				Attached: true,
				Cols:     80,
				Rows:     24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
		},
		ComposeReducers(
			NewBackNavigationReducer(CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20}),
			NewClipboardActionReducer(ClipboardActionDeps{Core: core, Clipboard: clipboard, Terminal: terminal}),
			NewCopyModeReducer(CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20}),
			NewCopyModeResizeRebindReducer(CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20}),
		),
		func(root state.Root) render.Frame {
			return renderer.Render(builder.Build(root))
		},
		host,
		runner,
	)
}

func newInteractiveCopyModeRuntimeWithRunner(host *FakeTerminalHost, core port.CoreClient, clipboard port.ClipboardService, terminal port.TerminalService, runner EffectRunner) *AppRuntime {
	if cols, rows, _ := host.Size(); cols <= 0 || rows <= 0 {
		host.SetSize(80, 24)
	}
	return NewInteractiveRuntimeWithWorkbench(
		state.Root{
			Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
			Session: state.TerminalSessionStore{
				TerminalID: "term-1",
				Channel:    4,
				InputChannels: map[string]uint16{
					"term-1": 4,
				},
				Attached: true,
				Cols:     80,
				Rows:     24,
			},
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				Lines:      []string{"live"},
			},
			TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
				state.DefaultPaneID, "term-1", 4, 78, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
			)),
		},
		host,
		runner,
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Clipboard: clipboard, Rows: 20},
		WorkbenchDeps{},
	)
}

func copyHistoryRowHitRegionForPane(t *testing.T, frame render.Frame, paneID string, row int) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionHistoryRow && region.PaneID == paneID && region.Row == row {
			return region
		}
	}
	t.Fatalf("missing copy history row hit region pane=%s row=%d in %#v", paneID, row, frame.HitRegions)
	return render.HitRegion{}
}

type recordingEffectRunner struct {
	Effects      []Effect
	CancelTokens []CancelToken
}

func (runner *recordingEffectRunner) Run(_ context.Context, effect Effect, _ func(Msg)) {
	runner.Effects = append(runner.Effects, effect)
}

func (runner *recordingEffectRunner) Cancel(token CancelToken) {
	runner.CancelTokens = append(runner.CancelTokens, token)
}

func historyRequestEffectCount(effects []Effect) int {
	count := 0
	for _, effect := range effects {
		fn, ok := effect.(FuncEffect)
		if !ok || fn.Run == nil || !fn.ForceSyncInTests || fn.Token == shellShortcutPassthroughTimeoutToken {
			continue
		}
		count++
	}
	return count
}

type blockingHistoryClient struct {
	mu            sync.Mutex
	allowOlder    bool
	latestReqs    []port.HistoryLatestRequest
	olderReqs     []port.HistoryOlderRequest
	latestResultC chan blockingHistoryResult
	olderResultC  chan blockingHistoryResult
}

type blockingHistoryResult struct {
	result port.HistoryResult
	err    error
}

func (client *blockingHistoryClient) HistoryLatest(ctx context.Context, req port.HistoryLatestRequest) (port.HistoryResult, error) {
	client.mu.Lock()
	client.latestReqs = append(client.latestReqs, req)
	if client.latestResultC == nil {
		client.latestResultC = make(chan blockingHistoryResult, 1)
	}
	resultC := client.latestResultC
	client.mu.Unlock()
	select {
	case <-ctx.Done():
		return port.HistoryResult{}, ctx.Err()
	case item := <-resultC:
		item.result.RequestID = req.RequestID
		return item.result, item.err
	}
}

func (client *blockingHistoryClient) HistoryOlder(ctx context.Context, req port.HistoryOlderRequest) (port.HistoryResult, error) {
	client.mu.Lock()
	client.olderReqs = append(client.olderReqs, req)
	if !client.allowOlder {
		client.mu.Unlock()
		return port.HistoryResult{}, errors.New("unexpected older request")
	}
	if client.olderResultC == nil {
		client.olderResultC = make(chan blockingHistoryResult, 1)
	}
	resultC := client.olderResultC
	client.mu.Unlock()
	select {
	case <-ctx.Done():
		return port.HistoryResult{}, ctx.Err()
	case item := <-resultC:
		item.result.RequestID = req.RequestID
		return item.result, item.err
	}
}

func (client *blockingHistoryClient) HistoryOldest(context.Context, port.HistoryOldestRequest) (port.HistoryResult, error) {
	return port.HistoryResult{}, errors.New("unexpected oldest request")
}

func (client *blockingHistoryClient) HistoryNewer(context.Context, port.HistoryNewerRequest) (port.HistoryResult, error) {
	return port.HistoryResult{}, errors.New("unexpected newer request")
}

func (client *blockingHistoryClient) HistoryCopyRange(context.Context, port.HistoryCopyRangeRequest) (port.HistoryCopyRangeResult, error) {
	return port.HistoryCopyRangeResult{}, errors.New("unexpected copy range request")
}

func (client *blockingHistoryClient) HistorySearch(context.Context, port.HistorySearchRequest) (port.HistorySearchResult, error) {
	return port.HistorySearchResult{}, errors.New("unexpected search request")
}

func (client *blockingHistoryClient) ReleaseHistory(context.Context, port.HistoryReleaseRequest) error {
	return nil
}

func (client *blockingHistoryClient) latestRequests() []port.HistoryLatestRequest {
	client.mu.Lock()
	defer client.mu.Unlock()
	out := make([]port.HistoryLatestRequest, len(client.latestReqs))
	copy(out, client.latestReqs)
	return out
}

func (client *blockingHistoryClient) olderRequests() []port.HistoryOlderRequest {
	client.mu.Lock()
	defer client.mu.Unlock()
	out := make([]port.HistoryOlderRequest, len(client.olderReqs))
	copy(out, client.olderReqs)
	return out
}

func (client *blockingHistoryClient) finishLatest(result port.HistoryResult, err error) {
	client.mu.Lock()
	if client.latestResultC == nil {
		client.latestResultC = make(chan blockingHistoryResult, 1)
	}
	resultC := client.latestResultC
	client.mu.Unlock()
	resultC <- blockingHistoryResult{result: result, err: err}
}

func (client *blockingHistoryClient) finishOlder(result port.HistoryResult, err error) {
	client.mu.Lock()
	if client.olderResultC == nil {
		client.olderResultC = make(chan blockingHistoryResult, 1)
	}
	resultC := client.olderResultC
	client.mu.Unlock()
	resultC <- blockingHistoryResult{result: result, err: err}
}

func historyWindowForApp(
	op state.HistoryWindowOp,
	terminalID string,
	token string,
	cols int,
	generation uint64,
	rows []state.HistoryRow,
) state.HistoryWindow {
	firstLine := uint64(0)
	lastLine := uint64(0)
	if len(rows) > 0 {
		firstLine = rows[0].LineID
		lastLine = rows[len(rows)-1].LineID
	}
	return state.HistoryWindow{
		TerminalID:  terminalID,
		Token:       token,
		Op:          op,
		Cols:        cols,
		SourceLines: historyLogicalLinesForApp(rows),
		Rows:        rows,
		Lines:       []state.HistoryLineSpan{{LineID: firstLine, StartRow: 0, EndRow: len(rows) - 1}},
		Generation:  generation,
		Boundary:    state.HistoryBoundary{FirstLineID: firstLine, LastLineID: lastLine},
	}
}

func historyRowsForEnteringScrollTest(prefix string, count int, firstLine uint64) []state.HistoryRow {
	rows := make([]state.HistoryRow, count)
	for i := range rows {
		rows[i] = state.HistoryRow{
			Text:   fmt.Sprintf("%s-%02d", prefix, i),
			LineID: firstLine + uint64(i),
		}
	}
	return rows
}

func historyLogicalLinesForApp(rows []state.HistoryRow) []state.HistoryLogicalLine {
	if len(rows) == 0 {
		return nil
	}
	lines := make([]state.HistoryLogicalLine, len(rows))
	for i, row := range rows {
		lines[i] = state.HistoryLogicalLine{
			Text:         row.Text,
			Cells:        append([]state.HistoryCell(nil), row.Cells...),
			LineID:       row.LineID,
			Kind:         row.Kind,
			Segment:      row.Segment,
			SessionID:    row.SessionID,
			FrameID:      row.FrameID,
			FixedGrid:    row.FixedGrid,
			ScreenCols:   row.ScreenCols,
			ScreenRow:    row.ScreenRow,
			ScreenRowSet: row.ScreenRowSet,
			LiveTail:     row.LiveTail,
		}
	}
	return lines
}

func copyHistorySessionForWheelIsolation(paneID string, viewID string, terminalID string, token string, cols int, prefix string, rowCount int, viewRows int) (state.HistoryStore, state.CopyModeStore) {
	rows := make([]state.HistoryRow, rowCount)
	for i := range rows {
		rows[i] = state.HistoryRow{Text: fmt.Sprintf("%s-row-%02d", prefix, i), LineID: uint64(i + 1)}
	}
	history := state.HistoryStore{
		PaneID:      paneID,
		ViewID:      viewID,
		TerminalID:  terminalID,
		Token:       token,
		Cols:        cols,
		Generation:  1,
		Boundary:    state.HistoryBoundary{FirstLineID: 1, LastLineID: uint64(rowCount)},
		SourceLines: historyLogicalLinesForApp(rows),
		Rows:        rows,
		HasMore:     true,
	}
	cursor := rowCount / 2
	copyMode := state.CopyModeStore{
		Active:      true,
		PaneID:      paneID,
		ViewID:      viewID,
		TerminalID:  terminalID,
		BoundToken:  token,
		BoundCols:   cols,
		ViewRows:    viewRows,
		Cursor:      state.CopyPosition{Row: cursor},
		ViewportTop: cursor,
	}
	return history, copyMode
}

func historyStoreForCopySelection() state.HistoryStore {
	return state.HistoryStore{
		TerminalID: "term-1",
		Token:      "tok-1",
		Cols:       78,
		SourceLines: historyLogicalLinesForApp([]state.HistoryRow{
			{Text: "alpha", LineID: 10},
			{Text: "beta", LineID: 11},
		}),
		Rows: []state.HistoryRow{
			{Text: "alpha", LineID: 10},
			{Text: "beta", LineID: 11},
		},
	}
}

func beginPendingOlderForTest(root *state.Root, requestID state.RequestID, scrollDeltaAfterPrepend int) {
	root.History.Pending = &state.HistoryPendingRequest{
		ID:                 requestID,
		Kind:               state.HistoryRequestOlder,
		PaneID:             root.History.PaneID,
		ViewID:             root.History.ViewID,
		TerminalID:         root.History.TerminalID,
		Cols:               root.History.Cols,
		Token:              root.History.Token,
		Generation:         root.History.Generation,
		Cursor:             root.History.Cursor,
		Boundary:           root.History.Boundary,
		DeferredScrollRows: scrollDeltaAfterPrepend,
	}
}

func postHistoryResultForTest(runtime *AppRuntime, requestID port.RequestID, window state.HistoryWindow) error {
	window.PaneID = runtime.state.History.PaneID
	window.ViewID = runtime.state.History.ViewID
	return runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: requestID, Window: window}})
}

func historyRowTexts(rows []state.HistoryRow) []string {
	texts := make([]string, len(rows))
	for i, row := range rows {
		texts[i] = row.Text
	}
	return texts
}

func activeCopyContentStatus(runtime *AppRuntime) string {
	return activeCopyContent(runtime).Status
}

func activeCopyContentLines(runtime *AppRuntime) []string {
	content := activeCopyContent(runtime)
	lines := make([]string, len(content.Lines))
	for i, line := range content.Lines {
		lines[i] = line.PlainString()
	}
	return lines
}

func copyHistoryLinesContain(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

func activeCopyContent(runtime *AppRuntime) render.ContentVM {
	vm := render.NewRenderVMBuilder().Build(runtime.State())
	for _, panel := range vm.Shell.Layout.Panels {
		if panel.Active {
			return panel.Content
		}
	}
	if len(vm.Shell.Layout.Panels) > 0 {
		return vm.Shell.Layout.Panels[0].Content
	}
	return render.ContentVM{}
}
