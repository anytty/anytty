package state

import "testing"

func TestTerminalViewStoreKeepsMultipleViewsForSameTerminal(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	bindings := store.BindingsForTerminal("term-1")
	if len(bindings) != 2 {
		t.Fatalf("expected two view bindings for one terminal, got %#v", bindings)
	}
	first, ok := store.PaneBinding("pane-1")
	if !ok || first.Channel != 7 || first.ResizeRole != TerminalResizeRoleOwner {
		t.Fatalf("unexpected owner binding %#v ok=%v", first, ok)
	}
	second, ok := store.PaneBinding("pane-2")
	if !ok || second.Channel != 8 || second.ResizeRole != TerminalResizeRoleFollower {
		t.Fatalf("unexpected follower binding %#v ok=%v", second, ok)
	}
}

func TestTerminalViewStoreDetachAndKillHaveDifferentScope(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	store = store.DetachPane("pane-1")
	if _, ok := store.PaneBinding("pane-1"); ok {
		t.Fatal("detached pane should lose its view binding")
	}
	second, ok := store.PaneBinding("pane-2")
	if !ok {
		t.Fatal("sibling pane binding should survive detach")
	}
	if second.ResizeRole != TerminalResizeRoleFollower || second.CanResize || second.ResizePending {
		t.Fatalf("sibling pane must wait for daemon owner projection after detach, got %#v", second)
	}

	store = store.RemoveTerminal("term-1")
	if bindings := store.BindingsForTerminal("term-1"); len(bindings) != 0 {
		t.Fatalf("remove terminal should remove all view bindings, got %#v", bindings)
	}
}

func TestTerminalViewStoreMarkTerminalReattachingClearsOnlyRestartedChannels(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))
	store = store.BindPane(NewPaneTerminalView("pane-3", "term-2", 9, 100, 30, TerminalResizeRoleOwner, "surface", "view-3", true))

	store = store.MarkTerminalReattaching("term-1")
	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	third, _ := store.PaneBinding("pane-3")
	if first.Channel != 0 || first.Attached || first.TerminalID != "term-1" || first.ResizeRole != TerminalResizeRoleFollower || first.CanResize {
		t.Fatalf("restart should keep binding intent but clear runtime owner and channel, got %#v", first)
	}
	if second.Channel != 0 || second.Attached || second.TerminalID != "term-1" || second.ResizeRole != TerminalResizeRoleFollower {
		t.Fatalf("restart should keep follower binding intent but clear old channel, got %#v", second)
	}
	if third.Channel != 9 || !third.Attached || third.TerminalID != "term-2" {
		t.Fatalf("unrelated terminal binding must be preserved, got %#v", third)
	}
}

func TestTerminalViewStoreBeginAttachPreservesCommittedBinding(t *testing.T) {
	store := TerminalViewStore{}.
		BindPane(NewPaneTerminalView("pane-1", "term-old", 7, 80, 24, TerminalResizeRoleOwner, "surface-old", "view-1", true))

	var candidate TerminalAttachCandidate
	store, candidate = store.BeginAttach(TerminalViewBinding{
		ViewID:      "view-1",
		SurfaceID:   "surface-new",
		TerminalID:  "term-new",
		ResizeRole:  TerminalResizeRoleFollower,
		DesiredCols: 100,
		DesiredRows: 30,
		PaneID:      "pane-1",
	})

	binding, ok := store.PaneBinding("pane-1")
	if !ok || !binding.AttachPending || !binding.Attached || binding.Channel != 7 {
		t.Fatalf("pending attach should preserve committed pane attachment, binding=%#v ok=%v", binding, ok)
	}
	if binding.TerminalID != "term-old" || binding.AttachCandidate == nil || binding.AttachCandidate.TerminalID != "term-new" || binding.AttachCandidate.SurfaceID != "surface-new" || binding.AttachCandidate.DesiredCols != 100 || binding.AttachCandidate.DesiredRows != 30 {
		t.Fatalf("pending attach should isolate candidate identity and size, got %#v", binding)
	}

	store, _ = store.FailAttach("view-1", candidate.OperationID, "attach failed")
	binding, _ = store.PaneBinding("pane-1")
	if binding.AttachPending || binding.LastError != "" || !binding.Attached || binding.Channel != 7 {
		t.Fatalf("candidate failure should preserve committed binding without poisoning it, got %#v", binding)
	}
}

