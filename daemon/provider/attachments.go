package provider

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/anytty/anytty/daemon/core"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

const (
	attachmentModeCollaborator = "collaborator"
	attachmentModeObserver     = "observer"

	resizePolicyOwner    = "owner"
	resizePolicyFollower = "follower"
	resizePolicyObserver = "observer"
)

// providerAttachment 是单条 provider session 上的 attachment/stream 投影。
// 它不保存 TUI workspace/pane truth。
type providerAttachment struct {
	sessionID    uint64
	terminalID   string
	channel      uint16
	mode         string
	resizePolicy string
	surfaceID    string
	viewID       string
	epoch        uint64
	token        []byte
}

func attachmentKey(attachment *providerAttachment) string {
	return fmt.Sprintf("%d:%d", attachment.sessionID, attachment.channel)
}

// attachmentRegistry 是 server 级 attachment ownership/epoch 仲裁器。
type attachmentRegistry struct {
	mu        sync.Mutex
	byKey     map[string]*providerAttachment
	owners    map[string]string
	epochs    map[string]uint64
	sizeLocks map[string]bool
	nextEpoch uint64
}

func newAttachmentRegistry() *attachmentRegistry {
	return &attachmentRegistry{
		byKey:     make(map[string]*providerAttachment),
		owners:    make(map[string]string),
		epochs:    make(map[string]uint64),
		sizeLocks: make(map[string]bool),
	}
}

func (registry *attachmentRegistry) register(attachment *providerAttachment, size core.Size, initialSizeLocked bool) *providerv1.ResizeControl {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := attachmentKey(attachment)
	if _, initialized := registry.sizeLocks[attachment.terminalID]; !initialized && initialSizeLocked {
		registry.sizeLocks[attachment.terminalID] = true
	}
	takeOwner := attachment.resizePolicy == resizePolicyOwner ||
		(registry.owners[attachment.terminalID] == "" && attachment.resizePolicy != resizePolicyObserver)
	if takeOwner {
		attachment.resizePolicy = resizePolicyOwner
		attachment.epoch = registry.advanceEpochLocked(attachment.terminalID)
		registry.owners[attachment.terminalID] = key
	}
	stored := *attachment
	registry.byKey[key] = &stored
	return registry.controlLocked(&stored, size)
}

func (registry *attachmentRegistry) update(attachment *providerAttachment, size core.Size, takeOwner bool, expectedOwnerEpoch uint64) *providerv1.ResizeControl {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := attachmentKey(attachment)
	if current, ok := registry.byKey[key]; ok {
		current.resizePolicy = attachment.resizePolicy
		current.surfaceID = attachment.surfaceID
		current.viewID = attachment.viewID
		attachment.epoch = current.epoch
	} else {
		registry.byKey[key] = attachment
	}
	ownerKey := registry.owners[attachment.terminalID]
	ownerEpoch := registry.epochs[attachment.terminalID]
	if ownerEpoch == 0 {
		if owner, ok := registry.byKey[ownerKey]; ok {
			ownerEpoch = owner.epoch
		}
	}
	canTakeOwner := takeOwner && (expectedOwnerEpoch == 0 || expectedOwnerEpoch == ownerEpoch)
	switch {
	case canTakeOwner && ownerKey != key:
		attachment.resizePolicy = resizePolicyOwner
		attachment.epoch = registry.advanceEpochLocked(attachment.terminalID)
		registry.owners[attachment.terminalID] = key
	case ownerKey == key && attachment.resizePolicy != resizePolicyOwner:
		delete(registry.owners, attachment.terminalID)
		if !registry.promoteOwnerLocked(attachment.terminalID, key) {
			registry.advanceEpochLocked(attachment.terminalID)
		}
	case ownerKey == key:
		attachment.resizePolicy = resizePolicyOwner
		attachment.epoch = ownerEpoch
	}
	registry.byKey[key] = attachment
	return registry.controlLocked(attachment, size)
}

