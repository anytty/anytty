package provider

import (
	"context"

	"github.com/anytty/anytty/internal/providerproto"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/terminalmeta"
)

func (session *session) dispatchAttach(command *providerv1.TerminalAttachCommand) (*providerv1.Response, error) {
	if command == nil || command.GetTerminal().GetTerminalId() == "" {
		return nil, &ProviderError{Code: providerproto.ErrorBadRequest, Message: "provider: attach terminal id is required"}
	}
	mode := normalizeAttachmentMode(command.GetMode())
	policy := normalizeResizePolicy(command.GetResizePolicy())
	if mode == attachmentModeObserver && policy != resizePolicyObserver {
		return nil, &ProviderError{Code: providerproto.ErrorBadRequest, Message: "provider: observer attachment cannot request resize ownership"}
	}
	info, err := session.server.core.GetTerminal(command.GetTerminal().GetTerminalId())
	if err != nil {
		return nil, mapCoreError(err)
	}
	session.streamMu.Lock()
	channel, err := session.allocateChannelLocked()
	if err != nil {
		session.streamMu.Unlock()
		return nil, err
	}
	token, err := newAttachmentToken(channel)
	if err != nil {
		session.streamMu.Unlock()
		return nil, err
	}
	attachment := &providerAttachment{
		sessionID:    session.sessionID,
		terminalID:   command.GetTerminal().GetTerminalId(),
		channel:      channel,
		mode:         mode,
		resizePolicy: policy,
		surfaceID:    command.GetSurfaceId(),
		viewID:       command.GetViewId(),
		token:        token,
	}
	session.attachments[channel] = attachment
	session.tokens[string(token)] = attachment
	session.streamMu.Unlock()

	control := session.server.registry.register(attachment, info.Size, terminalmeta.SizeLocked(info.Tags))
	if control.GetResizeOwnership() != nil {
		attachment.epoch = control.GetResizeOwnership().GetEpoch()
	}
	if control.GetReason() == providerv1.ResizeReason_RESIZE_REASON_OWNER || control.GetReason() == providerv1.ResizeReason_RESIZE_REASON_SIZE_LOCKED {
		policy = resizePolicyOwner
	}
	return &providerv1.Response{Result: &providerv1.Response_TerminalAttach{TerminalAttach: &providerv1.TerminalAttachResult{
		Attachment:    attachmentHandle(attachment, info.Size, mode, policy),
		ResizeControl: control,
	}}}, nil
}

func (session *session) dispatchResize(ctx context.Context, command *providerv1.TerminalResizeCommand) (*providerv1.Response, error) {
	attachment, err := session.attachmentForToken(command.GetOpaqueToken())
	if err != nil {
		return nil, err
	}
	policy := normalizeResizePolicy(command.GetResizePolicy())
	if attachment.mode == attachmentModeObserver && (command.GetTakeOwnership() || policy != resizePolicyObserver) {
		return nil, &ProviderError{Code: providerproto.ErrorForbidden, Message: "provider: observer attachment cannot request resize ownership"}
	}
	if command.GetResizePolicy() != providerv1.ResizePolicy_RESIZE_POLICY_UNSPECIFIED {
		attachment.resizePolicy = policy
	}
	info, err := session.server.core.GetTerminal(attachment.terminalID)
	if err != nil {
		return nil, mapCoreError(err)
	}
	requested := sizeFromProto(command.GetSize())
	control := session.server.registry.update(attachment, info.Size, command.GetTakeOwnership(), command.GetExpectedOwnerEpoch())
	resized := false
	if control.GetCanResize() {
		if err := session.server.core.ResizeTerminal(ctx, attachment.terminalID, requested.Cols, requested.Rows); err != nil {
			return nil, mapCoreError(err)
		}
		resized = true
	}
	if current, err := session.server.core.GetTerminal(attachment.terminalID); err == nil {
		control = session.server.registry.control(attachment, current.Size)
	}
	return &providerv1.Response{Result: &providerv1.Response_TerminalResize{TerminalResize: &providerv1.TerminalResizeResult{
		Size: sizeToProto(requested), Resized: resized, ResizeControl: control,
	}}}, nil
}

func (session *session) detachAttachment(attachment *providerAttachment) {
	session.stopRawStream(attachment.channel)
	session.streamMu.Lock()
	delete(session.attachments, attachment.channel)
	delete(session.tokens, string(attachment.token))
	session.streamMu.Unlock()
	session.server.registry.unregister(attachment)
}
