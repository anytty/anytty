package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/terminalmeta"
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

func TestProtocolAttachRestoresPersistedSizeLock(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-locked",
		Command: []string{"shell"},
		Tags:    map[string]string{terminalmeta.SizeLockTag: terminalmeta.SizeLockLock},
	}); err != nil {
		t.Fatal(err)
	}
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	transaction, err := session.ApplicationTerminalAttach(context.Background(), TerminalAttachmentRequest{
		TerminalID:   "term-locked",
		Mode:         TerminalAttachmentModeCollaborator,
		ResizePolicy: TerminalResizePolicyOwner,
		SurfaceID:    "app:studio:terminal:term-locked",
		ViewID:       "primary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	control := transaction.Result().ResizeControl
	if control == nil || control.CanResize || !control.SizeLocked || control.Reason != TerminalResizeReasonSizeLocked {
		t.Fatalf("persisted size lock was not restored for owner attachment: %#v", control)
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

func TestProtocolOwnerDetachPromotesExactlyOneOfTwoFollowers(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 3, MaxAttachments: 3})
	ownerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	followerBSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	followerCSession := newProtocolSession(server, nil, fullDaemonTransportScope())

	owner := attachProtocolViewForTest(t, ownerSession, "owner-surface", "owner-view", TerminalResizePolicyOwner)
	followerB := attachProtocolViewForTest(t, followerBSession, "follower-b-surface", "follower-b-view", TerminalResizePolicyFollower)
	followerC := attachProtocolViewForTest(t, followerCSession, "follower-c-surface", "follower-c-view", TerminalResizePolicyFollower)
	initialEpoch := owner.ResizeControl.ResizeOwnership.Epoch

	if err := ownerSession.ApplicationTerminalDetach(context.Background(), owner.Token); err != nil {
		t.Fatal(err)
	}
	bAttachment, err := followerBSession.attachmentForToken(followerB.Token)
	if err != nil {
		t.Fatal(err)
	}
	cAttachment, err := followerCSession.attachmentForToken(followerC.Token)
	if err != nil {
		t.Fatal(err)
	}
	bKey := attachmentKey(bAttachment)
	cKey := attachmentKey(cAttachment)

	server.protocolAttachmentMu.Lock()
	ownerKey := server.protocolResizeOwners["term-resource"]
	bProjection := server.protocolAttachments[bKey]
	cProjection := server.protocolAttachments[cKey]
	server.protocolAttachmentMu.Unlock()

	if ownerKey != bKey {
		t.Fatalf("owner detach promoted %q, want first waiting follower %q", ownerKey, bKey)
	}
	if bProjection.ResizePolicy != attachmentResizePolicyOwner || bProjection.Epoch <= initialEpoch {
		t.Fatalf("promoted follower did not receive a newer owner epoch: %#v", bProjection)
	}
	if cProjection.ResizePolicy != attachmentResizePolicyFollower || cProjection.Epoch != 0 {
		t.Fatalf("second follower was also promoted: %#v", cProjection)
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

func TestProtocolResizeOwnerCanReleaseToAnotherFollower(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 2, MaxAttachments: 2})
	ownerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	followerSession := newProtocolSession(server, nil, fullDaemonTransportScope())

	owner := attachProtocolViewForTest(t, ownerSession, "owner-surface", "owner-view", TerminalResizePolicyOwner)
	_ = attachProtocolViewForTest(t, followerSession, "follower-surface", "follower-view", TerminalResizePolicyFollower)
	initialEpoch := owner.ResizeControl.ResizeOwnership.Epoch

	released, err := ownerSession.ApplicationTerminalResize(context.Background(), owner.Token, owner.Size, TerminalResizePolicyFollower, false, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if released.ResizeControl.CanResize || released.ResizeControl.Reason != TerminalResizeReasonFollower {
		t.Fatalf("released attachment kept resize ownership: %#v", released.ResizeControl)
	}
	if ownership := released.ResizeControl.ResizeOwnership; ownership == nil || ownership.OwnerViewID != "follower-view" || ownership.Epoch <= initialEpoch {
		t.Fatalf("resize ownership was not handed to the follower: %#v", released.ResizeControl)
	}
}

func TestProtocolResizeOwnerCanReleaseWithoutSuccessor(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	owner := attachProtocolViewForTest(t, session, "owner-surface", "owner-view", TerminalResizePolicyOwner)

	released, err := session.ApplicationTerminalResize(context.Background(), owner.Token, owner.Size, TerminalResizePolicyFollower, false, owner.ResizeControl.ResizeOwnership.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if released.ResizeControl.CanResize || released.ResizeControl.Reason != TerminalResizeReasonFollower || released.ResizeControl.ResizeOwnership != nil {
		t.Fatalf("released attachment should remain a follower without an owner: %#v", released.ResizeControl)
	}
	if ownerKey := server.protocolResizeOwners["term-resource"]; ownerKey != "" {
		t.Fatalf("released terminal retained owner key %q", ownerKey)
	}
	events := session.protocolAttachmentSnapshotEvents(EventFilter{
		TerminalID: "term-resource",
		Types:      []EventType{EventTerminalMetadataChanged},
	})
	if len(events) != 1 || events[0].Attachment == nil {
		t.Fatalf("missing ownerless attachment snapshot: %#v", events)
	}
	projection := events[0].Attachment
	if projection.ResizeControl == nil || projection.ResizeControl.Reason != TerminalResizeReasonFollower || projection.ResizeControl.ResizeOwnership != nil || projection.ResizeEpoch <= owner.ResizeControl.ResizeOwnership.Epoch {
		t.Fatalf("ownerless projection must carry a newer terminal resize epoch: %#v", projection)
	}
}

func TestProtocolObserverDoesNotBecomeOwnerWhenTerminalHasNoOwner(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	observer := attachProtocolViewForTest(t, session, "observer-surface", "observer-view", TerminalResizePolicyObserver)
	if observer.ResizeControl == nil || observer.ResizeControl.ResizeOwnership != nil || observer.ResizeControl.Reason != TerminalResizeReasonObserver {
		t.Fatalf("observer became resize owner: %#v", observer.ResizeControl)
	}
	if ownerKey := server.protocolResizeOwners["term-resource"]; ownerKey != "" {
		t.Fatalf("observer populated resize owner registry: %q", ownerKey)
	}
}

func TestProtocolEventSubscriptionReplaysCurrentAttachmentProjection(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 2, MaxAttachments: 1, MaxEventSubscriptions: 1})
	ownerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	owner := attachProtocolViewForTest(t, ownerSession, "mobile-surface", "mobile-view", TerminalResizePolicyOwner)
	observerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	replayed := make(chan Event, 1)
	token, err := observerSession.ApplicationEventSubscribe(context.Background(), EventFilter{
		TerminalID: "term-resource",
		Types:      []EventType{EventTerminalMetadataChanged},
	}, func(event Event, _ []byte) ([]byte, error) {
		replayed <- event
		return nil, errors.New("capture replay without a transport")
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observerSession.ReleaseApplicationResource(context.Background(), token); err != nil {
			t.Errorf("release event subscription: %v", err)
		}
	}()

	select {
	case event := <-replayed:
		projection := event.Attachment
		if projection == nil || projection.AttachmentCount != 1 || projection.ResizeEpoch != owner.ResizeControl.ResizeOwnership.Epoch || projection.ResizeControl == nil || projection.ResizeControl.OwnerSurfaceID != "mobile-surface" || projection.ResizeControl.OwnerViewID != "mobile-view" {
			t.Fatalf("subscription did not replay current owner projection: %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event subscription did not replay its initial attachment projection")
	}
}

func TestProtocolSizeLockSurvivesOwnerTransferAndBlocksResize(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 2, MaxAttachments: 2})
	ownerSession := newProtocolSession(server, nil, fullDaemonTransportScope())
	followerSession := newProtocolSession(server, nil, fullDaemonTransportScope())

	owner := attachProtocolViewForTest(t, ownerSession, "owner-surface", "owner-view", TerminalResizePolicyOwner)
	follower := attachProtocolViewForTest(t, followerSession, "follower-surface", "follower-view", TerminalResizePolicyFollower)
	locked, err := ownerSession.ApplicationTerminalResizeLock(context.Background(), owner.Token, true)
	if err != nil {
		t.Fatal(err)
	}
	if locked.ResizeControl == nil || !locked.ResizeControl.SizeLocked || locked.ResizeControl.Reason != TerminalResizeReasonSizeLocked {
		t.Fatalf("owner did not lock terminal size: %#v", locked.ResizeControl)
	}

	taken, err := followerSession.ApplicationTerminalResize(context.Background(), follower.Token, Size{Cols: 120, Rows: 40}, TerminalResizePolicyOwner, true, locked.ResizeControl.ResizeOwnership.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if taken.Resized || taken.Size != owner.Size || taken.ResizeControl == nil || !taken.ResizeControl.SizeLocked || taken.ResizeControl.Reason != TerminalResizeReasonSizeLocked || taken.ResizeControl.OwnerViewID != "follower-view" {
		t.Fatalf("locked owner transfer changed size or lost lock: %#v", taken)
	}

	blocked, err := followerSession.ApplicationTerminalResize(context.Background(), follower.Token, Size{Cols: 120, Rows: 40}, TerminalResizePolicyOwner, false, taken.ResizeControl.ResizeOwnership.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Resized || blocked.Size != owner.Size || !blocked.ResizeControl.SizeLocked {
		t.Fatalf("new owner resized a locked terminal: %#v", blocked)
	}

	unlocked, err := followerSession.ApplicationTerminalResizeLock(context.Background(), follower.Token, false)
	if err != nil {
		t.Fatal(err)
	}
	if unlocked.ResizeControl == nil || unlocked.ResizeControl.SizeLocked || !unlocked.ResizeControl.CanResize || unlocked.ResizeControl.Reason != TerminalResizeReasonOwner {
		t.Fatalf("new owner could not unlock terminal size: %#v", unlocked.ResizeControl)
	}
	resized, err := followerSession.ApplicationTerminalResize(context.Background(), follower.Token, Size{Cols: 120, Rows: 40}, TerminalResizePolicyOwner, false, unlocked.ResizeControl.ResizeOwnership.Epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !resized.Resized || resized.Size != (Size{Cols: 120, Rows: 40}) {
		t.Fatalf("unlocked owner could not resize terminal: %#v", resized)
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