func TestTerminalViewStoreTransfersResizeOwnerWithinTerminal(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))
	store = store.BindPane(NewPaneTerminalView("pane-3", "term-2", 9, 100, 30, TerminalResizeRoleOwner, "surface", "view-3", true))

	store = store.TransferPaneResizeOwner("pane-2")
	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	third, _ := store.PaneBinding("pane-3")
	if first.ResizeRole != TerminalResizeRoleFollower || first.CanResize {
		t.Fatalf("previous owner should become follower, got %#v", first)
	}
	if second.ResizeRole != TerminalResizeRoleOwner || !second.CanResize {
		t.Fatalf("target pane should become owner, got %#v", second)
	}
	if !second.ResizePending {
		t.Fatalf("target pane should force one owner resize check after transfer, got %#v", second)
	}
	if third.ResizeRole != TerminalResizeRoleOwner || !third.CanResize {
		t.Fatalf("unrelated terminal owner should be preserved, got %#v", third)
	}
}

func TestTerminalViewStoreScopesResizeOwnerByEndpoint(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewEndpointPaneTerminalView(DefaultEndpointID, "pane-local-owner", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-local-owner", true))
	store = store.BindPane(NewEndpointPaneTerminalView(DefaultEndpointID, "pane-local-follower", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-local-follower", false))
	store = store.BindPane(NewEndpointPaneTerminalView("west", "pane-west-owner", "term-1", 9, 100, 30, TerminalResizeRoleOwner, "surface", "view-west-owner", true))
	store = store.BindPane(NewEndpointPaneTerminalView("west", "pane-west-follower", "term-1", 10, 60, 20, TerminalResizeRoleFollower, "surface", "view-west-follower", false))

	store = store.TransferPaneResizeOwner("pane-west-follower")
	localOwner, _ := store.PaneBinding("pane-local-owner")
	localFollower, _ := store.PaneBinding("pane-local-follower")
	westOwner, _ := store.PaneBinding("pane-west-owner")
	westFollower, _ := store.PaneBinding("pane-west-follower")
	if !localOwner.HasResizeOwner() || !localOwner.CanResize {
		t.Fatalf("local owner must survive west endpoint owner transfer, got %#v", localOwner)
	}
	if localFollower.HasResizeOwner() || localFollower.CanResize {
		t.Fatalf("local follower must not be changed by west endpoint owner transfer, got %#v", localFollower)
	}
	if westOwner.HasResizeOwner() || westOwner.CanResize {
		t.Fatalf("previous west owner should be demoted, got %#v", westOwner)
	}
	if !westFollower.HasResizeOwner() || !westFollower.CanResize || !westFollower.ResizePending {
		t.Fatalf("west follower should become pending owner, got %#v", westFollower)
	}

	store = store.RemoveTerminalRef(NewTerminalRef("west", "term-1"))
	if bindings := store.BindingsForTerminalRef(NewTerminalRef("west", "term-1")); len(bindings) != 0 {
		t.Fatalf("west terminal removal should remove only west bindings, got %#v", bindings)
	}
	if bindings := store.BindingsForTerminalRef(LocalTerminalRef("term-1")); len(bindings) != 2 {
		t.Fatalf("local bindings for same terminal id must survive west removal, got %#v", bindings)
	}
}

func TestTerminalViewStoreMarksWorkbenchBindingsUnresolvedByEndpointStatus(t *testing.T) {
	store := TerminalViewStore{}.
		BindPane(NewEndpointPaneTerminalView("west", "pane-west", "term-1", 0, 80, 24, TerminalResizeRoleFollower, "", TerminalPaneViewID("pane-west"), false)).
		BindPane(NewEndpointPaneTerminalView("disabled", "pane-disabled", "term-1", 0, 80, 24, TerminalResizeRoleFollower, "", TerminalPaneViewID("pane-disabled"), false)).
		BindPane(NewPaneTerminalView("pane-local", "term-1", 0, 80, 24, TerminalResizeRoleFollower, "", TerminalPaneViewID("pane-local"), false))
	endpoints := EndpointStore{}.Upsert(DefaultLocalEndpoint()).Upsert(EndpointItem{ID: "disabled", Enabled: false, Status: EndpointStatusDisabled})

	resolved := store.ApplyWorkbenchEndpointResolution(endpoints)

	if binding, ok := resolved.PaneBinding("pane-west"); !ok || !binding.Unresolved || binding.UnresolvedReason != string(EndpointStatusUnregistered) {
		t.Fatalf("missing endpoint should preserve unresolved binding, binding=%#v ok=%v", binding, ok)
	}
	if binding, ok := resolved.PaneBinding("pane-disabled"); !ok || !binding.Unresolved || binding.UnresolvedReason != string(EndpointStatusDisabled) {
		t.Fatalf("disabled endpoint should preserve unresolved binding, binding=%#v ok=%v", binding, ok)
	}
	if binding, ok := resolved.PaneBinding("pane-local"); !ok || binding.Unresolved {
		t.Fatalf("local endpoint should stay resolved, binding=%#v ok=%v", binding, ok)
	}
}

func TestTerminalViewStoreMarksEndpointRuntimeError(t *testing.T) {
	store := TerminalViewStore{}.
		BindPane(NewEndpointPaneTerminalView("west", "pane-west", "term-1", 9, 80, 24, TerminalResizeRoleOwner, "surface", TerminalPaneViewID("pane-west"), true)).
		BindPane(NewPaneTerminalView("pane-local", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", TerminalPaneViewID("pane-local"), true))
	store, _ = store.RequestPaneResize("pane-west", 100, 30)

	next := store.MarkEndpointRuntimeError("west", "transport-closed: ssh transport closed")

	west, _ := next.PaneBinding("pane-west")
	if west.Channel != 0 || west.Attached || west.CanResize || west.AttachPending || west.ResizePending || west.LastError == "" || west.ResizeRole != TerminalResizeRoleFollower || west.OwnerViewID != "" || west.OwnerSurfaceID != "" || west.ResizeEpoch != 0 {
		t.Fatalf("west binding should be marked offline without removing intent, got %#v", west)
	}
	local, _ := next.PaneBinding("pane-local")
	if local.Channel != 7 || !local.Attached || local.LastError != "" {
		t.Fatalf("local binding should survive west endpoint error, got %#v", local)
	}
	next = next.ApplyTerminalRefResizeControl(NewTerminalRef("west", "term-1"), TerminalResizeControlProjection{
		OwnerSurfaceID: west.SurfaceID, OwnerViewID: west.ViewID, ResizeEpoch: 8,
	})
	west, _ = next.PaneBinding("pane-west")
	if west.HasProjectedResizeOwner() || west.CanResize || west.ResizeRole != TerminalResizeRoleFollower {
		t.Fatalf("daemon events must not grant owner to a disconnected binding, got %#v", west)
	}
}

func TestTerminalViewStoreMarksSingleViewRuntimeError(t *testing.T) {
	store := TerminalViewStore{}.
		BindPane(NewEndpointPaneTerminalView("west", "pane-west", "term-1", 9, 80, 24, TerminalResizeRoleOwner, "surface", TerminalPaneViewID("pane-west"), true)).
		BindPane(NewEndpointPaneTerminalView("west", "pane-logs", "term-1", 10, 80, 24, TerminalResizeRoleFollower, "surface", TerminalPaneViewID("pane-logs"), false))

	next := store.MarkViewRuntimeError(TerminalPaneViewID("pane-west"), "remote-daemon: daemon socket closed")

	west, _ := next.PaneBinding("pane-west")
	if west.Channel != 0 || west.Attached || west.CanResize || west.LastError == "" || west.TerminalID != "term-1" {
		t.Fatalf("target view should keep intent and show runtime error, got %#v", west)
	}
	logs, _ := next.PaneBinding("pane-logs")
	if logs.Channel != 10 || !logs.Attached || logs.LastError != "" {
		t.Fatalf("other view should stay untouched, got %#v", logs)
	}
}

func TestTerminalViewStoreWaitsForDaemonOwnerAfterClosingOwner(t *testing.T) {
	store := TerminalViewStore{}
	owner := NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true)
	owner.OwnerSurfaceID = "surface"
	owner.OwnerViewID = "view-1"
	follower := NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface", "view-2", false)
	follower.OwnerSurfaceID = "surface"
	follower.OwnerViewID = "view-1"
	store = store.BindPane(owner)
	store = store.BindPane(follower)

	store = store.DetachPane("pane-1")
	next, _ := store.PaneBinding("pane-2")
	if next.HasResizeOwner() || next.CanResize || next.ResizePending || next.ResizeRole != TerminalResizeRoleFollower {
		t.Fatalf("replacement view must remain follower until daemon projection, got %#v", next)
	}
	if next.OwnerSurfaceID != "" || next.OwnerViewID != "" {
		t.Fatalf("replacement owner must not keep closed owner identity, got %#v", next)
	}
}

func TestTerminalViewStoreNeverPromotesReplacementOwnerLocally(t *testing.T) {
	store := TerminalViewStore{}
	owner := NewPaneTerminalView("pane-owner", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-owner", true)
	owner.OwnerSurfaceID = "surface"
	owner.OwnerViewID = "view-owner"
	store = store.BindPane(owner)
	store = store.BindPane(NewPaneTerminalView("pane-b", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface", "view-b", false))
	store = store.BindPane(NewPaneTerminalView("pane-a", "term-1", 9, 80, 24, TerminalResizeRoleFollower, "surface", "view-a", false))
	store = store.BindPane(NewPaneTerminalView("pane-disconnected", "term-1", 0, 80, 24, TerminalResizeRoleFollower, "surface", "view-0", false))

	store = store.DetachPane("pane-owner")
	first, _ := store.PaneBinding("pane-a")
	second, _ := store.PaneBinding("pane-b")
	disconnected, _ := store.PaneBinding("pane-disconnected")
	if first.HasResizeOwner() || first.CanResize || first.ResizePending || second.HasResizeOwner() || second.CanResize || second.ResizePending || disconnected.HasResizeOwner() || disconnected.CanResize {
		t.Fatalf("remaining views must await daemon arbitration, first=%#v second=%#v disconnected=%#v", first, second, disconnected)
	}
}

func TestTerminalViewStoreDemotesPreviousOwnerOnBind(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleOwner, "surface", "view-2", true))
	store = store.BindPane(NewPaneTerminalView("pane-3", "term-1", 9, 30, 10, TerminalResizeRoleFollower, "surface", "view-3", false))

	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	third, _ := store.PaneBinding("pane-3")
	if first.ResizeRole != TerminalResizeRoleFollower || first.CanResize {
		t.Fatalf("previous owner should be demoted by new owner bind, got %#v", first)
	}
	if second.ResizeRole != TerminalResizeRoleOwner || !second.CanResize {
		t.Fatalf("new owner should stay owner, got %#v", second)
	}
	if third.ResizeRole != TerminalResizeRoleFollower || third.CanResize {
		t.Fatalf("follower bind should not become owner, got %#v", third)
	}
}

func TestTerminalViewStoreDoesNotPromoteOwnerRoleToResizePermission(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", false))

	binding, _ := store.PaneBinding("pane-1")
	if binding.ResizeRole != TerminalResizeRoleOwner || binding.CanResize || !binding.HasResizeOwner() || binding.HasAuthoritativeResizeOwner() {
		t.Fatalf("locked owner should remain owner without resize permission, got %#v", binding)
	}
}

func TestTerminalViewBindingResizeOwnerRequiresMatchingSurfaceAndView(t *testing.T) {
	binding := NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface-a", "view-1", true)
	binding.OwnerSurfaceID = "surface-b"
	binding.OwnerViewID = "view-1"
	if binding.HasResizeOwner() || binding.HasAuthoritativeResizeOwner() {
		t.Fatalf("owner from another runtime surface must not be local owner, binding=%#v", binding)
	}
	binding.OwnerSurfaceID = "surface-a"
	if !binding.HasResizeOwner() || !binding.HasAuthoritativeResizeOwner() {
		t.Fatalf("matching surface+view should be local owner, binding=%#v", binding)
	}
	binding.OwnerViewID = "view-2"
	if binding.HasResizeOwner() || binding.HasAuthoritativeResizeOwner() {
		t.Fatalf("matching surface with different view should not be local owner, binding=%#v", binding)
	}
	binding.OwnerViewID = "view-1"
	binding.OwnerSurfaceID = ""
	if binding.HasResizeOwner() || binding.HasAuthoritativeResizeOwner() {
		t.Fatalf("core owner projection with view but no surface must not be treated as local owner, binding=%#v", binding)
	}
}

func TestTerminalViewStoreAppliesTerminalResizeControlBySurfaceAndView(t *testing.T) {
	projection := TerminalResizeControlProjection{
		OwnerSurfaceID: "surface-b",
		OwnerViewID:    "pane:main",
		ResizeEpoch:    3,
	}
	firstStore := TerminalViewStore{}.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface-a", "pane:main", true))
	firstStore = firstStore.ApplyTerminalResizeControl("term-1", projection)
	first, _ := firstStore.PaneBinding("pane-1")
	if first.HasResizeOwner() || first.CanResize || first.ResizeRole != TerminalResizeRoleFollower {
		t.Fatalf("same panel id in another TUI surface must be follower, got %#v", first)
	}

	secondStore := TerminalViewStore{}.BindPane(NewPaneTerminalView("pane-1", "term-1", 8, 80, 24, TerminalResizeRoleOwner, "surface-b", "pane:main", true))
	secondStore = secondStore.ApplyTerminalResizeControl("term-1", projection)
	second, _ := secondStore.PaneBinding("pane-1")
	if !second.HasResizeOwner() || !second.CanResize || second.ResizeRole != TerminalResizeRoleOwner {
		t.Fatalf("matching surface+panel should be owner, got %#v", second)
	}
}

func TestTerminalViewStoreIgnoresStaleTerminalResizeControlDuringPendingOwner(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface-a", "view-owner", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface-a", "view-next", false))
	store = store.TransferPaneResizeOwner("pane-2")

	store = store.ApplyTerminalResizeControl("term-1", TerminalResizeControlProjection{
		OwnerSurfaceID: "surface-a",
		OwnerViewID:    "view-owner",
		ResizeEpoch:    1,
	})
	next, _ := store.PaneBinding("pane-2")
	if !next.HasResizeOwner() || !next.ResizePending || !next.CanResize {
		t.Fatalf("stale old-owner projection must not clear pending local owner, got %#v", next)
	}
}

func TestTerminalViewStoreIgnoresLocalOldOwnerProjectionAfterTransfer(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface-a", "view-owner", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface-a", "view-next", false))
	store = store.TransferPaneResizeOwner("pane-2")
	store, _ = store.RequestPaneResize("pane-2", 80, 24)

	store = store.ApplyTerminalResizeControl("term-1", TerminalResizeControlProjection{
		OwnerSurfaceID: "surface-a",
		OwnerViewID:    "view-owner",
		ResizeEpoch:    99,
	})
	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	if first.HasResizeOwner() || first.CanResize {
		t.Fatalf("old local owner projection must not re-promote previous owner, got %#v", first)
	}
	if !second.HasResizeOwner() || !second.CanResize {
		t.Fatalf("current local owner must survive old-owner projection, got %#v", second)
	}
}

func TestTerminalViewStoreResizeControlDemotesPreviousOwner(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	store, applied := store.ApplyResizeControl("view-2", TerminalResizeControlProjection{ResizeRole: TerminalResizeRoleOwner, CanResize: true})
	if !applied {
		t.Fatal("expected resize control projection to apply")
	}
	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	if first.ResizeRole != TerminalResizeRoleFollower || first.CanResize {
		t.Fatalf("previous owner should be demoted by control projection, got %#v", first)
	}
	if second.ResizeRole != TerminalResizeRoleOwner || !second.CanResize {
		t.Fatalf("projected owner should become owner, got %#v", second)
	}
}

func TestTerminalViewStoreResizeControlDoesNotDemoteForStaleOwnerProjection(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	store, applied := store.ApplyResizeControl("view-2", TerminalResizeControlProjection{ResizeRole: TerminalResizeRoleOwner, CanResize: true, OwnerViewID: "view-1"})
	if !applied {
		t.Fatal("expected resize control projection to apply")
	}
	first, _ := store.PaneBinding("pane-1")
	second, _ := store.PaneBinding("pane-2")
	if first.ResizeRole != TerminalResizeRoleOwner || !first.CanResize {
		t.Fatalf("stale owner projection must not demote current owner, got %#v", first)
	}
	if second.ResizeRole != TerminalResizeRoleOwner || !second.CanResize || second.OwnerViewID != "view-1" {
		t.Fatalf("projected binding should retain control metadata without becoming authoritative, got %#v", second)
	}
}

func TestTerminalViewStoreResizeRequestRequiresOwnerAndGuardsStaleResults(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	unchanged, follower := store.RequestPaneResize("pane-2", 50, 14)
	if follower.Allowed || follower.Changed || follower.Reason != "not-owner" {
		t.Fatalf("follower must not request PTY resize, decision=%#v", follower)
	}
	if got, _ := unchanged.PaneBinding("pane-2"); got.DesiredCols != 40 || got.DesiredRows != 12 || got.RequestSeq != 0 {
		t.Fatalf("follower resize must not mutate binding, got %#v", got)
	}

	next, owner := store.RequestPaneResize("pane-1", 100, 30)
	if !owner.Allowed || !owner.Changed || owner.Seq != 1 {
		t.Fatalf("owner resize should advance request seq, decision=%#v", owner)
	}
	stale, applied := next.ApplyResizeResult("view-1", 0, 90, 20, "")
	if !applied {
		t.Fatalf("zero seq bootstrap result should apply")
	}
	current, _ := stale.PaneBinding("pane-1")
	if current.DesiredCols != 90 || current.DesiredRows != 20 {
		t.Fatalf("bootstrap result should update desired size, got %#v", current)
	}
	stale, newer := stale.RequestPaneResize("pane-1", 110, 32)
	if !newer.Changed || newer.Seq != 2 {
		t.Fatalf("newer owner resize should advance pending seq, decision=%#v", newer)
	}
	stale, applied = stale.ApplyResizeResult("view-1", 1, 80, 24, "")
	if applied {
		t.Fatalf("old result should be stale after newer request")
	}
	current, _ = stale.PaneBinding("pane-1")
	if current.DesiredCols != 110 || current.DesiredRows != 32 {
		t.Fatalf("stale result must not overwrite desired size, got %#v", current)
	}
}

func TestTerminalViewStoreResizeRequestRequiresAttachedOwner(t *testing.T) {
	binding := NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true)
	binding.Channel = 0
	store := (TerminalViewStore{}).BindPane(binding)

	next, decision := store.RequestPaneResize("pane-1", 100, 30)
	if decision.Allowed || decision.Changed || decision.Reason != "not-attached" {
		t.Fatalf("detached owner must not request PTY resize, decision=%#v", decision)
	}
	current, _ := next.PaneBinding("pane-1")
	if current.DesiredCols != 80 || current.DesiredRows != 24 || current.RequestSeq != 0 {
		t.Fatalf("detached resize must not mutate desired geometry, binding=%#v", current)
	}
}

