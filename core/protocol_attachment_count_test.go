package core

import (
	"context"
	"testing"
)

func TestProtocolAttachmentCountCountsViewsAcrossAndWithinSessions(t *testing.T) {
	server := NewServer()
	attachments := []protocolAttachment{
		{SessionID: 1, Channel: 1, TerminalID: "term-1", SurfaceID: "tui-1", ViewID: "pane:one"},
		{SessionID: 1, Channel: 2, TerminalID: "term-1", SurfaceID: "tui-1", ViewID: "pane:two"},
		// A different client may legally reuse the same public surface/view IDs.
		{SessionID: 2, Channel: 1, TerminalID: "term-1", SurfaceID: "tui-1", ViewID: "pane:one"},
		// Defensive duplicate registry entries for the same logical view count once.
		{SessionID: 1, Channel: 3, TerminalID: "term-1", SurfaceID: "tui-1", ViewID: "pane:one"},
		{SessionID: 1, Channel: 4, TerminalID: "term-2", SurfaceID: "tui-1", ViewID: "pane:three"},
	}
	for _, attachment := range attachments {
		server.protocolAttachments[attachmentKey(attachment)] = attachment
	}

	if got := server.protocolAttachmentCount("term-1"); got != 3 {
		t.Fatalf("attachment view count = %d, want 3", got)
	}
}

func TestProtocolOwnerPromotionUsesStableAttachmentOrder(t *testing.T) {
	server := NewServer()
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	attachments := []protocolAttachment{
		{SessionID: 3, Channel: 1, TerminalID: "term-1", ResizePolicy: attachmentResizePolicyFollower},
		{SessionID: 1, Channel: 9, TerminalID: "term-1", ResizePolicy: attachmentResizePolicyFollower},
		{SessionID: 1, Channel: 2, TerminalID: "term-1", ResizePolicy: attachmentResizePolicyFollower},
		{SessionID: 0, Channel: 1, TerminalID: "term-1", ResizePolicy: attachmentResizePolicyObserver},
	}
	for _, attachment := range attachments {
		server.protocolAttachments[attachmentKey(attachment)] = attachment
	}

	session.promoteGlobalResizeOwnerLocked("term-1")
	want := attachmentKey(attachments[2])
	if got := server.protocolResizeOwners["term-1"]; got != want {
		t.Fatalf("promoted owner key = %q, want stable lowest collaborator %q", got, want)
	}
	if promoted := server.protocolAttachments[want]; promoted.ResizePolicy != attachmentResizePolicyOwner || promoted.Epoch == 0 {
		t.Fatalf("promoted attachment did not receive owner projection: %#v", promoted)
	}
}

func TestProtocolResizeSeparatesOwnerTransferFromOrdinaryResize(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 4, MaxAttachments: 4})
	ownerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	followerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	staleSession := newProtocolSession(server, nil, fullDaemonTransportScope())

	owner := attachProtocolViewForTest(t, ownerSession, "owner-surface", "owner-view", TerminalResizePolicyOwner)
	follower := attachProtocolViewForTest(t, followerSession, "follower-surface", "follower-view", TerminalResizePolicyFollower)
	stale := attachProtocolViewForTest(t, staleSession, "stale-surface", "stale-view", TerminalResizePolicyFollower)
	initialEpoch := owner.ResizeControl.ResizeOwnership.Epoch

	ordinary, err := ownerSession.ApplicationTerminalResize(context.Background(), owner.Token, Size{Cols: 90, Rows: 25}, TerminalResizePolicyOwner, false, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if !ordinary.Resized || !ordinary.ResizeControl.CanResize || ordinary.ResizeControl.ResizeOwnership.Epoch != initialEpoch {
		t.Fatalf("ordinary owner resize changed ownership: %#v", ordinary)
	}

	taken, err := followerSession.ApplicationTerminalResize(context.Background(), follower.Token, Size{Cols: 100, Rows: 30}, TerminalResizePolicyOwner, true, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	takenEpoch := taken.ResizeControl.ResizeOwnership.Epoch
	if !taken.Resized || !taken.ResizeControl.CanResize || takenEpoch != initialEpoch+1 || taken.ResizeControl.OwnerViewID != "follower-view" {
		t.Fatalf("explicit owner transfer did not commit exactly once: %#v", taken)
	}

	late, err := ownerSession.ApplicationTerminalResize(context.Background(), owner.Token, Size{Cols: 110, Rows: 35}, TerminalResizePolicyOwner, false, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if late.Resized || late.ResizeControl.CanResize || late.ResizeControl.ResizeOwnership.Epoch != takenEpoch || late.ResizeControl.OwnerViewID != "follower-view" {
		t.Fatalf("late ordinary resize stole ownership back: %#v", late)
	}

	staleTake, err := staleSession.ApplicationTerminalResize(context.Background(), stale.Token, Size{Cols: 120, Rows: 40}, TerminalResizePolicyOwner, true, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if staleTake.Resized || staleTake.ResizeControl.CanResize || staleTake.ResizeControl.ResizeOwnership.Epoch != takenEpoch || staleTake.ResizeControl.OwnerViewID != "follower-view" {
		t.Fatalf("stale epoch takeover should be rejected: %#v", staleTake)
	}
}

func TestProtocolObserverCannotRequestResizeOwnership(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	_, err := session.ApplicationTerminalAttach(context.Background(), TerminalAttachmentRequest{
		TerminalID: "term-resource", Mode: TerminalAttachmentModeObserver, ResizePolicy: TerminalResizePolicyOwner,
		SurfaceID: "observer-surface", ViewID: "observer-view",
	})
	if err == nil {
		t.Fatal("observer attachment unexpectedly acquired resize ownership")
	}
}

func attachProtocolViewForTest(t *testing.T, session *protocolSession, surfaceID, viewID string, policy TerminalResizePolicy) TerminalAttachment {
	t.Helper()
	transaction, err := session.ApplicationTerminalAttach(context.Background(), TerminalAttachmentRequest{
		TerminalID: "term-resource", Mode: TerminalAttachmentModeCollaborator, ResizePolicy: policy,
		SurfaceID: surfaceID, ViewID: viewID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	return transaction.Result()
}