func (registry *attachmentRegistry) unregister(attachment *providerAttachment) {
	if attachment == nil {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := attachmentKey(attachment)
	delete(registry.byKey, key)
	if registry.owners[attachment.terminalID] == key {
		delete(registry.owners, attachment.terminalID)
		if !registry.promoteOwnerLocked(attachment.terminalID, key) {
			registry.advanceEpochLocked(attachment.terminalID)
		}
	}
}

func (registry *attachmentRegistry) control(attachment *providerAttachment, size core.Size) *providerv1.ResizeControl {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.controlLocked(attachment, size)
}

func (registry *attachmentRegistry) controlLocked(attachment *providerAttachment, size core.Size) *providerv1.ResizeControl {
	key := attachmentKey(attachment)
	ownerKey := registry.owners[attachment.terminalID]
	owner, hasOwner := registry.byKey[ownerKey]
	sizeLocked := registry.sizeLocks[attachment.terminalID]
	if !hasOwner || owner.terminalID != attachment.terminalID {
		control := &providerv1.ResizeControl{
			Reason:     providerv1.ResizeReason_RESIZE_REASON_FOLLOWER,
			SizeLocked: sizeLocked,
			SurfaceId:  attachment.surfaceID,
		}
		if attachment.resizePolicy == resizePolicyObserver {
			control.Reason = providerv1.ResizeReason_RESIZE_REASON_OBSERVER
		}
		return control
	}
	ownership := &providerv1.ResizeOwnership{
		OwnerAttachmentId: attachmentKey(owner),
		OwnerSurfaceId:    owner.surfaceID,
		OwnerViewId:       owner.viewID,
		Size:              sizeToProto(size),
		SizeLocked:        sizeLocked,
		Epoch:             owner.epoch,
	}
	control := &providerv1.ResizeControl{
		CanResize:       ownerKey == key && attachment.resizePolicy == resizePolicyOwner && !sizeLocked,
		Reason:          providerv1.ResizeReason_RESIZE_REASON_FOLLOWER,
		SizeLocked:      sizeLocked,
		SurfaceId:       attachment.surfaceID,
		OwnerSurfaceId:  owner.surfaceID,
		OwnerViewId:     owner.viewID,
		ResizeOwnership: ownership,
	}
	switch {
	case sizeLocked && ownerKey == key && attachment.resizePolicy == resizePolicyOwner:
		control.Reason = providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED
	case attachment.resizePolicy == resizePolicyObserver:
		control.Reason = providerv1.ResizeReason_RESIZE_REASON_OBSERVER
	case control.GetCanResize():
		control.Reason = providerv1.ResizeReason_RESIZE_REASON_OWNER
	}
	return control
}

func (registry *attachmentRegistry) setSizeLock(attachment *providerAttachment, size core.Size, locked bool) (*providerv1.ResizeControl, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	key := attachmentKey(attachment)
	current, ok := registry.byKey[key]
	if !ok {
		return nil, fmt.Errorf("provider: attachment is no longer active")
	}
	if registry.owners[attachment.terminalID] != key || current.resizePolicy != resizePolicyOwner {
		return registry.controlLocked(current, size), nil
	}
	if registry.sizeLocks[attachment.terminalID] != locked {
		epoch := registry.advanceEpochLocked(attachment.terminalID)
		registry.sizeLocks[attachment.terminalID] = locked
		if owner, ok := registry.byKey[registry.owners[attachment.terminalID]]; ok {
			owner.epoch = epoch
		}
	}
	if current, ok := registry.byKey[key]; ok {
		*attachment = *current
	}
	return registry.controlLocked(attachment, size), nil
}

// projection 返回 terminal 的 attachment count / resize epoch / owner control。
func (registry *attachmentRegistry) projection(terminalID string, size core.Size) *providerv1.TerminalAttachmentProjection {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	projection := &providerv1.TerminalAttachmentProjection{
		AttachmentCount: int32(registry.viewCountLocked(terminalID)),
		ResizeEpoch:     registry.epochs[terminalID],
		ResizeControl: &providerv1.ResizeControl{
			Reason:     providerv1.ResizeReason_RESIZE_REASON_FOLLOWER,
			SizeLocked: registry.sizeLocks[terminalID],
		},
	}
	if ownerKey := registry.owners[terminalID]; ownerKey != "" {
		if owner, ok := registry.byKey[ownerKey]; ok {
			if projection.ResizeEpoch == 0 {
				projection.ResizeEpoch = owner.epoch
			}
			projection.ResizeControl = registry.controlLocked(owner, size)
		}
	}
	return projection
}

func (registry *attachmentRegistry) viewCount(terminalID string) int {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.viewCountLocked(terminalID)
}

func (registry *attachmentRegistry) viewCountLocked(terminalID string) int {
	seen := make(map[string]struct{})
	for _, attachment := range registry.byKey {
		if attachment.terminalID != terminalID {
			continue
		}
		key := attachmentKey(attachment)
		if attachment.viewID != "" {
			key = fmt.Sprintf("%d\x00%s\x00%s", attachment.sessionID, attachment.surfaceID, attachment.viewID)
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func (registry *attachmentRegistry) advanceEpochLocked(terminalID string) uint64 {
	registry.nextEpoch++
	registry.epochs[terminalID] = registry.nextEpoch
	return registry.nextEpoch
}

func (registry *attachmentRegistry) promoteOwnerLocked(terminalID string, excludedKey string) bool {
	selectedKey := ""
	var selected *providerAttachment
	for key, attachment := range registry.byKey {
		if key == excludedKey || attachment.terminalID != terminalID || attachment.resizePolicy == resizePolicyObserver {
			continue
		}
		if selectedKey == "" || attachmentPromotionLess(attachment, selected) {
			selectedKey = key
			selected = attachment
		}
	}
	if selectedKey == "" {
		return false
	}
	selected.resizePolicy = resizePolicyOwner
	selected.epoch = registry.advanceEpochLocked(terminalID)
	registry.owners[terminalID] = selectedKey
	return true
}

func attachmentPromotionLess(left, right *providerAttachment) bool {
	if left.sessionID != right.sessionID {
		return left.sessionID < right.sessionID
	}
	if left.channel != right.channel {
		return left.channel < right.channel
	}
	if left.surfaceID != right.surfaceID {
		return left.surfaceID < right.surfaceID
	}
	return left.viewID < right.viewID
}

func newAttachmentToken(channel uint16) ([]byte, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(token[:2], channel)
	return token, nil
}

func normalizeAttachmentMode(mode providerv1.AttachmentMode) string {
	if mode == providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER {
		return attachmentModeObserver
	}
	return attachmentModeCollaborator
}

func normalizeResizePolicy(policy providerv1.ResizePolicy) string {
	switch policy {
	case providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return resizePolicyFollower
	case providerv1.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return resizePolicyObserver
	default:
		return resizePolicyOwner
	}
}

func sizeToProto(size core.Size) *providerv1.Size {
	return &providerv1.Size{Cols: uint32(size.Cols), Rows: uint32(size.Rows)}
}

func sizeFromProto(size *providerv1.Size) core.Size {
	return core.Size{Cols: uint16(size.GetCols()), Rows: uint16(size.GetRows())}
}

func attachmentHandle(attachment *providerAttachment, size core.Size, mode string, policy string) *providerv1.AttachmentHandle {
	return &providerv1.AttachmentHandle{
		OpaqueToken:  append([]byte(nil), attachment.token...),
		TerminalId:   attachment.terminalID,
		Mode:         attachmentModeToProto(mode),
		ResizePolicy: resizePolicyToProto(policy),
		SurfaceId:    attachment.surfaceID,
		ViewId:       attachment.viewID,
		Size:         sizeToProto(size),
		Epoch:        attachment.epoch,
	}
}

func attachmentModeToProto(mode string) providerv1.AttachmentMode {
	if mode == attachmentModeObserver {
		return providerv1.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	}
	return providerv1.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
}

func resizePolicyToProto(policy string) providerv1.ResizePolicy {
	switch policy {
	case resizePolicyFollower:
		return providerv1.ResizePolicy_RESIZE_POLICY_FOLLOWER
	case resizePolicyObserver:
		return providerv1.ResizePolicy_RESIZE_POLICY_OBSERVER
	default:
		return providerv1.ResizePolicy_RESIZE_POLICY_OWNER
	}
}