func TestTerminalViewStoreOwnerTransferForcesSameSizeResizeOnce(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface", "view-2", false))
	store = store.TransferPaneResizeOwner("pane-2")

	next, decision := store.RequestPaneResize("pane-2", 80, 24)
	if !decision.Allowed || !decision.Changed || decision.Seq != 1 {
		t.Fatalf("new owner must issue one same-size resize check, decision=%#v", decision)
	}
	binding, _ := next.PaneBinding("pane-2")
	if binding.ResizePending || binding.RequestSeq != 1 || binding.DesiredCols != 80 || binding.DesiredRows != 24 {
		t.Fatalf("resize check should clear pending and keep desired size, got %#v", binding)
	}
	_, duplicate := next.RequestPaneResize("pane-2", 80, 24)
	if !duplicate.Allowed || duplicate.Changed || duplicate.Reason != "unchanged" {
		t.Fatalf("same-size resize after pending check must dedupe, decision=%#v", duplicate)
	}
}

func TestTerminalViewLayoutStateIsViewLocal(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, TerminalResizeRoleFollower, "surface", "view-2", false))

	next, changed, ok := store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "pan", DeltaX: 4, DeltaY: -2})
	if !ok {
		t.Fatal("expected pane layout command to apply")
	}
	if changed.Layout.PanX != 4 || changed.Layout.PanY != -2 {
		t.Fatalf("expected pane-1 pan to change, got %#v", changed.Layout)
	}
	sibling, _ := next.PaneBinding("pane-2")
	if sibling.Layout.Normalize().PanX != 0 || sibling.Layout.Normalize().PanY != 0 {
		t.Fatalf("sibling binding for same terminal must keep independent layout, got %#v", sibling.Layout)
	}
}

