package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestReplacedAttachCandidateCleansLateResourceWithoutReplacingCommittedBinding(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	committed := state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-old", viewID, true)
	committed.Session = testEndpointSessionStamp(state.DefaultEndpointID)
	committed.OperationID = "attach-old"
	root := state.Root{Shell: state.DefaultShell(), TerminalViews: state.TerminalViewStore{}.BindPane(committed)}

	first := state.TerminalViewBinding{ViewID: viewID, EndpointID: state.DefaultEndpointID, TerminalID: "term-first", SurfaceID: "surface", PaneID: state.DefaultPaneID}
	var firstCandidate state.TerminalAttachCandidate
	root.TerminalViews, firstCandidate = root.TerminalViews.BeginAttach(first)
	second := first
	second.TerminalID = "term-second"
	var secondCandidate state.TerminalAttachCandidate
	root.TerminalViews, secondCandidate = root.TerminalViews.BeginAttach(second)

	late := port.TerminalAttachResult{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", Channel: 9, SurfaceID: "surface", ViewID: viewID,
		Session: testEndpointSessionStamp(state.DefaultEndpointID), OperationID: firstCandidate.OperationID,
	}
	next, effects := reduceLiveAttachResult(root, LiveAttachResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", ViewID: viewID,
		OperationID: firstCandidate.OperationID, Result: late,
	}, LiveDeps{})
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.Channel != 7 || binding.TerminalID != "term-old" || binding.AttachCandidate == nil || binding.AttachCandidate.OperationID != secondCandidate.OperationID {
		t.Fatalf("late candidate changed committed binding: %#v", binding)
	}
	if len(effects) != 1 {
		t.Fatalf("late candidate cleanup effects = %#v", effects)
	}
	request, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveDetachRequestMsg)
	if !ok || request.Request.Channel != 9 || request.Request.OperationID != "cleanup:"+firstCandidate.OperationID || !protoEndpointSessionEqual(request.Request.Session, late.Session) {
		t.Fatalf("late candidate cleanup request = %#v", request)
	}
}

func TestLateInputAttachResultDoesNotSendCachedInputToReplacementBinding(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	root := state.Root{Shell: state.DefaultShell()}
	first := state.TerminalViewBinding{ViewID: viewID, EndpointID: state.DefaultEndpointID, TerminalID: "term-first", SurfaceID: "surface", PaneID: state.DefaultPaneID}
	var firstCandidate state.TerminalAttachCandidate
	root.TerminalViews, firstCandidate = root.TerminalViews.BeginAttach(first)
	second := first
	second.TerminalID = "term-second"
	var secondCandidate state.TerminalAttachCandidate
	root.TerminalViews, secondCandidate = root.TerminalViews.BeginAttach(second)
	secondBinding := state.NewEndpointPaneTerminalView(state.DefaultEndpointID, state.DefaultPaneID, "term-second", 10, 80, 24, state.TerminalResizeRoleOwner, "surface", viewID, true)
	secondBinding.Session = &apipb.EndpointSessionStamp{EndpointId: string(state.DefaultEndpointID), RouteId: "local", Generation: 2}
	secondBinding.OperationID = secondCandidate.OperationID
	var committed bool
	root.TerminalViews, _, committed = root.TerminalViews.CommitAttach(viewID, secondCandidate.OperationID, secondBinding)
	if !committed {
		t.Fatal("replacement candidate did not commit")
	}
	late := port.TerminalAttachResult{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", Channel: 9, SurfaceID: "surface", ViewID: viewID,
		Session: &apipb.EndpointSessionStamp{EndpointId: string(state.DefaultEndpointID), RouteId: "local", Generation: 1}, OperationID: firstCandidate.OperationID,
	}
	next, effects := reduceLiveInputAttachResult(root, LiveInputAttachResultMsg{
		Target: liveInputTargetInfo{EndpointID: state.DefaultEndpointID, TerminalID: "term-first", ViewID: viewID, SurfaceID: "surface"},
		Event:  input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}, Bytes: []byte("x"), OperationID: firstCandidate.OperationID, Result: late,
	}, LiveDeps{})
	binding := next.TerminalViews.Views[viewID]
	if binding.TerminalID != "term-second" || binding.Channel != 10 || binding.OperationID != secondCandidate.OperationID {
		t.Fatalf("late input attach changed replacement binding: %#v", binding)
	}
	if len(effects) != 1 {
		t.Fatalf("late input attach effects = %#v", effects)
	}
	if _, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveDetachRequestMsg); !ok {
		t.Fatalf("late input attach must only cleanup its resource: %#v", effects)
	}
}

