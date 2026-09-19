package provider_test

import (
	"context"
	"testing"

	"github.com/anytty/anytty/daemon/provider"
	"github.com/anytty/anytty/internal/providerproto"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/terminalmeta"
)

func dialOwnershipClient(t *testing.T, socketPath string) *provider.Client {
	t.Helper()
	client, err := provider.Dial(context.Background(), socketPath)
	if err != nil {
		t.Fatalf("dial provider: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func attachForOwnershipTest(t *testing.T, client *provider.Client, terminalID, surfaceID, viewID string, mode providerv1.AttachmentMode, policy providerv1.ResizePolicy) *providerv1.TerminalAttachResult {
	t.Helper()
	attached, err := client.Attach(context.Background(), &providerv1.TerminalAttachCommand{
		Terminal:     &providerv1.TerminalRef{TerminalId: terminalID},
		Mode:         mode,
		ResizePolicy: policy,
		SurfaceId:    surfaceID,
		ViewId:       viewID,
	})
	if err != nil {
		t.Fatalf("attach %s/%s: %v", surfaceID, viewID, err)
	}
	return attached
}

// TestProviderResizeOwnershipTransferAndEpochFence 迁移自旧 daemon protocol 的
// TestProtocolResizeSeparatesOwnerTransferFromOrdinaryResize 与 stale epoch 拒绝。
func TestProviderResizeOwnershipTransferAndEpochFence(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	ownerClient := dialOwnershipClient(t, socketPath)
	followerClient := dialOwnershipClient(t, socketPath)
	staleClient := dialOwnershipClient(t, socketPath)

	if _, err := ownerClient.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-arbitration", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	owner := attachForOwnershipTest(t, ownerClient, "term-arbitration", "owner-surface", "owner-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_OWNER)
	initialEpoch := owner.GetResizeControl().GetResizeOwnership().GetEpoch()
	if initialEpoch == 0 || !owner.GetResizeControl().GetCanResize() {
		t.Fatalf("owner attach control = %#v", owner.GetResizeControl())
	}

	ordinary, err := ownerClient.Resize(ctx, owner.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 90, Rows: 25},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, false, initialEpoch)
	if err != nil || !ordinary.GetResized() {
		t.Fatalf("ordinary resize = %#v err=%v", ordinary, err)
	}
	if epoch := ordinary.GetResizeControl().GetResizeOwnership().GetEpoch(); epoch != initialEpoch {
		t.Fatalf("ordinary resize changed ownership epoch: %d want %d", epoch, initialEpoch)
	}

	follower := attachForOwnershipTest(t, followerClient, "term-arbitration", "follower-surface", "follower-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	taken, err := followerClient.Resize(ctx, follower.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 100, Rows: 30},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, true, initialEpoch)
	if err != nil || !taken.GetResized() {
		t.Fatalf("owner transfer resize = %#v err=%v", taken, err)
	}
	takenEpoch := taken.GetResizeControl().GetResizeOwnership().GetEpoch()
	if takenEpoch != initialEpoch+1 || taken.GetResizeControl().GetResizeOwnership().GetOwnerViewId() != "follower-view" {
		t.Fatalf("owner transfer control = %#v", taken.GetResizeControl())
	}

	late, err := ownerClient.Resize(ctx, owner.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 110, Rows: 35},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, false, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if late.GetResized() || late.GetResizeControl().GetCanResize() ||
		late.GetResizeControl().GetResizeOwnership().GetEpoch() != takenEpoch ||
		late.GetResizeControl().GetResizeOwnership().GetOwnerViewId() != "follower-view" {
		t.Fatalf("late ordinary resize stole ownership: %#v", late.GetResizeControl())
	}

	stale := attachForOwnershipTest(t, staleClient, "term-arbitration", "stale-surface", "stale-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	staleTake, err := staleClient.Resize(ctx, stale.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 120, Rows: 40},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, true, initialEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if staleTake.GetResized() || staleTake.GetResizeControl().GetCanResize() ||
		staleTake.GetResizeControl().GetResizeOwnership().GetEpoch() != takenEpoch ||
		staleTake.GetResizeControl().GetResizeOwnership().GetOwnerViewId() != "follower-view" {
		t.Fatalf("stale epoch takeover was accepted: %#v", staleTake.GetResizeControl())
	}
}

// TestProviderAttachmentPromotionAndOwnerlessRelease 迁移自旧 protocol 的
// owner detach 晋升顺序、release-without-successor 与 observer 规则。
func TestProviderAttachmentPromotionAndOwnerlessRelease(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	ownerClient := dialOwnershipClient(t, socketPath)
	firstFollower := dialOwnershipClient(t, socketPath)
	secondFollower := dialOwnershipClient(t, socketPath)

	if _, err := ownerClient.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-promotion", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	owner := attachForOwnershipTest(t, ownerClient, "term-promotion", "owner-surface", "owner-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_OWNER)
	initialEpoch := owner.GetResizeControl().GetResizeOwnership().GetEpoch()
	_ = attachForOwnershipTest(t, firstFollower, "term-promotion", "follower-b-surface", "follower-b-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	second := attachForOwnershipTest(t, secondFollower, "term-promotion", "follower-c-surface", "follower-c-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)

	if err := ownerClient.Detach(ctx, owner.GetAttachment().GetOpaqueToken()); err != nil {
		t.Fatal(err)
	}
	promoted, err := secondFollower.Resize(ctx, second.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 40, Rows: 10},
		providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	ownership := promoted.GetResizeControl().GetResizeOwnership()
	if ownership == nil || ownership.GetOwnerViewId() != "follower-b-view" || ownership.GetEpoch() <= initialEpoch {
		t.Fatalf("owner detach did not promote the first waiting follower: %#v", promoted.GetResizeControl())
	}
	if second.GetAttachment().GetOpaqueToken() == nil {
		t.Fatal("second follower token missing")
	}

	// ownerless terminal: observer never becomes owner and single collaborator
	// can release ownership without a successor.
	if _, err := ownerClient.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-ownerless", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	observerClient := dialOwnershipClient(t, socketPath)
	if _, err := observerClient.Attach(ctx, &providerv1.TerminalAttachCommand{
		Terminal:     &providerv1.TerminalRef{TerminalId: "term-ownerless"},
		Mode:         providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER,
		ResizePolicy: providerv1.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "observer-surface", ViewId: "observer-view",
	}); providerCode(err) != providerproto.ErrorBadRequest {
		t.Fatalf("observer owner attach err = %v, want bad request", err)
	}
	observer := attachForOwnershipTest(t, observerClient, "term-ownerless", "observer-surface", "observer-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER, providerv1.ResizePolicy_RESIZE_POLICY_OBSERVER)
	if observer.GetResizeControl().GetResizeOwnership() != nil ||
		observer.GetResizeControl().GetReason() != providerv1.ResizeReason_RESIZE_REASON_OBSERVER {
		t.Fatalf("observer attach control = %#v", observer.GetResizeControl())
	}
	if _, err := observerClient.Resize(ctx, observer.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 50, Rows: 12},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, true, 0); providerCode(err) != providerproto.ErrorForbidden {
		t.Fatalf("observer resize err = %v, want forbidden", err)
	}
	info, err := observerClient.Get(ctx, "term-ownerless")
	if err != nil || info.GetAttachmentCount() != 1 {
		t.Fatalf("observer projected ownership into registry: %#v err=%v", info, err)
	}

	solo := attachForOwnershipTest(t, ownerClient, "term-ownerless", "solo-surface", "solo-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_OWNER)
	epoch := solo.GetResizeControl().GetResizeOwnership().GetEpoch()
	released, err := ownerClient.Resize(ctx, solo.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 40, Rows: 10},
		providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER, false, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if released.GetResizeControl().GetCanResize() || released.GetResizeControl().GetResizeOwnership() != nil {
		t.Fatalf("released attachment kept ownership: %#v", released.GetResizeControl())
	}
	if released.GetResizeControl().GetReason() != providerv1.ResizeReason_RESIZE_REASON_FOLLOWER {
		t.Fatalf("released attachment reason = %v", released.GetResizeControl().GetReason())
	}
}

// TestProviderAttachmentSizeLockAndViewCount 迁移自旧 protocol 的
// persisted size lock、size lock 阻断 owner transfer，以及 attachment view 计数。
func TestProviderAttachmentSizeLockAndViewCount(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	ownerClient := dialOwnershipClient(t, socketPath)
	followerClient := dialOwnershipClient(t, socketPath)

	if _, err := ownerClient.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-locked", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
		Tags: map[string]string{terminalmeta.SizeLockTag: terminalmeta.SizeLockLock},
	}); err != nil {
		t.Fatal(err)
	}
	owner := attachForOwnershipTest(t, ownerClient, "term-locked", "owner-surface", "owner-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_OWNER)
	if control := owner.GetResizeControl(); !control.GetSizeLocked() || control.GetCanResize() ||
		control.GetReason() != providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED {
		t.Fatalf("persisted size lock was not restored: %#v", control)
	}
	lockedEpoch := owner.GetResizeControl().GetResizeOwnership().GetEpoch()

	follower := attachForOwnershipTest(t, followerClient, "term-locked", "follower-surface", "follower-view",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	taken, err := followerClient.Resize(ctx, follower.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 120, Rows: 40},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, true, lockedEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if taken.GetResized() || !taken.GetResizeControl().GetSizeLocked() ||
		taken.GetResizeControl().GetReason() != providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED ||
		taken.GetResizeControl().GetResizeOwnership().GetOwnerViewId() != "follower-view" {
		t.Fatalf("locked owner transfer changed size or lost lock: %#v", taken.GetResizeControl())
	}

	unlocked, err := followerClient.ResizeLock(ctx, follower.GetAttachment().GetOpaqueToken(), false)
	if err != nil {
		t.Fatal(err)
	}
	if unlocked.GetResizeControl().GetSizeLocked() || !unlocked.GetResizeControl().GetCanResize() ||
		unlocked.GetResizeControl().GetReason() != providerv1.ResizeReason_RESIZE_REASON_OWNER {
		t.Fatalf("new owner could not unlock size: %#v", unlocked.GetResizeControl())
	}
	resized, err := followerClient.Resize(ctx, follower.GetAttachment().GetOpaqueToken(), &providerv1.Size{Cols: 120, Rows: 40},
		providerv1.ResizePolicy_RESIZE_POLICY_OWNER, false, unlocked.GetResizeControl().GetResizeOwnership().GetEpoch())
	if err != nil || !resized.GetResized() {
		t.Fatalf("unlocked owner resize = %#v err=%v", resized, err)
	}

	if _, err := ownerClient.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-views", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	_ = attachForOwnershipTest(t, ownerClient, "term-views", "surface-a", "pane:one",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_OWNER)
	_ = attachForOwnershipTest(t, ownerClient, "term-views", "surface-a", "pane:one",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	_ = attachForOwnershipTest(t, followerClient, "term-views", "surface-a", "pane:one",
		providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER)
	views, err := ownerClient.Get(ctx, "term-views")
	if err != nil || views.GetAttachmentCount() != 2 {
		t.Fatalf("attachment view count = %d err=%v, want 2", views.GetAttachmentCount(), err)
	}

	subscribed, err := followerClient.EventSubscribe(ctx, &providerv1.EventSubscribeCommand{
		TerminalId: "term-views",
		Types:      []providerv1.TerminalEventType{providerv1.TerminalEventType_TERMINAL_EVENT_TYPE_METADATA_CHANGED},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := subscribed.GetInitialEvents()
	if len(initial) != 1 || initial[0].GetAttachment() == nil {
		t.Fatalf("event subscription initial events = %#v", initial)
	}
	projection := initial[0].GetAttachment()
	if projection.GetAttachmentCount() != 2 || projection.GetResizeControl() == nil ||
		projection.GetResizeControl().GetResizeOwnership().GetOwnerViewId() != "pane:one" {
		t.Fatalf("replayed attachment projection = %#v", projection)
	}
	if err := followerClient.EventRelease(ctx, subscribed.GetOpaqueToken()); err != nil {
		t.Fatal(err)
	}
}