func TestTerminalViewLayoutCommandsNormalizeAndReset(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))

	store, binding, ok := store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "toggle-lock"})
	if !ok || !binding.Layout.SizeLocked || binding.Layout.Mode != TerminalViewLayoutAuto {
		t.Fatalf("lock should toggle on default normalized layout, binding=%#v ok=%v", binding, ok)
	}
	store, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "toggle-layout"})
	if binding.Layout.Mode != TerminalViewLayoutFit {
		t.Fatalf("layout toggle should move auto to fit, got %#v", binding.Layout)
	}
	store, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "center"})
	if binding.Layout.Mode != TerminalViewLayoutCenter || binding.Layout.AlignX != TerminalViewAlignCenter || binding.Layout.AlignY != TerminalViewAlignCenter {
		t.Fatalf("center should set centered layout, got %#v", binding.Layout)
	}
	store, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "align", AlignX: TerminalViewAlignStart})
	if binding.Layout.Mode != TerminalViewLayoutAuto || binding.Layout.AlignX != TerminalViewAlignStart || binding.Layout.AlignY != TerminalViewAlignCenter {
		t.Fatalf("align should override full-center mode without reset, got %#v", binding.Layout)
	}
	store, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "center-x"})
	if binding.Layout.Mode != TerminalViewLayoutAuto || binding.Layout.AlignX != TerminalViewAlignCenter || binding.Layout.AlignY != TerminalViewAlignStart {
		t.Fatalf("center-x should express horizontal-only center, got %#v", binding.Layout)
	}
	store, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "center-y"})
	if binding.Layout.Mode != TerminalViewLayoutAuto || binding.Layout.AlignX != TerminalViewAlignStart || binding.Layout.AlignY != TerminalViewAlignCenter {
		t.Fatalf("center-y should express vertical-only center, got %#v", binding.Layout)
	}
	_, binding, _ = store.ApplyPaneLayoutCommand("pane-1", TerminalViewLayoutCommand{Action: "reset"})
	if binding.Layout.SizeLocked || binding.Layout.Mode != TerminalViewLayoutAuto || binding.Layout.AlignX != TerminalViewAlignStart || binding.Layout.AlignY != TerminalViewAlignStart {
		t.Fatalf("reset should restore normalized default layout, got %#v", binding.Layout)
	}
}