func TestOldGenerationInputAndResizeErrorsDoNotMutateCurrentBinding(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	currentSession := &apipb.EndpointSessionStamp{EndpointId: string(state.DefaultEndpointID), RouteId: "local", Generation: 2}
	binding := state.NewEndpointPaneTerminalView(state.DefaultEndpointID, state.DefaultPaneID, "term-current", 10, 100, 30, state.TerminalResizeRoleOwner, "surface", viewID, true)
	binding.Session = currentSession
	binding.RequestSeq = 3
	root := state.Root{Shell: state.DefaultShell(), TerminalViews: state.TerminalViewStore{}.BindPane(binding)}
	before := root
	reducer := NewLiveReducer(LiveDeps{})
	oldSession := &apipb.EndpointSessionStamp{EndpointId: string(state.DefaultEndpointID), RouteId: "local", Generation: 1}
	next, effects := reducer(root, LiveInputResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-current", ViewID: viewID, Channel: 9, Session: oldSession,
		OperationID: "input-old", Err: errors.New("terminal not found"),
	})
	if len(effects) != 0 || !reflect.DeepEqual(next, before) {
		t.Fatalf("old input error mutated current state: next=%#v effects=%#v", next, effects)
	}
	next, effects = reducer(root, LiveResizeResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-current", ViewID: viewID, Seq: 1, Session: oldSession,
		OperationID: "resize-old", Err: errors.New("terminal not found"),
	})
	if len(effects) != 0 || !reflect.DeepEqual(next, before) {
		t.Fatalf("old resize error mutated current state: next=%#v effects=%#v", next, effects)
	}
}

func TestReplacedAttachErrorsCannotChangeTerminalOrPoolProjection(t *testing.T) {
	viewID := state.TerminalPaneViewID(state.DefaultPaneID)
	root := state.Root{Shell: state.DefaultShell()}
	first := state.TerminalViewBinding{ViewID: viewID, EndpointID: state.DefaultEndpointID, TerminalID: "term-first", SurfaceID: "surface", PaneID: state.DefaultPaneID}
	var firstCandidate state.TerminalAttachCandidate
	root.TerminalViews, firstCandidate = root.TerminalViews.BeginAttach(first)
	second := first
	second.TerminalID = "term-second"
	var secondCandidate state.TerminalAttachCandidate
	root.TerminalViews, secondCandidate = root.TerminalViews.BeginAttach(second)
	before := root
	next, effects := reduceLiveAttachResult(root, LiveAttachResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", ViewID: viewID, OperationID: firstCandidate.OperationID, Err: errors.New("terminal not found"),
	}, LiveDeps{})
	if len(effects) != 0 || !reflect.DeepEqual(next, before) {
		t.Fatalf("replaced attach error mutated current state: next=%#v effects=%#v", next, effects)
	}

	staleResult := port.TerminalAttachResult{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", ViewID: viewID, Channel: 9, SurfaceID: "surface",
		Session: testEndpointSessionStamp(state.DefaultEndpointID), OperationID: firstCandidate.OperationID,
	}
	next, effects = reduceTerminalPoolAttachResult(root, TerminalPoolAttachResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", TargetPaneID: state.DefaultPaneID,
		OperationID: firstCandidate.OperationID, Result: staleResult,
	}, LiveDeps{})
	if !reflect.DeepEqual(next.TerminalPool, before.TerminalPool) || !reflect.DeepEqual(next.Endpoints, before.Endpoints) {
		t.Fatalf("stale pool attach changed projections: pool=%#v endpoints=%#v", next.TerminalPool, next.Endpoints)
	}
	if binding := next.TerminalViews.Views[viewID]; binding.AttachCandidate == nil || binding.AttachCandidate.OperationID != secondCandidate.OperationID {
		t.Fatalf("stale pool attach changed current candidate: %#v", binding)
	}
	if len(effects) != 1 {
		t.Fatalf("stale pool attach cleanup effects = %#v", effects)
	}
	next, effects = reduceTerminalPoolAttachResult(root, TerminalPoolAttachResultMsg{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-first", TargetPaneID: state.DefaultPaneID,
		OperationID: firstCandidate.OperationID, Err: errors.New("terminal not found"),
	}, LiveDeps{})
	if len(effects) != 0 || !reflect.DeepEqual(next, before) {
		t.Fatalf("stale pool attach error mutated projections: next=%#v effects=%#v", next, effects)
	}
}

