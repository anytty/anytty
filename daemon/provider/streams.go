package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/proto/access/wire"
)

// providerRawStream 是一个 attachment channel 的原始 PTY 输出泵。
type providerRawStream struct {
	cancel context.CancelFunc
}

// handleStreamFrame 处理 attachment channel 上的 stream frame：
// client bootstrap 后订阅 raw PTY、回 TypeStreamReady，并转发 PTY 输出；
// TypeClosed 停止输出但保留 attachment resource。
func (session *session) handleStreamFrame(ctx context.Context, channel uint16, typ uint8, payload []byte) error {
	attachment, err := session.attachmentForChannel(channel)
	if err != nil {
		return err
	}
	switch typ {
	case providerproto.StreamBootstrapDone:
		if len(payload) != 0 {
			return fmt.Errorf("provider: attachment bootstrap payload must be empty")
		}
		return session.startRawStream(ctx, attachment)
	case providerproto.StreamClosed:
		if len(payload) != 0 {
			return fmt.Errorf("provider: attachment close payload must be empty")
		}
		session.stopRawStream(channel)
		return nil
	default:
		return fmt.Errorf("provider: unsupported attachment stream frame type %d", typ)
	}
}

func (session *session) startRawStream(ctx context.Context, attachment *providerAttachment) error {
	terminal, err := session.server.core.Terminal(attachment.terminalID)
	if err != nil {
		return mapCoreError(err)
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stream := &providerRawStream{cancel: cancel}
	session.streamMu.Lock()
	if session.streams[attachment.channel] != nil {
		session.streamMu.Unlock()
		cancel()
		return fmt.Errorf("provider: raw PTY stream is already open for this attachment")
	}
	session.streams[attachment.channel] = stream
	session.streamMu.Unlock()

	subscription := terminal.SubscribeRawPTY(streamCtx)
	if err := session.sendFrame(attachment.channel, providerproto.StreamReady, nil); err != nil {
		session.clearRawStream(attachment.channel, stream)
		return err
	}
	subscriptionCtx := streamCtx
	go session.forwardRawStream(subscriptionCtx, attachment.channel, stream, subscription)
	return nil
}

func (session *session) forwardRawStream(ctx context.Context, channel uint16, stream *providerRawStream, subscription interface {
	Receive(context.Context) ([]byte, error)
	Termination() (uint64, *int)
}) {
	defer session.clearRawStream(channel, stream)
	for {
		chunk, err := subscription.Receive(ctx)
		if err == nil {
			if ctx.Err() != nil {
				return
			}
			if sendErr := session.sendFrame(channel, providerproto.StreamPTYOutput, chunk); sendErr != nil {
				return
			}
			continue
		}
		if ctx.Err() != nil || err == context.Canceled {
			return
		}
		droppedBytes, exitCode := subscription.Termination()
		if err == io.EOF || err == nil {
			code := -1
			if exitCode != nil {
				code = *exitCode
			}
			_ = session.sendFrame(channel, providerproto.StreamClosed, wire.EncodeClosedPayload(code))
			return
		}
		// 队列溢出或终端错误：先报告 dropped bytes，再终止流。
		code := -1
		if exitCode != nil {
			code = *exitCode
		}
		if droppedBytes > 0 {
			_ = session.sendFrame(channel, providerproto.StreamSyncLost, wire.EncodeSyncLostPayload(droppedBytes))
		}
		_ = session.sendFrame(channel, providerproto.StreamClosed, wire.EncodeClosedPayload(code))
		return
	}
}

func (session *session) stopRawStream(channel uint16) {
	session.streamMu.Lock()
	stream := session.streams[channel]
	delete(session.streams, channel)
	session.streamMu.Unlock()
	if stream != nil && stream.cancel != nil {
		stream.cancel()
	}
}

func (session *session) clearRawStream(channel uint16, stream *providerRawStream) {
	if stream == nil {
		return
	}
	session.streamMu.Lock()
	if session.streams[channel] == stream {
		delete(session.streams, channel)
	}
	session.streamMu.Unlock()
	if stream.cancel != nil {
		stream.cancel()
	}
}

func (session *session) releaseAttachments() {
	session.streamMu.Lock()
	streams := make([]*providerRawStream, 0, len(session.streams))
	for _, stream := range session.streams {
		streams = append(streams, stream)
	}
	attachments := make([]*providerAttachment, 0, len(session.attachments))
	for _, attachment := range session.attachments {
		attachments = append(attachments, attachment)
	}
	session.streams = make(map[uint16]*providerRawStream)
	session.attachments = make(map[uint16]*providerAttachment)
	session.tokens = make(map[string]*providerAttachment)
	session.streamMu.Unlock()
	for _, stream := range streams {
		if stream.cancel != nil {
			stream.cancel()
		}
	}
	for _, attachment := range attachments {
		session.server.registry.unregister(attachment)
	}
}

func (session *session) attachmentForChannel(channel uint16) (*providerAttachment, error) {
	session.streamMu.Lock()
	defer session.streamMu.Unlock()
	attachment := session.attachments[channel]
	if attachment == nil {
		return nil, &ProviderError{Code: providerproto.ErrorNotFound, Message: "provider: attachment is not bound"}
	}
	return attachment, nil
}

func (session *session) attachmentForToken(token []byte) (*providerAttachment, error) {
	if len(token) == 0 {
		return nil, &ProviderError{Code: providerproto.ErrorNotFound, Message: "provider: attachment token is required"}
	}
	session.streamMu.Lock()
	defer session.streamMu.Unlock()
	attachment := session.tokens[string(token)]
	if attachment == nil {
		return nil, &ProviderError{Code: providerproto.ErrorNotFound, Message: "provider: attachment is not bound"}
	}
	return attachment, nil
}

func (session *session) allocateChannelLocked() (uint16, error) {
	if session.nextChannel >= 1<<16-1 {
		return 0, &ProviderError{Code: providerproto.ErrorExhausted, Message: "provider: stream channels are exhausted"}
	}
	session.nextChannel++
	return uint16(session.nextChannel), nil
}