func TestTerminalViewApplyTerminalSizeLockKeepsOwnerIdentityAndBlocksResize(t *testing.T) {
	store := TerminalViewStore{}
	store = store.BindPane(NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, TerminalResizeRoleOwner, "surface", "view-1", true))
	store = store.BindPane(NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, TerminalResizeRoleFollower, "surface", "view-2", false))

	store = store.ApplyTerminalSizeLock("term-1", true)
	owner, _ := store.PaneBinding("pane-1")
	follower, _ := store.PaneBinding("pane-2")
	if owner.ResizeRole != TerminalResizeRoleOwner || !owner.HasResizeOwner() || owner.HasAuthoritativeResizeOwner() || !owner.SizeLocked || owner.ControlReason != "size_locked" {
		t.Fatalf("locked owner should keep owner identity without resize authority, got %#v", owner)
	}
	if follower.HasResizeOwner() || follower.CanResize || !follower.SizeLocked || follower.ControlReason != "size_locked" {
		t.Fatalf("follower should inherit terminal lock without owner identity, got %#v", follower)
	}
	_, decision := store.RequestPaneResize("pane-1", 100, 30)
	if decision.Allowed || decision.Reason != "size-locked" {
		t.Fatalf("locked owner must not allow resize requests, got %#v", decision)
	}

	store = store.BindPane(NewPaneTerminalView("pane-3", "term-1", 9, 80, 24, TerminalResizeRoleOwner, "surface", "view-3", true))
	store = store.TransferPaneResizeOwner("pane-3")
	newOwner, _ := store.PaneBinding("pane-3")
	if newOwner.ResizeRole != TerminalResizeRoleOwner || newOwner.CanResize || !newOwner.SizeLocked || newOwner.ControlReason != "size_locked" {
		t.Fatalf("owner transfer on locked terminal must preserve lock and block resize, got %#v", newOwner)
	}
	oldOwner, _ := store.PaneBinding("pane-1")
	if oldOwner.SizeLocked != true || oldOwner.CanResize {
		t.Fatalf("previous view should stay locked after owner transfer, got %#v", oldOwner)
	}

	store = store.ApplyTerminalSizeLock("term-1", false)
	owner, _ = store.PaneBinding("pane-3")
	follower, _ = store.PaneBinding("pane-2")
	if owner.SizeLocked || !owner.CanResize || owner.ControlReason != "" {
		t.Fatalf("unlock should restore owner resize authority, got %#v", owner)
	}
	if follower.SizeLocked || follower.CanResize || follower.ControlReason != "" {
		t.Fatalf("unlock should clear follower lock projection without granting authority, got %#v", follower)
	}
}