func prepareLiveAttachResultForTest(root state.Root, msg LiveAttachResultMsg) (state.Root, LiveAttachResultMsg) {
	viewID := msg.ViewID
	if viewID == "" {
		viewID = msg.Result.ViewID
	}
	if viewID == "" {
		viewID = state.TerminalPaneViewID(root.Shell.EnsureDefaults().ActivePaneID)
	}
	endpointID := msg.EndpointID
	if endpointID == "" {
		endpointID = msg.Result.EndpointID
	}
	endpointID = state.NormalizeEndpointID(endpointID)
	terminalID := msg.TerminalID
	if terminalID == "" {
		terminalID = msg.Result.TerminalID
	}
	target, ok := liveAttachTargetForViewID(root, viewID)
	if !ok {
		target.PaneID = root.Shell.EnsureDefaults().ActivePaneID
	}
	binding := state.TerminalViewBinding{
		ViewID: viewID, SurfaceID: msg.Result.SurfaceID, EndpointID: endpointID, TerminalID: terminalID,
		ResizeRole: msg.Result.ResizePolicy, DesiredCols: msg.Result.Cols, DesiredRows: msg.Result.Rows,
		PaneID: target.PaneID, FloatingID: target.FloatingID,
	}
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	msg.EndpointID = endpointID
	msg.TerminalID = terminalID
	msg.ViewID = viewID
	msg.OperationID = candidate.OperationID
	msg.Result.EndpointID = endpointID
	msg.Result.TerminalID = terminalID
	msg.Result.ViewID = viewID
	msg.Result.OperationID = candidate.OperationID
	if msg.Result.Session == nil {
		msg.Result.Session = testEndpointSessionStamp(endpointID)
	}
	return root, msg
}

func reduceLiveAttachResultPrepared(root state.Root, msg LiveAttachResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.OperationID == "" {
		root, msg = prepareLiveAttachResultForTest(root, msg)
	}
	return reduceLiveAttachResult(root, msg, deps)
}

func newLiveReducerPrepared(deps LiveDeps) Reducer {
	reducer := NewLiveReducer(deps)
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		if attach, ok := msg.(LiveAttachResultMsg); ok && attach.OperationID == "" {
			root, attach = prepareLiveAttachResultForTest(root, attach)
			msg = attach
		}
		if resize, ok := msg.(LiveResizeResultMsg); ok && resize.Session == nil && resize.ViewID != "" {
			if binding, found := root.TerminalViews.Views[resize.ViewID]; found {
				if binding.Session == nil {
					binding.Session = testEndpointSessionStamp(binding.EndpointID)
					root.TerminalViews.Views = cloneTerminalViewBindingsForTest(root.TerminalViews.Views)
					root.TerminalViews.Views[resize.ViewID] = binding
				}
				resize.Session = binding.AttachmentSession()
				if resize.OperationID == "" {
					resize.OperationID = "resize-test"
				}
				msg = resize
			}
		}
		return reducer(root, msg)
	}
}

func cloneTerminalViewBindingsForTest(source map[string]state.TerminalViewBinding) map[string]state.TerminalViewBinding {
	clone := make(map[string]state.TerminalViewBinding, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func testEndpointSessionStamp(endpointID state.EndpointID) *apipb.EndpointSessionStamp {
	return &apipb.EndpointSessionStamp{EndpointId: string(state.NormalizeEndpointID(endpointID)), RouteId: "test", Generation: 1}
}

func newTerminalPoolReducerPrepared(deps LiveDeps) Reducer {
	reducer := NewTerminalPoolReducer(deps)
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch value := msg.(type) {
		case TerminalPoolAttachResultMsg:
			if value.OperationID == "" {
				root, value = prepareTerminalPoolAttachResultForTest(root, value)
				msg = value
			}
		case TerminalPoolReconnectResultMsg:
			if value.OperationID == "" {
				attach := TerminalPoolAttachResultMsg{
					EndpointID: value.EndpointID, TerminalID: value.TerminalID, TargetPaneID: value.TargetPaneID,
					TargetFloatingID: value.TargetFloatingID, ResizePolicy: value.ResizePolicy, Result: value.Result, Err: value.Err,
				}
				root, attach = prepareTerminalPoolAttachResultForTest(root, attach)
				value.OperationID = attach.OperationID
				value.Result = attach.Result
				msg = value
			}
		}
		return reducer(root, msg)
	}
}

func prepareTerminalPoolAttachResultForTest(root state.Root, msg TerminalPoolAttachResultMsg) (state.Root, TerminalPoolAttachResultMsg) {
	target, _ := terminalPoolTargetFromRequest(root, msg.TargetPaneID, msg.TargetFloatingID)
	endpointID := state.NormalizeEndpointID(msg.EndpointID)
	if msg.Result.EndpointID != "" {
		endpointID = state.NormalizeEndpointID(msg.Result.EndpointID)
	}
	terminalID := msg.TerminalID
	if terminalID == "" {
		terminalID = msg.Result.TerminalID
	}
	viewID := msg.Result.ViewID
	if viewID == "" {
		viewID = target.ViewID
	}
	binding := state.TerminalViewBinding{
		ViewID: viewID, SurfaceID: msg.Result.SurfaceID, EndpointID: endpointID, TerminalID: terminalID,
		ResizeRole: msg.ResizePolicy, DesiredCols: msg.Result.Cols, DesiredRows: msg.Result.Rows,
		PaneID: target.PaneID, FloatingID: target.FloatingID,
	}
	var candidate state.TerminalAttachCandidate
	root.TerminalViews, candidate = root.TerminalViews.BeginAttach(binding)
	msg.OperationID = candidate.OperationID
	msg.Result.EndpointID = endpointID
	msg.Result.TerminalID = terminalID
	msg.Result.ViewID = viewID
	msg.Result.OperationID = candidate.OperationID
	if msg.Result.Session == nil {
		msg.Result.Session = testEndpointSessionStamp(endpointID)
	}
	return root, msg
}

func reduceTerminalPoolAttachResultPrepared(root state.Root, msg TerminalPoolAttachResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.OperationID == "" {
		root, msg = prepareTerminalPoolAttachResultForTest(root, msg)
	}
	return reduceTerminalPoolAttachResult(root, msg, deps)
}

func postPreparedLiveAttachResult(runtime *AppRuntime, msg LiveAttachResultMsg) error {
	runtime.state, msg = prepareLiveAttachResultForTest(runtime.state, msg)
	return runtime.Post(msg)
}

func postPreparedTerminalPoolAttachResult(runtime *AppRuntime, msg TerminalPoolAttachResultMsg) error {
	runtime.state, msg = prepareTerminalPoolAttachResultForTest(runtime.state, msg)
	return runtime.Post(msg)
}

func postPreparedTerminalPoolReconnectResult(runtime *AppRuntime, msg TerminalPoolReconnectResultMsg) error {
	attach := TerminalPoolAttachResultMsg{
		EndpointID: msg.EndpointID, TerminalID: msg.TerminalID, TargetPaneID: msg.TargetPaneID,
		TargetFloatingID: msg.TargetFloatingID, ResizePolicy: msg.ResizePolicy, Result: msg.Result, Err: msg.Err,
	}
	runtime.state, attach = prepareTerminalPoolAttachResultForTest(runtime.state, attach)
	msg.OperationID = attach.OperationID
	msg.Result = attach.Result
	return runtime.Post(msg)
}

func reduceTerminalPoolReconnectResultPrepared(root state.Root, msg TerminalPoolReconnectResultMsg, deps LiveDeps) (state.Root, []Effect) {
	if msg.OperationID == "" {
		attach := TerminalPoolAttachResultMsg{
			EndpointID: msg.EndpointID, TerminalID: msg.TerminalID, TargetPaneID: msg.TargetPaneID,
			TargetFloatingID: msg.TargetFloatingID, ResizePolicy: msg.ResizePolicy, Result: msg.Result, Err: msg.Err,
		}
		root, attach = prepareTerminalPoolAttachResultForTest(root, attach)
		msg.OperationID = attach.OperationID
		msg.Result = attach.Result
	}
	return reduceTerminalPoolReconnectResult(root, msg, deps)
}
